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
	"encoding/json/jsontext"
	"encoding/json/v2"
	"slices"
	"strings"
	"uuid"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	ontology "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v0"
	project "github.com/synnaxlabs/synnax/pkg/service/project/versions/v1"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	task "github.com/synnaxlabs/synnax/pkg/service/task/versions/v2"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/spatial"
	"go.uber.org/zap"
)

// Create stores a panel converted from a legacy layout of the project with the given
// resource ID.
type Create func(ctx context.Context, tx gorp.Tx, project resource.ID, p Panel) error

// CompositionMigrations adopts other services' staged legacy data (project layouts,
// task re-key mapping) into panels, passing each new panel to create. The service layer
// runs them after every table is open, since the producing services open after the
// panel table.
func CompositionMigrations(create Create) []gorp.Migration {
	return []gorp.Migration{
		migrate.New(
			"v56_migrate_project_layouts_to_panels",
			func(ctx context.Context, tx gorp.Tx, ins alamos.Instrumentation) error {
				return migrateProjectLayouts(ctx, tx, ins, create)
			},
		),
		taskKeysMigration,
	}
}

// migratableLayoutTypes maps the Console layout types whose layout key is the key of a
// backing core document to that document's ontology resource type. Consoles up to v0.50
// stored Arc editors as arc_editor.
var migratableLayoutTypes = map[string]resource.Type{
	"arc":        resource.TypeArc,
	"arc_editor": resource.TypeArc,
	"lineplot":   resource.TypeLineplot,
	"log":        resource.TypeLog,
	"overview":   resource.TypeRange,
	"schematic":  resource.TypeSchematic,
	"table":      resource.TypeTable,
}

// viewLayoutTypes maps the legacy Console layout types that render inline app views
// (no backing document) to the view type the current Console renders them as in a
// panel. Both legacy new-tab selectors collapse into the current selector picker.
var viewLayoutTypes = map[string]string{
	"range_explorer":        "range_explorer",
	"status_explorer":       "status_explorer",
	"taskSelector":          "taskSelector",
	"layoutSelector":        "selector",
	"visualizationSelector": "selector",
}

// legacyLayout is the subset of the Console's persisted layout record consumed by the
// migration.
type legacyLayout struct {
	// Type identifies the Console renderer for the layout and, for visualization
	// layouts, the ontology resource type of the backing document.
	Type string `json:"type"`
	// Name is the human-readable layout name.
	Name string `json:"name"`
	// Args is the layout's renderer arguments, kept raw so one malformed record
	// cannot fail the whole blob. Task layouts carry the task's key in args.
	Args jsontext.Value `json:"args"`
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
// migration: the per-window mosaic trees, the layout records they reference, and the
// layout-key-to-alternate-key map task layouts register their task key in.
type legacySlice struct {
	Mosaics     map[string]legacyMosaic `json:"mosaics"`
	Layouts     map[string]legacyLayout `json:"layouts"`
	KeyToAltKey map[string]string       `json:"keyToAltKey"`
}

// mainWindowKey is the key the Console uses for the main window's mosaic.
const mainWindowKey = "main"

// stagedLayout is a project layout gathered from the staging prefix for conversion: the
// KV key it was read from, the owning project's resource ID, and the parsed mosaic
// state.
type stagedLayout struct {
	key       []byte
	projectID resource.ID
	slice     legacySlice
}

// migrateProjectLayouts converts the legacy layout blobs the project migration stages
// under projectv1.LegacyLayoutKVPrefix into panels, deleting each staged entry as it is
// consumed, so this migration never reads the project layout field directly. Every
// window mosaic that retains at least one convertible tab becomes a panel of the
// project: document-backed tabs become resource tabs, and app views with a current
// equivalent become view tabs. Tabs whose layout entry is missing, whose type
// has no conversion, or whose document no longer exists are dropped; splits that lose a
// side collapse into the surviving child. Blobs that cannot be parsed are skipped,
// since the Console wrote them best-effort and an unreadable layout must not block the
// upgrade.
func migrateProjectLayouts(
	ctx context.Context,
	tx gorp.Tx,
	ins alamos.Instrumentation,
	create Create,
) error {
	staged, err := scanStagedLayouts(tx, ins)
	if err != nil {
		return err
	}
	for _, s := range staged {
		if err = createPanels(ctx, tx, s.projectID, s.slice, create); err != nil {
			return err
		}
		if err = tx.Delete(ctx, s.key); err != nil {
			return err
		}
	}
	return nil
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
			projectID: resource.ID{
				Type: resource.TypeProject,
				Key:  strings.TrimPrefix(string(key), project.LegacyLayoutKVPrefix),
			},
			slice: slice,
		})
	}
	return out, iter.Error()
}

// createPanels converts every migratable window mosaic in slice into a panel of the
// project with the given resource ID.
func createPanels(
	ctx context.Context,
	tx gorp.Tx,
	projectID resource.ID,
	slice legacySlice,
	create Create,
) error {
	for _, windowKey := range sortedWindowKeys(slice.Mosaics) {
		root, err := convertNode(ctx, tx, slice.Mosaics[windowKey].Root, slice)
		if err != nil {
			return err
		}
		if root == nil {
			continue
		}
		if err := create(ctx, tx, projectID, Panel{
			Key:  uuid.New(),
			Name: panelName(slice.Layouts, windowKey),
			Root: *root,
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
	slice legacySlice,
) (*Node, error) {
	if n == nil {
		return nil, nil
	}
	if n.First != nil || n.Last != nil {
		first, err := convertNode(ctx, tx, n.First, slice)
		if err != nil {
			return nil, err
		}
		last, err := convertNode(ctx, tx, n.Last, slice)
		if err != nil {
			return nil, err
		}
		if first == nil {
			return last, nil
		}
		if last == nil {
			return first, nil
		}
		return &Node{Variant: SplitNode{
			Direction: convertDirection(n.Direction),
			Size:      convertSize(n.Size),
			First:     *first,
			Last:      *last,
		}}, nil
	}
	tabs := make([]Tab, 0, len(n.Tabs))
	for _, t := range n.Tabs {
		layout, ok := slice.Layouts[t.TabKey]
		if !ok {
			continue
		}
		resourceType, ok := migratableLayoutTypes[layout.Type]
		if !ok {
			if viewType, ok := viewLayoutTypes[layout.Type]; ok {
				tabs = append(tabs, Tab{Variant: ViewTab{
					Key:  uuid.New(),
					Type: viewType,
				}})
				continue
			}
			tab, err := convertTaskTab(ctx, tx, taskKeyCandidates(slice, t.TabKey))
			if err != nil {
				return nil, err
			}
			if tab != nil {
				tabs = append(tabs, *tab)
			}
			continue
		}
		id := resource.ID{Type: resourceType, Key: t.TabKey}
		exists, err := gorp.NewRetrieve[string, ontology.Resource]().
			Where(gorp.MatchKeys[string, ontology.Resource](id.String())).
			Exists(ctx, tx)
		if err != nil {
			return nil, err
		}
		if !exists {
			continue
		}
		tabs = append(tabs, Tab{Variant: ResourceTab{Key: uuid.New(), Resource: id}})
	}
	if len(tabs) == 0 {
		return nil, nil
	}
	return &Node{Variant: LeafNode{Tabs: tabs}}, nil
}

// taskKeyCandidates lists every place a legacy mosaic tab can carry its task's key.
// Tabs for tasks opened from the tree use the task key as the layout key; tabs for
// tasks created in-session keep a placeholder layout key and carry the task key in
// the layout's args and the slice's alt-key map.
func taskKeyCandidates(slice legacySlice, tabKey string) []string {
	candidates := []string{tabKey}
	if l, ok := slice.Layouts[tabKey]; ok && len(l.Args) > 0 {
		var args struct {
			TaskKey string `json:"taskKey"`
		}
		if err := json.Unmarshal(l.Args, &args); err == nil && args.TaskKey != "" {
			candidates = append(candidates, args.TaskKey)
		}
	}
	if alt, ok := slice.KeyToAltKey[tabKey]; ok && alt != "" {
		candidates = append(candidates, alt)
	}
	return candidates
}

// convertTaskTab converts a legacy task layout tab into a resource tab pointing at
// the task's re-keyed UUID. A tab is a task tab exactly when one of its candidate
// keys is in the staging map written by the task re-key migration; other tabs are
// dropped.
func convertTaskTab(
	ctx context.Context,
	tx gorp.Tx,
	candidates []string,
) (*Tab, error) {
	for _, candidate := range candidates {
		val, closer, err := tx.Get(ctx, []byte(task.LegacyKeyKVPrefix+candidate))
		if err != nil {
			if errors.Is(err, query.ErrNotFound) {
				continue
			}
			return nil, err
		}
		key := string(val)
		if err := closer.Close(); err != nil {
			return nil, err
		}
		return &Tab{Variant: ResourceTab{
			Key:      uuid.New(),
			Resource: resource.ID{Type: resource.TypeTask, Key: key},
		}}, nil
	}
	return nil, nil
}

// drainTaskKeys deletes the legacy-to-UUID task key mapping the task re-key migration
// stages, once the project layouts migration has read it.
func drainTaskKeys(ctx context.Context, tx gorp.Tx, _ alamos.Instrumentation) error {
	var stagedKeys [][]byte
	iter, err := tx.OpenIterator(kv.IterPrefix([]byte(task.LegacyKeyKVPrefix)))
	if err != nil {
		return err
	}
	for iter.First(); iter.Valid(); iter.Next() {
		stagedKeys = append(stagedKeys, bytes.Clone(iter.Key()))
	}
	if err := errors.Combine(iter.Error(), iter.Close()); err != nil {
		return err
	}
	for _, key := range stagedKeys {
		if err := tx.Delete(ctx, key); err != nil {
			return err
		}
	}
	return nil
}

// resourceTabTypes is the set of resource types a tab can display, as the panel schema
// currently enforces.
var resourceTabTypes = set.New(
	resource.TypeSchematic,
	resource.TypeLineplot,
	resource.TypeLog,
	resource.TypeTable,
	resource.TypeArc,
	resource.TypeTask,
	resource.TypeRange,
)

// stripInvalidResourceTabs removes every resource tab whose type is outside
// resourceTabTypes from persisted panels, so panels written before the type whitelist
// still validate on their next dispatch.
func stripInvalidResourceTabs(
	ctx context.Context,
	tx gorp.Tx,
	_ alamos.Instrumentation,
) error {
	var panels []Panel
	if err := gorp.NewRetrieve[Key, Panel]().
		Entries(&panels).
		Exec(ctx, tx); err != nil && !errors.Is(err, query.ErrNotFound) {
		return err
	}
	w := gorp.WrapWriter[Key, Panel](tx)
	for _, p := range panels {
		if !stripNodeInvalidTabs(&p.Root) {
			continue
		}
		if err := w.Set(ctx, p); err != nil {
			return err
		}
	}
	return nil
}

// stripNodeInvalidTabs removes resource tabs with a type outside resourceTabTypes
// under n, reporting whether anything changed. Emptied leaves stay in place: the
// schema allows them.
func stripNodeInvalidTabs(n *Node) bool {
	switch v := n.Variant.(type) {
	case SplitNode:
		first := stripNodeInvalidTabs(&v.First)
		last := stripNodeInvalidTabs(&v.Last)
		if !first && !last {
			return false
		}
		n.Variant = v
		return true
	case LeafNode:
		tabs := make([]Tab, 0, len(v.Tabs))
		for _, t := range v.Tabs {
			if r, ok := t.Variant.(ResourceTab); ok &&
				!resourceTabTypes.Contains(r.Resource.Type) {
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

// taskKeysMigration drains the staged task key mapping. Panels and task UUID keys
// shipped in the same release, so no stored panel holds a legacy task key.
var taskKeysMigration = migrate.New("v56_task_tab_uuid_keys", drainTaskKeys)

// ResourceTabTypesMigration strips the resource tabs the type whitelist rejects.
var ResourceTabTypesMigration = migrate.New(
	"v56_strip_invalid_resource_tabs", stripInvalidResourceTabs,
)
