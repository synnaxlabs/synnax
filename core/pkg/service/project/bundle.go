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
	"github.com/synnaxlabs/synnax/pkg/service/panel"
	"github.com/synnaxlabs/x/encoding"
	"github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/encoding/zip"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/filename"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

const (
	manifestVersion = 1
	manifestType    = "project"
	// legacyLayoutFileName is the tiling file every stable release wrote at the root of
	// a project export. It marks the legacy (version 0) format only when no manifest is
	// present.
	legacyLayoutFileName = "LAYOUT.json"
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
		Version: manifestVersion,
		Type:    manifestType,
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

// Import creates a fresh project from the bundle in files on tx. The manifest names
// the project, with the extension-stripped fileName as the fallback; each bundle
// directory becomes a group; each non-panel member imports through the leaf registry;
// each panel is recreated with its resource tabs resolved to the members' minted IDs.
// A directory with no manifest but a root LAYOUT.json is the legacy (version 0)
// format: its documents are recreated and its tiling dropped. Invalid bundles return
// validation errors naming the offending file.
func (s *Service) Import(
	ctx context.Context,
	tx gorp.Tx,
	files zip.Files,
	fileName string,
) (Project, error) {
	manifestFileName := imex.ManifestBaseName + json.Codec.Extension()
	manifestData, ok := files[manifestFileName]
	if !ok {
		if layoutData, ok := files[legacyLayoutFileName]; ok {
			return s.importLegacy(ctx, tx, layoutData, files, fileName)
		}
		return Project{}, errors.Wrapf(
			validate.ErrValidation, "bundle holds no %s", manifestFileName,
		)
	}
	manifest, err := decodeManifest(ctx, manifestData, manifestFileName)
	if err != nil {
		return Project{}, err
	}
	name := manifest.Name
	if name == "" {
		name = imex.BaseName(fileName)
	}
	members, err := s.bundleMembers(ctx, files)
	if err != nil {
		return Project{}, err
	}
	proj := Project{Name: name}
	if err = s.NewWriter(tx).Create(ctx, &proj); err != nil {
		return Project{}, err
	}
	projID := OntologyID(proj.Key)
	groups, err := s.createGroups(ctx, tx, projID, members)
	if err != nil {
		return Project{}, err
	}
	var (
		refs      = make(map[string]ontology.ID, len(members))
		otgWriter = s.cfg.Ontology.NewWriter(tx)
	)
	for _, m := range members {
		if m.env.Type == string(ontology.ResourceTypePanel) {
			continue
		}
		id, err := s.cfg.ImEx.Import(ctx, tx, m.env, imex.ImportOptions{
			FileName: m.path,
			Parent:   projID,
		})
		if err != nil {
			return Project{}, errors.Wrap(err, m.path)
		}
		refs[m.path] = id
		dir := parentDir(m.path)
		// A task parents under its rack's task group, never the project: it entered
		// the bundle through a panel reference, so it stays where its importer put it.
		if dir == "" || id.Type == ontology.ResourceTypeTask {
			continue
		}
		if err = otgWriter.DeleteRelationships(ctx, ontology.Relationship{
			From: projID,
			Type: ontology.RelationshipTypeParentOf,
			To:   id,
		}); err != nil {
			return Project{}, err
		}
		if err = otgWriter.DefineRelationships(
			ctx, groups[dir], ontology.RelationshipTypeParentOf, id,
		); err != nil {
			return Project{}, err
		}
	}
	panelWriter := s.cfg.Panel.NewWriter(tx)
	for _, m := range members {
		if m.env.Type != string(ontology.ResourceTypePanel) {
			continue
		}
		p, err := panel.DecodeBundle(ctx, m.env, refs)
		if err != nil {
			return Project{}, errors.Wrap(err, m.path)
		}
		if p.Name == "" {
			p.Name = imex.BaseName(m.path)
		}
		parent := projID
		if dir := parentDir(m.path); dir != "" {
			parent = groups[dir]
		}
		p.Parent = &parent
		if err = panelWriter.Create(ctx, &p); err != nil {
			return Project{}, errors.Wrap(err, m.path)
		}
	}
	return proj, nil
}

// ImportObjects returns the type-level ontology ID of every resource kind Import
// creates for files: the project, each distinct member type, and the group when the
// bundle carries directories. Callers enforce access on the result before any import
// work runs. Format routing mirrors Import, so an invalid bundle fails here with the
// same error.
func (s *Service) ImportObjects(
	ctx context.Context,
	files zip.Files,
) ([]ontology.ID, error) {
	var (
		objects = []ontology.ID{{Type: ontology.ResourceTypeProject}}
		seen    = set.New(ontology.ResourceTypeProject)
	)
	add := func(t ontology.ResourceType) {
		if seen.Contains(t) {
			return
		}
		seen.Add(t)
		objects = append(objects, ontology.ID{Type: t})
	}
	manifestFileName := imex.ManifestBaseName + json.Codec.Extension()
	manifestData, ok := files[manifestFileName]
	var (
		members []importMember
		err     error
	)
	if !ok {
		layoutData, legacy := files[legacyLayoutFileName]
		if !legacy {
			return nil, errors.Wrapf(
				validate.ErrValidation, "bundle holds no %s", manifestFileName,
			)
		}
		var slice legacySlice
		if err = json.Codec.Decode(ctx, layoutData, &slice); err != nil {
			return nil, errors.Wrap(err, legacyLayoutFileName)
		}
		legMembers, err := s.legacyMembers(ctx, slice, files)
		if err != nil {
			return nil, err
		}
		for _, m := range legMembers {
			members = append(members, m.importMember)
		}
		if legacyHasPanels(ctx, layoutData, legMembers) {
			add(ontology.ResourceTypePanel)
		}
	} else {
		if _, err = decodeManifest(ctx, manifestData, manifestFileName); err != nil {
			return nil, err
		}
		if members, err = s.bundleMembers(ctx, files); err != nil {
			return nil, err
		}
	}
	for _, m := range members {
		if m.env.Type == string(ontology.ResourceTypePanel) {
			add(ontology.ResourceTypePanel)
			continue
		}
		resourceType, err := s.cfg.ImEx.ImporterType(m.env.Type)
		if err != nil {
			return nil, errors.Wrap(err, m.path)
		}
		add(resourceType)
	}
	for _, m := range members {
		if parentDir(m.path) != "" {
			add(ontology.ResourceTypeGroup)
			break
		}
	}
	return objects, nil
}

// importMember pairs a member path with its decoded envelope, its type resolved to a
// registration type.
type importMember struct {
	path string
	env  imex.Envelope
}

// decodeManifest reads and guards the bundle manifest: the type must be "project" and
// the version exactly the current one, since version 0 wrote no manifest.
func decodeManifest(
	ctx context.Context,
	data []byte,
	fileName string,
) (imex.Manifest, error) {
	var manifest imex.Manifest
	if err := json.Codec.Decode(ctx, data, &manifest); err != nil {
		return imex.Manifest{}, errors.Wrap(err, fileName)
	}
	if manifest.Type != manifestType {
		return imex.Manifest{}, errors.Wrapf(
			validate.ErrValidation,
			"bundle is a %q, not a %s", manifest.Type, manifestType,
		)
	}
	if manifest.Version > manifestVersion {
		return imex.Manifest{}, imex.NewErrUnsupportedVersion(
			manifestType, imex.Version(manifest.Version), manifestVersion,
		)
	}
	if manifest.Version != manifestVersion {
		return imex.Manifest{}, errors.Wrapf(
			validate.ErrValidation,
			"unsupported manifest version %d", manifest.Version,
		)
	}
	return manifest, nil
}

// bundleMembers enumerates a version 1 bundle's members — every JSON file under the
// root except the manifest — in sorted path order, each decoded and resolved to its
// registration type. Member names that collide case-folded within one directory are a
// validation error.
func (s *Service) bundleMembers(
	ctx context.Context,
	files zip.Files,
) ([]importMember, error) {
	var (
		ext            = json.Codec.Extension()
		foldedManifest = filename.Fold(imex.ManifestBaseName + ext)
		foldedExt      = filename.Fold(ext)
		paths          = make([]string, 0, len(files))
	)
	for path := range files {
		folded := filename.Fold(path)
		if folded == foldedManifest || !strings.HasSuffix(folded, foldedExt) {
			continue
		}
		paths = append(paths, path)
	}
	slices.Sort(paths)
	if err := validateMemberPaths(paths); err != nil {
		return nil, err
	}
	members := make([]importMember, 0, len(paths))
	for _, path := range paths {
		var env imex.Envelope
		if err := json.Codec.Decode(ctx, files[path], &env); err != nil {
			return nil, errors.Wrap(err, path)
		}
		if env.Type != string(ontology.ResourceTypePanel) {
			typ, err := s.cfg.ImEx.ResolveType(env)
			if err != nil {
				return nil, errors.Wrap(err, path)
			}
			env.Type = typ
		}
		members = append(members, importMember{path: path, env: env})
	}
	return members, nil
}

// validateMemberPaths rejects two names in one directory that compare equal
// case-folded: the Console extracts bundles onto case-insensitive filesystems, where
// the pair collides. File and directory names share each directory's namespace.
func validateMemberPaths(paths []string) error {
	type segment struct {
		path string
		name string
		dir  bool
	}
	claimed := map[string]segment{}
	for _, path := range paths {
		var (
			segs   = strings.Split(path, "/")
			prefix = ""
		)
		for i, seg := range segs {
			var (
				isDir = i < len(segs)-1
				key   = prefix + "\x00" + filename.Fold(seg)
			)
			prev, ok := claimed[key]
			if !ok {
				claimed[key] = segment{path: path, name: seg, dir: isDir}
			} else if !prev.dir || !isDir || prev.name != seg {
				return errors.Wrapf(
					validate.ErrValidation,
					"members %q and %q have the same file name; "+
						"rename one and import again",
					prev.path,
					path,
				)
			}
			prefix += filename.Fold(seg) + "/"
		}
	}
	return nil
}

// parentDir returns the directory holding path, or "" at the bundle root.
func parentDir(path string) string {
	if i := strings.LastIndexByte(path, '/'); i >= 0 {
		return path[:i]
	}
	return ""
}

// createGroups recreates each bundle directory as a group, outer before inner, rooted
// at the project, and returns each directory's group ontology ID keyed by path ("" is
// the project itself).
func (s *Service) createGroups(
	ctx context.Context,
	tx gorp.Tx,
	projID ontology.ID,
	members []importMember,
) (map[string]ontology.ID, error) {
	dirs := set.New[string]()
	for _, m := range members {
		for dir := parentDir(m.path); dir != ""; dir = parentDir(dir) {
			dirs.Add(dir)
		}
	}
	var (
		groups = map[string]ontology.ID{"": projID}
		writer = s.cfg.Group.NewWriter(tx)
	)
	for _, dir := range slices.Sorted(maps.Keys(dirs)) {
		name := dir[strings.LastIndexByte(dir, '/')+1:]
		g, err := writer.Create(ctx, name, groups[parentDir(dir)])
		if err != nil {
			return nil, errors.Wrap(err, dir)
		}
		groups[dir] = g.OntologyID()
	}
	return groups, nil
}
