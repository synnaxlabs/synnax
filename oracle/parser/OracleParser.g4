// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

parser grammar OracleParser;

options { tokenVocab = OracleLexer; }

// =============================================================================
// Entry Point
// =============================================================================

// A schema file consists of:
// 1. Optional imports
// 2. Optional file-level domains (apply to all definitions in file)
// 3. Type definitions (structs, enums)
schema
    : nl* (importStmt nl*)* (fileDomain nl*)* (definition nl*)* EOF
    ;

// Helper for optional/required newlines
nl  : NEWLINE ;

// =============================================================================
// Imports
// =============================================================================

// Import statement: import "schema/core/label"
importStmt
    : IMPORT STRING_LIT
    ;

// =============================================================================
// File-Level Domains
// =============================================================================

// File-level domain declarations apply to all definitions in the file
// Examples:
//   @ts output "client/ts/src/rack"
//   @py output "client/py/synnax/rack"
fileDomain
    : AT IDENT domainContent?
    ;

// =============================================================================
// Definitions
// =============================================================================

// Top-level definitions are structs, enums, or type definitions
definition
    : structDef
    | enumDef
    | typeDefDef
    ;

// =============================================================================
// Struct Definitions
// =============================================================================

// Name-first struct definitions:
//   Rack struct { ... }
//   Status struct<D extends schema> { ... }
//   Child struct extends Parent { ... }
//   Child struct extends Parent<T> { ... }
//   RackStatus struct<D extends json> extends Status<D> { ... }
//   RackStatus = status.Status<RackDetails>
//   Status<D> = status.Status<D> { @ts output "..." }
structDef
    : IDENT STRUCT typeParams? (EXTENDS typeRefList)? nl* LBRACE nl* structBody RBRACE  # StructFull
    | IDENT typeParams? EQUALS typeRef aliasBody?                                        # StructAlias
    ;

// List of type references for multiple inheritance
// Examples: Parent, Base1, Base2, Generic<T>
typeRefList
    : typeRef (COMMA nl* typeRef)*
    ;

// Optional body for struct aliases (domains only, no fields)
aliasBody
    : nl* LBRACE nl* (domain nl*)* RBRACE
    ;

// Type parameters for generic structs: <T>, <T, U>, <T extends schema>
// Supports multi-line formatting for long parameter lists
typeParams
    : LT nl* typeParam (COMMA nl* typeParam)* nl* GT
    ;

// Single type parameter with optional marker, constraint, and default
// Examples: T, T?, T extends Foo, T? extends Foo, T = Bar, T? = Bar
// The ? marker means fields using this type parameter are absent when not provided
typeParam
    : IDENT QUESTION? (EXTENDS typeRef)? (EQUALS typeRef)?
    ;

// Struct body contains fields, field omissions, actions, and/or struct-level domains
structBody
    : ((fieldDef | fieldOmit | actionDef | domain) nl*)*
    ;

// Field omission: remove an inherited field from parent struct
// Example: -parentFieldName
fieldOmit
    : MINUS IDENT
    ;

// Domain omission: remove a domain inherited from the overridden parent field.
// Only meaningful on a field that overrides a parent field (in an extends struct).
// Example: -@validate
domainOmit
    : MINUS AT IDENT
    ;

// Action definition within a struct: defines a named mutation with payload fields
// Examples:
//   action SetNodePosition {
//       key      string
//       position spatial.XY
//   }
//   action AddNode {
//       node  Node
//       props record?
//       @doc value "adds a node to the schematic"
//   }
actionDef
    : ACTION IDENT nl* LBRACE nl* actionBody RBRACE
    ;

// Action body contains payload fields and/or action-level domains
actionBody
    : ((fieldDef | domain) nl*)*
    ;

// =============================================================================
// Field Definitions
// =============================================================================

// Name-first field definitions with optional inline domains:
//   key uint32
//   key uint32 @key
//   name string @validate required
//   name string @validate { required, min_length 1, max_length 255 }
//   key uint32 @key @validate required
//   name string {
//       @validate { required, min_length 1 }
//   }
//
// In a struct that extends a parent, the type may be omitted to partially
// override an inherited field, in which case the type, optionality, and any
// unspecified default are inherited from the parent:
//   key = 0                  (change only the default)
//   name @validate required  (add a domain)
//   name -@validate          (remove an inherited domain)
// A standalone optionality marker overrides only the optionality, inheriting
// the type from the parent:
//   key?                     (inherit the type, make it soft-optional)
//   key??                    (inherit the type, make it hard-optional)
fieldDef
    : IDENT (typeRef | typeModifiers)? (EQUALS fieldDefault)? (inlineDomain | domainOmit)* fieldBody?
    ;

// A field default is a scalar/ident literal, an array literal, or a struct
// literal.
// Examples: = 0, = "v", = volts, = [], = [1, 2, 3], = { x = 0, y = 1 }
fieldDefault
    : expressionValue
    | arrayDefault
    | structDefault
    ;

// A default value in a nested position (array element or struct field). Arrays
// and structs nest recursively through this rule.
defaultValue
    : expressionValue
    | arrayDefault
    | structDefault
    ;

arrayDefault
    : LBRACKET nl* (defaultValue (COMMA nl* defaultValue)* nl*)? RBRACKET
    ;

// A struct literal binds field names to values: { field = value, field = value }
structDefault
    : LBRACE nl* (structFieldDefault (COMMA nl* structFieldDefault)* nl*)? RBRACE
    ;

structFieldDefault
    : IDENT EQUALS defaultValue
    ;

// Inline domain on a field (after type, on same line)
// Examples: @key, @validate required, @validate { required, min 1 }
inlineDomain
    : AT IDENT domainContent?
    ;

// Optional field body containing domain definitions and omissions (multi-line)
fieldBody
    : nl* LBRACE nl* ((domain | domainOmit) nl*)* RBRACE
    ;

// =============================================================================
// Domain Definitions
// =============================================================================

// Domain definition with @ prefix:
//   @key
//   @ts output "client/ts/src/rack"
//   @validate { required, min 1, max 255 }
domain
    : AT IDENT domainContent?
    ;

// Domain content: either a single expression or a block of expressions
domainContent
    : domainBlock       // { required, min 1, max 255 }
    | expression        // output "client/ts/src/rack"
    ;

// Domain block contains newline-separated expressions
domainBlock
    : nl* LBRACE nl* (expression (nl+ expression)*)? nl* RBRACE
    ;

// =============================================================================
// Type References
// =============================================================================

// Type reference with optional type args, array, optional, and nullable modifiers
// Examples: string, uuid, uuid[], string?, Status<D>, Result<T, E>[]?, map<string, uint32>
// Fixed-size arrays: uint8[4], byte[16]
typeRef
    : mapType typeModifiers?                                           # TypeRefMap
    | qualifiedIdent typeArgs? arrayModifier? typeModifiers?           # TypeRefNormal
    ;

// Array modifier: dynamic [] or fixed-size [N]
arrayModifier
    : LBRACKET RBRACKET           // [] dynamic array
    | LBRACKET INT_LIT RBRACKET   // [N] fixed-size array
    ;

// Map type: map<KeyType, ValueType>
mapType
    : MAP LT typeRef COMMA typeRef GT
    ;

// Type arguments when using a generic type: <string>, <Foo, Bar>
typeArgs
    : LT typeRef (COMMA typeRef)* GT
    ;

// Type modifiers: soft optional (?) or hard optional (??)
typeModifiers
    : QUESTION QUESTION   // ?? (hard optional - pointer in Go)
    | QUESTION            // ? (soft optional - zero value in Go)
    ;

// Qualified identifier for type names
// Simple: string, uuid, Range
// Qualified: label.Label, channel.Channel
qualifiedIdent
    : IDENT (DOT IDENT)?
    ;

// =============================================================================
// Expressions (within domains)
// =============================================================================

// Domain expressions are flexible: identifier with optional values
// Examples:
//   required                    (flag)
//   max_length 255              (identifier + number)
//   default "untitled"          (identifier + string)
//   target label.Label          (identifier + qualified ident)
//   cardinality many_to_many    (identifier + identifier)
expression
    : IDENT expressionValue*
    ;

// Expression values can be various literal types or identifiers
expressionValue
    : TRIPLE_STRING_LIT
    | STRING_LIT
    | INT_LIT
    | FLOAT_LIT
    | BOOL_LIT
    | qualifiedIdent
    ;

// =============================================================================
// Enum Definitions
// =============================================================================

// Name-first enum definition:
//   TaskState enum { pending = 0, running = 1 }
//   DataType enum { float32 = "float32", int32 = "int32" }
// An enum may extend one or more other enums, taking the union of their
// members (and optionally adding its own):
//   AxisKey enum extends XAxisKey, YAxisKey {}
enumDef
    : IDENT ENUM (EXTENDS typeRefList)? nl* LBRACE nl* enumBody RBRACE
    ;

// Enum body contains values and/or enum-level domains
enumBody
    : ((enumValue | domain) nl*)*
    ;

// Enum values require explicit values (integer or string)
// Optional body block for value-level domains (e.g., @doc)
enumValue
    : IDENT EQUALS (INT_LIT | STRING_LIT) enumValueBody?
    ;

// Optional body for enum values (domains only)
enumValueBody
    : nl* LBRACE nl* (domain nl*)* RBRACE
    ;

// =============================================================================
// Type Definitions
// =============================================================================

// Type definition creates a distinct named type based on a primitive or another typedef.
// This is NOT an alias - it creates a new type (like Go's "type Key uint32").
// Examples:
//   Key uint32
//   Key uint32 { @go output "core/pkg/service/rack" }
//   DeviceKey rack.Key
//   GoServiceStatus<Details?> Status<Details, Variant>
typeDefDef
    : IDENT typeParams? typeRef typeDefBody?
    ;

// Optional body for type definitions (domains only, no fields)
typeDefBody
    : nl* LBRACE nl* (domain nl*)* RBRACE
    ;
