// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2

import (
	"context"
	"uuid"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	v1 "github.com/synnaxlabs/synnax/pkg/service/ranger/versions/v1"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
)

const labeledBy ontology.RelationshipType = "labeled_by"

// Migration moves the parent of each range from a parent ontology relationship between
// two ranges into the parent field, and its labels from labeled_by relationships into
// the labels field. It deletes each relationship it moves, and drops labels with no
// ontology resource.
var Migration = migrate.New("range_parent_labels", migrateRanges)

func migrateRanges(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
	old, err := collect(ctx, gorp.WrapReader[Key, v1.Range](tx))
	if err != nil {
		return err
	}
	rels, err := collect(ctx, gorp.WrapReader[string, ontology.Relationship](tx))
	if err != nil {
		return err
	}
	ranges := make(map[Key]Range, len(old))
	for _, o := range old {
		if ranges[o.Key], err = autoMigrateRange(ctx, o); err != nil {
			return err
		}
	}
	var (
		resources = gorp.WrapReader[string, ontology.Resource](tx)
		relWriter = gorp.WrapWriter[string, ontology.Relationship](tx)
	)
	for _, rel := range rels {
		if rel.From.Type != resource.TypeRange {
			continue
		}
		isParent := rel.Type == ontology.RelationshipTypeParentOf &&
			rel.To.Type == resource.TypeRange
		if !isParent && rel.Type != labeledBy {
			continue
		}
		if err = relWriter.Delete(ctx, rel.GorpKey()); err != nil {
			return err
		}
		from, err := uuid.Parse(rel.From.Key)
		if err != nil {
			return errors.Wrapf(err, "range %s", rel.From.Key)
		}
		to, err := uuid.Parse(rel.To.Key)
		if err != nil {
			return errors.Wrapf(err, "%s of range %s", rel.To.Type, rel.From.Key)
		}
		if isParent {
			child, ok := ranges[to]
			if _, parentFound := ranges[from]; ok && parentFound &&
				child.Parent == nil {
				child.Parent = &from
				ranges[to] = child
			}
			continue
		}
		rng, ok := ranges[from]
		if !ok {
			continue
		}
		_, err = resources.Get(ctx, rel.To.String())
		if errors.Is(err, query.ErrNotFound) {
			continue
		}
		if err != nil {
			return err
		}
		rng.Labels = append(rng.Labels, to)
		ranges[from] = rng
	}
	writer := gorp.WrapWriter[Key, Range](tx)
	for _, rng := range ranges {
		if err = writer.Set(ctx, rng); err != nil {
			return err
		}
	}
	return nil
}

func collect[K gorp.Key, E gorp.Entry[K]](
	ctx context.Context,
	r gorp.Reader[K, E],
) (out []E, err error) {
	iter, err := r.OpenIterator(gorp.IterOptions{})
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Combine(err, iter.Close()) }()
	for iter.First(); iter.Valid(); iter.Next() {
		if e := iter.Value(ctx); e != nil {
			out = append(out, *e)
		}
	}
	return out, iter.Error()
}
