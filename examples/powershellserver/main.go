package main

import (
	"fmt"
	"log"

	"github.com/tg123/phabrik/naming"
	"github.com/tg123/phabrik/serialization"
	"github.com/tg123/phabrik/transport"
)

func main() {

	s, err := transport.ListenTCP("127.0.0.1:9998", transport.Config{
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
							NodeInstance: naming.NodeInstance{
								Id:         naming.NodeIDFromMD5("NodeName"),
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
					var uri naming.Uri
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
