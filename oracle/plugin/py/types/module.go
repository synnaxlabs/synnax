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

	"github.com/synnaxlabs/oracle/resolution"
)

// Module renders the Python types of a namespace for a module beside its types_gen.py,
// and collects the imports that the types need.
type Module struct{ data *templateData }

// NewModule returns a Module for the namespace whose Python output is outputPath. An
// import never takes one of names, the module's own top-level names.
func NewModule(namespace, outputPath string, names ...string) *Module {
	imports := newImportManager()
	for _, name := range names {
		imports.registerFieldName(name)
	}
	return &Module{data: &templateData{
		imports:     imports,
		Namespace:   namespace,
		OutputPath:  outputPath,
		TypesModule: toPythonModulePath(outputPath) + ".types_gen",
	}}
}

// Type returns the Python expression of typeRef. A type of the namespace comes from
// types_gen.py.
func (m *Module) Type(typeRef resolution.TypeRef, table *resolution.Table) string {
	return typeToPython(typeRef, table, m.data)
}

// Imports returns the import statements of the types that Type returned.
func (m *Module) Imports() (string, error) {
	var buf bytes.Buffer
	err := fileTemplate.ExecuteTemplate(&buf, "imports", m.data)
	return buf.String(), err
}
