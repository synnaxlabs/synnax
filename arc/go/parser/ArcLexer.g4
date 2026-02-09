lexer grammar ArcLexer;

// =============================================================================
// Keywords
// =============================================================================

FUNC        : 'func';
IF          : 'if';
ELSE        : 'else';
RETURN      : 'return';

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

// String literal
STR_LITERAL
    : '"' (~["\\\r\n] | ESCAPE_SEQUENCE)* '"'
    ;

fragment ESCAPE_SEQUENCE
    : '\\' [btnfr"\\]
    | '\\u' [0-9a-fA-F] [0-9a-fA-F] [0-9a-fA-F] [0-9a-fA-F]
    ;

// =============================================================================
// Identifiers
// =============================================================================

IDENTIFIER  : [a-zA-Z_] [a-zA-Z0-9_]*;

// =============================================================================
// Comments & Whitespace
// =============================================================================

// Single-line comment
SINGLE_LINE_COMMENT: '//' ~[\r\n]* -> channel(HIDDEN);

// Multi-line comment
MULTI_LINE_COMMENT: '/*' .*? '*/' -> channel(HIDDEN);

// Whitespace - on hidden channel to preserve position info for adjacency checks
WS          : [ \t\r\n]+ -> channel(HIDDEN);
