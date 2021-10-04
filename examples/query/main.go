//go:build windows
// +build windows

package main

import (
	"context"
	"crypto/sha1"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
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
	var tlsconf *tls.Config

	if false {
		cert, err := examples.FindCert(os.Args[2])
		if err != nil {
			panic(err)
		}

		tlsconf = &tls.Config{
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

	svcs, err := n.GetServiceList(context.Background(), apps[0].ApplicationName.String())
	if err != nil {
		panic(err)
	}
	log.Println("Services: ", svcs)

	/*
		props, err := n.EnumerateProperties(context.Background(), "fabric:/pinger0")
		if err != nil {
			panic(err)
		}

		log.Println("Properties: ", props)
	*/

	props, err := n.GetServiceTypeList(context.Background(), apps[0].ApplicationTypeName, apps[0].ApplicationTypeVersion, "")
	if err != nil {
		panic(err)
	}

	log.Println("ServiceTypes: ", props)

	parts, err := n.GetServicePartitionList(context.Background(), "fabric:/pinger0/PingerService")
	if err != nil {
		panic(err)
	}

	log.Println("Partitions: ", parts)

	repl, err := n.GetServicePartitionReplicaList(context.Background(), parts[0].PartitionInformation.PartitionId.String())
	if err != nil {
		panic(err)
	}

	log.Println("Replicas: ", repl)

	n.OnServiceNotification = func(notification *naming.ServiceNotification) {
		o, _ := json.MarshalIndent(notification, "", "  ")
		log.Printf("update callback %v", string(o))
	}

	// TODO uri parser
	_, err = n.RegisterFilter(context.Background(), common.Uri{
		Type:         common.UriTypeEmpty,
		Scheme:       "fabric",
		Authority:    "",
		HostType:     common.UriHostTypeNone,
		Host:         "",
		Port:         -1,
		Path:         "",
		PathSegments: []string{""},
	}, true, false)
	if err != nil {
		panic(err)
	}

	// wait for notification
	time.Sleep(1 * time.Hour)
}
