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
	resp := &plugin.Response{Files: make([]plugin.File, 0)}

	var skip SkipFunc
	if g.IncludeHand {
		skip = func(typ resolution.Type) bool { return omit.IsType(typ, g.Domain) }
	} else {
		skip = func(typ resolution.Type) bool { return omit.IsSkipped(typ, g.Domain) }
	}

	var typeDefCollector *Collector
	if g.CollectTypeDefs {
		typeDefCollector = NewCollector(g.Domain, req).WithSkipFunc(skip)
		if err := typeDefCollector.AddAll(req.Resolutions.DistinctTypes()); err != nil {
			return nil, err
		}
		if err := typeDefCollector.AddAll(req.Resolutions.AliasTypes()); err != nil {
			return nil, err
		}
	}

	structCollector := NewCollector(g.Domain, req).WithSkipFunc(skip)
	if err := structCollector.AddAll(req.Resolutions.StructTypes()); err != nil {
		return nil, err
	}

	var unionCollector *Collector
	if g.CollectUnions {
		unionCollector = NewCollector(g.Domain, req).WithSkipFunc(skip)
		if err := unionCollector.AddAll(req.Resolutions.UnionTypes()); err != nil {
			return nil, err
		}
	}

	var enumCollector *Collector
	if g.CollectEnums {
		enumCollector = NewCollector(g.Domain, req).
			WithPathFunc(func(typ resolution.Type) string { return output.GetPath(typ, g.Domain) }).
			WithSkipFunc(nil)
		for _, e := range enum.CollectWithOwnOutput(req.Resolutions.EnumTypes(), g.Domain) {
			if err := enumCollector.Add(e); err != nil {
				return nil, err
			}
		}
	}

	err := structCollector.ForEach(
		func(outputPath string, structs []resolution.Type) error {
			enums := enum.CollectReferenced(structs, req.Resolutions)
			if enumCollector != nil && enumCollector.Has(outputPath) {
				if g.MergeByName {
					enums = MergeTypesByName(enums, enumCollector.Remove(outputPath))
				} else {
					enums = MergeTypes(enums, enumCollector.Remove(outputPath))
				}
			}
			if g.CollectEnums && len(structs) > 0 {
				namespace := structs[0].Namespace
				namespaceEnums := enum.CollectNamespaceEnums(
					namespace,
					outputPath,
					req.Resolutions,
					g.Domain,
					nil,
				)
				if g.MergeByName {
					enums = MergeTypesByName(enums, namespaceEnums)
				} else {
					enums = MergeTypes(enums, namespaceEnums)
				}
			}
			if g.ExtraEnumsFunc != nil {
				enums = MergeTypes(
					enums,
					g.ExtraEnumsFunc(structs, req.Resolutions, outputPath),
				)
			}
			if g.FilterEnums != nil {
				enums = g.FilterEnums(enums, outputPath)
			}
			var typeDefs []resolution.Type
			if typeDefCollector != nil && typeDefCollector.Has(outputPath) {
				typeDefs = typeDefCollector.Remove(outputPath)
			}
			var unions []resolution.Type
			if unionCollector != nil && unionCollector.Has(outputPath) {
				unions = unionCollector.Remove(outputPath)
			}
			if !g.allows(outputPath) {
				return nil
			}

			ctx := &GenerateContext{
				Namespace:  structs[0].Namespace,
				OutputPath: outputPath,
				Structs:    structs,
				Enums:      enums,
				TypeDefs:   typeDefs,
				Unions:     unions,
				Table:      req.Resolutions,
				RepoRoot:   req.RepoRoot,
			}
			content, err := g.FileGenerator.GenerateFile(ctx)
			if err != nil {
				return errors.Wrapf(err, "failed to generate %s", outputPath)
			}
			if content == "" {
				return nil
			}
			resp.Files = append(resp.Files, plugin.File{
				Path:    fmt.Sprintf("%s/%s", outputPath, g.FilePattern),
				Content: []byte(content),
			})
			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	if unionCollector != nil {
		err = unionCollector.ForEach(
			func(outputPath string, unions []resolution.Type) error {
				namespace := unions[0].Namespace
				enums := enum.CollectReferenced(unions, req.Resolutions)
				if enumCollector != nil && enumCollector.Has(outputPath) {
					if g.MergeByName {
						enums = MergeTypesByName(
							enums,
							enumCollector.Remove(outputPath),
						)
					} else {
						enums = MergeTypes(enums, enumCollector.Remove(outputPath))
					}
				}
				if g.CollectEnums {
					namespaceEnums := enum.CollectNamespaceEnums(
						namespace,
						outputPath,
						req.Resolutions,
						g.Domain,
						nil,
					)
					if g.MergeByName {
						enums = MergeTypesByName(enums, namespaceEnums)
					} else {
						enums = MergeTypes(enums, namespaceEnums)
					}
				}
				var typeDefs []resolution.Type
				if typeDefCollector != nil && typeDefCollector.Has(outputPath) {
					typeDefs = typeDefCollector.Remove(outputPath)
				}
				if !g.allows(outputPath) {
					return nil
				}

				ctx := &GenerateContext{
					Namespace:  namespace,
					OutputPath: outputPath,
					Enums:      enums,
					TypeDefs:   typeDefs,
					Unions:     unions,
					Table:      req.Resolutions,
					RepoRoot:   req.RepoRoot,
				}
				content, err := g.FileGenerator.GenerateFile(ctx)
				if err != nil {
					return errors.Wrapf(err, "failed to generate %s", outputPath)
				}
				if content == "" {
					return nil
				}
				resp.Files = append(resp.Files, plugin.File{
					Path:    fmt.Sprintf("%s/%s", outputPath, g.FilePattern),
					Content: []byte(content),
				})
				return nil
			},
		)
		if err != nil {
			return nil, err
		}
	}

	if enumCollector != nil {
		err = enumCollector.ForEach(
			func(outputPath string, enums []resolution.Type) error {
				var typeDefs []resolution.Type
				if typeDefCollector != nil && typeDefCollector.Has(outputPath) {
					typeDefs = typeDefCollector.Remove(outputPath)
				}
				if !g.allows(outputPath) {
					return nil
				}

				ctx := &GenerateContext{
					Namespace:  enums[0].Namespace,
					OutputPath: outputPath,
					Structs:    nil,
					Enums:      enums,
					TypeDefs:   typeDefs,
					Table:      req.Resolutions,
					RepoRoot:   req.RepoRoot,
				}
				content, err := g.FileGenerator.GenerateFile(ctx)
				if err != nil {
					return errors.Wrapf(err, "failed to generate %s", outputPath)
				}
				if content == "" {
					return nil
				}
				resp.Files = append(resp.Files, plugin.File{
					Path:    fmt.Sprintf("%s/%s", outputPath, g.FilePattern),
					Content: []byte(content),
				})
				return nil
			},
		)
		if err != nil {
			return nil, err
		}
	}

	if typeDefCollector != nil {
		err = typeDefCollector.ForEach(
			func(outputPath string, typeDefs []resolution.Type) error {
				if !g.allows(outputPath) {
					return nil
				}
				ctx := &GenerateContext{
					Namespace:  typeDefs[0].Namespace,
					OutputPath: outputPath,
					Structs:    nil,
					Enums:      nil,
					TypeDefs:   typeDefs,
					Table:      req.Resolutions,
					RepoRoot:   req.RepoRoot,
				}
				content, err := g.FileGenerator.GenerateFile(ctx)
				if err != nil {
					return errors.Wrapf(err, "failed to generate %s", outputPath)
				}
				if content == "" {
					return nil
				}
				resp.Files = append(resp.Files, plugin.File{
					Path:    fmt.Sprintf("%s/%s", outputPath, g.FilePattern),
					Content: []byte(content),
				})
				return nil
			},
		)
		if err != nil {
			return nil, err
		}
	}

	return resp, nil
}
