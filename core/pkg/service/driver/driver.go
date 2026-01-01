// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package driver

import (
	"io"
	"os/exec"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
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
	// Address is the reachable address of the cluster for the driver to connect to.
	Address address.Address `json:"address"`
	// RackKey is the key of the rack that the driver should assume the identity of.
	RackKey rack.Key `json:"rack_key"`
	// ClusterKey is the key of the current cluster.
	ClusterKey uuid.UUID `json:"cluster_key"`
	// Integrations define which device integrations are enabled.
	Integrations []string `json:"integrations"`
	// Insecure sets whether not to use TLS for communication. If insecure
	// is set to true, CACertPath, ClientCertFile, and ClientKeyFile are ignored.
	Insecure *bool `json:"insecure"`
	// CACertPath sets the path to the CA certificate to use for authenticated/encrypted
	// communication. Not required if the CA is universally recognized or already
	// installed on the users' system.
	CACertPath string `json:"ca_cert_path"`
	// ClientCertFile sets the path to the client cert file to use for authenticated/
	// encrypted communication.
	ClientCertFile string `json:"client_cert_file"`
	// ClientKeyFile sets the secret key file used for authenticated/encrypted communication
	// between the driver and cluster.
	ClientKeyFile string `json:"client_key_file"`
	// Username sets the username to authenticate to the cluster with.
	Username string `json:"username"`
	// Password sets the password to authenticate to the cluster with.
	Password string `json:"password"`
	// Debug sets whether to enable debug logging.
	Debug *bool `json:"debug"`
	// StartTimeout sets the maximum acceptable time to wait for the driver to bootup
	// successfully before timing out and returning a failed startup error.
	StartTimeout time.Duration `json:"start_timeout"`
	// ParentDirname is the parent directory in which the driver will create a 'driver'
	// directory to extract and execute the driver binary and extract configuration files
	// into.
	ParentDirname string `json:"parent_dirname"`
}

func (c Config) format() map[string]any {
	if *c.Insecure {
		c.CACertPath = ""
		c.ClientCertFile = ""
		c.ClientKeyFile = ""
	}
	return map[string]any{
		"connection": map[string]any{
			"host":             c.Address.Host(),
			"port":             c.Address.Port(),
			"username":         c.Username,
			"password":         c.Password,
			"ca_cert_file":     c.CACertPath,
			"client_cert_file": c.ClientCertFile,
			"client_key_file":  c.ClientKeyFile,
		},
		"retry": map[string]any{
			"base_interval": 1,
			"max_retries":   40,
			"scale":         1.1,
		},
		"remote_info": map[string]any{
			"rack_key":    c.RackKey,
			"cluster_key": c.ClusterKey.String(),
		},
		"integrations": c.Integrations,
		"debug":        *c.Debug,
	}
}

var (
	_               config.Config[Config] = Config{}
	AllIntegrations                       = []string{
		"labjack", "modbus", "ni", "opc", "sequence",
	}
	DefaultConfig = Config{
		Integrations: []string{},
		Enabled:      config.True(),
		Debug:        config.False(),
		StartTimeout: time.Second * 10,
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
	c.Insecure = override.Nil(c.Insecure, other.Insecure)
	c.CACertPath = override.String(c.CACertPath, other.CACertPath)
	c.ClientCertFile = override.String(c.ClientCertFile, other.ClientCertFile)
	c.ClientKeyFile = override.String(c.ClientKeyFile, other.ClientKeyFile)
	c.Username = override.String(c.Username, other.Username)
	c.Password = override.String(c.Password, other.Password)
	c.Debug = override.Nil(c.Debug, other.Debug)
	c.StartTimeout = override.Numeric(c.StartTimeout, other.StartTimeout)
	c.ParentDirname = override.String(c.ParentDirname, other.ParentDirname)
	return c
}

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("driver.embedded")
	validate.NotNil(v, "enabled", c.Enabled)
	validate.NotNil(v, "insecure", c.Insecure)
	if v.Error() != nil {
		return v.Error()
	}
	if !*c.Enabled {
		return nil
	}
	validate.NotEmptyString(v, "address", c.Address)
	validate.NotNil(v, "debug", c.Debug)
	validate.NotEmptyString(v, "parent_dirname", c.ParentDirname)
	return v.Error()
}

type Driver struct {
	cfg       Config
	mu        sync.Mutex
	cmd       *exec.Cmd
	shutdown  io.Closer
	stdInPipe io.WriteCloser
	started   chan struct{}
}
