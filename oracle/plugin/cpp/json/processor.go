// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package json

import (
	"github.com/synnaxlabs/oracle/plugin/cpp/internal/includes"
	cppnaming "github.com/synnaxlabs/oracle/plugin/cpp/naming"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/set"
)

// FieldData holds the expressions that parse a field from JSON and write it to JSON.
type FieldData = fieldData

// FieldProcessor renders JSON expressions for sibling plugins, such as cpp/actions,
// that serialize their own structs.
type FieldProcessor struct {
	// Includes receives the headers that the expressions need.
	Includes *includes.Manager
	// Namespace is the schema namespace of the file. A type in this namespace renders
	// by its bare name.
	Namespace string
	// OutputPath is the output path of the file. It sets the C++ namespace that
	// qualifies calls to the union codecs of the namespace.
	OutputPath string
	// Table resolves the field types.
	Table *resolution.Table
	// JSONPaths holds the output paths that emit a json.gen.h, as ContentPaths
	// returns them.
	JSONPaths set.Set[string]
}

// ProcessField returns the JSON expressions of field in the struct parent. The parse
// expression reads from a variable named parser, and the write expression assigns to
// a variable named j.
func (fp *FieldProcessor) ProcessField(
	field resolution.Field,
	parent resolution.Type,
) FieldData {
	data := &templateData{
		Manager:    fp.Includes,
		table:      fp.Table,
		jsonPaths:  fp.JSONPaths,
		OutputPath: fp.OutputPath,
		Namespace:  cppnaming.Namespace(fp.OutputPath),
		rawNs:      fp.Namespace,
	}
	return New(DefaultOptions()).processField(field, parent, data)
}
