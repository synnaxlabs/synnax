// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package project

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
	"github.com/synnaxlabs/x/validate"
)

// legacySlice is the layout-slice body every stable release wrote to LAYOUT.json; only
// the layout records are needed to locate and type each component file.
type legacySlice struct {
	Layouts map[string]legacyLayout `json:"layouts"`
}

// legacyLayout is one layout record of a legacy export.
type legacyLayout struct {
	Key  string `json:"key"`
	Type string `json:"type"`
	Name string `json:"name"`
}

// legacyComponentTypes are the visualization layout types found in legacy project
// exports. Their component files are typeless legacy Console states. Frozen — legacy
// exports are no longer produced.
var legacyComponentTypes = set.New(
	string(ontology.ResourceTypeArc),
	string(ontology.ResourceTypeLineplot),
	string(ontology.ResourceTypeLog),
	string(ontology.ResourceTypeSchematic),
	string(ontology.ResourceTypeTable),
)

// legacyTaskTypes are the task layout types found in legacy project exports. Their
// component files are typed legacy task configs the leaf machinery migrates. Frozen
// for the same reason.
var legacyTaskTypes = set.New(
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

// importLegacy recreates a legacy (version 0) project directory, named after the
// extension-stripped fileName: the documents its layout records reference, then each
// window's mosaic tiling as a panel of resource tabs.
func (s *Service) importLegacy(
	ctx context.Context,
	tx gorp.Tx,
	layoutData []byte,
	files zip.Files,
	fileName string,
) (Project, error) {
	var slice legacySlice
	if err := json.Codec.Decode(ctx, layoutData, &slice); err != nil {
		return Project{}, errors.Wrap(err, legacyLayoutFileName)
	}
	members, err := s.legacyMembers(ctx, slice, files)
	if err != nil {
		return Project{}, err
	}
	proj := Project{Name: imex.BaseName(fileName)}
	if err = s.NewWriter(tx).Create(ctx, &proj); err != nil {
		return Project{}, err
	}
	projID := OntologyID(proj.Key)
	refs := make(map[string]ontology.ID, len(members))
	for _, m := range members {
		id, err := s.cfg.ImEx.Import(ctx, tx, m.env, imex.ImportOptions{
			FileName: m.path,
			Parent:   projID,
		})
		if err != nil {
			return Project{}, errors.Wrap(err, m.path)
		}
		refs[m.layoutKey] = id
	}
	if err = s.createLegacyPanels(
		ctx, tx, projID, layoutData, slice.Layouts, refs,
	); err != nil {
		return Project{}, err
	}
	return proj, nil
}

// legacyMainWindowKey is the key the Console used for the main window's mosaic.
const legacyMainWindowKey = "main"

// legacyTiling is the per-window mosaic portion of the persisted layout slice. It is
// decoded apart from legacySlice so an unrecognized tiling shape drops the tiling
// instead of failing the whole import.
type legacyTiling struct {
	Mosaics map[string]legacyMosaic `json:"mosaics"`
}

// legacyMosaic is a single window's persisted mosaic state.
type legacyMosaic struct {
	Root *legacyMosaicNode `json:"root"`
}

// legacyMosaicNode is a node in the Console's persisted mosaic tree. Leaves carry
// tabs; interior nodes carry a direction, size, and two children.
type legacyMosaicNode struct {
	Tabs      []legacyMosaicTab `json:"tabs"`
	Direction string            `json:"direction"`
	Size      float64           `json:"size"`
	First     *legacyMosaicNode `json:"first"`
	Last      *legacyMosaicNode `json:"last"`
}

// legacyMosaicTab is a tab in a legacy mosaic leaf. TabKey is the key of the layout
// the tab renders.
type legacyMosaicTab struct {
	TabKey string `json:"tabKey"`
}

// createLegacyPanels recreates the legacy tiling as panels: each window's mosaic whose
// tabs resolve to imported members becomes a panel under the project, main window
// first. The tiling is best-effort — a mosaics shape this decoder does not recognize
// leaves the documents imported with no panels rather than failing the import.
func (s *Service) createLegacyPanels(
	ctx context.Context,
	tx gorp.Tx,
	projID ontology.ID,
	layoutData []byte,
	layouts map[string]legacyLayout,
	refs map[string]ontology.ID,
) error {
	var tiling legacyTiling
	if err := json.Codec.Decode(ctx, layoutData, &tiling); err != nil {
		return nil
	}
	writer := s.cfg.Panel.NewWriter(tx)
	for _, windowKey := range sortedLegacyWindowKeys(tiling.Mosaics) {
		root := convertLegacyNode(tiling.Mosaics[windowKey].Root, refs)
		if root == nil {
			continue
		}
		p := panel.Panel{
			Name:   legacyPanelName(layouts, windowKey),
			Root:   *root,
			Parent: &projID,
		}
		if err := writer.Create(ctx, &p); err != nil {
			return errors.Wrap(err, legacyLayoutFileName)
		}
	}
	return nil
}

// sortedLegacyWindowKeys orders the mosaic window keys deterministically, with the
// main window first so it becomes the project's first panel.
func sortedLegacyWindowKeys(mosaics map[string]legacyMosaic) []string {
	keys := slices.Collect(maps.Keys(mosaics))
	slices.SortFunc(keys, func(a, b string) int {
		if a == b {
			return 0
		}
		if a == legacyMainWindowKey {
			return -1
		}
		if b == legacyMainWindowKey {
			return 1
		}
		return cmp.Compare(a, b)
	})
	return keys
}

// legacyPanelName resolves a converted panel's name from the window's layout entry,
// falling back to the window key when the entry is missing or unnamed.
func legacyPanelName(layouts map[string]legacyLayout, windowKey string) string {
	if l, ok := layouts[windowKey]; ok && l.Name != "" {
		return l.Name
	}
	return windowKey
}

// convertLegacyNode recursively converts a legacy mosaic node into a panel tree node,
// resolving each tab to the ontology ID minted for its layout key. It returns nil when
// the subtree retains no resolvable tabs; a split with one surviving side collapses
// into that side.
func convertLegacyNode(
	n *legacyMosaicNode,
	refs map[string]ontology.ID,
) *panel.Node {
	if n == nil {
		return nil
	}
	if n.First != nil || n.Last != nil {
		first := convertLegacyNode(n.First, refs)
		last := convertLegacyNode(n.Last, refs)
		if first == nil {
			return last
		}
		if last == nil {
			return first
		}
		return &panel.Node{Variant: panel.SplitNode{
			Direction: convertLegacyDirection(n.Direction),
			Size:      convertLegacySize(n.Size),
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

// legacyHasPanels reports whether the legacy tiling would convert to at least one
// panel given the importable members, for access enforcement before any import work.
func legacyHasPanels(
	ctx context.Context,
	layoutData []byte,
	members []legacyMember,
) bool {
	var tiling legacyTiling
	if err := json.Codec.Decode(ctx, layoutData, &tiling); err != nil {
		return false
	}
	refs := make(map[string]ontology.ID, len(members))
	for _, m := range members {
		refs[m.layoutKey] = ontology.ID{}
	}
	for _, m := range tiling.Mosaics {
		if convertLegacyNode(m.Root, refs) != nil {
			return true
		}
	}
	return false
}

// convertLegacyDirection parses a legacy mosaic split direction, defaulting to x when
// the value is missing or invalid.
func convertLegacyDirection(d string) spatial.Direction {
	if parsed := spatial.Direction(d); parsed.IsValid() {
		return parsed
	}
	return spatial.DirectionX
}

// convertLegacySize normalizes a legacy split size to the (0, 1) fraction panels
// store. Legacy mosaics persisted sizes in several units over time, so anything
// outside the fractional range falls back to an even split.
func convertLegacySize(s float64) spatial.Decimal {
	if s > 0 && s < 1 {
		return spatial.Decimal(s)
	}
	return 0.5
}

// legacyMember pairs an importable layout record's component file with the layout key
// mosaic tabs reference it by.
type legacyMember struct {
	importMember
	layoutKey string
}

// legacyMembers locates the component file behind each importable layout record, in
// layout-key order, each decoded and resolved to its registration type. A record whose
// component file the directory does not hold is a validation error.
func (s *Service) legacyMembers(
	ctx context.Context,
	slice legacySlice,
	files zip.Files,
) ([]legacyMember, error) {
	members := make([]legacyMember, 0, len(slice.Layouts))
	for _, key := range slices.Sorted(maps.Keys(slice.Layouts)) {
		layout := slice.Layouts[key]
		if !legacyComponentTypes.Contains(layout.Type) &&
			!legacyTaskTypes.Contains(layout.Type) {
			continue
		}
		path, err := findLegacyComponent(ctx, files, key, layout)
		if err != nil {
			return nil, err
		}
		var env imex.Envelope
		if err := json.Codec.Decode(ctx, files[path], &env); err != nil {
			return nil, errors.Wrap(err, path)
		}
		typ, err := s.cfg.ImEx.ResolveType(env)
		if err != nil {
			return nil, errors.Wrap(err, path)
		}
		env.Type = typ
		members = append(members, legacyMember{
			importMember: importMember{path: path, env: env},
			layoutKey:    key,
		})
	}
	return members, nil
}

// findLegacyComponent locates the component file behind a layout record: a file named
// after the layout or its key, or one whose body carries the matching key or name.
func findLegacyComponent(
	ctx context.Context,
	files zip.Files,
	key string,
	layout legacyLayout,
) (string, error) {
	paths := slices.Sorted(maps.Keys(files))
	for _, path := range paths {
		base := path[strings.LastIndexByte(path, '/')+1:]
		if base == layout.Name+".json" || base == key+".json" {
			return path, nil
		}
	}
	for _, path := range paths {
		if path == legacyLayoutFileName || !strings.HasSuffix(path, ".json") {
			continue
		}
		var body map[string]any
		if err := json.Codec.Decode(ctx, files[path], &body); err != nil {
			continue
		}
		if body["key"] == key || body["name"] == layout.Name {
			return path, nil
		}
	}
	return "", errors.Wrapf(
		validate.ErrValidation, "data for layout %q not found", key,
	)
}
