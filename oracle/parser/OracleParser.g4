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

// Top-level definitions are either structs or enums
definition
    : structDef
    | enumDef
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
    : IDENT STRUCT typeParams? (EXTENDS typeRef)? nl* LBRACE nl* structBody RBRACE  # StructFull
    | IDENT typeParams? EQUALS typeRef aliasBody?                                     # StructAlias
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

// Struct body contains fields, field omissions, and/or struct-level domains
structBody
    : ((fieldDef | fieldOmit | domain) nl*)*
    ;

// Field omission: remove an inherited field from parent struct
// Example: -parentFieldName
fieldOmit
    : MINUS IDENT
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
fieldDef
    : IDENT typeRef inlineDomain* fieldBody?
    ;

// Inline domain on a field (after type, on same line)
// Examples: @key, @validate required, @validate { required, min 1 }
inlineDomain
    : AT IDENT domainContent?
    ;

// Optional field body containing domain definitions (multi-line)
fieldBody
    : nl* LBRACE nl* (domain nl*)* RBRACE
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
typeRef
    : mapType typeModifiers?                                           # TypeRefMap
    | qualifiedIdent typeArgs? (LBRACKET RBRACKET)? typeModifiers?     # TypeRefNormal
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
    : STRING_LIT
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
enumDef
    : IDENT ENUM nl* LBRACE nl* enumBody RBRACE
    ;

// Enum body contains values and/or enum-level domains
enumBody
    : ((enumValue | domain) nl*)*
    ;

// Enum values require explicit values (integer or string)
enumValue
    : IDENT EQUALS (INT_LIT | STRING_LIT)
    ;
