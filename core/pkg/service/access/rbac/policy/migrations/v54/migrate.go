// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v54

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/migrate"
	"github.com/synnaxlabs/x/query"
)

const (
	migrationKey                = "v49_permission_extraction"
	legacyMigrationPerformedKey = "sy_rbac_migration_performed"
	LegacyMappingKVKey          = "sy_rbac_legacy_permission_mapping"
)

// LegacyUserMapping stores the legacy permission data for a single user,
// persisted to KV between Phase 1 (extraction) and Phase 2 (role assignment).
type LegacyUserMapping struct {
	UserOntologyID ontology.ID `json:"user_ontology_id"`
	Policies       []Policy    `json:"policies"`
}

// Migration (Phase 1) reads legacy policies with Subjects from
// the Policy table, persists the user-to-policy mapping in KV, and deletes the
// legacy entries. This runs before any oracle schema migrations that could
// re-encode entries and lose the Subjects field.
func Migration() migrate.Migration {
	return gorp.NewMigration(
		migrationKey,
		func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
			if alreadyMigrated(ctx, tx) {
				return nil
			}
			var legacyPolicies []Policy
			reader := gorp.WrapReader[uuid.UUID, Policy](tx)
			iter, err := reader.OpenIterator(gorp.IterOptions{})
			if err != nil {
				return err
			}
			defer func() { err = errors.Combine(err, iter.Close()) }()
			for iter.First(); iter.Valid(); iter.Next() {
				v := iter.Value(ctx)
				if v == nil {
					if err = iter.Error(); err != nil {
						return err
					}
					continue
				}
				if len(v.Subjects) > 0 {
					legacyPolicies = append(legacyPolicies, *v)
				}
			}
			if len(legacyPolicies) == 0 {
				return nil
			}

			userMappings := buildUserMappings(legacyPolicies)
			mappingBytes, err := json.Marshal(userMappings)
			if err != nil {
				return errors.Wrap(err, "failed to marshal legacy permission mapping")
			}
			if err := tx.Set(ctx, []byte(LegacyMappingKVKey), mappingBytes); err != nil {
				return err
			}

			legacyKeys := lo.Map(legacyPolicies, func(p Policy, _ int) uuid.UUID {
				return p.Key
			})
			writer := gorp.WrapWriter[uuid.UUID, Policy](tx)
			return writer.Delete(ctx, legacyKeys...)
		},
	)
}

func alreadyMigrated(ctx context.Context, tx gorp.Tx) bool {
	performed, closer, err := tx.Get(ctx, []byte(legacyMigrationPerformedKey))
	if err != nil {
		return false
	}
	_ = closer.Close()
	return string(performed) == string([]byte{1})
}

func buildUserMappings(legacyPolicies []Policy) []LegacyUserMapping {
	byUser := make(map[string][]Policy)
	for _, p := range legacyPolicies {
		for _, subject := range p.Subjects {
			key := subject.String()
			byUser[key] = append(byUser[key], p)
		}
	}
	mappings := make([]LegacyUserMapping, 0, len(byUser))
	for key, policies := range byUser {
		id, err := ontology.ParseID(key)
		if err != nil {
			continue
		}
		mappings = append(mappings, LegacyUserMapping{
			UserOntologyID: id,
			Policies:       policies,
		})
	}
	return mappings
}

// ReadLegacyMappings reads the persisted legacy permission mapping from KV.
// Returns nil if no mapping exists.
func ReadLegacyMappings(ctx context.Context, tx gorp.Tx) ([]LegacyUserMapping, error) {
	mappingBytes, closer, err := tx.Get(ctx, []byte(LegacyMappingKVKey))
	if err != nil {
		if errors.Is(err, query.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	if err := closer.Close(); err != nil {
		return nil, err
	}
	if len(mappingBytes) == 0 {
		return nil, nil
	}
	var mappings []LegacyUserMapping
	if err := json.Unmarshal(mappingBytes, &mappings); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal legacy permission mapping")
	}
	return mappings, nil
}

// DeleteLegacyMappings removes the persisted legacy permission mapping from KV.
func DeleteLegacyMappings(ctx context.Context, tx gorp.Tx) error {
	return tx.Delete(ctx, []byte(LegacyMappingKVKey))
}
