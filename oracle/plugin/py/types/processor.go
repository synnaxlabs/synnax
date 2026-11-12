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
	"bytes"
	"slices"
	"strings"

	"github.com/synnaxlabs/oracle/domain/omit"
	"github.com/synnaxlabs/oracle/plugin/enum"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

// FieldData is the Python declaration of a model field.
type FieldData = fieldData

// FieldProcessor renders Python model fields for sibling plugins, such as py/actions,
// that write a module beside the types module of a namespace. It collects the imports
// that the fields need, and imports the types of the namespace from its types module.
type FieldProcessor struct {
	data  *templateData
	table *resolution.Table
	// collisions holds the namespace types whose names the module binds to something
	// else, so their import cannot keep the name.
	collisions set.Set[string]
}

// NewFieldProcessor returns a FieldProcessor for a module of the given schema namespace
// at the given output path.
func NewFieldProcessor(
	namespace string,
	outputPath string,
	table *resolution.Table,
) *FieldProcessor {
	return &FieldProcessor{
		data: &templateData{
			Namespace:       namespace,
			OutputPath:      outputPath,
			GeneratedModule: generatedModule(),
			imports:         newImportManager(),
			missingIncludes: make(set.Set[string]),
		},
		table:      table,
		collisions: make(set.Set[string]),
	}
}

// generatedModule returns the name of the types module in each output package.
func generatedModule() string {
	return strings.TrimSuffix(DefaultOptions().FileNamePattern, ".py")
}

// Reserve marks name as bound in the module, so a module import takes an alias.
func (fp *FieldProcessor) Reserve(name string) { fp.data.imports.reserve(name) }

// ReserveField reserves the attribute name of field. Reserve every field of a model
// before processing any, so no module import takes a name that a class body binds.
func (fp *FieldProcessor) ReserveField(field resolution.Field) {
	fp.Reserve(fieldName(field.Name))
}

// AddTyping imports name from typing.
func (fp *FieldProcessor) AddTyping(name string) { fp.data.imports.addTyping(name) }

// AddPydantic imports name from pydantic.
func (fp *FieldProcessor) AddPydantic(name string) { fp.data.imports.addPydantic(name) }

// ProcessField returns the declaration of field, a field of a model in the module.
func (fp *FieldProcessor) ProcessField(field resolution.Field) FieldData {
	fd := processField(field, fp.table, fp.data, nil, nil)
	fp.importNamespaceTypes(field.Type)
	return fd
}

// importNamespaceTypes imports every type of the namespace that ref names from the
// types module that declares it. A hand-written type is imported by typeToPython.
func (fp *FieldProcessor) importNamespaceTypes(ref resolution.TypeRef) {
	if ref.IsTypeParam() {
		return
	}
	for _, arg := range ref.TypeArgs {
		fp.importNamespaceTypes(arg)
	}
	resolved, ok := ref.Resolve(fp.table)
	if !ok || resolved.Namespace != fp.data.Namespace ||
		omit.IsSkipped(resolved, "py") {
		return
	}
	var outputPath string
	if _, isEnum := resolved.Form.(resolution.EnumForm); isEnum {
		outputPath = enum.FindOutputPath(resolved, fp.table, "py")
	} else {
		outputPath = output.GetPath(resolved, "py")
	}
	if outputPath == "" {
		return
	}
	name := getPyName(resolved)
	module := toPythonModulePath(outputPath) + "." + generatedModule()
	if alias := fp.data.imports.addModuleImport(module, name); alias != name {
		fp.collisions.Add(resolved.QualifiedName)
	}
}

// Imports returns the import lines of the module. It returns an error when a field
// names a hand-written type with no @py include module, or a namespace type whose name
// the module binds to something else.
func (fp *FieldProcessor) Imports() (string, error) {
	if err := fp.data.missingIncludesErr(); err != nil {
		return "", err
	}
	if len(fp.collisions) > 0 {
		return "", errors.Wrapf(
			validate.ErrValidation,
			"types whose names the %s module binds to something else: %s",
			fp.data.OutputPath,
			sortedJoin(fp.collisions),
		)
	}
	var buf bytes.Buffer
	if err := fileTemplate.ExecuteTemplate(&buf, "imports", fp.data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func sortedJoin(names set.Set[string]) string {
	s := names.Slice()
	slices.Sort(s)
	return strings.Join(s, ", ")
}
