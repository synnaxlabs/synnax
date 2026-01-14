// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package primitives provides Go-specific primitive type mappings for Oracle code generation.
package primitives

import "github.com/synnaxlabs/oracle/plugin/primitives"

// mapping contains Go-specific primitive type mappings.
var mapping = map[string]primitives.Mapping{
	"uuid":    {TargetType: "uuid.UUID", Imports: []primitives.Import{{Category: "external", Path: "github.com/synnaxlabs/x/uuid"}}},
	"string":  {TargetType: "string"},
	"bool":    {TargetType: "bool"},
	"int8":    {TargetType: "int8"},
	"int16":   {TargetType: "int16"},
	"int32":   {TargetType: "int32"},
	"int64":   {TargetType: "int64"},
	"uint8":   {TargetType: "uint8"},
	"uint12":  {TargetType: "types.Uint12", Imports: []primitives.Import{{Category: "internal", Path: "github.com/synnaxlabs/x/types"}}},
	"uint16":  {TargetType: "uint16"},
	"uint20":  {TargetType: "types.Uint20", Imports: []primitives.Import{{Category: "internal", Path: "github.com/synnaxlabs/x/types"}}},
	"uint32":  {TargetType: "uint32"},
	"uint64":  {TargetType: "uint64"},
	"float32": {TargetType: "float32"},
	"float64": {TargetType: "float64"},
	"json":    {TargetType: "binary.MsgpackEncodedJSON", Imports: []primitives.Import{{Category: "internal", Path: "github.com/synnaxlabs/x/binary"}}},
	"bytes":   {TargetType: "[]byte"},
	"any":     {TargetType: "any"},
	"nil":     {TargetType: "gotypes.Nil", Imports: []primitives.Import{{Category: "internal", Path: "go/types", Name: "gotypes"}}},
}

// Mapper implements primitives.Mapper for Go code generation.
type Mapper struct{}

// Map returns the Go-specific mapping for a primitive type.
func (m *Mapper) Map(name string) primitives.Mapping {
	if mapping, ok := mapping[name]; ok {
		return mapping
	}
	return primitives.Mapping{TargetType: "any"}
}
