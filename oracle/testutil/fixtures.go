// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil

// Schema fixtures for testing oracle plugins.
// Each schema is a template that can be formatted with the appropriate domain directives.
//
// Example usage:
//
//	source := fmt.Sprintf(SimpleStructTemplate, "@go output \"core/user\"")
//	resp := testutil.MustGenerate(ctx, source, "user", loader, plugin)

// SimpleStructTemplate is a basic struct with common field types.
// Format: domain directive (e.g., @go output "path")
const SimpleStructTemplate = `
%s

User struct {
	key uuid
	name string
	age int32
	active bool
}
`

// AllPrimitivesTemplate contains all primitive types supported by oracle.
// Format: domain directive
const AllPrimitivesTemplate = `
%s

AllTypes struct {
	a uuid
	b string
	c bool
	d int8
	e int16
	f int32
	g int64
	h uint8
	i uint16
	j uint32
	k uint64
	l float32
	m float64
	n timestamp
	o timespan
	p time_range
	q json
	r bytes
}
`

// SoftOptionalTemplate contains fields with soft optional modifier (?).
// Soft optional uses zero-value semantics (omitempty in Go).
// Format: domain directive
const SoftOptionalTemplate = `
%s

OptionalFields struct {
	key uuid
	name string?
	age int32?
	parent uuid?
}
`

// HardOptionalTemplate contains fields with hard optional modifier (??).
// Hard optional uses pointer semantics (can distinguish nil from zero).
// Format: domain directive
const HardOptionalTemplate = `
%s

NullableFields struct {
	key uuid
	name string??
	age int32??
	parent uuid??
}
`

// ArrayTypesTemplate contains various array field types.
// Format: domain directive
const ArrayTypesTemplate = `
%s

Container struct {
	tags string[]
	labels uuid[]
	scores int32[]
	flags bool[]
}
`

// OptionalArrayTemplate contains optional array fields.
// Format: domain directive
const OptionalArrayTemplate = `
%s

OptionalArrays struct {
	tags string[]?
	labels uuid[]??
}
`

// MapTypeTemplate contains map field types.
// Format: domain directive
const MapTypeTemplate = `
%s

Config struct {
	settings map[string]string
	counts map[string]int32
	metadata map[string]json
}
`

// IntEnumTemplate defines an integer-based enumeration.
// Format: domain directive
const IntEnumTemplate = `
%s

Status enum {
	unknown = 0
	pending = 1
	active = 2
	completed = 3
}
`

// StringEnumTemplate defines a string-based enumeration.
// Format: domain directive
const StringEnumTemplate = `
%s

Priority enum {
	low = "low"
	medium = "medium"
	high = "high"
}
`

// EnumFieldTemplate contains a struct with enum-typed fields.
// Format: domain directive
const EnumFieldTemplate = `
%s

Status enum {
	active = 0
	inactive = 1
}

Task struct {
	key uuid
	status Status
}
`

// GenericStructTemplate defines a generic (type-parameterized) struct.
// Format: domain directive
const GenericStructTemplate = `
%s

Container struct<T> {
	value T
	count int32
}
`

// GenericWithConstraintTemplate defines a generic struct with a type constraint.
// Format: domain directive
const GenericWithConstraintTemplate = `
%s

Wrapper struct<T extends json> {
	value T
	min T
	max T
}
`

// StructExtensionTemplate demonstrates struct extension (inheritance).
// Format: domain directive
const StructExtensionTemplate = `
%s

Base struct {
	key uuid
	name string
	created_at timestamp
}

Extended struct extends Base {
	description string
	updated_at timestamp
}
`

// FieldOmissionTemplate demonstrates field omission in struct extension.
// Format: domain directive
const FieldOmissionTemplate = `
%s

Parent struct {
	key uuid
	name string
	age int32
	secret string
}

Child struct extends Parent {
	-secret
	role string
}
`

// FieldOverrideTemplate demonstrates field override in struct extension.
// Format: domain directive
const FieldOverrideTemplate = `
%s

Parent struct {
	key uuid
	name string
}

Child struct extends Parent {
	name string?
}
`

// TypeAliasTemplate demonstrates type alias definition.
// Format: domain directive
const TypeAliasTemplate = `
%s

UserKey = uuid

User struct {
	key UserKey
	name string
}
`

// DistinctTypeTemplate demonstrates distinct type definition.
// Format: domain directive
const DistinctTypeTemplate = `
%s

UserKey uuid

User struct {
	key UserKey
	name string
}
`

// SnakeCaseFieldsTemplate contains fields with snake_case naming.
// Format: domain directive
const SnakeCaseFieldsTemplate = `
%s

Record struct {
	created_at timestamp
	updated_at timestamp
	time_range time_range
	my_long_field_name string
}
`

// DocumentedStructTemplate contains a struct with documentation.
// Format: domain directive
const DocumentedStructTemplate = `
%s

/// User represents a user in the system.
/// It contains identification and profile information.
User struct {
	/// Unique identifier for the user.
	key uuid
	/// Display name of the user.
	name string
}
`

// MultipleStructsTemplate contains multiple struct definitions.
// Format: domain directive
const MultipleStructsTemplate = `
%s

User struct {
	key uuid
	name string
}

Group struct {
	key uuid
	name string
	owner uuid
}

Membership struct {
	user uuid
	group uuid
	role string
}
`

// StructReferenceTemplate contains structs that reference each other.
// Format: domain directive
const StructReferenceTemplate = `
%s

Parent struct {
	key uuid
	name string
}

Child struct {
	key uuid
	parent Parent
	name string
}
`

// NestedArrayTemplate contains arrays of struct types.
// Format: domain directive
const NestedArrayTemplate = `
%s

Item struct {
	key uuid
	name string
}

Container struct {
	key uuid
	items Item[]
}
`

// CrossNamespaceTemplate is for testing cross-namespace references.
// Use with multiple files/namespaces.
const CrossNamespaceStatusSchema = `
@go output "core/status"
@ts output "client/ts/status"

Status enum {
	active = 0
	inactive = 1
}
`

const CrossNamespaceTaskSchema = `
@go output "core/task"
@ts output "client/ts/task"

import status

Task struct {
	key uuid
	status status.Status
}
`

// DomainDirectives provides common domain directive strings for each plugin.
var DomainDirectives = map[string]string{
	"go":  `@go output "out"`,
	"ts":  `@ts output "out"`,
	"py":  `@py output "out"`,
	"cpp": `@cpp output "out"`,
	"pb":  `@go output "out"`, // pb derives from go output
}

// MultiDomainDirective returns a directive string for multiple domains.
func MultiDomainDirective(domains ...string) string {
	var result string
	for _, d := range domains {
		if directive, ok := DomainDirectives[d]; ok {
			result += directive + "\n"
		}
	}
	return result
}
