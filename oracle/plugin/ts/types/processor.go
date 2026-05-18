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
	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/ts/internal/imports"
	"github.com/synnaxlabs/oracle/plugin/ts/internal/paths"
	"github.com/synnaxlabs/oracle/resolution"
)

// FieldData is the public TS field representation produced by FieldProcessor.
// It matches the internal fieldData struct used by the types template.
type FieldData = fieldData

// FieldProcessor produces TS zod and TypeScript representations of schema
// fields, accumulating any required imports into the supplied Manager. It is
// the narrow API consumed by sibling plugins (e.g., ts/actions) that need to
// emit zod schemas for arbitrary fields without re-implementing the full
// type-resolution pipeline that lives inside ts/types.
type FieldProcessor struct {
	// Imports receives any TS imports required by the produced field data.
	Imports *imports.Manager
	// Namespace is the schema namespace the consumer is generating into.
	// Used to detect same-namespace zod imports.
	Namespace string
	// OutputPath is the consumer's TS output directory. Used to compute
	// relative import paths for same-namespace types.
	OutputPath string
	// Request is the active plugin Request, providing access to the
	// resolution table and other generation state.
	Request *plugin.Request
}

// ProcessField returns the FieldData for a struct field, registering any
// required zod imports on the configured Manager.
func (fp *FieldProcessor) ProcessField(
	field resolution.Field,
	parentType resolution.Type,
) FieldData {
	p := New(DefaultOptions())
	return p.processField(field, parentType, fp.Request.Resolutions, fp.data(), false, false)
}

// CollectTypeImports walks ref and adds a zod import for every same-namespace
// type encountered, recursing through TypeArgs. Type parameters are skipped.
func (fp *FieldProcessor) CollectTypeImports(ref *resolution.TypeRef) {
	if ref.IsTypeParam() {
		return
	}
	for i := range ref.TypeArgs {
		fp.CollectTypeImports(&ref.TypeArgs[i])
	}
	resolved, ok := ref.Resolve(fp.Request.Resolutions)
	if !ok {
		return
	}
	if resolved.Namespace == fp.Namespace {
		zodName := lo.CamelCase(resolved.Name) + "Z"
		importPath := paths.CalculateImport(fp.OutputPath, fp.OutputPath) + "/types.gen"
		fp.Imports.AddImport(importPath, zodName)
	}
}

func (fp *FieldProcessor) data() *templateData {
	return &templateData{
		Manager:    fp.Imports,
		Namespace:  fp.Namespace,
		OutputPath: fp.OutputPath,
		Request:    fp.Request,
	}
}
