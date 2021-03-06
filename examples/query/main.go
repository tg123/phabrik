//go:build windows
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
	"time"

	"github.com/tg123/phabrik/common"
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

	c, err := transport.DialTCP(os.Args[1], transport.ClientConfig{
		Config: transport.Config{
			TLS: tlsconf,
		},
	})

	if err != nil {
		panic(err)
	}

	defer c.Close()

	go func() {
		log.Println("loop error", c.Wait())
	}()

	n, err := naming.NewNamingClient(c)
	if err != nil {
		panic(err)
	}

	gateway, err := n.Ping(context.Background())

	if err != nil {
		panic(err)
	}

	log.Printf("Connected, Gateway info: %v", gateway)

	apps, err := n.GetApplicationList(context.Background(), "")

	if err != nil {
		panic(err)
	}

	log.Println("Applications: ", apps)

	n.OnServiceNotification = func(notification *naming.ServiceNotification) {
		log.Printf("update callback %v", notification)
	}

	// TODO uri parser
	_, err = n.RegisterFilter(context.Background(), common.Uri{
		Type:         common.UriTypeAbsolute,
		Scheme:       "fabric",
		Authority:    "",
		HostType:     common.UriHostTypeNone,
		Host:         "",
		Port:         -1,
		Path:         "/test",
		PathSegments: []string{"test"},
	}, true, false)
	if err != nil {
		panic(err)
	}

	// wait for notification
	time.Sleep(1 * time.Hour)
}
