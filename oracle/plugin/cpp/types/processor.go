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
	"github.com/synnaxlabs/oracle/plugin/cpp/internal/includes"
	"github.com/synnaxlabs/oracle/resolution"
)

// FieldData is the C++ member declaration of a field.
type FieldData = fieldData

// FieldProcessor renders C++ member declarations for sibling plugins, such as
// cpp/actions, that declare their own structs.
type FieldProcessor struct {
	// Includes receives the headers that the rendered types need.
	Includes *includes.Manager
	// FilePath is the schema file of the generated file. A type from this schema
	// renders by its bare name.
	FilePath string
	// Table resolves the field types.
	Table *resolution.Table
}

// ProcessField returns the member declaration of field in the struct parent.
func (fp *FieldProcessor) ProcessField(
	field resolution.Field,
	parent resolution.Type,
) FieldData {
	data := &templateData{Manager: fp.Includes, table: fp.Table, filePath: fp.FilePath}
	return New(DefaultOptions()).processField(field, parent, data)
}
