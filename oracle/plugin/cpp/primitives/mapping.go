// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package primitives provides C++-specific primitive type mappings for Oracle code generation.
package primitives

import "github.com/synnaxlabs/oracle/plugin/primitives"

// mapping contains C++-specific primitive type mappings.
var mapping = map[string]primitives.Mapping{
	"uuid":    {TargetType: "std::string", Imports: []primitives.Import{{Category: "system", Path: "string"}}},
	"string":  {TargetType: "std::string", Imports: []primitives.Import{{Category: "system", Path: "string"}}},
	"bool":    {TargetType: "bool"},
	"int8":    {TargetType: "std::int8_t", Imports: []primitives.Import{{Category: "system", Path: "cstdint"}}},
	"int16":   {TargetType: "std::int16_t", Imports: []primitives.Import{{Category: "system", Path: "cstdint"}}},
	"int32":   {TargetType: "std::int32_t", Imports: []primitives.Import{{Category: "system", Path: "cstdint"}}},
	"int64":   {TargetType: "std::int64_t", Imports: []primitives.Import{{Category: "system", Path: "cstdint"}}},
	"uint8":   {TargetType: "std::uint8_t", Imports: []primitives.Import{{Category: "system", Path: "cstdint"}}},
	"uint12":  {TargetType: "std::uint16_t", Imports: []primitives.Import{{Category: "system", Path: "cstdint"}}},
	"uint16":  {TargetType: "std::uint16_t", Imports: []primitives.Import{{Category: "system", Path: "cstdint"}}},
	"uint20":  {TargetType: "std::uint32_t", Imports: []primitives.Import{{Category: "system", Path: "cstdint"}}},
	"uint32":  {TargetType: "std::uint32_t", Imports: []primitives.Import{{Category: "system", Path: "cstdint"}}},
	"uint64":  {TargetType: "std::uint64_t", Imports: []primitives.Import{{Category: "system", Path: "cstdint"}}},
	"float32": {TargetType: "float"},
	"float64": {TargetType: "double"},
	"json":    {TargetType: "x::json::json", Imports: []primitives.Import{{Category: "internal", Path: "x/cpp/json/json.h"}}},
	"bytes":   {TargetType: "std::vector<std::uint8_t>", Imports: []primitives.Import{{Category: "system", Path: "vector"}, {Category: "system", Path: "cstdint"}}},
	"any":     {TargetType: "x::json::json", Imports: []primitives.Import{{Category: "internal", Path: "x/cpp/json/json.h"}}},
}

// Mapper implements primitives.Mapper for C++ code generation.
type Mapper struct{}

// Map returns the C++-specific mapping for a primitive type.
func (m *Mapper) Map(name string) primitives.Mapping {
	if mapping, ok := mapping[name]; ok {
		return mapping
	}
	return primitives.Mapping{TargetType: "void"}
}
