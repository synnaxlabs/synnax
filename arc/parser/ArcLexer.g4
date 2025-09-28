lexer grammar ArcLexer;

// =============================================================================
// Keywords
// =============================================================================

FUNC        : 'func';
STAGE        : 'stage';
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

// Simple digit sequences without separators or suffixes
fragment DIGITS : DIGIT+ ;

fragment DIGIT: [0-9];

// Temporal and frequency literals must be checked before plain numeric literals
TEMPORAL_LITERAL
    : (DIGITS | (DIGITS '.' DIGITS?) | ('.' DIGITS)) ('ns' | 'us' | 'ms' | 's' | 'm' | 'h')
    ;

FREQUENCY_LITERAL
    : (DIGITS | (DIGITS '.' DIGITS?) | ('.' DIGITS)) ([hH][zZ] | [kK][hH][zZ] | [mM][hH][zZ])
    ;

// Simple numeric literals without suffixes or special formats
INTEGER_LITERAL
    : DIGITS
    ;

FLOAT_LITERAL
    : DIGITS '.' DIGITS?
    | '.' DIGITS
    ;

// String literal
STRING_LITERAL
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
SINGLE_LINE_COMMENT: '//' ~[\r\n]* -> skip;

// Multi-line comment
MULTI_LINE_COMMENT: '/*' .*? '*/' -> skip;

// Whitespace
WS          : [ \t\r\n]+ -> skip;
