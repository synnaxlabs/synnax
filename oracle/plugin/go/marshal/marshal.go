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
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/synnaxlabs/oracle/exec"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/go/internal/naming"
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

func hasMarshalAnnotation(typ resolution.Type) bool {
	domain, ok := typ.Domains["go"]
	if !ok {
		return false
	}
	for _, expr := range domain.Expressions {
		if expr.Name == "marshal" {
			return true
		}
	}
	return false
}

func getGoName(s resolution.Type) string {
	if domain, ok := s.Domains["go"]; ok {
		for _, expr := range domain.Expressions {
			if expr.Name == "name" && len(expr.Values) > 0 {
				return expr.Values[0].StringValue
			}
		}
	}
	return ""
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
		if !hasMarshalAnnotation(entry) || !output.HasPB(entry) {
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
		goName := getGoName(entry)
		if goName == "" {
			goName = entry.Name
		}
		entryTypes = append(entryTypes, entryInfo{goName: goName, goPath: goPath})
	}

	// Merge all entry types' dependency trees per package.
	adapters := make(map[string]string)
	merged := make(map[string]map[string]resolution.Type)
	for _, ei := range entryTypes {
		adapters[ei.goName] = ei.goPath
		entry, _ := req.Resolutions.Get(ei.goName)
		if entry.QualifiedName == "" {
			for _, t := range req.Resolutions.StructTypes() {
				n := getGoName(t)
				if n == "" {
					n = t.Name
				}
				if n == ei.goName {
					entry = t
					break
				}
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
		var entries []CodecEntry
		for _, t := range typeMap {
			goName := getGoName(t)
			if goName == "" {
				goName = naming.ToPascalCase(t.Name)
			}
			ce := CodecEntry{GoName: goName, Type: t}
			if adapterPath, ok := adapters[goName]; ok && adapterPath == goPath {
				ce.Adapter = true
			}
			entries = append(entries, ce)
		}
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
			var entries []CodecEntry
			for _, t := range typeMap {
				goName := getGoName(t)
				if goName == "" {
					goName = naming.ToPascalCase(t.Name)
				}
				ce := CodecEntry{GoName: goName, Type: t}
				if adapterPath, ok := adapters[goName]; ok && adapterPath == goPath {
					ce.Adapter = true
				}
				entries = append(entries, ce)
			}
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

func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

func resolveGoImportPath(outputPath, repoRoot string) (string, error) {
	if repoRoot == "" {
		return "github.com/synnaxlabs/synnax/" + outputPath, nil
	}
	absPath := filepath.Join(repoRoot, outputPath)
	dir := absPath
	for {
		modPath := filepath.Join(dir, "go.mod")
		if fileExists(modPath) {
			moduleName, err := parseModuleName(modPath)
			if err != nil {
				return "", errors.Wrapf(err, "failed to parse go.mod at %s", modPath)
			}
			relPath, err := filepath.Rel(dir, absPath)
			if err != nil {
				return "", errors.Wrapf(err, "failed to compute relative path")
			}
			if relPath == "." {
				return moduleName, nil
			}
			return moduleName + "/" + filepath.ToSlash(relPath), nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "github.com/synnaxlabs/synnax/" + outputPath, nil
}

func parseModuleName(modPath string) (string, error) {
	file, err := os.Open(modPath)
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return parts[1], nil
			}
		}
	}
	return "", errors.New("module name not found in go.mod")
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
