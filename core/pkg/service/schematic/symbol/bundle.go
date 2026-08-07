// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package symbol

import (
	"context"
	"encoding/json"
	"path"
	"regexp"
	"strings"

	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/encoding/zip"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
	"golang.org/x/text/unicode/norm"
)

const (
	// GroupManifestVersion is the manifest version this Core writes. Version 1 is the
	// legacy Console-written format, which carried a member list.
	GroupManifestVersion = 2
	// GroupManifestType is the manifest type of a symbol group bundle. Each endpoint
	// rejects a bundle of another kind.
	GroupManifestType = "symbol_group"
	// manifestFileName is the fixed recognition point for a bundle on disk.
	manifestFileName = "manifest.json"
)

// GroupManifest is the body of manifest.json in a symbol group bundle. Membership is
// inferred from the files beside the manifest, so it names no members.
type GroupManifest struct {
	Version int    `json:"version"`
	Type    string `json:"type"`
	Name    string `json:"name"`
}

// GroupBundle is an exported symbol group.
type GroupBundle struct {
	// Files holds the bundle contents keyed by file name: one envelope per symbol plus
	// manifest.json. The namespace is flat.
	Files zip.Files
	// Members holds the ontology ID of every exported symbol, in file-name order.
	Members []ontology.ID
}

// ExportGroup serializes every symbol in the group identified by key into a bundle.
// Each symbol becomes one JSON envelope named after the symbol, beside a manifest.json
// naming the group.
//
// It returns query.ErrNotFound if no group has key. It returns a validation error if
// the group holds a child that is not a schematic symbol, if two symbols resolve to the
// same file name, or if a symbol claims a reserved file name.
func (s *Service) ExportGroup(ctx context.Context, key group.Key) (GroupBundle, error) {
	root, children, err := s.retrieveGroup(ctx, nil, key)
	if err != nil {
		return GroupBundle{}, err
	}
	var (
		files   = make(zip.Files, len(children)+1)
		members = make([]ontology.ID, 0, len(children))
		// claimed maps each folded file name to the symbol that took it.
		claimed = make(map[string]string, len(children))
	)
	for _, child := range children {
		if child.ID.Type != ontology.ResourceTypeSchematicSymbol {
			return GroupBundle{}, errors.Wrapf(
				validate.ErrValidation,
				"cannot export group %q: child %s is not a schematic symbol",
				root.Name, child.ID,
			)
		}
		fileName := sanitizeFileName(child.Name) + ".json"
		folded := foldFileName(fileName)
		if isReservedFileName(folded) {
			return GroupBundle{}, errors.Wrapf(
				validate.ErrValidation,
				"symbol %q takes the reserved file name %q; rename it and export again",
				child.Name, fileName,
			)
		}
		if prev, ok := claimed[folded]; ok {
			return GroupBundle{}, errors.Wrapf(
				validate.ErrValidation,
				"symbols %q and %q both export to %q; rename one and export again",
				prev, child.Name, fileName,
			)
		}
		claimed[folded] = child.Name
		env, err := s.Export(ctx, child.ID)
		if err != nil {
			return GroupBundle{}, err
		}
		if files[fileName], err = json.Marshal(env); err != nil {
			return GroupBundle{}, err
		}
		members = append(members, child.ID)
	}
	manifest, err := json.Marshal(GroupManifest{
		Version: GroupManifestVersion,
		Type:    GroupManifestType,
		Name:    root.Name,
	})
	if err != nil {
		return GroupBundle{}, err
	}
	files[manifestFileName] = manifest
	return GroupBundle{Files: files, Members: members}, nil
}

// unsafeFileNameChars matches the path separators and Windows-reserved characters a
// file
// name cannot hold. Mirrors the Console's sanitizeFileName.
var unsafeFileNameChars = regexp.MustCompile(`[/\\<>:"|?*]`)

// sanitizeFileName replaces every character that is unsafe in a file name with an
// underscore.
func sanitizeFileName(name string) string {
	return unsafeFileNameChars.ReplaceAllString(name, "_")
}

// foldFileName reduces name to the form file names are compared in: Unicode-normalized
// and case-folded, because a bundle extracts onto case-insensitive filesystems.
func foldFileName(name string) string { return strings.ToLower(norm.NFC.String(name)) }

// isReservedFileName reports whether the folded name carries structural meaning in a
// bundle, so no member may take it. LAYOUT.json marks a legacy project directory.
func isReservedFileName(folded string) bool {
	base := strings.TrimSuffix(folded, path.Ext(folded))
	return base == "manifest" || base == "layout"
}
