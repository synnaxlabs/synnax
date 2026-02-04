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
    | flowStatement
    | sequenceDeclaration
    | globalConstant
    ;

// =============================================================================
// Function Declarations
// =============================================================================

functionDeclaration
    : FUNC IDENTIFIER configBlock? LPAREN inputList? RPAREN outputType? block
    ;

inputList
    : input (COMMA input)* COMMA?
    ;

input
    : IDENTIFIER type (ASSIGN literal)?
    ;

outputType
    : type                                          // Unnamed single output: f64
    | IDENTIFIER type                               // Named single output without parens: result f64
    | multiOutputBlock                              // Multiple or single named outputs with parens
    ;

multiOutputBlock
    : LPAREN namedOutput (COMMA namedOutput)* COMMA? RPAREN
    ;

namedOutput
    : IDENTIFIER type
    ;

configBlock
    : LBRACE configList? RBRACE
    ;

configList
    : config (COMMA config)* COMMA?
    ;

config
    : IDENTIFIER type
    ;

// =============================================================================
// Sequence and Stage Declarations
// =============================================================================

// sequence main { stage precheck { } stage pressurization { } }
sequenceDeclaration
    : SEQUENCE IDENTIFIER LBRACE stageDeclaration* RBRACE
    ;

// stage precheck { items... }
stageDeclaration
    : STAGE IDENTIFIER stageBody
    ;

// { reactive flows and transitions, comma-separated }
stageBody
    : LBRACE (stageItem (COMMA stageItem)* COMMA?)? RBRACE
    ;

stageItem
    : flowStatement
    | singleInvocation
    ;

singleInvocation
    : function
    | expression
    ;

// =============================================================================
// Global Constants
// =============================================================================

// Top-level variable declarations are compile-time constants.
// Only literals are allowed (no expressions), and stateful declarations ($=) are prohibited.
globalConstant
    : IDENTIFIER DECLARE literal
    | IDENTIFIER type DECLARE literal
    ;

// =============================================================================
// Inter-Stage Flow
// =============================================================================

flowStatement
    : (routingTable | flowNode) (flowOperator (routingTable | flowNode))+
    ;

flowOperator
    : ARROW        // -> (continuous flow)
    | TRANSITION   // => (one-shot flow)
    ;

routingTable
    : LBRACE routingEntry (COMMA routingEntry)* COMMA? RBRACE
    ;

routingEntry
    : IDENTIFIER COLON flowNode (ARROW flowNode)* (COLON IDENTIFIER)?
    ;

flowNode
    : identifier        // Channel, stage, or sequence name - resolved in analysis
    | function
    | expression
    | NEXT              // Continue to next stage
    ;

identifier
    : IDENTIFIER
    ;

function
    : IDENTIFIER configValues
    ;

configValues
    : LBRACE RBRACE                       // Empty config
    | LBRACE namedConfigValues RBRACE     // All named
    | LBRACE anonymousConfigValues RBRACE // All anonymous
    ;

namedConfigValues
    : namedConfigValue (COMMA namedConfigValue)* COMMA?
    ;

namedConfigValue
    : IDENTIFIER ASSIGN expression
    ;

anonymousConfigValues
    : expression (COMMA expression)* COMMA?
    ;

arguments
    : LPAREN argumentList? RPAREN
    ;

argumentList
    : expression (COMMA expression)* COMMA?
    ;

// =============================================================================
// Blocks and Statements
// =============================================================================

block
    : LBRACE statement* RBRACE
    ;

statement
    : variableDeclaration
    | assignment
    | ifStatement
    | returnStatement
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
    | IDENTIFIER indexOrSlice ASSIGN expression
    | IDENTIFIER compoundOp expression
    | IDENTIFIER indexOrSlice compoundOp expression
    ;

compoundOp
    : PLUS_ASSIGN
    | MINUS_ASSIGN
    | STAR_ASSIGN
    | SLASH_ASSIGN
    | PERCENT_ASSIGN
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

// =============================================================================
// Types
// =============================================================================

type
    : primitiveType unitSuffix?
    | channelType
    | seriesType
    ;

unitSuffix
    : IDENTIFIER
    ;

primitiveType
    : numericType
    | STR
    ;

numericType
    : integerType
    | floatType
    ;

integerType
    : I8 | I16 | I32 | I64
    | U8 | U16 | U32 | U64
    ;

floatType
    : F32 | F64
    ;

channelType
    : CHAN primitiveType unitSuffix?
    | CHAN seriesType
    ;

seriesType
    : SERIES primitiveType unitSuffix?
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
    | postfixExpression
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
    ;

typeCast
    : type LPAREN expression RPAREN
    ;

// =============================================================================
// Literals
// =============================================================================

literal
    : numericLiteral
    | STR_LITERAL
    | seriesLiteral
    ;

// Numeric literal with optional unit suffix.
// The unit suffix (IDENTIFIER) is only consumed if it immediately follows
// the number with no whitespace. This is checked via semantic predicate.
numericLiteral
    : (INTEGER_LITERAL | FLOAT_LITERAL)
      ({p.TokensAdjacent(p.GetTokenStream().LT(-1), p.GetTokenStream().LT(1))}? IDENTIFIER)?
    ;

seriesLiteral
    : LBRACKET expressionList? RBRACKET
    ;

expressionList
    : expression (COMMA expression)* COMMA?
    ;
