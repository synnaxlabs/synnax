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

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/encoding"
	"github.com/synnaxlabs/x/encoding/zip"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/filename"
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
	Version uint8 `json:"version"`
	// Type is the bundle kind, letting an endpoint reject a bundle of another kind.
	Type string `json:"type"`
	// Name is the group's name, which an import gives the group it creates.
	Name string `json:"name"`
}

// ExportGroup serializes every symbol in the group identified by key into a flat file
// namespace: one envelope named after each symbol, beside a manifest naming the group.
// It also returns the ontology ID of every exported symbol, in the order the ontology
// returned the group's children, so a caller can enforce access on them. The encoder
// decides both the serialization and the extension every file takes. A name too long
// for a file name is shortened, which can make two symbols collide.
//
// Children that are not schematic symbols are skipped and logged as a warning.
//
// It returns query.ErrNotFound if no group has key. It returns a validation error if
// two symbols resolve to the same file name, or if a symbol claims a reserved file
// name.
func (s *Service) ExportGroup(
	ctx context.Context,
	key group.Key,
	encoder encoding.FileEncoder,
) (zip.Files, []ontology.ID, error) {
	root, children, err := s.retrieveGroup(ctx, nil, key)
	if err != nil {
		return nil, nil, err
	}
	symbols, skipped := partitionSymbols(children)
	s.warnSkipped(root.Name, skipped)
	members := lo.Map(symbols, func(r ontology.Resource, _ int) ontology.ID {
		return r.ID
	})
	ext := encoder.Extension()
	manifestFileName := manifestBaseName + ext
	foldedManifestFileName := filename.Fold(manifestFileName)
	var (
		files = make(zip.Files, len(symbols)+1)
		// claimed maps each folded file name to the symbol that took it.
		claimed = make(map[string]string, len(symbols))
	)
	for _, child := range symbols {
		fileName, err := filename.Sanitize(child.Name, ext)
		if err != nil {
			return nil, nil, err
		}
		folded := filename.Fold(fileName)
		if folded == foldedManifestFileName {
			return nil, nil, errors.Wrapf(
				validate.ErrValidation,
				"symbol %q takes the reserved file name %q; rename it and export again",
				child.Name, fileName,
			)
		}
		if prev, ok := claimed[folded]; ok {
			return nil, nil, errors.Wrapf(
				validate.ErrValidation,
				"symbols %q and %q both export to %q; rename one and export again",
				prev, child.Name, fileName,
			)
		}
		claimed[folded] = child.Name
		env, err := s.Export(ctx, child.ID)
		if err != nil {
			return nil, nil, err
		}
		if files[fileName], err = encoder.Encode(ctx, env); err != nil {
			return nil, nil, err
		}
	}
	manifest, err := encoder.Encode(ctx, GroupManifest{
		Version: manifestVersion,
		Type:    manifestType,
		Name:    root.Name,
	})
	if err != nil {
		return nil, nil, err
	}
	files[manifestFileName] = manifest
	return files, members, nil
}
