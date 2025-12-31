// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package migrate

import (
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// Config configures the Run function.
type Config struct {
	alamos.Instrumentation
	// Migrator is the type-specific migrator to run.
	// [REQUIRED]
	Migrator TypedMigrator
	// DB is the gorp database for reading/writing entities.
	// [REQUIRED]
	DB *gorp.DB
}

var (
	_             config.Config[Config] = Config{}
	DefaultConfig                       = Config{}
)

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.Migrator = override.Nil(c.Migrator, other.Migrator)
	c.DB = override.Nil(c.DB, other.DB)
	return c
}

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("jerky.migrate")
	validate.NotNil(v, "Migrator", c.Migrator)
	validate.NotNil(v, "DB", c.DB)
	return v.Error()
}
