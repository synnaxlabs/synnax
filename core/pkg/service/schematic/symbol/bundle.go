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

	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/encoding"
	"github.com/synnaxlabs/x/encoding/zip"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/os"
	"github.com/synnaxlabs/x/validate"
)

const (
	manifestVersion = 2
	manifestType    = "symbol_group"
	// manifestBaseName is the manifest's file name without an extension. The codec
	// supplies the extension, so every serialization has one recognition point.
	manifestBaseName = "manifest"
)

// GroupManifest is the body of manifest.json in a symbol group bundle. Membership is
// inferred from the files beside the manifest, so it names no members.
type GroupManifest struct {
	// Version governs the manifest schema and the bundle's layout rules.
	Version uint64 `json:"version"`
	// Type is the bundle kind, letting an endpoint reject a bundle of another kind.
	Type string `json:"type"`
	// Name is the group's name, which an import gives the group it creates.
	Name string `json:"name"`
}

// GroupBundle is an exported symbol group.
type GroupBundle struct {
	// Files holds the bundle contents keyed by file name: one envelope per symbol plus
	// manifest.json. The namespace is flat.
	Files zip.Files
	// Members holds the ontology ID of every exported symbol, in the order the ontology
	// returned the group's children. Members never reaches the wire: the archive
	// carries Files alone.
	Members []ontology.ID
}

var _ zip.Marshaler = GroupBundle{}

// MarshalZIP returns the bundle's files, so a transport can archive the bundle itself
// instead of reaching for its contents.
func (b GroupBundle) MarshalZIP() (zip.Files, error) { return b.Files, nil }

// ExportGroup serializes every symbol in the group identified by key into a bundle.
// Each symbol becomes one envelope named after the symbol, beside a manifest naming the
// group. The encoder decides both the serialization and the extension every file takes.
//
// It returns query.ErrNotFound if no group has key. It returns a validation error if
// the group holds a child that is not a schematic symbol, if a symbol's name holds no
// character a file name can keep, if two symbols resolve to the same file name, or if a
// symbol claims a reserved file name.
func (s *Service) ExportGroup(
	ctx context.Context,
	key group.Key,
	encoder encoding.FileEncoder,
) (GroupBundle, error) {
	root, children, err := s.retrieveGroup(ctx, nil, key)
	if err != nil {
		return GroupBundle{}, err
	}
	members, err := symbolIDs("export", root, children)
	if err != nil {
		return GroupBundle{}, err
	}
	manifestFileName := manifestBaseName + encoder.Extension()
	var (
		files = make(zip.Files, len(children)+1)
		// claimed maps each folded file name to the symbol that took it.
		claimed = make(map[string]string, len(children))
	)
	for _, child := range children {
		base := os.SanitizeFileName(child.Name)
		if base == "" {
			return GroupBundle{}, errors.Wrapf(
				validate.ErrValidation,
				"symbol %q holds no character a file name can keep; rename it and "+
					"export again",
				child.Name,
			)
		}
		fileName := base + encoder.Extension()
		folded := os.FoldFileName(fileName)
		if folded == os.FoldFileName(manifestFileName) {
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
		if files[fileName], err = encoder.Encode(ctx, env); err != nil {
			return GroupBundle{}, err
		}
	}
	manifest, err := encoder.Encode(ctx, GroupManifest{
		Version: manifestVersion,
		Type:    manifestType,
		Name:    root.Name,
	})
	if err != nil {
		return GroupBundle{}, err
	}
	files[manifestFileName] = manifest
	return GroupBundle{Files: files, Members: members}, nil
}
