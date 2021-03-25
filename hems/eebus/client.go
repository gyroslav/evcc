package eebus

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/andig/evcc/hems/eebus/ship"
	"github.com/gorilla/websocket"
	"github.com/grandcat/zeroconf"
	"github.com/mitchellh/mapstructure"
)

// ServiceDescription contains the ship service parameters
type ServiceDescription struct {
	Model, Brand string
	SKI          string
	Register     bool
	Path         string
	ID           string
}

// Service is the ship service
type Service struct {
	ServiceDescription
	URI  string
	Conn *ship.Client
}

// NewFromDNSEntry creates ship service from its DNS definition
func NewFromDNSEntry(zc *zeroconf.ServiceEntry) (*Service, error) {
	ss := Service{}

	txtM := make(map[string]interface{})
	for _, txtE := range zc.Text {
		split := strings.SplitN(txtE, "=", 2)
		if len(split) == 2 {
			txtM[split[0]] = split[1]
		}
	}

	decoderConfig := &mapstructure.DecoderConfig{
		Result:           &ss.ServiceDescription,
		WeaklyTypedInput: true,
	}

	decoder, err := mapstructure.NewDecoder(decoderConfig)
	if err == nil {
		err = decoder.Decode(txtM)
	}

	ss.URI = baseURIFromDNS(zc) + ss.ServiceDescription.Path

	return &ss, err
}

// baseURIFromDNS returns the service URI
func baseURIFromDNS(zc *zeroconf.ServiceEntry) string {
	uri := ship.Scheme + zc.HostName
	if zc.Port != 443 {
		uri += fmt.Sprintf(":%d", zc.Port)
	}

	return uri
}

// Connector is the connector used for establishing new websocket connections
var Connector func(uri string) (*websocket.Conn, error)

// Connect connects to the service endpoint and performs handshake
func (ss *Service) Connect(accessMethod string) error {
	conn, err := Connector(ss.URI)
	if err != nil {
		return err
	}

	sc := &ship.Client{
		Log:    log.New(&writer{os.Stdout, "2006/01/02 15:04:05 "}, "[client] ", 0),
		Local:  ship.Service{Pin: "", Methods: accessMethod},
		Remote: ship.Service{Pin: ""},
	}

	ss.Conn = sc
	if err = sc.Connect(conn); err != nil {
		log.Println(err)
	}

	return err
}

// Close closes the service connection
func (ss *Service) Close() error {
	return ss.Conn.Close()
}
