// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package imex generates the portable import/export machinery for any Oracle struct
// that declares the bare `@go imex` marker. Every versions/vK package gains a Version
// constant equal to K; the versions package root gains Latest, Floor (the earliest
// version whose schema snapshot already carried the marker), and a decodeMigrate
// ladder that lifts server-era envelopes through the per-version Migrate<Type>
// steps. Latest keeps the wire envelope version and the storage schema version a
// single sequence per resource, and the ladder extends itself on every version bump.
package imex

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"text/template"

	"github.com/synnaxlabs/oracle/exec"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/plugin/go/internal/naming"
	"github.com/synnaxlabs/oracle/plugin/go/internal/versioning"
	"github.com/synnaxlabs/oracle/plugin/gomod"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/x/errors"
)

// goModulePrefix resolves repo-relative output paths to Go import paths when no go.mod
// is discoverable, mirroring the go/types plugin.
const goModulePrefix = "github.com/synnaxlabs/synnax/"

// Options configures the imex plugin output.
type Options struct {
	// FileNamePattern is the basename written for each output package.
	FileNamePattern string
}

// DefaultOptions returns the production defaults: imex.gen.go.
func DefaultOptions() Options { return Options{FileNamePattern: "imex.gen.go"} }

// Plugin emits the generated imex version constants.
type Plugin struct{ options Options }

// New constructs a Plugin with the given options.
func New(opts Options) *Plugin { return &Plugin{options: opts} }

func (*Plugin) Name() string                { return "go/imex" }
func (*Plugin) Domains() []string           { return []string{"go"} }
func (*Plugin) Requires() []string          { return []string{"go/types"} }
func (*Plugin) Check(*plugin.Request) error { return nil }

var goPostWriter = &exec.PostWriter{Commands: [][]string{{"gofmt", "-w"}}}

// PostWrite runs gofmt on the generated files.
func (*Plugin) PostWrite(files []string) error { return goPostWriter.PostWrite(files) }

// Generate emits imex machinery into the versions tree of every output package whose
// type declares @go imex: one Version constant per versions/vK package (the version
// directories on disk, plus the current version), and Latest, Floor, and the
// decodeMigrate ladder in the versions package root. It errors when a marked type lacks
// a @go version or when two types at the same path both carry the marker.
func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	resp := &plugin.Response{}
	declared := make(map[string]string)
	for _, typ := range req.Resolutions.TypesWithDomain("go") {
		if !domain.HasExprFromType(typ, "go", "imex") {
			continue
		}
		outputPath := output.GetPath(typ, "go")
		if outputPath == "" {
			continue
		}
		version, ok := versioning.Version(typ)
		if !ok {
			return nil, errors.Newf(
				"%s declares @go imex without @go version", typ.QualifiedName,
			)
		}
		if prev, dup := declared[outputPath]; dup {
			return nil, errors.Newf(
				"duplicate @go imex declarations for %s: %s and %s",
				outputPath, prev, typ.QualifiedName,
			)
		}
		declared[outputPath] = typ.QualifiedName
		versions, err := versionDirs(req.RepoRoot, outputPath)
		if err != nil {
			return nil, err
		}
		if !slices.Contains(versions, version) {
			versions = append(versions, version)
			slices.Sort(versions)
		}
		goName := naming.GetGoName(typ)
		for _, k := range versions {
			path := versioning.VersionedPath(outputPath, k)
			var buf bytes.Buffer
			if err := versionTemplate.Execute(&buf, &versionData{
				Package: naming.DerivePackageName(path),
				Type:    goName,
				Version: k,
			}); err != nil {
				return nil, err
			}
			resp.Files = append(resp.Files, plugin.File{
				Path:    path + "/" + p.options.FileNamePattern,
				Content: buf.Bytes(),
			})
		}
		floor := firstImexVersion(req, typ.QualifiedName, version)
		arms, err := chainArms(req.RepoRoot, outputPath, goName, floor, version)
		if err != nil {
			return nil, err
		}
		data := &chainData{
			Type:       goName,
			CurrentPkg: versioning.Dir(version),
			FloorPkg:   versioning.Dir(floor),
			Arms:       arms,
		}
		for k := floor; k <= version; k++ {
			data.Imports = append(data.Imports, gomod.ResolveImportPath(
				versioning.VersionedPath(outputPath, k), req.RepoRoot, goModulePrefix,
			))
		}
		var buf bytes.Buffer
		if err := chainTemplate.Execute(&buf, data); err != nil {
			return nil, err
		}
		resp.Files = append(resp.Files, plugin.File{
			Path:    outputPath + "/versions/" + p.options.FileNamePattern,
			Content: buf.Bytes(),
		})
		// The constants used to be emitted at the package root; clean the old location
		// up.
		resp.Deletions = append(
			resp.Deletions, outputPath+"/"+p.options.FileNamePattern,
		)
	}
	return resp, nil
}

// versionDirs returns the numeric version sub-directories present under the output
// package's versions/ tree on disk, ascending.
func versionDirs(repoRoot, outputPath string) ([]int, error) {
	entries, err := os.ReadDir(filepath.Join(repoRoot, outputPath, "versions"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var versions []int
	for _, e := range entries {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), "v") {
			continue
		}
		k, err := strconv.Atoi(e.Name()[1:])
		if err != nil {
			continue
		}
		versions = append(versions, k)
	}
	slices.Sort(versions)
	return versions, nil
}

type versionData struct {
	Package string
	Type    string
	Version int
}

var versionTemplate = template.Must(
	template.New("go-imex-version").Parse(`// Code generated by oracle. DO NOT EDIT.

package {{.Package}}

import "github.com/synnaxlabs/synnax/pkg/service/imex"

// Version is the portable schema version of the {{.Type}} shape this package defines.
const Version imex.Version = {{.Version}}
`),
)

// firstImexVersion returns the earliest @go version at which the type already carried
// the @go imex marker, walking schema snapshots newest-first. The walk ends at the
// first snapshot that is unparseable, predates per-resource versioning, or lacks the
// marker: marker history is contiguous. With no marked snapshot the floor is the
// current version.
func firstImexVersion(req *plugin.Request, qualifiedName string, current int) int {
	floor := current
	if req.LoadSnapshot == nil || req.SnapshotVersion <= 0 {
		return floor
	}
	for s := req.SnapshotVersion; s >= 1; s-- {
		t, err := req.LoadSnapshot(s)
		if err != nil || t == nil || versioning.PreVersioning(t) {
			break
		}
		typ, ok := t.Get(qualifiedName)
		if !ok || !domain.HasExprFromType(typ, "go", "imex") {
			break
		}
		v, ok := versioning.Version(typ)
		if !ok {
			break
		}
		if v < floor {
			floor = v
		}
	}
	return floor
}

// chainArms builds one ladder arm per version in [floor, current): each decodes the
// stamped vK shape and lifts it through every later bump's exported Migrate<Type> step.
// Alias-only bumps (no step on disk) pass the value through unchanged.
func chainArms(
	repoRoot, outputPath, goName string,
	floor, current int,
) ([]chainArm, error) {
	if floor >= current {
		return nil, nil
	}
	steps := make(map[int]bool, current-floor)
	for j := floor + 1; j <= current; j++ {
		hasStep, err := migrateStepExists(
			filepath.Join(repoRoot, versioning.VersionedPath(outputPath, j)), goName,
		)
		if err != nil {
			return nil, err
		}
		steps[j] = hasStep
	}
	arms := make([]chainArm, 0, current-floor)
	for k := floor; k < current; k++ {
		var b strings.Builder
		cur := fmt.Sprintf("t%d", k)
		fmt.Fprintf(
			&b, "\t\t%s, err := imex.Decode[%s.%s](ctx, env)\n",
			cur, versioning.Dir(k), goName,
		)
		fmt.Fprintf(&b, "\t\tif err != nil {\n\t\t\treturn %s{}, err\n\t\t}\n", goName)
		for j := k + 1; j <= current; j++ {
			if !steps[j] {
				continue
			}
			next := fmt.Sprintf("t%d", j)
			fmt.Fprintf(
				&b, "\t\t%s, err := %s.Migrate%s(ctx, %s)\n",
				next, versioning.Dir(j), goName, cur,
			)
			fmt.Fprintf(
				&b, "\t\tif err != nil {\n\t\t\treturn %s{}, err\n\t\t}\n", goName,
			)
			cur = next
		}
		fmt.Fprintf(&b, "\t\treturn %s, nil", cur)
		arms = append(arms, chainArm{Pkg: versioning.Dir(k), Body: b.String()})
	}
	return arms, nil
}

// migrateStepExists reports whether the version package at dir declares the scaffolded
// per-bump lift Migrate<goName>.
func migrateStepExists(dir, goName string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	name := "Migrate" + goName
	fset := token.NewFileSet()
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") ||
			strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		f, err := parser.ParseFile(
			fset, filepath.Join(dir, e.Name()), nil, parser.SkipObjectResolution,
		)
		if err != nil {
			continue
		}
		for _, d := range f.Decls {
			if fn, ok := d.(*ast.FuncDecl); ok && fn.Recv == nil &&
				fn.Name.Name == name {
				return true, nil
			}
		}
	}
	return false, nil
}

type chainArm struct {
	Pkg  string
	Body string
}

type chainData struct {
	Type       string
	CurrentPkg string
	FloorPkg   string
	Imports    []string
	Arms       []chainArm
}

var chainTemplate = template.Must(
	template.New("go-imex-chain").Parse(`// Code generated by oracle. DO NOT EDIT.

package versions

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/service/imex"
{{- range .Imports}}
	"{{.}}"
{{- end}}
)

// Latest is the portable schema version stamped on exported {{.Type}} envelopes and the
// highest version import accepts. It equals the resource's current schema version.
const Latest = {{.CurrentPkg}}.Version

// Floor is the earliest server-exported schema version: the version the resource
// carried when it first declared @go imex.
const Floor = {{.FloorPkg}}.Version

// decodeMigrate decodes an envelope stamped in [Floor, Latest] as its version's
// {{.Type}} shape and lifts it through the per-version migration chain to the current
// shape. Envelopes outside the window are rejected with a path-scoped validation error.
func decodeMigrate(ctx context.Context, env imex.Envelope) ({{.Type}}, error) {
	switch env.Version {
{{- range .Arms}}
	case {{.Pkg}}.Version:
{{.Body}}
{{- end}}
	case {{.CurrentPkg}}.Version:
		return imex.Decode[{{.Type}}](ctx, env)
	}
	return {{.Type}}{}, imex.NewErrUnsupportedVersion(env.Type, env.Version, Latest)
}
`),
)
