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

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	ontology "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
)

// ParentMigration sets the parent of each stored device from the parent edge that
// points at it from another stored device. It then deletes every parent edge into a
// device from a rack or a device.
var ParentMigration = migrate.New("v61_device_parent", migrateParents)

func migrateParents(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
	edges, err := collect(ctx, tx, func(r ontology.Relationship) bool {
		return r.Type == ontology.RelationshipTypeParentOf &&
			r.To.Type == resource.TypeDevice &&
			(r.From.Type == resource.TypeRack || r.From.Type == resource.TypeDevice)
	})
	if err != nil || len(edges) == 0 {
		return err
	}
	devices, err := collect(ctx, tx, func(Device) bool { return true })
	if err != nil {
		return err
	}
	byKey := make(map[Key]int, len(devices))
	for i, d := range devices {
		byKey[d.Key] = i
	}
	var parented []Device
	for _, e := range edges {
		if e.From.Type != resource.TypeDevice || e.From.Key == e.To.Key {
			continue
		}
		i, ok := byKey[Key(e.To.Key)]
		if !ok || devices[i].Parent != nil {
			continue
		}
		if _, ok = byKey[Key(e.From.Key)]; !ok {
			continue
		}
		parent := Key(e.From.Key)
		devices[i].Parent = &parent
		parented = append(parented, devices[i])
	}
	if err = gorp.WrapWriter[Key, Device](tx).Set(ctx, parented...); err != nil {
		return err
	}
	keys := make([]string, len(edges))
	for i, e := range edges {
		keys[i] = e.GorpKey()
	}
	return gorp.WrapWriter[string, ontology.Relationship](tx).Delete(ctx, keys...)
}

func collect[K gorp.Key, E gorp.Entry[K]](
	ctx context.Context,
	tx gorp.Tx,
	keep func(E) bool,
) (out []E, err error) {
	iter, err := gorp.WrapReader[K, E](tx).OpenIterator(gorp.IterOptions{})
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Combine(err, iter.Close()) }()
	for iter.First(); iter.Valid(); iter.Next() {
		if e := iter.Value(ctx); e != nil && keep(*e) {
			out = append(out, *e)
		}
	}
	return out, iter.Error()
}
