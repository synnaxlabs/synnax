// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package imex generates the portable import/export version constants for any Oracle
// struct that declares the bare `@go imex` marker. Every versions/vK package gains a
// Version constant equal to K, and the versions package root gains Latest, aliasing
// the current version's constant. Latest keeps the wire envelope version and the
// storage schema version a single sequence per resource.
package imex

import (
	"bytes"
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

// goModulePrefix resolves repo-relative output paths to Go import paths when no
// go.mod is discoverable, mirroring the go/types plugin.
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

// Generate emits imex version constants into the versions tree of every output
// package whose type declares @go imex: one Version constant per versions/vK
// package (the version directories on disk, plus the current version), and a
// Latest constant in the versions package root. It errors when a marked type
// lacks a @go version or when two types at the same path both carry the marker.
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
		current := versioning.VersionedPath(outputPath, version)
		var buf bytes.Buffer
		if err := latestTemplate.Execute(&buf, &latestData{
			Type:       goName,
			ImportPath: gomod.ResolveImportPath(current, req.RepoRoot, goModulePrefix),
			Package:    naming.DerivePackageName(current),
		}); err != nil {
			return nil, err
		}
		resp.Files = append(resp.Files, plugin.File{
			Path:    outputPath + "/versions/" + p.options.FileNamePattern,
			Content: buf.Bytes(),
		})
		// The constants used to be emitted at the package root; clean the old
		// location up.
		resp.Deletions = append(
			resp.Deletions, outputPath+"/"+p.options.FileNamePattern,
		)
	}
	return resp, nil
}

// versionDirs returns the numeric version sub-directories present under the
// output package's versions/ tree on disk, ascending.
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

var versionTemplate = template.Must(template.New("go-imex-version").Parse(`// Code generated by oracle. DO NOT EDIT.

package {{.Package}}

import "github.com/synnaxlabs/synnax/pkg/service/imex"

// Version is the portable schema version of the {{.Type}} shape this package defines.
const Version imex.Version = {{.Version}}
`))

type latestData struct {
	Type       string
	ImportPath string
	Package    string
}

var latestTemplate = template.Must(template.New("go-imex-latest").Parse(`// Code generated by oracle. DO NOT EDIT.

package versions

import "{{.ImportPath}}"

// Latest is the portable schema version stamped on exported {{.Type}} envelopes and
// the highest version import accepts. It equals the resource's current schema version.
const Latest = {{.Package}}.Version
`))
