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
// of a device. It drops the device from its groups when its rack does not exist, and
// stages a removal for each such group. It then deletes each original group left with
// no devices, racks, or child groups. It must run before MembersMigration.
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
	racks, err := deviceRacks(ctx, tx)
	if err != nil {
		return err
	}
	existing, held, err := retrieveRacks(ctx, tx)
	if err != nil {
		return err
	}
	s := splitter{
		byKey:     make(map[string]groupv1.Group, len(groups)),
		copies:    make(map[rackCopy]groupv1.Key),
		split:     set.New[string](),
		remaining: make(map[string]int, len(edges)),
	}
	for _, g := range groups {
		s.byKey[g.Key.String()] = g
	}
	s.regroup(edges, racks, existing, ins)
	if len(s.removed) == 0 {
		return nil
	}
	deleted := s.deleteEmptied(groups, held)
	if err = gorp.WrapWriter[groupv1.Key, groupv1.Group](tx).
		Set(ctx, s.created...); err != nil {
		return err
	}
	if err = gorp.WrapWriter[groupv1.Key, groupv1.Group](tx).
		Delete(ctx, deleted...); err != nil {
		return err
	}
	if err = ontology.DeleteRelationships(ctx, tx, s.removed...); err != nil {
		return err
	}
	if err = groupv1.StageRemovals(ctx, tx, s.rackless...); err != nil {
		return err
	}
	return gorp.WrapWriter[string, ontology.Relationship](tx).Set(ctx, s.replaced...)
}

// deviceRacks maps each stored device key to the key of its rack.
func deviceRacks(ctx context.Context, tx gorp.Tx) (map[string]rack.Key, error) {
	var devices []Device
	if err := gorp.NewRetrieve[Key, Device]().Entries(&devices).
		Exec(ctx, tx); err != nil {
		return nil, err
	}
	racks := make(map[string]rack.Key, len(devices))
	for _, d := range devices {
		racks[d.Key] = d.Rack
	}
	return racks, nil
}

// retrieveRacks returns the keys of the stored racks and the keys of the groups that
// they hold.
func retrieveRacks(
	ctx context.Context,
	tx gorp.Tx,
) (set.Set[rack.Key], set.Set[string], error) {
	var rackRows []rackv3.Rack
	if err := gorp.NewRetrieve[rackv3.Key, rackv3.Rack]().
		Entries(&rackRows).
		Exec(ctx, tx); err != nil {
		return nil, nil, err
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
	return existing, held, nil
}

// splitter holds the groups and edge changes of one splitGroups run.
type splitter struct {
	byKey   map[string]groupv1.Group
	copies  map[rackCopy]groupv1.Key
	created []groupv1.Group
	// split holds the keys of the groups that were copied under a rack.
	split set.Set[string]
	// remaining counts the device edges that each group keeps.
	remaining map[string]int
	removed   []ontology.Relationship
	replaced  []ontology.Relationship
	rackless  []groupv1.Removal
}

// regroup moves each device edge to its group's copy under the device's rack, or drops
// it when the device is already grouped or its rack does not exist.
func (s *splitter) regroup(
	edges []ontology.Relationship,
	racks map[string]rack.Key,
	existing set.Set[rack.Key],
	ins alamos.Instrumentation,
) {
	grouped := set.New[string]()
	for _, e := range edges {
		s.remaining[e.From.Key]++
		g, ok := s.byKey[e.From.Key]
		r, found := racks[e.To.Key]
		if !ok || !found || g.Scope != resource.TypeDevice {
			continue
		}
		if grouped.Contains(e.To.Key) {
			s.remove(e)
			continue
		}
		grouped.Add(e.To.Key)
		if !existing.Contains(r) {
			s.dropRackless(e, g, ins)
			continue
		}
		if top, ok := groupv1.TopParent(g, s.byKey); ok && top == r.ResourceID() {
			continue
		}
		c := s.copyChain(g, r, set.New[string]())
		s.remove(e)
		s.replaced = append(s.replaced, ontology.Relationship{
			From: groupID(c),
			Type: ontology.RelationshipTypeParentOf,
			To:   e.To,
		})
	}
}

func (s *splitter) remove(e ontology.Relationship) {
	s.remaining[e.From.Key]--
	s.removed = append(s.removed, e)
}

func (s *splitter) dropRackless(
	e ontology.Relationship,
	g groupv1.Group,
	ins alamos.Instrumentation,
) {
	ins.L.Warn(
		"dropped a device from its group because its rack does not exist",
		zap.Stringer("group", g.Key),
		zap.String("group_name", g.Name),
		zap.Stringer("device", e.To),
	)
	s.remove(e)
	s.rackless = append(s.rackless, groupv1.Removal{
		Group:     g.Key,
		GroupName: g.Name,
		Member:    e.To,
		Reason:    groupv1.RemovedRackless,
	})
}

// copyChain copies g and its parent groups under rack r, and returns the key of the
// copy of g. A group already copied under r is reused. visited holds the groups of the
// chain copied so far, so a parent cycle ends the chain.
func (s *splitter) copyChain(
	g groupv1.Group,
	r rack.Key,
	visited set.Set[string],
) groupv1.Key {
	if k, ok := s.copies[rackCopy{group: g.Key, rack: r}]; ok {
		return k
	}
	visited.Add(g.Key.String())
	s.split.Add(g.Key.String())
	parent := r.ResourceID()
	if g.Parent != nil && g.Parent.Type == resource.TypeGroup {
		if p, ok := s.byKey[g.Parent.Key]; ok && !visited.Contains(g.Parent.Key) {
			parent = groupID(s.copyChain(p, r, visited))
		}
	}
	c := groupv1.Group{
		Key:    uuid.New(),
		Name:   g.Name,
		Scope:  g.Scope,
		Parent: &parent,
	}
	s.copies[rackCopy{group: g.Key, rack: r}] = c.Key
	s.created = append(s.created, c)
	return c.Key
}

// deleteEmptied returns the keys of the split groups left with no devices, child
// groups, or rack. It repeats until a pass deletes nothing, since a deletion can empty
// the parent.
func (s *splitter) deleteEmptied(
	groups []groupv1.Group,
	held set.Set[string],
) []groupv1.Key {
	children := make(map[string]int, len(groups))
	for _, g := range groups {
		if g.Parent != nil && g.Parent.Type == resource.TypeGroup {
			children[g.Parent.Key]++
		}
	}
	var deleted []groupv1.Key
	for emptied := true; emptied; {
		emptied = false
		for key := range s.split {
			if s.remaining[key] > 0 || children[key] > 0 || held.Contains(key) {
				continue
			}
			g := s.byKey[key]
			s.split.Remove(key)
			deleted = append(deleted, g.Key)
			if g.Parent != nil && g.Parent.Type == resource.TypeGroup {
				children[g.Parent.Key]--
			}
			emptied = true
		}
	}
	return deleted
}

func groupID(k groupv1.Key) resource.ID {
	return resource.ID{Type: resource.TypeGroup, Key: k.String()}
}
