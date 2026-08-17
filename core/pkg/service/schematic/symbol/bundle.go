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
	"slices"
	"strings"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/encoding"
	"github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/encoding/zip"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/filename"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/validate"
)

const (
	manifestVersion       = 2
	legacyManifestVersion = 1
	manifestType          = "symbol_group"
	// manifestBaseName is the manifest's file name without an extension. The codec
	// supplies the extension, so every serialization has one recognition point.
	manifestBaseName = "manifest"
	// layoutFileName is reserved for legacy project bundles and is never a member.
	layoutFileName = "LAYOUT.json"
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

// legacyGroupManifest is the version 1 manifest body the Console wrote before group
// export moved server-side. Its symbols list declares membership; version 2 infers
// membership from the files beside the manifest instead.
type legacyGroupManifest struct {
	Symbols []struct {
		File string `json:"file"`
	} `json:"symbols"`
}

// ImportGroup imports a symbol group bundle on tx: it creates a fresh group named by
// the manifest under the permanent symbol group, then imports every member into it
// through the leaf importer. Serialization is keyed by file extension, JSON being the
// only one supported: the manifest is manifest.json, and members are the JSON files
// beside it. A version 1 manifest declares members in its symbols list, version 2
// infers them from the files beside the manifest. It returns an error naming the
// offending file when the manifest is missing, malformed, of another bundle kind, or
// of an unsupported version, when two member names compare equal case-folded, or when
// a member is not a schematic symbol.
func (s *Service) ImportGroup(
	ctx context.Context,
	tx gorp.Tx,
	files zip.Files,
) (group.Group, error) {
	manifestFileName := manifestBaseName + json.Codec.Extension()
	manifestData, ok := files[manifestFileName]
	if !ok {
		return group.Group{}, errors.Wrapf(
			validate.ErrValidation, "bundle holds no %s", manifestFileName,
		)
	}
	var manifest GroupManifest
	if err := json.Codec.Decode(ctx, manifestData, &manifest); err != nil {
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
		members, err = declaredMembers(ctx, manifestData, files, manifestFileName)
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
	g, err := s.cfg.Group.NewWriter(tx).Create(ctx, manifest.Name, s.group.OntologyID())
	if err != nil {
		return group.Group{}, err
	}
	for _, name := range members {
		if err = s.importMember(
			ctx,
			tx,
			name,
			files[name],
			g.OntologyID(),
		); err != nil {
			return group.Group{}, err
		}
	}
	return g, nil
}

// declaredMembers reads a version 1 manifest's membership from its symbols list,
// validating that every listed file is in the bundle and none names the manifest.
func declaredMembers(
	ctx context.Context,
	manifestData []byte,
	files zip.Files,
	manifestFileName string,
) ([]string, error) {
	var legacy legacyGroupManifest
	if err := json.Codec.Decode(ctx, manifestData, &legacy); err != nil {
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
// in the given extension beside the manifest, minus the reserved names.
func inferredMembers(files zip.Files, manifestFileName, ext string) []string {
	var (
		foldedManifest = filename.Fold(manifestFileName)
		foldedLayout   = filename.Fold(layoutFileName)
		foldedExt      = filename.Fold(ext)
		members        = make([]string, 0, len(files))
	)
	for name := range files {
		folded := filename.Fold(name)
		if folded == foldedManifest || folded == foldedLayout ||
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

// importMember decodes one member file and imports it under parent through the leaf
// registry. Every error names the member's file.
func (s *Service) importMember(
	ctx context.Context,
	tx gorp.Tx,
	name string,
	data []byte,
	parent ontology.ID,
) error {
	var env imex.Envelope
	if err := json.Codec.Decode(ctx, data, &env); err != nil {
		return errors.Wrap(err, name)
	}
	typ, err := s.cfg.ImEx.ResolveType(env)
	if err != nil {
		return errors.Wrap(err, name)
	}
	if typ != string(ontology.ResourceTypeSchematicSymbol) {
		return errors.Wrapf(
			validate.ErrValidation,
			"member %q is a %q, not a schematic symbol", name, typ,
		)
	}
	env.Type = typ
	if _, err = s.cfg.ImEx.Import(ctx, tx, env, imex.ImportOptions{
		FileName: name,
		Parent:   parent,
	}); err != nil {
		return errors.Wrap(err, name)
	}
	return nil
}
