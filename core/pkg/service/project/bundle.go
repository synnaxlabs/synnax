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

	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/panel"
	"github.com/synnaxlabs/x/encoding"
	"github.com/synnaxlabs/x/encoding/zip"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/filename"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
)

// Export serializes the project identified by key and its ontology descendants as a
// bundle: one envelope per member document and panel, group children as directories,
// and a manifest naming the project at the root. It also returns the ontology ID of
// every resource that shaped the artifact — documents, panels, and groups — so a caller
// can enforce access on them. The encoder decides both the serialization and the
// extension every file takes.
//
// A child that is not a panel, a group, or a type the leaf registry exports is skipped,
// along with every panel tab that references it. The tasks a panel's tabs reference
// export into the panel's directory. A member with multiple parents in the project
// keeps its first placement and is skipped on later encounters, and a group with no
// exported descendants is dropped. Two members of one directory that resolve to one
// file name, or a member claiming a reserved root file name, keep distinct names
// through a numeric suffix. It returns query.ErrNotFound if no project has key.
func (s *Service) Export(
	ctx context.Context,
	key Key,
	encoder encoding.FileEncoder,
) (files zip.Files, members []ontology.ID, err error) {
	tx := s.cfg.DB.OpenTx()
	defer func() { err = errors.Combine(err, tx.Close()) }()
	var proj Project
	if err = s.NewRetrieve().
		Where(MatchKeys(key)).
		Entry(&proj).
		Exec(ctx, tx); err != nil {
		return nil, nil, err
	}
	w := &bundleWalk{
		svc:     s,
		tx:      tx,
		ext:     encoder.Extension(),
		visited: set.New[ontology.ID](),
		refs:    map[ontology.ID]string{},
	}
	manifestFileName := imex.ManifestBaseName + w.ext
	claims := imex.NewClaims(manifestFileName)
	if err = w.directory(ctx, OntologyID(key), "", claims); err != nil {
		return nil, nil, err
	}
	files = make(zip.Files, len(w.refs)+len(w.panels)+1)
	for id, path := range w.refs {
		env, err := s.cfg.ImEx.Export(ctx, id)
		if err != nil {
			return nil, nil, err
		}
		if files[path], err = encoder.Encode(ctx, env); err != nil {
			return nil, nil, err
		}
	}
	if err = w.encodePanels(ctx, files, encoder); err != nil {
		return nil, nil, err
	}
	manifest, err := encoder.Encode(ctx, imex.Manifest{
		Version: 1,
		Type:    "project",
		Name:    proj.Name,
	})
	if err != nil {
		return nil, nil, err
	}
	files[manifestFileName] = manifest
	return files, w.members(), nil
}

// bundleWalk accumulates the bundle's members while directory recurses through the
// project's ontology descendants.
type bundleWalk struct {
	svc *Service
	// tx is the transaction every read in the walk runs on, so the bundle reflects one
	// snapshot of the cluster.
	tx gorp.Tx
	// ext is the extension the encoder gives every member file.
	ext string
	// visited holds every resource the walk already placed, so a resource with multiple
	// parents in the project is placed only once.
	visited set.Set[ontology.ID]
	// refs maps each member document to its path from the bundle root. Panel encoding
	// resolves resource tabs through it.
	refs map[ontology.ID]string
	// panels pairs each member panel's document with its path from the bundle root.
	panels []bundlePanel
	// groups holds every group that contributed a directory to the artifact.
	groups []ontology.ID
}

// bundlePanel pairs a member panel's document with its path from the bundle root.
type bundlePanel struct {
	panel panel.Panel
	path  string
}

// members returns the ID of every resource that shaped the artifact, for access
// enforcement by the caller.
func (w *bundleWalk) members() []ontology.ID {
	members := make([]ontology.ID, 0, len(w.refs)+len(w.panels)+len(w.groups))
	for id := range w.refs {
		members = append(members, id)
	}
	for _, p := range w.panels {
		members = append(members, p.panel.OntologyID())
	}
	return append(members, w.groups...)
}

// directory walks the children of parent into the directory at prefix ("" for the
// bundle root, "a/b/" otherwise). claims tracks the file names taken in this directory.
func (w *bundleWalk) directory(
	ctx context.Context,
	parent ontology.ID,
	prefix string,
	claims *imex.Claims,
) error {
	var children []ontology.Resource
	if err := w.svc.cfg.Ontology.NewRetrieve().
		WhereIDs(parent).
		TraverseTo(ontology.ChildrenTraverser).
		Entries(&children).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	imex.SortResources(children)
	for _, child := range children {
		if w.visited.Contains(child.ID) {
			continue
		}
		switch {
		case child.ID.Type == ontology.ResourceTypeGroup:
			dirName, err := filename.Sanitize(child.Name, "")
			if err != nil {
				return err
			}
			dirName = claims.Claim(dirName, "")
			w.visited.Add(child.ID)
			before := len(w.refs) + len(w.panels)
			err = w.directory(ctx, child.ID, prefix+dirName+"/", imex.NewClaims())
			if err != nil {
				return err
			}
			// An empty group is dropped: it enforces no access, and the name it claimed
			// only pushes a same-named sibling group to the next suffix.
			if len(w.refs)+len(w.panels) == before {
				continue
			}
			w.groups = append(w.groups, child.ID)
		case child.ID.Type == ontology.ResourceTypePanel:
			fileName, err := filename.Sanitize(child.Name, w.ext)
			if err != nil {
				return err
			}
			fileName = claims.Claim(fileName, w.ext)
			w.visited.Add(child.ID)
			p, err := w.retrievePanel(ctx, child.ID)
			if err != nil {
				return err
			}
			w.panels = append(w.panels, bundlePanel{
				panel: p,
				path:  prefix + fileName,
			})
			if err = w.panelTasks(ctx, p, prefix, claims); err != nil {
				return err
			}
		case w.svc.cfg.ImEx.ExporterRegistered(child.ID.Type):
			fileName, err := filename.Sanitize(child.Name, w.ext)
			if err != nil {
				return err
			}
			fileName = claims.Claim(fileName, w.ext)
			w.visited.Add(child.ID)
			w.refs[child.ID] = prefix + fileName
		}
	}
	return nil
}

// retrievePanel reads the panel document behind id on the walk's transaction.
func (w *bundleWalk) retrievePanel(
	ctx context.Context,
	id ontology.ID,
) (panel.Panel, error) {
	keys, err := panel.KeysFromOntologyIDs([]ontology.ID{id})
	if err != nil {
		return panel.Panel{}, err
	}
	var p panel.Panel
	err = w.svc.cfg.Panel.NewRetrieve().
		Where(panel.MatchKeys(keys...)).
		Entry(&p).
		Exec(ctx, w.tx)
	return p, err
}

// panelTasks places the tasks p's resource tabs reference into the panel's directory.
// Tasks are not ontology descendants of the project, so the walk pulls them in from the
// panel document itself. A task that no longer exists is skipped, along with the tab
// that references it.
func (w *bundleWalk) panelTasks(
	ctx context.Context,
	p panel.Panel,
	prefix string,
	claims *imex.Claims,
) error {
	if !w.svc.cfg.ImEx.ExporterRegistered(ontology.ResourceTypeTask) {
		return nil
	}
	var tasks []ontology.Resource
	for _, id := range panel.TaskRefs(p.Root) {
		if w.visited.Contains(id) {
			continue
		}
		var res ontology.Resource
		if err := w.svc.cfg.Ontology.NewRetrieve().
			WhereIDs(id).
			Entry(&res).
			Exec(ctx, w.tx); err != nil {
			if errors.Is(err, query.ErrNotFound) {
				continue
			}
			return err
		}
		tasks = append(tasks, res)
	}
	imex.SortResources(tasks)
	for _, t := range tasks {
		fileName, err := filename.Sanitize(t.Name, w.ext)
		if err != nil {
			return err
		}
		w.visited.Add(t.ID)
		w.refs[t.ID] = prefix + claims.Claim(fileName, w.ext)
	}
	return nil
}

func (w *bundleWalk) encodePanels(
	ctx context.Context,
	files zip.Files,
	encoder encoding.FileEncoder,
) error {
	for _, bp := range w.panels {
		env, err := panel.EncodeBundle(bp.panel, w.refs)
		if err != nil {
			return err
		}
		if files[bp.path], err = encoder.Encode(ctx, env); err != nil {
			return err
		}
	}
	return nil
}
