// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

lexer grammar OracleLexer;

// =============================================================================
// Keywords (only these are reserved)
// =============================================================================

STRUCT      : 'struct' ;
ENUM        : 'enum' ;
IMPORT      : 'import' ;
EXTENDS     : 'extends' ;
MAP         : 'map' ;

// =============================================================================
// Symbols
// =============================================================================

LBRACE      : '{' ;
RBRACE      : '}' ;
LBRACKET    : '[' ;
RBRACKET    : ']' ;
LT          : '<' ;
GT          : '>' ;
COMMA       : ',' ;
QUESTION    : '?' ;
DOT         : '.' ;
EQUALS      : '=' ;
AT          : '@' ;
MINUS       : '-' ;

// =============================================================================
// Literals
// =============================================================================

// String literal with escape sequences
STRING_LIT  : '"' (~["\r\n\\] | ESCAPE_SEQUENCE)* '"' ;

fragment ESCAPE_SEQUENCE
    : '\\' [btnfr"\\]
    | '\\u' [0-9a-fA-F] [0-9a-fA-F] [0-9a-fA-F] [0-9a-fA-F]
    ;

// Float must come before INT to properly match decimal numbers
FLOAT_LIT   : '-'? [0-9]+ '.' [0-9]+ ;

// Integer literal
INT_LIT     : '-'? [0-9]+ ;

// Boolean literals
BOOL_LIT    : 'true' | 'false' ;

// =============================================================================
// Identifiers
// =============================================================================

// Identifiers include type names, field names, domain names, etc.
// Primitive types (uuid, string, int32, etc.) are NOT reserved keywords -
// they are just identifiers that plugins interpret specially.
IDENT       : [a-zA-Z_][a-zA-Z0-9_]* ;

// =============================================================================
// Comments & Whitespace
// =============================================================================

// Single-line comment (sent to hidden channel for formatter preservation)
LINE_COMMENT    : '//' ~[\r\n]* -> channel(HIDDEN) ;

// Multi-line comment (sent to hidden channel for formatter preservation)
BLOCK_COMMENT   : '/*' .*? '*/' -> channel(HIDDEN) ;

// Newlines are significant for separating expressions in domains
NEWLINE         : [\r\n]+ ;

// Whitespace (excluding newlines)
WS              : [ \t]+ -> skip ;
