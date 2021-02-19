package transport

import (
	"errors"
	"time"

	"github.com/andig/evcc/hems/eebus/ship/message"
)

func (c *Transport) HandshakeReceiveSelect() error {
	timer := time.NewTimer(CmiReadWriteTimeout)
	msg, err := c.ReadMessage(timer.C)
	if err != nil {
		return err
	}

	switch typed := msg.(type) {
	case message.MessageProtocolHandshake:
		if typed.HandshakeType != message.ProtocolHandshakeTypeSelect || !typed.Formats.IsSupported(message.ProtocolHandshakeFormatJSON) {
			_ = c.WriteJSON(message.CmiTypeControl, message.CmiProtocolHandshakeError{
				Error: message.CmiProtocolHandshakeErrorUnexpectedMessage,
			})

			err = errors.New("handshake: invalid format")
		}

		return nil

	case message.ConnectionClose:
		err = errors.New("handshake: remote closed")

	default:
		err = errors.New("handshake: invalid type")
	}

	return err
}
