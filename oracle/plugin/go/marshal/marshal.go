// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package marshal provides an Oracle plugin that generates gorp.Codec implementations
// using direct binary encoding for zero-allocation serialization.
package marshal

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/plugin/go/internal/naming"
	"github.com/synnaxlabs/oracle/plugin/go/internal/versioning"
	gotypes "github.com/synnaxlabs/oracle/plugin/go/types"
	"github.com/synnaxlabs/oracle/plugin/gomod"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

// Plugin generates gorp.Codec implementations for structs annotated with @go marshal.
type Plugin struct{ Options Options }

// Options configures the go/marshal plugin.
type Options struct {
	FileNamePattern     string
	TestFileNamePattern string
	GenerateTests       bool
	// RequireVersioned rejects codecs targeting a package outside versions/vN so
	// persisted wire formats stay pinned to an immutable shape. Disabled by unit tests
	// that exercise codec mechanics on ad-hoc unversioned schemas.
	RequireVersioned bool
}

// DefaultOptions returns the default plugin options.
func DefaultOptions() Options {
	return Options{
		FileNamePattern:     "codec.gen.go",
		TestFileNamePattern: "codec_gen_test.go",
		GenerateTests:       true,
		RequireVersioned:    true,
	}
}

// New creates a new go/marshal plugin with the given options.
func New(opts Options) *Plugin { return &Plugin{Options: opts} }

func (p *Plugin) Name() string                { return "go/marshal" }
func (p *Plugin) Domains() []string           { return []string{"go"} }
func (p *Plugin) Requires() []string          { return []string{"go/types"} }
func (p *Plugin) Check(*plugin.Request) error { return nil }

func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	resp := &plugin.Response{Files: make([]plugin.File, 0)}

	// Types that alias their predecessor version carry its codec methods
	// through the alias; only defined types get codecs in the current package.
	aliased, err := gotypes.AliasedTypes(req)
	if err != nil {
		return nil, err
	}

	// Version-laid-out packages emit their codecs alongside the current
	// types in types/vN; the rewrite shifts every affected path at once so
	// cross-package codec references stay version-pinned.
	rewritten, _, err := versioning.RewriteCurrent(req.Resolutions)
	if err != nil {
		return nil, err
	}
	versionedReq := *req
	versionedReq.Resolutions = rewritten
	req = &versionedReq

	// Collect all entry types.
	var entryTypes []resolution.Type
	for _, entry := range req.Resolutions.StructTypes() {
		if !domain.HasExprFromType(entry, "go", "marshal") {
			continue
		}
		goPath := output.GetPath(entry, "go")
		if goPath == "" {
			continue
		}
		if req.RepoRoot != "" {
			if err := req.ValidateOutputPath(goPath); err != nil {
				return nil, errors.Wrapf(err, "invalid output path for %s", entry.Name)
			}
		}
		entryTypes = append(entryTypes, entry)
	}

	// Collect DistinctForm types with @go marshal flex.
	flexByPkg := make(map[string][]FlexCodec)
	for _, dt := range req.Resolutions.DistinctTypes() {
		if aliased.Contains(dt.QualifiedName) {
			continue
		}
		marshalVal := domain.GetStringFromType(dt, "go", "marshal")
		if marshalVal != "flex" {
			continue
		}
		goPath := output.GetPath(dt, "go")
		if goPath == "" {
			continue
		}
		form := dt.Form.(resolution.DistinctForm)
		goName := naming.GetGoName(dt)
		flexByPkg[goPath] = append(flexByPkg[goPath], FlexCodec{
			GoName:   goName,
			Receiver: ReceiverName(goName),
			BaseType: form.Base.Name,
		})
	}

	// Merge all entry types' dependency trees per package.
	merged := make(map[string]map[string]resolution.Type)
	for _, entry := range entryTypes {
		byPkg, _ := collectSerializableTypes(entry, req.Resolutions)
		for goPath, types := range byPkg {
			if merged[goPath] == nil {
				merged[goPath] = make(map[string]resolution.Type)
			}
			for _, t := range types {
				if aliased.Contains(t.QualifiedName) {
					continue
				}
				merged[goPath][t.QualifiedName] = t
			}
		}
	}

	// Collect all packages that need a codec file (from structs or flex types).
	allPkgs := make(set.Set[string])
	for goPath := range merged {
		allPkgs.Add(goPath)
	}
	for goPath := range flexByPkg {
		allPkgs.Add(goPath)
	}

	// A codec pins a persisted wire format to a type shape, so every marshalled type
	// must live in a versions/vN package where that shape is immutable. A codec target
	// outside versions/vN means the type (or one it persists) is missing @go version.
	if p.Options.RequireVersioned {
		for goPath := range allPkgs {
			if isVersionedPath(goPath) {
				continue
			}
			names := make([]string, 0, len(merged[goPath])+len(flexByPkg[goPath]))
			for _, t := range merged[goPath] {
				names = append(names, naming.GetGoName(t))
			}
			for _, f := range flexByPkg[goPath] {
				names = append(names, f.GoName)
			}
			sort.Strings(names)
			return nil, errors.Newf(
				"cannot generate a codec for %s in %s: @go marshal types must be versioned; add @go version to them",
				strings.Join(names, ", "),
				goPath,
			)
		}
	}

	// Generate one file per package in sorted order for deterministic output.
	sortedPkgs := allPkgs.Slice()
	sort.Strings(sortedPkgs)
	for _, goPath := range sortedPkgs {
		packageName := naming.DerivePackageName(goPath)
		entries := buildCodecEntries(merged[goPath])
		flex := flexByPkg[goPath]
		if len(entries) == 0 && len(flex) == 0 {
			continue
		}
		content, err := generateEncoderCodecFile(
			packageName, goPath, entries, flex, req.Resolutions, req.RepoRoot,
		)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate codec for %s", goPath)
		}
		resp.Files = append(resp.Files, plugin.File{
			Path:    fmt.Sprintf("%s/%s", goPath, p.Options.FileNamePattern),
			Content: content,
		})
	}

	if p.Options.GenerateTests {
		sortedMergedPkgs := make([]string, 0, len(merged))
		for goPath := range merged {
			sortedMergedPkgs = append(sortedMergedPkgs, goPath)
		}
		sort.Strings(sortedMergedPkgs)
		for _, goPath := range sortedMergedPkgs {
			typeMap := merged[goPath]
			packageName := naming.DerivePackageName(goPath)
			entries := buildCodecEntries(typeMap)
			testContent, testErr := generateTestCodecFile(
				packageName, goPath, entries, req.Resolutions, req.RepoRoot,
			)
			if testErr != nil {
				return nil, errors.Wrapf(
					testErr,
					"failed to generate codec tests for %s",
					goPath,
				)
			}
			if testContent != nil {
				resp.Files = append(resp.Files, plugin.File{
					Path: fmt.Sprintf(
						"%s/%s",
						goPath,
						p.Options.TestFileNamePattern,
					),
					Content: testContent,
				})
			}
		}
	}

	return resp, nil
}

// CodecEntry describes a type for which a codec should be generated.
type CodecEntry struct {
	GoName string
	Type   resolution.Type
}

func buildCodecEntries(
	typeMap map[string]resolution.Type,
) []CodecEntry {
	keys := make([]string, 0, len(typeMap))
	for k := range typeMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	entries := make([]CodecEntry, 0, len(keys))
	for _, k := range keys {
		t := typeMap[k]
		goName := naming.GetGoName(t)
		entries = append(entries, CodecEntry{GoName: goName, Type: t})
	}
	return entries
}

type importEntry struct {
	Path  string
	Alias string
}

// versionDir matches version sub-directory names ("v0", "v12").
var versionDir = regexp.MustCompile(`/v\d+$`)

func sortedImports(m map[string]string) []importEntry {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	entries := make([]importEntry, 0, len(keys))
	for _, k := range keys {
		alias := m[k]
		// Version directories always import under an explicit alias so the
		// qualifier's origin stays visible.
		if alias == "" && versionDir.MatchString(k) {
			alias = filepath.Base(k)
		}
		entries = append(entries, importEntry{Path: k, Alias: alias})
	}
	return entries
}

// GenerateCodecFile generates a complete codec file for the given entries using the
// specified package name and output path context. This is used by the migrate plugin to
// generate frozen codecs for old schema versions. Each entry gets EncodeOrc/DecodeOrc
// methods that implement the orc.SelfEncoder and orc.SelfDecoder interfaces.
func GenerateCodecFile(
	packageName string,
	parentPath string,
	entries []CodecEntry,
	flex []FlexCodec,
	table *resolution.Table,
	repoRoot string,
) ([]byte, error) {
	return generateEncoderCodecFile(
		packageName,
		parentPath,
		entries,
		flex,
		table,
		repoRoot,
	)
}

func resolveGoImportPath(outputPath, repoRoot string) (string, error) {
	return gomod.ResolveImportPath(outputPath, repoRoot, gomod.DefaultModulePrefix), nil
}

// isVersionedPath reports whether goPath is a versions/vN package (its parent directory
// is "versions"), the only place a codec may be generated.
func isVersionedPath(goPath string) bool {
	return filepath.Base(filepath.Dir(goPath)) == "versions"
}
