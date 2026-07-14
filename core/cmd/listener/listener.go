// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package listener turns the polymorphic listen configuration into resolved
// server.Listeners, each backed by its own certificate source.
package listener

import (
	"fmt"

	"github.com/spf13/viper"
	cmdcert "github.com/synnaxlabs/synnax/cmd/cert"
	"github.com/synnaxlabs/synnax/pkg/security"
	"github.com/synnaxlabs/synnax/pkg/security/cert"
	"github.com/synnaxlabs/synnax/pkg/security/cert/auto"
	"github.com/synnaxlabs/synnax/pkg/security/cert/file"
	"github.com/synnaxlabs/synnax/pkg/security/cert/tailscale"
	"github.com/synnaxlabs/synnax/pkg/server"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

// CertConfig is the parsed certificate configuration for a single listener.
type CertConfig struct {
	Source string
	Cert   string
	Key    string
}

// Config is the parsed configuration for a single listener. It is resolved into a
// server.Listener once the security provider is available.
type Config struct {
	Address   address.Address
	Cert      CertConfig
	Advertise bool
	Name      string
}

// Validate implements config.Config. Per-source certificate rules are enforced when the
// source is built, not here.
func (c Config) Validate() error {
	v := validate.New("listener")
	validate.NotEmptyString(v, "address", c.Address)
	return v.Error()
}

// sourceConfig builds the cert.SourceConfig for the listener, drawing CA material from
// the node's cert factory configuration.
func (c Config) sourceConfig(fc cert.FactoryConfig) cert.SourceConfig {
	return cert.SourceConfig{
		Type:       c.Cert.Source,
		Cert:       c.Cert.Cert,
		Key:        c.Cert.Key,
		Address:    c.Address,
		CertsDir:   fc.CertsDir,
		CAKeyPath:  fc.CAKeyPath,
		CACertPath: fc.CACertPath,
		KeySize:    fc.KeySize,
	}
}

// Configs is a set of listener configurations, the whole set a Core binds.
type Configs []Config

// Validate implements config.Config, applying the cross-listener rules: at least one
// listener, unique addresses, and at most one advertised listener.
func (cs Configs) Validate() error {
	v := validate.New("listeners")
	if len(cs) == 0 {
		v.Ternary("listeners", true, "at least one listener is required")
		return v.Error()
	}
	advertised := 0
	seen := make(set.Set[address.Address], len(cs))
	for i, c := range cs {
		field := fmt.Sprintf("listeners[%d]", i)
		validate.NotEmptyString(v, field+".address", c.Address)
		v.Ternaryf(field+".address", seen.Contains(c.Address), "duplicate listener address %q", c.Address)
		seen.Add(c.Address)
		if c.Advertise {
			advertised++
		}
	}
	v.Ternary("advertise", advertised > 1, "at most one listener may advertise")
	return v.Error()
}

// ValidateAdvertiseSource rejects a tailscale source on the advertised listener. Peers
// dial the advertised address and verify the certificate against the cluster CA, which a
// public-CA tailscale certificate cannot satisfy. It applies only in secure mode, so the
// caller gates it on the insecure flag.
func (cs Configs) ValidateAdvertiseSource() error {
	v := validate.New("listeners")
	if len(cs) == 0 {
		return v.Error()
	}
	v.Ternary(
		"advertise",
		cs.advertised().Cert.Source == tailscale.SourceType,
		"advertised listener cannot use the tailscale source; peers verify certificates against the cluster CA",
	)
	return v.Error()
}

// advertised returns the listener peers reach: the one marked advertise, or the first
// when none is marked. It assumes a non-empty set.
func (cs Configs) advertised() Config {
	for _, c := range cs {
		if c.Advertise {
			return c
		}
	}
	return cs[0]
}

// AdvertiseAddress returns the address peers use to reach this node.
func (cs Configs) AdvertiseAddress() address.Address { return cs.advertised().Address }

// sourceFactories is the set of certificate sources a listener may select. This package
// owns the list so implementation-specific sources (tailscale) never enter the base cert
// package.
var sourceFactories = []cert.SourceFactory{
	file.Factory{},
	auto.Factory{},
	tailscale.Factory{},
}

// Resolve resolves each listener config into a server.Listener, backing every secure
// listener with a TLS config from its certificate source.
func (cs Configs) Resolve(
	p security.Provider,
	fc cert.FactoryConfig,
	insecure bool,
) ([]server.Listener, error) {
	out := make([]server.Listener, len(cs))
	advertised := cs.advertised().Address
	for i, c := range cs {
		l := server.Listener{Address: c.Address}
		if !insecure {
			src, err := cert.Resolve(sourceFactories, c.sourceConfig(fc))
			if err != nil {
				return nil, err
			}
			if c.Address == advertised {
				if err = p.VerifyClusterCert(src, advertised.Host()); err != nil {
					return nil, errors.Wrapf(err, "[listener] - advertised listener %q must serve a certificate the cluster CA signs for that host; peers cannot verify it otherwise", c.Address)
				}
			}
			l.TLS = p.TLSConfigFor(src)
		}
		out[i] = l
	}
	return out, nil
}

// Parse reads the polymorphic listen configuration. A scalar address yields a single
// file-source listener over the node certificate paths, identical to the
// pre-multi-listener behavior. A list yields one listener per entry.
func Parse(fc cert.FactoryConfig) (Configs, error) {
	switch v := viper.Get(FlagListen).(type) {
	case string:
		return Configs{scalar(v, fc)}, nil
	case nil:
		return Configs{scalar(viper.GetString(FlagListen), fc)}, nil
	case []any:
		return parseList(v)
	default:
		return nil, errors.Newf("[listener] - invalid listen configuration of type %T", v)
	}
}

func scalar(addr string, fc cert.FactoryConfig) Config {
	return Config{
		Address: address.Address(addr),
		Cert: CertConfig{
			Source: file.SourceType,
			Cert:   fc.AbsoluteNodeCertPath(),
			Key:    fc.AbsoluteNodeKeyPath(),
		},
		Advertise: true,
	}
}

func parseList(items []any) (Configs, error) {
	if err := rejectGlobalCertFlags(); err != nil {
		return nil, err
	}
	configs := make(Configs, len(items))
	for i, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			return nil, errors.Newf("[listener] - listen[%d] must be an object", i)
		}
		c := asMap(m, "cert")
		configs[i] = Config{
			Address: address.Address(asString(m, "address")),
			Cert: CertConfig{
				Source: asString(c, "source"),
				Cert:   asString(c, "cert"),
				Key:    asString(c, "key"),
			},
			Advertise: asBool(m, "advertise"),
			Name:      asString(m, "name"),
		}
	}
	return configs, nil
}

// rejectGlobalCertFlags enforces that a listen list is not combined with a global flag
// that configures the single node certificate. --certs-dir and the CA flags stay legal:
// they locate the cluster CA and node outbound identity, which every listener still
// needs for gossip.
func rejectGlobalCertFlags() error {
	var offenders []string
	if viper.GetBool(cmdcert.FlagAutoCert) {
		offenders = append(offenders, "--"+cmdcert.FlagAutoCert)
	}
	if viper.GetString(cmdcert.FlagNodeCert) != "" {
		offenders = append(offenders, "--"+cmdcert.FlagNodeCert)
	}
	if viper.GetString(cmdcert.FlagNodeKey) != "" {
		offenders = append(offenders, "--"+cmdcert.FlagNodeKey)
	}
	if len(offenders) == 0 {
		return nil
	}
	return errors.Newf(
		"[listener] - %v cannot be combined with a listen list; set each listener's cert block instead",
		offenders,
	)
}

func asString(m map[string]any, key string) string {
	s, _ := m[key].(string)
	return s
}

func asBool(m map[string]any, key string) bool {
	b, _ := m[key].(bool)
	return b
}

func asMap(m map[string]any, key string) map[string]any {
	sub, _ := m[key].(map[string]any)
	return sub
}
