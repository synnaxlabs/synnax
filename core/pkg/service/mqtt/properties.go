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

	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
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

// sparkplugProperties are the Sparkplug B settings of a broker device.
type sparkplugProperties struct {
	// HostID makes the Core a Sparkplug B host application with this ID: it publishes
	// a retained STATE message. Empty keeps the Core passive.
	HostID string `json:"host_id"`
	// Groups are the group IDs that a browse for edge nodes listens to when it names no
	// group. Empty listens to every group.
	Groups []string `json:"groups"`
}

// validate checks that every ID of p can be a level of a Sparkplug B topic.
func (p sparkplugProperties) validate() error {
	if p.HostID != "" {
		if err := sparkplug.ValidateID("sparkplug.host_id", p.HostID); err != nil {
			return err
		}
	}
	for _, g := range p.Groups {
		if err := sparkplug.ValidateID("sparkplug.groups", g); err != nil {
			return err
		}
	}
	return nil
}

// properties are the connection settings stored in the properties of a broker device.
// The host is the location of the device.
type properties struct {
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
	Sparkplug sparkplugProperties `json:"sparkplug"`
	// Port is the TCP port of the broker, from 1 to 65535. Nil selects 1883, or 8883
	// when Secure.
	Port *int `json:"port"`
	// KeepAlive is the MQTT keep-alive interval in seconds, from 1 to 65535. Nil
	// selects
	// 30.
	KeepAlive *int `json:"keep_alive"`
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

// parseProperties decodes the properties of dev. The error wraps
// validate.ErrValidation.
func parseProperties(dev device.Device) (properties, error) {
	var props properties
	if err := dev.Properties.Unmarshal(&props); err != nil {
		return properties{}, errors.Wrapf(
			validate.ErrValidation, "invalid broker properties: %s", err.Error(),
		)
	}
	return props, nil
}

// newClientConfig parses and validates the properties of dev, and loads the TLS files
// that they name. Errors wrap validate.ErrValidation, except for certificate files that
// fail to load.
func newClientConfig(dev device.Device) (clientConfig, error) {
	props, err := parseProperties(dev)
	if err != nil {
		return clientConfig{}, err
	}
	return newClientConfigOf(dev, props)
}

// newClientConfigOf validates props, the decoded properties of dev, and returns the
// config of its client.
func newClientConfigOf(dev device.Device, props properties) (clientConfig, error) {
	cfg, err := newClientSettings(dev, props)
	if err != nil {
		return clientConfig{}, err
	}
	if cfg.tls, err = loadTLS(props); err != nil {
		return clientConfig{}, err
	}
	return cfg, nil
}

// newClientSettings validates props, the decoded properties of dev, and returns the
// config of its client with no TLS config. It reads no files.
func newClientSettings(dev device.Device, props properties) (clientConfig, error) {
	v := validate.New("mqtt.properties")
	v.NotEmptyString("location", dev.Location)
	v.Ternary(
		"port",
		props.Port != nil && (*props.Port < 1 || *props.Port > 65535),
		"must be 1 to 65535",
	)
	v.Ternary(
		"keep_alive",
		props.KeepAlive != nil && (*props.KeepAlive < 1 || *props.KeepAlive > 65535),
		"must be 1 to 65535",
	)
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
	if err := props.Sparkplug.validate(); err != nil {
		return clientConfig{}, errors.Wrap(err, "invalid broker properties")
	}
	cfg := clientConfig{
		hostID:    props.Sparkplug.HostID,
		clientID:  props.ClientID,
		username:  props.Username,
		password:  props.Password,
		keepAlive: defaultKeepAlive,
	}
	if cfg.clientID == "" {
		cfg.clientID = deriveClientID(dev.Key)
	}
	if props.KeepAlive != nil {
		cfg.keepAlive = time.Duration(*props.KeepAlive) * time.Second
	}
	port, scheme := defaultPort, "tcp"
	if props.Secure {
		port, scheme = defaultSecurePort, "ssl"
		cfg.tlsID = strings.Join([]string{
			props.CAFile,
			props.CertFile,
			props.KeyFile,
			strconv.FormatBool(props.VerificationSkipped),
		}, "|")
	}
	if props.Port != nil {
		port = *props.Port
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

// loadTLS returns the TLS config of props, or nil for a connection with no TLS.
func loadTLS(props properties) (*tls.Config, error) {
	if !props.Secure {
		return nil, nil
	}
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
