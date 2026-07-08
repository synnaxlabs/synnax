// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

lexer grammar ArcLexer;

// =============================================================================
// Keywords
// =============================================================================

FUNC        : 'func';
IF          : 'if';
ELSE        : 'else';
RETURN      : 'return';
FOR         : 'for';
BREAK       : 'break';
CONTINUE    : 'continue';
IMPORT      : 'import';
AS          : 'as';

// Sequencing keywords
SEQUENCE    : 'sequence';
STAGE       : 'stage';
NEXT        : 'next';

// Channel keywords
CHAN        : 'chan';

// Authority keywords
AUTHORITY   : 'authority';

// Primitive types
I8          : 'i8';
I16         : 'i16';
I32         : 'i32';
I64         : 'i64';
U8          : 'u8';
U16         : 'u16';
U32         : 'u32';
U64         : 'u64';
F32         : 'f32';
F64         : 'f64';
STR         : 'str';
SERIES      : 'series';

// =============================================================================
// Operators
// =============================================================================

// Channel operators
ARROW       : '->';


DECLARE     : ':=';  // Local variable declaration
STATE_DECLARE: '$='; // Stateful variable declaration
TRANSITION  : '=>';  // Stage transition operator
ASSIGN      : '=';   // Assignment to existing variable

// Compound assignment
PLUS_ASSIGN     : '+=';
MINUS_ASSIGN    : '-=';
STAR_ASSIGN     : '*=';
SLASH_ASSIGN    : '/=';
PERCENT_ASSIGN  : '%=';

// Arithmetic
PLUS        : '+';
MINUS       : '-';
STAR        : '*';
SLASH       : '/';
PERCENT     : '%';
CARET       : '^';

// Comparison
EQ          : '==';
NEQ         : '!=';
LT          : '<';
GT          : '>';
LEQ         : '<=';
GEQ         : '>=';

// Logical
AND         : 'and';
OR          : 'or';
NOT         : 'not';

// =============================================================================
// Delimiters
// =============================================================================

LPAREN      : '(';
RPAREN      : ')';
LBRACE      : '{';
RBRACE      : '}';
LBRACKET    : '[';
RBRACKET    : ']';
COMMA       : ',';
COLON       : ':';
DOT         : '.';

// =============================================================================
// Literals
// =============================================================================

// Simple digit sequences without separators or suffixes
fragment DIGITS : DIGIT+ ;

fragment DIGIT: [0-9];

// Simple numeric literals without suffixes or special formats
INTEGER_LITERAL
    : DIGITS
    ;

FLOAT_LITERAL
    : DIGITS '.' DIGITS?
    | '.' DIGITS
    ;

// Multi-line string literal. Backtick-delimited; may span newlines. An optional
// prefix selects raw and/or format semantics. The lexer is permissive: any
// '\\' followed by any character is accepted, and the literal package
// interprets escapes (or skips them for raw strings).
STR_LITERAL_MULTI
    : STR_PREFIX? '`' (~[`\\] | '\\' .)* '`'
    ;

// Single-line string literal. An optional prefix selects raw and/or format
// semantics. The lexer accepts any '\\' followed by any character; the literal
// package validates and interprets escapes.
STR_LITERAL
    : STR_PREFIX? '"' (~["\\\r\n] | '\\' .)* '"'
    ;

fragment STR_PREFIX
    : 'r' | 'f' | 'rf' | 'fr'
    ;

// =============================================================================
// Identifiers
// =============================================================================

// '-' joins an identifier (dashed names on) unless it starts '->' or '-='.
IDENTIFIER  : [a-zA-Z_] ( [a-zA-Z0-9_] | {p.dashJoinsIdentifier()}? '-' )*;

// =============================================================================
// Comments & Whitespace
// =============================================================================

// Single-line comment
SINGLE_LINE_COMMENT: '//' ~[\r\n]* -> channel(HIDDEN);

// Multi-line comment
MULTI_LINE_COMMENT: '/*' .*? '*/' -> channel(HIDDEN);

// Whitespace - on hidden channel to preserve position info for adjacency checks
WS          : [ \t\r\n]+ -> channel(HIDDEN);
