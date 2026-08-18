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
	"context"
	"maps"
	"slices"
	"strings"

	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/encoding/zip"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/set"
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

// importLegacy recreates the documents of a legacy (version 0) project directory, named
// after the extension-stripped fileName. The mosaic tiling is dropped (SY-4370), so
// only the documents are recreated.
func (s *Service) importLegacy(
	ctx context.Context,
	tx gorp.Tx,
	layoutData []byte,
	files zip.Files,
	fileName string,
) (Project, error) {
	members, err := s.legacyMembers(ctx, layoutData, files)
	if err != nil {
		return Project{}, err
	}
	proj := Project{Name: imex.BaseName(fileName)}
	if err = s.NewWriter(tx).Create(ctx, &proj); err != nil {
		return Project{}, err
	}
	projID := OntologyID(proj.Key)
	for _, m := range members {
		if _, err = s.cfg.ImEx.Import(ctx, tx, m.env, imex.ImportOptions{
			FileName: m.path,
			Parent:   projID,
		}); err != nil {
			return Project{}, errors.Wrap(err, m.path)
		}
	}
	return proj, nil
}

// legacyMembers locates the component file behind each importable layout record, in
// layout-key order, each decoded and resolved to its registration type. A record whose
// component file the directory does not hold is a validation error.
func (s *Service) legacyMembers(
	ctx context.Context,
	layoutData []byte,
	files zip.Files,
) ([]importMember, error) {
	var slice legacySlice
	if err := json.Codec.Decode(ctx, layoutData, &slice); err != nil {
		return nil, errors.Wrap(err, legacyLayoutFileName)
	}
	members := make([]importMember, 0, len(slice.Layouts))
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
		members = append(members, importMember{path: path, env: env})
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
