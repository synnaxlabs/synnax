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

	"github.com/synnaxlabs/oracle/exec"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/plugin/go/internal/naming"
	"github.com/synnaxlabs/oracle/plugin/gomod"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
)

// Plugin generates gorp.Codec implementations for structs annotated with @go marshal.
type Plugin struct{ Options Options }

// Options configures the go/marshal plugin.
type Options struct {
	FileNamePattern     string
	TestFileNamePattern string
	GenerateTests       bool
}

// DefaultOptions returns the default plugin options.
func DefaultOptions() Options {
	return Options{
		FileNamePattern:     "codec.gen.go",
		TestFileNamePattern: "codec_gen_test.go",
		GenerateTests:       true,
	}
}

// New creates a new go/marshal plugin with the given options.
func New(opts Options) *Plugin { return &Plugin{Options: opts} }

func (p *Plugin) Name() string                { return "go/marshal" }
func (p *Plugin) Domains() []string           { return []string{"go"} }
func (p *Plugin) Requires() []string          { return []string{"go/types", "go/pb"} }
func (p *Plugin) Check(*plugin.Request) error { return nil }

var goPostWriter = &exec.PostWriter{
	Extensions: []string{".go"},
	Commands:   [][]string{{"gofmt", "-w"}},
}

func (p *Plugin) PostWrite(files []string) error {
	return goPostWriter.PostWrite(files)
}

func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	resp := &plugin.Response{Files: make([]plugin.File, 0)}

	// Collect all entry types and their adapter status.
	type entryInfo struct {
		goName string
		goPath string
	}
	var entryTypes []entryInfo
	for _, entry := range req.Resolutions.StructTypes() {
		if !domain.HasExprFromType(entry, "go", "marshal") || !output.HasPB(entry) {
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
		entryTypes = append(entryTypes, entryInfo{goName: naming.GetGoName(entry), goPath: goPath})
	}

	// Merge all entry types' dependency trees per package.
	adapters := make(map[string]string)
	merged := make(map[string]map[string]resolution.Type)
	for _, ei := range entryTypes {
		adapters[ei.goName] = ei.goPath
		var entry resolution.Type
		for _, t := range req.Resolutions.StructTypes() {
			if naming.GetGoName(t) == ei.goName {
				entry = t
				break
			}
		}
		byPkg, _ := collectSerializableTypes(entry, req.Resolutions)
		for goPath, types := range byPkg {
			if merged[goPath] == nil {
				merged[goPath] = make(map[string]resolution.Type)
			}
			for _, t := range types {
				merged[goPath][t.QualifiedName] = t
			}
		}
	}

	// Generate one file per package.
	for goPath, typeMap := range merged {
		packageName := naming.DerivePackageName(goPath)
		entries := buildCodecEntries(typeMap, adapters, goPath)
		if len(entries) == 0 {
			continue
		}
		content, err := generateEncoderCodecFile(
			packageName, goPath, entries, req.Resolutions, req.RepoRoot,
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
		for goPath, typeMap := range merged {
			packageName := naming.DerivePackageName(goPath)
			entries := buildCodecEntries(typeMap, adapters, goPath)
			testContent, testErr := generateTestCodecFile(
				packageName, goPath, entries, req.Resolutions, req.RepoRoot,
			)
			if testErr != nil {
				return nil, errors.Wrapf(testErr, "failed to generate codec tests for %s", goPath)
			}
			if testContent != nil {
				resp.Files = append(resp.Files, plugin.File{
					Path:    fmt.Sprintf("%s/%s", goPath, p.Options.TestFileNamePattern),
					Content: testContent,
				})
			}
		}
	}

	return resp, nil
}

// CodecEntry describes a type for which a codec should be generated.
type CodecEntry struct {
	GoName  string
	Type    resolution.Type
	Adapter bool
}

func buildCodecEntries(
	typeMap map[string]resolution.Type,
	adapters map[string]string,
	goPath string,
) []CodecEntry {
	var entries []CodecEntry
	for _, t := range typeMap {
		goName := naming.GetGoName(t)
		ce := CodecEntry{GoName: goName, Type: t}
		if adapterPath, ok := adapters[goName]; ok && adapterPath == goPath {
			ce.Adapter = true
		}
		entries = append(entries, ce)
	}
	return entries
}

// GenerateCodecFile generates a complete codec file for the given entries using the
// specified package name and output path context. This is used by the migrate plugin
// to generate frozen codecs for old schema versions. Each entry gets exported
// EncodeX/DecodeX functions. Entries with Adapter=true also get an xbinary.Codec
// implementation with sync.Pool-based Writer/Reader reuse.
func GenerateCodecFile(
	packageName string,
	parentPath string,
	entries []CodecEntry,
	table *resolution.Table,
	repoRoot string,
) ([]byte, error) {
	return generateEncoderCodecFile(packageName, parentPath, entries, table, repoRoot)
}

func resolveGoImportPath(outputPath, repoRoot string) (string, error) {
	return gomod.ResolveImportPath(outputPath, repoRoot, gomod.DefaultModulePrefix), nil
}
