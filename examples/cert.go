package examples

import (
	"crypto/sha1"
	"crypto/tls"
	"fmt"

	"github.com/github/certstore"
)

func FindCert(thumbprint string) (*tls.Certificate, error) {
	store, err := certstore.Open()
	if err != nil {
		return nil, err
	}
	defer store.Close()

	idents, err := store.Identities()
	if err != nil {
		return nil, err
	}

	found := false

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

			found = true
		}
	}

	if !found {
		return nil, fmt.Errorf("cert not found")
	}

	return &cert, nil
}
