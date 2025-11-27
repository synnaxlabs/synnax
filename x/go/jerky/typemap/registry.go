// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package typemap provides type mapping from Go types to protobuf types.
package typemap

// Mapping defines how a Go type maps to a proto type and the translation expressions.
type Mapping struct {
	// GoType is the Go type name (e.g., "uuid.UUID", "string").
	GoType string
	// ProtoType is the protobuf type (e.g., "string", "int64").
	ProtoType string
	// ForwardExpr is the template expression for Go -> Proto translation.
	// Use {{.Field}} as placeholder for the field name.
	ForwardExpr string
	// BackwardExpr is the template expression for Proto -> Go translation.
	// Use {{.Field}} as placeholder for the field name.
	BackwardExpr string
	// NeedsImport specifies required imports for translation code.
	NeedsImport []string
	// CanFail indicates if backward translation can return an error.
	CanFail bool
	// ProtoImport is the proto import path if this type requires importing another proto.
	ProtoImport string
}

// Registry holds type mappings.
type Registry struct {
	mappings map[string]Mapping
}

// NewRegistry creates a new type mapping registry.
func NewRegistry() *Registry {
	return &Registry{
		mappings: make(map[string]Mapping),
	}
}

// Register adds a type mapping to the registry.
func (r *Registry) Register(m Mapping) {
	r.mappings[m.GoType] = m
}

// Get retrieves a type mapping by Go type name.
func (r *Registry) Get(goType string) (Mapping, bool) {
	m, ok := r.mappings[goType]
	return m, ok
}

// Has checks if a mapping exists for the given Go type.
func (r *Registry) Has(goType string) bool {
	_, ok := r.mappings[goType]
	return ok
}

// DefaultRegistry returns a registry with default type mappings.
func DefaultRegistry() *Registry {
	r := NewRegistry()

	// Primitives - direct mappings
	r.Register(Mapping{
		GoType:       "bool",
		ProtoType:    "bool",
		ForwardExpr:  "{{.Field}}",
		BackwardExpr: "{{.Field}}",
	})
	r.Register(Mapping{
		GoType:       "string",
		ProtoType:    "string",
		ForwardExpr:  "{{.Field}}",
		BackwardExpr: "{{.Field}}",
	})
	r.Register(Mapping{
		GoType:       "[]byte",
		ProtoType:    "bytes",
		ForwardExpr:  "{{.Field}}",
		BackwardExpr: "{{.Field}}",
	})

	// Integers - widened to 64-bit
	r.Register(Mapping{
		GoType:       "int",
		ProtoType:    "int64",
		ForwardExpr:  "int64({{.Field}})",
		BackwardExpr: "int({{.Field}})",
	})
	r.Register(Mapping{
		GoType:       "int8",
		ProtoType:    "int32",
		ForwardExpr:  "int32({{.Field}})",
		BackwardExpr: "int8({{.Field}})",
	})
	r.Register(Mapping{
		GoType:       "int16",
		ProtoType:    "int32",
		ForwardExpr:  "int32({{.Field}})",
		BackwardExpr: "int16({{.Field}})",
	})
	r.Register(Mapping{
		GoType:       "int32",
		ProtoType:    "int32",
		ForwardExpr:  "{{.Field}}",
		BackwardExpr: "{{.Field}}",
	})
	r.Register(Mapping{
		GoType:       "int64",
		ProtoType:    "int64",
		ForwardExpr:  "{{.Field}}",
		BackwardExpr: "{{.Field}}",
	})

	// Unsigned integers
	r.Register(Mapping{
		GoType:       "uint",
		ProtoType:    "uint64",
		ForwardExpr:  "uint64({{.Field}})",
		BackwardExpr: "uint({{.Field}})",
	})
	r.Register(Mapping{
		GoType:       "uint8",
		ProtoType:    "uint32",
		ForwardExpr:  "uint32({{.Field}})",
		BackwardExpr: "uint8({{.Field}})",
	})
	r.Register(Mapping{
		GoType:       "uint16",
		ProtoType:    "uint32",
		ForwardExpr:  "uint32({{.Field}})",
		BackwardExpr: "uint16({{.Field}})",
	})
	r.Register(Mapping{
		GoType:       "uint32",
		ProtoType:    "uint32",
		ForwardExpr:  "{{.Field}}",
		BackwardExpr: "{{.Field}}",
	})
	r.Register(Mapping{
		GoType:       "uint64",
		ProtoType:    "uint64",
		ForwardExpr:  "{{.Field}}",
		BackwardExpr: "{{.Field}}",
	})

	// Floats
	r.Register(Mapping{
		GoType:       "float32",
		ProtoType:    "float",
		ForwardExpr:  "{{.Field}}",
		BackwardExpr: "{{.Field}}",
	})
	r.Register(Mapping{
		GoType:       "float64",
		ProtoType:    "double",
		ForwardExpr:  "{{.Field}}",
		BackwardExpr: "{{.Field}}",
	})

	// Common external types
	r.Register(Mapping{
		GoType:       "uuid.UUID",
		ProtoType:    "string",
		ForwardExpr:  "{{.Field}}.String()",
		BackwardExpr: "uuid.MustParse({{.Field}})",
		NeedsImport:  []string{"github.com/google/uuid"},
		CanFail:      false,
	})
	r.Register(Mapping{
		GoType:       "time.Time",
		ProtoType:    "int64",
		ForwardExpr:  "{{.Field}}.UnixNano()",
		BackwardExpr: "time.Unix(0, {{.Field}})",
		NeedsImport:  []string{"time"},
	})
	r.Register(Mapping{
		GoType:       "time.Duration",
		ProtoType:    "int64",
		ForwardExpr:  "int64({{.Field}})",
		BackwardExpr: "time.Duration({{.Field}})",
		NeedsImport:  []string{"time"},
	})

	// Synnax-specific types
	r.Register(Mapping{
		GoType:       "telem.TimeStamp",
		ProtoType:    "int64",
		ForwardExpr:  "int64({{.Field}})",
		BackwardExpr: "telem.TimeStamp({{.Field}})",
		NeedsImport:  []string{"github.com/synnaxlabs/x/telem"},
	})
	r.Register(Mapping{
		GoType:       "telem.TimeSpan",
		ProtoType:    "int64",
		ForwardExpr:  "int64({{.Field}})",
		BackwardExpr: "telem.TimeSpan({{.Field}})",
		NeedsImport:  []string{"github.com/synnaxlabs/x/telem"},
	})

	return r
}
