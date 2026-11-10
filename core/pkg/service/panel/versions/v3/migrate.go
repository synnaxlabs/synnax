// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v3

import (
	"context"
	"uuid"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	groupv1 "github.com/synnaxlabs/synnax/pkg/service/group/versions/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/panel/versions/v2"
	projectv3 "github.com/synnaxlabs/synnax/pkg/service/project/versions/v3"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

// MigratePanel lifts a v2 panel into v3. The panel sits in no group.
func MigratePanel(ctx context.Context, old v2.Panel) (Panel, error) {
	return autoMigratePanel(ctx, old)
}

var Migration = migrate.NewEntry("v61_add_group", MigratePanel)

// MembersMigration sets the group of each stored panel from the ontology edge that
// placed it in a group.
var MembersMigration = groupv1.NewMembersMigration(groupv1.MembersConfig[Key, Panel]{
	Name:         "v61_panel_group_membership",
	Type:         resource.TypePanel,
	SetGroup:     func(e *Panel, g groupv1.Key) { e.Group = &g },
	ProjectOwned: true,
})

// OwnersMigration sets the project or range of each stored panel from the ontology edge
// that owned it, and deletes each panel with no owner.
var OwnersMigration = projectv3.NewOwnersMigration(
	projectv3.OwnersConfig[Key, Panel]{
		Name:     "v61_panel_owner",
		Type:     resource.TypePanel,
		Owned:    func(e Panel) bool { return e.Project != nil || e.Range != nil },
		NameOf:   func(e Panel) string { return e.Name },
		SetOwner: func(e *Panel, p, r *uuid.UUID) { e.Project, e.Range = p, r },
	},
)

// arcExplorerViewType is the view type of the Arc explorer, which the Console no longer
// renders.
const arcExplorerViewType = "arc_explorer"

// ArcExplorerTabsMigration removes each Arc explorer tab from the stored panels.
var ArcExplorerTabsMigration = migrate.New(
	"v61_strip_arc_explorer_tabs",
	func(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
		var panels []Panel
		if err := gorp.NewRetrieve[Key, Panel]().
			Entries(&panels).
			Exec(ctx, tx); err != nil {
			return err
		}
		w := gorp.WrapWriter[Key, Panel](tx)
		for _, p := range panels {
			if !StripArcExplorerTabs(&p.Root) {
				continue
			}
			if err := w.Set(ctx, p); err != nil {
				return err
			}
		}
		return nil
	},
)

// StripArcExplorerTabs removes the Arc explorer tabs under n, reporting whether
// anything changed. Emptied leaves stay in place: the schema allows them.
func StripArcExplorerTabs(n *Node) bool {
	switch v := n.Variant.(type) {
	case SplitNode:
		first := StripArcExplorerTabs(&v.First)
		last := StripArcExplorerTabs(&v.Last)
		if !first && !last {
			return false
		}
		n.Variant = v
		return true
	case LeafNode:
		tabs := make([]Tab, 0, len(v.Tabs))
		for _, t := range v.Tabs {
			if view, ok := t.Variant.(ViewTab); ok && view.Type == arcExplorerViewType {
				continue
			}
			tabs = append(tabs, t)
		}
		if len(tabs) == len(v.Tabs) {
			return false
		}
		v.Tabs = tabs
		n.Variant = v
		return true
	}
	return false
}
