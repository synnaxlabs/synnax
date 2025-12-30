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

// A schema file consists of optional imports followed by definitions
schema
    : nl* (importStmt nl*)* (definition nl*)* EOF
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

// struct Range { ... }
structDef
    : STRUCT IDENT nl* LBRACE nl* structBody RBRACE
    ;

// Struct body contains fields and/or struct-level domains
structBody
    : ((fieldDef | domainDef) nl*)*
    ;

// =============================================================================
// Field Definitions
// =============================================================================

// field name string { ... }
// field key uuid
fieldDef
    : FIELD IDENT typeRef fieldBody?
    ;

// Optional field body containing domain definitions
fieldBody
    : nl* LBRACE nl* (domainDef nl*)* RBRACE
    ;

// =============================================================================
// Domain Definitions
// =============================================================================

// domain validate { required, max_length 255 }
// domain sort (empty domain)
domainDef
    : DOMAIN IDENT domainBody?
    ;

// Domain body contains newline-separated expressions
domainBody
    : nl* LBRACE nl* (expression (nl+ expression)*)? nl* RBRACE
    ;

// =============================================================================
// Type References
// =============================================================================

// Type reference with optional array, optional, and nullable modifiers
// Examples: string, uuid, uuid[], string?, string!, string?!, label.Label[]?
typeRef
    : qualifiedIdent (LBRACKET RBRACKET)? typeModifiers?
    ;

// Type modifiers: optional (?), nullable (!), or both in any order
typeModifiers
    : QUESTION BANG?   // ?  or ?!
    | BANG QUESTION?   // !  or !?
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

// enum TaskState { pending = 0, running = 1 }
// enum DataType { float32 = "float32", int32 = "int32" }
enumDef
    : ENUM IDENT nl* LBRACE nl* (enumValue nl*)* RBRACE
    ;

// Enum values require explicit values (integer or string)
enumValue
    : IDENT EQUALS (INT_LIT | STRING_LIT)
    ;
