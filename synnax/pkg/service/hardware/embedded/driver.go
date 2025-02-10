// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package embedded

import (
	"github.com/google/uuid"
	"io"
	"os/exec"
	"sync"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/service/hardware/rack"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

type Config struct {
	// Instrumentation is used for logging, tracing, and metrics.
	alamos.Instrumentation
	// Enabled is used to enable or disable the embedded driver.
	Enabled *bool `json:"enabled"`
	// Address
	Address        address.Address `json:"address"`
	RackKey        rack.Key        `json:"rack_key"`
	ClusterKey     uuid.UUID       `json:"cluster_key"`
	Integrations   []string        `json:"integrations"`
	CACertPath     string          `json:"ca_cert_path"`
	ClientCertFile string          `json:"client_cert_file"`
	ClientKeyFile  string          `json:"client_key_file"`
	Username       string          `json:"username"`
	Password       string          `json:"password"`
	Debug          *bool           `json:"debug"`
}

func (c Config) format() map[string]interface{} {
	return map[string]interface{}{
		"connection": map[string]interface{}{
			"host":             c.Address.HostString(),
			"port":             c.Address.Port(),
			"username":         c.Username,
			"password":         c.Password,
			"ca_cert_file":     c.CACertPath,
			"client_cert_file": c.ClientCertFile,
			"client_key_file":  c.ClientKeyFile,
		},
		"retry": map[string]interface{}{
			"base_interval": 1,
			"max_retries":   40,
			"scale":         1.1,
		},
		"rack_key":     c.RackKey,
		"cluster_key":  c.ClusterKey.String(),
		"integrations": c.Integrations,
		"debug":        *c.Debug,
	}
}

var (
	_               config.Config[Config] = Config{}
	AllIntegrations                       = []string{"opc", "ni", "labjack", "sequence"}
	DefaultConfig                         = Config{
		Integrations: []string{},
		Enabled:      config.Bool(true),
		Debug:        config.False(),
	}
)

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.Enabled = override.Nil(c.Enabled, other.Enabled)
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.Address = override.String(c.Address, other.Address)
	c.RackKey = override.Numeric(c.RackKey, other.RackKey)
	c.ClusterKey = override.UUID(c.ClusterKey, other.ClusterKey)
	c.Integrations = override.Slice(c.Integrations, other.Integrations)
	c.CACertPath = override.String(c.CACertPath, other.CACertPath)
	c.ClientCertFile = override.String(c.ClientCertFile, other.ClientCertFile)
	c.ClientKeyFile = override.String(c.ClientKeyFile, other.ClientKeyFile)
	c.Username = override.String(c.Username, other.Username)
	c.Password = override.String(c.Password, other.Password)
	c.Debug = override.Nil(c.Debug, other.Debug)
	return c
}

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("driver.embedded")
	validate.NotNil(v, "enabled", c.Enabled)
	if v.Error() != nil {
		return v.Error()
	}
	if !*c.Enabled {
		return nil
	}
	validate.NotEmptyString(v, "address", c.Address)
	validate.NotNil(v, "debug", c.Debug)
	return v.Error()
}

type Driver struct {
	cfg       Config
	mu        sync.Mutex
	cmd       *exec.Cmd
	shutdown  io.Closer
	stdInPipe io.WriteCloser
}
