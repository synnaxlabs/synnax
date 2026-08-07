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
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/encoding"
	"github.com/synnaxlabs/x/encoding/zip"
	"github.com/synnaxlabs/x/filename"
)

const (
	manifestVersion = 2
	manifestType    = "symbol_group"
)

// ExportGroup serializes every symbol in the group identified by key into a flat file
// namespace: one envelope named after each symbol, beside a manifest naming the group.
// It also returns the ontology ID of every exported symbol, sorted by name and then by
// ID, so a caller can enforce access on them. The encoder decides both the
// serialization and the extension every file takes. A symbol whose file name is too
// long is shortened; one whose file name is taken or reserved gains a numeric suffix.
//
// Children that are not schematic symbols are skipped and logged as a warning.
//
// It returns query.ErrNotFound if no group has key.
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
	imex.SortResources(symbols)
	members := lo.Map(symbols, func(r ontology.Resource, _ int) ontology.ID {
		return r.ID
	})
	ext := encoder.Extension()
	manifestFileName := imex.ManifestBaseName + ext
	var (
		files  = make(zip.Files, len(symbols)+1)
		claims = imex.NewClaims(manifestFileName)
	)
	for _, child := range symbols {
		fileName, err := filename.Sanitize(child.Name, ext)
		if err != nil {
			return nil, nil, err
		}
		fileName = claims.Claim(fileName, ext)
		env, err := s.Export(ctx, child.ID)
		if err != nil {
			return nil, nil, err
		}
		if files[fileName], err = encoder.Encode(env); err != nil {
			return nil, nil, err
		}
	}
	manifest, err := encoder.Encode(imex.Manifest{
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
