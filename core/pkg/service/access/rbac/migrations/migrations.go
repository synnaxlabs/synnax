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
	v1 "github.com/synnaxlabs/synnax/pkg/service/access/rbac/migrations/v1"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/user"
)

// Config is the configuration for New.
type Config struct {
	// User holds the role of each user.
	User *user.Service
	// Role holds the roles.
	Role *role.Service
	// Policy holds the policies of each role.
	Policy *policy.Service
	// Builtin is the set of provisioned built-in roles.
	Builtin builtin.ProvisionResult
}

// New returns the ordered migration chain for RBAC.
func New(cfg Config) []gorp.Migration {
	return []gorp.Migration{
		v0.NewMigration(v0.MigrationConfig{
			User:  cfg.User,
			Roles: cfg.Builtin,
		}),
		v1.NewMigration(v1.MigrationConfig{
			User:   cfg.User,
			Role:   cfg.Role,
			Policy: cfg.Policy,
			Owner:  cfg.Builtin.OwnerKey,
		}),
	}
}
