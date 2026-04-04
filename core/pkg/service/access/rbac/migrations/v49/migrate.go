// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v49

import (
	"context"
	"slices"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	policyv49 "github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy/migrations/v49"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/migrate"
)

const MigrationKey = "legacy_permission_assignment"

// ProvisionResult contains the keys of all provisioned built-in roles.
type ProvisionResult struct {
	OwnerKey    uuid.UUID
	EngineerKey uuid.UUID
	OperatorKey uuid.UUID
	ViewerKey   uuid.UUID
}

// MigrationConfig contains the dependencies needed by the Phase 2 migration.
type MigrationConfig struct {
	User     *user.Service
	Ontology *ontology.Ontology
	Role     *role.Service
	Roles    ProvisionResult
}

// Migration (Phase 2) reads the persisted user-to-policy mapping from KV
// (written by Phase 1 in the policy package), queries users for their RootUser
// flag, determines the appropriate role for each user, and creates the ontology
// relationships.
func Migration(cfg MigrationConfig) migrate.Migration {
	return gorp.NewMigration(
		MigrationKey,
		func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
			mappings, err := policyv49.ReadLegacyMappings(ctx, tx)
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

			policyByUser := make(map[string][]policyv49.Policy)
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
				return policyv49.DeleteLegacyMappings(ctx, tx)
			}
			return nil
		},
	)
}

func determineRole(u user.User, policies []policyv49.Policy, roles ProvisionResult) uuid.UUID {
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

func isAdminPolicy(p policyv49.Policy) bool {
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
	return hasUserType && hasPolicyType && lo.Contains(p.Actions, access.Action("all"))
}

func isSchematicPolicy(p policyv49.Policy) bool {
	for _, obj := range p.Objects {
		if obj.Type == "schematic" {
			return lo.Contains(p.Actions, access.Action("all"))
		}
	}
	return false
}
