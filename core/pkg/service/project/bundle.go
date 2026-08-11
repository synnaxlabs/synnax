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

	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/panel"
	"github.com/synnaxlabs/x/encoding"
	"github.com/synnaxlabs/x/encoding/zip"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/os"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

const (
	manifestVersion = 1
	manifestType    = "project"
	// manifestBaseName is the manifest's file name without an extension. The codec
	// supplies the extension, so every serialization has one recognition point.
	manifestBaseName = "manifest"
	// legacyLayoutFileName is reserved so a stable-release project directory migrated
	// in place keeps working.
	legacyLayoutFileName = "LAYOUT.json"
)

// Manifest is the body of manifest.json in a project bundle. Membership is inferred
// from the files around the manifest, so it names no members.
type Manifest struct {
	// Version governs the manifest schema and the bundle's layout rules.
	Version uint64 `json:"version"`
	// Type is the bundle kind, letting an endpoint reject a bundle of another kind.
	Type string `json:"type"`
	// Name is the project's name, which an import gives the project it creates.
	Name string `json:"name"`
}

// Export serializes the project identified by key and its ontology descendants as a
// bundle: one envelope per member document and panel, group children as directories,
// and a manifest naming the project at the root. It also returns the ontology ID of
// every resource that shaped the artifact — documents, panels, and groups — so a
// caller can enforce access on them. The encoder decides both the serialization and
// the extension every file takes.
//
// A child that is not a panel, a group, or a type the leaf registry exports is
// skipped, along with every panel tab that references it. A group with no exported
// descendants is dropped. It returns query.ErrNotFound if no project has key, and a
// validation error if two members in one directory resolve to the same file name or
// a member claims a reserved root file name.
func (s *Service) Export(
	ctx context.Context,
	key Key,
	encoder encoding.FileEncoder,
) (zip.Files, []ontology.ID, error) {
	var proj Project
	if err := s.NewRetrieve().
		Where(MatchKeys(key)).
		Entry(&proj).
		Exec(ctx, nil); err != nil {
		return nil, nil, err
	}
	w := &bundleWalk{
		ctx:  ctx,
		svc:  s,
		ext:  encoder.Extension(),
		refs: map[ontology.ID]string{},
	}
	manifestFileName := manifestBaseName + w.ext
	reserved := map[string]string{
		os.FoldFileName(manifestFileName):     manifestFileName,
		os.FoldFileName(legacyLayoutFileName): legacyLayoutFileName,
	}
	if err := w.directory(OntologyID(key), "", reserved); err != nil {
		return nil, nil, err
	}
	files := make(zip.Files, len(w.refs)+len(w.panelIDs)+1)
	for id, path := range w.refs {
		env, err := s.cfg.ImEx.Export(ctx, id)
		if err != nil {
			return nil, nil, err
		}
		if files[path], err = encoder.Encode(ctx, env); err != nil {
			return nil, nil, err
		}
	}
	if err := w.encodePanels(files, encoder); err != nil {
		return nil, nil, err
	}
	manifest, err := encoder.Encode(ctx, Manifest{
		Version: manifestVersion,
		Type:    manifestType,
		Name:    proj.Name,
	})
	if err != nil {
		return nil, nil, err
	}
	files[manifestFileName] = manifest
	return files, w.members, nil
}

// bundleWalk accumulates the bundle's members while directory recurses through the
// project's ontology descendants.
type bundleWalk struct {
	ctx context.Context
	svc *Service
	// ext is the extension the encoder gives every member file.
	ext string
	// refs maps each member document to its path from the bundle root. Panel encoding
	// resolves resource tabs through it.
	refs map[ontology.ID]string
	// panelIDs and panelPaths pair each member panel with its path from the bundle
	// root.
	panelIDs   []ontology.ID
	panelPaths []string
	// members holds the ID of every resource that shaped the artifact, for access
	// enforcement by the caller.
	members []ontology.ID
}

// directory walks the children of parent into the directory at prefix ("" for the
// bundle root, "a/b/" otherwise). reserved maps folded file names no member in this
// directory may claim to their display form.
func (w *bundleWalk) directory(
	parent ontology.ID,
	prefix string,
	reserved map[string]string,
) error {
	var children []ontology.Resource
	if err := w.svc.cfg.Ontology.NewRetrieve().
		WhereIDs(parent).
		TraverseTo(ontology.ChildrenTraverser).
		Entries(&children).
		Exec(w.ctx, nil); err != nil {
		return err
	}
	// claimed maps each folded file name to the resource that took it.
	claimed := make(map[string]string, len(children))
	claim := func(resourceName, fileName string) error {
		folded := os.FoldFileName(fileName)
		if display, ok := reserved[folded]; ok {
			return errors.Wrapf(
				validate.ErrValidation,
				"%q takes the reserved file name %q; rename it and export again",
				resourceName, display,
			)
		}
		if prev, ok := claimed[folded]; ok {
			return errors.Wrapf(
				validate.ErrValidation,
				"%q and %q both export to %q; rename one and export again",
				prev, resourceName, prefix+fileName,
			)
		}
		claimed[folded] = resourceName
		return nil
	}
	for _, child := range children {
		switch {
		case child.ID.Type == ontology.ResourceTypeGroup:
			dirName, err := os.SanitizeFileName(child.Name, "")
			if err != nil {
				return err
			}
			before := len(w.refs) + len(w.panelIDs)
			if err = w.directory(child.ID, prefix+dirName+"/", nil); err != nil {
				return err
			}
			// An empty group is dropped: it claims no name and enforces no access.
			if len(w.refs)+len(w.panelIDs) == before {
				continue
			}
			if err = claim(child.Name, dirName); err != nil {
				return err
			}
			w.members = append(w.members, child.ID)
		case child.ID.Type == ontology.ResourceTypePanel:
			fileName, err := os.SanitizeFileName(child.Name, w.ext)
			if err != nil {
				return err
			}
			if err = claim(child.Name, fileName); err != nil {
				return err
			}
			w.panelIDs = append(w.panelIDs, child.ID)
			w.panelPaths = append(w.panelPaths, prefix+fileName)
			w.members = append(w.members, child.ID)
		case w.svc.cfg.ImEx.ExporterRegistered(child.ID.Type):
			fileName, err := os.SanitizeFileName(child.Name, w.ext)
			if err != nil {
				return err
			}
			if err = claim(child.Name, fileName); err != nil {
				return err
			}
			w.refs[child.ID] = prefix + fileName
			w.members = append(w.members, child.ID)
		}
	}
	return nil
}

// encodePanels retrieves every member panel and writes its bundle envelope into
// files, resolving resource tabs through the walk's refs table.
func (w *bundleWalk) encodePanels(files zip.Files, encoder encoding.FileEncoder) error {
	if len(w.panelIDs) == 0 {
		return nil
	}
	keys, err := panel.KeysFromOntologyIDs(w.panelIDs)
	if err != nil {
		return err
	}
	var panels []panel.Panel
	if err = w.svc.cfg.Panel.NewRetrieve().
		Where(panel.MatchKeys(keys...)).
		Entries(&panels).
		Exec(w.ctx, nil); err != nil {
		return err
	}
	byKey := make(map[panel.Key]panel.Panel, len(panels))
	for _, p := range panels {
		byKey[p.Key] = p
	}
	for i, key := range keys {
		p, ok := byKey[key]
		if !ok {
			return errors.Wrapf(query.ErrNotFound, "panel %s", key)
		}
		env, err := panel.EncodeBundle(p, w.refs)
		if err != nil {
			return err
		}
		if files[w.panelPaths[i]], err = encoder.Encode(w.ctx, env); err != nil {
			return err
		}
	}
	return nil
}
