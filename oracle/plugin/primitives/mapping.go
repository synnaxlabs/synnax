// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package primitives

// Import represents an import/include required for a type mapping.
type Import struct {
	Category string // Import category (e.g., "external", "internal", "system", "typing")
	Path     string // Import path or module name
	Name     string // Specific import name (for "from X import Y" patterns)
}

// Mapping defines how a primitive maps to a target language.
type Mapping struct {
	TargetType string   // The type string in the target language
	Imports    []Import // Required imports/includes
	ZeroValue  string   // Default/zero value expression (optional)
}

// Mapper provides language-specific primitive type mappings.
type Mapper interface {
	// Map returns the language-specific mapping for a primitive.
	Map(name string) Mapping
	// ZeroValue returns the zero/default value for a primitive (optional).
	ZeroValue(name string) string
}

// GoMapping contains Go-specific primitive type mappings.
var GoMapping = map[string]Mapping{
	"uuid":               {TargetType: "uuid.UUID", Imports: []Import{{Category: "external", Path: "github.com/google/uuid"}}},
	"string":             {TargetType: "string"},
	"bool":               {TargetType: "bool"},
	"int8":               {TargetType: "int8"},
	"int16":              {TargetType: "int16"},
	"int32":              {TargetType: "int32"},
	"int64":              {TargetType: "int64"},
	"uint8":              {TargetType: "uint8"},
	"uint12":             {TargetType: "types.Uint12", Imports: []Import{{Category: "internal", Path: "github.com/synnaxlabs/x/types"}}},
	"uint16":             {TargetType: "uint16"},
	"uint20":             {TargetType: "types.Uint20", Imports: []Import{{Category: "internal", Path: "github.com/synnaxlabs/x/types"}}},
	"uint32":             {TargetType: "uint32"},
	"uint64":             {TargetType: "uint64"},
	"float32":            {TargetType: "float32"},
	"float64":            {TargetType: "float64"},
	"timestamp":          {TargetType: "telem.TimeStamp", Imports: []Import{{Category: "internal", Path: "github.com/synnaxlabs/x/telem"}}},
	"timespan":           {TargetType: "telem.TimeSpan", Imports: []Import{{Category: "internal", Path: "github.com/synnaxlabs/x/telem"}}},
	"time_range":         {TargetType: "telem.TimeRange", Imports: []Import{{Category: "internal", Path: "github.com/synnaxlabs/x/telem"}}},
	"time_range_bounded": {TargetType: "telem.TimeRange", Imports: []Import{{Category: "internal", Path: "github.com/synnaxlabs/x/telem"}}},
	"data_type":          {TargetType: "telem.DataType", Imports: []Import{{Category: "internal", Path: "github.com/synnaxlabs/x/telem"}}},
	"color":              {TargetType: "color.Color", Imports: []Import{{Category: "internal", Path: "github.com/synnaxlabs/x/color"}}},
	"json":               {TargetType: "map[string]any"},
	"bytes":              {TargetType: "[]byte"},
}

// PythonMapping contains Python-specific primitive type mappings.
var PythonMapping = map[string]Mapping{
	"uuid":       {TargetType: "UUID", Imports: []Import{{Category: "uuid", Name: "UUID"}}},
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
	"timestamp":  {TargetType: "TimeStamp", Imports: []Import{{Category: "synnax", Name: "TimeStamp"}}},
	"timespan":   {TargetType: "TimeSpan", Imports: []Import{{Category: "synnax", Name: "TimeSpan"}}},
	"time_range": {TargetType: "TimeRange", Imports: []Import{{Category: "synnax", Name: "TimeRange"}}},
	"data_type":  {TargetType: "DataType", Imports: []Import{{Category: "synnax", Name: "DataType"}}},
	"color":      {TargetType: "str"},
	"json":       {TargetType: "dict[str, Any]", Imports: []Import{{Category: "typing", Name: "Any"}}},
	"bytes":      {TargetType: "bytes"},
}

// TypeScriptMapping contains TypeScript-specific primitive type mappings for Zod schemas.
var TypeScriptZodMapping = map[string]Mapping{
	"uuid":               {TargetType: "z.string().uuid()"},
	"string":             {TargetType: "z.string()"},
	"bool":               {TargetType: "z.boolean()"},
	"int8":               {TargetType: "z.number().int()"},
	"int16":              {TargetType: "z.number().int()"},
	"int32":              {TargetType: "z.number().int()"},
	"int64":              {TargetType: "z.bigint()"},
	"uint8":              {TargetType: "z.number().int().nonnegative()"},
	"uint12":             {TargetType: "z.number().int().nonnegative()"},
	"uint16":             {TargetType: "z.number().int().nonnegative()"},
	"uint20":             {TargetType: "z.number().int().nonnegative()"},
	"uint32":             {TargetType: "z.number().int().nonnegative()"},
	"uint64":             {TargetType: "z.bigint().nonnegative()"},
	"float32":            {TargetType: "z.number()"},
	"float64":            {TargetType: "z.number()"},
	"timestamp":          {TargetType: "z.number()", Imports: []Import{{Category: "external", Path: "@synnaxlabs/x/telem"}}},
	"timespan":           {TargetType: "z.number()", Imports: []Import{{Category: "external", Path: "@synnaxlabs/x/telem"}}},
	"time_range":         {TargetType: "TimeRangeZ", Imports: []Import{{Category: "external", Path: "@synnaxlabs/x/telem", Name: "TimeRangeZ"}}},
	"time_range_bounded": {TargetType: "TimeRangeZ", Imports: []Import{{Category: "external", Path: "@synnaxlabs/x/telem", Name: "TimeRangeZ"}}},
	"data_type":          {TargetType: "DataTypeZ", Imports: []Import{{Category: "external", Path: "@synnaxlabs/x/telem", Name: "DataTypeZ"}}},
	"color":              {TargetType: "z.string()"},
	"json":               {TargetType: "z.record(z.string(), z.unknown())"},
	"bytes":              {TargetType: "z.instanceof(Uint8Array)"},
}

// TypeScriptTypeMapping contains TypeScript-specific primitive type mappings for type annotations.
var TypeScriptTypeMapping = map[string]Mapping{
	"uuid":               {TargetType: "string"},
	"string":             {TargetType: "string"},
	"bool":               {TargetType: "boolean"},
	"int8":               {TargetType: "number"},
	"int16":              {TargetType: "number"},
	"int32":              {TargetType: "number"},
	"int64":              {TargetType: "bigint"},
	"uint8":              {TargetType: "number"},
	"uint12":             {TargetType: "number"},
	"uint16":             {TargetType: "number"},
	"uint20":             {TargetType: "number"},
	"uint32":             {TargetType: "number"},
	"uint64":             {TargetType: "bigint"},
	"float32":            {TargetType: "number"},
	"float64":            {TargetType: "number"},
	"timestamp":          {TargetType: "TimeStamp", Imports: []Import{{Category: "external", Path: "@synnaxlabs/x/telem", Name: "TimeStamp"}}},
	"timespan":           {TargetType: "TimeSpan", Imports: []Import{{Category: "external", Path: "@synnaxlabs/x/telem", Name: "TimeSpan"}}},
	"time_range":         {TargetType: "TimeRange", Imports: []Import{{Category: "external", Path: "@synnaxlabs/x/telem", Name: "TimeRange"}}},
	"time_range_bounded": {TargetType: "TimeRange", Imports: []Import{{Category: "external", Path: "@synnaxlabs/x/telem", Name: "TimeRange"}}},
	"data_type":          {TargetType: "DataType", Imports: []Import{{Category: "external", Path: "@synnaxlabs/x/telem", Name: "DataType"}}},
	"color":              {TargetType: "string"},
	"json":               {TargetType: "Record<string, unknown>"},
	"bytes":              {TargetType: "Uint8Array"},
}

// CppMapping contains C++-specific primitive type mappings.
var CppMapping = map[string]Mapping{
	"uuid":               {TargetType: "std::string", Imports: []Import{{Category: "system", Path: "string"}}},
	"string":             {TargetType: "std::string", Imports: []Import{{Category: "system", Path: "string"}}},
	"bool":               {TargetType: "bool"},
	"int8":               {TargetType: "std::int8_t", Imports: []Import{{Category: "system", Path: "cstdint"}}},
	"int16":              {TargetType: "std::int16_t", Imports: []Import{{Category: "system", Path: "cstdint"}}},
	"int32":              {TargetType: "std::int32_t", Imports: []Import{{Category: "system", Path: "cstdint"}}},
	"int64":              {TargetType: "std::int64_t", Imports: []Import{{Category: "system", Path: "cstdint"}}},
	"uint8":              {TargetType: "std::uint8_t", Imports: []Import{{Category: "system", Path: "cstdint"}}},
	"uint12":             {TargetType: "std::uint16_t", Imports: []Import{{Category: "system", Path: "cstdint"}}},
	"uint16":             {TargetType: "std::uint16_t", Imports: []Import{{Category: "system", Path: "cstdint"}}},
	"uint20":             {TargetType: "std::uint32_t", Imports: []Import{{Category: "system", Path: "cstdint"}}},
	"uint32":             {TargetType: "std::uint32_t", Imports: []Import{{Category: "system", Path: "cstdint"}}},
	"uint64":             {TargetType: "std::uint64_t", Imports: []Import{{Category: "system", Path: "cstdint"}}},
	"float32":            {TargetType: "float"},
	"float64":            {TargetType: "double"},
	"timestamp":          {TargetType: "telem::TimeStamp", Imports: []Import{{Category: "internal", Path: "x/cpp/telem/telem.h"}}},
	"timespan":           {TargetType: "telem::TimeSpan", Imports: []Import{{Category: "internal", Path: "x/cpp/telem/telem.h"}}},
	"time_range":         {TargetType: "telem::TimeRange", Imports: []Import{{Category: "internal", Path: "x/cpp/telem/telem.h"}}},
	"time_range_bounded": {TargetType: "telem::TimeRange", Imports: []Import{{Category: "internal", Path: "x/cpp/telem/telem.h"}}},
	"data_type":          {TargetType: "telem::DataType", Imports: []Import{{Category: "internal", Path: "x/cpp/telem/telem.h"}}},
	"color":              {TargetType: "std::string", Imports: []Import{{Category: "system", Path: "string"}}},
	"json":               {TargetType: "nlohmann::json", Imports: []Import{{Category: "internal", Path: "nlohmann/json.hpp"}}},
	"bytes":              {TargetType: "std::vector<std::uint8_t>", Imports: []Import{{Category: "system", Path: "vector"}, {Category: "system", Path: "cstdint"}}},
}

// ProtobufMapping contains Protobuf-specific primitive type mappings.
var ProtobufMapping = map[string]Mapping{
	"uuid":               {TargetType: "string"},
	"string":             {TargetType: "string"},
	"bool":               {TargetType: "bool"},
	"int8":               {TargetType: "int32"},
	"int16":              {TargetType: "int32"},
	"int32":              {TargetType: "int32"},
	"int64":              {TargetType: "int64"},
	"uint8":              {TargetType: "uint32"},
	"uint12":             {TargetType: "uint32"},
	"uint16":             {TargetType: "uint32"},
	"uint20":             {TargetType: "uint32"},
	"uint32":             {TargetType: "uint32"},
	"uint64":             {TargetType: "uint64"},
	"float32":            {TargetType: "float"},
	"float64":            {TargetType: "double"},
	"timestamp":          {TargetType: "int64"},
	"timespan":           {TargetType: "int64"},
	"time_range":         {TargetType: "telem.PBTimeRange", Imports: []Import{{Category: "internal", Path: "x/go/telem/telem.proto"}}},
	"time_range_bounded": {TargetType: "telem.PBTimeRange", Imports: []Import{{Category: "internal", Path: "x/go/telem/telem.proto"}}},
	"data_type":          {TargetType: "string"},
	"color":              {TargetType: "string"},
	"json":               {TargetType: "google.protobuf.Struct", Imports: []Import{{Category: "external", Path: "google/protobuf/struct.proto"}}},
	"bytes":              {TargetType: "bytes"},
}

// GetMapping returns the mapping for a primitive in a specific language.
// If the primitive is not found, returns a default "any/void" mapping.
func GetMapping(name string, lang string) Mapping {
	var m map[string]Mapping
	switch lang {
	case "go":
		m = GoMapping
	case "py":
		m = PythonMapping
	case "ts":
		m = TypeScriptZodMapping
	case "ts_type":
		m = TypeScriptTypeMapping
	case "cpp":
		m = CppMapping
	case "pb":
		m = ProtobufMapping
	default:
		return Mapping{TargetType: "any"}
	}
	if mapping, ok := m[name]; ok {
		return mapping
	}
	return Mapping{TargetType: "any"}
}
