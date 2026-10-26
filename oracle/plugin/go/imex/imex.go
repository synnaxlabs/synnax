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
// constant equal to K; the versions package root gains Latest, an autoDecodeEnvelope
// ladder that lifts server-era envelopes through the per-version Migrate<Type> steps,
// and liftFirst, which lifts the first exported shape through the same steps; the
// service package gains the Service.Export half of imex.ImportExporter. Latest keeps
// the wire envelope version and the storage schema version a single sequence per
// resource, and the ladder extends itself on every version bump. Import stays
// hand-written: every resource parents and persists differently, and a decoder of
// shapes older than the first export builds the first exported shape and calls
// liftFirst, so a version bump never edits it. A type another resource exports declares
// `@go imex body` and gets an exported <Type>Ladder over body blobs instead of the
// envelope ladder, liftFirst, and Export.
package imex

import (
	"bytes"
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"text/template"

	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/plugin/go/internal/naming"
	"github.com/synnaxlabs/oracle/plugin/go/internal/versioning"
	"github.com/synnaxlabs/oracle/plugin/gomod"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/oracle/versions"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

// Options configures the imex plugin output.
type Options struct {
	// FileNamePattern is the basename written for each output package.
	FileNamePattern string
	// RuntimeImportPath is the Go package supplying the Version type, Envelope, Ladder,
	// DecodeBlob, and NewErrUnsupportedVersion the generated files call. Its package
	// name must be imex — the templates qualify every call with that identifier.
	// Required.
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

// Generate emits imex machinery for every output package whose types declare @go imex:
// a Version constant in each versions/vK package the Core has exported, and Latest in
// the versions package root. Constants below the earliest exported version are deleted.
// A bare marker adds the autoDecodeEnvelope ladder, liftFirst, and Service.Export;
// `@go imex body` adds an exported <Type>Ladder per type instead and deletes any
// Export. It errors when a marked type has no version chain, when a bare marker shares
// its package with another marked type, when body types in one package come from
// different chains, when a version redeclares a type without its Migrate<Type> step, or
// when a version package does not parse.
//
// The emitted Export calls the service's NewRetrieve, MatchKeys, and Type, and parses
// id.Key as a UUID. A resource missing any of them, or keyed by something else, fails
// to compile.
func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	// The templates qualify against these import paths; emitting without them
	// would produce files that do not compile.
	if p.options.RuntimeImportPath == "" {
		return nil, errors.New("go/imex requires Options.RuntimeImportPath")
	}
	if p.options.OntologyImportPath == "" {
		return nil, errors.New("go/imex requires Options.OntologyImportPath")
	}
	ctx := context.Background()
	var (
		paths    []string
		packages = make(map[string][]target)
	)
	for _, typ := range req.Resolutions.TypesWithDomain("go") {
		if !domain.HasExprFromType(typ, "go", "imex") {
			continue
		}
		outputPath := output.GetPath(typ, "go")
		if outputPath == "" {
			continue
		}
		t, err := newTarget(ctx, req, typ)
		if err != nil {
			return nil, err
		}
		if prev := packages[outputPath]; len(prev) > 0 {
			if !prev[0].body || !t.body {
				return nil, errors.Newf(
					"duplicate @go imex declarations for %s: %s and %s",
					outputPath, prev[0].typ.QualifiedName, typ.QualifiedName,
				)
			}
			if prev[0].livePath != t.livePath {
				return nil, errors.Newf(
					"@go imex body types for %s span version chains %s and %s",
					outputPath, prev[0].livePath, t.livePath,
				)
			}
		} else {
			paths = append(paths, outputPath)
		}
		packages[outputPath] = append(packages[outputPath], t)
	}
	resp := &plugin.Response{}
	for _, outputPath := range paths {
		if err := p.generatePackage(
			req, resp, outputPath, packages[outputPath],
		); err != nil {
			return nil, err
		}
	}
	return resp, nil
}

// target is one type that declares @go imex.
type target struct {
	typ    resolution.Type
	goName string
	// body is true for `@go imex body`: another resource exports the type.
	body     bool
	livePath string
	current  int
	floor    int
	// exported holds the versions the Core has exported the type at, oldest first.
	exported []int
	// steps holds the exported versions whose package declares Migrate<Type>.
	steps set.Set[int]
}

func newTarget(
	ctx context.Context,
	req *plugin.Request,
	typ resolution.Type,
) (target, error) {
	livePath := strings.TrimSuffix(typ.FilePath, ".oracle")
	var chains map[string]versions.Chain
	if req.Versions != nil {
		chains = req.Versions.Chains()
	}
	chain, ok := chains[livePath]
	if !ok {
		return target{}, errors.Newf(
			"%s declares @go imex without a version chain", typ.QualifiedName,
		)
	}
	if ended, err := req.Versions.Ended(ctx, livePath); err != nil {
		return target{}, err
	} else if ended {
		return target{}, errors.Newf(
			"%s declares @go imex, but its version chain ended", typ.QualifiedName,
		)
	}
	current := chain.Current()
	floor, err := firstImexVersion(ctx, req, livePath, typ.Name, current)
	if err != nil {
		return target{}, err
	}
	exported := exportedVersions(req, livePath, floor, current)
	steps, err := migrateSteps(ctx, req, typ, livePath, exported)
	if err != nil {
		return target{}, err
	}
	return target{
		typ:      typ,
		goName:   naming.GetGoName(typ),
		body:     domain.GetStringFromType(typ, "go", "imex") == bodyMarker,
		livePath: livePath,
		current:  current,
		floor:    floor,
		exported: exported,
		steps:    steps,
	}, nil
}

// bodyMarker is the @go imex argument that marks a type another resource exports.
const bodyMarker = "body"

func (p *Plugin) generatePackage(
	req *plugin.Request,
	resp *plugin.Response,
	outputPath string,
	targets []target,
) error {
	var (
		exported = make(set.Set[int])
		names    = make([]string, 0, len(targets))
		floor    = targets[0].floor
	)
	for _, t := range targets {
		exported.Add(t.exported...)
		names = append(names, t.goName)
		floor = min(floor, t.floor)
	}
	subject := "the shapes"
	if len(names) == 1 {
		subject = "the " + names[0] + " shape"
	}
	versionNums := slices.Sorted(maps.Keys(exported))
	for _, k := range versionNums {
		path := versioning.VersionedPath(outputPath, k)
		var buf bytes.Buffer
		if err := versionTemplate.Execute(&buf, &versionData{
			Package: naming.DerivePackageName(path),
			Subject: subject,
			Version: k,
			Runtime: p.options.RuntimeImportPath,
		}); err != nil {
			return err
		}
		resp.Files = append(resp.Files, plugin.File{
			Path:    path + "/" + p.options.FileNamePattern,
			Content: buf.Bytes(),
		})
	}
	stale, err := versioning.VersionDirs(req.RepoRoot, outputPath)
	if err != nil {
		return err
	}
	for _, k := range stale {
		if k < floor {
			resp.Deletions = append(
				resp.Deletions,
				versioning.VersionedPath(outputPath, k)+"/"+p.options.FileNamePattern,
			)
		}
	}
	imports := make([]string, 0, len(versionNums))
	for _, k := range versionNums {
		imports = append(imports, gomod.ResolveImportPath(
			versioning.VersionedPath(outputPath, k),
			req.RepoRoot,
			gomod.DefaultModulePrefix,
		))
	}
	var buf bytes.Buffer
	if targets[0].body {
		err = p.executeBody(&buf, targets, imports)
	} else {
		err = p.executeChain(&buf, targets[0], imports)
	}
	if err != nil {
		return err
	}
	resp.Files = append(resp.Files, plugin.File{
		Path:    outputPath + "/versions/" + p.options.FileNamePattern,
		Content: buf.Bytes(),
	})
	if targets[0].body {
		resp.Deletions = append(
			resp.Deletions, outputPath+"/"+p.options.FileNamePattern,
		)
		return nil
	}
	// A fresh buffer: plugin.File.Content aliases the backing array, so reusing buf
	// would overwrite the chain file appended above.
	var exportBuf bytes.Buffer
	if err := exportTemplate.Execute(&exportBuf, &exportData{
		Package: naming.DerivePackageName(outputPath),
		Type:    targets[0].goName,
		Versions: gomod.ResolveImportPath(
			outputPath+"/versions", req.RepoRoot, gomod.DefaultModulePrefix,
		),
		Runtime:  p.options.RuntimeImportPath,
		Ontology: p.options.OntologyImportPath,
	}); err != nil {
		return err
	}
	resp.Files = append(resp.Files, plugin.File{
		Path:    outputPath + "/" + p.options.FileNamePattern,
		Content: exportBuf.Bytes(),
	})
	return nil
}

func (p *Plugin) executeChain(buf *bytes.Buffer, t target, imports []string) error {
	return chainTemplate.Execute(buf, &chainData{
		Type:       t.goName,
		CurrentPkg: versioning.Dir(t.current),
		Arms:       chainArms(t, envelopeDecode),
		First:      firstLift(t),
		Imports:    imports,
		Runtime:    p.options.RuntimeImportPath,
	})
}

func (p *Plugin) executeBody(
	buf *bytes.Buffer,
	targets []target,
	imports []string,
) error {
	data := &bodyData{
		CurrentPkg: versioning.Dir(targets[0].current),
		Imports:    imports,
		Runtime:    p.options.RuntimeImportPath,
	}
	for _, t := range targets {
		data.Ladders = append(data.Ladders, ladderData{
			Type:     t.goName,
			Name:     t.typ.QualifiedName,
			FirstPkg: versioning.Dir(t.floor),
			Arms:     chainArms(t, blobDecode(t.typ.QualifiedName)),
		})
	}
	return bodyTemplate.Execute(buf, data)
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
	"uuid"

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
	if err = env.Encode(v); err != nil {
		return imex.Envelope{}, err
	}
	return env, nil
}
`),
)

type versionData struct {
	Package string
	// Subject names the shapes the package defines, e.g. "the Log shape".
	Subject string
	Version int
	Runtime string
}

var versionTemplate = template.Must(
	template.New("go-imex-version").Parse(`// Code generated by oracle. DO NOT EDIT.

package {{.Package}}

import "{{.Runtime}}"

// Version is the portable schema version of {{.Subject}} this package defines.
const Version imex.Version = {{.Version}}
`),
)

// firstImexVersion returns the earliest version at which the type  carried the @go imex
// marker, walking the resource's version chain newest-first. The walk ends at the first
// version whose file drops the marker or omits the type; marker history is contiguous.
// Versions below the floor predate Core export.
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
	for _, k := range slices.Backward(chain.Numbers) {

		if k >= current {
			continue
		}
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

// exportedVersions returns the chain's declared versions in [floor, current]: the
// versions the Core has exported. Falls back to the bare current version when the
// resource has no chain.
func exportedVersions(req *plugin.Request, livePath string, floor, current int) []int {
	if req.Versions == nil {
		return []int{current}
	}
	chain, ok := req.Versions.Chains()[livePath]
	if !ok {
		return []int{current}
	}
	out := make([]int, 0, len(chain.Numbers))
	for _, k := range chain.Numbers {
		if k >= floor && k <= current {
			out = append(out, k)
		}
	}
	if len(out) == 0 {
		return []int{current}
	}
	return out
}

// decodeExpr returns the expression that decodes the stamped body as pkg.goName.
type decodeExpr func(pkg, goName string) string

// envelopeDecode decodes a resource's envelope.
func envelopeDecode(pkg, goName string) string {
	return fmt.Sprintf("env.Decode[%s.%s](ctx)", pkg, goName)
}

// blobDecode decodes a body blob, naming it name in errors.
func blobDecode(name string) decodeExpr {
	return func(pkg, goName string) string {
		return fmt.Sprintf(
			"imex.DecodeBlob[%s.%s](data, %q, version)",
			pkg,
			goName,
			name,
		)
	}
}

// chainArms builds one ladder arm per exported version of t below current: each decodes
// the stamped vK shape and lifts it through every later bump's Migrate<Type> step.
// Alias-only bumps pass the value through unchanged.
func chainArms(t target, decode decodeExpr) []chainArm {
	nums, goName := t.exported, t.goName
	arms := make([]chainArm, 0, len(nums)-1)
	for i, k := range nums[:len(nums)-1] {
		var b strings.Builder
		cur := fmt.Sprintf("t%d", k)
		fmt.Fprintf(&b, "\t\t%s, err := %s\n", cur, decode(versioning.Dir(k), goName))
		fmt.Fprintf(&b, "\t\tif err != nil {\n\t\t\treturn %s{}, err\n\t\t}\n", goName)
		cur = writeSteps(&b, t, i, cur, "\t\t")
		fmt.Fprintf(&b, "\t\treturn %s, nil", cur)
		arms = append(arms, chainArm{Pkg: versioning.Dir(k), Body: b.String()})
	}
	return arms
}

// firstLift builds liftFirst, which lifts the first exported shape through every later
// bump's Migrate<Type> step.
func firstLift(t target) liftData {
	var b strings.Builder
	in := fmt.Sprintf("t%d", t.floor)
	out := writeSteps(&b, t, 0, in, "\t")
	fmt.Fprintf(&b, "\treturn %s, nil", out)
	return liftData{
		Param: fmt.Sprintf("%s %s.%s", in, versioning.Dir(t.floor), t.goName),
		Body:  b.String(),
	}
}

// writeSteps writes, at indent, the Migrate<Type> calls that lift cur, the value at
// t.exported[i], to the current shape. It returns the variable holding the result.
func writeSteps(b *strings.Builder, t target, i int, cur, indent string) string {
	for _, j := range t.exported[i+1:] {
		if !t.steps.Contains(j) {
			continue
		}
		next := fmt.Sprintf("t%d", j)
		fmt.Fprintf(
			b, "%s%s, err := %s.Migrate%s(ctx, %s)\n",
			indent, next, versioning.Dir(j), t.goName, cur,
		)
		fmt.Fprintf(
			b, "%sif err != nil {\n%s\treturn %s{}, err\n%s}\n",
			indent, indent, t.goName, indent,
		)
		cur = next
	}
	return cur
}

// migrateSteps returns the versions in exported, above the first, whose package
// declares Migrate<Type>. It errors when a version redeclares typ without that step,
// since the ladder would pass the old shape through unchanged. A version that only
// aliases typ keeps its shape and needs no step.
func migrateSteps(
	ctx context.Context,
	req *plugin.Request,
	typ resolution.Type,
	livePath string,
	exported []int,
) (set.Set[int], error) {
	var (
		goName     = naming.GetGoName(typ)
		outputPath = output.GetPath(typ, "go")
		steps      = make(set.Set[int])
	)
	for _, k := range exported[1:] {
		dir := versioning.VersionedPath(outputPath, k)
		hasStep, err := migrateStepExists(filepath.Join(req.RepoRoot, dir), goName)
		if err != nil {
			return nil, err
		}
		if hasStep {
			steps.Add(k)
			continue
		}
		surf, err := req.Versions.Surface(ctx, livePath, k)
		if err != nil {
			return nil, err
		}
		if def, ok := surf[typ.Name]; ok && def.Version == k {
			return nil, errors.Newf(
				"%s redeclares %s without the Migrate%s step its import ladder needs",
				dir, typ.QualifiedName, goName,
			)
		}
	}
	return steps, nil
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

type liftData struct {
	Param string
	Body  string
}

type chainData struct {
	Type       string
	CurrentPkg string
	Imports    []string
	Arms       []chainArm
	First      liftData
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
		return env.Decode[{{.Type}}](ctx)
	}
	return {{.Type}}{}, imex.NewErrUnsupportedVersion(env.Type, env.Version, Latest)
}

// liftFirst lifts the first exported {{.Type}} shape to the current shape. A decoder
// of an older shape builds the first exported shape and enters the ladder here.
func liftFirst(ctx context.Context, {{.First.Param}}) ({{.Type}}, error) {
{{.First.Body}}
}
`),
)

type ladderData struct {
	Type     string
	Name     string
	FirstPkg string
	Arms     []chainArm
}

type bodyData struct {
	CurrentPkg string
	Imports    []string
	Ladders    []ladderData
	Runtime    string
}

var bodyTemplate = template.Must(
	template.New("go-imex-body").Parse(`// Code generated by oracle. DO NOT EDIT.

package versions

import (
	"context"

	"{{.Runtime}}"
	"github.com/synnaxlabs/x/encoding/msgpack"
{{- range .Imports}}
	"{{.}}"
{{- end}}
)

// Latest is the portable schema version stamped on exported bodies of every type in
// this package and the highest version import accepts. It equals the current schema
// version.
const Latest = {{.CurrentPkg}}.Version
{{range .Ladders}}
// {{.Type}}Ladder decodes {{.Type}} bodies stamped at any exported version and
// migrates them to the current shape.
var {{.Type}}Ladder = imex.Ladder[{{.Type}}]{
	First:  {{.FirstPkg}}.Version,
	Latest: Latest,
	Decode: decode{{.Type}},
}

func decode{{.Type}}(
	ctx context.Context,
	version imex.Version,
	data msgpack.EncodedJSON,
) ({{.Type}}, error) {
	switch version {
{{- range .Arms}}
	case {{.Pkg}}.Version:
{{.Body}}
{{- end}}
	case {{$.CurrentPkg}}.Version:
		return imex.DecodeBlob[{{.Type}}](data, "{{.Name}}", version)
	}
	return {{.Type}}{}, imex.NewErrUnsupportedVersion("{{.Name}}", version, Latest)
}
{{end -}}
`),
)
