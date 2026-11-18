// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package framework

import (
	"fmt"

	"github.com/synnaxlabs/oracle/domain/omit"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/enum"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
)

type GenerateContext struct {
	Table      *resolution.Table
	Namespace  string
	OutputPath string
	RepoRoot   string
	Structs    []resolution.Type
	Enums      []resolution.Type
	TypeDefs   []resolution.Type
	Unions     []resolution.Type
	// FilePath is the schema file that the output renders. Two schemas with one file
	// name share a namespace, so locality checks compare this path.
	FilePath string
}

type FileGenerator interface {
	GenerateFile(ctx *GenerateContext) (string, error)
}

// ExtraEnumsFunc is a callback for collecting additional enums beyond the standard
// referenced and standalone enums. This is used by Go to collect namespace-level enums.
type ExtraEnumsFunc func(structs []resolution.Type, table *resolution.Table, outputPath string) []resolution.Type

type Generator struct {
	FileGenerator  FileGenerator
	ExtraEnumsFunc ExtraEnumsFunc
	// PathFilter, when non-nil, restricts generation to output paths for
	// which it returns true. Types at filtered-out paths are still collected
	// (so grouping and enum merging stay identical) but produce no file.
	PathFilter func(outputPath string) bool
	// FilterEnums, when non-nil, filters the referenced-enum set merged into
	// each file. Used when a file's structs may reference enums that belong to
	// a different output path and must not be re-declared.
	FilterEnums     func(enums []resolution.Type, outputPath string) []resolution.Type
	Domain          string
	FilePattern     string
	MergeByName     bool
	CollectTypeDefs bool
	CollectEnums    bool
	CollectUnions   bool
	// IncludeHand collects hand-written types alongside generated ones, so
	// re-export surfaces can alias them. Omitted types are still skipped.
	IncludeHand bool
}

// allows reports whether outputPath passes the generator's PathFilter.
func (g *Generator) allows(outputPath string) bool {
	return g.PathFilter == nil || g.PathFilter(outputPath)
}

func (g *Generator) Generate(req *plugin.Request) (*plugin.Response, error) {
	gen, err := g.collect(req)
	if err != nil {
		return nil, err
	}
	if err := gen.structs.ForEach(gen.emitStructs); err != nil {
		return nil, err
	}
	if gen.unions != nil {
		if err := gen.unions.ForEach(gen.emitUnions); err != nil {
			return nil, err
		}
	}
	if gen.enums != nil {
		if err := gen.enums.ForEach(gen.emitEnums); err != nil {
			return nil, err
		}
	}
	if gen.typeDefs != nil {
		if err := gen.typeDefs.ForEach(gen.emitTypeDefs); err != nil {
			return nil, err
		}
	}
	return &plugin.Response{Files: gen.files}, nil
}

// generation is the state of one Generate call. A collector is nil when the generator
// does not collect its kind of type.
type generation struct {
	g        *Generator
	req      *plugin.Request
	typeDefs *Collector
	structs  *Collector
	unions   *Collector
	enums    *Collector
	files    []plugin.File
}

func (g *Generator) collect(req *plugin.Request) (*generation, error) {
	gen := &generation{g: g, req: req, files: make([]plugin.File, 0)}
	skip := g.skipFunc()
	if g.CollectTypeDefs {
		gen.typeDefs = NewCollector(g.Domain, req).WithSkipFunc(skip)
		if err := gen.typeDefs.AddAll(req.Resolutions.DistinctTypes()); err != nil {
			return nil, err
		}
		if err := gen.typeDefs.AddAll(req.Resolutions.AliasTypes()); err != nil {
			return nil, err
		}
	}
	gen.structs = NewCollector(g.Domain, req).WithSkipFunc(skip)
	if err := gen.structs.AddAll(req.Resolutions.StructTypes()); err != nil {
		return nil, err
	}
	if g.CollectUnions {
		gen.unions = NewCollector(g.Domain, req).WithSkipFunc(skip)
		if err := gen.unions.AddAll(req.Resolutions.UnionTypes()); err != nil {
			return nil, err
		}
	}
	if g.CollectEnums {
		gen.enums = NewCollector(g.Domain, req).
			WithPathFunc(func(typ resolution.Type) string { return output.GetPath(typ, g.Domain) }).
			WithSkipFunc(nil)
		for _, e := range enum.CollectWithOwnOutput(req.Resolutions.EnumTypes(), g.Domain) {
			if err := gen.enums.Add(e); err != nil {
				return nil, err
			}
		}
	}
	return gen, nil
}

func (g *Generator) skipFunc() SkipFunc {
	if g.IncludeHand {
		return func(typ resolution.Type) bool { return omit.IsType(typ, g.Domain) }
	}
	return func(typ resolution.Type) bool { return omit.IsSkipped(typ, g.Domain) }
}

func (gen *generation) emitStructs(outputPath string, structs []resolution.Type) error {
	referenced := enum.CollectReferenced(structs, gen.req.Resolutions)
	enums := gen.ownEnums(referenced, outputPath, structs[0].Namespace)
	if gen.g.ExtraEnumsFunc != nil {
		extra := gen.g.ExtraEnumsFunc(structs, gen.req.Resolutions, outputPath)
		enums = MergeTypes(enums, extra)
	}
	if gen.g.FilterEnums != nil {
		enums = gen.g.FilterEnums(enums, outputPath)
	}
	typeDefs := take(gen.typeDefs, outputPath)
	unions := take(gen.unions, outputPath)
	return gen.emit(&GenerateContext{
		Namespace:  structs[0].Namespace,
		FilePath:   structs[0].FilePath,
		OutputPath: outputPath,
		Structs:    structs,
		Enums:      enums,
		TypeDefs:   typeDefs,
		Unions:     unions,
	})
}

func (gen *generation) emitUnions(outputPath string, unions []resolution.Type) error {
	referenced := enum.CollectReferenced(unions, gen.req.Resolutions)
	enums := gen.ownEnums(referenced, outputPath, unions[0].Namespace)
	typeDefs := take(gen.typeDefs, outputPath)
	return gen.emit(&GenerateContext{
		Namespace:  unions[0].Namespace,
		FilePath:   unions[0].FilePath,
		OutputPath: outputPath,
		Enums:      enums,
		TypeDefs:   typeDefs,
		Unions:     unions,
	})
}

func (gen *generation) emitEnums(outputPath string, enums []resolution.Type) error {
	typeDefs := take(gen.typeDefs, outputPath)
	return gen.emit(&GenerateContext{
		Namespace:  enums[0].Namespace,
		FilePath:   enums[0].FilePath,
		OutputPath: outputPath,
		Enums:      enums,
		TypeDefs:   typeDefs,
	})
}

func (gen *generation) emitTypeDefs(
	outputPath string,
	typeDefs []resolution.Type,
) error {
	return gen.emit(&GenerateContext{
		Namespace:  typeDefs[0].Namespace,
		FilePath:   typeDefs[0].FilePath,
		OutputPath: outputPath,
		TypeDefs:   typeDefs,
	})
}

// ownEnums merges into enums the enums that generate into outputPath on their own, and,
// when the generator collects enums, the enums of namespace. It removes the merged
// enums from the enum collector so that they get no file of their own.
func (gen *generation) ownEnums(
	enums []resolution.Type,
	outputPath, namespace string,
) []resolution.Type {
	if gen.enums != nil && gen.enums.Has(outputPath) {
		enums = gen.merge(enums, gen.enums.Remove(outputPath))
	}
	if !gen.g.CollectEnums {
		return enums
	}
	namespaceEnums := enum.CollectNamespaceEnums(
		namespace,
		outputPath,
		gen.req.Resolutions,
		gen.g.Domain,
		nil,
	)
	return gen.merge(enums, namespaceEnums)
}

func (gen *generation) merge(a, b []resolution.Type) []resolution.Type {
	if gen.g.MergeByName {
		return MergeTypesByName(a, b)
	}
	return MergeTypes(a, b)
}

// emit generates the file of ctx when the path filter allows its output path. It sets
// the fields of ctx that every file shares.
func (gen *generation) emit(ctx *GenerateContext) error {
	if !gen.g.allows(ctx.OutputPath) {
		return nil
	}
	ctx.Table = gen.req.Resolutions
	ctx.RepoRoot = gen.req.RepoRoot
	content, err := gen.g.FileGenerator.GenerateFile(ctx)
	if err != nil {
		return errors.Wrapf(err, "failed to generate %s", ctx.OutputPath)
	}
	if content == "" {
		return nil
	}
	gen.files = append(gen.files, plugin.File{
		Path:    fmt.Sprintf("%s/%s", ctx.OutputPath, gen.g.FilePattern),
		Content: []byte(content),
	})
	return nil
}

// take removes and returns the types that c holds for outputPath. c may be nil.
func take(c *Collector, outputPath string) []resolution.Type {
	if c == nil || !c.Has(outputPath) {
		return nil
	}
	return c.Remove(outputPath)
}
