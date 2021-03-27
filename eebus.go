package main

import (
	"context"
	"crypto/elliptic"
	"crypto/sha1"
	"crypto/tls"
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os/signal"
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
	zeroconfType     = "_ship._tcp"
	zeroconfDomain   = "local."
	zeroconfInstance = "evcc"
)

func discoverDNS(results <-chan *zeroconf.ServiceEntry) {
	for entry := range results {
		log.Println("mDNS:", entry.HostName, entry.AddrIPv4, entry.Text)

		go connectService(entry)
		// if entry.Instance == zeroconfInstance {
		// 	go connectService(entry)
		// }
	}
}

func connectService(entry *zeroconf.ServiceEntry) {
	ss, err := eebus.NewFromDNSEntry(entry)
	if err == nil {
		log.Printf("%s: client connect", entry.HostName)
		err = ss.Connect(mdnsID.String())
	}

	if err == nil {
		err = ss.Close()
	}

	log.Printf("%s: client done: %v", entry.HostName, err)
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

func createCertificate(isCA bool, cn string, hosts ...string) (tls.Certificate, error) {
	// priv, err := rsa.GenerateKey(rand.Reader, 2048)
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}

	// convert pubkey to ski
	pub, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return tls.Certificate{}, err
	}
	ski := sha1.Sum(pub)

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:   cn,
			Country:      []string{"DE"},
			Organization: []string{"EVCC"},
		},
		SignatureAlgorithm:    x509.ECDSAWithSHA256,
		SubjectKeyId:          ski[:],
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour * 24 * 365 * 10),
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
		dialer := &websocket.Dialer{
			Proxy:            http.ProxyFromEnvironment,
			HandshakeTimeout: 5 * time.Second,
			TLSClientConfig: &tls.Config{
				Certificates:       []tls.Certificate{cert},
				InsecureSkipVerify: true,
				CipherSuites:       ship.CipherSuites,
			},
			Subprotocols: []string{ship.SubProtocol},
		}

		conn, _, err := dialer.Dial(uri, http.Header{})
		// log.Println("dial:", uri, resp, err)

		return conn, err
	}
}

const (
	serverPort = 4712
	certFile   = "evcc.crt"
	keyFile    = "evcc.key"
)

var mdnsID = eebus.UniqueID{Prefix: "evcc"}

func main() {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		if os.IsNotExist(err) {
			if cert, err = createCertificate(true, zeroconfInstance); err == nil {
				err = SaveX509KeyPair(certFile, keyFile, cert)
			}
		}

		if err != nil {
			log.Fatal(err)
		}
	}

	// create signed connections
	eebus.Connector = selfSignedConnection(cert)

	// have certificate now
	leaf, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		log.Fatal("failed parsing certificate:", err.Error())
	}
	ski := fmt.Sprintf("%0x", leaf.SubjectKeyId)

	service, err := eebus.NewServer(fmt.Sprintf(":%d", serverPort), cert, mdnsID.String())
	if err != nil {
		log.Fatal(err)
	}
	_ = service

	log.Printf("mDNS: announcing id: %s ski: %s", mdnsID.String(), ski)

	// Discover all services on the network (e.g. _workstation._tcp)
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		log.Fatal("mDNS: failed initializing resolver:", err)
	}

	server, err := zeroconf.Register(zeroconfInstance, zeroconfType, zeroconfDomain, serverPort, []string{
		"txtvers=1", "id=" + mdnsID.String(), "path=/ship/", "ski=" + ski, "register=true", "brand=evcc", "model=evcc", "type=EnergyManagementSystem",
	}, nil)
	if err != nil {
		log.Fatal("mDNS: failed registering service:", err)
	}
	defer server.Shutdown()

	entries := make(chan *zeroconf.ServiceEntry)
	go discoverDNS(entries)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err = resolver.Browse(ctx, zeroconfType, zeroconfDomain, entries); err != nil {
		log.Fatal("failed to browse:", err.Error())
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			log.Println("mDNS: shutdown")
			server.Shutdown()
			cancel()
			os.Exit(0)
		}
	}()

	<-ctx.Done()
}
