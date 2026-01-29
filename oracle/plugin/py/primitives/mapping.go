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
	"int8":    {TargetType: "int"},
	"int16":   {TargetType: "int"},
	"int32":   {TargetType: "int"},
	"int64":   {TargetType: "int"},
	"uint8":   {TargetType: "int"},
	"uint12":  {TargetType: "int"},
	"uint16":  {TargetType: "int"},
	"uint20":  {TargetType: "int"},
	"uint32":  {TargetType: "int"},
	"uint64":  {TargetType: "int"},
	"float32": {TargetType: "float"},
	"float64": {TargetType: "float"},
	"json":    {TargetType: "dict[str, Any]", Imports: []primitives.Import{{Category: "typing", Name: "Any"}}},
	"bytes":   {TargetType: "bytes"},
	"any":     {TargetType: "Any", Imports: []primitives.Import{{Category: "typing", Name: "Any"}}},
}

// Mapper returns a primitives.Mapper for Python code generation.
func Mapper() primitives.Mapper {
	return primitives.NewMapper(Mappings, "Any")
}
