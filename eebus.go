package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"time"

	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"

	"github.com/andig/evcc/hems/eebus"
	"github.com/andig/evcc/hems/eebus/ship"
	"github.com/gorilla/websocket"
	"github.com/grandcat/zeroconf"
)

const (
	zeroconfType   = "_ship._tcp"
	zeroconfDomain = "local."
)

func discoverDNS(results <-chan *zeroconf.ServiceEntry) {
	for entry := range results {
		log.Printf("mdns: %v\n", entry)
		ss, err := eebus.NewFromDNSEntry(entry)
		if err == nil {
			err = ss.Connect()
			log.Printf("connect %s: %v", entry.HostName, err)
		}

		if err == nil {
			err = ss.Close()
		}

		if err != nil {
			log.Println(err)
		}
	}
}

func publicKey(priv interface{}) interface{} {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey
	case *ecdsa.PrivateKey:
		return &k.PublicKey
	default:
		return nil
	}
}

func pemBlockForKey(priv interface{}) *pem.Block {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}
	case *ecdsa.PrivateKey:
		b, err := x509.MarshalECPrivateKey(k)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to marshal ECDSA private key: %v", err)
			os.Exit(2)
		}
		return &pem.Block{Type: "EC PRIVATE KEY", Bytes: b}
	default:
		return nil
	}
}

func createCertificate(isCA bool, hosts ...string) (tls.Certificate, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, err
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Acme Co"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour * 24 * 180),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, h)
		}
	}
	if isCA {
		template.IsCA = true
		template.KeyUsage |= x509.KeyUsageCertSign
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, publicKey(priv), priv)
	if err != nil {
		return tls.Certificate{}, err
	}

	tlsCert := tls.Certificate{
		Certificate: [][]byte{derBytes},
		PrivateKey:  priv,
	}

	return tlsCert, nil
}

func SaveX509KeyPair(certFile, keyFile string, cert tls.Certificate) error {
	out := &bytes.Buffer{}
	err := pem.Encode(out, &pem.Block{Type: "CERTIFICATE", Bytes: cert.Certificate[0]})
	if err == nil {
		fmt.Println(out.String())
		err = ioutil.WriteFile(certFile, out.Bytes(), fs.ModePerm)
	}

	if err == nil {
		out.Reset()
		err = pem.Encode(out, pemBlockForKey(cert.PrivateKey))
	}

	if err == nil {
		fmt.Println(out.String())
		err = ioutil.WriteFile(keyFile, out.Bytes(), fs.ModePerm)
	}

	return err
}

func selfSignedConnection(cert tls.Certificate) func(uri string) (*websocket.Conn, error) {
	return func(uri string) (*websocket.Conn, error) {
		// ips := semp.LocalIPs()
		// ip := ips[0]
		// log.Println("using ip: " + ip.String())

		// certPool, err := x509.SystemCertPool()
		// if err != nil {
		// 	return nil, err
		// }

		dialer := &websocket.Dialer{
			Proxy:            http.ProxyFromEnvironment,
			HandshakeTimeout: 5 * time.Second,
			TLSClientConfig: &tls.Config{
				// RootCAs:            certPool,
				Certificates:       []tls.Certificate{cert},
				InsecureSkipVerify: true,
			},
			Subprotocols: []string{ship.SubProtocol},
		}

		log.Println("using uri: " + uri)

		conn, resp, err := dialer.Dial(uri, http.Header{})
		fmt.Println(resp)

		return conn, err
	}
}

const (
	serverPort = 4712
	certFile   = "eebus.crt"
	keyFile    = "eebus.key"
)

func main() {
	// Discover all services on the network (e.g. _workstation._tcp)
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		log.Fatalln("Failed to initialize resolver:", err.Error())
	}

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	// err = os.ErrNotExist
	if err != nil {
		if os.IsNotExist(err) {
			if cert, err = createCertificate(true, "evcc"); err == nil {
				err = SaveX509KeyPair(certFile, keyFile, cert)
			}
		}

		if err != nil {
			log.Fatal(err)
		}
	}

	// created signed connections
	eebus.Connector = selfSignedConnection(cert)

	// have certificate now
	leaf, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		log.Fatalln("failed parsing certificate:", err.Error())
	}
	ski := fmt.Sprintf("%04X", leaf.SubjectKeyId)
	fmt.Println("ski:", ski)

	service, err := eebus.NewServer(fmt.Sprintf(":%d", serverPort), cert)
	if err != nil {
		log.Fatalln(err)
	}
	_ = service

	server, err := zeroconf.Register("evcc", zeroconfType, zeroconfDomain, serverPort,
		// []string{"txtvers=1", "id=evcc-01", "path=/ship/", "ski=" + ski, "register=true", "type=EnergyManagementSystem"}, nil)
		[]string{"txtvers=1", "id=evcc-01", "path=/ship/", "ski=" + ski, "register=true", "type=Energy Manager"}, nil)
	if err != nil {
		log.Fatalln(err)
	}
	defer server.Shutdown()

	// os.Exit(0)

	entries := make(chan *zeroconf.ServiceEntry)
	go discoverDNS(entries)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err = resolver.Browse(ctx, zeroconfType, zeroconfDomain, entries); err != nil {
		log.Fatalln("Failed to browse:", err.Error())
	}

	<-ctx.Done()
}