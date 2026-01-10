// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package primitives provides Python-specific primitive type mappings for Oracle code generation.
package primitives

import "github.com/synnaxlabs/oracle/plugin/primitives"

// mapping contains Python-specific primitive type mappings.
var mapping = map[string]primitives.Mapping{
	"uuid":       {TargetType: "UUID", Imports: []primitives.Import{{Category: "uuid", Name: "UUID"}}},
	"string":     {TargetType: "str"},
	"bool":       {TargetType: "bool"},
	"int8":       {TargetType: "int"},
	"int16":      {TargetType: "int"},
	"int32":      {TargetType: "int"},
	"int64":      {TargetType: "int"},
	"uint8":      {TargetType: "int"},
	"uint12":     {TargetType: "int"},
	"uint16":     {TargetType: "int"},
	"uint20":     {TargetType: "int"},
	"uint32":     {TargetType: "int"},
	"uint64":     {TargetType: "int"},
	"float32":    {TargetType: "float"},
	"float64":    {TargetType: "float"},
	"timestamp":  {TargetType: "TimeStamp", Imports: []primitives.Import{{Category: "synnax", Name: "TimeStamp"}}},
	"timespan":   {TargetType: "TimeSpan", Imports: []primitives.Import{{Category: "synnax", Name: "TimeSpan"}}},
	"time_range": {TargetType: "TimeRange", Imports: []primitives.Import{{Category: "synnax", Name: "TimeRange"}}},
	"data_type":  {TargetType: "DataType", Imports: []primitives.Import{{Category: "synnax", Name: "DataType"}}},
	"color":      {TargetType: "str"},
	"json":       {TargetType: "dict[str, Any]", Imports: []primitives.Import{{Category: "typing", Name: "Any"}}},
	"bytes":      {TargetType: "bytes"},
	"any":        {TargetType: "Any", Imports: []primitives.Import{{Category: "typing", Name: "Any"}}},
}

// Mapper implements primitives.Mapper for Python code generation.
type Mapper struct{}

// Map returns the Python-specific mapping for a primitive type.
func (m *Mapper) Map(name string) primitives.Mapping {
	if mapping, ok := mapping[name]; ok {
		return mapping
	}
	return primitives.Mapping{TargetType: "Any"}
}
