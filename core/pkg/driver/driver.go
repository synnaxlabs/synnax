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
	// Insecure sets whether not to use TLS for communication. If insecure
	// is set to true, CACertPath, ClientCertFile, and ClientKeyFile are ignored.
	Insecure *bool `json:"insecure"`
	// Enabled is used to enable or disable the embedded driver.
	Enabled *bool `json:"enabled"`
	// Debug sets whether to enable debug logging.
	Debug *bool `json:"debug"`
	// Instrumentation is used for logging, tracing, and metrics.
	alamos.Instrumentation
	// Username sets the username to authenticate to the cluster with.
	Username string `json:"username"`
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
	// Address is the reachable address of the cluster for the driver to connect to.
	Address address.Address `json:"address"`
	// Password sets the password to authenticate to the cluster with.
	Password string `json:"password"`
	// ParentDirname is the parent directory in which the driver will create a 'driver'
	// directory to extract and execute the driver binary and extract configuration files
	// into.
	ParentDirname string `json:"parent_dirname"`
	// BinaryPath overrides the embedded driver binary with an external path.
	// Intended for testing only.
	BinaryPath string `json:"-"`
	// Integrations define which device integrations are enabled.
	Integrations []string `json:"integrations"`
	// StartTimeout sets the maximum acceptable time to wait for the driver to bootup
	// successfully before timing out and returning a failed startup error.
	StartTimeout time.Duration `json:"start_timeout"`
	// StopTimeout is the time to wait for the driver to exit gracefully
	// after sending STOP before escalating to a forceful kill.
	StopTimeout time.Duration `json:"stop_timeout"`
	// TaskOpTimeout sets the duration before reporting stuck task operations.
	TaskOpTimeout time.Duration `json:"task_op_timeout"`
	// TaskPollInterval sets the interval between task timeout checks.
	TaskPollInterval time.Duration `json:"task_poll_interval"`
	// TaskShutdownTimeout sets the max time to wait for task workers during shutdown.
	TaskShutdownTimeout time.Duration `json:"task_shutdown_timeout"`
	// RackKey is the key of the rack that the driver should assume the identity of.
	RackKey rack.Key `json:"rack_key"`
	// ClusterKey is the key of the current cluster.
	ClusterKey uuid.UUID `json:"cluster_key"`
	// TaskWorkerCount sets the number of worker threads for task operations.
	TaskWorkerCount uint8 `json:"task_worker_count"`
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
		"manager": map[string]any{
			"op_timeout":       c.TaskOpTimeout.Seconds(),
			"poll_interval":    c.TaskPollInterval.Seconds(),
			"shutdown_timeout": c.TaskShutdownTimeout.Seconds(),
			"worker_count":     c.TaskWorkerCount,
		},
		"integrations": c.Integrations,
		"debug":        *c.Debug,
	}
}

var (
	_               config.Config[Config] = Config{}
	AllIntegrations                       = []string{
		"arc",
		"labjack",
		"modbus",
		"ni",
		"opc",
		"sequence",
		"ethercat",
	}
	DefaultConfig = Config{
		Integrations:        []string{},
		Enabled:             new(true),
		Debug:               new(false),
		StartTimeout:        time.Second * 10,
		StopTimeout:         10 * time.Second,
		TaskOpTimeout:       time.Second * 60,
		TaskPollInterval:    time.Second * 1,
		TaskShutdownTimeout: time.Second * 30,
		TaskWorkerCount:     4,
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
	c.BinaryPath = override.String(c.BinaryPath, other.BinaryPath)
	c.TaskOpTimeout = override.Numeric(c.TaskOpTimeout, other.TaskOpTimeout)
	c.TaskPollInterval = override.Numeric(c.TaskPollInterval, other.TaskPollInterval)
	c.TaskShutdownTimeout = override.Numeric(c.TaskShutdownTimeout, other.TaskShutdownTimeout)
	c.TaskWorkerCount = override.Numeric(c.TaskWorkerCount, other.TaskWorkerCount)
	c.StopTimeout = override.Numeric(c.StopTimeout, other.StopTimeout)
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
	validate.InBounds(v, "task_worker_count", c.TaskWorkerCount, 1, 64)
	return v.Error()
}
