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

// Mappings contains Python-specific primitive type mappings.
var Mappings = map[string]primitives.Mapping{
	"uuid":    {TargetType: "UUID", Imports: []primitives.Import{{Category: "uuid", Name: "UUID"}}},
	"string":  {TargetType: "str"},
	"bool":    {TargetType: "bool"},
	"int8":    {TargetType: "Int8", Imports: []primitives.Import{{Category: "x", Name: "Int8"}}},
	"int16":   {TargetType: "Int16", Imports: []primitives.Import{{Category: "x", Name: "Int16"}}},
	"int32":   {TargetType: "Int32", Imports: []primitives.Import{{Category: "x", Name: "Int32"}}},
	"int64":   {TargetType: "Int64", Imports: []primitives.Import{{Category: "x", Name: "Int64"}}},
	"uint8":   {TargetType: "Uint8", Imports: []primitives.Import{{Category: "x", Name: "Uint8"}}},
	"uint12":  {TargetType: "Uint12", Imports: []primitives.Import{{Category: "x", Name: "Uint12"}}},
	"uint16":  {TargetType: "Uint16", Imports: []primitives.Import{{Category: "x", Name: "Uint16"}}},
	"uint20":  {TargetType: "Uint20", Imports: []primitives.Import{{Category: "x", Name: "Uint20"}}},
	"uint32":  {TargetType: "Uint32", Imports: []primitives.Import{{Category: "x", Name: "Uint32"}}},
	"uint64":  {TargetType: "Uint64", Imports: []primitives.Import{{Category: "x", Name: "Uint64"}}},
	"float32": {TargetType: "float"},
	"float64": {TargetType: "float"},
	"record":  {TargetType: "dict[str, Any]", Imports: []primitives.Import{{Category: "typing", Name: "Any"}}},
	"bytes":   {TargetType: "bytes"},
	"any":     {TargetType: "Any", Imports: []primitives.Import{{Category: "typing", Name: "Any"}}},
	"nil":     {TargetType: "None"},
}

// Mapper returns a primitives.Mapper for Python code generation.
func Mapper() primitives.Mapper {
	return primitives.NewMapper(Mappings, "Any")
}
