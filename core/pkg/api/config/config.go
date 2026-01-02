// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package config

import (
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// Config is all required configuration parameters and services necessary to instantiate
// the API.
type Config struct {
	alamos.Instrumentation
	Service      *service.Layer
	Distribution *distribution.Layer
}

var (
	_       config.Config[Config] = Config{}
	Default                       = Config{}
)

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("api")
	validate.NotNil(v, "service", c.Service)
	validate.NotNil(v, "distribution", c.Distribution)
	return v.Error()
}

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.Service = override.Nil(c.Service, other.Service)
	c.Distribution = override.Nil(c.Distribution, other.Distribution)
	return c
}
