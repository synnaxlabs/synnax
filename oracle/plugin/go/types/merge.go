// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types

import (
	"sort"
	"strings"

	"github.com/synnaxlabs/oracle/plugin"
)

// mergeByPath appends incoming files to existing, combining any pair that
// targets the same path into a single Go file. Used to join the version alias
// re-exports and the transient declarations of a package root into one
// types.gen.go.
func mergeByPath(existing, incoming []plugin.File) []plugin.File {
	byPath := make(map[string]int, len(existing))
	for i, f := range existing {
		byPath[f.Path] = i
	}
	for _, f := range incoming {
		if i, ok := byPath[f.Path]; ok {
			existing[i].Content = mergeGeneratedGo(f.Content, existing[i].Content)
			continue
		}
		byPath[f.Path] = len(existing)
		existing = append(existing, f)
	}
	return existing
}

// mergeGeneratedGo combines two generated Go files for one package: the header
// and package clause from a, the union of both import sets, then a's body
// followed by b's.
func mergeGeneratedGo(a, b []byte) []byte {
	headA, importsA, bodyA := splitGeneratedGo(string(a))
	_, importsB, bodyB := splitGeneratedGo(string(b))
	imports := append(importsA, importsB...)
	sort.Strings(imports)
	imports = dedupe(imports)
	var out strings.Builder
	out.WriteString(headA)
	if len(imports) == 1 {
		out.WriteString("import " + imports[0] + "\n\n")
	} else if len(imports) > 1 {
		out.WriteString("import (\n")
		for _, imp := range imports {
			out.WriteString("\t" + imp + "\n")
		}
		out.WriteString(")\n\n")
	}
	out.WriteString(strings.TrimLeft(bodyA, "\n"))
	out.WriteString("\n")
	out.WriteString(strings.TrimLeft(bodyB, "\n"))
	return []byte(out.String())
}

// splitGeneratedGo separates a generated file into everything through the
// package clause, its import specs, and the remaining body.
func splitGeneratedGo(src string) (head string, imports []string, body string) {
	pkg := strings.Index(src, "\npackage ")
	end := strings.Index(src[pkg+1:], "\n") + pkg + 2
	head = src[:end] + "\n"
	rest := strings.TrimLeft(src[end:], "\n")
	switch {
	case strings.HasPrefix(rest, "import (\n"):
		end := strings.Index(rest, "\n)")
		for l := range strings.SplitSeq(rest[len("import (\n"):end], "\n") {
			if l = strings.TrimSpace(l); l != "" {
				imports = append(imports, l)
			}
		}
		body = rest[end+2:]
	case strings.HasPrefix(rest, "import "):
		nl := strings.Index(rest, "\n")
		imports = append(imports, strings.TrimSpace(rest[len("import "):nl]))
		body = rest[nl+1:]
	default:
		body = rest
	}
	return head, imports, body
}

func dedupe(in []string) []string {
	out := in[:0]
	var last string
	for _, s := range in {
		if s != last {
			out = append(out, s)
		}
		last = s
	}
	return out
}
