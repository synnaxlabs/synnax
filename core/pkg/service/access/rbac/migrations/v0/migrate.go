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
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/builtin"
	policy "github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/user"
)

// MigrationConfig is the configuration for NewMigration.
type MigrationConfig struct {
	// User retrieves the users whose legacy permission flags are converted.
	User *user.Service
	// Ontology holds the legacy group relationships of users, which the migration
	// deletes.
	Ontology *ontology.Ontology
	// Roles is the set of provisioned built-in roles to assign.
	Roles builtin.ProvisionResult
}

// Migration (Phase 2) reads the persisted user-to-policy mapping from KV (written by
// Phase 1 in the policy package), queries users for their RootUser flag, determines the
// appropriate role for each user, and adds it to the user's roles. It does nothing for
// a store that a v0.53 Core already migrated.
func NewMigration(cfg MigrationConfig) gorp.Migration {
	return migrate.New(
		"v0.permission_assignment",
		func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
			migrated, err := policy.AlreadyMigrated(ctx, tx)
			if migrated || err != nil {
				return err
			}
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
				policyByUser[m.UserResourceID.String()] = m.Policies
			}

			userWriter := cfg.User.NewWriter(tx)
			otgWriter := cfg.Ontology.NewWriter(tx)
			for _, u := range users {
				userResourceID := u.ResourceID()
				policies := policyByUser[userResourceID.String()]
				roleKey := determineRole(u, policies, cfg.Roles)
				var groups []ontology.Resource
				if err = cfg.Ontology.NewRetrieve().
					WhereIDs(userResourceID).
					TraverseTo(ontology.ParentsTraverser).
					WhereTypes(resource.TypeGroup).
					ExcludeFieldData(true).
					Entries(&groups).
					Exec(ctx, tx); err != nil {
					return err
				}
				for _, g := range groups {
					if err = otgWriter.DeleteRelationships(ctx, ontology.Relationship{
						From: g.ID,
						Type: ontology.RelationshipTypeParentOf,
						To:   userResourceID,
					}); err != nil {
						return err
					}
				}
				if err = userWriter.AssignRole(ctx, roleKey, u.Key); err != nil {
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
// default because all pre-RBAC users had write access. Viewer is not used here since no
// legacy deployment had read-only users.
func determineRole(
	u user.User,
	policies []policy.Policy,
	roles builtin.ProvisionResult,
) uuid.UUID {
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
		if obj.Type == resource.TypeUser {
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
