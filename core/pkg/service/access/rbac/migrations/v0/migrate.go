// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0

import (
	"context"
	"slices"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/builtin"
	policy "github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy/migrations/v0"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/migrate"
)

// MigrationConfig contains the dependencies needed by the Phase 2 migration.
type MigrationConfig struct {
	User     *user.Service
	Ontology *ontology.Ontology
	Role     *role.Service
	Roles    builtin.ProvisionResult
}

// Migration (Phase 2) reads the persisted user-to-policy mapping from KV
// (written by Phase 1 in the policy package), queries users for their RootUser
// flag, determines the appropriate role for each user, and creates the ontology
// relationships.
func Migration(cfg MigrationConfig) migrate.Migration {
	return gorp.NewMigration(
		"v0.permission_assignment",
		func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
			mappings, err := policy.ReadLegacyMappings(ctx, tx)
			if err != nil {
				return err
			}

			var users []user.User
			if err := cfg.User.NewRetrieve().Entries(&users).Exec(ctx, tx); err != nil {
				return err
			}
			if len(users) == 0 {
				return nil
			}

			policyByUser := make(map[string][]policy.Policy)
			for _, m := range mappings {
				policyByUser[m.UserOntologyID.String()] = m.Policies
			}

			roleWriter := cfg.Role.NewWriter(tx, true)
			otgWriter := cfg.Ontology.NewWriter(tx)
			for _, u := range users {
				userOntologyID := user.OntologyID(u.Key)
				policies := policyByUser[userOntologyID.String()]
				roleKey := determineRole(u, policies, cfg.Roles)
				if err = otgWriter.DeleteRelationship(
					ctx,
					cfg.Role.UsersGroup().OntologyID(),
					ontology.RelationshipTypeParentOf,
					userOntologyID,
				); err != nil {
					return err
				}
				if err = roleWriter.AssignRole(ctx, userOntologyID, roleKey); err != nil {
					return err
				}
			}

			if len(mappings) > 0 {
				return policy.DeleteLegacyMappings(ctx, tx)
			}
			return nil
		},
	)
}

// determineRole maps a legacy user to a built-in role. Operator is the intentional
// default because all pre-RBAC users had write access. Viewer is not used here
// since no legacy deployment had read-only users.
func determineRole(u user.User, policies []policy.Policy, roles builtin.ProvisionResult) uuid.UUID {
	if u.RootUser {
		return roles.OwnerKey
	}
	if slices.ContainsFunc(policies, isAdminPolicy) {
		return roles.OwnerKey
	}
	if slices.ContainsFunc(policies, isSchematicPolicy) {
		return roles.EngineerKey
	}
	return roles.OperatorKey
}

func isAdminPolicy(p policy.Policy) bool {
	hasUserType := false
	hasPolicyType := false
	for _, obj := range p.Objects {
		if obj.Type == ontology.ResourceTypeUser {
			hasUserType = true
		}
		if obj.Type == "policy" {
			hasPolicyType = true
		}
	}
	return hasUserType && hasPolicyType && lo.Contains(p.Actions, "all")
}

func isSchematicPolicy(p policy.Policy) bool {
	for _, obj := range p.Objects {
		if obj.Type == "schematic" {
			return lo.Contains(p.Actions, "all")
		}
	}
	return false
}
