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
	"context"
	"slices"

	"github.com/google/uuid"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/spatial"
	"go.uber.org/zap"
)

// migratableLayoutTypes maps the Console layout types whose layout key is the key of
// a backing core document to that document's ontology resource type. Tabs of any
// other layout type (inline app views, modals) have no server-side document to
// reference and are dropped by the migration.
var migratableLayoutTypes = map[string]ontology.ResourceType{
	"arc":       ontology.ResourceTypeArc,
	"lineplot":  ontology.ResourceTypeLineplot,
	"log":       ontology.ResourceTypeLog,
	"schematic": ontology.ResourceTypeSchematic,
	"table":     ontology.ResourceTypeTable,
}

// legacyLayout is the subset of the Console's persisted layout record consumed by the
// migration.
type legacyLayout struct {
	// Type identifies the Console renderer for the layout and, for visualization
	// layouts, the ontology resource type of the backing document.
	Type string `json:"type"`
	// Name is the human-readable layout name.
	Name string `json:"name"`
}

// legacyTab is a tab in a legacy mosaic leaf. TabKey is the key of the layout the tab
// renders.
type legacyTab struct {
	TabKey string `json:"tabKey"`
}

// legacyNode is a node in the Console's persisted mosaic tree. Leaves carry tabs;
// interior nodes carry a direction, size, and two children.
type legacyNode struct {
	Tabs      []legacyTab `json:"tabs"`
	Direction string      `json:"direction"`
	Size      float64     `json:"size"`
	First     *legacyNode `json:"first"`
	Last      *legacyNode `json:"last"`
}

// legacyMosaic is a single window's persisted mosaic state.
type legacyMosaic struct {
	Root *legacyNode `json:"root"`
}

// legacySlice is the subset of the Console's persisted layout slice consumed by the
// migration: the per-window mosaic trees and the layout record they reference.
type legacySlice struct {
	Mosaics map[string]legacyMosaic `json:"mosaics"`
	Layouts map[string]legacyLayout `json:"layouts"`
}

// mainWindowKey is the key the Console uses for the main window's mosaic.
const mainWindowKey = "main"

// MigrateProjectLayouts returns a migration that converts each project's legacy
// layout blob into panels. Every window mosaic that references at least one live
// visualization document becomes a panel parented under the project and the panels
// group identified by groupID. Tabs whose layout entry is missing, whose layout type
// has no backing document, or whose document no longer exists are dropped; splits
// that lose a side collapse into the surviving child. Blobs that cannot be parsed
// are skipped, since the Console wrote them best-effort and an unreadable layout
// must not block the upgrade.
func MigrateProjectLayouts(
	groupID ontology.ID,
) func(context.Context, gorp.Tx, alamos.Instrumentation) error {
	return func(ctx context.Context, tx gorp.Tx, ins alamos.Instrumentation) error {
		projects, err := collectProjects(ctx, tx)
		if err != nil {
			return err
		}
		for _, p := range projects {
			if len(p.Layout) == 0 {
				continue
			}
			var slice legacySlice
			if err := p.Layout.Unmarshal(&slice); err != nil {
				ins.L.Warn(
					"skipping project with unparseable layout",
					zap.String("project", p.Key.String()),
					zap.Error(err),
				)
				continue
			}
			if err := createPanels(ctx, tx, groupID, p, slice); err != nil {
				return err
			}
		}
		return nil
	}
}

// createPanels converts every migratable window mosaic in slice into a panel owned
// by p.
func createPanels(
	ctx context.Context,
	tx gorp.Tx,
	groupID ontology.ID,
	p project.Project,
	slice legacySlice,
) error {
	panelWriter := gorp.WrapWriter[Key, Panel](tx)
	resourceWriter := gorp.WrapWriter[string, ontology.Resource](tx)
	relWriter := gorp.WrapWriter[string, ontology.Relationship](tx)
	for _, windowKey := range sortedWindowKeys(slice.Mosaics) {
		root, err := convertNode(ctx, tx, slice.Mosaics[windowKey].Root, slice.Layouts)
		if err != nil {
			return err
		}
		if root == nil {
			continue
		}
		pan := Panel{Key: uuid.New(), Name: panelName(slice.Layouts, windowKey), Root: *root}
		if err := panelWriter.Set(ctx, pan); err != nil {
			return err
		}
		panelID := OntologyID(pan.Key)
		if err := resourceWriter.Set(ctx, ontology.Resource{ID: panelID}); err != nil {
			return err
		}
		for _, parent := range []ontology.ID{groupID, project.OntologyID(p.Key)} {
			if err := relWriter.Set(ctx, ontology.Relationship{
				From: parent,
				Type: ontology.RelationshipTypeParentOf,
				To:   panelID,
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

// sortedWindowKeys orders the mosaic window keys deterministically, with the main
// window first so it becomes the project's first panel.
func sortedWindowKeys(mosaics map[string]legacyMosaic) []string {
	keys := make([]string, 0, len(mosaics))
	for k := range mosaics {
		keys = append(keys, k)
	}
	slices.SortFunc(keys, func(a, b string) int {
		if a == mainWindowKey {
			return -1
		}
		if b == mainWindowKey {
			return 1
		}
		if a < b {
			return -1
		}
		if a > b {
			return 1
		}
		return 0
	})
	return keys
}

// panelName resolves a migrated panel's name from the window's layout entry, falling
// back to the window key when the entry is missing or unnamed.
func panelName(layouts map[string]legacyLayout, windowKey string) string {
	if l, ok := layouts[windowKey]; ok && l.Name != "" {
		return l.Name
	}
	return windowKey
}

// convertNode recursively converts a legacy mosaic node into a panel tree node. It
// returns nil when the subtree retains no migratable tabs; a split with one surviving
// side collapses into that side.
func convertNode(
	ctx context.Context,
	tx gorp.Tx,
	n *legacyNode,
	layouts map[string]legacyLayout,
) (*Node, error) {
	if n == nil {
		return nil, nil
	}
	if n.First != nil || n.Last != nil {
		first, err := convertNode(ctx, tx, n.First, layouts)
		if err != nil {
			return nil, err
		}
		last, err := convertNode(ctx, tx, n.Last, layouts)
		if err != nil {
			return nil, err
		}
		if first == nil {
			return last, nil
		}
		if last == nil {
			return first, nil
		}
		return &Node{Split: &Split{
			Direction: convertDirection(n.Direction),
			Size:      convertSize(n.Size),
			First:     first,
			Last:      last,
		}}, nil
	}
	tabs := make([]Tab, 0, len(n.Tabs))
	for _, t := range n.Tabs {
		layout, ok := layouts[t.TabKey]
		if !ok {
			continue
		}
		resourceType, ok := migratableLayoutTypes[layout.Type]
		if !ok {
			continue
		}
		id := ontology.ID{Type: resourceType, Key: t.TabKey}
		exists, err := gorp.NewRetrieve[string, ontology.Resource]().
			Where(gorp.MatchKeys[string, ontology.Resource](id.String())).
			Exists(ctx, tx)
		if err != nil {
			return nil, err
		}
		if !exists {
			continue
		}
		tabs = append(tabs, Tab{Key: uuid.New(), Resource: &id})
	}
	if len(tabs) == 0 {
		return nil, nil
	}
	return &Node{Leaf: &Leaf{Tabs: tabs}}, nil
}

// convertDirection parses a legacy mosaic split direction, defaulting to x when the
// value is missing or invalid.
func convertDirection(d string) spatial.Direction {
	if parsed := spatial.Direction(d); parsed.IsValid() {
		return parsed
	}
	return spatial.DirectionX
}

// convertSize normalizes a legacy split size to the (0, 1) fraction panels store.
// Legacy mosaics persisted sizes in several units over time, so anything outside the
// fractional range falls back to an even split.
func convertSize(s float64) float64 {
	if s > 0 && s < 1 {
		return s
	}
	return 0.5
}

// collectProjects drains every project record into a slice. Mutating other gorp
// prefixes while iterating is unsafe, so the migration gathers first and writes
// after.
func collectProjects(
	ctx context.Context,
	tx gorp.Tx,
) (out []project.Project, err error) {
	iter, err := gorp.WrapReader[project.Key, project.Project](tx).
		OpenIterator(gorp.IterOptions{})
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Combine(err, iter.Close()) }()
	for iter.First(); iter.Valid(); iter.Next() {
		if p := iter.Value(ctx); p != nil {
			out = append(out, *p)
		}
	}
	return out, iter.Error()
}
