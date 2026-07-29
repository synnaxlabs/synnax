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
	xfs "github.com/synnaxlabs/x/io/fs"
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

// Validate implements config.Config with the per-listener rules: a non-empty address
// and PEM paths only on the file source. Setting a cert or key path on any other source
// is a silent misconfiguration; remaining per-source rules are enforced at build time.
func (c Config) Validate() error {
	v := validate.New("listener")
	validate.NotEmptyString(v, "address", c.Address)
	v.Ternaryf(
		"cert",
		c.Cert.Source != file.SourceType && (c.Cert.Cert != "" || c.Cert.Key != ""),
		"the %q certificate source does not read a cert or key path",
		c.Cert.Source,
	)
	return v.Error()
}

// source builds the certificate source this listener selects, handing each strategy only
// its own inputs: file gets the PEM paths (falling back to the node certificate when the
// listener sets none), auto and tailscale get the listener host. The node-wide filesystem
// and CA authority are injected, not read from the listener config.
func (c Config) source(fs xfs.FS, ca *cert.Factory) (cert.Source, error) {
	switch c.Cert.Source {
	case file.SourceType:
		certPath, keyPath := c.Cert.Cert, c.Cert.Key
		if certPath == "" && keyPath == "" {
			certPath, keyPath = ca.AbsoluteNodeCertPath(), ca.AbsoluteNodeKeyPath()
		}
		return file.NewSource(fs, certPath, keyPath)
	case auto.SourceType:
		return auto.NewSource(ca, c.Address)
	case tailscale.SourceType:
		return tailscale.NewSource(tailscale.DefaultClient(), c.Address.Host())
	default:
		return nil, errors.Newf("unknown certificate source %q", c.Cert.Source)
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
		v.Exec(func() error { return validate.PathedError(c.Validate(), field) })
		v.Ternaryf(
			field+".address",
			seen.Contains(c.Address),
			"duplicate listener address %q",
			c.Address,
		)
		seen.Add(c.Address)
		if c.Advertise {
			advertised++
		}
	}
	v.Ternary("advertise", advertised > 1, "at most one listener may advertise")
	return v.Error()
}

// ValidateAdvertiseSource rejects a Tailscale source on the advertised listener. Peers
// dial the advertised address and verify the certificate against the Core CA, which a
// public-CA Tailscale certificate cannot satisfy. It applies only in secure mode, so the
// caller gates it on the insecure flag.
func (cs Configs) ValidateAdvertiseSource() error {
	v := validate.New("listeners")
	if len(cs) == 0 {
		return v.Error()
	}
	v.Ternary(
		"advertise",
		cs.advertised().Cert.Source == tailscale.SourceType,
		"advertised listener cannot use the Tailscale source; peers verify certificates against the Core CA",
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

// Resolve resolves each listener config into a server.Listener, backing every secure
// listener with a TLS config from its certificate source. fc supplies the node-wide
// filesystem and CA authority that the file and auto sources draw on.
func (cs Configs) Resolve(
	p security.Provider,
	fc cert.FactoryConfig,
	insecure bool,
) ([]server.Listener, error) {
	out := make([]server.Listener, len(cs))
	if insecure {
		for i, c := range cs {
			out[i] = server.Listener{Address: c.Address}
		}
		return out, nil
	}
	fs := fc.FS
	if fs == nil {
		fs = xfs.Default
	}
	ca, err := cert.NewFactory(fc)
	if err != nil {
		return nil, err
	}
	advertised := cs.advertised().Address
	for i, c := range cs {
		src, err := c.source(fs, ca)
		if err != nil {
			return nil, err
		}
		if c.Address == advertised {
			if err = p.VerifyCoreCert(src, advertised.Host()); err != nil {
				return nil, errors.Wrapf(
					err,
					"advertised listener %q must serve a certificate the Core CA signs for that host; peers cannot verify it otherwise",
					c.Address,
				)
			}
		}
		out[i] = server.Listener{Address: c.Address, TLS: p.TLSConfigFor(src)}
	}
	return out, nil
}

// Parse reads the polymorphic listen configuration. A scalar address yields a single
// file-source listener serving the node certificate, identical to the
// pre-multi-listener behavior. A list yields one listener per entry.
func Parse() (Configs, error) {
	switch v := viper.Get(FlagListen).(type) {
	case string:
		return Configs{scalar(v)}, nil
	case nil:
		return Configs{scalar(viper.GetString(FlagListen))}, nil
	case []any:
		return parseList(v)
	default:
		return nil, errors.Newf("invalid listen configuration of type %T", v)
	}
}

func scalar(addr string) Config {
	return Config{
		Address:   address.Address(addr),
		Cert:      CertConfig{Source: file.SourceType},
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
			return nil, errors.Newf("listen[%d] must be an object", i)
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
// they locate the Core CA and node outbound identity, which every listener still
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
		"%v cannot be combined with a listen list; set each listener's cert block instead",
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
