// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package start

import (
	"fmt"

	"github.com/spf13/viper"
	cmdcert "github.com/synnaxlabs/synnax/cmd/cert"
	"github.com/synnaxlabs/synnax/pkg/security/cert"
	"github.com/synnaxlabs/synnax/pkg/security/cert/file"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// certSpec is the parsed certificate configuration for a single listener.
type certSpec struct {
	source string
	cert   string
	key    string
}

// listenerSpec is the parsed configuration for a single listener. It is resolved into a
// server.Listener once the security provider is available.
type listenerSpec struct {
	address   address.Address
	cert      certSpec
	advertise bool
	name      string
}

// sourceConfig builds the cert.SourceConfig for the listener, drawing CA material from
// the node's cert factory configuration.
func (s listenerSpec) sourceConfig(fc cert.FactoryConfig) cert.SourceConfig {
	return cert.SourceConfig{
		Type:       s.cert.source,
		Cert:       s.cert.cert,
		Key:        s.cert.key,
		Address:    s.address,
		CertsDir:   fc.CertsDir,
		CAKeyPath:  fc.CAKeyPath,
		CACertPath: fc.CACertPath,
		KeySize:    fc.KeySize,
	}
}

// parseListeners reads the polymorphic listen configuration. A scalar address yields a
// single file-source listener over the node certificate paths, identical to the
// pre-multi-listener behavior. A list yields one listener per entry.
func parseListeners(fc cert.FactoryConfig) ([]listenerSpec, error) {
	switch v := viper.Get(FlagListen).(type) {
	case string:
		return []listenerSpec{scalarListener(v, fc)}, nil
	case nil:
		return []listenerSpec{scalarListener(viper.GetString(FlagListen), fc)}, nil
	case []any:
		return parseListenerList(v)
	default:
		return nil, errors.Newf("[start] - invalid listen configuration of type %T", v)
	}
}

func scalarListener(addr string, fc cert.FactoryConfig) listenerSpec {
	return listenerSpec{
		address: address.Address(addr),
		cert: certSpec{
			source: file.SourceType,
			cert:   fc.AbsoluteNodeCertPath(),
			key:    fc.AbsoluteNodeKeyPath(),
		},
		advertise: true,
	}
}

func parseListenerList(items []any) ([]listenerSpec, error) {
	if err := rejectGlobalCertFlags(); err != nil {
		return nil, err
	}
	specs := make([]listenerSpec, len(items))
	for i, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			return nil, errors.Newf("[start] - listen[%d] must be an object", i)
		}
		c := asMap(m, "cert")
		specs[i] = listenerSpec{
			address: address.Address(asString(m, "address")),
			cert: certSpec{
				source: asString(c, "source"),
				cert:   asString(c, "cert"),
				key:    asString(c, "key"),
			},
			advertise: asBool(m, "advertise"),
			name:      asString(m, "name"),
		}
	}
	return specs, nil
}

// rejectGlobalCertFlags enforces that a listen list is not combined with a global flag
// that configures the single node certificate. --certs-dir and the CA flags stay legal:
// they locate the cluster CA and node outbound identity, which every listener still
// needs for gossip.
func rejectGlobalCertFlags() error {
	var offenders []string
	if viper.GetBool(FlagAutoCert) {
		offenders = append(offenders, "--"+FlagAutoCert)
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
		"[start] - %v cannot be combined with a listen list; set each listener's cert block instead",
		offenders,
	)
}

// validateListeners applies the multi-listener validation rules. Per-source certificate
// rules are enforced when the source is built, not here.
func validateListeners(v *validate.Validator, listeners []listenerSpec) {
	if len(listeners) == 0 {
		v.Ternary("listeners", true, "at least one listener is required")
		return
	}
	advertised := 0
	seen := make(map[address.Address]bool, len(listeners))
	for i, l := range listeners {
		field := fmt.Sprintf("listeners[%d]", i)
		validate.NotEmptyString(v, field+".address", l.address)
		v.Ternaryf(field+".address", seen[l.address], "duplicate listener address %q", l.address)
		seen[l.address] = true
		if l.advertise {
			advertised++
		}
	}
	v.Ternary("advertise", advertised > 1, "at most one listener may advertise")
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
