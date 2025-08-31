lexer grammar SlateLexer;

// =============================================================================
// Keywords
// =============================================================================

FUNC        : 'func';
TASK        : 'task';
IF          : 'if';
ELSE        : 'else';
RETURN      : 'return';
NOW         : 'now';
LEN         : 'len';

// Channel keywords
CHAN        : 'chan';
RECV_CHAN   : '<-chan';
SEND_CHAN   : '->chan';

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
STRING      : 'string';
TIMESTAMP   : 'timestamp';
TIMESPAN    : 'timespan';
SERIES      : 'series';

// =============================================================================
// Operators
// =============================================================================

// Channel operators (order matters - longer tokens first)
ARROW       : '->';
RECV        : '<-';


DECLARE     : ':=';  // Local variable declaration
STATE_DECLARE: '$='; // Stateful variable declaration
ASSIGN      : '=';   // Assignment to existing variable

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
AND         : '&&';
OR          : '||';
NOT         : '!';

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
SEMICOLON   : ';';

// =============================================================================
// Literals
// =============================================================================

// Helpers that forbid leading/trailing underscores and forbid `_` next to '.'
fragment DEC_DIGITS : DIGIT ('_'? DIGIT)* ;
fragment HEX_DIGITS : HEX_DIGIT ('_'? HEX_DIGIT)* ;
fragment BIN_DIGITS : BINARY_DIGIT ('_'? BINARY_DIGIT)* ;

fragment DIGIT: [0-9];
fragment HEX_DIGIT: [0-9a-fA-F];
fragment BINARY_DIGIT: [01];

fragment TYPE_SUFFIX
    : 'i8' | 'i16' | 'i32' | 'i64'
    | 'u8' | 'u16' | 'u32' | 'u64'
    | 'f32' | 'f64'
    ;

fragment EXPONENT
    : [eE] [+-]? DEC_DIGITS
    ;

// Temporal and frequency literals must be checked before plain numeric literals
TEMPORAL_LITERAL
    : (DEC_DIGITS | (DEC_DIGITS '.' DEC_DIGITS?) | ('.' DEC_DIGITS)) ('ns' | 'us' | 'ms' | 's' | 'm' | 'h')
    ;

FREQUENCY_LITERAL
    : (DEC_DIGITS | (DEC_DIGITS '.' DEC_DIGITS?) | ('.' DEC_DIGITS)) ([hH][zZ] | [kK][hH][zZ] | [mM][hH][zZ])
    ;

// Numeric literals with optional suffixes and separators
INTEGER_LITERAL
    : DEC_DIGITS TYPE_SUFFIX?
    | '0' [xX] HEX_DIGITS TYPE_SUFFIX?
    | '0' [bB] BIN_DIGITS TYPE_SUFFIX?
    ;

FLOAT_LITERAL
    : (DEC_DIGITS '.' DEC_DIGITS? | '.' DEC_DIGITS) EXPONENT? TYPE_SUFFIX?
    | DEC_DIGITS EXPONENT TYPE_SUFFIX?
    ;

// String literal
STRING_LITERAL
    : '"' (~["\\\r\n] | ESCAPE_SEQUENCE)* '"'
    ;

fragment ESCAPE_SEQUENCE
    : '\\' [btnfr"\\]
    | '\\u' HEX_DIGIT HEX_DIGIT HEX_DIGIT HEX_DIGIT
    ;

// =============================================================================
// Identifiers
// =============================================================================

IDENTIFIER  : [a-zA-Z_] [a-zA-Z0-9_]*;

// =============================================================================
// Comments & Whitespace
// =============================================================================

// Single-line comment
SINGLE_LINE_COMMENT: '//' ~[\r\n]* -> skip;

// Multi-line comment
MULTI_LINE_COMMENT: '/*' .*? '*/' -> skip;

// Whitespace
WS          : [ \t\r\n]+ -> skip;
