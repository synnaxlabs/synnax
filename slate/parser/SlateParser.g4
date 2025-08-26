parser grammar SlateParser;

options {
    tokenVocab = SlateLexer;
}

// =============================================================================
// Entry Point
// =============================================================================

program     : topLevelStatement* EOF;

topLevelStatement
            : functionDecl
            | reactiveBinding
            ;

// =============================================================================
// Reactive Bindings (Top-level only)
// =============================================================================

reactiveBinding
            : channelList CHANNEL_SEND functionCall          // [ch1, ch2] -> func()
            | IDENTIFIER CHANNEL_SEND functionCall           // ch -> func()  
            | intervalBinding
            ;

intervalBinding
            : INTERVAL LPAREN NUMBER_LITERAL RPAREN CHANNEL_SEND functionCall
            ;

channelList : LBRACKET IDENTIFIER (COMMA IDENTIFIER)* RBRACKET;

// =============================================================================
// Functions
// =============================================================================

functionDecl: FUNC IDENTIFIER LPAREN parameterList? RPAREN returnType? block;

parameterList: parameter (COMMA parameter)*;

parameter   : IDENTIFIER type;

type        : NUMBER
            | BOOL  
            | VOID
            | CHAN
            | CHANNEL_RECV CHAN     // <-chan
            | CHANNEL_SEND CHAN     // ->chan
            ;

returnType  : type;

// =============================================================================  
// Statements (Function body only)
// =============================================================================

statement   : variableDecl
            | assignment
            | channelWrite
            | ifStatement
            | returnStatement
            | expressionStatement
            ;

variableDecl: IDENTIFIER LOCAL_ASSIGN expression      // x := value
            | IDENTIFIER STATE_ASSIGN expression      // x $= value
            ;

assignment  : IDENTIFIER ASSIGN expression;          // x = value

channelWrite: expression CHANNEL_SEND IDENTIFIER     // value -> channel
            | IDENTIFIER CHANNEL_RECV expression     // channel <- value  
            ;

channelRead : CHANNEL_RECV IDENTIFIER;               // <- channel

ifStatement : IF LPAREN expression RPAREN statement (ELSE statement)?;

returnStatement: RETURN expression?;

expressionStatement: expression;

block       : LBRACE statement* RBRACE;

// =============================================================================
// Expressions (with precedence)  
// =============================================================================

expression  : logicalOrExpr;

logicalOrExpr: logicalAndExpr (OR logicalAndExpr)*;

logicalAndExpr: equalityExpr (AND equalityExpr)*;

equalityExpr: relationalExpr ((EQUAL | NOT_EQUAL) relationalExpr)*;

relationalExpr: additiveExpr ((LESS_THAN | LESS_EQUAL | GREATER_THAN | GREATER_EQUAL) additiveExpr)*;

additiveExpr: multiplicativeExpr ((PLUS | MINUS) multiplicativeExpr)*;

multiplicativeExpr: unaryExpr ((MULTIPLY | DIVIDE) unaryExpr)*;

unaryExpr   : (NOT | MINUS) unaryExpr
            | primaryExpr
            ;

primaryExpr : NUMBER_LITERAL
            | STRING
            | TRUE
            | FALSE
            | IDENTIFIER
            | channelRead
            | functionCall
            | LPAREN expression RPAREN
            ;

functionCall: IDENTIFIER LPAREN argumentList? RPAREN;

argumentList: expression (COMMA expression)*;