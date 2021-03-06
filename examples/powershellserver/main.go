//go:build windows
// +build windows

package main

import (
	"crypto/sha1"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"os"

	"github.com/tg123/phabrik/common"
	"github.com/tg123/phabrik/examples"
	"github.com/tg123/phabrik/federation"
	"github.com/tg123/phabrik/naming"
	"github.com/tg123/phabrik/serialization"
	"github.com/tg123/phabrik/transport"
)

func main() {
	// usage powershellserver <listen address> <server thumbprint>

	cert, err := examples.FindCert(os.Args[2])
	if err != nil {
		panic(err)
	}

	tlsconf := &tls.Config{
		Certificates: []tls.Certificate{*cert},
		ClientAuth:   tls.RequestClientCert,
		VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
			for _, rawCert := range rawCerts {
				thumb := fmt.Sprintf("%x", sha1.Sum(rawCert))
				fmt.Println("Client thumbprint", thumb)
			}

			return nil
		},
	}

	s, err := transport.ListenTCP(os.Args[1], transport.ServerConfig{
		Config: transport.Config{
			TLS: tlsconf,
		},
		MessageCallback: func(c transport.Conn, bam *transport.ByteArrayMessage) {
			switch bam.Headers.Actor {
			case transport.MessageActorTypeNamingGateway:
				// fmt.Println(bam)

				switch bam.Headers.Action {
				case "PingRequest":
					msg, err := naming.NewNamingMessage("PingRequest")
					if err != nil {
						log.Printf("new msg %v", err)
						return
					}
					msg.Headers.RelatesTo = bam.Headers.Id

					msg.Body = &struct {
						GatewayDescription naming.GatewayDescription
					}{
						GatewayDescription: naming.GatewayDescription{
							Address: "127.0.0.1:9998",
							NodeInstance: federation.NodeInstance{
								Id:         federation.NodeIDFromMD5("NodeName"),
								InstanceId: 1000,
							},
							NodeName: "NodeName",
						},
					}

					if err := c.SendOneWay(msg); err != nil {
						log.Printf("send err %v", err)
						return
					}

				case "NameExistsRequest":
					var uri common.Uri
					if err := serialization.Unmarshal(bam.Body, &uri); err != nil {
						log.Printf("NameExistsRequest body err %v", err)
						return
					}

					msg, err := naming.NewNamingMessage("NameOperationReply")
					if err != nil {
						log.Printf("new msg %v", err)
						return
					}
					msg.Headers.RelatesTo = bam.Headers.Id

					msg.Body = &struct {
						NameExists       bool
						UserServiceState int64
					}{
						NameExists:       false,
						UserServiceState: 0,
					}

					if err := c.SendOneWay(msg); err != nil {
						log.Printf("send err %v", err)
						return
					}

					fmt.Println("a client connected")
				// case "QueryRequest":

				default:
				}
			default:
			}
		},
	})

	if err != nil {
		panic(err)
	}

	panic(s.Serve())
}
