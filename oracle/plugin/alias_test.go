// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package plugin_test

import (
	"maps"
	"regexp"
	"slices"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/plugin"
	cppjson "github.com/synnaxlabs/oracle/plugin/cpp/json"
	cpppb "github.com/synnaxlabs/oracle/plugin/cpp/pb"
	cpptypes "github.com/synnaxlabs/oracle/plugin/cpp/types"
	"github.com/synnaxlabs/oracle/plugin/go/marshal"
	gopb "github.com/synnaxlabs/oracle/plugin/go/pb"
	gotypes "github.com/synnaxlabs/oracle/plugin/go/types"
	pbtypes "github.com/synnaxlabs/oracle/plugin/pb/types"
	pytypes "github.com/synnaxlabs/oracle/plugin/py/types"
	tstypes "github.com/synnaxlabs/oracle/plugin/ts/types"
	. "github.com/synnaxlabs/oracle/testutil"
)

const aliasBase = `
@go output "core/pkg/service/base"
@ts output "client/ts/src/base"
@py output "client/py/synnax/base"
@cpp output "client/cpp/base"
@pb

Key = uuid

Size enum {
	small = "small"
	large = "large"
}

Level enum {
	low = 0
	high = 1
}

Method enum {
	password = "password"
}

Labeled struct {
	label  string
	hidden bool
	@go marshal
}

Valve struct extends Labeled {
	open bool
	@go marshal
}

Credentials struct {
	username string
	@go marshal
}

Code uint32

ValveList Valve[]

Box struct<T> {
	item T
	@go marshal
}

Bin struct<U?> {
	item U?
	@go marshal
}

Edge union on variant {
	pipe Labeled
	@go marshal
}

Signal union on kind {
	none {}
	level {
		value float64
	}
	@go marshal
}
`

const aliasHead = `
import "schemas/base"

@go output "core/pkg/service/test"
@ts output "client/ts/src/test"
@py output "client/py/synnax/test"
@cpp output "client/cpp/test"
@pb
`

// aliasTargets maps each alias to the base type it stands for. Gate is an alias of an
// alias.
var aliasTargets = map[string]string{
	"Key":         "Key",
	"Size":        "Size",
	"Level":       "Level",
	"Method":      "Method",
	"Labeled":     "Labeled",
	"Valve":       "Valve",
	"Gate":        "Valve",
	"Credentials": "Credentials",
	"Code":        "Code",
	"ValveList":   "ValveList",
	"Edge":        "Edge",
	"Signal":      "Signal",
	"Box":         "Box",
	"Crate":       "Box",
	"Bin":         "Bin",
	"Tub":         "Bin",
}

const aliasDecls = `
Key = base.Key
Size = base.Size
Level = base.Level
Method = base.Method
Labeled = base.Labeled
Valve = base.Valve
Gate = Valve
Credentials = base.Credentials
Code = base.Code
ValveList = base.ValveList
Edge = base.Edge
Signal = base.Signal
Box<T> = base.Box<T>
Crate<T> = Box<T>
Bin<U?> = base.Bin<U>
Tub<U?> = Bin<U>
`

// aliasUses puts the aliases in each position where a generator decides what to emit
// from the kind of a type.
const aliasUses = `
Panel struct {
	size    Size = large
	level   Level
	method  Method?
	sizes   Size[]
	valve   Valve
	spare   Valve?
	door    Gate
	doors   Gate[]
	valves  Valve[]
	labeled Labeled = { label = "x" }
	edge    Edge
	edges   Edge[]
	backup  Edge?
	code    Code
	codes   Code[]
	limit   Code = 5
	signal  Signal = none
	key     Key
	keys    Key[]
	cap     Code?
	list    ValveList
	gates   Gates
	byName  map<string, Edge>
	pulse   Signal = { kind = "level", value = 1 }
	hatch   Gate?
	box     Box<string>
	crates  Box<Gate>[]
	crate   Crate<string>
	byGate  map<string, Gate>
	byValve map<string, Valve>?
	byKey   map<Key, Valve>
	grid    Gates[]
	lists   ValveList[]
	bin     Bin
	tub     Tub
	@go marshal
}

Plain = Tub

Tray struct<T> {
	item T
	box  Box<T>
	@go marshal
}

Crib struct<U?> {
	item U?
	bin  Bin<U>
	@go marshal
}

Gates Gate[]

Child struct extends Labeled {
	hidden bool?
	width  float64
}

Grandchild struct extends Gate {
	open  bool?
	depth float64
}

Node union on variant {
	valve Valve
	door  Gate
}

Shape union on kind extends Labeled {
	circle Credentials
}

Proof union on method Method {
	password Credentials
}
`

var (
	aliasNames = slices.Sorted(maps.Keys(aliasTargets))
	names      = strings.Join(aliasNames, "|")
	schemas    = strings.Join(lo.Map(aliasNames, func(n string, _ int) string {
		return schemaName(n)
	}), "|")
	aliasRef = regexp.MustCompile(`\b(` + names + `)\b`)
	// aliasDecl matches the whole declaration of an alias in Go, TypeScript, Python,
	// and C++, including the lines a generic declaration spans.
	aliasDecl = regexp.MustCompile(`(?m)^\s*(` + strings.Join([]string{
		`type (` + names + `)(\[[^\]]*\])? = .*$`,
		`export (const|type) (` + names + `|` + schemas + `)\b[^;]*;`,
		`(` + names + `): TypeAlias = .*$`,
		`(template <[^>]*>\s*)?using (` + names + `) = [^;]*;`,
	}, "|") + `)`)
	// qualifiedName matches a reference to a base or test type through its package.
	qualifiedName = regexp.MustCompile(
		`(\b(base|test)\.|::synnax::base::)(` + names + `|` + schemas + `)\b`,
	)
	// renamed maps each alias named unlike its target to that target, as a type and as
	// a TypeScript schema.
	renamed      = renamedAliases()
	renamedAlias = regexp.MustCompile(
		`\b(` + strings.Join(slices.Sorted(maps.Keys(renamed)), "|") + `)\b`,
	)
	// cppStringEnum matches a C++ string enum, which a direct reference spells as
	// std::string and an alias spells by name.
	cppStringEnum = regexp.MustCompile(`(^|[^:\w])(Size|Method)\b`)
)

func renamedAliases() map[string]string {
	out := make(map[string]string)
	for alias, target := range aliasTargets {
		if alias != target {
			out[alias] = target
			out[schemaName(alias)] = schemaName(target)
		}
	}
	return out
}

func schemaName(typeName string) string {
	return strings.ToLower(typeName[:1]) + typeName[1:] + "Z"
}

// directUses returns aliasUses with each alias replaced by its base type.
func directUses() string {
	return aliasRef.ReplaceAllStringFunc(aliasUses, func(name string) string {
		return "base." + aliasTargets[name]
	})
}

// normalizeLines returns the sorted, non-blank lines of generated code with alias
// declarations dropped, every type spelled as the base type it stands for, and the
// names of each Python import sorted.
func normalizeLines(path, content string) []string {
	var lines []string
	content = aliasDecl.ReplaceAllString(content, "")
	for line := range strings.SplitSeq(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		line = qualifiedName.ReplaceAllString(line, "$3")
		line = renamedAlias.ReplaceAllStringFunc(line, func(name string) string {
			return renamed[name]
		})
		if strings.HasSuffix(path, ".h") {
			line = cppStringEnum.ReplaceAllString(line, "${1}std::string")
		}
		if from, imported, ok := strings.Cut(line, " import "); ok &&
			strings.HasPrefix(from, "from ") {
			names := strings.Split(imported, ", ")
			slices.Sort(names)
			line = from + " import " + strings.Join(names, ", ")
		}
		lines = append(lines, line)
	}
	slices.Sort(lines)
	return lines
}

// lineDiff returns the lines only a holds and the lines only b holds, counting repeats.
// Both inputs must be sorted.
func lineDiff(a, b []string) (onlyA, onlyB []string) {
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		switch strings.Compare(a[i], b[j]) {
		case -1:
			onlyA = append(onlyA, a[i])
			i++
		case 1:
			onlyB = append(onlyB, b[j])
			j++
		default:
			i++
			j++
		}
	}
	return append(onlyA, a[i:]...), append(onlyB, b[j:]...)
}

func generateFiles(ctx SpecContext, source string, p plugin.Plugin) map[string]string {
	GinkgoHelper()
	loader := NewMockFileLoader().Add("schemas/base", aliasBase)
	resp := MustGenerate(ctx, source, "test", loader, p)
	files := make(map[string]string, len(resp.Files))
	for _, f := range resp.Files {
		files[f.Path] = string(f.Content)
	}
	return files
}

var _ = Describe("Aliases", func() {
	DescribeTable("Should generate a use of an alias as a use of its target",
		func(ctx SpecContext, newPlugin func() plugin.Plugin) {
			aliased := generateFiles(ctx, aliasHead+aliasDecls+aliasUses, newPlugin())
			direct := generateFiles(ctx, aliasHead+directUses(), newPlugin())
			Expect(aliased).To(HaveLen(len(direct)))
			for path, content := range direct {
				Expect(aliased).To(HaveKey(path))
				onlyAliased, onlyDirect := lineDiff(
					normalizeLines(path, aliased[path]),
					normalizeLines(path, content),
				)
				Expect(onlyAliased).To(BeEmpty(), path)
				Expect(onlyDirect).To(BeEmpty(), path)
			}
		},
		Entry("go/types", func() plugin.Plugin {
			return gotypes.New(gotypes.DefaultOptions())
		}),
		Entry("go/pb", func() plugin.Plugin { return gopb.New(gopb.DefaultOptions()) }),
		Entry("go/marshal", func() plugin.Plugin {
			opts := marshal.DefaultOptions()
			opts.RequireVersioned = false
			return marshal.New(opts)
		}),
		Entry("ts/types", func() plugin.Plugin {
			return tstypes.New(tstypes.DefaultOptions())
		}),
		Entry("py/types", func() plugin.Plugin {
			return pytypes.New(pytypes.DefaultOptions())
		}),
		Entry("cpp/types", func() plugin.Plugin {
			return cpptypes.New(cpptypes.DefaultOptions())
		}),
		Entry("cpp/json", func() plugin.Plugin {
			return cppjson.New(cppjson.DefaultOptions())
		}),
		Entry("cpp/pb", func() plugin.Plugin {
			return cpppb.New(cpppb.DefaultOptions())
		}),
		Entry("pb/types", func() plugin.Plugin {
			return pbtypes.New(pbtypes.DefaultOptions())
		}),
	)
})
