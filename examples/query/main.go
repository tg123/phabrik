// +build windows
package main

import (
	"crypto/sha1"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"os"

	"github.com/tg123/phabrik/examples"
	"github.com/tg123/phabrik/naming"
	"github.com/tg123/phabrik/transport"
)

func main() {
	// usage query <service fabric endpoint> <client thumbprint>

	cert, err := examples.FindCert(os.Args[2])
	if err != nil {
		panic(err)
	}

	tlsconf := &tls.Config{
		Certificates:       []tls.Certificate{*cert},
		InsecureSkipVerify: true,
		VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
			for _, rawCert := range rawCerts {
				thumb := fmt.Sprintf("%x", sha1.Sum(rawCert))
				fmt.Println("Remote thumbprint", thumb)

			}
			return nil
		},
	}

	c, err := transport.DialTCP(os.Args[1], transport.Config{
		TLS: tlsconf,
	})

	if err != nil {
		panic(err)
	}

	defer c.Close()

	go func() {
		log.Println("loop error", c.Wait())
	}()

	n := naming.NamingClient{c}

	gateway, err := n.Ping()

	if err != nil {
		panic(err)
	}

	log.Printf("Connected, Gateway info: %v", gateway)

	apps, err := n.GetApplicationList("")

	if err != nil {
		panic(err)
	}

	log.Println("Applications: ", apps)
}
