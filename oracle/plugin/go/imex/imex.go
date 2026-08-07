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
// constant equal to K; the versions package root gains Latest and an autoDecodeEnvelope
// ladder that lifts server-era envelopes through the per-version Migrate<Type> steps;
// the service package gains the Service.Export half of imex.ImportExporter. Latest
// keeps the wire envelope version and the storage schema version a single sequence per
// resource, and the ladder extends itself on every version bump. Import stays
// hand-written: every resource parents and persists differently.
package imex

import (
	"bytes"
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
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
	"github.com/synnaxlabs/x/set"
)

// goModulePrefix resolves repo-relative output paths to Go import paths when no go.mod
// is discoverable, mirroring the go/types plugin.
const goModulePrefix = "github.com/synnaxlabs/synnax/"

// Options configures the imex plugin output.
type Options struct {
	// FileNamePattern is the basename written for each output package.
	FileNamePattern string
	// RuntimeImportPath is the Go package supplying the Version type, Envelope, Decode,
	// and NewErrUnsupportedVersion the generated files call. Its package name must be
	// imex — the templates qualify every call with that identifier. Required.
	RuntimeImportPath string
	// OntologyImportPath is the Go package supplying the ID the generated Export takes.
	// Its package name must be ontology. Required.
	OntologyImportPath string
}

// DefaultOptions returns the production defaults: imex.gen.go against the Core's imex
// service package.
func DefaultOptions() Options {
	return Options{
		FileNamePattern:    "imex.gen.go",
		RuntimeImportPath:  "github.com/synnaxlabs/synnax/pkg/service/imex",
		OntologyImportPath: "github.com/synnaxlabs/synnax/pkg/service/ontology",
	}
}

// Plugin emits the generated imex version constants and Export methods.
type Plugin struct{ options Options }

// New constructs a Plugin with the given options.
func New(opts Options) *Plugin { return &Plugin{options: opts} }

func (*Plugin) Name() string       { return "go/imex" }
func (*Plugin) Domains() []string  { return []string{"go"} }
func (*Plugin) Requires() []string { return []string{"go/types"} }

// Check reports a Plugin constructed without one of the import paths the templates
// qualify against, which would emit files that do not compile.
func (p *Plugin) Check(*plugin.Request) error {
	if p.options.RuntimeImportPath == "" {
		return errors.New("go/imex requires Options.RuntimeImportPath")
	}
	if p.options.OntologyImportPath == "" {
		return errors.New("go/imex requires Options.OntologyImportPath")
	}
	return nil
}

var goPostWriter = &exec.PostWriter{Commands: [][]string{{"gofmt", "-w"}}}

// PostWrite runs gofmt on the generated files.
func (*Plugin) PostWrite(files []string) error { return goPostWriter.PostWrite(files) }

// Generate emits imex machinery for every output package whose type declares @go imex:
// one Version constant per versions/vK package the Core has exported, Latest plus the
// autoDecodeEnvelope ladder in the versions package root, and Service.Export in the
// service package. Versions below the floor predate Core export, so their constant is
// deleted instead. It errors when a marked type lacks a @go version, when two types at
// the same path both carry the marker, or when a version package on disk does not
// parse.
//
// The emitted Export calls the service's NewRetrieve, MatchKeys, and Type, and parses
// id.Key as a UUID. A resource missing any of them, or keyed by something else, fails
// to compile.
func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	ctx := context.Background()
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
		goName := naming.GetGoName(typ)
		floor, err := firstImexVersion(
			ctx, req, strings.TrimSuffix(typ.FilePath, ".oracle"), typ.Name, version,
		)
		if err != nil {
			return nil, err
		}
		for k := floor; k <= version; k++ {
			path := versioning.VersionedPath(outputPath, k)
			var buf bytes.Buffer
			if err := versionTemplate.Execute(&buf, &versionData{
				Package: naming.DerivePackageName(path),
				Type:    goName,
				Version: k,
				Runtime: p.options.RuntimeImportPath,
			}); err != nil {
				return nil, err
			}
			resp.Files = append(resp.Files, plugin.File{
				Path:    path + "/" + p.options.FileNamePattern,
				Content: buf.Bytes(),
			})
		}
		stale, err := versioning.VersionDirs(req.RepoRoot, outputPath)
		if err != nil {
			return nil, err
		}
		for _, k := range stale {
			if k < floor {
				resp.Deletions = append(
					resp.Deletions,
					versioning.VersionedPath(
						outputPath,
						k,
					)+"/"+p.options.FileNamePattern,
				)
			}
		}
		arms, err := chainArms(req.RepoRoot, outputPath, goName, floor, version)
		if err != nil {
			return nil, err
		}
		data := &chainData{
			Type:       goName,
			CurrentPkg: versioning.Dir(version),
			Arms:       arms,
			Runtime:    p.options.RuntimeImportPath,
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
		// A fresh buffer: plugin.File.Content aliases the backing array, so reusing buf
		// would overwrite the chain file appended above.
		var exportBuf bytes.Buffer
		if err := exportTemplate.Execute(&exportBuf, &exportData{
			Package: naming.DerivePackageName(outputPath),
			Type:    goName,
			Versions: gomod.ResolveImportPath(
				outputPath+"/versions", req.RepoRoot, goModulePrefix,
			),
			Runtime:  p.options.RuntimeImportPath,
			Ontology: p.options.OntologyImportPath,
		}); err != nil {
			return nil, err
		}
		resp.Files = append(resp.Files, plugin.File{
			Path:    outputPath + "/" + p.options.FileNamePattern,
			Content: exportBuf.Bytes(),
		})
	}
	return resp, nil
}

type exportData struct {
	Package  string
	Type     string
	Versions string
	Runtime  string
	Ontology string
}

var exportTemplate = template.Must(
	template.New("go-imex-export").Parse(`// Code generated by oracle. DO NOT EDIT.

package {{.Package}}

import (
	"context"

	"github.com/google/uuid"
	"{{.Runtime}}"
	"{{.Ontology}}"
	"{{.Versions}}"
)

// Export retrieves the {{.Type}} identified by id and serializes it, stamping
// versions.Latest. It returns query.ErrNotFound if no {{.Type}} has id.Key.
func (s *Service) Export(ctx context.Context, id ontology.ID) (imex.Envelope, error) {
	key, err := uuid.Parse(id.Key)
	if err != nil {
		return imex.Envelope{}, err
	}
	var v {{.Type}}
	if err = s.NewRetrieve().
		Where(MatchKeys(key)).
		Entry(&v).
		Exec(ctx, nil); err != nil {
		return imex.Envelope{}, err
	}
	env := imex.Envelope{
		Version: versions.Latest, Type: string(s.Type()), Name: v.Name,
	}
	if err = imex.Encode(&env, v); err != nil {
		return imex.Envelope{}, err
	}
	return env, nil
}
`),
)

type versionData struct {
	Package string
	Type    string
	Version int
	Runtime string
}

var versionTemplate = template.Must(
	template.New("go-imex-version").Parse(`// Code generated by oracle. DO NOT EDIT.

package {{.Package}}

import "{{.Runtime}}"

// Version is the portable schema version of the {{.Type}} shape this package defines.
const Version imex.Version = {{.Version}}
`),
)

// firstImexVersion returns the earliest version at which the type already carried the
// @go imex marker, walking the resource's version chain newest-first. The walk ends at
// the first version whose file drops the marker or omits the type; marker history is
// contiguous. Versions below the floor predate Core export.
func firstImexVersion(
	ctx context.Context,
	req *plugin.Request,
	livePath, name string,
	current int,
) (int, error) {
	floor := current
	if req.Versions == nil {
		return floor, nil
	}
	chain, ok := req.Versions.Chains()[livePath]
	if !ok {
		return floor, nil
	}
	for k := current - 1; k >= chain.First(); k-- {
		surf, err := req.Versions.Surface(ctx, livePath, k)
		if err != nil {
			return 0, err
		}
		def, member := surf[name]
		if !member || !domain.HasExprFromType(def.Type, "go", "imex") {
			break
		}
		floor = k
	}
	return floor, nil
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
	steps := make(set.Set[int], current-floor)
	for j := floor + 1; j <= current; j++ {
		hasStep, err := migrateStepExists(
			filepath.Join(repoRoot, versioning.VersionedPath(outputPath, j)), goName,
		)
		if err != nil {
			return nil, err
		}
		if hasStep {
			steps.Add(j)
		}
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
			if !steps.Contains(j) {
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
// per-bump lift Migrate<goName>. An unparseable file is an error: treating it as
// stepless would silently drop that bump from the ladder, so an envelope stamped below
// it would lift to the current shape without ever running the migration.
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
		path := filepath.Join(dir, e.Name())
		f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			return false, errors.Wrapf(err, "parse %s", path)
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
	Imports    []string
	Arms       []chainArm
	Runtime    string
}

var chainTemplate = template.Must(
	template.New("go-imex-chain").Parse(`// Code generated by oracle. DO NOT EDIT.

package versions

import (
	"context"

	"{{.Runtime}}"
{{- range .Imports}}
	"{{.}}"
{{- end}}
)

// Latest is the portable schema version stamped on exported {{.Type}} envelopes and the
// highest version import accepts. It equals the resource's current schema version.
const Latest = {{.CurrentPkg}}.Version

// autoDecodeEnvelope decodes a server-exported envelope as its version's {{.Type}}
// shape and lifts it through the per-version migration chain to the current shape. A
// version the ladder does not cover is rejected with a path-scoped validation error.
func autoDecodeEnvelope(ctx context.Context, env imex.Envelope) ({{.Type}}, error) {
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
