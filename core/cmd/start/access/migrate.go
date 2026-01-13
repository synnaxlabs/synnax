// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package access

import (
	"context"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv"
)

var (
	migrationKey   = []byte("sy_rbac_migration_performed")
	migrationValue = []byte{1}
)

// LegacyPolicy represents the old policy format with Subjects field.
// Used only for reading legacy data during migration.
// Implements CustomTypeName to read from the same table as policy.Policy.
type LegacyPolicy struct {
	Key      uuid.UUID       `json:"key" msgpack:"key"`
	Subjects []ontology.ID   `json:"subjects" msgpack:"subjects"`
	Objects  []ontology.ID   `json:"objects" msgpack:"objects"`
	Actions  []access.Action `json:"actions" msgpack:"actions"`
}

var _ gorp.Entry[uuid.UUID] = LegacyPolicy{}

func (p LegacyPolicy) GorpKey() uuid.UUID { return p.Key }
func (p LegacyPolicy) SetOptions() []any  { return nil }

// CustomTypeName makes LegacyPolicy read from the same gorp table as policy.Policy.
// This is necessary because the old policies were stored with type name "Policy".
func (p LegacyPolicy) CustomTypeName() string { return "Policy" }

// MigratePermissions migrates users from the legacy permission system to role-based
// access control. This migration:
//   - Removes old UsersGroup -> ParentOf -> User relationships
//   - Assigns Owner role to users with RootUser=true or admin-like policies
//   - Assigns Engineer role to users with schematic policies
//   - Assigns Operator role to all other users
//   - Deletes legacy policies (those with Subjects field)
//
// The migration is idempotent and only runs once, tracked via a KV flag.
func MigratePermissions(
	ctx context.Context,
	tx gorp.Tx,
	dist *distribution.Layer,
	svc *service.Layer,
	roles ProvisionResult,
) error {
	// Check if migration already performed
	performed, closer, err := tx.Get(ctx, migrationKey)
	if err != nil && !errors.Is(err, kv.NotFound) {
		return err
	} else if err == nil {
		if err := closer.Close(); err != nil {
			return err
		}
	}
	if string(performed) == string(migrationValue) {
		return nil
	}

	// Query all users
	var users []user.User
	if err := svc.User.NewRetrieve().Entries(&users).Exec(ctx, tx); err != nil {
		return err
	}

	// Query all legacy policies
	var legacyPolicies []LegacyPolicy
	if err := gorp.NewRetrieve[uuid.UUID, LegacyPolicy]().
		Entries(&legacyPolicies).
		Exec(ctx, tx); err != nil {
		return err
	}

	// Filter to only legacy policies (those with non-empty Subjects)
	legacyPolicies = lo.Filter(legacyPolicies, func(p LegacyPolicy, _ int) bool {
		return len(p.Subjects) > 0
	})

	// Build a map of user ontology ID -> legacy policies for that user
	userPolicies := make(map[string][]LegacyPolicy)
	for _, p := range legacyPolicies {
		for _, subject := range p.Subjects {
			key := subject.String()
			userPolicies[key] = append(userPolicies[key], p)
		}
	}

	roleWriter := svc.RBAC.Role.NewWriter(tx, true)
	otgWriter := dist.Ontology.NewWriter(tx)

	// Migrate each user
	for _, u := range users {
		userOntologyID := user.OntologyID(u.Key)
		policies := userPolicies[userOntologyID.String()]

		// Determine the appropriate role
		roleKey := determineRole(u, policies, roles)

		// Remove the old UsersGroup -> ParentOf -> User relationship.
		// This cleans up the legacy ontology structure where users were
		// direct children of the Users group.
		if err = otgWriter.DeleteRelationship(
			ctx,
			svc.RBAC.Role.UsersGroup().OntologyID(),
			ontology.RelationshipTypeParentOf,
			userOntologyID,
		); err != nil {
			return err
		}

		// Create the new Role -> ParentOf -> User relationship
		if err = roleWriter.AssignRole(ctx, userOntologyID, roleKey); err != nil {
			return err
		}
	}

	// Delete legacy policies
	if len(legacyPolicies) > 0 {
		legacyKeys := lo.Map(legacyPolicies, func(p LegacyPolicy, _ int) uuid.UUID {
			return p.Key
		})
		if err := gorp.NewDelete[uuid.UUID, LegacyPolicy]().
			WhereKeys(legacyKeys...).
			Exec(ctx, tx); err != nil {
			return err
		}
	}

	// Mark migration complete
	return tx.Set(ctx, migrationKey, migrationValue)
}

// determineRole determines which role to assign to a user based on their legacy
// permissions. Priority order:
//  1. RootUser=true -> Owner
//  2. Admin policy (objects contain user and policy types) -> Owner
//  3. Schematic policy (objects contain schematic type) -> Engineer
//  4. Default -> Operator
func determineRole(u user.User, policies []LegacyPolicy, roles ProvisionResult) uuid.UUID {
	// Check RootUser flag
	if u.RootUser {
		return roles.OwnerKey
	}

	// Check for admin-like policy
	for _, p := range policies {
		if isAdminPolicy(p) {
			return roles.OwnerKey
		}
	}

	// Check for schematic policy
	for _, p := range policies {
		if isSchematicPolicy(p) {
			return roles.EngineerKey
		}
	}

	// Default to Operator
	return roles.OperatorKey
}

// isAdminPolicy checks if a legacy policy grants admin-level access.
// Admin policies have objects containing both "user" and "policy" types with "all" action.
func isAdminPolicy(p LegacyPolicy) bool {
	hasUserType := false
	hasPolicyType := false
	for _, obj := range p.Objects {
		if obj.Type == user.OntologyType {
			hasUserType = true
		}
		if obj.Type == "policy" {
			hasPolicyType = true
		}
	}
	return hasUserType && hasPolicyType && containsLegacyAllAction(p.Actions)
}

// isSchematicPolicy checks if a legacy policy grants schematic access.
// Schematic policies have objects containing the "schematic" type with "all" action.
func isSchematicPolicy(p LegacyPolicy) bool {
	for _, obj := range p.Objects {
		if obj.Type == "schematic" {
			return containsLegacyAllAction(p.Actions)
		}
	}
	return false
}

// containsLegacyAllAction checks if the actions slice contains the "all" action.
func containsLegacyAllAction(actions []access.Action) bool {
	return lo.Contains(actions, "all")
}
