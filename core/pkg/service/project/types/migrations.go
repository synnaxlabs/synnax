// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types

import (
	"github.com/synnaxlabs/synnax/pkg/service/project/types/v1"
	"github.com/synnaxlabs/x/migrate"
)

// LegacyLayoutKVPrefix is the KV key prefix under which the layout staging
// migration stages each project's legacy layout blob.
const LegacyLayoutKVPrefix = v1.LegacyLayoutKVPrefix

// LegacyLayoutKVKey returns the staging KV key holding the legacy layout blob for the
// project with the given key.
func LegacyLayoutKVKey(key Key) []byte { return v1.LegacyLayoutKVKey(key) }

// MigrationsConfig is the configuration for NewMigrations.
type MigrationsConfig = v1.MigrationsConfig

// NewMigrations returns the ordered migration chain for stored projects.
func NewMigrations(cfg MigrationsConfig) []migrate.Migration {
	return v1.NewMigrations(cfg)
}
