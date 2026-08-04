// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	project "github.com/synnaxlabs/synnax/pkg/service/project/versions/v1"
	task "github.com/synnaxlabs/synnax/pkg/service/task/versions/v2"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/migrate"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/spatial"
	"go.uber.org/zap"
)

// codecMigration re-encodes stored panels from MessagePack to Orc.
var codecMigration = gorp.CodecMigration[Key, Panel]("msgpack_to_orc")

// migratableLayoutTypes maps the Console layout types whose layout key is the key of a
// backing core document to that document's ontology resource type. Tabs of any other
// layout type (inline app views, modals) have no server-side document to reference and
// are dropped by the migration.
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

// stagedLayout is a project layout gathered from the staging prefix for conversion: the
// KV key it was read from, the owning project's ontology ID, and the parsed mosaic
// state.
type stagedLayout struct {
	key       []byte
	projectID ontology.ID
	slice     legacySlice
}

// migrateProjectLayouts returns a migration that converts the legacy layout blobs the
// project migration stages under projectv1.LegacyLayoutKVPrefix into panels, deleting
// each staged entry as it is consumed, so this migration never reads the project layout
// field directly. Every window mosaic that references at least one live visualization
// document becomes a panel parented under the project. Tabs whose layout entry is
// missing, whose layout type has no backing document, or whose document no longer
// exists are dropped; splits that lose a side collapse into the surviving child. Blobs
// that cannot be parsed are skipped, since the Console wrote them best-effort and an
// unreadable layout must not block the upgrade.
func migrateProjectLayouts() func(context.Context, gorp.Tx, alamos.Instrumentation) error {
	return func(ctx context.Context, tx gorp.Tx, ins alamos.Instrumentation) error {
		staged, err := scanStagedLayouts(tx, ins)
		if err != nil {
			return err
		}
		for _, s := range staged {
			if err = createPanels(ctx, tx, s.projectID, s.slice); err != nil {
				return err
			}
			if err = tx.Delete(ctx, s.key); err != nil {
				return err
			}
		}
		return nil
	}
}

// scanStagedLayouts gathers every staged layout before any are converted, since writing
// panels while iterating the staging prefix would be unsafe. Blobs that fail to parse
// are skipped with a warning.
func scanStagedLayouts(
	tx gorp.Tx,
	ins alamos.Instrumentation,
) (out []stagedLayout, err error) {
	iter, err := tx.OpenIterator(kv.IterPrefix([]byte(project.LegacyLayoutKVPrefix)))
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Combine(err, iter.Close()) }()
	for iter.First(); iter.Valid(); iter.Next() {
		key := bytes.Clone(iter.Key())
		var slice legacySlice
		if jerr := json.Unmarshal(iter.Value(), &slice); jerr != nil {
			ins.L.Warn(
				"skipping project with unparseable staged layout",
				zap.String("key", string(key)),
				zap.Error(jerr),
			)
			continue
		}
		out = append(out, stagedLayout{
			key: key,
			projectID: ontology.ID{
				Type: ontology.ResourceTypeProject,
				Key:  strings.TrimPrefix(string(key), project.LegacyLayoutKVPrefix),
			},
			slice: slice,
		})
	}
	return out, iter.Error()
}

// createPanels converts every migratable window mosaic in slice into a panel owned by
// the project with the given ontology ID.
func createPanels(
	ctx context.Context,
	tx gorp.Tx,
	projectID ontology.ID,
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
		pan := Panel{
			Key:  uuid.New(),
			Name: panelName(slice.Layouts, windowKey),
			Root: *root,
		}
		if err := panelWriter.Set(ctx, pan); err != nil {
			return err
		}
		panelID := pan.OntologyID()
		if err := resourceWriter.Set(ctx, ontology.Resource{ID: panelID}); err != nil {
			return err
		}
		if err := relWriter.Set(ctx, ontology.Relationship{
			From: projectID,
			Type: ontology.RelationshipTypeParentOf,
			To:   panelID,
		}); err != nil {
			return err
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
		if a == b {
			return 0
		}
		if a == mainWindowKey {
			return -1
		}
		if b == mainWindowKey {
			return 1
		}
		return cmp.Compare(a, b)
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
		return &Node{Variant: NodeSplit{Split: Split{
			Direction: convertDirection(n.Direction),
			Size:      convertSize(n.Size),
			First:     *first,
			Last:      *last,
		}}}, nil
	}
	tabs := make([]Tab, 0, len(n.Tabs))
	for _, t := range n.Tabs {
		layout, ok := layouts[t.TabKey]
		if !ok {
			continue
		}
		resourceType, ok := migratableLayoutTypes[layout.Type]
		if !ok {
			tab, err := convertTaskTab(ctx, tx, t.TabKey)
			if err != nil {
				return nil, err
			}
			if tab != nil {
				tabs = append(tabs, *tab)
			}
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
		tabs = append(tabs, Tab{Variant: TabResource{
			TabBase:  TabBase{Key: uuid.New()},
			Resource: id,
		}})
	}
	if len(tabs) == 0 {
		return nil, nil
	}
	return &Node{Variant: NodeLeaf{Leaf: Leaf{Tabs: tabs}}}, nil
}

// convertTaskTab converts a legacy task layout tab into a resource tab pointing at
// the task's re-keyed UUID. Legacy task layouts were keyed by the task's uint64 key,
// so a tab is a task tab exactly when its key is in the staging map written by the
// task re-key migration; other tab keys are dropped.
func convertTaskTab(ctx context.Context, tx gorp.Tx, tabKey string) (*Tab, error) {
	val, closer, err := tx.Get(ctx, []byte(task.LegacyKeyKVPrefix+tabKey))
	if err != nil {
		if errors.Is(err, query.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	key := string(val)
	if err := closer.Close(); err != nil {
		return nil, err
	}
	return &Tab{Variant: TabResource{
		TabBase:  TabBase{Key: uuid.New()},
		Resource: ontology.ID{Type: ontology.ResourceTypeTask, Key: key},
	}}, nil
}

// MigrateTaskTabKeys converts every panel view tab holding a legacy uint64 task key
// into a resource tab pointing at the UUID minted by the task re-key migration, then
// drains the staging map.
func MigrateTaskTabKeys(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
	mapping := make(map[string]string)
	var stagedKeys [][]byte
	iter, err := tx.OpenIterator(kv.IterPrefix([]byte(task.LegacyKeyKVPrefix)))
	if err != nil {
		return err
	}
	for iter.First(); iter.Valid(); iter.Next() {
		key := bytes.Clone(iter.Key())
		stagedKeys = append(stagedKeys, key)
		legacy := strings.TrimPrefix(string(key), task.LegacyKeyKVPrefix)
		mapping[legacy] = string(iter.Value())
	}
	if err := errors.Combine(iter.Error(), iter.Close()); err != nil {
		return err
	}
	if len(mapping) == 0 {
		return nil
	}
	var panels []Panel
	if err := gorp.NewRetrieve[Key, Panel]().
		Entries(&panels).
		Exec(ctx, tx); err != nil && !errors.Is(err, query.ErrNotFound) {
		return err
	}
	w := gorp.WrapWriter[Key, Panel](tx)
	for _, p := range panels {
		if !convertNodeTaskTabs(&p.Root, mapping) {
			continue
		}
		if err := w.Set(ctx, p); err != nil {
			return err
		}
	}
	for _, key := range stagedKeys {
		if err := tx.Delete(ctx, key); err != nil {
			return err
		}
	}
	return nil
}

// convertNodeTaskTabs converts legacy-keyed task view tabs under n into resource
// tabs, reporting whether anything changed.
func convertNodeTaskTabs(n *Node, mapping map[string]string) bool {
	switch v := n.Variant.(type) {
	case NodeSplit:
		first := convertNodeTaskTabs(&v.First, mapping)
		last := convertNodeTaskTabs(&v.Last, mapping)
		if !first && !last {
			return false
		}
		n.Variant = v
		return true
	case NodeLeaf:
		changed := false
		for i := range v.Tabs {
			tab := &v.Tabs[i]
			view, ok := tab.Variant.(TabView)
			if !ok {
				continue
			}
			legacy, ok := view.Args["taskKey"].(string)
			if !ok {
				continue
			}
			key, ok := mapping[legacy]
			if !ok {
				continue
			}
			tab.Variant = TabResource{
				TabBase:  view.TabBase,
				Resource: ontology.ID{Type: ontology.ResourceTypeTask, Key: key},
			}
			changed = true
		}
		if !changed {
			return false
		}
		n.Variant = v
		return true
	}
	return false
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

// projectLayoutsMigration adopts the project service's staged legacy layouts as panels.
// It runs after the codec migration so it always reads Orc-encoded panels.
var projectLayoutsMigration = gorp.NewMigration(
	"v56_migrate_project_layouts_to_panels", migrateProjectLayouts(),
)

// taskTabKeysMigration re-keys task view tabs from the legacy-to-UUID mapping staged
// by the task re-key migration. It runs after the project-layouts migration so
// converted tabs are re-keyed in the same pass.
var taskTabKeysMigration = gorp.NewMigration(
	"v56_task_tab_uuid_keys", MigrateTaskTabKeys,
)

// Migrations is the ordered set of migrations introduced at this version.
var Migrations = []migrate.Migration{
	codecMigration,
	projectLayoutsMigration,
	taskTabKeysMigration,
}
