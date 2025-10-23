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
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	policyv0 "github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy/v0"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/version"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv"
)

var (
	migrationKey   = []byte("sy_rbac_policy_migration_v0_to_v1")
	migrationValue = []byte{1}
)

type MigrationState struct {
	V0Policies  []policyv0.Policy
	V1Policies  []policy.Policy
	Roles       []role.Role
	Assignments map[ontology.ID][]uuid.UUID
}

func MigratePolicies(
	ctx context.Context,
	db *gorp.DB,
	otg *ontology.Ontology,
) error {
	return db.WithTx(ctx, func(tx gorp.Tx) error {
		performed, closer, err := tx.Get(ctx, migrationKey)
		if err != nil && !errors.Is(err, kv.NotFound) {
			return err
		}
		if err == nil {
			defer closer.Close()
			if string(performed) == string(migrationValue) {
				return nil
			}
		}
		v0Policies, err := readV0Policies(ctx, tx)
		if err != nil {
			return errors.Wrap(err, "failed to read V0 policies")
		}
		if len(v0Policies) == 0 {
			return tx.Set(ctx, migrationKey, migrationValue)
		}
		state, err := applyHybridMigration(ctx, v0Policies)
		if err != nil {
			return errors.Wrap(err, "failed to apply migration")
		}
		if err := writeMigratedData(ctx, tx, otg, state); err != nil {
			return errors.Wrap(err, "failed to write migrated data")
		}
		return tx.Set(ctx, migrationKey, migrationValue)
	})
}

func readV0Policies(
	ctx context.Context,
	tx gorp.Tx,
) ([]policyv0.Policy, error) {
	var v0Policies []policyv0.Policy
	err := gorp.NewRetrieve[uuid.UUID, policyv0.Policy]().
		Entries(&v0Policies).
		Exec(ctx, tx)
	if errors.Is(err, kv.NotFound) {
		return []policyv0.Policy{}, nil
	}
	if err != nil {
		return nil, err
	}
	filteredV0 := make([]policyv0.Policy, 0, len(v0Policies))
	for _, pol := range v0Policies {
		if len(pol.Subjects) > 0 {
			filteredV0 = append(filteredV0, pol)
		}
	}
	return filteredV0, nil
}

func applyHybridMigration(
	ctx context.Context,
	v0Policies []policyv0.Policy,
) (*MigrationState, error) {
	state := &MigrationState{
		V0Policies:  v0Policies,
		V1Policies:  make([]policy.Policy, 0, len(v0Policies)),
		Roles:       make([]role.Role, 0),
		Assignments: make(map[ontology.ID][]uuid.UUID),
	}
	subjectPolicies := make(map[string][]uuid.UUID)
	for _, pol := range v0Policies {
		for _, subject := range pol.Subjects {
			if subject.Type == "user" {
				subjectPolicies[subject.Key] = append(
					subjectPolicies[subject.Key],
					pol.Key,
				)
			}
		}
		v1Policy := pol.ToV1()
		v1Policy.Version = version.V1
		state.V1Policies = append(state.V1Policies, v1Policy)
	}
	policySetRoles := make(map[string]*role.Role)
	for subjectKey, policyKeys := range subjectPolicies {
		sort.Slice(policyKeys, func(i, j int) bool {
			return policyKeys[i].String() < policyKeys[j].String()
		})
		policyHash := hashPolicySet(policyKeys)
		var roleKey uuid.UUID
		if existingRole, exists := policySetRoles[policyHash]; exists {
			roleKey = existingRole.Key
		} else {
			newRole := role.Role{
				Key:         uuid.New(),
				Name:        generateRoleName(policyKeys, len(policySetRoles)),
				Description: fmt.Sprintf("Auto-migrated from %d V0 policies", len(policyKeys)),
				Policies:    policyKeys,
				Internal:    false,
			}
			state.Roles = append(state.Roles, newRole)
			policySetRoles[policyHash] = &newRole
			roleKey = newRole.Key
		}
		subjectOntologyID := ontology.ID{Type: "user", Key: subjectKey}
		state.Assignments[subjectOntologyID] = append(
			state.Assignments[subjectOntologyID],
			roleKey,
		)
	}
	return state, nil
}

func hashPolicySet(policyKeys []uuid.UUID) string {
	hasher := sha256.New()
	for _, key := range policyKeys {
		hasher.Write([]byte(key.String()))
	}
	return hex.EncodeToString(hasher.Sum(nil))[:16]
}

func generateRoleName(policyKeys []uuid.UUID, index int) string {
	if len(policyKeys) == 1 {
		return fmt.Sprintf("Migrated-Policy-%s", policyKeys[0].String()[:8])
	}
	return fmt.Sprintf("Migrated-Role-%d", index+1)
}

func writeMigratedData(
	ctx context.Context,
	tx gorp.Tx,
	otg *ontology.Ontology,
	state *MigrationState,
) error {
	otgWriter := otg.NewWriter(tx)
	for i, v0Policy := range state.V0Policies {
		if err := gorp.NewDelete[uuid.UUID, policyv0.Policy]().
			WhereKeys(v0Policy.Key).
			Exec(ctx, tx); err != nil {
			return errors.Wrapf(err, "failed to delete V0 policy %s", v0Policy.Key)
		}
		if err := gorp.NewCreate[uuid.UUID, policy.Policy]().
			Entry(&state.V1Policies[i]).
			Exec(ctx, tx); err != nil {
			return errors.Wrapf(err, "failed to create V1 policy %s", state.V1Policies[i].Key)
		}
	}
	for _, r := range state.Roles {
		if err := gorp.NewCreate[uuid.UUID, role.Role]().
			Entry(&r).
			Exec(ctx, tx); err != nil {
			return errors.Wrapf(err, "failed to create role %s", r.Key)
		}
		if err := otgWriter.DefineResource(ctx, role.OntologyID(r.Key)); err != nil {
			return errors.Wrapf(err, "failed to define role resource %s", r.Key)
		}
	}
	for subjectID, roleKeys := range state.Assignments {
		for _, roleKey := range roleKeys {
			if err := otgWriter.DefineRelationship(
				ctx,
				subjectID,
				role.HasRole,
				role.OntologyID(roleKey),
			); err != nil {
				return errors.Wrapf(err, "failed to assign role %s to subject %s", roleKey, subjectID)
			}
		}
	}
	return nil
}
