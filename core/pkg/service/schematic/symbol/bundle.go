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
	"cmp"
	"context"
	"slices"
	"strings"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/encoding"
	"github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/encoding/zip"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/filename"
	"github.com/synnaxlabs/x/validate"
)

const (
	manifestVersion       = 2
	legacyManifestVersion = 1
	manifestType          = "symbol_group"
)

// ExportGroup serializes every symbol in the group identified by key, as snap holds it,
// into a flat file namespace: one envelope per symbol, encoded by encoder, beside a
// manifest naming the group. File names are sanitized and deduplicated. It also returns
// the sorted resource IDs of the exported symbols so a caller can enforce access on
// them. It returns query.ErrNotFound if no group has key.
func (s *Service) ExportGroup(
	ctx context.Context,
	snap gorp.Snapshot,
	key group.Key,
	encoder encoding.FileEncoder,
) (zip.Files, []resource.ID, error) {
	var (
		g       group.Group
		symbols []Symbol
	)
	if err := s.cfg.Group.NewRetrieve().
		Where(group.MatchKeys(key)).
		Entry(&g).
		Exec(ctx, snap); err != nil {
		return nil, nil, err
	}
	if err := s.NewRetrieve().
		Where(MatchGroups(key)).
		Entries(&symbols).
		Exec(ctx, snap); err != nil {
		return nil, nil, err
	}
	slices.SortFunc(symbols, func(a, b Symbol) int {
		return cmp.Or(
			cmp.Compare(a.Name, b.Name),
			cmp.Compare(a.Key.String(), b.Key.String()),
		)
	})
	members := lo.Map(symbols, func(sym Symbol, _ int) resource.ID {
		return sym.ResourceID()
	})
	ext := encoder.Extension()
	manifestFileName := imex.ManifestBaseName + ext
	var (
		files  = make(zip.Files, len(symbols)+1)
		claims = imex.NewClaims(manifestFileName)
	)
	for _, sym := range symbols {
		fileName, err := filename.Sanitize(sym.Name, ext)
		if err != nil {
			return nil, nil, err
		}
		fileName = claims.Claim(fileName, ext)
		env, err := s.Export(ctx, snap, sym.ResourceID())
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
		Name:    g.Name,
	})
	if err != nil {
		return nil, nil, err
	}
	files[manifestFileName] = manifest
	return files, members, nil
}

// legacyGroupManifest is the version 1 manifest body the Console wrote; its symbols
// list declares membership.
type legacyGroupManifest struct {
	Symbols []struct {
		File string `json:"file"`
	} `json:"symbols"`
}

// ImportGroup creates a top-level symbol group named by the manifest, then imports
// every member into it on tx. Bundles are JSON: the manifest is
// manifest.json and members are the JSON files beside it. A version 1 manifest lists
// its members; version 2 infers them from the files beside the manifest. It returns a
// validation error naming the offending file for a missing or invalid manifest, member
// names that collide case-folded, or a member that is not a schematic symbol.
func (s *Service) ImportGroup(
	ctx context.Context,
	tx gorp.Tx,
	files zip.Files,
) (group.Group, error) {
	manifestFileName := imex.ManifestBaseName + json.Codec.Extension()
	manifestData, ok := files[manifestFileName]
	if !ok {
		return group.Group{}, errors.Wrapf(
			validate.ErrValidation, "bundle holds no %s", manifestFileName,
		)
	}
	var manifest imex.Manifest
	if err := imex.Codec.Decode(manifestData, &manifest); err != nil {
		return group.Group{}, errors.Wrap(err, manifestFileName)
	}
	if manifest.Type != manifestType {
		return group.Group{}, errors.Wrapf(
			validate.ErrValidation,
			"bundle is a %q, not a %s", manifest.Type, manifestType,
		)
	}
	if manifest.Version > manifestVersion {
		return group.Group{}, imex.NewErrUnsupportedVersion(
			manifestType, imex.Version(manifest.Version), manifestVersion,
		)
	}
	if manifest.Name == "" {
		return group.Group{}, errors.Wrapf(
			validate.ErrValidation, "%s names no group", manifestFileName,
		)
	}
	var (
		members []string
		err     error
	)
	switch manifest.Version {
	case legacyManifestVersion:
		members, err = declaredMembers(manifestData, files, manifestFileName)
	case manifestVersion:
		members = inferredMembers(files, manifestFileName, json.Codec.Extension())
	default:
		err = errors.Wrapf(
			validate.ErrValidation, "unsupported manifest version %d", manifest.Version,
		)
	}
	if err != nil {
		return group.Group{}, err
	}
	if err = validateMemberNames(members); err != nil {
		return group.Group{}, err
	}
	g := group.Group{Name: manifest.Name, Scope: resource.TypeSchematicSymbol}
	if err = s.cfg.Group.NewWriter(tx).Create(ctx, &g); err != nil {
		return group.Group{}, err
	}
	for _, name := range members {
		if err = s.importMember(
			ctx,
			tx,
			name,
			files[name],
			g.Key,
		); err != nil {
			return group.Group{}, err
		}
	}
	return g, nil
}

// declaredMembers reads a version 1 manifest's membership from its symbols list,
// rejecting a listed file the bundle does not hold and a manifest that lists itself.
func declaredMembers(
	manifestData []byte,
	files zip.Files,
	manifestFileName string,
) ([]string, error) {
	var legacy legacyGroupManifest
	if err := imex.Codec.Decode(manifestData, &legacy); err != nil {
		return nil, errors.Wrap(err, manifestFileName)
	}
	foldedManifest := filename.Fold(manifestFileName)
	members := make([]string, 0, len(legacy.Symbols))
	for _, sym := range legacy.Symbols {
		if filename.Fold(sym.File) == foldedManifest {
			return nil, errors.Wrapf(
				validate.ErrValidation,
				"manifest lists itself, %q, as a member",
				sym.File,
			)
		}
		if _, ok := files[sym.File]; !ok {
			return nil, errors.Wrapf(
				validate.ErrValidation,
				"manifest lists %q, which the bundle does not hold", sym.File,
			)
		}
		members = append(members, sym.File)
	}
	return members, nil
}

// inferredMembers reads a version 2 bundle's membership from the directory: every file
// in the given extension beside the manifest. Files under a subdirectory are skipped.
func inferredMembers(files zip.Files, manifestFileName, ext string) []string {
	var (
		foldedManifest = filename.Fold(manifestFileName)
		foldedExt      = filename.Fold(ext)
		members        = make([]string, 0, len(files))
	)
	for name := range files {
		folded := filename.Fold(name)
		if folded == foldedManifest || strings.ContainsRune(name, '/') ||
			!strings.HasSuffix(folded, foldedExt) {
			continue
		}
		members = append(members, name)
	}
	slices.Sort(members)
	return members
}

// validateMemberNames rejects two member names that compare equal case-folded: the
// Console extracts bundles onto case-insensitive filesystems, where the pair collides.
func validateMemberNames(members []string) error {
	claimed := make(map[string]string, len(members))
	for _, name := range members {
		folded := filename.Fold(name)
		if prev, ok := claimed[folded]; ok {
			return errors.Wrapf(
				validate.ErrValidation,
				"members %q and %q have the same file name; rename one and import again",
				prev,
				name,
			)
		}
		claimed[folded] = name
	}
	return nil
}

// importMember decodes one member file and imports it into group g through the leaf
// registry. Every error names the member's file.
func (s *Service) importMember(
	ctx context.Context,
	tx gorp.Tx,
	name string,
	data []byte,
	g group.Key,
) error {
	var env imex.Envelope
	if err := imex.Codec.Decode(data, &env); err != nil {
		return errors.Wrap(err, name)
	}
	typ, err := s.cfg.ImEx.ResolveType(env)
	if err != nil {
		return errors.Wrap(err, name)
	}
	if typ != string(resource.TypeSchematicSymbol) {
		return errors.Wrapf(
			validate.ErrValidation,
			"member %q is a %q, not a schematic symbol", name, typ,
		)
	}
	env.Type = typ
	if _, err = s.cfg.ImEx.Import(ctx, tx, env, imex.ImportOptions{
		FileName: name,
		Group:    &g,
	}); err != nil {
		return errors.Wrap(err, name)
	}
	return nil
}
