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

	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/enum"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
)

type GenerateContext struct {
	Namespace  string
	OutputPath string
	Structs    []resolution.Type
	Enums      []resolution.Type
	TypeDefs   []resolution.Type
	Table      *resolution.Table
	RepoRoot   string
}

type FileGenerator interface {
	GenerateFile(ctx *GenerateContext) (string, error)
}

// ExtraEnumsFunc is a callback for collecting additional enums beyond the standard
// referenced and standalone enums. This is used by Go to collect namespace-level enums.
type ExtraEnumsFunc func(structs []resolution.Type, table *resolution.Table, outputPath string) []resolution.Type

type Generator struct {
	Domain          string
	FilePattern     string
	FileGenerator   FileGenerator
	MergeByName     bool
	CollectTypeDefs bool
	CollectEnums    bool
	ExtraEnumsFunc  ExtraEnumsFunc
}

func (g *Generator) Generate(req *plugin.Request) (*plugin.Response, error) {
	resp := &plugin.Response{Files: make([]plugin.File, 0)}

	var typeDefCollector *Collector
	if g.CollectTypeDefs {
		typeDefCollector = NewCollector(g.Domain, req)
		if err := typeDefCollector.AddAll(req.Resolutions.DistinctTypes()); err != nil {
			return nil, err
		}
		if err := typeDefCollector.AddAll(req.Resolutions.AliasTypes()); err != nil {
			return nil, err
		}
	}

	structCollector, err := CollectStructs(g.Domain, req)
	if err != nil {
		return nil, err
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

	err = structCollector.ForEach(func(outputPath string, structs []resolution.Type) error {
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
			namespaceEnums := enum.CollectNamespaceEnums(namespace, outputPath, req.Resolutions, g.Domain, nil)
			if g.MergeByName {
				enums = MergeTypesByName(enums, namespaceEnums)
			} else {
				enums = MergeTypes(enums, namespaceEnums)
			}
		}
		if g.ExtraEnumsFunc != nil {
			enums = MergeTypes(enums, g.ExtraEnumsFunc(structs, req.Resolutions, outputPath))
		}
		var typeDefs []resolution.Type
		if typeDefCollector != nil && typeDefCollector.Has(outputPath) {
			typeDefs = typeDefCollector.Remove(outputPath)
		}

		ctx := &GenerateContext{
			Namespace:  structs[0].Namespace,
			OutputPath: outputPath,
			Structs:    structs,
			Enums:      enums,
			TypeDefs:   typeDefs,
			Table:      req.Resolutions,
			RepoRoot:   req.RepoRoot,
		}
		content, err := g.FileGenerator.GenerateFile(ctx)
		if err != nil {
			return errors.Wrapf(err, "failed to generate %s", outputPath)
		}
		resp.Files = append(resp.Files, plugin.File{
			Path:    fmt.Sprintf("%s/%s", outputPath, g.FilePattern),
			Content: []byte(content),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	if enumCollector != nil {
		err = enumCollector.ForEach(func(outputPath string, enums []resolution.Type) error {
			var typeDefs []resolution.Type
			if typeDefCollector != nil && typeDefCollector.Has(outputPath) {
				typeDefs = typeDefCollector.Remove(outputPath)
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
			resp.Files = append(resp.Files, plugin.File{
				Path:    fmt.Sprintf("%s/%s", outputPath, g.FilePattern),
				Content: []byte(content),
			})
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	if typeDefCollector != nil {
		err = typeDefCollector.ForEach(func(outputPath string, typeDefs []resolution.Type) error {
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
			resp.Files = append(resp.Files, plugin.File{
				Path:    fmt.Sprintf("%s/%s", outputPath, g.FilePattern),
				Content: []byte(content),
			})
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	return resp, nil
}
