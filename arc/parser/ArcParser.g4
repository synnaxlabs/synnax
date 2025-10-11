parser grammar ArcParser;

options {
    tokenVocab = ArcLexer;
}

// =============================================================================
// Entry Point
// =============================================================================

program
    : topLevelItem* EOF
    ;

topLevelItem
    : functionDeclaration
    | stageDeclaration
    | flowStatement
    ;

// =============================================================================
// Function Declarations
// =============================================================================

functionDeclaration
    : FUNC IDENTIFIER LPAREN parameterList? RPAREN returnType? block
    ;

parameterList
    : parameter (COMMA parameter)*
    ;

parameter
    : IDENTIFIER type
    ;

returnType
    : type
    ;

// =============================================================================
// Stage Declarations
// =============================================================================

stageDeclaration
    : STAGE IDENTIFIER configBlock? LPAREN parameterList? RPAREN returnType? block
    ;

configBlock
    : LBRACE configParameter* RBRACE
    ;

configParameter
    : IDENTIFIER type
    ;

// =============================================================================
// Inter-Stage Flow
// =============================================================================

flowStatement
    : flowNode (ARROW flowNode)+ SEMICOLON?
    ;

flowNode
    : channelIdentifier
    | stageInvocation
    | expression
    ;

channelIdentifier
    : IDENTIFIER
    ;

stageInvocation
    : IDENTIFIER configValues? arguments?
    ;

configValues
    : LBRACE RBRACE                       // Empty config
    | LBRACE namedConfigValues RBRACE     // All named
    | LBRACE anonymousConfigValues RBRACE // All anonymous
    ;

namedConfigValues
    : namedConfigValue (COMMA namedConfigValue)*
    ;

namedConfigValue
    : IDENTIFIER COLON expression
    ;

anonymousConfigValues
    : expression (COMMA expression)*
    ;

arguments
    : LPAREN argumentList? RPAREN
    ;

argumentList
    : expression (COMMA expression)*
    ;

// =============================================================================
// Blocks and Statements
// =============================================================================

block
    : LBRACE statement* RBRACE
    ;

statement
    : variableDeclaration
    | channelOperation
    | assignment
    | ifStatement
    | returnStatement
    | functionCall
    | expression
    ;

variableDeclaration
    : localVariable
    | statefulVariable
    ;

localVariable
    : IDENTIFIER DECLARE expression
    | IDENTIFIER type DECLARE expression
    ;

statefulVariable
    : IDENTIFIER STATE_DECLARE expression
    | IDENTIFIER type STATE_DECLARE expression
    ;

assignment
    : IDENTIFIER ASSIGN expression
    ;

channelOperation
    : channelWrite
    | channelRead
    ;

channelWrite
    : expression ARROW IDENTIFIER
    | IDENTIFIER RECV expression
    ;

channelRead
    : blockingRead
    | nonBlockingRead
    ;

blockingRead
    : IDENTIFIER DECLARE RECV IDENTIFIER
    ;

nonBlockingRead
    : IDENTIFIER DECLARE IDENTIFIER
    ;

ifStatement
    : IF expression block elseIfClause* elseClause?
    ;

elseIfClause
    : ELSE IF expression block
    ;

elseClause
    : ELSE block
    ;

returnStatement
    : RETURN expression?
    ;

functionCall
    : IDENTIFIER LPAREN argumentList? RPAREN
    ;

// =============================================================================
// Types
// =============================================================================

type
    : primitiveType
    | channelType
    | seriesType
    ;

primitiveType
    : numericType
    | STRING
    ;

numericType
    : integerType
    | floatType
    | temporalType
    ;

integerType
    : I8 | I16 | I32 | I64
    | U8 | U16 | U32 | U64
    ;

floatType
    : F32 | F64
    ;

temporalType
    : TIMESTAMP | TIMESPAN
    ;

channelType
    : (CHAN | RECV_CHAN | SEND_CHAN) (primitiveType | seriesType)
    ;

seriesType
    : SERIES primitiveType
    ;

// =============================================================================
// Expressions
// =============================================================================

expression
    : logicalOrExpression
    ;

logicalOrExpression
    : logicalAndExpression (OR logicalAndExpression)*
    ;

logicalAndExpression
    : equalityExpression (AND equalityExpression)*
    ;

equalityExpression
    : relationalExpression ((EQ | NEQ) relationalExpression)*
    ;

relationalExpression
    : additiveExpression ((LT | GT | LEQ | GEQ) additiveExpression)*
    ;

additiveExpression
    : multiplicativeExpression ((PLUS | MINUS) multiplicativeExpression)*
    ;

multiplicativeExpression
    : powerExpression ((STAR | SLASH | PERCENT) powerExpression)*
    ;

// ^ is right-associative and binds tighter than unary
powerExpression
    : unaryExpression (CARET powerExpression)?
    ;

unaryExpression
    : MINUS unaryExpression
    | NOT unaryExpression
    | blockingReadExpr
    | postfixExpression
    ;

// Blocking read as a true unary operator
blockingReadExpr
    : RECV IDENTIFIER
    ;

postfixExpression
    : primaryExpression (indexOrSlice | functionCallSuffix)*
    ;

indexOrSlice
    : LBRACKET expression RBRACKET                           // Index
    | LBRACKET expression? COLON expression? RBRACKET        // Slice
    ;

functionCallSuffix
    : LPAREN argumentList? RPAREN
    ;

primaryExpression
    : literal
    | IDENTIFIER
    | LPAREN expression RPAREN
    | typeCast
    | builtinFunction
    ;

typeCast
    : type LPAREN expression RPAREN
    ;

builtinFunction
    : LEN LPAREN expression RPAREN
    | NOW LPAREN RPAREN
    ;

// =============================================================================
// Literals
// =============================================================================

literal
    : numericLiteral
    | temporalLiteral
    | STRING_LITERAL
    | seriesLiteral
    ;

numericLiteral
    : INTEGER_LITERAL
    | FLOAT_LITERAL
    ;

temporalLiteral
    : TEMPORAL_LITERAL
    | FREQUENCY_LITERAL
    ;

seriesLiteral
    : LBRACKET expressionList? RBRACKET
    ;

expressionList
    : expression (COMMA expression)*
    ;
