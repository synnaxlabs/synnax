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
	groupv1 "github.com/synnaxlabs/synnax/pkg/service/group/versions/v1"
	ontology "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v0"
	rack "github.com/synnaxlabs/synnax/pkg/service/rack/versions/v1"
	rackv3 "github.com/synnaxlabs/synnax/pkg/service/rack/versions/v3"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/set"
	"go.uber.org/zap"
)

// SplitGroupsMigration moves each device in a group whose chain does not top out at the
// device's rack into a copy of that chain under the rack. It keeps only the first group
// of a device and drops the device from its groups when its rack does not exist. It
// then deletes each original group left with no devices, racks, or child groups. It
// must run before MembersMigration.
var SplitGroupsMigration = migrate.New("v61_device_group_split", splitGroups)

type rackCopy struct {
	group groupv1.Key
	rack  rack.Key
}

func splitGroups(ctx context.Context, tx gorp.Tx, ins alamos.Instrumentation) error {
	edges, err := ontology.RetrieveRelationships(
		ctx,
		tx,
		func(r ontology.Relationship) bool {
			return r.Type == ontology.RelationshipTypeParentOf &&
				r.From.Type == resource.TypeGroup &&
				r.To.Type == resource.TypeDevice
		},
	)
	if err != nil || len(edges) == 0 {
		return err
	}
	var groups []groupv1.Group
	if err = gorp.NewRetrieve[groupv1.Key, groupv1.Group]().
		Entries(&groups).
		Exec(ctx, tx); err != nil {
		return err
	}
	byKey := make(map[string]groupv1.Group, len(groups))
	for _, g := range groups {
		byKey[g.Key.String()] = g
	}
	var devices []Device
	if err = gorp.NewRetrieve[Key, Device]().Entries(&devices).
		Exec(ctx, tx); err != nil {
		return err
	}
	racks := make(map[string]rack.Key, len(devices))
	for _, d := range devices {
		racks[d.Key] = d.Rack
	}
	var rackRows []rackv3.Rack
	if err = gorp.NewRetrieve[rackv3.Key, rackv3.Rack]().
		Entries(&rackRows).
		Exec(ctx, tx); err != nil {
		return err
	}
	var (
		existing = set.New[rack.Key]()
		held     = set.New[string]()
	)
	for _, r := range rackRows {
		existing.Add(r.Key)
		if r.Group != nil {
			held.Add(r.Group.String())
		}
	}
	var (
		copies   = make(map[rackCopy]groupv1.Key)
		created  []groupv1.Group
		split    = set.New[string]()
		grouped  = set.New[string]()
		removed  []ontology.Relationship
		replaced []ontology.Relationship
	)
	var copyChain func(g groupv1.Group, r rack.Key, visited set.Set[string]) groupv1.Key
	copyChain = func(g groupv1.Group, r rack.Key, visited set.Set[string]) groupv1.Key {
		if k, ok := copies[rackCopy{group: g.Key, rack: r}]; ok {
			return k
		}
		visited.Add(g.Key.String())
		split.Add(g.Key.String())
		parent := r.ResourceID()
		if g.Parent != nil && g.Parent.Type == resource.TypeGroup {
			if p, ok := byKey[g.Parent.Key]; ok && !visited.Contains(g.Parent.Key) {
				parent = groupID(copyChain(p, r, visited))
			}
		}
		c := groupv1.Group{
			Key:    uuid.New(),
			Name:   g.Name,
			Scope:  g.Scope,
			Parent: &parent,
		}
		copies[rackCopy{group: g.Key, rack: r}] = c.Key
		created = append(created, c)
		return c.Key
	}
	remaining := make(map[string]int, len(edges))
	remove := func(e ontology.Relationship) {
		remaining[e.From.Key]--
		removed = append(removed, e)
	}
	for _, e := range edges {
		remaining[e.From.Key]++
		g, ok := byKey[e.From.Key]
		r, found := racks[e.To.Key]
		if !ok || !found || g.Scope != resource.TypeDevice {
			continue
		}
		if grouped.Contains(e.To.Key) {
			remove(e)
			continue
		}
		grouped.Add(e.To.Key)
		if !existing.Contains(r) {
			ins.L.Warn(
				"dropped a device from its group because its rack does not exist",
				zap.Stringer("group", g.Key),
				zap.String("group_name", g.Name),
				zap.Stringer("device", e.To),
			)
			remove(e)
			continue
		}
		if top, ok := groupv1.TopParent(g, byKey); ok && top == r.ResourceID() {
			continue
		}
		c := copyChain(g, r, set.New[string]())
		remove(e)
		replaced = append(replaced, ontology.Relationship{
			From: groupID(c),
			Type: ontology.RelationshipTypeParentOf,
			To:   e.To,
		})
	}
	if len(removed) == 0 {
		return nil
	}
	children := make(map[string]int, len(groups))
	for _, g := range groups {
		if g.Parent != nil && g.Parent.Type == resource.TypeGroup {
			children[g.Parent.Key]++
		}
	}
	var deleted []groupv1.Key
	for emptied := true; emptied; {
		emptied = false
		for key := range split {
			if remaining[key] > 0 || children[key] > 0 || held.Contains(key) {
				continue
			}
			g := byKey[key]
			split.Remove(key)
			deleted = append(deleted, g.Key)
			if g.Parent != nil && g.Parent.Type == resource.TypeGroup {
				children[g.Parent.Key]--
			}
			emptied = true
		}
	}
	if err = gorp.WrapWriter[groupv1.Key, groupv1.Group](tx).
		Set(ctx, created...); err != nil {
		return err
	}
	if err = gorp.WrapWriter[groupv1.Key, groupv1.Group](tx).
		Delete(ctx, deleted...); err != nil {
		return err
	}
	if err = ontology.DeleteRelationships(ctx, tx, removed...); err != nil {
		return err
	}
	return gorp.WrapWriter[string, ontology.Relationship](tx).Set(ctx, replaced...)
}

func groupID(k groupv1.Key) resource.ID {
	return resource.ID{Type: resource.TypeGroup, Key: k.String()}
}
