// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package doctor

import (
	"slices"

	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/device"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/panel"
	"github.com/synnaxlabs/synnax/pkg/service/ranger/alias"
	rangekv "github.com/synnaxlabs/synnax/pkg/service/ranger/kv"
	"github.com/synnaxlabs/synnax/pkg/service/task"
)

// checkChannel reports data channels whose index channel is gone.
func checkChannel(s *state, c channel.Channel) {
	if c.Virtual || c.IsIndex || c.LocalIndex == 0 {
		return
	}
	index := channel.NewKey(c.Leaseholder, c.LocalIndex)
	if _, found := s.channels[index]; !found {
		s.note(
			CheckChannelIndex,
			"channel references a deleted index channel",
			c.Key().String(),
		)
	}
}

// checkAlias reports range aliases whose range or channel is gone.
func checkAlias(s *state, a alias.Alias) {
	if !s.ranges.Contains(a.Range) {
		s.note(CheckAlias, "alias references a deleted range", a.GorpKey())
	}
	if _, found := s.channels[a.Channel]; !found {
		s.note(CheckAlias, "alias references a deleted channel", a.GorpKey())
	}
}

// checkRangeKV reports range key-value pairs whose range is gone.
func checkRangeKV(s *state, p rangekv.Pair) {
	if !s.ranges.Contains(p.Range) {
		s.note(CheckRangeKV, "key-value pair references a deleted range", p.GorpKey())
	}
}

// checkTask reports tasks with a deleted rack or no stored configuration.
func checkTask(s *state, t task.Task) {
	if !t.Rack.IsZero() && !s.racks.Contains(t.Rack) {
		s.note(CheckRack, "task references a deleted rack", t.Key.String())
	}
	if !s.configs.Contains(t.Key) {
		s.note(CheckTaskConfig, "task has no stored configuration", t.Key.String())
	}
}

// checkDevice reports devices whose rack is gone.
func checkDevice(s *state, d device.Device) {
	if !s.racks.Contains(d.Rack) {
		s.note(CheckRack, "device references a deleted rack", d.Key)
	}
}

// checkCredentials reports credentials whose user is gone.
func checkCredentials(s *state, c auth.SecureCredentials) {
	if !s.usernames.Contains(c.Username) {
		s.note(CheckCredentials, "credentials have no user", c.Username)
	}
}

// checkPolicy reports policy objects naming a deleted resource. Objects with an empty
// key name a whole resource type rather than one entity, and are always valid.
func checkPolicy(s *state, p policy.Policy) {
	for _, obj := range p.Objects {
		if obj.Key == "" || s.resources.Contains(obj.String()) {
			continue
		}
		s.note(
			CheckPolicyObject,
			"policy object references a deleted resource",
			p.Key.String()+" -> "+obj.String(),
		)
	}
}

// checkPanel reports panel tabs displaying a deleted resource.
func checkPanel(s *state, p panel.Panel) {
	for _, id := range tabResources(p.Root) {
		if s.resources.Contains(id.String()) {
			continue
		}
		s.note(
			CheckPanelTab,
			"panel tab references a deleted resource",
			p.Key.String()+" -> "+id.String(),
		)
	}
}

// tabResources collects the resource every tab in the node's subtree displays.
func tabResources(n panel.Node) []ontology.ID {
	switch v := n.Variant.(type) {
	case panel.LeafNode:
		ids := make([]ontology.ID, 0, len(v.Tabs))
		for _, t := range v.Tabs {
			if rt, isResource := t.Variant.(panel.ResourceTab); isResource {
				ids = append(ids, rt.Resource)
			}
		}
		return ids
	case panel.SplitNode:
		return slices.Concat(tabResources(v.First), tabResources(v.Last))
	}
	return nil
}

// checkResource reports resources of an unknown type and resources whose backing
// entity is gone.
func checkResource(s *state, r ontology.Resource) {
	if !r.ID.Type.IsValid() {
		s.note(CheckResourceType, "resource has an unknown type", r.ID.String())
		return
	}
	if slices.Contains(whitelist, r.ID.Type) {
		return
	}
	keys, backed := s.entities[r.ID.Type]
	if !backed {
		s.note(
			CheckResourceOrphan,
			"resource type is backed by no table",
			r.ID.String(),
		)
		return
	}
	if !keys.Contains(r.ID.Key) {
		s.note(CheckResourceOrphan, "resource has no backing entity", r.ID.String())
	}
}

// checkRelationshipKey reports stored relationship keys that cannot be parsed. An
// unparsable key is unreachable: every ontology traversal decodes it.
func checkRelationshipKey(s *state, key string) {
	if _, err := ontology.ParseRelationship(key); err != nil {
		s.note(CheckRelationshipKey, "relationship key cannot be parsed", key)
	}
}

// checkRelationship reports relationships pointing at a deleted resource.
func checkRelationship(s *state, r ontology.Relationship) {
	for _, id := range []ontology.ID{r.From, r.To} {
		if !s.resources.Contains(id.String()) {
			s.note(
				CheckRelationshipEndpoint,
				"relationship references a deleted resource",
				r.GorpKey(),
			)
		}
	}
}
