// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package bundle exports a project and its members as a directory of files, and imports
// one back as a new project.
package bundle

import (
	"context"
	"maps"
	"path"
	"slices"
	"strings"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/panel"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/project/versions/legacy"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/encoding"
	"github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/encoding/zip"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/filename"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

// Config is the configuration for creating a Service.
type Config struct {
	// Resource names the members of a bundle.
	//
	// [REQUIRED]
	Resource *resource.Service
	// Project stores the projects bundles hold.
	//
	// [REQUIRED]
	Project *project.Service
	// Group stores the groups that become bundle directories.
	//
	// [REQUIRED]
	Group *group.Service
	// Panel stores the panels a bundle holds.
	//
	// [REQUIRED]
	Panel *panel.Service
	// ImEx is the leaf import/export registry member documents go through.
	//
	// [REQUIRED]
	ImEx *imex.Service
}

var _ config.Config[Config] = Config{}

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.Resource = override.Nil(c.Resource, other.Resource)
	c.Project = override.Nil(c.Project, other.Project)
	c.Group = override.Nil(c.Group, other.Group)
	c.Panel = override.Nil(c.Panel, other.Panel)
	c.ImEx = override.Nil(c.ImEx, other.ImEx)
	return c
}

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("project.bundle")
	v.NotNil("resource", c.Resource)
	v.NotNil("project", c.Project)
	v.NotNil("group", c.Group)
	v.NotNil("panel", c.Panel)
	v.NotNil("imex", c.ImEx)
	return v.Error()
}

// Service exports and imports project bundles.
type Service struct{ cfg Config }

// NewService returns a Service built from the configs, each overriding the one before.
func NewService(cfgs ...Config) (*Service, error) {
	cfg, err := config.New(Config{}, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{cfg: cfg}, nil
}

const (
	manifestVersion = 1
	manifestType    = "project"
)

// unreachedTypes are the resource types a bundle can hold that the walk over the
// project's members never reaches, since no project holds a task or an Arc. A panel tab
// is the only way one enters a bundle, and an import leaves each where its importer put
// it rather than under the directory that held it.
var unreachedTypes = set.New(
	resource.TypeTask,
	resource.TypeArc,
)

// Export serializes the project identified by key, as snap holds it, and the groups and
// visualizations it holds as a bundle: one envelope per member document and panel,
// group children as directories, and a manifest naming the project at the root. It also
// returns the resource ID of every document, panel, and group that shaped the artifact,
// so a caller can enforce access on them. The encoder decides both the serialization
// and the extension every file takes.
//
// A child that is not a panel, a group, or a type the leaf registry exports is skipped,
// along with every panel tab that references it. The tasks and Arcs a panel's tabs
// reference export into the panel's directory, since the walk never reaches them
// otherwise. A group with no exported descendants is dropped. Two members of one
// directory that resolve to one file name, or a member claiming a reserved root file
// name, keep distinct names through a numeric suffix. It returns query.ErrNotFound if
// no project has key.
func (s *Service) Export(
	ctx context.Context,
	snap gorp.Snapshot,
	key project.Key,
	encoder encoding.FileEncoder,
) (zip.Files, []resource.ID, error) {
	var proj project.Project
	if err := s.cfg.Project.NewRetrieve().
		Where(project.MatchKeys(key)).
		Entry(&proj).
		Exec(ctx, snap); err != nil {
		return nil, nil, err
	}
	w := &bundleWalk{
		svc:     s,
		snap:    snap,
		ext:     encoder.Extension(),
		visited: set.New[resource.ID](),
		refs:    map[resource.ID]string{},
	}
	grouped, err := s.groupedMembers(ctx, snap, project.ResourceID(key))
	if err != nil {
		return nil, nil, err
	}
	w.grouped = grouped
	manifestFileName := imex.ManifestBaseName + w.ext
	claims := imex.NewClaims(manifestFileName)
	if err = w.directory(ctx, project.ResourceID(key), "", claims); err != nil {
		return nil, nil, err
	}
	files := make(zip.Files, len(w.refs)+len(w.panels)+1)
	for id, path := range w.refs {
		env, err := s.cfg.ImEx.Export(ctx, snap, id)
		if err != nil {
			return nil, nil, err
		}
		if files[path], err = encoder.Encode(env); err != nil {
			return nil, nil, err
		}
	}
	if err = w.encodePanels(files, encoder); err != nil {
		return nil, nil, err
	}
	manifest, err := encoder.Encode(imex.Manifest{
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
// project's groups.
type bundleWalk struct {
	svc *Service
	// snap is the snapshot every read in the walk runs on, so the bundle reflects one
	// committed state.
	snap gorp.Snapshot
	// ext is the extension the encoder gives every member file.
	ext string
	// visited holds every resource the walk already placed, so a resource with multiple
	// parents in the project is placed only once.
	visited set.Set[resource.ID]
	// refs maps each member document to its path from the bundle root. Panel encoding
	// resolves resource tabs through it.
	refs map[resource.ID]string
	// panels pairs each member panel's document with its path from the bundle root.
	panels []bundlePanel
	// groups holds every group that contributed a directory to the artifact.
	groups []resource.ID
	// grouped holds every member that sits in one of the project's groups, so the walk
	// places it in its group's directory and not at the bundle root.
	grouped set.Set[resource.ID]
}

// bundlePanel pairs a member panel's document with its path from the bundle root.
type bundlePanel struct {
	panel panel.Panel
	path  string
}

// members returns the ID of every resource that shaped the artifact, for access
// enforcement by the caller.
func (w *bundleWalk) members() []resource.ID {
	members := make([]resource.ID, 0, len(w.refs)+len(w.panels)+len(w.groups))
	for id := range w.refs {
		members = append(members, id)
	}
	for _, p := range w.panels {
		members = append(members, p.panel.ResourceID())
	}
	return append(members, w.groups...)
}

// directory walks the children of parent into the directory at prefix ("" for the
// bundle root, "a/b/" otherwise). claims tracks the file names taken in this directory.
func (w *bundleWalk) directory(
	ctx context.Context,
	parent resource.ID,
	prefix string,
	claims *imex.Claims,
) error {
	children, err := w.children(ctx, parent)
	if err != nil {
		return err
	}
	imex.SortResources(children)
	for _, child := range children {
		if w.visited.Contains(child.ID) {
			continue
		}
		switch {
		case child.ID.Type == resource.TypeGroup:
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
		case child.ID.Type == resource.TypePanel:
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
			if err = w.panelRefs(ctx, p, prefix, claims); err != nil {
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

// children returns the resources in the directory of parent: the groups and members of
// a group, or the groups and ungrouped members of the project.
func (w *bundleWalk) children(
	ctx context.Context,
	parent resource.ID,
) ([]resource.Resource, error) {
	var ids []resource.ID
	if parent.Type == resource.TypeProject {
		key, err := project.KeyFromResourceID(parent)
		if err != nil {
			return nil, err
		}
		deps, err := w.svc.cfg.Project.Children(ctx, w.snap, key)
		if err != nil {
			return nil, err
		}
		for _, id := range deps {
			if !w.grouped.Contains(id) {
				ids = append(ids, id)
			}
		}
	} else {
		key, err := group.KeyFromResourceID(parent)
		if err != nil {
			return nil, err
		}
		if ids, err = w.svc.cfg.Group.Children(ctx, w.snap, key); err != nil {
			return nil, err
		}
	}
	return w.svc.cfg.Resource.Retrieve(ctx, w.snap, ids...)
}

// groupedMembers returns every member of the groups under the project with id, as seen
// by snap.
func (s *Service) groupedMembers(
	ctx context.Context,
	snap gorp.Snapshot,
	id resource.ID,
) (set.Set[resource.ID], error) {
	var (
		members = set.New[resource.ID]()
		next    = []resource.ID{id}
	)
	for len(next) > 0 {
		var groups []group.Group
		if err := s.cfg.Group.NewRetrieve().
			Where(group.MatchParents(next...)).
			Entries(&groups).
			Exec(ctx, snap); err != nil {
			return nil, err
		}
		next = next[:0]
		for _, g := range groups {
			children, err := s.cfg.Group.Children(ctx, snap, g.Key)
			if err != nil {
				return nil, err
			}
			members.Add(children...)
			next = append(next, g.ResourceID())
		}
	}
	return members, nil
}

// retrievePanel reads the panel document behind id from the snapshot of the walk.
func (w *bundleWalk) retrievePanel(
	ctx context.Context,
	id resource.ID,
) (panel.Panel, error) {
	keys, err := panel.KeysFromResourceIDs([]resource.ID{id})
	if err != nil {
		return panel.Panel{}, err
	}
	var p panel.Panel
	err = w.svc.cfg.Panel.NewRetrieve().
		Where(panel.MatchKeys(keys...)).
		Entry(&p).
		Exec(ctx, w.snap)
	return p, err
}

// panelRefs places the resources p's resource tabs reference, and that the directory
// walk cannot reach, into the panel's directory. A reference to a resource that no
// longer exists is skipped, along with the tab that holds it.
func (w *bundleWalk) panelRefs(
	ctx context.Context,
	p panel.Panel,
	prefix string,
	claims *imex.Claims,
) error {
	var ids []resource.ID
	for _, id := range panel.ResourceRefs(p.Root) {
		if !w.visited.Contains(id) &&
			unreachedTypes.Contains(id.Type) &&
			w.svc.cfg.ImEx.ExporterRegistered(id.Type) {
			ids = append(ids, id)
		}
	}
	members, err := w.svc.cfg.Resource.Retrieve(ctx, w.snap, ids...)
	if err != nil {
		return err
	}
	imex.SortResources(members)
	for _, m := range members {
		fileName, err := filename.Sanitize(m.Name, w.ext)
		if err != nil {
			return err
		}
		w.visited.Add(m.ID)
		w.refs[m.ID] = prefix + claims.Claim(fileName, w.ext)
	}
	return nil
}

func (w *bundleWalk) encodePanels(files zip.Files, encoder encoding.FileEncoder) error {
	for _, bp := range w.panels {
		env, err := panel.EncodeBundle(bp.panel, w.refs)
		if err != nil {
			return err
		}
		if files[bp.path], err = encoder.Encode(env); err != nil {
			return err
		}
	}
	return nil
}

// Bundle is a project bundle that Decode read, ready for Import.
type Bundle struct {
	// name names the project that Import creates.
	name string
	// members are the members of a version 1 bundle.
	members []importMember
	// layout is the LAYOUT.json of a legacy (version 0) bundle, and nil for a version 1
	// bundle.
	layout []byte
	// legacyMembers are the members of a legacy bundle.
	legacyMembers []legacyMember
}

type legacyMember struct {
	path string
	// layoutKey is the key mosaic tabs reference the member by.
	layoutKey string
	write     imex.Write
}

// Decode reads the bundle in files and migrates each member, except panels, to its
// current shape. It needs no transaction, so a caller can run it before taking the
// write lock. The manifest, or the extension-stripped fileName, names the project. A
// root LAYOUT.json with no manifest is the legacy (version 0) format. Invalid bundles
// return validation errors naming the offending file.
func (s *Service) Decode(
	ctx context.Context,
	files zip.Files,
	fileName string,
) (Bundle, error) {
	manifestFileName := imex.ManifestBaseName + json.Codec.Extension()
	manifestData, ok := files[manifestFileName]
	if !ok {
		layout, ok := files[legacy.LayoutFileName]
		if !ok {
			return Bundle{}, errors.Wrapf(
				validate.ErrValidation, "bundle holds no %s", manifestFileName,
			)
		}
		members, err := legacy.Members(ctx, layout, files)
		if err != nil {
			return Bundle{}, err
		}
		b := Bundle{
			name:          filename.Stem(fileName),
			layout:        layout,
			legacyMembers: make([]legacyMember, len(members)),
		}
		for i, m := range members {
			write, err := s.cfg.ImEx.Decode(ctx, m.Env, m.Path)
			if err != nil {
				return Bundle{}, errors.Wrap(err, m.Path)
			}
			b.legacyMembers[i] = legacyMember{
				path:      m.Path,
				layoutKey: m.LayoutKey,
				write:     write,
			}
		}
		return b, nil
	}
	manifest, err := decodeManifest(manifestData, manifestFileName)
	if err != nil {
		return Bundle{}, err
	}
	name := manifest.Name
	if name == "" {
		name = filename.Stem(fileName)
	}
	members, err := s.bundleMembers(files)
	if err != nil {
		return Bundle{}, err
	}
	for i, m := range members {
		if m.env.Type == string(resource.TypePanel) {
			continue
		}
		if members[i].write, err = s.cfg.ImEx.Decode(ctx, m.env, m.path); err != nil {
			return Bundle{}, errors.Wrap(err, m.path)
		}
	}
	return Bundle{name: name, members: members}, nil
}

// Import creates a fresh project from b on tx: directories become groups, members
// import through the leaf registry, and panels are recreated with their resource tabs
// resolved to the minted IDs.
func (s *Service) Import(
	ctx context.Context,
	tx gorp.Tx,
	b Bundle,
) (project.Project, error) {
	if b.layout != nil {
		return s.importLegacy(ctx, tx, b)
	}
	proj := project.Project{Name: b.name}
	if err := s.cfg.Project.NewWriter(tx).Create(ctx, &proj); err != nil {
		return project.Project{}, err
	}
	projID := project.ResourceID(proj.Key)
	groups, err := s.createGroups(ctx, tx, projID, b.members)
	if err != nil {
		return project.Project{}, err
	}
	refs := make(map[string]resource.ID, len(b.members))
	for _, m := range b.members {
		if m.env.Type == string(resource.TypePanel) {
			continue
		}
		id, err := m.write(ctx, tx, imex.ImportOptions{
			Parent: projID,
			Group:  groups[path.Dir(m.path)],
		})
		if err != nil {
			return project.Project{}, errors.Wrap(err, m.path)
		}
		refs[m.path] = id
	}
	panelWriter := s.cfg.Panel.NewWriter(tx)
	for _, m := range b.members {
		if m.env.Type != string(resource.TypePanel) {
			continue
		}
		p, err := panel.DecodeBundle(ctx, m.env, refs)
		if err != nil {
			return project.Project{}, errors.Wrap(err, m.path)
		}
		if p.Name == "" {
			p.Name = filename.Stem(m.path)
		}
		p.Project = &proj.Key
		p.Group = groups[path.Dir(m.path)]
		if err = panelWriter.Create(ctx, &p); err != nil {
			return project.Project{}, errors.Wrap(err, m.path)
		}
	}
	return proj, nil
}

// importLegacy recreates a legacy (version 0) project directory: the documents its
// layout records reference, then each window's mosaic tiling as a panel of resource
// tabs.
func (s *Service) importLegacy(
	ctx context.Context,
	tx gorp.Tx,
	b Bundle,
) (project.Project, error) {
	proj := project.Project{Name: b.name}
	if err := s.cfg.Project.NewWriter(tx).Create(ctx, &proj); err != nil {
		return project.Project{}, err
	}
	projID := project.ResourceID(proj.Key)
	refs := make(map[string]resource.ID, len(b.legacyMembers))
	for _, m := range b.legacyMembers {
		id, err := m.write(ctx, tx, imex.ImportOptions{Parent: projID})
		if err != nil {
			return project.Project{}, errors.Wrap(err, m.path)
		}
		refs[m.layoutKey] = id
	}
	if err := legacy.CreatePanels(
		ctx, tx, s.cfg.Panel, proj.Key, b.layout, refs,
	); err != nil {
		return project.Project{}, err
	}
	return proj, nil
}

type importMember struct {
	path string
	// env is the member's decoded envelope, its type resolved to a registration type.
	env imex.Envelope
	// write persists the member, and is nil for a panel.
	write imex.Write
}

// decodeManifest reads and guards the bundle manifest: the type must be "project" and
// the version exactly the current one, since version 0 wrote no manifest.
func decodeManifest(data []byte, fileName string) (imex.Manifest, error) {
	var manifest imex.Manifest
	if err := imex.Codec.Decode(data, &manifest); err != nil {
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

// bundleMembers enumerates and decodes a version 1 bundle's members — every JSON file
// under the root except the manifest — in sorted path order.
func (s *Service) bundleMembers(files zip.Files) ([]importMember, error) {
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
		if err := imex.Codec.Decode(files[path], &env); err != nil {
			return nil, errors.Wrap(err, path)
		}
		if env.Type != string(resource.TypePanel) {
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
// case-folded, which collide when the Console extracts a bundle onto a case-insensitive
// filesystem. File and directory names share the namespace.
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
					"members %q and %q have the same file name; rename one and import again",
					prev.path,
					path,
				)
			}
			prefix += filename.Fold(seg) + "/"
		}
	}
	return nil
}

// createGroups recreates each bundle directory as a group, outer before inner, and
// returns each directory's group key keyed by path. The bundle root (".") has no group.
func (s *Service) createGroups(
	ctx context.Context,
	tx gorp.Tx,
	projID resource.ID,
	members []importMember,
) (map[string]*group.Key, error) {
	dirs := set.New[string]()
	for _, m := range members {
		for dir := path.Dir(m.path); dir != "."; dir = path.Dir(dir) {
			dirs.Add(dir)
		}
	}
	var (
		groups = map[string]*group.Key{".": nil}
		writer = s.cfg.Group.NewWriter(tx)
	)
	for _, dir := range slices.Sorted(maps.Keys(dirs)) {
		parent := projID
		if outer := groups[path.Dir(dir)]; outer != nil {
			parent = group.ResourceID(*outer)
		}
		g := group.Group{Name: dir[strings.LastIndexByte(dir, '/')+1:], Parent: &parent}
		if err := writer.Create(ctx, &g); err != nil {
			return nil, errors.Wrap(err, dir)
		}
		groups[dir] = &g.Key
	}
	return groups, nil
}
