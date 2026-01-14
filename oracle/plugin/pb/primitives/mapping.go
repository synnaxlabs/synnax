// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package primitives provides Protocol Buffers-specific primitive type mappings for Oracle code generation.
package primitives

import "github.com/synnaxlabs/oracle/plugin/primitives"

// mapping contains Protobuf-specific primitive type mappings.
var mapping = map[string]primitives.Mapping{
	"uuid":    {TargetType: "string"},
	"string":  {TargetType: "string"},
	"bool":    {TargetType: "bool"},
	"int8":    {TargetType: "int32"},
	"int16":   {TargetType: "int32"},
	"int32":   {TargetType: "int32"},
	"int64":   {TargetType: "int64"},
	"uint8":   {TargetType: "uint32"},
	"uint12":  {TargetType: "uint32"},
	"uint16":  {TargetType: "uint32"},
	"uint20":  {TargetType: "uint32"},
	"uint32":  {TargetType: "uint32"},
	"uint64":  {TargetType: "uint64"},
	"float32": {TargetType: "float"},
	"float64": {TargetType: "double"},
	"json":    {TargetType: "google.protobuf.Struct", Imports: []primitives.Import{{Category: "external", Path: "google/protobuf/struct.proto"}}},
	"bytes":   {TargetType: "bytes"},
	"any":     {TargetType: "google.protobuf.Value", Imports: []primitives.Import{{Category: "external", Path: "google/protobuf/struct.proto"}}},
}

// Mapper implements primitives.Mapper for Protocol Buffers code generation.
type Mapper struct{}

// Map returns the Protobuf-specific mapping for a primitive type.
func (m *Mapper) Map(name string) primitives.Mapping {
	if mapping, ok := mapping[name]; ok {
		return mapping
	}
	return primitives.Mapping{TargetType: "bytes"}
}
