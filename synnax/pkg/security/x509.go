package security

import (
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"time"
)

const (
	validFrom    = -time.Hour * 24
	validFor     = time.Hour * 24 * 365
	caCommonName = "Synnax CA"
)

func NewBaseX509() (*x509.Certificate, error) {
	sn, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, err
	}

	cert := &x509.Certificate{
		SerialNumber: sn,
		Subject: pkix.Name{
			CommonName: caCommonName,
		},
		NotBefore: time.Now().Add(validFrom),
		NotAfter:  time.Now().Add(validFor),
		KeyUsage:  x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}

	return cert, nil
}
