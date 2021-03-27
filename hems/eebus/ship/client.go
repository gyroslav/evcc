package ship

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/andig/evcc/hems/eebus/ship/message"
	"github.com/andig/evcc/hems/eebus/ship/ship"
	"github.com/andig/evcc/hems/eebus/ship/transport"
	"github.com/andig/evcc/hems/eebus/util"
	"github.com/gorilla/websocket"
)

// Client is the ship client
type Client struct {
	mux    sync.Mutex
	Log    util.Logger
	Local  Service
	Remote Service
	t      *transport.Transport
	closed bool
}

// init creates the connection
func (c *Client) init() error {
	init := []byte{message.CmiTypeInit, 0x00}

	// CMI_STATE_CLIENT_SEND
	if err := c.t.WriteBinary(init); err != nil {
		return err
	}

	timer := time.NewTimer(message.CmiTimeout)

	// CMI_STATE_CLIENT_WAIT
	msg, err := c.t.ReadBinary(timer.C)
	if err != nil {
		return err
	}

	// CMI_STATE_CLIENT_EVALUATE
	if !bytes.Equal(init, msg) {
		return fmt.Errorf("init: invalid response")
	}

	return nil
}

func (c *Client) protocolHandshake() error {
	hs := ship.CmiMessageProtocolHandshake{
		MessageProtocolHandshake: ship.MessageProtocolHandshake{
			HandshakeType: ship.ProtocolHandshakeTypeTypeAnnouncemax,
			Version:       ship.Version{Major: 1, Minor: 0},
			Formats: ship.MessageProtocolFormatsType{
				Format: []ship.MessageProtocolFormatType{ship.ProtocolHandshakeFormatJSON},
			},
		},
	}
	if err := c.t.WriteJSON(message.CmiTypeControl, hs); err != nil {
		return fmt.Errorf("handshake: %w", err)
	}

	// receive server selection and send selection back to server
	err := c.t.HandshakeReceiveSelect()
	if err == nil {
		hs.MessageProtocolHandshake.HandshakeType = ship.ProtocolHandshakeTypeTypeSelect
		err = c.t.WriteJSON(message.CmiTypeControl, hs)
	}

	return err
}

// Close performs ordered close of client connection
func (c *Client) Close() error {
	c.mux.Lock()
	defer c.mux.Unlock()

	if c.closed {
		return os.ErrClosed
	}

	c.closed = true

	// stop readPump
	// defer close(c.closeC)

	return c.t.Close()
}

func (c *Client) dataHandshake() error {
	target := "19667_PorscheEVSE_0001503"

	discovery := `{"datagram":[{"header":[{"specificationVersion":"1.2.0"},{"addressSource":[{"device":"d:_i:3210_ESystemsMtg-CEM"},{"entity":[0]},{"feature":0}]},{"addressDestination":[{"entity":[0]},{"feature":0}]},{"msgCounter":5876},{"cmdClassifier":"read"}]},{"payload":[{"cmd":[[{"nodeManagementDetailedDiscoveryData":[]}]]}]}]}`
	hs := ship.CmiData{
		Data: ship.Data{
			Header: ship.HeaderType{
				ProtocolId: ship.ProtocolIdType("ee1.0"),
			},
			Payload: json.RawMessage(discovery),
		},
	}
	err := c.t.WriteJSON(message.CmiTypeData, hs)

	services := `{"datagram":[{"header":[{"specificationVersion":"1.2.0"},{"addressSource":[{"device":"d:_i:3210_ESystemsMtg-CEM"},{"entity":[0]},{"feature":0}]},{"addressDestination":[{"device":"d:_i:%s"},{"entity":[0]},{"feature":0}]},{"msgCounter":5881},{"msgCounterReference":1},{"cmdClassifier":"reply"}]},{"payload":[{"cmd":[[{"nodeManagementDetailedDiscoveryData":[{"specificationVersionList":[{"specificationVersion":["1.2.0"]}]},{"deviceInformation":[{"description":[{"deviceAddress":[{"device":"d:_i:3210_ESystemsMtg-CEM"}]},{"deviceType":"EnergyManagementSystem"},{"networkFeatureSet":"smart"}]}]},{"entityInformation":[[{"description":[{"entityAddress":[{"entity":[0]}]},{"entityType":"DeviceInformation"}]}],[{"description":[{"entityAddress":[{"entity":[1]}]},{"entityType":"CEM"},{"description":"CEM Energy Guard"}]}],[{"description":[{"entityAddress":[{"entity":[2]}]},{"entityType":"HeatPumpAppliance"},{"description":"CEM Controllable System"}]}]]},{"featureInformation":[[{"description":[{"featureAddress":[{"entity":[0]},{"feature":0}]},{"featureType":"NodeManagement"},{"role":"special"},{"supportedFunction":[[{"function":"nodeManagementDetailedDiscoveryData"},{"possibleOperations":[{"read":[]}]}],[{"function":"nodeManagementSubscriptionRequestCall"},{"possibleOperations":[]}],[{"function":"nodeManagementBindingRequestCall"},{"possibleOperations":[]}],[{"function":"nodeManagementSubscriptionDeleteCall"},{"possibleOperations":[]}],[{"function":"nodeManagementBindingDeleteCall"},{"possibleOperations":[]}],[{"function":"nodeManagementSubscriptionData"},{"possibleOperations":[{"read":[]}]}],[{"function":"nodeManagementBindingData"},{"possibleOperations":[{"read":[]}]}],[{"function":"nodeManagementUseCaseData"},{"possibleOperations":[{"read":[]}]}]]}]}],[{"description":[{"featureAddress":[{"entity":[0]},{"feature":1}]},{"featureType":"DeviceClassification"},{"role":"server"},{"supportedFunction":[[{"function":"deviceClassificationManufacturerData"},{"possibleOperations":[{"read":[]}]}]]}]}],[{"description":[{"featureAddress":[{"entity":[1]},{"feature":1}]},{"featureType":"DeviceClassification"},{"role":"client"},{"description":"Device Classification"}]}],[{"description":[{"featureAddress":[{"entity":[1]},{"feature":2}]},{"featureType":"DeviceDiagnosis"},{"role":"client"},{"description":"Device Diagnosis"}]}],[{"description":[{"featureAddress":[{"entity":[1]},{"feature":3}]},{"featureType":"Measurement"},{"role":"client"},{"description":"Measurement for client"}]}],[{"description":[{"featureAddress":[{"entity":[1]},{"feature":4}]},{"featureType":"DeviceConfiguration"},{"role":"client"},{"description":"Device Configuration"}]}],[{"description":[{"featureAddress":[{"entity":[1]},{"feature":5}]},{"featureType":"DeviceDiagnosis"},{"role":"server"},{"supportedFunction":[[{"function":"deviceDiagnosisStateData"},{"possibleOperations":[{"read":[]}]}],[{"function":"deviceDiagnosisHeartbeatData"},{"possibleOperations":[{"read":[]}]}]]},{"description":"DeviceDiag"}]}],[{"description":[{"featureAddress":[{"entity":[1]},{"feature":7}]},{"featureType":"LoadControl"},{"role":"client"},{"description":"LoadControl client for CEM"}]}],[{"description":[{"featureAddress":[{"entity":[1]},{"feature":8}]},{"featureType":"Identification"},{"role":"client"},{"description":"EV identification"}]}],[{"description":[{"featureAddress":[{"entity":[1]},{"feature":9}]},{"featureType":"ElectricalConnection"},{"role":"client"},{"description":"Electrical Connection"}]}],[{"description":[{"featureAddress":[{"entity":[2]},{"feature":10}]},{"featureType":"LoadControl"},{"role":"server"},{"supportedFunction":[[{"function":"loadControlLimitDescriptionListData"},{"possibleOperations":[{"read":[]}]}],[{"function":"loadControlLimitListData"},{"possibleOperations":[{"read":[]},{"write":[]}]}]]},{"description":"Load Control"}]}],[{"description":[{"featureAddress":[{"entity":[2]},{"feature":11}]},{"featureType":"ElectricalConnection"},{"role":"server"},{"supportedFunction":[[{"function":"electricalConnectionCharacteristicListData"},{"possibleOperations":[{"read":[]}]}],[{"function":"electricalConnectionDescriptionListData"},{"possibleOperations":[{"read":[]}]}]]},{"description":"Electrical Connection"}]}],[{"description":[{"featureAddress":[{"entity":[2]},{"feature":12}]},{"featureType":"DeviceConfiguration"},{"role":"server"},{"supportedFunction":[[{"function":"deviceConfigurationKeyValueDescriptionListData"},{"possibleOperations":[{"read":[]}]}],[{"function":"deviceConfigurationKeyValueListData"},{"possibleOperations":[{"read":[]},{"write":[]}]}]]},{"description":"Device Configuration"}]}]]}]}]]}]}]}`

	services = fmt.Sprintf(services, target)

	var servicesSent bool
	for err == nil {
		err = c.t.DataReceive()
		if err == nil && !servicesSent {
			hs := ship.CmiData{
				Data: ship.Data{
					Header: ship.HeaderType{
						ProtocolId: ship.ProtocolIdType("ee1.0"),
					},
					Payload: json.RawMessage(services),
				},
			}

			err = c.t.WriteJSON(message.CmiTypeData, hs)
			servicesSent = true
		}
	}

	return err
}

// Connect performs the client connection handshake
func (c *Client) Connect(conn *websocket.Conn) error {
	c.t = transport.New(c.Log, conn)

	if err := c.init(); err != nil {
		return err
	}

	err := c.t.Hello()
	if err == nil {
		err = c.protocolHandshake()
	}
	if err == nil {
		err = c.t.PinState(
			ship.PinValueType(c.Local.Pin),
			ship.PinValueType(c.Remote.Pin),
		)
	}
	if err == nil {
		c.Remote.Methods, err = c.t.AccessMethodsRequest(c.Local.Methods)
	}
	if err == nil {
		err = c.dataHandshake()
	}

	// close connection if handshake or hello fails
	if err != nil {
		_ = c.t.Close()
	}

	return err
}
