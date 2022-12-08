package security

import (
	"crypto"
	"crypto/tls"
	"crypto/x509"
	"github.com/synnaxlabs/synnax/pkg/security/cert"
	"github.com/synnaxlabs/x/config"
)

// TLSProvider provides the node's TLS configuration for services that require it.
type TLSProvider interface {
	// TLS returns the node's TLS configuration. It's important to note that although
	// the reference returned by this method will remain constant, the underlying
	// configuration may change its behavior over time (e.g. when the node's TLS
	// certificate is rotated).
	TLS() *tls.Config
}

// KeyProvider provides information of private keys for the node.
type KeyProvider interface {
	// NodeSecret returns the private key of the node's TLS certificate.
	NodeSecret() crypto.PrivateKey
}

// Provider provides security information and services for the node. It's important to note
// that Provider itself does not implement any security mechanisms, but rather provides
// configuration and information for other components to implement them.
type Provider interface {
	TLSProvider
	KeyProvider
}

type provider struct {
	loader   *cert.Loader
	tls      *tls.Certificate
	certPool *x509.CertPool
}

type ProviderConfig struct {
	cert.LoaderConfig
}

var (
	_ config.Config[ProviderConfig] = ProviderConfig{}
	// DefaultServiceConfig is the default configuration for the security provider.
	DefaultServiceConfig = ProviderConfig{
		LoaderConfig: cert.DefaultLoaderConfig,
	}
)

// Override implements Config.
func (s ProviderConfig) Override(other ProviderConfig) ProviderConfig {
	s.LoaderConfig = s.LoaderConfig.Override(other.LoaderConfig)
	return s
}

// Validate implements Config.
func (s ProviderConfig) Validate() error { return s.LoaderConfig.Validate() }

// NewProvider opens a new security Provider using the given configuration.
func NewProvider(configs ...ProviderConfig) (Provider, error) {
	cfg, err := config.OverrideAndValidate(DefaultServiceConfig, configs...)
	l, err := cert.NewLoader(cfg.LoaderConfig)
	if err != nil {
		return nil, err
	}
	c, err := l.LoadNodeTLS()
	if err != nil {
		return nil, err
	}
	cas, err := l.LoadCAs()
	if err != nil {
		return nil, err
	}
	s := &provider{loader: l, tls: c, certPool: x509.NewCertPool()}
	for _, ca := range cas {
		s.certPool.AddCert(ca)
	}
	return s, nil
}

// defaultCipherSuites is the default list of cipher suites used by the node. Order of
// the list is irrelevant, as prioritization is hard-coded in the go standard library.
// This is a subset of the cipher suites supported by the go standard library[1], which
// are also marked as recommended by IETF[2]. Mozilla recommends the same suites because
// athey provide forward secrecy and authentication[3].
//
// Thanks to the CockroachDB team for this list of cipher suites[4].

//
// [1]: https://github.com/golang/go/blob/4aa1efed4853ea067d665a952eee77c52faac774/src/crypto/tls/cipher_suites.go#L215-L270
// [2]: https://www.iana.org/assignments/tls-parameters/tls-parameters.xhtml#tls-parameters-4
// [3]: https://wiki.mozilla.org/Security/Server_Side_TLS#Intermediate_compatibility_.28recommended.29
// [4]: https://github.com/cockroachdb/cockroach/blob/a9fcbd15b24bcf09e139785921be8d9ff8cad729/pkg/security/tls.go#L106
var defaultCipherSuites = []uint16{
	tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
	tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
	tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
	tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
	tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
	tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
}

// TLS implements the TLSProvider interface
func (s *provider) TLS() *tls.Config {
	return &tls.Config{
		GetCertificate:       s.getCert,
		RootCAs:              s.certPool,
		ClientAuth:           tls.VerifyClientCertIfGiven,
		ClientCAs:            s.certPool,
		GetClientCertificate: s.getClientCert,
		CipherSuites:         defaultCipherSuites,
		MinVersion:           tls.VersionTLS12,
	}
}

// NodeSecret implements the KeyProvider interface
func (s *provider) NodeSecret() crypto.PrivateKey { return s.tls.PrivateKey }

func (s *provider) getClientCert(info *tls.CertificateRequestInfo) (*tls.Certificate, error) {
	return s.tls, nil
}

func (s *provider) getCert(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
	return s.tls, nil
}
