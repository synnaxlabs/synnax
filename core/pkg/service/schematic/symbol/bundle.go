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
	"uuid"

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
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

const (
	manifestVersion        = 3
	keylessManifestVersion = 2
	legacyManifestVersion  = 1
	manifestType           = "symbol_group"
)

// groupManifest is the manifest of a group bundle. Version 3 adds the group key.
type groupManifest struct {
	imex.Manifest
	Key group.Key `json:"key,omitzero"`
}

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
	manifest, err := encoder.Encode(groupManifest{
		Manifest: imex.Manifest{Version: manifestVersion, Type: manifestType, Name: g.Name},
		Key:      g.Key,
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

// ImportGroup imports every member of a bundle on tx into the top-level symbol group
// with the manifest's key, or into a new group named by the manifest when the Core has
// none. A member reuses a stored symbol as Decode does, with retrievable. Bundles are
// JSON: the manifest is manifest.json and members are the JSON files beside it. A
// version 1 manifest lists its members; later versions infer them from the files
// beside the manifest. It returns a validation error naming the offending file for a
// missing or invalid manifest, member names that collide case-folded, a member that is
// not a schematic symbol, or a manifest key of a group that is not a top-level symbol
// group.
func (s *Service) ImportGroup(
	ctx context.Context,
	tx gorp.Tx,
	files zip.Files,
	retrievable func(resource.ID) bool,
) (group.Group, error) {
	manifestFileName := imex.ManifestBaseName + json.Codec.Extension()
	manifest, manifestData, err := decodeGroupManifest(files, manifestFileName)
	if err != nil {
		return group.Group{}, err
	}
	members, err := groupMembers(manifest.Version, manifestData, files, manifestFileName)
	if err != nil {
		return group.Group{}, err
	}
	g, err := s.bundleGroup(ctx, tx, manifest)
	if err != nil {
		return group.Group{}, err
	}
	for _, name := range members {
		if err = s.importMember(ctx, tx, name, files[name], imex.ImportOptions{
			FileName:    name,
			Group:       &g.Key,
			Retrievable: retrievable,
		}); err != nil {
			return group.Group{}, err
		}
	}
	return g, nil
}

// bundleGroup returns the top-level symbol group with the key of manifest, or creates a
// group named by manifest under that key when the Core has no group with it. A
// manifest with no key always gets a new group. It returns a validation error when the
// key names a group that is not a top-level symbol group.
func (s *Service) bundleGroup(
	ctx context.Context,
	tx gorp.Tx,
	manifest groupManifest,
) (group.Group, error) {
	g := group.Group{
		Key:   manifest.Key,
		Name:  manifest.Name,
		Scope: resource.TypeSchematicSymbol,
	}
	if manifest.Key == uuid.Nil() {
		return g, s.cfg.Group.NewWriter(tx).Create(ctx, &g)
	}
	var stored group.Group
	err := s.cfg.Group.NewRetrieve().
		Where(group.MatchKeys(manifest.Key)).
		Entry(&stored).
		Exec(ctx, tx)
	if errors.Is(err, query.ErrNotFound) {
		return g, s.cfg.Group.NewWriter(tx).Create(ctx, &g)
	}
	if err != nil {
		return group.Group{}, err
	}
	if stored.Scope != resource.TypeSchematicSymbol || stored.Parent != nil {
		return group.Group{}, errors.Wrapf(
			validate.ErrValidation,
			"group %q is not a top-level schematic symbol group", stored.Name,
		)
	}
	return stored, nil
}

// decodeGroupManifest reads and validates the bundle's manifest, returning it with its
// raw bytes.
func decodeGroupManifest(
	files zip.Files,
	manifestFileName string,
) (groupManifest, []byte, error) {
	manifestData, ok := files[manifestFileName]
	if !ok {
		return groupManifest{}, nil, errors.Wrapf(
			validate.ErrValidation, "bundle holds no %s", manifestFileName,
		)
	}
	var manifest groupManifest
	if err := imex.Codec.Decode(manifestData, &manifest); err != nil {
		return groupManifest{}, nil, errors.Wrap(err, manifestFileName)
	}
	if manifest.Type != manifestType {
		return groupManifest{}, nil, errors.Wrapf(
			validate.ErrValidation,
			"bundle is a %q, not a %s", manifest.Type, manifestType,
		)
	}
	if manifest.Version > manifestVersion {
		return groupManifest{}, nil, imex.NewErrUnsupportedVersion(
			manifestType, imex.Version(manifest.Version), manifestVersion,
		)
	}
	if manifest.Name == "" {
		return groupManifest{}, nil, errors.Wrapf(
			validate.ErrValidation, "%s names no group", manifestFileName,
		)
	}
	return manifest, manifestData, nil
}

// groupMembers resolves the member file names for the manifest's version and rejects
// names that collide case-folded.
func groupMembers(
	version uint8,
	manifestData []byte,
	files zip.Files,
	manifestFileName string,
) ([]string, error) {
	var (
		members []string
		err     error
	)
	switch version {
	case legacyManifestVersion:
		members, err = declaredMembers(manifestData, files, manifestFileName)
	case keylessManifestVersion, manifestVersion:
		members = inferredMembers(files, manifestFileName, json.Codec.Extension())
	default:
		err = errors.Wrapf(
			validate.ErrValidation, "unsupported manifest version %d", version,
		)
	}
	if err != nil {
		return nil, err
	}
	if err = validateMemberNames(members); err != nil {
		return nil, err
	}
	return members, nil
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

// importMember decodes one member file and imports it with opts through the leaf
// registry. Every error names the member's file.
func (s *Service) importMember(
	ctx context.Context,
	tx gorp.Tx,
	name string,
	data []byte,
	opts imex.ImportOptions,
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
	if _, err = s.cfg.ImEx.Import(ctx, tx, env, opts); err != nil {
		return errors.Wrap(err, name)
	}
	return nil
}
