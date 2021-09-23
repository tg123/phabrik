package transport

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBasicServer(t *testing.T) {
	server, err := ListenTCP("127.0.0.1:0", ServerConfig{
		MessageCallback: func(c Conn, bam *ByteArrayMessage) {
			msg := &Message{}
			msg.Headers.RelatesTo = bam.Headers.Id
			msg.Body = []byte(hex.EncodeToString(bam.Body))

			err := c.SendOneWay(msg)
			if err != nil {
				t.Error(err)
			}
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	defer server.Close()
	go server.Serve()

	t.Run("request reply", func(t *testing.T) {
		client, err := DialTCP(server.listener.Addr().String(), ClientConfig{})
		if err != nil {
			t.Fatal(err)
		}
		defer client.Close()

		go client.Wait()

		reply, err := client.RequestReply(context.TODO(), &Message{
			Body: []byte{1, 2, 3, 4, 5, 6},
		})

		if err != nil {
			t.Error(err)
		}

		assert.Equal(t, hex.EncodeToString([]byte{1, 2, 3, 4, 5, 6}), string(reply.Body))
	})

	t.Run("connect again", func(t *testing.T) {
		client, err := DialTCP(server.listener.Addr().String(), ClientConfig{})
		if err != nil {
			t.Fatal(err)
		}
		defer client.Close()

		go client.Wait()

		reply, err := client.RequestReply(context.TODO(), &Message{
			Body: []byte{6, 5, 4, 3, 2, 1},
		})

		if err != nil {
			t.Error(err)
		}

		assert.Equal(t, hex.EncodeToString([]byte{6, 5, 4, 3, 2, 1}), string(reply.Body))
	})
}

func TestTlsServer(t *testing.T) {

	certPem := []byte(`-----BEGIN CERTIFICATE-----
MIIBhTCCASugAwIBAgIQIRi6zePL6mKjOipn+dNuaTAKBggqhkjOPQQDAjASMRAw
DgYDVQQKEwdBY21lIENvMB4XDTE3MTAyMDE5NDMwNloXDTE4MTAyMDE5NDMwNlow
EjEQMA4GA1UEChMHQWNtZSBDbzBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABD0d
7VNhbWvZLWPuj/RtHFjvtJBEwOkhbN/BnnE8rnZR8+sbwnc/KhCk3FhnpHZnQz7B
5aETbbIgmuvewdjvSBSjYzBhMA4GA1UdDwEB/wQEAwICpDATBgNVHSUEDDAKBggr
BgEFBQcDATAPBgNVHRMBAf8EBTADAQH/MCkGA1UdEQQiMCCCDmxvY2FsaG9zdDo1
NDUzgg4xMjcuMC4wLjE6NTQ1MzAKBggqhkjOPQQDAgNIADBFAiEA2zpJEPQyz6/l
Wf86aX6PepsntZv2GYlA5UpabfT2EZICICpJ5h/iI+i341gBmLiAFQOyTDT+/wQc
6MF9+Yw1Yy0t
-----END CERTIFICATE-----`)
	keyPem := []byte(`-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIIrYSSNQFaA2Hwf1duRSxKtLYX5CB04fSeQ6tF1aY/PuoAoGCCqGSM49
AwEHoUQDQgAEPR3tU2Fta9ktY+6P9G0cWO+0kETA6SFs38GecTyudlHz6xvCdz8q
EKTcWGekdmdDPsHloRNtsiCa697B2O9IFA==
-----END EC PRIVATE KEY-----`)
	cert, err := tls.X509KeyPair(certPem, keyPem)
	if err != nil {
		t.Fatal(err)
	}

	serverCertCallback := false
	clientCertCallback := false

	s, err := ListenTCP("127.0.0.1:0", ServerConfig{
		Config: Config{
			TLS: &tls.Config{
				Certificates: []tls.Certificate{cert},
				ClientAuth:   tls.RequestClientCert,
				VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
					assert.Equal(t, cert.Certificate, rawCerts)
					serverCertCallback = true

					return nil
				},
			},
		},
		MessageCallback: func(c Conn, bam *ByteArrayMessage) {
			if bam.Headers.Actor != MessageActorTypeGenericTestActor {
				return
			}

			assert.Equal(t, "TEST", bam.Headers.Action)
			assert.Equal(t, []byte{1, 2, 3, 4}, bam.Body)

			msg := &Message{}
			msg.Headers.RelatesTo = bam.Headers.Id
			msg.Headers.Action = "TEST_REPLY"
			msg.Headers.Actor = MessageActorTypeGenericTestActor
			msg.Body = []byte{4, 3, 2, 1}
			err := c.SendOneWay(msg)
			if err != nil {
				t.Fatal(err)
			}
		},
	})
	if err != nil {
		t.Error(err)
	}

	go s.Serve()

	{
		c, err := DialTCP(s.listener.Addr().String(), ClientConfig{
			Config: Config{
				TLS: &tls.Config{
					InsecureSkipVerify: true,
					Certificates:       []tls.Certificate{cert},
					VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
						assert.Equal(t, cert.Certificate, rawCerts)
						clientCertCallback = true
						return nil
					},
				},
			},
		})
		if err != nil {
			t.Error(err)
		}

		go c.Wait()

		{
			msg := &Message{}
			msg.Headers.Action = "TEST"
			msg.Headers.Actor = MessageActorTypeGenericTestActor
			msg.Body = []byte{1, 2, 3, 4}
			reply, err := c.RequestReply(context.Background(), msg)
			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, "TEST_REPLY", reply.Headers.Action)
			assert.Equal(t, MessageActorTypeGenericTestActor, reply.Headers.Actor)
			assert.Equal(t, []byte{4, 3, 2, 1}, reply.Body)
		}

		assert.True(t, serverCertCallback)
		assert.True(t, clientCertCallback)
	}
}
