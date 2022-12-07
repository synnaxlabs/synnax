package security

import (
	"crypto"
	"crypto/tls"
	"crypto/x509"
	"github.com/synnaxlabs/synnax/pkg/security/cert"
)

type TLSService interface {
	TLS() *tls.Config
}

type KeyService interface {
	Key() crypto.PrivateKey
}

type Service interface {
	TLSService
	KeyService
}

type service struct {
	loader  *cert.Loader
	tls     *tls.Certificate
	rootCAs *x509.CertPool
}

type ServiceConfig struct {
	cert.LoaderConfig
}

func NewService(loader *cert.Loader) (Service, error) {
	c, err := loader.LoadNodeTLSCertAndKey()
	if err != nil {
		return nil, err
	}
	ca, _, err := loader.LoadCACertAndKey()
	if err != nil {
		return nil, err
	}
	s := &service{loader: loader, tls: c, rootCAs: x509.NewCertPool()}
	s.rootCAs.AddCert(ca)
	return s, nil
}

// TLS implements the TLSService interface
func (s *service) TLS() *tls.Config {
	return &tls.Config{
		GetCertificate:       s.getCert,
		RootCAs:              s.rootCAs,
		ClientAuth:           tls.VerifyClientCertIfGiven,
		ClientCAs:            s.rootCAs,
		GetClientCertificate: s.getClientCert,
	}
}

// Key implements the KeyService interface
func (s *service) Key() crypto.PrivateKey { return s.tls.PrivateKey }

func (s *service) getClientCert(info *tls.CertificateRequestInfo) (*tls.Certificate, error) {
	return s.tls, nil
}

func (s *service) getCert(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
	return s.tls, nil
}
