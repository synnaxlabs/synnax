// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ranger

import (
	"context"
	"unsafe"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/encoding"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

type rangeGroupsMigration struct {
	otg   *ontology.Ontology
	group *group.Service
	codec encoding.Codec
}

func (m *rangeGroupsMigration) Name() string { return "range_groups" }

func (m *rangeGroupsMigration) Run(
	ctx context.Context,
	kvTx kv.Tx,
	migCfg gorp.MigrationConfig,
) error {
	gorpTx := gorp.WrapTx(kvTx, m.codec)

	migCfg.L.Debug("swapping invalid time ranges")
	if err := m.swapRanges(ctx, kvTx, migCfg.Prefix); err != nil {
		return err
	}

	var topLevelGroup group.Group
	if err := m.group.
		NewRetrieve().
		Entry(&topLevelGroup).
		WhereNames("Ranges").
		Exec(ctx, gorpTx); err != nil {
		if errors.Is(err, query.ErrNotFound) {
			return nil
		}
		return err
	}

	var groups []ontology.Resource
	if err := m.otg.
		NewRetrieve().
		WhereIDs(topLevelGroup.OntologyID()).
		TraverseTo(ontology.ChildrenTraverser).
		WhereTypes(ontology.ResourceTypeGroup).
		Entries(&groups).
		Exec(ctx, gorpTx); err != nil {
		return err
	}
	migCfg.L.Debug("converting groups to parent ranges", zap.Int("groups", len(groups)))

	otgWriter := m.otg.NewWriter(gorpTx)
	if err := otgWriter.DeleteOutgoingRelationshipsOfType(
		ctx,
		topLevelGroup.OntologyID(),
		ontology.RelationshipTypeParentOf,
	); err != nil {
		return err
	}
	if err := m.group.NewWriter(gorpTx).Delete(ctx, topLevelGroup.Key); err != nil {
		return err
	}

	rangeMap, err := m.loadAllRanges(ctx, kvTx, migCfg.Prefix)
	if err != nil {
		return err
	}

	for _, g := range groups {
		var childRanges []ontology.Resource
		// ExcludeFieldData is required because this migration runs during
		// OpenService before the range ontology service is registered. Without
		// it, the ontology would panic trying to retrieve range resources.
		if err = m.otg.
			NewRetrieve().
			WhereIDs(g.ID).
			TraverseTo(ontology.ChildrenTraverser).
			WhereTypes(ontology.ResourceTypeRange).
			ExcludeFieldData(true).
			Entries(&childRanges).
			Exec(ctx, gorpTx); err != nil {
			return err
		}
		migCfg.L.Debug(
			"migrating range group",
			zap.String("group", g.Name),
			zap.Int("children", len(childRanges)),
		)
		gKey, err := uuid.Parse(g.ID.Key)
		if err != nil {
			return err
		}
		if err = otgWriter.DeleteOutgoingRelationshipsOfType(
			ctx,
			g.ID,
			ontology.RelationshipTypeParentOf,
		); err != nil {
			return err
		}
		if err = m.group.NewWriter(gorpTx).Delete(ctx, gKey); err != nil {
			return err
		}
		if len(childRanges) == 0 {
			continue
		}
		tr := telem.TimeRange{Start: telem.TimeStampMax, End: telem.TimeStampMin}
		for _, cr := range childRanges {
			rKey, err := uuid.Parse(cr.ID.Key)
			if err != nil {
				return err
			}
			r, ok := rangeMap[rKey]
			if !ok {
				continue
			}
			if r.TimeRange.Start.Before(tr.Start) {
				tr.Start = r.TimeRange.Start
			}
			if r.TimeRange.End.After(tr.End) {
				tr.End = r.TimeRange.End
			}
		}
		newParentRange := Range{
			Key:       uuid.New(),
			Name:      g.Name,
			TimeRange: tr.MakeValid(),
		}
		if err = m.writeRange(ctx, kvTx, migCfg.Prefix, newParentRange); err != nil {
			return err
		}
		if err = otgWriter.DefineResource(ctx, OntologyID(newParentRange.Key)); err != nil {
			return err
		}
		if err = otgWriter.DefineFromOneToManyRelationships(
			ctx,
			newParentRange.OntologyID(),
			ontology.RelationshipTypeParentOf,
			lo.Map(childRanges, func(r ontology.Resource, _ int) ontology.ID {
				return r.ID
			}),
		); err != nil {
			return err
		}
	}

	return nil
}

func (m *rangeGroupsMigration) swapRanges(
	ctx context.Context,
	kvTx kv.Tx,
	prefix []byte,
) (err error) {
	iter, err := kvTx.OpenIterator(kv.IterPrefix(prefix))
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Combine(err, iter.Close())
	}()
	for iter.First(); iter.Valid(); iter.Next() {
		var r Range
		if err = m.codec.Decode(ctx, iter.Value(), &r); err != nil {
			return err
		}
		r.TimeRange = r.TimeRange.MakeValid()
		data, encErr := m.codec.Encode(ctx, r)
		if encErr != nil {
			return encErr
		}
		if err = kvTx.Set(ctx, iter.Key(), data); err != nil {
			return err
		}
	}
	return err
}

func (m *rangeGroupsMigration) loadAllRanges(
	ctx context.Context,
	kvTx kv.Tx,
	prefix []byte,
) (result map[uuid.UUID]Range, err error) {
	result = make(map[uuid.UUID]Range)
	iter, err := kvTx.OpenIterator(kv.IterPrefix(prefix))
	if err != nil {
		return nil, err
	}
	defer func() {
		err = errors.Combine(err, iter.Close())
	}()
	for iter.First(); iter.Valid(); iter.Next() {
		var rng Range
		if decErr := m.codec.Decode(ctx, iter.Value(), &rng); decErr != nil {
			return nil, decErr
		}
		result[rng.Key] = rng
	}
	return result, err
}

func (m *rangeGroupsMigration) writeRange(
	ctx context.Context,
	kvTx kv.Tx,
	prefix []byte,
	r Range,
) error {
	data, err := m.codec.Encode(ctx, r)
	if err != nil {
		return err
	}
	return kvTx.Set(ctx, encodeUUIDKey(prefix, r.Key), data)
}

func encodeUUIDKey(prefix []byte, id uuid.UUID) []byte {
	const uuidSize = int(unsafe.Sizeof(uuid.UUID{}))
	key := make([]byte, len(prefix)+uuidSize)
	copy(key, prefix)
	for i := range uuidSize {
		key[len(prefix)+i] = id[uuidSize-1-i]
	}
	return key
}

// newRangeGroupsMigration constructs the migration. It uses gorp.Codec and
// binary.Codec to remain independent of the DB's default codec. This is
// necessary because the migration runs AFTER NewCodecTransition and the
// entries are already in ORC binary format.
func newRangeGroupsMigration(cfg ServiceConfig) gorp.Migration {
	return gorp.WithDependencies(&rangeGroupsMigration{
		otg:   cfg.Ontology,
		group: cfg.Group,
		codec: RangeCodec,
	}, "msgpack_to_binary")
}

// Ensure rangeGroupsMigration implements the Migration interface at compile time.
var _ gorp.Migration = (*rangeGroupsMigration)(nil)
