package ship

import (
	"bytes"
	"errors"
	"fmt"
	"time"

	"github.com/andig/evcc/hems/eebus/ship/message"
	"github.com/andig/evcc/hems/eebus/ship/transport"
	"github.com/andig/evcc/hems/eebus/util"
	"github.com/gorilla/websocket"
)

// Server is the SHIP server
type Server struct {
	Log     util.Logger
	Local   Service
	Remote  Service
	t       *transport.Transport
	Handler func(req interface{}) error
}

// Init creates the connection
func (c *Server) init() error {
	timer := time.NewTimer(message.CmiTimeout)

	// CMI_STATE_SERVER_WAIT
	msg, err := c.t.ReadBinary(timer.C)
	if err != nil {
		return err
	}

	// CMI_STATE_SERVER_EVALUATE
	init := []byte{message.CmiTypeInit, 0x00}
	if bytes.Compare(init, msg) != 0 {
		return fmt.Errorf("init: invalid response")
	}

	return c.t.WriteBinary(init)
}

func (c *Server) protocolHandshake() error {
	timer := time.NewTimer(transport.CmiReadWriteTimeout)
	msg, err := c.t.ReadMessage(timer.C)
	if err != nil {
		if errors.Is(err, transport.ErrTimeout) {
			_ = c.t.WriteJSON(message.CmiTypeControl, message.CmiMessageProtocolHandshakeError{
				MessageProtocolHandshakeError: message.MessageProtocolHandshakeError{
					Error: "2", // TODO
				}})
		}

		return err
	}

	switch typed := msg.(type) {
	case message.MessageProtocolHandshake:
		if typed.HandshakeType != message.ProtocolHandshakeTypeTypeAnnouncemax || !typed.Formats.IsSupported(message.ProtocolHandshakeFormatJSON) {
			msg := message.CmiMessageProtocolHandshakeError{
				MessageProtocolHandshakeError: message.MessageProtocolHandshakeError{
					Error: "2", // TODO
				},
			}

			_ = c.t.WriteJSON(message.CmiTypeControl, msg)
			err = errors.New("handshake: invalid response")
			break
		}

		// send selection to client
		typed.HandshakeType = message.ProtocolHandshakeTypeTypeSelect
		err = c.t.WriteJSON(message.CmiTypeControl, message.CmiMessageProtocolHandshake{
			MessageProtocolHandshake: typed,
		})

	default:
		return fmt.Errorf("handshake: invalid type")
	}

	// receive selection back from client
	if err == nil {
		err = c.t.HandshakeReceiveSelect()
	}

	return err
}

// Close performs ordered close of server connection
func (c *Server) Close() error {
	return c.t.Close()
}

// Serve performs the server connection handshake
func (c *Server) Serve(conn *websocket.Conn) error {
	c.t = transport.New(c.Log, conn)

	if err := c.init(); err != nil {
		return err
	}

	// CMI_STATE_DATA_PREPARATION
	err := c.t.Hello()

	if err == nil {
		err = c.protocolHandshake()
	}
	if err == nil {
		err = c.t.PinState(
			message.PinValueType(c.Local.Pin),
			message.PinValueType(c.Remote.Pin),
		)
	}
	if err == nil {
		c.Remote.Methods, err = c.t.AccessMethodsRequest(c.Local.Methods)
	}

	for err == nil {
		endless := make(chan time.Time)

		var msg interface{}
		msg, err = c.t.ReadMessage(endless)
		if err != nil {
			break
		}

		switch typed := msg.(type) {
		case message.ConnectionClose:
			return c.t.AcceptClose()

		case message.Data:
			// c.log().Printf("serv: %+v", msg)
			if c.Handler == nil {
				err = errors.New("no handler")
				break
			}

			if err = c.Handler(typed); err != nil {
				break
			}

		default:
			err = errors.New("invalid type")
		}
	}

	// close connection if handshake or hello fails
	if err != nil {
		_ = c.Close()
	}

	return err
}