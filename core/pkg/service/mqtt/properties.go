// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package mqtt

import (
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/synnaxlabs/synnax/pkg/service/device"
	"github.com/synnaxlabs/synnax/pkg/service/mqtt/sparkplug"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// Make is the make of a device that is an MQTT broker.
const Make = "mqtt"

const (
	defaultPort       = 1883
	defaultSecurePort = 8883
	defaultKeepAlive  = 30 * time.Second
)

// SparkplugProperties are the Sparkplug B settings of a broker device.
type SparkplugProperties struct {
	// HostID makes the Core a Sparkplug B host application with this ID: it publishes
	// a retained STATE message. Empty keeps the Core passive.
	HostID string `json:"host_id"`
	// Groups limits the Sparkplug B subscription to these group IDs. Empty takes
	// every group.
	Groups []string `json:"groups"`
}

// Properties are the connection settings stored in the properties of a broker
// device. The host is the location of the device.
type Properties struct {
	// CAFile is the path on the Core host to the CA certificates that verify the
	// broker. Empty uses the system roots.
	CAFile string `json:"ca_file"`
	// CertFile is the path on the Core host to the client certificate.
	CertFile string `json:"cert_file"`
	// KeyFile is the path on the Core host to the key of the client certificate.
	KeyFile  string `json:"key_file"`
	Username string `json:"username"`
	Password string `json:"password"`
	// ClientID is the MQTT client ID. Empty derives one from the device key.
	ClientID  string              `json:"client_id"`
	Sparkplug SparkplugProperties `json:"sparkplug"`
	// Port is the TCP port of the broker. Zero selects 1883, or 8883 when Secure.
	Port int `json:"port"`
	// KeepAlive is the MQTT keep-alive interval in seconds. Zero selects 30.
	KeepAlive int `json:"keep_alive"`
	// Secure is true when the connection uses TLS.
	Secure bool `json:"secure"`
	// VerificationSkipped is true when the client accepts any broker certificate.
	VerificationSkipped bool `json:"verification_skipped"`
}

// clientConfig is what a client needs to connect to one broker.
type clientConfig struct {
	tls *tls.Config
	// tlsID identifies the TLS settings, because a tls.Config does not compare.
	tlsID string
	url   string
	// hostID is the Sparkplug B host application ID. Empty for a passive host.
	hostID    string
	clientID  string
	username  string
	password  string
	keepAlive time.Duration
}

// newClientConfig parses and validates the properties of dev. Errors wrap
// validate.ErrValidation, except for certificate files that fail to load.
func newClientConfig(dev device.Device) (clientConfig, error) {
	var props Properties
	if err := dev.Properties.Unmarshal(&props); err != nil {
		return clientConfig{}, errors.Wrapf(
			validate.ErrValidation, "invalid broker properties: %s", err.Error(),
		)
	}
	v := validate.New("mqtt.properties")
	v.NotEmptyString("location", dev.Location)
	v.Ternary("port", props.Port < 0 || props.Port > 65535, "must be 0 to 65535")
	v.Ternary("keep_alive", props.KeepAlive < 0, "must not be negative")
	v.Ternary(
		"key_file",
		(props.CertFile == "") != (props.KeyFile == ""),
		"cert_file and key_file must be set together",
	)
	if err := v.Error(); err != nil {
		return clientConfig{}, errors.Wrapf(
			validate.ErrValidation, "invalid broker properties: %s", err.Error(),
		)
	}
	if props.Sparkplug.HostID != "" {
		if err := sparkplug.ValidateID(
			"sparkplug.host_id", props.Sparkplug.HostID,
		); err != nil {
			return clientConfig{}, errors.Wrapf(
				validate.ErrValidation, "invalid broker properties: %s", err.Error(),
			)
		}
	}
	cfg := clientConfig{
		hostID:    props.Sparkplug.HostID,
		clientID:  props.ClientID,
		username:  props.Username,
		password:  props.Password,
		keepAlive: time.Duration(props.KeepAlive) * time.Second,
	}
	if cfg.clientID == "" {
		cfg.clientID = deriveClientID(dev.Key)
	}
	if cfg.keepAlive == 0 {
		cfg.keepAlive = defaultKeepAlive
	}
	port, scheme := props.Port, "tcp"
	if props.Secure {
		scheme = "ssl"
		tlsCfg, err := newTLSConfig(props)
		if err != nil {
			return clientConfig{}, err
		}
		cfg.tls = tlsCfg
		cfg.tlsID = strings.Join([]string{
			props.CAFile,
			props.CertFile,
			props.KeyFile,
			strconv.FormatBool(props.VerificationSkipped),
		}, "|")
	}
	if port == 0 {
		port = defaultPort
		if props.Secure {
			port = defaultSecurePort
		}
	}
	cfg.url = scheme + "://" + net.JoinHostPort(dev.Location, strconv.Itoa(port))
	return cfg, nil
}

// deriveClientID returns a stable client ID of 23 characters, the longest that every
// MQTT 3.1.1 broker must accept.
func deriveClientID(seed string) string {
	sum := sha256.Sum256([]byte(seed))
	return "synnax-" + hex.EncodeToString(sum[:8])
}

func newTLSConfig(props Properties) (*tls.Config, error) {
	cfg := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: props.VerificationSkipped,
	}
	if props.CAFile != "" {
		pem, err := os.ReadFile(props.CAFile)
		if err != nil {
			return nil, errors.Wrap(err, "failed to read the CA file")
		}
		cfg.RootCAs = x509.NewCertPool()
		if !cfg.RootCAs.AppendCertsFromPEM(pem) {
			return nil, errors.Wrapf(
				validate.ErrValidation,
				"CA file %s holds no PEM certificates", props.CAFile,
			)
		}
	}
	if props.CertFile != "" {
		cert, err := tls.LoadX509KeyPair(props.CertFile, props.KeyFile)
		if err != nil {
			return nil, errors.Wrap(err, "failed to load the client certificate")
		}
		cfg.Certificates = []tls.Certificate{cert}
	}
	return cfg, nil
}
