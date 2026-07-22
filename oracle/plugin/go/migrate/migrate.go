// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package migrate provides an Oracle plugin that generates migration files.
// Every version of a resource — current included — lives in its own
// types/vN/ sub-package, numbered by the per-resource @go version. A bump
// scaffolds the incoming version (auto-copy, developer transform template);
// the outgoing package freezes by no longer being a generation target.
package migrate

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"text/template"

	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/go/internal/naming"
	"github.com/synnaxlabs/oracle/plugin/go/internal/schemadiff"
	"github.com/synnaxlabs/oracle/plugin/go/internal/versioning"
	"github.com/synnaxlabs/oracle/plugin/gomod"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
)

type Plugin struct{}

func New() *Plugin                            { return &Plugin{} }
func (p *Plugin) Name() string                { return "go/migrate" }
func (p *Plugin) Domains() []string           { return []string{"go"} }
func (p *Plugin) Requires() []string          { return []string{"go/types", "go/marshal"} }
func (p *Plugin) Check(*plugin.Request) error { return nil }

// SnapshotPreVersioning reports whether the snapshot table predates
// per-resource @go version declarations and therefore cannot drive migration
// diffing.
func SnapshotPreVersioning(table *resolution.Table) bool {
	return versioning.PreVersioning(table)
}

// bump records one path's version transition between the snapshot and the
// working schemas.
type bump struct {
	OldVersion int
	NewVersion int
	Changed    bool
}

// generation carries the cross-pass state for one Generate run.
type generation struct {
	req          *plugin.Request
	resp         *plugin.Response
	oldVersions  map[string]int
	newVersions  map[string]int
	oldLaidOut   map[string]int
	newLaidOut   map[string]int
	rewrittenNew *resolution.Table
	bumps        map[string]bump
}

func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	resp := &plugin.Response{Files: make([]plugin.File, 0)}
	if req.OldResolutions == nil {
		return resp, nil
	}

	g := &generation{req: req, resp: resp}
	var err error
	if g.newVersions, err = versioning.PathVersions(req.Resolutions); err != nil {
		return nil, err
	}
	if g.newLaidOut, err = versioning.EntryPaths(req.Resolutions); err != nil {
		return nil, err
	}
	// A snapshot cut before per-resource versioning existed carries no @go
	// version tags, so it cannot anchor bump detection or freezing. Version
	// diffing resumes once a post-versioning snapshot is cut.
	if versioning.PreVersioning(req.OldResolutions) {
		return resp, nil
	}
	if g.oldVersions, err = versioning.PathVersions(req.OldResolutions); err != nil {
		return nil, err
	}
	if g.oldLaidOut, err = versioning.EntryPaths(req.OldResolutions); err != nil {
		return nil, err
	}
	newCurrentMap := make(map[string]string, len(g.newLaidOut))
	for path, v := range g.newLaidOut {
		newCurrentMap[path] = versioning.VersionedPath(path, v)
	}
	g.rewrittenNew = versioning.RewriteOutputPaths(req.Resolutions, newCurrentMap)

	if err := g.detectBumps(); err != nil {
		return nil, err
	}

	paths := make([]string, 0, len(g.bumps))
	for path := range g.bumps {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		if _, laidOut := g.oldLaidOut[path]; !laidOut {
			// Paths first versioned after the snapshot have no outgoing
			// version to migrate from.
			continue
		}
		if err := g.scaffoldPath(path); err != nil {
			return nil, err
		}
	}
	return resp, nil
}

// detectBumps validates version discipline for every versioned path and
// records the paths transitioning to a new version. A shape change without a
// version bump, a skipped version, or a version decrease is an error.
func (g *generation) detectBumps() error {
	g.bumps = make(map[string]bump)
	paths := make([]string, 0, len(g.newVersions))
	for path := range g.newVersions {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		newV := g.newVersions[path]
		oldV, ok := g.oldVersions[path]
		if !ok {
			continue
		}
		changed := g.pathChanged(path)
		switch {
		case newV == oldV && !changed:
			continue
		case newV == oldV && changed:
			return errors.Newf(
				"schema shape for %s changed but @go version is still %d; bump it to %d",
				path, oldV, oldV+1,
			)
		case newV == oldV+1:
			g.bumps[path] = bump{OldVersion: oldV, NewVersion: newV, Changed: changed}
		case newV < oldV:
			return errors.Newf(
				"@go version for %s decreased from %d to %d; versions never decrease",
				path, oldV, newV,
			)
		default:
			return errors.Newf(
				"@go version for %s jumped from %d to %d; versions are dense — bump one at a time",
				path, oldV, newV,
			)
		}
	}
	return nil
}

// pathChanged reports whether any type present in the snapshot changed shape
// or was removed at path. Brand-new types never count: no frozen shape can
// reference a type that didn't exist yet, so they fold into the current
// version without a bump.
func (g *generation) pathChanged(path string) bool {
	oldByName := versioning.TypesAtPath(g.req.OldResolutions, path)
	newByName := versioning.TypesAtPath(g.req.Resolutions, path)
	for name, oldType := range oldByName {
		newType, ok := newByName[name]
		if !ok {
			return true
		}
		if !schemadiff.SchemasEqual(oldType, newType, g.req.OldResolutions, g.req.Resolutions) {
			return true
		}
	}
	return false
}

// scaffoldPath scaffolds the incoming version of a bumped path: auto-copy,
// developer transform template. The outgoing version package is already on
// disk in its final form — generation stops targeting it, which is what
// freezes it.
func (g *generation) scaffoldPath(path string) error {
	b := g.bumps[path]
	oldTable := g.req.OldResolutions
	roots := versioning.TypesAtPath(oldTable, path)
	// Pin every versioned path in the old table at its snapshot-declared
	// version so scaffolding references frozen directories. Paths without a
	// version stay unversioned, where their old types still live.
	oldPathMap, err := versioning.CurrentPathMap(oldTable)
	if err != nil {
		return err
	}
	rewrittenOld := versioning.RewriteOutputPaths(oldTable, oldPathMap)
	diff := g.pathDiff(path, roots)
	names := make([]string, 0, len(roots))
	for name := range roots {
		names = append(names, name)
	}
	sort.Strings(names)
	oldEntryTypes := make([]resolution.Type, 0, len(names))
	for _, name := range names {
		oldEntryTypes = append(oldEntryTypes, roots[name])
	}
	return g.scaffoldIncoming(path, b, roots, oldEntryTypes, diff, rewrittenOld)
}

// pathDiff computes the schema diff for a path's freeze, walking from its
// @go migrate entries when present and from every keyed struct otherwise.
// Codec-only bumps (no shape change) synthesize a full-copy diff for each
// root so the passthrough auto-copy still generates.
func (g *generation) pathDiff(
	path string,
	roots map[string]resolution.Type,
) map[string]schemadiff.TypeDiff {
	diff := make(map[string]schemadiff.TypeDiff)
	newByName := versioning.TypesAtPath(g.req.Resolutions, path)
	names := make([]string, 0, len(roots))
	for name := range roots {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		oldType := roots[name]
		if _, ok := oldType.Form.(resolution.StructForm); !ok {
			continue
		}
		newType, ok := newByName[name]
		if !ok {
			continue
		}
		for qn, td := range schemadiff.SchemaDiff(oldType, newType, g.req.OldResolutions, g.req.Resolutions) {
			if _, exists := diff[qn]; !exists {
				diff[qn] = td
			}
		}
	}
	if b := g.bumps[path]; !b.Changed {
		for _, name := range names {
			oldType := roots[name]
			if _, ok := oldType.Form.(resolution.StructForm); !ok {
				continue
			}
			if _, exists := diff[oldType.QualifiedName]; !exists {
				diff[oldType.QualifiedName] = schemadiff.TypeDiff{
					QualifiedName: oldType.QualifiedName,
					GoPath:        path,
					Kind:          schemadiff.TypeChanged,
				}
			}
		}
	}
	return diff
}

// scaffoldIncoming generates the incoming version's migration scaffolding:
// migrate_auto.gen.go and a migrate.go template for @go migrate entries, and
// the helpers.go carry-forward for every version-laid-out path.
func (g *generation) scaffoldIncoming(
	path string,
	b bump,
	roots map[string]resolution.Type,
	oldEntryTypes []resolution.Type,
	diff map[string]schemadiff.TypeDiff,
	rewrittenOld *resolution.Table,
) error {
	newPath := versioning.VersionedPath(path, b.NewVersion)
	newDir := versioning.Dir(b.NewVersion)

	newByName := versioning.TypesAtPath(g.req.Resolutions, path)
	hasMigrateEntry := false
	for name := range roots {
		if newType, ok := newByName[name]; ok && isMigrateEntry(newType) {
			hasMigrateEntry = true
			break
		}
	}
	if !hasMigrateEntry {
		// Paths without @go migrate entries (value types, hand-migrated
		// resources) get dep-style scaffolding: per-type Migrate wrappers over
		// the generated auto-copies.
		if !needsAutoMigrate(oldEntryTypes, diff) {
			return nil
		}
		return g.generateDepMigration(newDir, newPath, oldEntryTypes, diff, rewrittenOld)
	}

	if needsAutoMigrate(oldEntryTypes, diff) {
		autoCopyContent, err := generateAutoCopy(
			newDir, newPath, g.req.RepoRoot,
			oldEntryTypes, diff, rewrittenOld, g.rewrittenNew,
			false,
		)
		if err != nil {
			return errors.Wrapf(err, "failed to generate auto-copy for %s", newPath)
		}
		if autoCopyContent != nil {
			g.resp.Files = append(g.resp.Files, plugin.File{
				Path:    newPath + "/migrate_auto.gen.go",
				Content: autoCopyContent,
			})
		}
	}

	templateFile := newPath + "/migrate.go"
	if _, statErr := os.Stat(filepath.Join(g.req.RepoRoot, templateFile)); !os.IsNotExist(statErr) {
		return nil
	}
	oldVersionedPath := versioning.VersionedPath(path, b.OldVersion)
	oldImport := gomod.ResolveImportPath(
		oldVersionedPath, g.req.RepoRoot, gomod.DefaultModulePrefix,
	)
	// The predecessor is always a sibling version of the same resource, so it
	// imports under its bare directory name.
	oldAlias := filepath.Base(oldVersionedPath)
	names := make([]string, 0, len(roots))
	for name := range roots {
		names = append(names, name)
	}
	sort.Strings(names)
	var buf bytes.Buffer
	for _, name := range names {
		newType, ok := newByName[name]
		if !ok || !isMigrateEntry(newType) {
			continue
		}
		var tparams []resolution.TypeParam
		if sf, ok := newType.Form.(resolution.StructForm); ok {
			tparams = sf.TypeParams
		}
		tc, err := renderTransformTemplate(
			newDir, naming.GetGoName(newType), b.NewVersion, oldAlias, oldImport, tparams,
			buf.Len() > 0,
		)
		if err != nil {
			return errors.Wrap(err, "failed to generate transform template")
		}
		buf.Write(tc)
	}
	// Changed non-entry structs get wrappers too: the auto-copies reference
	// MigrateX for every nested changed struct.
	for _, name := range names {
		newType, ok := newByName[name]
		if !ok || isMigrateEntry(newType) {
			continue
		}
		td, ok := diff[roots[name].QualifiedName]
		if !ok || td.Kind != schemadiff.TypeChanged {
			continue
		}
		sf, isStruct := newType.Form.(resolution.StructForm)
		if !isStruct {
			continue
		}
		tc, err := renderTransformTemplate(
			newDir, naming.GetGoName(newType), b.NewVersion, oldAlias, oldImport,
			sf.TypeParams, buf.Len() > 0,
		)
		if err != nil {
			return errors.Wrap(err, "failed to generate transform template")
		}
		buf.Write(tc)
	}
	if buf.Len() > 0 {
		g.resp.Files = append(g.resp.Files, plugin.File{Path: templateFile, Content: buf.Bytes()})
	}
	return nil
}

// generateDepMigration emits the auto-copy and migrate template for a changed
// value-type dependency's frozen package.
func (g *generation) generateDepMigration(
	versionDir, mirroredPath string,
	types []resolution.Type,
	diff map[string]schemadiff.TypeDiff,
	rewrittenOld *resolution.Table,
) error {
	autoCopyContent, err := generateAutoCopy(
		versionDir, mirroredPath, g.req.RepoRoot,
		types, diff, rewrittenOld, g.rewrittenNew,
		true,
	)
	if err != nil {
		return errors.Wrapf(err, "failed to generate auto-copy for %s", mirroredPath)
	}
	if autoCopyContent != nil {
		g.resp.Files = append(g.resp.Files, plugin.File{
			Path:    mirroredPath + "/migrate_auto.gen.go",
			Content: autoCopyContent,
		})
	}

	migrateFile := mirroredPath + "/migrate.go"
	if _, statErr := os.Stat(filepath.Join(g.req.RepoRoot, migrateFile)); !os.IsNotExist(statErr) {
		return nil
	}
	tc, err := renderTypeMigrateTemplate(
		versionDir, mirroredPath, types, diff, g.rewrittenNew, g.req.RepoRoot,
	)
	if err != nil {
		return errors.Wrapf(err, "failed to generate type migrate template for %s", mirroredPath)
	}
	if tc != nil {
		g.resp.Files = append(g.resp.Files, plugin.File{Path: migrateFile, Content: tc})
	}
	return nil
}

// --- Templates ---

type versionImport struct{ Alias, Path string }

var transformTmpl = template.Must(template.New("transform").Parse(
	`{{if not .Continuation}}// Generated by oracle as a template. Edit this file.
//
// AutoMigrate handles field copying. Customize non-zero defaults below.

package {{.Package}}

import (
	"context"

	{{.VersionDir}} "{{.MigrationsImport}}"
)
{{end}}
// Migrate{{.GoName}} lifts a {{.VersionDir}} {{.GoName}} into the current shape.
func Migrate{{.GoName}}{{.TypeParamsDecl}}(ctx context.Context, old {{.VersionDir}}.{{.GoName}}{{.TypeParamsRef}}) ({{.GoName}}{{.TypeParamsRef}}, error) {
	return autoMigrate{{.GoName}}{{.TypeParamsRef}}(ctx, old)
}
`))

func renderTransformTemplate(
	pkg, goName string,
	version int,
	vDir, migrationsImport string,
	tparams []resolution.TypeParam,
	continuation bool,
) ([]byte, error) {
	var buf bytes.Buffer
	err := transformTmpl.Execute(&buf, struct {
		Package, GoName, VersionDir, MigrationsImport string
		TypeParamsDecl, TypeParamsRef                 string
		Version                                       int
		Continuation                                  bool
	}{
		Package:          pkg,
		GoName:           goName,
		VersionDir:       vDir,
		MigrationsImport: migrationsImport,
		TypeParamsDecl:   formatTypeParamsDecl(tparams),
		TypeParamsRef:    formatTypeParamsRef(tparams),
		Version:          version,
		Continuation:     continuation,
	})
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

var typeMigrateTmpl = template.Must(template.New("typeMigrate").Parse(
	`// Generated by oracle as a template. Edit this file.

package {{.Package}}

import (
	"context"
{{range .Imports}}
	{{.Alias}} "{{.Path}}"
{{- end}}
)
{{range .Functions}}
// Migrate{{.GoName}} lifts a {{.OldTypeName}} into the current shape.
func Migrate{{.GoName}}{{.TypeParamsDecl}}(ctx context.Context, old {{.OldTypeName}}) ({{.NewTypeName}}, error) {
	migrated, err := autoMigrate{{.GoName}}{{.TypeParamsRef}}(ctx, old)
	if err != nil {
		return {{.NewTypeName}}{}, err
	}
	// New/changed fields - set non-zero defaults if needed:
{{- range .NewFields}}
	// migrated.{{.}} is zero-valued
{{- end}}
	return migrated, nil
}
{{end}}`))

func renderTypeMigrateTemplate(
	pkg, mirroredPath string,
	types []resolution.Type,
	diff map[string]schemadiff.TypeDiff,
	newTable *resolution.Table,
	repoRoot string,
) ([]byte, error) {
	type tmplFunc struct {
		GoName, OldTypeName, NewTypeName string
		TypeParamsDecl, TypeParamsRef    string
		NewFields                        []string
	}
	type tmplData struct {
		Package   string
		Imports   []versionImport
		Functions []tmplFunc
	}
	data := tmplData{Package: pkg}
	importSet := make(map[string]versionImport)
	for _, typ := range types {
		td, ok := diff[typ.QualifiedName]
		if !ok || td.Kind != schemadiff.TypeChanged {
			continue
		}
		sf, isStruct := typ.Form.(resolution.StructForm)
		if !isStruct {
			continue
		}
		// Skip types that are their own migrate entries: their developer-edited
		// transform lives in their own incoming version package. Generating
		// both would duplicate the template.
		if newType, ok := newTable.Get(typ.QualifiedName); ok && isMigrateEntry(newType) {
			continue
		}
		goName := naming.GetGoName(typ)
		newType, _ := newTable.Get(typ.QualifiedName)
		newGoPath := output.GetPath(newType, "go")
		newTypeName := naming.GetGoName(newType)
		if newGoPath != mirroredPath {
			ip := gomod.ResolveImportPath(newGoPath, repoRoot, gomod.DefaultModulePrefix)
			alias := naming.DeriveVersionedAlias(newGoPath, pkg)
			if filepath.Dir(newGoPath) == filepath.Dir(mirroredPath) {
				alias = filepath.Base(newGoPath)
			}
			importSet[ip] = versionImport{Alias: alias, Path: ip}
			newTypeName = alias + "." + newTypeName
		}
		var newFields []string
		for _, fd := range td.ChangedFields {
			if fd.Kind == schemadiff.FieldKindAdded {
				newFields = append(newFields, naming.GetFieldName(*fd.NewField))
			}
		}
		tref := formatTypeParamsRef(sf.TypeParams)
		data.Functions = append(data.Functions, tmplFunc{
			GoName:         goName,
			OldTypeName:    goName + tref,
			NewTypeName:    newTypeName + tref,
			TypeParamsDecl: formatTypeParamsDecl(sf.TypeParams),
			TypeParamsRef:  tref,
			NewFields:      newFields,
		})
	}
	if len(data.Functions) == 0 {
		return nil, nil
	}
	for _, v := range importSet {
		data.Imports = append(data.Imports, v)
	}
	var buf bytes.Buffer
	if err := typeMigrateTmpl.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
