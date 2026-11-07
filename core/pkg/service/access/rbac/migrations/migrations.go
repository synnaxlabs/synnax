// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package migrations

import (
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/builtin"
	v0 "github.com/synnaxlabs/synnax/pkg/service/access/rbac/migrations/v0"
	"github.com/synnaxlabs/synnax/pkg/service/user"
)

// Config is the configuration for New.
type Config struct {
	// User holds the roles of each user.
	User *user.Service
	// Roles is the set of provisioned built-in roles.
	Roles builtin.ProvisionResult
}

// New returns the ordered migration chain for RBAC.
func New(cfg Config) []gorp.Migration {
	return []gorp.Migration{
		v0.NewMigration(v0.MigrationConfig{
			User:  cfg.User,
			Roles: cfg.Roles,
		}),
	}
}
