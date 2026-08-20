// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package legacy decodes the version 0 project bundle every stable release exported: a
// flat directory whose LAYOUT.json holds the Console's persisted layout slice. Frozen —
// legacy bundles are no longer produced.
package legacy

import (
	"cmp"
	"context"
	"maps"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/panel"
	"github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/encoding/zip"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/spatial"
)

// LayoutFileName is the tiling file every stable release wrote at the root of a project
// export. It marks the legacy format only when no manifest is present.
const LayoutFileName = "LAYOUT.json"

// layoutSlice is the layout-slice body every stable release wrote to LAYOUT.json.
type layoutSlice struct {
	Layouts map[string]layout `json:"layouts"`
}

type layout struct {
	Key  string `json:"key"`
	Type string `json:"type"`
	Name string `json:"name"`
}

// importableTypes are the layout types whose component files import through the leaf
// machinery: typeless visualization states and typed task configs.
var importableTypes = set.New(
	string(ontology.ResourceTypeArc),
	string(ontology.ResourceTypeLineplot),
	string(ontology.ResourceTypeLog),
	string(ontology.ResourceTypeSchematic),
	string(ontology.ResourceTypeTable),
	"ethercat_read",
	"ethercat_write",
	"http_read",
	"http_write",
	"labjack_read",
	"labjack_write",
	"modbus_read",
	"modbus_write",
	"ni_analog_read",
	"ni_analog_write",
	"ni_counter_read",
	"ni_digital_read",
	"ni_digital_write",
	"opc_read",
	"opc_write",
	"pagerduty_alert",
)

// Member is an importable component the layout slice references.
type Member struct {
	// Path is the member file's path from the bundle root.
	Path string
	// Env is the member's decoded envelope, its type resolved to a registration type.
	Env imex.Envelope
	// LayoutKey is the key mosaic tabs reference the member by.
	LayoutKey string
}

// Members locates and decodes the component file behind each importable layout record
// in layoutData, in layout-key order. A record whose file the bundle does not hold is
// skipped: released Consoles wrote layouts whose component file the export never
// produced, and the tabs referencing one drop with it.
func Members(
	ctx context.Context,
	svc *imex.Service,
	layoutData []byte,
	files zip.Files,
) ([]Member, error) {
	var slice layoutSlice
	if err := json.Codec.Decode(ctx, layoutData, &slice); err != nil {
		return nil, errors.Wrap(err, LayoutFileName)
	}
	members := make([]Member, 0, len(slice.Layouts))
	for _, key := range slices.Sorted(maps.Keys(slice.Layouts)) {
		l := slice.Layouts[key]
		if !importableTypes.Contains(l.Type) {
			continue
		}
		path, ok := findComponent(ctx, files, key, l)
		if !ok {
			continue
		}
		var env imex.Envelope
		if err := json.Codec.Decode(ctx, files[path], &env); err != nil {
			return nil, errors.Wrap(err, path)
		}
		typ, err := svc.ResolveType(env)
		if err != nil {
			return nil, errors.Wrap(err, path)
		}
		env.Type = typ
		members = append(members, Member{Path: path, Env: env, LayoutKey: key})
	}
	return members, nil
}

// findComponent matches a layout record to a file named after the layout or its key, or
// one whose body carries the matching key or name. It reports false when the bundle
// holds no such file.
func findComponent(
	ctx context.Context,
	files zip.Files,
	key string,
	l layout,
) (string, bool) {
	paths := slices.Sorted(maps.Keys(files))
	for _, path := range paths {
		base := path[strings.LastIndexByte(path, '/')+1:]
		if base == l.Name+".json" || base == key+".json" {
			return path, true
		}
	}
	for _, path := range paths {
		if path == LayoutFileName || !strings.HasSuffix(path, ".json") {
			continue
		}
		var body map[string]any
		if err := json.Codec.Decode(ctx, files[path], &body); err != nil {
			continue
		}
		if body["key"] == key || body["name"] == l.Name {
			return path, true
		}
	}
	return "", false
}

// mainWindowKey is the key the Console used for the main window's mosaic.
const mainWindowKey = "main"

// tiling is the per-window mosaic portion of the persisted layout slice. It is decoded
// apart from layoutSlice so an unrecognized tiling shape drops the tiling instead of
// failing the whole import.
type tiling struct {
	Mosaics map[string]mosaic `json:"mosaics"`
}

type mosaic struct {
	Root *mosaicNode `json:"root"`
}

// mosaicNode is a node in the Console's persisted mosaic tree.
type mosaicNode struct {
	Tabs      []mosaicTab `json:"tabs"`
	Direction string      `json:"direction"`
	Size      float64     `json:"size"`
	First     *mosaicNode `json:"first"`
	Last      *mosaicNode `json:"last"`
}

type mosaicTab struct {
	// TabKey is the key of the layout the tab renders.
	TabKey string `json:"tabKey"`
}

// CreatePanels recreates layoutData's tiling as panels on tx, one per window whose tabs
// resolve through refs, main window first, each parented under parent. Best-effort: an
// unrecognized tiling shape creates no panels instead of failing.
func CreatePanels(
	ctx context.Context,
	tx gorp.Tx,
	svc *panel.Service,
	parent ontology.ID,
	layoutData []byte,
	refs map[string]ontology.ID,
) error {
	var t tiling
	if err := json.Codec.Decode(ctx, layoutData, &t); err != nil {
		return nil
	}
	var slice layoutSlice
	// The slice only names panels, so a failed decode falls back to window keys.
	_ = json.Codec.Decode(ctx, layoutData, &slice)
	writer := svc.NewWriter(tx)
	for _, windowKey := range sortedWindowKeys(t.Mosaics) {
		root := convertNode(t.Mosaics[windowKey].Root, refs)
		if root == nil {
			continue
		}
		p := panel.Panel{
			Name:   panelName(slice.Layouts, windowKey),
			Root:   *root,
			Parent: &parent,
		}
		if err := writer.Create(ctx, &p); err != nil {
			return errors.Wrap(err, LayoutFileName)
		}
	}
	return nil
}

// sortedWindowKeys orders window keys with main first, so the main window becomes the
// project's first panel.
func sortedWindowKeys(mosaics map[string]mosaic) []string {
	keys := slices.Collect(maps.Keys(mosaics))
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

func panelName(layouts map[string]layout, windowKey string) string {
	if l, ok := layouts[windowKey]; ok && l.Name != "" {
		return l.Name
	}
	return windowKey
}

// convertNode converts a mosaic node into a panel tree node, resolving each tab through
// refs. It returns nil when no tabs resolve; a split with one surviving side collapses
// into it.
func convertNode(n *mosaicNode, refs map[string]ontology.ID) *panel.Node {
	if n == nil {
		return nil
	}
	if n.First != nil || n.Last != nil {
		first := convertNode(n.First, refs)
		last := convertNode(n.Last, refs)
		if first == nil {
			return last
		}
		if last == nil {
			return first
		}
		return &panel.Node{Variant: panel.SplitNode{
			Direction: convertDirection(n.Direction),
			Size:      convertSize(n.Size),
			First:     *first,
			Last:      *last,
		}}
	}
	tabs := make([]panel.Tab, 0, len(n.Tabs))
	for _, t := range n.Tabs {
		id, ok := refs[t.TabKey]
		if !ok {
			continue
		}
		tabs = append(tabs, panel.Tab{Variant: panel.ResourceTab{
			TabBase:  panel.TabBase{Key: uuid.New()},
			Resource: id,
		}})
	}
	if len(tabs) == 0 {
		return nil
	}
	return &panel.Node{Variant: panel.LeafNode{Tabs: tabs}}
}

// HasPanels reports whether layoutData's tiling would convert to at least one panel, so
// callers can enforce access before any import work.
func HasPanels(ctx context.Context, layoutData []byte, members []Member) bool {
	var t tiling
	if err := json.Codec.Decode(ctx, layoutData, &t); err != nil {
		return false
	}
	refs := make(map[string]ontology.ID, len(members))
	for _, m := range members {
		refs[m.LayoutKey] = ontology.ID{}
	}
	for _, m := range t.Mosaics {
		if convertNode(m.Root, refs) != nil {
			return true
		}
	}
	return false
}

func convertDirection(d string) spatial.Direction {
	if parsed := spatial.Direction(d); parsed.IsValid() {
		return parsed
	}
	return spatial.DirectionX
}

// convertSize normalizes a split size to the (0, 1) fraction panels store; the
// persisted units drifted across releases, so anything else becomes an even split.
func convertSize(s float64) spatial.Decimal {
	if s > 0 && s < 1 {
		return spatial.Decimal(s)
	}
	return 0.5
}
