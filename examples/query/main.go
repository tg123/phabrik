// +build windows
package main

import (
	"context"
	"crypto/sha1"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"os"

	"github.com/github/certstore"
	"github.com/tg123/phabrik/naming"
	"github.com/tg123/phabrik/transport"
)

func findCert(thumbprint string) (*tls.Certificate, error) {
	store, err := certstore.Open()
	if err != nil {
		return nil, err
	}
	defer store.Close()

	idents, err := store.Identities()
	if err != nil {
		return nil, err
	}

	var cert tls.Certificate
	for _, i := range idents {
		c, err := i.Certificate()

		if err != nil {
			continue
		}

		thumb := fmt.Sprintf("%x", sha1.Sum(c.Raw))
		if thumb == thumbprint {
			s, err := i.Signer()
			if err != nil {
				return nil, err
			}
			cert.Certificate = [][]byte{c.Raw}
			cert.PrivateKey = s
		}
	}

	return &cert, nil
}

func main() {
	// usage query <service fabric endpoint> <client thumbprint>

	cert, err := findCert(os.Args[2])
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

	c, err := transport.Dial(os.Args[1], transport.Config{
		TLS: tlsconf,
	})

	if err != nil {
		panic(err)
	}

	defer c.Close()

	go func() {
		log.Println("loop error", c.Run(context.Background()))
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
