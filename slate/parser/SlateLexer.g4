lexer grammar SlateLexer;

// =============================================================================
// Keywords
// =============================================================================

FUNC        : 'func';
IF          : 'if';
ELSE        : 'else';
RETURN      : 'return';
INTERVAL    : 'interval';

// Built-in types
NUMBER      : 'number';
CHAN        : 'chan';
VOID        : 'void';
BOOL        : 'bool';

// Built-in constants  
TRUE        : 'true';
FALSE       : 'false';

// =============================================================================
// Operators
// =============================================================================

// Channel operators (order matters - longer tokens first)
CHANNEL_SEND    : '->';
CHANNEL_RECV    : '<-';

// Assignment operators
LOCAL_ASSIGN    : ':=';
STATE_ASSIGN    : '$=';
ASSIGN          : '=';

// Arithmetic
PLUS        : '+';
MINUS       : '-';
MULTIPLY    : '*';
DIVIDE      : '/';

// Comparison  
EQUAL       : '==';
NOT_EQUAL   : '!=';
LESS_THAN   : '<';
LESS_EQUAL  : '<=';
GREATER_THAN: '>';
GREATER_EQUAL: '>=';

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

// =============================================================================
// Literals
// =============================================================================

// Numeric literal (simplified - just numbers)
NUMBER_LITERAL : [0-9]+ ('.' [0-9]+)?;

// String literal
STRING      : '"' (~["\\\r\n] | '\\' .)* '"';

// =============================================================================
// Identifiers
// =============================================================================

IDENTIFIER  : [a-zA-Z_] [a-zA-Z0-9_]*;

// =============================================================================
// Comments & Whitespace
// =============================================================================

// Single-line comment
LINE_COMMENT: '//' ~[\r\n]* -> skip;

// Multi-line comment  
BLOCK_COMMENT: '/*' .*? '*/' -> skip;

// Whitespace
WS          : [ \t\r\n]+ -> skip;