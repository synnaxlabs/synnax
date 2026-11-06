// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package panel

import (
	"slices"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	"github.com/synnaxlabs/synnax/pkg/service/panel/versions"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

func tabTarget(id resource.ID) gorp.Target {
	return gorp.Target{Table: string(id.Type), Key: id.Key}
}

// tabsReference returns the reference from a panel to the resources its tabs display.
// Deleting a resource removes its tab and collapses the leaves the removal empties.
func tabsReference() *gorp.TargetReference[Key, Panel] {
	tables := make([]string, 0, len(resourceTabTypes))
	for t := range resourceTabTypes {
		tables = append(tables, string(t))
	}
	slices.Sort(tables)
	return gorp.NewTargetReference(gorp.TargetReferenceConfig[Key, Panel]{
		Name:     "root.tabs.resource",
		Tables:   tables,
		OnDelete: gorp.Detach,
		Targets: func(p *Panel) []gorp.Target {
			ids := ResourceRefs(p.Root)
			targets := make([]gorp.Target, len(ids))
			for i, id := range ids {
				targets[i] = tabTarget(id)
			}
			return targets
		},
		Detach: func(p *Panel, t gorp.Target) {
			p.Root = stripTabs(p.Root, func(id resource.ID) bool {
				return tabTarget(id) == t
			})
			collapseEmptyLeaves(&p.Root)
		},
	})
}

// CompositionMigrations returns the panel migrations that the service layer runs after
// every service table is open. The last one removes the tabs of resources the Core does
// not hold.
func (s *Service) CompositionMigrations() []gorp.Migration {
	return append(
		versions.CompositionMigrations(s.table),
		migrate.DropDangling(
			"v61_panel_dangling_tabs",
			s.cfg.DB,
			[]gorp.Reference[Key, Panel]{tabsReference()},
		),
	)
}
