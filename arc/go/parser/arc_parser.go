// Code generated from ArcParser.g4 by ANTLR 4.13.2. DO NOT EDIT.

package parser // ArcParser
import (
	"fmt"
	"strconv"
	"sync"

	"github.com/antlr4-go/antlr/v4"
)

// Suppress unused import errors
var _ = fmt.Printf
var _ = strconv.Itoa
var _ = sync.Once{}

type ArcParser struct {
	*antlr.BaseParser
}

var ArcParserParserStaticData struct {
	once                   sync.Once
	serializedATN          []int32
	LiteralNames           []string
	SymbolicNames          []string
	RuleNames              []string
	PredictionContextCache *antlr.PredictionContextCache
	atn                    *antlr.ATN
	decisionToDFA          []*antlr.DFA
}

func arcparserParserInit() {
	staticData := &ArcParserParserStaticData
	staticData.LiteralNames = []string{
		"", "'func'", "'if'", "'else'", "'return'", "'now'", "'len'", "'sequence'",
		"'stage'", "'match'", "'next'", "'chan'", "'<-chan'", "'->chan'", "'i8'",
		"'i16'", "'i32'", "'i64'", "'u8'", "'u16'", "'u32'", "'u64'", "'f32'",
		"'f64'", "'str'", "'timestamp'", "'timespan'", "'series'", "'->'", "'<-'",
		"':='", "'$='", "'=>'", "'='", "'+'", "'-'", "'*'", "'/'", "'%'", "'^'",
		"'=='", "'!='", "'<'", "'>'", "'<='", "'>='", "'and'", "'or'", "'not'",
		"'('", "')'", "'{'", "'}'", "'['", "']'", "','", "':'", "';'",
	}
	staticData.SymbolicNames = []string{
		"", "FUNC", "IF", "ELSE", "RETURN", "NOW", "LEN", "SEQUENCE", "STAGE",
		"MATCH", "NEXT", "CHAN", "RECV_CHAN", "SEND_CHAN", "I8", "I16", "I32",
		"I64", "U8", "U16", "U32", "U64", "F32", "F64", "STR", "TIMESTAMP",
		"TIMESPAN", "SERIES", "ARROW", "RECV", "DECLARE", "STATE_DECLARE", "TRANSITION",
		"ASSIGN", "PLUS", "MINUS", "STAR", "SLASH", "PERCENT", "CARET", "EQ",
		"NEQ", "LT", "GT", "LEQ", "GEQ", "AND", "OR", "NOT", "LPAREN", "RPAREN",
		"LBRACE", "RBRACE", "LBRACKET", "RBRACKET", "COMMA", "COLON", "SEMICOLON",
		"TEMPORAL_LITERAL", "FREQUENCY_LITERAL", "INTEGER_LITERAL", "FLOAT_LITERAL",
		"STR_LITERAL", "IDENTIFIER", "SINGLE_LINE_COMMENT", "MULTI_LINE_COMMENT",
		"WS",
	}
	staticData.RuleNames = []string{
		"program", "topLevelItem", "functionDeclaration", "inputList", "input",
		"outputType", "multiOutputBlock", "namedOutput", "configBlock", "config",
		"sequenceDeclaration", "sequenceChain", "sequenceEntry", "stageDeclaration",
		"stageBody", "stageItem", "stageFlow", "transitionStatement", "transitionTarget",
		"matchBlock", "matchEntry", "imperativeTransition", "flowStatement",
		"routingTable", "routingEntry", "flowNode", "channelIdentifier", "function",
		"configValues", "namedConfigValues", "namedConfigValue", "anonymousConfigValues",
		"arguments", "argumentList", "block", "statement", "variableDeclaration",
		"localVariable", "statefulVariable", "assignment", "channelOperation",
		"channelWrite", "channelRead", "blockingRead", "nonBlockingRead", "ifStatement",
		"elseIfClause", "elseClause", "returnStatement", "functionCall", "type",
		"primitiveType", "numericType", "integerType", "floatType", "temporalType",
		"channelType", "seriesType", "expression", "logicalOrExpression", "logicalAndExpression",
		"equalityExpression", "relationalExpression", "additiveExpression",
		"multiplicativeExpression", "powerExpression", "unaryExpression", "blockingReadExpr",
		"postfixExpression", "indexOrSlice", "functionCallSuffix", "primaryExpression",
		"typeCast", "builtinFunction", "literal", "numericLiteral", "temporalLiteral",
		"seriesLiteral", "expressionList",
	}
	staticData.PredictionContextCache = antlr.NewPredictionContextCache()
	staticData.serializedATN = []int32{
		4, 1, 66, 703, 2, 0, 7, 0, 2, 1, 7, 1, 2, 2, 7, 2, 2, 3, 7, 3, 2, 4, 7,
		4, 2, 5, 7, 5, 2, 6, 7, 6, 2, 7, 7, 7, 2, 8, 7, 8, 2, 9, 7, 9, 2, 10, 7,
		10, 2, 11, 7, 11, 2, 12, 7, 12, 2, 13, 7, 13, 2, 14, 7, 14, 2, 15, 7, 15,
		2, 16, 7, 16, 2, 17, 7, 17, 2, 18, 7, 18, 2, 19, 7, 19, 2, 20, 7, 20, 2,
		21, 7, 21, 2, 22, 7, 22, 2, 23, 7, 23, 2, 24, 7, 24, 2, 25, 7, 25, 2, 26,
		7, 26, 2, 27, 7, 27, 2, 28, 7, 28, 2, 29, 7, 29, 2, 30, 7, 30, 2, 31, 7,
		31, 2, 32, 7, 32, 2, 33, 7, 33, 2, 34, 7, 34, 2, 35, 7, 35, 2, 36, 7, 36,
		2, 37, 7, 37, 2, 38, 7, 38, 2, 39, 7, 39, 2, 40, 7, 40, 2, 41, 7, 41, 2,
		42, 7, 42, 2, 43, 7, 43, 2, 44, 7, 44, 2, 45, 7, 45, 2, 46, 7, 46, 2, 47,
		7, 47, 2, 48, 7, 48, 2, 49, 7, 49, 2, 50, 7, 50, 2, 51, 7, 51, 2, 52, 7,
		52, 2, 53, 7, 53, 2, 54, 7, 54, 2, 55, 7, 55, 2, 56, 7, 56, 2, 57, 7, 57,
		2, 58, 7, 58, 2, 59, 7, 59, 2, 60, 7, 60, 2, 61, 7, 61, 2, 62, 7, 62, 2,
		63, 7, 63, 2, 64, 7, 64, 2, 65, 7, 65, 2, 66, 7, 66, 2, 67, 7, 67, 2, 68,
		7, 68, 2, 69, 7, 69, 2, 70, 7, 70, 2, 71, 7, 71, 2, 72, 7, 72, 2, 73, 7,
		73, 2, 74, 7, 74, 2, 75, 7, 75, 2, 76, 7, 76, 2, 77, 7, 77, 2, 78, 7, 78,
		1, 0, 5, 0, 160, 8, 0, 10, 0, 12, 0, 163, 9, 0, 1, 0, 1, 0, 1, 1, 1, 1,
		1, 1, 1, 1, 3, 1, 171, 8, 1, 1, 2, 1, 2, 1, 2, 3, 2, 176, 8, 2, 1, 2, 1,
		2, 3, 2, 180, 8, 2, 1, 2, 1, 2, 3, 2, 184, 8, 2, 1, 2, 1, 2, 1, 3, 1, 3,
		1, 3, 5, 3, 191, 8, 3, 10, 3, 12, 3, 194, 9, 3, 1, 4, 1, 4, 1, 4, 1, 4,
		3, 4, 200, 8, 4, 1, 5, 1, 5, 1, 5, 1, 5, 3, 5, 206, 8, 5, 1, 6, 1, 6, 1,
		6, 1, 6, 5, 6, 212, 8, 6, 10, 6, 12, 6, 215, 9, 6, 1, 6, 1, 6, 1, 7, 1,
		7, 1, 7, 1, 8, 1, 8, 5, 8, 224, 8, 8, 10, 8, 12, 8, 227, 9, 8, 1, 8, 1,
		8, 1, 9, 1, 9, 1, 9, 1, 10, 1, 10, 1, 10, 1, 10, 1, 10, 1, 10, 5, 10, 240,
		8, 10, 10, 10, 12, 10, 243, 9, 10, 1, 10, 1, 10, 1, 11, 1, 11, 1, 11, 5,
		11, 250, 8, 11, 10, 11, 12, 11, 253, 9, 11, 1, 12, 1, 12, 1, 12, 1, 12,
		1, 12, 1, 12, 1, 12, 3, 12, 262, 8, 12, 1, 13, 1, 13, 1, 13, 1, 13, 1,
		14, 1, 14, 5, 14, 270, 8, 14, 10, 14, 12, 14, 273, 9, 14, 1, 14, 1, 14,
		1, 15, 1, 15, 1, 15, 3, 15, 280, 8, 15, 1, 16, 1, 16, 3, 16, 284, 8, 16,
		1, 16, 1, 16, 1, 16, 3, 16, 289, 8, 16, 4, 16, 291, 8, 16, 11, 16, 12,
		16, 292, 1, 16, 3, 16, 296, 8, 16, 1, 17, 1, 17, 1, 17, 1, 17, 1, 18, 1,
		18, 1, 18, 3, 18, 305, 8, 18, 1, 19, 1, 19, 1, 19, 1, 19, 1, 19, 5, 19,
		312, 8, 19, 10, 19, 12, 19, 315, 9, 19, 1, 19, 1, 19, 1, 20, 1, 20, 1,
		20, 1, 20, 1, 21, 1, 21, 1, 21, 1, 21, 1, 22, 1, 22, 3, 22, 329, 8, 22,
		1, 22, 1, 22, 1, 22, 3, 22, 334, 8, 22, 4, 22, 336, 8, 22, 11, 22, 12,
		22, 337, 1, 22, 3, 22, 341, 8, 22, 1, 23, 1, 23, 1, 23, 1, 23, 5, 23, 347,
		8, 23, 10, 23, 12, 23, 350, 9, 23, 1, 23, 1, 23, 1, 24, 1, 24, 1, 24, 1,
		24, 1, 24, 5, 24, 359, 8, 24, 10, 24, 12, 24, 362, 9, 24, 1, 24, 1, 24,
		3, 24, 366, 8, 24, 1, 25, 1, 25, 1, 25, 3, 25, 371, 8, 25, 1, 26, 1, 26,
		1, 27, 1, 27, 3, 27, 377, 8, 27, 1, 27, 3, 27, 380, 8, 27, 1, 28, 1, 28,
		1, 28, 1, 28, 1, 28, 1, 28, 1, 28, 1, 28, 1, 28, 1, 28, 3, 28, 392, 8,
		28, 1, 29, 1, 29, 1, 29, 5, 29, 397, 8, 29, 10, 29, 12, 29, 400, 9, 29,
		1, 30, 1, 30, 1, 30, 1, 30, 1, 31, 1, 31, 1, 31, 5, 31, 409, 8, 31, 10,
		31, 12, 31, 412, 9, 31, 1, 32, 1, 32, 3, 32, 416, 8, 32, 1, 32, 1, 32,
		1, 33, 1, 33, 1, 33, 5, 33, 423, 8, 33, 10, 33, 12, 33, 426, 9, 33, 1,
		34, 1, 34, 5, 34, 430, 8, 34, 10, 34, 12, 34, 433, 9, 34, 1, 34, 1, 34,
		1, 35, 1, 35, 1, 35, 1, 35, 1, 35, 1, 35, 1, 35, 3, 35, 444, 8, 35, 1,
		36, 1, 36, 3, 36, 448, 8, 36, 1, 37, 1, 37, 1, 37, 1, 37, 1, 37, 1, 37,
		1, 37, 1, 37, 3, 37, 458, 8, 37, 1, 38, 1, 38, 1, 38, 1, 38, 1, 38, 1,
		38, 1, 38, 1, 38, 3, 38, 468, 8, 38, 1, 39, 1, 39, 1, 39, 1, 39, 1, 40,
		1, 40, 3, 40, 476, 8, 40, 1, 41, 1, 41, 1, 41, 1, 41, 1, 41, 1, 41, 1,
		41, 3, 41, 485, 8, 41, 1, 42, 1, 42, 3, 42, 489, 8, 42, 1, 43, 1, 43, 1,
		43, 1, 43, 1, 43, 1, 44, 1, 44, 1, 44, 1, 44, 1, 45, 1, 45, 1, 45, 1, 45,
		5, 45, 504, 8, 45, 10, 45, 12, 45, 507, 9, 45, 1, 45, 3, 45, 510, 8, 45,
		1, 46, 1, 46, 1, 46, 1, 46, 1, 46, 1, 47, 1, 47, 1, 47, 1, 48, 1, 48, 3,
		48, 522, 8, 48, 1, 49, 1, 49, 1, 49, 3, 49, 527, 8, 49, 1, 49, 1, 49, 1,
		50, 1, 50, 1, 50, 3, 50, 534, 8, 50, 1, 51, 1, 51, 3, 51, 538, 8, 51, 1,
		52, 1, 52, 1, 52, 3, 52, 543, 8, 52, 1, 53, 1, 53, 1, 54, 1, 54, 1, 55,
		1, 55, 1, 56, 1, 56, 1, 56, 3, 56, 554, 8, 56, 1, 57, 1, 57, 1, 57, 1,
		58, 1, 58, 1, 59, 1, 59, 1, 59, 5, 59, 564, 8, 59, 10, 59, 12, 59, 567,
		9, 59, 1, 60, 1, 60, 1, 60, 5, 60, 572, 8, 60, 10, 60, 12, 60, 575, 9,
		60, 1, 61, 1, 61, 1, 61, 5, 61, 580, 8, 61, 10, 61, 12, 61, 583, 9, 61,
		1, 62, 1, 62, 1, 62, 5, 62, 588, 8, 62, 10, 62, 12, 62, 591, 9, 62, 1,
		63, 1, 63, 1, 63, 5, 63, 596, 8, 63, 10, 63, 12, 63, 599, 9, 63, 1, 64,
		1, 64, 1, 64, 5, 64, 604, 8, 64, 10, 64, 12, 64, 607, 9, 64, 1, 65, 1,
		65, 1, 65, 3, 65, 612, 8, 65, 1, 66, 1, 66, 1, 66, 1, 66, 1, 66, 1, 66,
		3, 66, 620, 8, 66, 1, 67, 1, 67, 1, 67, 1, 68, 1, 68, 1, 68, 5, 68, 628,
		8, 68, 10, 68, 12, 68, 631, 9, 68, 1, 69, 1, 69, 1, 69, 1, 69, 1, 69, 1,
		69, 3, 69, 639, 8, 69, 1, 69, 1, 69, 3, 69, 643, 8, 69, 1, 69, 3, 69, 646,
		8, 69, 1, 70, 1, 70, 3, 70, 650, 8, 70, 1, 70, 1, 70, 1, 71, 1, 71, 1,
		71, 1, 71, 1, 71, 1, 71, 1, 71, 1, 71, 3, 71, 662, 8, 71, 1, 72, 1, 72,
		1, 72, 1, 72, 1, 72, 1, 73, 1, 73, 1, 73, 1, 73, 1, 73, 1, 73, 1, 73, 1,
		73, 3, 73, 677, 8, 73, 1, 74, 1, 74, 1, 74, 1, 74, 3, 74, 683, 8, 74, 1,
		75, 1, 75, 1, 76, 1, 76, 1, 77, 1, 77, 3, 77, 691, 8, 77, 1, 77, 1, 77,
		1, 78, 1, 78, 1, 78, 5, 78, 698, 8, 78, 10, 78, 12, 78, 701, 9, 78, 1,
		78, 0, 0, 79, 0, 2, 4, 6, 8, 10, 12, 14, 16, 18, 20, 22, 24, 26, 28, 30,
		32, 34, 36, 38, 40, 42, 44, 46, 48, 50, 52, 54, 56, 58, 60, 62, 64, 66,
		68, 70, 72, 74, 76, 78, 80, 82, 84, 86, 88, 90, 92, 94, 96, 98, 100, 102,
		104, 106, 108, 110, 112, 114, 116, 118, 120, 122, 124, 126, 128, 130, 132,
		134, 136, 138, 140, 142, 144, 146, 148, 150, 152, 154, 156, 0, 10, 1, 0,
		14, 21, 1, 0, 22, 23, 1, 0, 25, 26, 1, 0, 11, 13, 1, 0, 40, 41, 1, 0, 42,
		45, 1, 0, 34, 35, 1, 0, 36, 38, 1, 0, 60, 61, 1, 0, 58, 59, 716, 0, 161,
		1, 0, 0, 0, 2, 170, 1, 0, 0, 0, 4, 172, 1, 0, 0, 0, 6, 187, 1, 0, 0, 0,
		8, 195, 1, 0, 0, 0, 10, 205, 1, 0, 0, 0, 12, 207, 1, 0, 0, 0, 14, 218,
		1, 0, 0, 0, 16, 221, 1, 0, 0, 0, 18, 230, 1, 0, 0, 0, 20, 233, 1, 0, 0,
		0, 22, 246, 1, 0, 0, 0, 24, 261, 1, 0, 0, 0, 26, 263, 1, 0, 0, 0, 28, 267,
		1, 0, 0, 0, 30, 279, 1, 0, 0, 0, 32, 283, 1, 0, 0, 0, 34, 297, 1, 0, 0,
		0, 36, 304, 1, 0, 0, 0, 38, 306, 1, 0, 0, 0, 40, 318, 1, 0, 0, 0, 42, 322,
		1, 0, 0, 0, 44, 328, 1, 0, 0, 0, 46, 342, 1, 0, 0, 0, 48, 353, 1, 0, 0,
		0, 50, 370, 1, 0, 0, 0, 52, 372, 1, 0, 0, 0, 54, 374, 1, 0, 0, 0, 56, 391,
		1, 0, 0, 0, 58, 393, 1, 0, 0, 0, 60, 401, 1, 0, 0, 0, 62, 405, 1, 0, 0,
		0, 64, 413, 1, 0, 0, 0, 66, 419, 1, 0, 0, 0, 68, 427, 1, 0, 0, 0, 70, 443,
		1, 0, 0, 0, 72, 447, 1, 0, 0, 0, 74, 457, 1, 0, 0, 0, 76, 467, 1, 0, 0,
		0, 78, 469, 1, 0, 0, 0, 80, 475, 1, 0, 0, 0, 82, 484, 1, 0, 0, 0, 84, 488,
		1, 0, 0, 0, 86, 490, 1, 0, 0, 0, 88, 495, 1, 0, 0, 0, 90, 499, 1, 0, 0,
		0, 92, 511, 1, 0, 0, 0, 94, 516, 1, 0, 0, 0, 96, 519, 1, 0, 0, 0, 98, 523,
		1, 0, 0, 0, 100, 533, 1, 0, 0, 0, 102, 537, 1, 0, 0, 0, 104, 542, 1, 0,
		0, 0, 106, 544, 1, 0, 0, 0, 108, 546, 1, 0, 0, 0, 110, 548, 1, 0, 0, 0,
		112, 550, 1, 0, 0, 0, 114, 555, 1, 0, 0, 0, 116, 558, 1, 0, 0, 0, 118,
		560, 1, 0, 0, 0, 120, 568, 1, 0, 0, 0, 122, 576, 1, 0, 0, 0, 124, 584,
		1, 0, 0, 0, 126, 592, 1, 0, 0, 0, 128, 600, 1, 0, 0, 0, 130, 608, 1, 0,
		0, 0, 132, 619, 1, 0, 0, 0, 134, 621, 1, 0, 0, 0, 136, 624, 1, 0, 0, 0,
		138, 645, 1, 0, 0, 0, 140, 647, 1, 0, 0, 0, 142, 661, 1, 0, 0, 0, 144,
		663, 1, 0, 0, 0, 146, 676, 1, 0, 0, 0, 148, 682, 1, 0, 0, 0, 150, 684,
		1, 0, 0, 0, 152, 686, 1, 0, 0, 0, 154, 688, 1, 0, 0, 0, 156, 694, 1, 0,
		0, 0, 158, 160, 3, 2, 1, 0, 159, 158, 1, 0, 0, 0, 160, 163, 1, 0, 0, 0,
		161, 159, 1, 0, 0, 0, 161, 162, 1, 0, 0, 0, 162, 164, 1, 0, 0, 0, 163,
		161, 1, 0, 0, 0, 164, 165, 5, 0, 0, 1, 165, 1, 1, 0, 0, 0, 166, 171, 3,
		4, 2, 0, 167, 171, 3, 44, 22, 0, 168, 171, 3, 20, 10, 0, 169, 171, 3, 26,
		13, 0, 170, 166, 1, 0, 0, 0, 170, 167, 1, 0, 0, 0, 170, 168, 1, 0, 0, 0,
		170, 169, 1, 0, 0, 0, 171, 3, 1, 0, 0, 0, 172, 173, 5, 1, 0, 0, 173, 175,
		5, 63, 0, 0, 174, 176, 3, 16, 8, 0, 175, 174, 1, 0, 0, 0, 175, 176, 1,
		0, 0, 0, 176, 177, 1, 0, 0, 0, 177, 179, 5, 49, 0, 0, 178, 180, 3, 6, 3,
		0, 179, 178, 1, 0, 0, 0, 179, 180, 1, 0, 0, 0, 180, 181, 1, 0, 0, 0, 181,
		183, 5, 50, 0, 0, 182, 184, 3, 10, 5, 0, 183, 182, 1, 0, 0, 0, 183, 184,
		1, 0, 0, 0, 184, 185, 1, 0, 0, 0, 185, 186, 3, 68, 34, 0, 186, 5, 1, 0,
		0, 0, 187, 192, 3, 8, 4, 0, 188, 189, 5, 55, 0, 0, 189, 191, 3, 8, 4, 0,
		190, 188, 1, 0, 0, 0, 191, 194, 1, 0, 0, 0, 192, 190, 1, 0, 0, 0, 192,
		193, 1, 0, 0, 0, 193, 7, 1, 0, 0, 0, 194, 192, 1, 0, 0, 0, 195, 196, 5,
		63, 0, 0, 196, 199, 3, 100, 50, 0, 197, 198, 5, 33, 0, 0, 198, 200, 3,
		148, 74, 0, 199, 197, 1, 0, 0, 0, 199, 200, 1, 0, 0, 0, 200, 9, 1, 0, 0,
		0, 201, 206, 3, 100, 50, 0, 202, 203, 5, 63, 0, 0, 203, 206, 3, 100, 50,
		0, 204, 206, 3, 12, 6, 0, 205, 201, 1, 0, 0, 0, 205, 202, 1, 0, 0, 0, 205,
		204, 1, 0, 0, 0, 206, 11, 1, 0, 0, 0, 207, 208, 5, 49, 0, 0, 208, 213,
		3, 14, 7, 0, 209, 210, 5, 55, 0, 0, 210, 212, 3, 14, 7, 0, 211, 209, 1,
		0, 0, 0, 212, 215, 1, 0, 0, 0, 213, 211, 1, 0, 0, 0, 213, 214, 1, 0, 0,
		0, 214, 216, 1, 0, 0, 0, 215, 213, 1, 0, 0, 0, 216, 217, 5, 50, 0, 0, 217,
		13, 1, 0, 0, 0, 218, 219, 5, 63, 0, 0, 219, 220, 3, 100, 50, 0, 220, 15,
		1, 0, 0, 0, 221, 225, 5, 51, 0, 0, 222, 224, 3, 18, 9, 0, 223, 222, 1,
		0, 0, 0, 224, 227, 1, 0, 0, 0, 225, 223, 1, 0, 0, 0, 225, 226, 1, 0, 0,
		0, 226, 228, 1, 0, 0, 0, 227, 225, 1, 0, 0, 0, 228, 229, 5, 52, 0, 0, 229,
		17, 1, 0, 0, 0, 230, 231, 5, 63, 0, 0, 231, 232, 3, 100, 50, 0, 232, 19,
		1, 0, 0, 0, 233, 234, 5, 7, 0, 0, 234, 235, 5, 63, 0, 0, 235, 236, 5, 51,
		0, 0, 236, 241, 3, 22, 11, 0, 237, 238, 5, 55, 0, 0, 238, 240, 3, 22, 11,
		0, 239, 237, 1, 0, 0, 0, 240, 243, 1, 0, 0, 0, 241, 239, 1, 0, 0, 0, 241,
		242, 1, 0, 0, 0, 242, 244, 1, 0, 0, 0, 243, 241, 1, 0, 0, 0, 244, 245,
		5, 52, 0, 0, 245, 21, 1, 0, 0, 0, 246, 251, 3, 24, 12, 0, 247, 248, 5,
		32, 0, 0, 248, 250, 3, 24, 12, 0, 249, 247, 1, 0, 0, 0, 250, 253, 1, 0,
		0, 0, 251, 249, 1, 0, 0, 0, 251, 252, 1, 0, 0, 0, 252, 23, 1, 0, 0, 0,
		253, 251, 1, 0, 0, 0, 254, 255, 5, 63, 0, 0, 255, 256, 5, 56, 0, 0, 256,
		257, 5, 8, 0, 0, 257, 262, 3, 28, 14, 0, 258, 259, 5, 8, 0, 0, 259, 262,
		3, 28, 14, 0, 260, 262, 5, 63, 0, 0, 261, 254, 1, 0, 0, 0, 261, 258, 1,
		0, 0, 0, 261, 260, 1, 0, 0, 0, 262, 25, 1, 0, 0, 0, 263, 264, 5, 8, 0,
		0, 264, 265, 5, 63, 0, 0, 265, 266, 3, 28, 14, 0, 266, 27, 1, 0, 0, 0,
		267, 271, 5, 51, 0, 0, 268, 270, 3, 30, 15, 0, 269, 268, 1, 0, 0, 0, 270,
		273, 1, 0, 0, 0, 271, 269, 1, 0, 0, 0, 271, 272, 1, 0, 0, 0, 272, 274,
		1, 0, 0, 0, 273, 271, 1, 0, 0, 0, 274, 275, 5, 52, 0, 0, 275, 29, 1, 0,
		0, 0, 276, 280, 3, 32, 16, 0, 277, 280, 3, 34, 17, 0, 278, 280, 3, 42,
		21, 0, 279, 276, 1, 0, 0, 0, 279, 277, 1, 0, 0, 0, 279, 278, 1, 0, 0, 0,
		280, 31, 1, 0, 0, 0, 281, 284, 3, 46, 23, 0, 282, 284, 3, 50, 25, 0, 283,
		281, 1, 0, 0, 0, 283, 282, 1, 0, 0, 0, 284, 290, 1, 0, 0, 0, 285, 288,
		5, 28, 0, 0, 286, 289, 3, 46, 23, 0, 287, 289, 3, 50, 25, 0, 288, 286,
		1, 0, 0, 0, 288, 287, 1, 0, 0, 0, 289, 291, 1, 0, 0, 0, 290, 285, 1, 0,
		0, 0, 291, 292, 1, 0, 0, 0, 292, 290, 1, 0, 0, 0, 292, 293, 1, 0, 0, 0,
		293, 295, 1, 0, 0, 0, 294, 296, 5, 57, 0, 0, 295, 294, 1, 0, 0, 0, 295,
		296, 1, 0, 0, 0, 296, 33, 1, 0, 0, 0, 297, 298, 3, 116, 58, 0, 298, 299,
		5, 32, 0, 0, 299, 300, 3, 36, 18, 0, 300, 35, 1, 0, 0, 0, 301, 305, 5,
		10, 0, 0, 302, 305, 3, 38, 19, 0, 303, 305, 5, 63, 0, 0, 304, 301, 1, 0,
		0, 0, 304, 302, 1, 0, 0, 0, 304, 303, 1, 0, 0, 0, 305, 37, 1, 0, 0, 0,
		306, 307, 5, 9, 0, 0, 307, 308, 5, 51, 0, 0, 308, 313, 3, 40, 20, 0, 309,
		310, 5, 55, 0, 0, 310, 312, 3, 40, 20, 0, 311, 309, 1, 0, 0, 0, 312, 315,
		1, 0, 0, 0, 313, 311, 1, 0, 0, 0, 313, 314, 1, 0, 0, 0, 314, 316, 1, 0,
		0, 0, 315, 313, 1, 0, 0, 0, 316, 317, 5, 52, 0, 0, 317, 39, 1, 0, 0, 0,
		318, 319, 5, 63, 0, 0, 319, 320, 5, 32, 0, 0, 320, 321, 3, 36, 18, 0, 321,
		41, 1, 0, 0, 0, 322, 323, 3, 68, 34, 0, 323, 324, 5, 32, 0, 0, 324, 325,
		3, 38, 19, 0, 325, 43, 1, 0, 0, 0, 326, 329, 3, 46, 23, 0, 327, 329, 3,
		50, 25, 0, 328, 326, 1, 0, 0, 0, 328, 327, 1, 0, 0, 0, 329, 335, 1, 0,
		0, 0, 330, 333, 5, 28, 0, 0, 331, 334, 3, 46, 23, 0, 332, 334, 3, 50, 25,
		0, 333, 331, 1, 0, 0, 0, 333, 332, 1, 0, 0, 0, 334, 336, 1, 0, 0, 0, 335,
		330, 1, 0, 0, 0, 336, 337, 1, 0, 0, 0, 337, 335, 1, 0, 0, 0, 337, 338,
		1, 0, 0, 0, 338, 340, 1, 0, 0, 0, 339, 341, 5, 57, 0, 0, 340, 339, 1, 0,
		0, 0, 340, 341, 1, 0, 0, 0, 341, 45, 1, 0, 0, 0, 342, 343, 5, 51, 0, 0,
		343, 348, 3, 48, 24, 0, 344, 345, 5, 55, 0, 0, 345, 347, 3, 48, 24, 0,
		346, 344, 1, 0, 0, 0, 347, 350, 1, 0, 0, 0, 348, 346, 1, 0, 0, 0, 348,
		349, 1, 0, 0, 0, 349, 351, 1, 0, 0, 0, 350, 348, 1, 0, 0, 0, 351, 352,
		5, 52, 0, 0, 352, 47, 1, 0, 0, 0, 353, 354, 5, 63, 0, 0, 354, 355, 5, 56,
		0, 0, 355, 360, 3, 50, 25, 0, 356, 357, 5, 28, 0, 0, 357, 359, 3, 50, 25,
		0, 358, 356, 1, 0, 0, 0, 359, 362, 1, 0, 0, 0, 360, 358, 1, 0, 0, 0, 360,
		361, 1, 0, 0, 0, 361, 365, 1, 0, 0, 0, 362, 360, 1, 0, 0, 0, 363, 364,
		5, 56, 0, 0, 364, 366, 5, 63, 0, 0, 365, 363, 1, 0, 0, 0, 365, 366, 1,
		0, 0, 0, 366, 49, 1, 0, 0, 0, 367, 371, 3, 52, 26, 0, 368, 371, 3, 54,
		27, 0, 369, 371, 3, 116, 58, 0, 370, 367, 1, 0, 0, 0, 370, 368, 1, 0, 0,
		0, 370, 369, 1, 0, 0, 0, 371, 51, 1, 0, 0, 0, 372, 373, 5, 63, 0, 0, 373,
		53, 1, 0, 0, 0, 374, 376, 5, 63, 0, 0, 375, 377, 3, 56, 28, 0, 376, 375,
		1, 0, 0, 0, 376, 377, 1, 0, 0, 0, 377, 379, 1, 0, 0, 0, 378, 380, 3, 64,
		32, 0, 379, 378, 1, 0, 0, 0, 379, 380, 1, 0, 0, 0, 380, 55, 1, 0, 0, 0,
		381, 382, 5, 51, 0, 0, 382, 392, 5, 52, 0, 0, 383, 384, 5, 51, 0, 0, 384,
		385, 3, 58, 29, 0, 385, 386, 5, 52, 0, 0, 386, 392, 1, 0, 0, 0, 387, 388,
		5, 51, 0, 0, 388, 389, 3, 62, 31, 0, 389, 390, 5, 52, 0, 0, 390, 392, 1,
		0, 0, 0, 391, 381, 1, 0, 0, 0, 391, 383, 1, 0, 0, 0, 391, 387, 1, 0, 0,
		0, 392, 57, 1, 0, 0, 0, 393, 398, 3, 60, 30, 0, 394, 395, 5, 55, 0, 0,
		395, 397, 3, 60, 30, 0, 396, 394, 1, 0, 0, 0, 397, 400, 1, 0, 0, 0, 398,
		396, 1, 0, 0, 0, 398, 399, 1, 0, 0, 0, 399, 59, 1, 0, 0, 0, 400, 398, 1,
		0, 0, 0, 401, 402, 5, 63, 0, 0, 402, 403, 5, 33, 0, 0, 403, 404, 3, 116,
		58, 0, 404, 61, 1, 0, 0, 0, 405, 410, 3, 116, 58, 0, 406, 407, 5, 55, 0,
		0, 407, 409, 3, 116, 58, 0, 408, 406, 1, 0, 0, 0, 409, 412, 1, 0, 0, 0,
		410, 408, 1, 0, 0, 0, 410, 411, 1, 0, 0, 0, 411, 63, 1, 0, 0, 0, 412, 410,
		1, 0, 0, 0, 413, 415, 5, 49, 0, 0, 414, 416, 3, 66, 33, 0, 415, 414, 1,
		0, 0, 0, 415, 416, 1, 0, 0, 0, 416, 417, 1, 0, 0, 0, 417, 418, 5, 50, 0,
		0, 418, 65, 1, 0, 0, 0, 419, 424, 3, 116, 58, 0, 420, 421, 5, 55, 0, 0,
		421, 423, 3, 116, 58, 0, 422, 420, 1, 0, 0, 0, 423, 426, 1, 0, 0, 0, 424,
		422, 1, 0, 0, 0, 424, 425, 1, 0, 0, 0, 425, 67, 1, 0, 0, 0, 426, 424, 1,
		0, 0, 0, 427, 431, 5, 51, 0, 0, 428, 430, 3, 70, 35, 0, 429, 428, 1, 0,
		0, 0, 430, 433, 1, 0, 0, 0, 431, 429, 1, 0, 0, 0, 431, 432, 1, 0, 0, 0,
		432, 434, 1, 0, 0, 0, 433, 431, 1, 0, 0, 0, 434, 435, 5, 52, 0, 0, 435,
		69, 1, 0, 0, 0, 436, 444, 3, 72, 36, 0, 437, 444, 3, 80, 40, 0, 438, 444,
		3, 78, 39, 0, 439, 444, 3, 90, 45, 0, 440, 444, 3, 96, 48, 0, 441, 444,
		3, 98, 49, 0, 442, 444, 3, 116, 58, 0, 443, 436, 1, 0, 0, 0, 443, 437,
		1, 0, 0, 0, 443, 438, 1, 0, 0, 0, 443, 439, 1, 0, 0, 0, 443, 440, 1, 0,
		0, 0, 443, 441, 1, 0, 0, 0, 443, 442, 1, 0, 0, 0, 444, 71, 1, 0, 0, 0,
		445, 448, 3, 74, 37, 0, 446, 448, 3, 76, 38, 0, 447, 445, 1, 0, 0, 0, 447,
		446, 1, 0, 0, 0, 448, 73, 1, 0, 0, 0, 449, 450, 5, 63, 0, 0, 450, 451,
		5, 30, 0, 0, 451, 458, 3, 116, 58, 0, 452, 453, 5, 63, 0, 0, 453, 454,
		3, 100, 50, 0, 454, 455, 5, 30, 0, 0, 455, 456, 3, 116, 58, 0, 456, 458,
		1, 0, 0, 0, 457, 449, 1, 0, 0, 0, 457, 452, 1, 0, 0, 0, 458, 75, 1, 0,
		0, 0, 459, 460, 5, 63, 0, 0, 460, 461, 5, 31, 0, 0, 461, 468, 3, 116, 58,
		0, 462, 463, 5, 63, 0, 0, 463, 464, 3, 100, 50, 0, 464, 465, 5, 31, 0,
		0, 465, 466, 3, 116, 58, 0, 466, 468, 1, 0, 0, 0, 467, 459, 1, 0, 0, 0,
		467, 462, 1, 0, 0, 0, 468, 77, 1, 0, 0, 0, 469, 470, 5, 63, 0, 0, 470,
		471, 5, 33, 0, 0, 471, 472, 3, 116, 58, 0, 472, 79, 1, 0, 0, 0, 473, 476,
		3, 82, 41, 0, 474, 476, 3, 84, 42, 0, 475, 473, 1, 0, 0, 0, 475, 474, 1,
		0, 0, 0, 476, 81, 1, 0, 0, 0, 477, 478, 3, 116, 58, 0, 478, 479, 5, 28,
		0, 0, 479, 480, 5, 63, 0, 0, 480, 485, 1, 0, 0, 0, 481, 482, 5, 63, 0,
		0, 482, 483, 5, 29, 0, 0, 483, 485, 3, 116, 58, 0, 484, 477, 1, 0, 0, 0,
		484, 481, 1, 0, 0, 0, 485, 83, 1, 0, 0, 0, 486, 489, 3, 86, 43, 0, 487,
		489, 3, 88, 44, 0, 488, 486, 1, 0, 0, 0, 488, 487, 1, 0, 0, 0, 489, 85,
		1, 0, 0, 0, 490, 491, 5, 63, 0, 0, 491, 492, 5, 30, 0, 0, 492, 493, 5,
		29, 0, 0, 493, 494, 5, 63, 0, 0, 494, 87, 1, 0, 0, 0, 495, 496, 5, 63,
		0, 0, 496, 497, 5, 30, 0, 0, 497, 498, 5, 63, 0, 0, 498, 89, 1, 0, 0, 0,
		499, 500, 5, 2, 0, 0, 500, 501, 3, 116, 58, 0, 501, 505, 3, 68, 34, 0,
		502, 504, 3, 92, 46, 0, 503, 502, 1, 0, 0, 0, 504, 507, 1, 0, 0, 0, 505,
		503, 1, 0, 0, 0, 505, 506, 1, 0, 0, 0, 506, 509, 1, 0, 0, 0, 507, 505,
		1, 0, 0, 0, 508, 510, 3, 94, 47, 0, 509, 508, 1, 0, 0, 0, 509, 510, 1,
		0, 0, 0, 510, 91, 1, 0, 0, 0, 511, 512, 5, 3, 0, 0, 512, 513, 5, 2, 0,
		0, 513, 514, 3, 116, 58, 0, 514, 515, 3, 68, 34, 0, 515, 93, 1, 0, 0, 0,
		516, 517, 5, 3, 0, 0, 517, 518, 3, 68, 34, 0, 518, 95, 1, 0, 0, 0, 519,
		521, 5, 4, 0, 0, 520, 522, 3, 116, 58, 0, 521, 520, 1, 0, 0, 0, 521, 522,
		1, 0, 0, 0, 522, 97, 1, 0, 0, 0, 523, 524, 5, 63, 0, 0, 524, 526, 5, 49,
		0, 0, 525, 527, 3, 66, 33, 0, 526, 525, 1, 0, 0, 0, 526, 527, 1, 0, 0,
		0, 527, 528, 1, 0, 0, 0, 528, 529, 5, 50, 0, 0, 529, 99, 1, 0, 0, 0, 530,
		534, 3, 102, 51, 0, 531, 534, 3, 112, 56, 0, 532, 534, 3, 114, 57, 0, 533,
		530, 1, 0, 0, 0, 533, 531, 1, 0, 0, 0, 533, 532, 1, 0, 0, 0, 534, 101,
		1, 0, 0, 0, 535, 538, 3, 104, 52, 0, 536, 538, 5, 24, 0, 0, 537, 535, 1,
		0, 0, 0, 537, 536, 1, 0, 0, 0, 538, 103, 1, 0, 0, 0, 539, 543, 3, 106,
		53, 0, 540, 543, 3, 108, 54, 0, 541, 543, 3, 110, 55, 0, 542, 539, 1, 0,
		0, 0, 542, 540, 1, 0, 0, 0, 542, 541, 1, 0, 0, 0, 543, 105, 1, 0, 0, 0,
		544, 545, 7, 0, 0, 0, 545, 107, 1, 0, 0, 0, 546, 547, 7, 1, 0, 0, 547,
		109, 1, 0, 0, 0, 548, 549, 7, 2, 0, 0, 549, 111, 1, 0, 0, 0, 550, 553,
		7, 3, 0, 0, 551, 554, 3, 102, 51, 0, 552, 554, 3, 114, 57, 0, 553, 551,
		1, 0, 0, 0, 553, 552, 1, 0, 0, 0, 554, 113, 1, 0, 0, 0, 555, 556, 5, 27,
		0, 0, 556, 557, 3, 102, 51, 0, 557, 115, 1, 0, 0, 0, 558, 559, 3, 118,
		59, 0, 559, 117, 1, 0, 0, 0, 560, 565, 3, 120, 60, 0, 561, 562, 5, 47,
		0, 0, 562, 564, 3, 120, 60, 0, 563, 561, 1, 0, 0, 0, 564, 567, 1, 0, 0,
		0, 565, 563, 1, 0, 0, 0, 565, 566, 1, 0, 0, 0, 566, 119, 1, 0, 0, 0, 567,
		565, 1, 0, 0, 0, 568, 573, 3, 122, 61, 0, 569, 570, 5, 46, 0, 0, 570, 572,
		3, 122, 61, 0, 571, 569, 1, 0, 0, 0, 572, 575, 1, 0, 0, 0, 573, 571, 1,
		0, 0, 0, 573, 574, 1, 0, 0, 0, 574, 121, 1, 0, 0, 0, 575, 573, 1, 0, 0,
		0, 576, 581, 3, 124, 62, 0, 577, 578, 7, 4, 0, 0, 578, 580, 3, 124, 62,
		0, 579, 577, 1, 0, 0, 0, 580, 583, 1, 0, 0, 0, 581, 579, 1, 0, 0, 0, 581,
		582, 1, 0, 0, 0, 582, 123, 1, 0, 0, 0, 583, 581, 1, 0, 0, 0, 584, 589,
		3, 126, 63, 0, 585, 586, 7, 5, 0, 0, 586, 588, 3, 126, 63, 0, 587, 585,
		1, 0, 0, 0, 588, 591, 1, 0, 0, 0, 589, 587, 1, 0, 0, 0, 589, 590, 1, 0,
		0, 0, 590, 125, 1, 0, 0, 0, 591, 589, 1, 0, 0, 0, 592, 597, 3, 128, 64,
		0, 593, 594, 7, 6, 0, 0, 594, 596, 3, 128, 64, 0, 595, 593, 1, 0, 0, 0,
		596, 599, 1, 0, 0, 0, 597, 595, 1, 0, 0, 0, 597, 598, 1, 0, 0, 0, 598,
		127, 1, 0, 0, 0, 599, 597, 1, 0, 0, 0, 600, 605, 3, 130, 65, 0, 601, 602,
		7, 7, 0, 0, 602, 604, 3, 130, 65, 0, 603, 601, 1, 0, 0, 0, 604, 607, 1,
		0, 0, 0, 605, 603, 1, 0, 0, 0, 605, 606, 1, 0, 0, 0, 606, 129, 1, 0, 0,
		0, 607, 605, 1, 0, 0, 0, 608, 611, 3, 132, 66, 0, 609, 610, 5, 39, 0, 0,
		610, 612, 3, 130, 65, 0, 611, 609, 1, 0, 0, 0, 611, 612, 1, 0, 0, 0, 612,
		131, 1, 0, 0, 0, 613, 614, 5, 35, 0, 0, 614, 620, 3, 132, 66, 0, 615, 616,
		5, 48, 0, 0, 616, 620, 3, 132, 66, 0, 617, 620, 3, 134, 67, 0, 618, 620,
		3, 136, 68, 0, 619, 613, 1, 0, 0, 0, 619, 615, 1, 0, 0, 0, 619, 617, 1,
		0, 0, 0, 619, 618, 1, 0, 0, 0, 620, 133, 1, 0, 0, 0, 621, 622, 5, 29, 0,
		0, 622, 623, 5, 63, 0, 0, 623, 135, 1, 0, 0, 0, 624, 629, 3, 142, 71, 0,
		625, 628, 3, 138, 69, 0, 626, 628, 3, 140, 70, 0, 627, 625, 1, 0, 0, 0,
		627, 626, 1, 0, 0, 0, 628, 631, 1, 0, 0, 0, 629, 627, 1, 0, 0, 0, 629,
		630, 1, 0, 0, 0, 630, 137, 1, 0, 0, 0, 631, 629, 1, 0, 0, 0, 632, 633,
		5, 53, 0, 0, 633, 634, 3, 116, 58, 0, 634, 635, 5, 54, 0, 0, 635, 646,
		1, 0, 0, 0, 636, 638, 5, 53, 0, 0, 637, 639, 3, 116, 58, 0, 638, 637, 1,
		0, 0, 0, 638, 639, 1, 0, 0, 0, 639, 640, 1, 0, 0, 0, 640, 642, 5, 56, 0,
		0, 641, 643, 3, 116, 58, 0, 642, 641, 1, 0, 0, 0, 642, 643, 1, 0, 0, 0,
		643, 644, 1, 0, 0, 0, 644, 646, 5, 54, 0, 0, 645, 632, 1, 0, 0, 0, 645,
		636, 1, 0, 0, 0, 646, 139, 1, 0, 0, 0, 647, 649, 5, 49, 0, 0, 648, 650,
		3, 66, 33, 0, 649, 648, 1, 0, 0, 0, 649, 650, 1, 0, 0, 0, 650, 651, 1,
		0, 0, 0, 651, 652, 5, 50, 0, 0, 652, 141, 1, 0, 0, 0, 653, 662, 3, 148,
		74, 0, 654, 662, 5, 63, 0, 0, 655, 656, 5, 49, 0, 0, 656, 657, 3, 116,
		58, 0, 657, 658, 5, 50, 0, 0, 658, 662, 1, 0, 0, 0, 659, 662, 3, 144, 72,
		0, 660, 662, 3, 146, 73, 0, 661, 653, 1, 0, 0, 0, 661, 654, 1, 0, 0, 0,
		661, 655, 1, 0, 0, 0, 661, 659, 1, 0, 0, 0, 661, 660, 1, 0, 0, 0, 662,
		143, 1, 0, 0, 0, 663, 664, 3, 100, 50, 0, 664, 665, 5, 49, 0, 0, 665, 666,
		3, 116, 58, 0, 666, 667, 5, 50, 0, 0, 667, 145, 1, 0, 0, 0, 668, 669, 5,
		6, 0, 0, 669, 670, 5, 49, 0, 0, 670, 671, 3, 116, 58, 0, 671, 672, 5, 50,
		0, 0, 672, 677, 1, 0, 0, 0, 673, 674, 5, 5, 0, 0, 674, 675, 5, 49, 0, 0,
		675, 677, 5, 50, 0, 0, 676, 668, 1, 0, 0, 0, 676, 673, 1, 0, 0, 0, 677,
		147, 1, 0, 0, 0, 678, 683, 3, 150, 75, 0, 679, 683, 3, 152, 76, 0, 680,
		683, 5, 62, 0, 0, 681, 683, 3, 154, 77, 0, 682, 678, 1, 0, 0, 0, 682, 679,
		1, 0, 0, 0, 682, 680, 1, 0, 0, 0, 682, 681, 1, 0, 0, 0, 683, 149, 1, 0,
		0, 0, 684, 685, 7, 8, 0, 0, 685, 151, 1, 0, 0, 0, 686, 687, 7, 9, 0, 0,
		687, 153, 1, 0, 0, 0, 688, 690, 5, 53, 0, 0, 689, 691, 3, 156, 78, 0, 690,
		689, 1, 0, 0, 0, 690, 691, 1, 0, 0, 0, 691, 692, 1, 0, 0, 0, 692, 693,
		5, 54, 0, 0, 693, 155, 1, 0, 0, 0, 694, 699, 3, 116, 58, 0, 695, 696, 5,
		55, 0, 0, 696, 698, 3, 116, 58, 0, 697, 695, 1, 0, 0, 0, 698, 701, 1, 0,
		0, 0, 699, 697, 1, 0, 0, 0, 699, 700, 1, 0, 0, 0, 700, 157, 1, 0, 0, 0,
		701, 699, 1, 0, 0, 0, 71, 161, 170, 175, 179, 183, 192, 199, 205, 213,
		225, 241, 251, 261, 271, 279, 283, 288, 292, 295, 304, 313, 328, 333, 337,
		340, 348, 360, 365, 370, 376, 379, 391, 398, 410, 415, 424, 431, 443, 447,
		457, 467, 475, 484, 488, 505, 509, 521, 526, 533, 537, 542, 553, 565, 573,
		581, 589, 597, 605, 611, 619, 627, 629, 638, 642, 645, 649, 661, 676, 682,
		690, 699,
	}
	deserializer := antlr.NewATNDeserializer(nil)
	staticData.atn = deserializer.Deserialize(staticData.serializedATN)
	atn := staticData.atn
	staticData.decisionToDFA = make([]*antlr.DFA, len(atn.DecisionToState))
	decisionToDFA := staticData.decisionToDFA
	for index, state := range atn.DecisionToState {
		decisionToDFA[index] = antlr.NewDFA(state, index)
	}
}

// ArcParserInit initializes any static state used to implement ArcParser. By default the
// static state used to implement the parser is lazily initialized during the first call to
// NewArcParser(). You can call this function if you wish to initialize the static state ahead
// of time.
func ArcParserInit() {
	staticData := &ArcParserParserStaticData
	staticData.once.Do(arcparserParserInit)
}

// NewArcParser produces a new parser instance for the optional input antlr.TokenStream.
func NewArcParser(input antlr.TokenStream) *ArcParser {
	ArcParserInit()
	this := new(ArcParser)
	this.BaseParser = antlr.NewBaseParser(input)
	staticData := &ArcParserParserStaticData
	this.Interpreter = antlr.NewParserATNSimulator(this, staticData.atn, staticData.decisionToDFA, staticData.PredictionContextCache)
	this.RuleNames = staticData.RuleNames
	this.LiteralNames = staticData.LiteralNames
	this.SymbolicNames = staticData.SymbolicNames
	this.GrammarFileName = "ArcParser.g4"

	return this
}

// ArcParser tokens.
const (
	ArcParserEOF                 = antlr.TokenEOF
	ArcParserFUNC                = 1
	ArcParserIF                  = 2
	ArcParserELSE                = 3
	ArcParserRETURN              = 4
	ArcParserNOW                 = 5
	ArcParserLEN                 = 6
	ArcParserSEQUENCE            = 7
	ArcParserSTAGE               = 8
	ArcParserMATCH               = 9
	ArcParserNEXT                = 10
	ArcParserCHAN                = 11
	ArcParserRECV_CHAN           = 12
	ArcParserSEND_CHAN           = 13
	ArcParserI8                  = 14
	ArcParserI16                 = 15
	ArcParserI32                 = 16
	ArcParserI64                 = 17
	ArcParserU8                  = 18
	ArcParserU16                 = 19
	ArcParserU32                 = 20
	ArcParserU64                 = 21
	ArcParserF32                 = 22
	ArcParserF64                 = 23
	ArcParserSTR                 = 24
	ArcParserTIMESTAMP           = 25
	ArcParserTIMESPAN            = 26
	ArcParserSERIES              = 27
	ArcParserARROW               = 28
	ArcParserRECV                = 29
	ArcParserDECLARE             = 30
	ArcParserSTATE_DECLARE       = 31
	ArcParserTRANSITION          = 32
	ArcParserASSIGN              = 33
	ArcParserPLUS                = 34
	ArcParserMINUS               = 35
	ArcParserSTAR                = 36
	ArcParserSLASH               = 37
	ArcParserPERCENT             = 38
	ArcParserCARET               = 39
	ArcParserEQ                  = 40
	ArcParserNEQ                 = 41
	ArcParserLT                  = 42
	ArcParserGT                  = 43
	ArcParserLEQ                 = 44
	ArcParserGEQ                 = 45
	ArcParserAND                 = 46
	ArcParserOR                  = 47
	ArcParserNOT                 = 48
	ArcParserLPAREN              = 49
	ArcParserRPAREN              = 50
	ArcParserLBRACE              = 51
	ArcParserRBRACE              = 52
	ArcParserLBRACKET            = 53
	ArcParserRBRACKET            = 54
	ArcParserCOMMA               = 55
	ArcParserCOLON               = 56
	ArcParserSEMICOLON           = 57
	ArcParserTEMPORAL_LITERAL    = 58
	ArcParserFREQUENCY_LITERAL   = 59
	ArcParserINTEGER_LITERAL     = 60
	ArcParserFLOAT_LITERAL       = 61
	ArcParserSTR_LITERAL         = 62
	ArcParserIDENTIFIER          = 63
	ArcParserSINGLE_LINE_COMMENT = 64
	ArcParserMULTI_LINE_COMMENT  = 65
	ArcParserWS                  = 66
)

// ArcParser rules.
const (
	ArcParserRULE_program                  = 0
	ArcParserRULE_topLevelItem             = 1
	ArcParserRULE_functionDeclaration      = 2
	ArcParserRULE_inputList                = 3
	ArcParserRULE_input                    = 4
	ArcParserRULE_outputType               = 5
	ArcParserRULE_multiOutputBlock         = 6
	ArcParserRULE_namedOutput              = 7
	ArcParserRULE_configBlock              = 8
	ArcParserRULE_config                   = 9
	ArcParserRULE_sequenceDeclaration      = 10
	ArcParserRULE_sequenceChain            = 11
	ArcParserRULE_sequenceEntry            = 12
	ArcParserRULE_stageDeclaration         = 13
	ArcParserRULE_stageBody                = 14
	ArcParserRULE_stageItem                = 15
	ArcParserRULE_stageFlow                = 16
	ArcParserRULE_transitionStatement      = 17
	ArcParserRULE_transitionTarget         = 18
	ArcParserRULE_matchBlock               = 19
	ArcParserRULE_matchEntry               = 20
	ArcParserRULE_imperativeTransition     = 21
	ArcParserRULE_flowStatement            = 22
	ArcParserRULE_routingTable             = 23
	ArcParserRULE_routingEntry             = 24
	ArcParserRULE_flowNode                 = 25
	ArcParserRULE_channelIdentifier        = 26
	ArcParserRULE_function                 = 27
	ArcParserRULE_configValues             = 28
	ArcParserRULE_namedConfigValues        = 29
	ArcParserRULE_namedConfigValue         = 30
	ArcParserRULE_anonymousConfigValues    = 31
	ArcParserRULE_arguments                = 32
	ArcParserRULE_argumentList             = 33
	ArcParserRULE_block                    = 34
	ArcParserRULE_statement                = 35
	ArcParserRULE_variableDeclaration      = 36
	ArcParserRULE_localVariable            = 37
	ArcParserRULE_statefulVariable         = 38
	ArcParserRULE_assignment               = 39
	ArcParserRULE_channelOperation         = 40
	ArcParserRULE_channelWrite             = 41
	ArcParserRULE_channelRead              = 42
	ArcParserRULE_blockingRead             = 43
	ArcParserRULE_nonBlockingRead          = 44
	ArcParserRULE_ifStatement              = 45
	ArcParserRULE_elseIfClause             = 46
	ArcParserRULE_elseClause               = 47
	ArcParserRULE_returnStatement          = 48
	ArcParserRULE_functionCall             = 49
	ArcParserRULE_type                     = 50
	ArcParserRULE_primitiveType            = 51
	ArcParserRULE_numericType              = 52
	ArcParserRULE_integerType              = 53
	ArcParserRULE_floatType                = 54
	ArcParserRULE_temporalType             = 55
	ArcParserRULE_channelType              = 56
	ArcParserRULE_seriesType               = 57
	ArcParserRULE_expression               = 58
	ArcParserRULE_logicalOrExpression      = 59
	ArcParserRULE_logicalAndExpression     = 60
	ArcParserRULE_equalityExpression       = 61
	ArcParserRULE_relationalExpression     = 62
	ArcParserRULE_additiveExpression       = 63
	ArcParserRULE_multiplicativeExpression = 64
	ArcParserRULE_powerExpression          = 65
	ArcParserRULE_unaryExpression          = 66
	ArcParserRULE_blockingReadExpr         = 67
	ArcParserRULE_postfixExpression        = 68
	ArcParserRULE_indexOrSlice             = 69
	ArcParserRULE_functionCallSuffix       = 70
	ArcParserRULE_primaryExpression        = 71
	ArcParserRULE_typeCast                 = 72
	ArcParserRULE_builtinFunction          = 73
	ArcParserRULE_literal                  = 74
	ArcParserRULE_numericLiteral           = 75
	ArcParserRULE_temporalLiteral          = 76
	ArcParserRULE_seriesLiteral            = 77
	ArcParserRULE_expressionList           = 78
)

// IProgramContext is an interface to support dynamic dispatch.
type IProgramContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	EOF() antlr.TerminalNode
	AllTopLevelItem() []ITopLevelItemContext
	TopLevelItem(i int) ITopLevelItemContext

	// IsProgramContext differentiates from other interfaces.
	IsProgramContext()
}

type ProgramContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyProgramContext() *ProgramContext {
	var p = new(ProgramContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_program
	return p
}

func InitEmptyProgramContext(p *ProgramContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_program
}

func (*ProgramContext) IsProgramContext() {}

func NewProgramContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ProgramContext {
	var p = new(ProgramContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_program

	return p
}

func (s *ProgramContext) GetParser() antlr.Parser { return s.parser }

func (s *ProgramContext) EOF() antlr.TerminalNode {
	return s.GetToken(ArcParserEOF, 0)
}

func (s *ProgramContext) AllTopLevelItem() []ITopLevelItemContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(ITopLevelItemContext); ok {
			len++
		}
	}

	tst := make([]ITopLevelItemContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(ITopLevelItemContext); ok {
			tst[i] = t.(ITopLevelItemContext)
			i++
		}
	}

	return tst
}

func (s *ProgramContext) TopLevelItem(i int) ITopLevelItemContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ITopLevelItemContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(ITopLevelItemContext)
}

func (s *ProgramContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ProgramContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ProgramContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterProgram(s)
	}
}

func (s *ProgramContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitProgram(s)
	}
}

func (p *ArcParser) Program() (localctx IProgramContext) {
	localctx = NewProgramContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 0, ArcParserRULE_program)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(161)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&-276126916988110366) != 0 {
		{
			p.SetState(158)
			p.TopLevelItem()
		}

		p.SetState(163)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(164)
		p.Match(ArcParserEOF)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// ITopLevelItemContext is an interface to support dynamic dispatch.
type ITopLevelItemContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	FunctionDeclaration() IFunctionDeclarationContext
	FlowStatement() IFlowStatementContext
	SequenceDeclaration() ISequenceDeclarationContext
	StageDeclaration() IStageDeclarationContext

	// IsTopLevelItemContext differentiates from other interfaces.
	IsTopLevelItemContext()
}

type TopLevelItemContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyTopLevelItemContext() *TopLevelItemContext {
	var p = new(TopLevelItemContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_topLevelItem
	return p
}

func InitEmptyTopLevelItemContext(p *TopLevelItemContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_topLevelItem
}

func (*TopLevelItemContext) IsTopLevelItemContext() {}

func NewTopLevelItemContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *TopLevelItemContext {
	var p = new(TopLevelItemContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_topLevelItem

	return p
}

func (s *TopLevelItemContext) GetParser() antlr.Parser { return s.parser }

func (s *TopLevelItemContext) FunctionDeclaration() IFunctionDeclarationContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IFunctionDeclarationContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IFunctionDeclarationContext)
}

func (s *TopLevelItemContext) FlowStatement() IFlowStatementContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IFlowStatementContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IFlowStatementContext)
}

func (s *TopLevelItemContext) SequenceDeclaration() ISequenceDeclarationContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ISequenceDeclarationContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ISequenceDeclarationContext)
}

func (s *TopLevelItemContext) StageDeclaration() IStageDeclarationContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IStageDeclarationContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IStageDeclarationContext)
}

func (s *TopLevelItemContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *TopLevelItemContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *TopLevelItemContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterTopLevelItem(s)
	}
}

func (s *TopLevelItemContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitTopLevelItem(s)
	}
}

func (p *ArcParser) TopLevelItem() (localctx ITopLevelItemContext) {
	localctx = NewTopLevelItemContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 2, ArcParserRULE_topLevelItem)
	p.SetState(170)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case ArcParserFUNC:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(166)
			p.FunctionDeclaration()
		}

	case ArcParserNOW, ArcParserLEN, ArcParserCHAN, ArcParserRECV_CHAN, ArcParserSEND_CHAN, ArcParserI8, ArcParserI16, ArcParserI32, ArcParserI64, ArcParserU8, ArcParserU16, ArcParserU32, ArcParserU64, ArcParserF32, ArcParserF64, ArcParserSTR, ArcParserTIMESTAMP, ArcParserTIMESPAN, ArcParserSERIES, ArcParserRECV, ArcParserMINUS, ArcParserNOT, ArcParserLPAREN, ArcParserLBRACE, ArcParserLBRACKET, ArcParserTEMPORAL_LITERAL, ArcParserFREQUENCY_LITERAL, ArcParserINTEGER_LITERAL, ArcParserFLOAT_LITERAL, ArcParserSTR_LITERAL, ArcParserIDENTIFIER:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(167)
			p.FlowStatement()
		}

	case ArcParserSEQUENCE:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(168)
			p.SequenceDeclaration()
		}

	case ArcParserSTAGE:
		p.EnterOuterAlt(localctx, 4)
		{
			p.SetState(169)
			p.StageDeclaration()
		}

	default:
		p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
		goto errorExit
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IFunctionDeclarationContext is an interface to support dynamic dispatch.
type IFunctionDeclarationContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	FUNC() antlr.TerminalNode
	IDENTIFIER() antlr.TerminalNode
	LPAREN() antlr.TerminalNode
	RPAREN() antlr.TerminalNode
	Block() IBlockContext
	ConfigBlock() IConfigBlockContext
	InputList() IInputListContext
	OutputType() IOutputTypeContext

	// IsFunctionDeclarationContext differentiates from other interfaces.
	IsFunctionDeclarationContext()
}

type FunctionDeclarationContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyFunctionDeclarationContext() *FunctionDeclarationContext {
	var p = new(FunctionDeclarationContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_functionDeclaration
	return p
}

func InitEmptyFunctionDeclarationContext(p *FunctionDeclarationContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_functionDeclaration
}

func (*FunctionDeclarationContext) IsFunctionDeclarationContext() {}

func NewFunctionDeclarationContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *FunctionDeclarationContext {
	var p = new(FunctionDeclarationContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_functionDeclaration

	return p
}

func (s *FunctionDeclarationContext) GetParser() antlr.Parser { return s.parser }

func (s *FunctionDeclarationContext) FUNC() antlr.TerminalNode {
	return s.GetToken(ArcParserFUNC, 0)
}

func (s *FunctionDeclarationContext) IDENTIFIER() antlr.TerminalNode {
	return s.GetToken(ArcParserIDENTIFIER, 0)
}

func (s *FunctionDeclarationContext) LPAREN() antlr.TerminalNode {
	return s.GetToken(ArcParserLPAREN, 0)
}

func (s *FunctionDeclarationContext) RPAREN() antlr.TerminalNode {
	return s.GetToken(ArcParserRPAREN, 0)
}

func (s *FunctionDeclarationContext) Block() IBlockContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IBlockContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IBlockContext)
}

func (s *FunctionDeclarationContext) ConfigBlock() IConfigBlockContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IConfigBlockContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IConfigBlockContext)
}

func (s *FunctionDeclarationContext) InputList() IInputListContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IInputListContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IInputListContext)
}

func (s *FunctionDeclarationContext) OutputType() IOutputTypeContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IOutputTypeContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IOutputTypeContext)
}

func (s *FunctionDeclarationContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *FunctionDeclarationContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *FunctionDeclarationContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterFunctionDeclaration(s)
	}
}

func (s *FunctionDeclarationContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitFunctionDeclaration(s)
	}
}

func (p *ArcParser) FunctionDeclaration() (localctx IFunctionDeclarationContext) {
	localctx = NewFunctionDeclarationContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 4, ArcParserRULE_functionDeclaration)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(172)
		p.Match(ArcParserFUNC)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(173)
		p.Match(ArcParserIDENTIFIER)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(175)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ArcParserLBRACE {
		{
			p.SetState(174)
			p.ConfigBlock()
		}

	}
	{
		p.SetState(177)
		p.Match(ArcParserLPAREN)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(179)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ArcParserIDENTIFIER {
		{
			p.SetState(178)
			p.InputList()
		}

	}
	{
		p.SetState(181)
		p.Match(ArcParserRPAREN)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(183)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&-9222809086632921088) != 0 {
		{
			p.SetState(182)
			p.OutputType()
		}

	}
	{
		p.SetState(185)
		p.Block()
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IInputListContext is an interface to support dynamic dispatch.
type IInputListContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AllInput() []IInputContext
	Input(i int) IInputContext
	AllCOMMA() []antlr.TerminalNode
	COMMA(i int) antlr.TerminalNode

	// IsInputListContext differentiates from other interfaces.
	IsInputListContext()
}

type InputListContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyInputListContext() *InputListContext {
	var p = new(InputListContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_inputList
	return p
}

func InitEmptyInputListContext(p *InputListContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_inputList
}

func (*InputListContext) IsInputListContext() {}

func NewInputListContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *InputListContext {
	var p = new(InputListContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_inputList

	return p
}

func (s *InputListContext) GetParser() antlr.Parser { return s.parser }

func (s *InputListContext) AllInput() []IInputContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IInputContext); ok {
			len++
		}
	}

	tst := make([]IInputContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IInputContext); ok {
			tst[i] = t.(IInputContext)
			i++
		}
	}

	return tst
}

func (s *InputListContext) Input(i int) IInputContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IInputContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IInputContext)
}

func (s *InputListContext) AllCOMMA() []antlr.TerminalNode {
	return s.GetTokens(ArcParserCOMMA)
}

func (s *InputListContext) COMMA(i int) antlr.TerminalNode {
	return s.GetToken(ArcParserCOMMA, i)
}

func (s *InputListContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *InputListContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *InputListContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterInputList(s)
	}
}

func (s *InputListContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitInputList(s)
	}
}

func (p *ArcParser) InputList() (localctx IInputListContext) {
	localctx = NewInputListContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 6, ArcParserRULE_inputList)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(187)
		p.Input()
	}
	p.SetState(192)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == ArcParserCOMMA {
		{
			p.SetState(188)
			p.Match(ArcParserCOMMA)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(189)
			p.Input()
		}

		p.SetState(194)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IInputContext is an interface to support dynamic dispatch.
type IInputContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	IDENTIFIER() antlr.TerminalNode
	Type_() ITypeContext
	ASSIGN() antlr.TerminalNode
	Literal() ILiteralContext

	// IsInputContext differentiates from other interfaces.
	IsInputContext()
}

type InputContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyInputContext() *InputContext {
	var p = new(InputContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_input
	return p
}

func InitEmptyInputContext(p *InputContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_input
}

func (*InputContext) IsInputContext() {}

func NewInputContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *InputContext {
	var p = new(InputContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_input

	return p
}

func (s *InputContext) GetParser() antlr.Parser { return s.parser }

func (s *InputContext) IDENTIFIER() antlr.TerminalNode {
	return s.GetToken(ArcParserIDENTIFIER, 0)
}

func (s *InputContext) Type_() ITypeContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ITypeContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ITypeContext)
}

func (s *InputContext) ASSIGN() antlr.TerminalNode {
	return s.GetToken(ArcParserASSIGN, 0)
}

func (s *InputContext) Literal() ILiteralContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ILiteralContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ILiteralContext)
}

func (s *InputContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *InputContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *InputContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterInput(s)
	}
}

func (s *InputContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitInput(s)
	}
}

func (p *ArcParser) Input() (localctx IInputContext) {
	localctx = NewInputContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 8, ArcParserRULE_input)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(195)
		p.Match(ArcParserIDENTIFIER)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(196)
		p.Type_()
	}
	p.SetState(199)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ArcParserASSIGN {
		{
			p.SetState(197)
			p.Match(ArcParserASSIGN)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(198)
			p.Literal()
		}

	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IOutputTypeContext is an interface to support dynamic dispatch.
type IOutputTypeContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	Type_() ITypeContext
	IDENTIFIER() antlr.TerminalNode
	MultiOutputBlock() IMultiOutputBlockContext

	// IsOutputTypeContext differentiates from other interfaces.
	IsOutputTypeContext()
}

type OutputTypeContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyOutputTypeContext() *OutputTypeContext {
	var p = new(OutputTypeContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_outputType
	return p
}

func InitEmptyOutputTypeContext(p *OutputTypeContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_outputType
}

func (*OutputTypeContext) IsOutputTypeContext() {}

func NewOutputTypeContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *OutputTypeContext {
	var p = new(OutputTypeContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_outputType

	return p
}

func (s *OutputTypeContext) GetParser() antlr.Parser { return s.parser }

func (s *OutputTypeContext) Type_() ITypeContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ITypeContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ITypeContext)
}

func (s *OutputTypeContext) IDENTIFIER() antlr.TerminalNode {
	return s.GetToken(ArcParserIDENTIFIER, 0)
}

func (s *OutputTypeContext) MultiOutputBlock() IMultiOutputBlockContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IMultiOutputBlockContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IMultiOutputBlockContext)
}

func (s *OutputTypeContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *OutputTypeContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *OutputTypeContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterOutputType(s)
	}
}

func (s *OutputTypeContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitOutputType(s)
	}
}

func (p *ArcParser) OutputType() (localctx IOutputTypeContext) {
	localctx = NewOutputTypeContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 10, ArcParserRULE_outputType)
	p.SetState(205)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case ArcParserCHAN, ArcParserRECV_CHAN, ArcParserSEND_CHAN, ArcParserI8, ArcParserI16, ArcParserI32, ArcParserI64, ArcParserU8, ArcParserU16, ArcParserU32, ArcParserU64, ArcParserF32, ArcParserF64, ArcParserSTR, ArcParserTIMESTAMP, ArcParserTIMESPAN, ArcParserSERIES:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(201)
			p.Type_()
		}

	case ArcParserIDENTIFIER:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(202)
			p.Match(ArcParserIDENTIFIER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(203)
			p.Type_()
		}

	case ArcParserLPAREN:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(204)
			p.MultiOutputBlock()
		}

	default:
		p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
		goto errorExit
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IMultiOutputBlockContext is an interface to support dynamic dispatch.
type IMultiOutputBlockContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	LPAREN() antlr.TerminalNode
	AllNamedOutput() []INamedOutputContext
	NamedOutput(i int) INamedOutputContext
	RPAREN() antlr.TerminalNode
	AllCOMMA() []antlr.TerminalNode
	COMMA(i int) antlr.TerminalNode

	// IsMultiOutputBlockContext differentiates from other interfaces.
	IsMultiOutputBlockContext()
}

type MultiOutputBlockContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyMultiOutputBlockContext() *MultiOutputBlockContext {
	var p = new(MultiOutputBlockContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_multiOutputBlock
	return p
}

func InitEmptyMultiOutputBlockContext(p *MultiOutputBlockContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_multiOutputBlock
}

func (*MultiOutputBlockContext) IsMultiOutputBlockContext() {}

func NewMultiOutputBlockContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *MultiOutputBlockContext {
	var p = new(MultiOutputBlockContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_multiOutputBlock

	return p
}

func (s *MultiOutputBlockContext) GetParser() antlr.Parser { return s.parser }

func (s *MultiOutputBlockContext) LPAREN() antlr.TerminalNode {
	return s.GetToken(ArcParserLPAREN, 0)
}

func (s *MultiOutputBlockContext) AllNamedOutput() []INamedOutputContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(INamedOutputContext); ok {
			len++
		}
	}

	tst := make([]INamedOutputContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(INamedOutputContext); ok {
			tst[i] = t.(INamedOutputContext)
			i++
		}
	}

	return tst
}

func (s *MultiOutputBlockContext) NamedOutput(i int) INamedOutputContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(INamedOutputContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(INamedOutputContext)
}

func (s *MultiOutputBlockContext) RPAREN() antlr.TerminalNode {
	return s.GetToken(ArcParserRPAREN, 0)
}

func (s *MultiOutputBlockContext) AllCOMMA() []antlr.TerminalNode {
	return s.GetTokens(ArcParserCOMMA)
}

func (s *MultiOutputBlockContext) COMMA(i int) antlr.TerminalNode {
	return s.GetToken(ArcParserCOMMA, i)
}

func (s *MultiOutputBlockContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *MultiOutputBlockContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *MultiOutputBlockContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterMultiOutputBlock(s)
	}
}

func (s *MultiOutputBlockContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitMultiOutputBlock(s)
	}
}

func (p *ArcParser) MultiOutputBlock() (localctx IMultiOutputBlockContext) {
	localctx = NewMultiOutputBlockContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 12, ArcParserRULE_multiOutputBlock)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(207)
		p.Match(ArcParserLPAREN)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(208)
		p.NamedOutput()
	}
	p.SetState(213)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == ArcParserCOMMA {
		{
			p.SetState(209)
			p.Match(ArcParserCOMMA)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(210)
			p.NamedOutput()
		}

		p.SetState(215)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(216)
		p.Match(ArcParserRPAREN)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// INamedOutputContext is an interface to support dynamic dispatch.
type INamedOutputContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	IDENTIFIER() antlr.TerminalNode
	Type_() ITypeContext

	// IsNamedOutputContext differentiates from other interfaces.
	IsNamedOutputContext()
}

type NamedOutputContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyNamedOutputContext() *NamedOutputContext {
	var p = new(NamedOutputContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_namedOutput
	return p
}

func InitEmptyNamedOutputContext(p *NamedOutputContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_namedOutput
}

func (*NamedOutputContext) IsNamedOutputContext() {}

func NewNamedOutputContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *NamedOutputContext {
	var p = new(NamedOutputContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_namedOutput

	return p
}

func (s *NamedOutputContext) GetParser() antlr.Parser { return s.parser }

func (s *NamedOutputContext) IDENTIFIER() antlr.TerminalNode {
	return s.GetToken(ArcParserIDENTIFIER, 0)
}

func (s *NamedOutputContext) Type_() ITypeContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ITypeContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ITypeContext)
}

func (s *NamedOutputContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *NamedOutputContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *NamedOutputContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterNamedOutput(s)
	}
}

func (s *NamedOutputContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitNamedOutput(s)
	}
}

func (p *ArcParser) NamedOutput() (localctx INamedOutputContext) {
	localctx = NewNamedOutputContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 14, ArcParserRULE_namedOutput)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(218)
		p.Match(ArcParserIDENTIFIER)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(219)
		p.Type_()
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IConfigBlockContext is an interface to support dynamic dispatch.
type IConfigBlockContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	LBRACE() antlr.TerminalNode
	RBRACE() antlr.TerminalNode
	AllConfig() []IConfigContext
	Config(i int) IConfigContext

	// IsConfigBlockContext differentiates from other interfaces.
	IsConfigBlockContext()
}

type ConfigBlockContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyConfigBlockContext() *ConfigBlockContext {
	var p = new(ConfigBlockContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_configBlock
	return p
}

func InitEmptyConfigBlockContext(p *ConfigBlockContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_configBlock
}

func (*ConfigBlockContext) IsConfigBlockContext() {}

func NewConfigBlockContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ConfigBlockContext {
	var p = new(ConfigBlockContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_configBlock

	return p
}

func (s *ConfigBlockContext) GetParser() antlr.Parser { return s.parser }

func (s *ConfigBlockContext) LBRACE() antlr.TerminalNode {
	return s.GetToken(ArcParserLBRACE, 0)
}

func (s *ConfigBlockContext) RBRACE() antlr.TerminalNode {
	return s.GetToken(ArcParserRBRACE, 0)
}

func (s *ConfigBlockContext) AllConfig() []IConfigContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IConfigContext); ok {
			len++
		}
	}

	tst := make([]IConfigContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IConfigContext); ok {
			tst[i] = t.(IConfigContext)
			i++
		}
	}

	return tst
}

func (s *ConfigBlockContext) Config(i int) IConfigContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IConfigContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IConfigContext)
}

func (s *ConfigBlockContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ConfigBlockContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ConfigBlockContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterConfigBlock(s)
	}
}

func (s *ConfigBlockContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitConfigBlock(s)
	}
}

func (p *ArcParser) ConfigBlock() (localctx IConfigBlockContext) {
	localctx = NewConfigBlockContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 16, ArcParserRULE_configBlock)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(221)
		p.Match(ArcParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(225)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == ArcParserIDENTIFIER {
		{
			p.SetState(222)
			p.Config()
		}

		p.SetState(227)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(228)
		p.Match(ArcParserRBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IConfigContext is an interface to support dynamic dispatch.
type IConfigContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	IDENTIFIER() antlr.TerminalNode
	Type_() ITypeContext

	// IsConfigContext differentiates from other interfaces.
	IsConfigContext()
}

type ConfigContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyConfigContext() *ConfigContext {
	var p = new(ConfigContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_config
	return p
}

func InitEmptyConfigContext(p *ConfigContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_config
}

func (*ConfigContext) IsConfigContext() {}

func NewConfigContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ConfigContext {
	var p = new(ConfigContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_config

	return p
}

func (s *ConfigContext) GetParser() antlr.Parser { return s.parser }

func (s *ConfigContext) IDENTIFIER() antlr.TerminalNode {
	return s.GetToken(ArcParserIDENTIFIER, 0)
}

func (s *ConfigContext) Type_() ITypeContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ITypeContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ITypeContext)
}

func (s *ConfigContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ConfigContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ConfigContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterConfig(s)
	}
}

func (s *ConfigContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitConfig(s)
	}
}

func (p *ArcParser) Config() (localctx IConfigContext) {
	localctx = NewConfigContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 18, ArcParserRULE_config)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(230)
		p.Match(ArcParserIDENTIFIER)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(231)
		p.Type_()
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// ISequenceDeclarationContext is an interface to support dynamic dispatch.
type ISequenceDeclarationContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	SEQUENCE() antlr.TerminalNode
	IDENTIFIER() antlr.TerminalNode
	LBRACE() antlr.TerminalNode
	AllSequenceChain() []ISequenceChainContext
	SequenceChain(i int) ISequenceChainContext
	RBRACE() antlr.TerminalNode
	AllCOMMA() []antlr.TerminalNode
	COMMA(i int) antlr.TerminalNode

	// IsSequenceDeclarationContext differentiates from other interfaces.
	IsSequenceDeclarationContext()
}

type SequenceDeclarationContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptySequenceDeclarationContext() *SequenceDeclarationContext {
	var p = new(SequenceDeclarationContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_sequenceDeclaration
	return p
}

func InitEmptySequenceDeclarationContext(p *SequenceDeclarationContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_sequenceDeclaration
}

func (*SequenceDeclarationContext) IsSequenceDeclarationContext() {}

func NewSequenceDeclarationContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *SequenceDeclarationContext {
	var p = new(SequenceDeclarationContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_sequenceDeclaration

	return p
}

func (s *SequenceDeclarationContext) GetParser() antlr.Parser { return s.parser }

func (s *SequenceDeclarationContext) SEQUENCE() antlr.TerminalNode {
	return s.GetToken(ArcParserSEQUENCE, 0)
}

func (s *SequenceDeclarationContext) IDENTIFIER() antlr.TerminalNode {
	return s.GetToken(ArcParserIDENTIFIER, 0)
}

func (s *SequenceDeclarationContext) LBRACE() antlr.TerminalNode {
	return s.GetToken(ArcParserLBRACE, 0)
}

func (s *SequenceDeclarationContext) AllSequenceChain() []ISequenceChainContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(ISequenceChainContext); ok {
			len++
		}
	}

	tst := make([]ISequenceChainContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(ISequenceChainContext); ok {
			tst[i] = t.(ISequenceChainContext)
			i++
		}
	}

	return tst
}

func (s *SequenceDeclarationContext) SequenceChain(i int) ISequenceChainContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ISequenceChainContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(ISequenceChainContext)
}

func (s *SequenceDeclarationContext) RBRACE() antlr.TerminalNode {
	return s.GetToken(ArcParserRBRACE, 0)
}

func (s *SequenceDeclarationContext) AllCOMMA() []antlr.TerminalNode {
	return s.GetTokens(ArcParserCOMMA)
}

func (s *SequenceDeclarationContext) COMMA(i int) antlr.TerminalNode {
	return s.GetToken(ArcParserCOMMA, i)
}

func (s *SequenceDeclarationContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *SequenceDeclarationContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *SequenceDeclarationContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterSequenceDeclaration(s)
	}
}

func (s *SequenceDeclarationContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitSequenceDeclaration(s)
	}
}

func (p *ArcParser) SequenceDeclaration() (localctx ISequenceDeclarationContext) {
	localctx = NewSequenceDeclarationContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 20, ArcParserRULE_sequenceDeclaration)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(233)
		p.Match(ArcParserSEQUENCE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(234)
		p.Match(ArcParserIDENTIFIER)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(235)
		p.Match(ArcParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(236)
		p.SequenceChain()
	}
	p.SetState(241)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == ArcParserCOMMA {
		{
			p.SetState(237)
			p.Match(ArcParserCOMMA)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(238)
			p.SequenceChain()
		}

		p.SetState(243)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(244)
		p.Match(ArcParserRBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// ISequenceChainContext is an interface to support dynamic dispatch.
type ISequenceChainContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AllSequenceEntry() []ISequenceEntryContext
	SequenceEntry(i int) ISequenceEntryContext
	AllTRANSITION() []antlr.TerminalNode
	TRANSITION(i int) antlr.TerminalNode

	// IsSequenceChainContext differentiates from other interfaces.
	IsSequenceChainContext()
}

type SequenceChainContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptySequenceChainContext() *SequenceChainContext {
	var p = new(SequenceChainContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_sequenceChain
	return p
}

func InitEmptySequenceChainContext(p *SequenceChainContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_sequenceChain
}

func (*SequenceChainContext) IsSequenceChainContext() {}

func NewSequenceChainContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *SequenceChainContext {
	var p = new(SequenceChainContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_sequenceChain

	return p
}

func (s *SequenceChainContext) GetParser() antlr.Parser { return s.parser }

func (s *SequenceChainContext) AllSequenceEntry() []ISequenceEntryContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(ISequenceEntryContext); ok {
			len++
		}
	}

	tst := make([]ISequenceEntryContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(ISequenceEntryContext); ok {
			tst[i] = t.(ISequenceEntryContext)
			i++
		}
	}

	return tst
}

func (s *SequenceChainContext) SequenceEntry(i int) ISequenceEntryContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ISequenceEntryContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(ISequenceEntryContext)
}

func (s *SequenceChainContext) AllTRANSITION() []antlr.TerminalNode {
	return s.GetTokens(ArcParserTRANSITION)
}

func (s *SequenceChainContext) TRANSITION(i int) antlr.TerminalNode {
	return s.GetToken(ArcParserTRANSITION, i)
}

func (s *SequenceChainContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *SequenceChainContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *SequenceChainContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterSequenceChain(s)
	}
}

func (s *SequenceChainContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitSequenceChain(s)
	}
}

func (p *ArcParser) SequenceChain() (localctx ISequenceChainContext) {
	localctx = NewSequenceChainContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 22, ArcParserRULE_sequenceChain)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(246)
		p.SequenceEntry()
	}
	p.SetState(251)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == ArcParserTRANSITION {
		{
			p.SetState(247)
			p.Match(ArcParserTRANSITION)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(248)
			p.SequenceEntry()
		}

		p.SetState(253)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// ISequenceEntryContext is an interface to support dynamic dispatch.
type ISequenceEntryContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	IDENTIFIER() antlr.TerminalNode
	COLON() antlr.TerminalNode
	STAGE() antlr.TerminalNode
	StageBody() IStageBodyContext

	// IsSequenceEntryContext differentiates from other interfaces.
	IsSequenceEntryContext()
}

type SequenceEntryContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptySequenceEntryContext() *SequenceEntryContext {
	var p = new(SequenceEntryContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_sequenceEntry
	return p
}

func InitEmptySequenceEntryContext(p *SequenceEntryContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_sequenceEntry
}

func (*SequenceEntryContext) IsSequenceEntryContext() {}

func NewSequenceEntryContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *SequenceEntryContext {
	var p = new(SequenceEntryContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_sequenceEntry

	return p
}

func (s *SequenceEntryContext) GetParser() antlr.Parser { return s.parser }

func (s *SequenceEntryContext) IDENTIFIER() antlr.TerminalNode {
	return s.GetToken(ArcParserIDENTIFIER, 0)
}

func (s *SequenceEntryContext) COLON() antlr.TerminalNode {
	return s.GetToken(ArcParserCOLON, 0)
}

func (s *SequenceEntryContext) STAGE() antlr.TerminalNode {
	return s.GetToken(ArcParserSTAGE, 0)
}

func (s *SequenceEntryContext) StageBody() IStageBodyContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IStageBodyContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IStageBodyContext)
}

func (s *SequenceEntryContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *SequenceEntryContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *SequenceEntryContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterSequenceEntry(s)
	}
}

func (s *SequenceEntryContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitSequenceEntry(s)
	}
}

func (p *ArcParser) SequenceEntry() (localctx ISequenceEntryContext) {
	localctx = NewSequenceEntryContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 24, ArcParserRULE_sequenceEntry)
	p.SetState(261)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 12, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(254)
			p.Match(ArcParserIDENTIFIER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(255)
			p.Match(ArcParserCOLON)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(256)
			p.Match(ArcParserSTAGE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(257)
			p.StageBody()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(258)
			p.Match(ArcParserSTAGE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(259)
			p.StageBody()
		}

	case 3:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(260)
			p.Match(ArcParserIDENTIFIER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case antlr.ATNInvalidAltNumber:
		goto errorExit
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IStageDeclarationContext is an interface to support dynamic dispatch.
type IStageDeclarationContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	STAGE() antlr.TerminalNode
	IDENTIFIER() antlr.TerminalNode
	StageBody() IStageBodyContext

	// IsStageDeclarationContext differentiates from other interfaces.
	IsStageDeclarationContext()
}

type StageDeclarationContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyStageDeclarationContext() *StageDeclarationContext {
	var p = new(StageDeclarationContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_stageDeclaration
	return p
}

func InitEmptyStageDeclarationContext(p *StageDeclarationContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_stageDeclaration
}

func (*StageDeclarationContext) IsStageDeclarationContext() {}

func NewStageDeclarationContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *StageDeclarationContext {
	var p = new(StageDeclarationContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_stageDeclaration

	return p
}

func (s *StageDeclarationContext) GetParser() antlr.Parser { return s.parser }

func (s *StageDeclarationContext) STAGE() antlr.TerminalNode {
	return s.GetToken(ArcParserSTAGE, 0)
}

func (s *StageDeclarationContext) IDENTIFIER() antlr.TerminalNode {
	return s.GetToken(ArcParserIDENTIFIER, 0)
}

func (s *StageDeclarationContext) StageBody() IStageBodyContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IStageBodyContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IStageBodyContext)
}

func (s *StageDeclarationContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *StageDeclarationContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *StageDeclarationContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterStageDeclaration(s)
	}
}

func (s *StageDeclarationContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitStageDeclaration(s)
	}
}

func (p *ArcParser) StageDeclaration() (localctx IStageDeclarationContext) {
	localctx = NewStageDeclarationContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 26, ArcParserRULE_stageDeclaration)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(263)
		p.Match(ArcParserSTAGE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(264)
		p.Match(ArcParserIDENTIFIER)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(265)
		p.StageBody()
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IStageBodyContext is an interface to support dynamic dispatch.
type IStageBodyContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	LBRACE() antlr.TerminalNode
	RBRACE() antlr.TerminalNode
	AllStageItem() []IStageItemContext
	StageItem(i int) IStageItemContext

	// IsStageBodyContext differentiates from other interfaces.
	IsStageBodyContext()
}

type StageBodyContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyStageBodyContext() *StageBodyContext {
	var p = new(StageBodyContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_stageBody
	return p
}

func InitEmptyStageBodyContext(p *StageBodyContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_stageBody
}

func (*StageBodyContext) IsStageBodyContext() {}

func NewStageBodyContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *StageBodyContext {
	var p = new(StageBodyContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_stageBody

	return p
}

func (s *StageBodyContext) GetParser() antlr.Parser { return s.parser }

func (s *StageBodyContext) LBRACE() antlr.TerminalNode {
	return s.GetToken(ArcParserLBRACE, 0)
}

func (s *StageBodyContext) RBRACE() antlr.TerminalNode {
	return s.GetToken(ArcParserRBRACE, 0)
}

func (s *StageBodyContext) AllStageItem() []IStageItemContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IStageItemContext); ok {
			len++
		}
	}

	tst := make([]IStageItemContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IStageItemContext); ok {
			tst[i] = t.(IStageItemContext)
			i++
		}
	}

	return tst
}

func (s *StageBodyContext) StageItem(i int) IStageItemContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IStageItemContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IStageItemContext)
}

func (s *StageBodyContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *StageBodyContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *StageBodyContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterStageBody(s)
	}
}

func (s *StageBodyContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitStageBody(s)
	}
}

func (p *ArcParser) StageBody() (localctx IStageBodyContext) {
	localctx = NewStageBodyContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 28, ArcParserRULE_stageBody)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(267)
		p.Match(ArcParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(271)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&-276126916988110752) != 0 {
		{
			p.SetState(268)
			p.StageItem()
		}

		p.SetState(273)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(274)
		p.Match(ArcParserRBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IStageItemContext is an interface to support dynamic dispatch.
type IStageItemContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	StageFlow() IStageFlowContext
	TransitionStatement() ITransitionStatementContext
	ImperativeTransition() IImperativeTransitionContext

	// IsStageItemContext differentiates from other interfaces.
	IsStageItemContext()
}

type StageItemContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyStageItemContext() *StageItemContext {
	var p = new(StageItemContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_stageItem
	return p
}

func InitEmptyStageItemContext(p *StageItemContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_stageItem
}

func (*StageItemContext) IsStageItemContext() {}

func NewStageItemContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *StageItemContext {
	var p = new(StageItemContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_stageItem

	return p
}

func (s *StageItemContext) GetParser() antlr.Parser { return s.parser }

func (s *StageItemContext) StageFlow() IStageFlowContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IStageFlowContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IStageFlowContext)
}

func (s *StageItemContext) TransitionStatement() ITransitionStatementContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ITransitionStatementContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ITransitionStatementContext)
}

func (s *StageItemContext) ImperativeTransition() IImperativeTransitionContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IImperativeTransitionContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IImperativeTransitionContext)
}

func (s *StageItemContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *StageItemContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *StageItemContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterStageItem(s)
	}
}

func (s *StageItemContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitStageItem(s)
	}
}

func (p *ArcParser) StageItem() (localctx IStageItemContext) {
	localctx = NewStageItemContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 30, ArcParserRULE_stageItem)
	p.SetState(279)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 14, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(276)
			p.StageFlow()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(277)
			p.TransitionStatement()
		}

	case 3:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(278)
			p.ImperativeTransition()
		}

	case antlr.ATNInvalidAltNumber:
		goto errorExit
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IStageFlowContext is an interface to support dynamic dispatch.
type IStageFlowContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AllRoutingTable() []IRoutingTableContext
	RoutingTable(i int) IRoutingTableContext
	AllFlowNode() []IFlowNodeContext
	FlowNode(i int) IFlowNodeContext
	AllARROW() []antlr.TerminalNode
	ARROW(i int) antlr.TerminalNode
	SEMICOLON() antlr.TerminalNode

	// IsStageFlowContext differentiates from other interfaces.
	IsStageFlowContext()
}

type StageFlowContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyStageFlowContext() *StageFlowContext {
	var p = new(StageFlowContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_stageFlow
	return p
}

func InitEmptyStageFlowContext(p *StageFlowContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_stageFlow
}

func (*StageFlowContext) IsStageFlowContext() {}

func NewStageFlowContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *StageFlowContext {
	var p = new(StageFlowContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_stageFlow

	return p
}

func (s *StageFlowContext) GetParser() antlr.Parser { return s.parser }

func (s *StageFlowContext) AllRoutingTable() []IRoutingTableContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IRoutingTableContext); ok {
			len++
		}
	}

	tst := make([]IRoutingTableContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IRoutingTableContext); ok {
			tst[i] = t.(IRoutingTableContext)
			i++
		}
	}

	return tst
}

func (s *StageFlowContext) RoutingTable(i int) IRoutingTableContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IRoutingTableContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IRoutingTableContext)
}

func (s *StageFlowContext) AllFlowNode() []IFlowNodeContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IFlowNodeContext); ok {
			len++
		}
	}

	tst := make([]IFlowNodeContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IFlowNodeContext); ok {
			tst[i] = t.(IFlowNodeContext)
			i++
		}
	}

	return tst
}

func (s *StageFlowContext) FlowNode(i int) IFlowNodeContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IFlowNodeContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IFlowNodeContext)
}

func (s *StageFlowContext) AllARROW() []antlr.TerminalNode {
	return s.GetTokens(ArcParserARROW)
}

func (s *StageFlowContext) ARROW(i int) antlr.TerminalNode {
	return s.GetToken(ArcParserARROW, i)
}

func (s *StageFlowContext) SEMICOLON() antlr.TerminalNode {
	return s.GetToken(ArcParserSEMICOLON, 0)
}

func (s *StageFlowContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *StageFlowContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *StageFlowContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterStageFlow(s)
	}
}

func (s *StageFlowContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitStageFlow(s)
	}
}

func (p *ArcParser) StageFlow() (localctx IStageFlowContext) {
	localctx = NewStageFlowContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 32, ArcParserRULE_stageFlow)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(283)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case ArcParserLBRACE:
		{
			p.SetState(281)
			p.RoutingTable()
		}

	case ArcParserNOW, ArcParserLEN, ArcParserCHAN, ArcParserRECV_CHAN, ArcParserSEND_CHAN, ArcParserI8, ArcParserI16, ArcParserI32, ArcParserI64, ArcParserU8, ArcParserU16, ArcParserU32, ArcParserU64, ArcParserF32, ArcParserF64, ArcParserSTR, ArcParserTIMESTAMP, ArcParserTIMESPAN, ArcParserSERIES, ArcParserRECV, ArcParserMINUS, ArcParserNOT, ArcParserLPAREN, ArcParserLBRACKET, ArcParserTEMPORAL_LITERAL, ArcParserFREQUENCY_LITERAL, ArcParserINTEGER_LITERAL, ArcParserFLOAT_LITERAL, ArcParserSTR_LITERAL, ArcParserIDENTIFIER:
		{
			p.SetState(282)
			p.FlowNode()
		}

	default:
		p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
		goto errorExit
	}
	p.SetState(290)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for ok := true; ok; ok = _la == ArcParserARROW {
		{
			p.SetState(285)
			p.Match(ArcParserARROW)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		p.SetState(288)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}

		switch p.GetTokenStream().LA(1) {
		case ArcParserLBRACE:
			{
				p.SetState(286)
				p.RoutingTable()
			}

		case ArcParserNOW, ArcParserLEN, ArcParserCHAN, ArcParserRECV_CHAN, ArcParserSEND_CHAN, ArcParserI8, ArcParserI16, ArcParserI32, ArcParserI64, ArcParserU8, ArcParserU16, ArcParserU32, ArcParserU64, ArcParserF32, ArcParserF64, ArcParserSTR, ArcParserTIMESTAMP, ArcParserTIMESPAN, ArcParserSERIES, ArcParserRECV, ArcParserMINUS, ArcParserNOT, ArcParserLPAREN, ArcParserLBRACKET, ArcParserTEMPORAL_LITERAL, ArcParserFREQUENCY_LITERAL, ArcParserINTEGER_LITERAL, ArcParserFLOAT_LITERAL, ArcParserSTR_LITERAL, ArcParserIDENTIFIER:
			{
				p.SetState(287)
				p.FlowNode()
			}

		default:
			p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
			goto errorExit
		}

		p.SetState(292)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	p.SetState(295)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ArcParserSEMICOLON {
		{
			p.SetState(294)
			p.Match(ArcParserSEMICOLON)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// ITransitionStatementContext is an interface to support dynamic dispatch.
type ITransitionStatementContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	Expression() IExpressionContext
	TRANSITION() antlr.TerminalNode
	TransitionTarget() ITransitionTargetContext

	// IsTransitionStatementContext differentiates from other interfaces.
	IsTransitionStatementContext()
}

type TransitionStatementContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyTransitionStatementContext() *TransitionStatementContext {
	var p = new(TransitionStatementContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_transitionStatement
	return p
}

func InitEmptyTransitionStatementContext(p *TransitionStatementContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_transitionStatement
}

func (*TransitionStatementContext) IsTransitionStatementContext() {}

func NewTransitionStatementContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *TransitionStatementContext {
	var p = new(TransitionStatementContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_transitionStatement

	return p
}

func (s *TransitionStatementContext) GetParser() antlr.Parser { return s.parser }

func (s *TransitionStatementContext) Expression() IExpressionContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IExpressionContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IExpressionContext)
}

func (s *TransitionStatementContext) TRANSITION() antlr.TerminalNode {
	return s.GetToken(ArcParserTRANSITION, 0)
}

func (s *TransitionStatementContext) TransitionTarget() ITransitionTargetContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ITransitionTargetContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ITransitionTargetContext)
}

func (s *TransitionStatementContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *TransitionStatementContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *TransitionStatementContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterTransitionStatement(s)
	}
}

func (s *TransitionStatementContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitTransitionStatement(s)
	}
}

func (p *ArcParser) TransitionStatement() (localctx ITransitionStatementContext) {
	localctx = NewTransitionStatementContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 34, ArcParserRULE_transitionStatement)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(297)
		p.Expression()
	}
	{
		p.SetState(298)
		p.Match(ArcParserTRANSITION)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(299)
		p.TransitionTarget()
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// ITransitionTargetContext is an interface to support dynamic dispatch.
type ITransitionTargetContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	NEXT() antlr.TerminalNode
	MatchBlock() IMatchBlockContext
	IDENTIFIER() antlr.TerminalNode

	// IsTransitionTargetContext differentiates from other interfaces.
	IsTransitionTargetContext()
}

type TransitionTargetContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyTransitionTargetContext() *TransitionTargetContext {
	var p = new(TransitionTargetContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_transitionTarget
	return p
}

func InitEmptyTransitionTargetContext(p *TransitionTargetContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_transitionTarget
}

func (*TransitionTargetContext) IsTransitionTargetContext() {}

func NewTransitionTargetContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *TransitionTargetContext {
	var p = new(TransitionTargetContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_transitionTarget

	return p
}

func (s *TransitionTargetContext) GetParser() antlr.Parser { return s.parser }

func (s *TransitionTargetContext) NEXT() antlr.TerminalNode {
	return s.GetToken(ArcParserNEXT, 0)
}

func (s *TransitionTargetContext) MatchBlock() IMatchBlockContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IMatchBlockContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IMatchBlockContext)
}

func (s *TransitionTargetContext) IDENTIFIER() antlr.TerminalNode {
	return s.GetToken(ArcParserIDENTIFIER, 0)
}

func (s *TransitionTargetContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *TransitionTargetContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *TransitionTargetContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterTransitionTarget(s)
	}
}

func (s *TransitionTargetContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitTransitionTarget(s)
	}
}

func (p *ArcParser) TransitionTarget() (localctx ITransitionTargetContext) {
	localctx = NewTransitionTargetContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 36, ArcParserRULE_transitionTarget)
	p.SetState(304)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case ArcParserNEXT:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(301)
			p.Match(ArcParserNEXT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case ArcParserMATCH:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(302)
			p.MatchBlock()
		}

	case ArcParserIDENTIFIER:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(303)
			p.Match(ArcParserIDENTIFIER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	default:
		p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
		goto errorExit
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IMatchBlockContext is an interface to support dynamic dispatch.
type IMatchBlockContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	MATCH() antlr.TerminalNode
	LBRACE() antlr.TerminalNode
	AllMatchEntry() []IMatchEntryContext
	MatchEntry(i int) IMatchEntryContext
	RBRACE() antlr.TerminalNode
	AllCOMMA() []antlr.TerminalNode
	COMMA(i int) antlr.TerminalNode

	// IsMatchBlockContext differentiates from other interfaces.
	IsMatchBlockContext()
}

type MatchBlockContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyMatchBlockContext() *MatchBlockContext {
	var p = new(MatchBlockContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_matchBlock
	return p
}

func InitEmptyMatchBlockContext(p *MatchBlockContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_matchBlock
}

func (*MatchBlockContext) IsMatchBlockContext() {}

func NewMatchBlockContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *MatchBlockContext {
	var p = new(MatchBlockContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_matchBlock

	return p
}

func (s *MatchBlockContext) GetParser() antlr.Parser { return s.parser }

func (s *MatchBlockContext) MATCH() antlr.TerminalNode {
	return s.GetToken(ArcParserMATCH, 0)
}

func (s *MatchBlockContext) LBRACE() antlr.TerminalNode {
	return s.GetToken(ArcParserLBRACE, 0)
}

func (s *MatchBlockContext) AllMatchEntry() []IMatchEntryContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IMatchEntryContext); ok {
			len++
		}
	}

	tst := make([]IMatchEntryContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IMatchEntryContext); ok {
			tst[i] = t.(IMatchEntryContext)
			i++
		}
	}

	return tst
}

func (s *MatchBlockContext) MatchEntry(i int) IMatchEntryContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IMatchEntryContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IMatchEntryContext)
}

func (s *MatchBlockContext) RBRACE() antlr.TerminalNode {
	return s.GetToken(ArcParserRBRACE, 0)
}

func (s *MatchBlockContext) AllCOMMA() []antlr.TerminalNode {
	return s.GetTokens(ArcParserCOMMA)
}

func (s *MatchBlockContext) COMMA(i int) antlr.TerminalNode {
	return s.GetToken(ArcParserCOMMA, i)
}

func (s *MatchBlockContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *MatchBlockContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *MatchBlockContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterMatchBlock(s)
	}
}

func (s *MatchBlockContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitMatchBlock(s)
	}
}

func (p *ArcParser) MatchBlock() (localctx IMatchBlockContext) {
	localctx = NewMatchBlockContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 38, ArcParserRULE_matchBlock)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(306)
		p.Match(ArcParserMATCH)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(307)
		p.Match(ArcParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(308)
		p.MatchEntry()
	}
	p.SetState(313)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == ArcParserCOMMA {
		{
			p.SetState(309)
			p.Match(ArcParserCOMMA)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(310)
			p.MatchEntry()
		}

		p.SetState(315)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(316)
		p.Match(ArcParserRBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IMatchEntryContext is an interface to support dynamic dispatch.
type IMatchEntryContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	IDENTIFIER() antlr.TerminalNode
	TRANSITION() antlr.TerminalNode
	TransitionTarget() ITransitionTargetContext

	// IsMatchEntryContext differentiates from other interfaces.
	IsMatchEntryContext()
}

type MatchEntryContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyMatchEntryContext() *MatchEntryContext {
	var p = new(MatchEntryContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_matchEntry
	return p
}

func InitEmptyMatchEntryContext(p *MatchEntryContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_matchEntry
}

func (*MatchEntryContext) IsMatchEntryContext() {}

func NewMatchEntryContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *MatchEntryContext {
	var p = new(MatchEntryContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_matchEntry

	return p
}

func (s *MatchEntryContext) GetParser() antlr.Parser { return s.parser }

func (s *MatchEntryContext) IDENTIFIER() antlr.TerminalNode {
	return s.GetToken(ArcParserIDENTIFIER, 0)
}

func (s *MatchEntryContext) TRANSITION() antlr.TerminalNode {
	return s.GetToken(ArcParserTRANSITION, 0)
}

func (s *MatchEntryContext) TransitionTarget() ITransitionTargetContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ITransitionTargetContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ITransitionTargetContext)
}

func (s *MatchEntryContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *MatchEntryContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *MatchEntryContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterMatchEntry(s)
	}
}

func (s *MatchEntryContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitMatchEntry(s)
	}
}

func (p *ArcParser) MatchEntry() (localctx IMatchEntryContext) {
	localctx = NewMatchEntryContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 40, ArcParserRULE_matchEntry)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(318)
		p.Match(ArcParserIDENTIFIER)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(319)
		p.Match(ArcParserTRANSITION)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(320)
		p.TransitionTarget()
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IImperativeTransitionContext is an interface to support dynamic dispatch.
type IImperativeTransitionContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	Block() IBlockContext
	TRANSITION() antlr.TerminalNode
	MatchBlock() IMatchBlockContext

	// IsImperativeTransitionContext differentiates from other interfaces.
	IsImperativeTransitionContext()
}

type ImperativeTransitionContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyImperativeTransitionContext() *ImperativeTransitionContext {
	var p = new(ImperativeTransitionContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_imperativeTransition
	return p
}

func InitEmptyImperativeTransitionContext(p *ImperativeTransitionContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_imperativeTransition
}

func (*ImperativeTransitionContext) IsImperativeTransitionContext() {}

func NewImperativeTransitionContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ImperativeTransitionContext {
	var p = new(ImperativeTransitionContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_imperativeTransition

	return p
}

func (s *ImperativeTransitionContext) GetParser() antlr.Parser { return s.parser }

func (s *ImperativeTransitionContext) Block() IBlockContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IBlockContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IBlockContext)
}

func (s *ImperativeTransitionContext) TRANSITION() antlr.TerminalNode {
	return s.GetToken(ArcParserTRANSITION, 0)
}

func (s *ImperativeTransitionContext) MatchBlock() IMatchBlockContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IMatchBlockContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IMatchBlockContext)
}

func (s *ImperativeTransitionContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ImperativeTransitionContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ImperativeTransitionContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterImperativeTransition(s)
	}
}

func (s *ImperativeTransitionContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitImperativeTransition(s)
	}
}

func (p *ArcParser) ImperativeTransition() (localctx IImperativeTransitionContext) {
	localctx = NewImperativeTransitionContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 42, ArcParserRULE_imperativeTransition)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(322)
		p.Block()
	}
	{
		p.SetState(323)
		p.Match(ArcParserTRANSITION)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(324)
		p.MatchBlock()
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IFlowStatementContext is an interface to support dynamic dispatch.
type IFlowStatementContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AllRoutingTable() []IRoutingTableContext
	RoutingTable(i int) IRoutingTableContext
	AllFlowNode() []IFlowNodeContext
	FlowNode(i int) IFlowNodeContext
	AllARROW() []antlr.TerminalNode
	ARROW(i int) antlr.TerminalNode
	SEMICOLON() antlr.TerminalNode

	// IsFlowStatementContext differentiates from other interfaces.
	IsFlowStatementContext()
}

type FlowStatementContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyFlowStatementContext() *FlowStatementContext {
	var p = new(FlowStatementContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_flowStatement
	return p
}

func InitEmptyFlowStatementContext(p *FlowStatementContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_flowStatement
}

func (*FlowStatementContext) IsFlowStatementContext() {}

func NewFlowStatementContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *FlowStatementContext {
	var p = new(FlowStatementContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_flowStatement

	return p
}

func (s *FlowStatementContext) GetParser() antlr.Parser { return s.parser }

func (s *FlowStatementContext) AllRoutingTable() []IRoutingTableContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IRoutingTableContext); ok {
			len++
		}
	}

	tst := make([]IRoutingTableContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IRoutingTableContext); ok {
			tst[i] = t.(IRoutingTableContext)
			i++
		}
	}

	return tst
}

func (s *FlowStatementContext) RoutingTable(i int) IRoutingTableContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IRoutingTableContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IRoutingTableContext)
}

func (s *FlowStatementContext) AllFlowNode() []IFlowNodeContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IFlowNodeContext); ok {
			len++
		}
	}

	tst := make([]IFlowNodeContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IFlowNodeContext); ok {
			tst[i] = t.(IFlowNodeContext)
			i++
		}
	}

	return tst
}

func (s *FlowStatementContext) FlowNode(i int) IFlowNodeContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IFlowNodeContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IFlowNodeContext)
}

func (s *FlowStatementContext) AllARROW() []antlr.TerminalNode {
	return s.GetTokens(ArcParserARROW)
}

func (s *FlowStatementContext) ARROW(i int) antlr.TerminalNode {
	return s.GetToken(ArcParserARROW, i)
}

func (s *FlowStatementContext) SEMICOLON() antlr.TerminalNode {
	return s.GetToken(ArcParserSEMICOLON, 0)
}

func (s *FlowStatementContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *FlowStatementContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *FlowStatementContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterFlowStatement(s)
	}
}

func (s *FlowStatementContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitFlowStatement(s)
	}
}

func (p *ArcParser) FlowStatement() (localctx IFlowStatementContext) {
	localctx = NewFlowStatementContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 44, ArcParserRULE_flowStatement)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(328)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case ArcParserLBRACE:
		{
			p.SetState(326)
			p.RoutingTable()
		}

	case ArcParserNOW, ArcParserLEN, ArcParserCHAN, ArcParserRECV_CHAN, ArcParserSEND_CHAN, ArcParserI8, ArcParserI16, ArcParserI32, ArcParserI64, ArcParserU8, ArcParserU16, ArcParserU32, ArcParserU64, ArcParserF32, ArcParserF64, ArcParserSTR, ArcParserTIMESTAMP, ArcParserTIMESPAN, ArcParserSERIES, ArcParserRECV, ArcParserMINUS, ArcParserNOT, ArcParserLPAREN, ArcParserLBRACKET, ArcParserTEMPORAL_LITERAL, ArcParserFREQUENCY_LITERAL, ArcParserINTEGER_LITERAL, ArcParserFLOAT_LITERAL, ArcParserSTR_LITERAL, ArcParserIDENTIFIER:
		{
			p.SetState(327)
			p.FlowNode()
		}

	default:
		p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
		goto errorExit
	}
	p.SetState(335)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for ok := true; ok; ok = _la == ArcParserARROW {
		{
			p.SetState(330)
			p.Match(ArcParserARROW)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		p.SetState(333)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}

		switch p.GetTokenStream().LA(1) {
		case ArcParserLBRACE:
			{
				p.SetState(331)
				p.RoutingTable()
			}

		case ArcParserNOW, ArcParserLEN, ArcParserCHAN, ArcParserRECV_CHAN, ArcParserSEND_CHAN, ArcParserI8, ArcParserI16, ArcParserI32, ArcParserI64, ArcParserU8, ArcParserU16, ArcParserU32, ArcParserU64, ArcParserF32, ArcParserF64, ArcParserSTR, ArcParserTIMESTAMP, ArcParserTIMESPAN, ArcParserSERIES, ArcParserRECV, ArcParserMINUS, ArcParserNOT, ArcParserLPAREN, ArcParserLBRACKET, ArcParserTEMPORAL_LITERAL, ArcParserFREQUENCY_LITERAL, ArcParserINTEGER_LITERAL, ArcParserFLOAT_LITERAL, ArcParserSTR_LITERAL, ArcParserIDENTIFIER:
			{
				p.SetState(332)
				p.FlowNode()
			}

		default:
			p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
			goto errorExit
		}

		p.SetState(337)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	p.SetState(340)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ArcParserSEMICOLON {
		{
			p.SetState(339)
			p.Match(ArcParserSEMICOLON)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IRoutingTableContext is an interface to support dynamic dispatch.
type IRoutingTableContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	LBRACE() antlr.TerminalNode
	AllRoutingEntry() []IRoutingEntryContext
	RoutingEntry(i int) IRoutingEntryContext
	RBRACE() antlr.TerminalNode
	AllCOMMA() []antlr.TerminalNode
	COMMA(i int) antlr.TerminalNode

	// IsRoutingTableContext differentiates from other interfaces.
	IsRoutingTableContext()
}

type RoutingTableContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyRoutingTableContext() *RoutingTableContext {
	var p = new(RoutingTableContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_routingTable
	return p
}

func InitEmptyRoutingTableContext(p *RoutingTableContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_routingTable
}

func (*RoutingTableContext) IsRoutingTableContext() {}

func NewRoutingTableContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *RoutingTableContext {
	var p = new(RoutingTableContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_routingTable

	return p
}

func (s *RoutingTableContext) GetParser() antlr.Parser { return s.parser }

func (s *RoutingTableContext) LBRACE() antlr.TerminalNode {
	return s.GetToken(ArcParserLBRACE, 0)
}

func (s *RoutingTableContext) AllRoutingEntry() []IRoutingEntryContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IRoutingEntryContext); ok {
			len++
		}
	}

	tst := make([]IRoutingEntryContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IRoutingEntryContext); ok {
			tst[i] = t.(IRoutingEntryContext)
			i++
		}
	}

	return tst
}

func (s *RoutingTableContext) RoutingEntry(i int) IRoutingEntryContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IRoutingEntryContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IRoutingEntryContext)
}

func (s *RoutingTableContext) RBRACE() antlr.TerminalNode {
	return s.GetToken(ArcParserRBRACE, 0)
}

func (s *RoutingTableContext) AllCOMMA() []antlr.TerminalNode {
	return s.GetTokens(ArcParserCOMMA)
}

func (s *RoutingTableContext) COMMA(i int) antlr.TerminalNode {
	return s.GetToken(ArcParserCOMMA, i)
}

func (s *RoutingTableContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *RoutingTableContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *RoutingTableContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterRoutingTable(s)
	}
}

func (s *RoutingTableContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitRoutingTable(s)
	}
}

func (p *ArcParser) RoutingTable() (localctx IRoutingTableContext) {
	localctx = NewRoutingTableContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 46, ArcParserRULE_routingTable)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(342)
		p.Match(ArcParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(343)
		p.RoutingEntry()
	}
	p.SetState(348)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == ArcParserCOMMA {
		{
			p.SetState(344)
			p.Match(ArcParserCOMMA)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(345)
			p.RoutingEntry()
		}

		p.SetState(350)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(351)
		p.Match(ArcParserRBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IRoutingEntryContext is an interface to support dynamic dispatch.
type IRoutingEntryContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AllIDENTIFIER() []antlr.TerminalNode
	IDENTIFIER(i int) antlr.TerminalNode
	AllCOLON() []antlr.TerminalNode
	COLON(i int) antlr.TerminalNode
	AllFlowNode() []IFlowNodeContext
	FlowNode(i int) IFlowNodeContext
	AllARROW() []antlr.TerminalNode
	ARROW(i int) antlr.TerminalNode

	// IsRoutingEntryContext differentiates from other interfaces.
	IsRoutingEntryContext()
}

type RoutingEntryContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyRoutingEntryContext() *RoutingEntryContext {
	var p = new(RoutingEntryContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_routingEntry
	return p
}

func InitEmptyRoutingEntryContext(p *RoutingEntryContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_routingEntry
}

func (*RoutingEntryContext) IsRoutingEntryContext() {}

func NewRoutingEntryContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *RoutingEntryContext {
	var p = new(RoutingEntryContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_routingEntry

	return p
}

func (s *RoutingEntryContext) GetParser() antlr.Parser { return s.parser }

func (s *RoutingEntryContext) AllIDENTIFIER() []antlr.TerminalNode {
	return s.GetTokens(ArcParserIDENTIFIER)
}

func (s *RoutingEntryContext) IDENTIFIER(i int) antlr.TerminalNode {
	return s.GetToken(ArcParserIDENTIFIER, i)
}

func (s *RoutingEntryContext) AllCOLON() []antlr.TerminalNode {
	return s.GetTokens(ArcParserCOLON)
}

func (s *RoutingEntryContext) COLON(i int) antlr.TerminalNode {
	return s.GetToken(ArcParserCOLON, i)
}

func (s *RoutingEntryContext) AllFlowNode() []IFlowNodeContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IFlowNodeContext); ok {
			len++
		}
	}

	tst := make([]IFlowNodeContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IFlowNodeContext); ok {
			tst[i] = t.(IFlowNodeContext)
			i++
		}
	}

	return tst
}

func (s *RoutingEntryContext) FlowNode(i int) IFlowNodeContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IFlowNodeContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IFlowNodeContext)
}

func (s *RoutingEntryContext) AllARROW() []antlr.TerminalNode {
	return s.GetTokens(ArcParserARROW)
}

func (s *RoutingEntryContext) ARROW(i int) antlr.TerminalNode {
	return s.GetToken(ArcParserARROW, i)
}

func (s *RoutingEntryContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *RoutingEntryContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *RoutingEntryContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterRoutingEntry(s)
	}
}

func (s *RoutingEntryContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitRoutingEntry(s)
	}
}

func (p *ArcParser) RoutingEntry() (localctx IRoutingEntryContext) {
	localctx = NewRoutingEntryContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 48, ArcParserRULE_routingEntry)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(353)
		p.Match(ArcParserIDENTIFIER)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(354)
		p.Match(ArcParserCOLON)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(355)
		p.FlowNode()
	}
	p.SetState(360)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == ArcParserARROW {
		{
			p.SetState(356)
			p.Match(ArcParserARROW)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(357)
			p.FlowNode()
		}

		p.SetState(362)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	p.SetState(365)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ArcParserCOLON {
		{
			p.SetState(363)
			p.Match(ArcParserCOLON)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(364)
			p.Match(ArcParserIDENTIFIER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IFlowNodeContext is an interface to support dynamic dispatch.
type IFlowNodeContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	ChannelIdentifier() IChannelIdentifierContext
	Function() IFunctionContext
	Expression() IExpressionContext

	// IsFlowNodeContext differentiates from other interfaces.
	IsFlowNodeContext()
}

type FlowNodeContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyFlowNodeContext() *FlowNodeContext {
	var p = new(FlowNodeContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_flowNode
	return p
}

func InitEmptyFlowNodeContext(p *FlowNodeContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_flowNode
}

func (*FlowNodeContext) IsFlowNodeContext() {}

func NewFlowNodeContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *FlowNodeContext {
	var p = new(FlowNodeContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_flowNode

	return p
}

func (s *FlowNodeContext) GetParser() antlr.Parser { return s.parser }

func (s *FlowNodeContext) ChannelIdentifier() IChannelIdentifierContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IChannelIdentifierContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IChannelIdentifierContext)
}

func (s *FlowNodeContext) Function() IFunctionContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IFunctionContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IFunctionContext)
}

func (s *FlowNodeContext) Expression() IExpressionContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IExpressionContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IExpressionContext)
}

func (s *FlowNodeContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *FlowNodeContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *FlowNodeContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterFlowNode(s)
	}
}

func (s *FlowNodeContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitFlowNode(s)
	}
}

func (p *ArcParser) FlowNode() (localctx IFlowNodeContext) {
	localctx = NewFlowNodeContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 50, ArcParserRULE_flowNode)
	p.SetState(370)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 28, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(367)
			p.ChannelIdentifier()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(368)
			p.Function()
		}

	case 3:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(369)
			p.Expression()
		}

	case antlr.ATNInvalidAltNumber:
		goto errorExit
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IChannelIdentifierContext is an interface to support dynamic dispatch.
type IChannelIdentifierContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	IDENTIFIER() antlr.TerminalNode

	// IsChannelIdentifierContext differentiates from other interfaces.
	IsChannelIdentifierContext()
}

type ChannelIdentifierContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyChannelIdentifierContext() *ChannelIdentifierContext {
	var p = new(ChannelIdentifierContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_channelIdentifier
	return p
}

func InitEmptyChannelIdentifierContext(p *ChannelIdentifierContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_channelIdentifier
}

func (*ChannelIdentifierContext) IsChannelIdentifierContext() {}

func NewChannelIdentifierContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ChannelIdentifierContext {
	var p = new(ChannelIdentifierContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_channelIdentifier

	return p
}

func (s *ChannelIdentifierContext) GetParser() antlr.Parser { return s.parser }

func (s *ChannelIdentifierContext) IDENTIFIER() antlr.TerminalNode {
	return s.GetToken(ArcParserIDENTIFIER, 0)
}

func (s *ChannelIdentifierContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ChannelIdentifierContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ChannelIdentifierContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterChannelIdentifier(s)
	}
}

func (s *ChannelIdentifierContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitChannelIdentifier(s)
	}
}

func (p *ArcParser) ChannelIdentifier() (localctx IChannelIdentifierContext) {
	localctx = NewChannelIdentifierContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 52, ArcParserRULE_channelIdentifier)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(372)
		p.Match(ArcParserIDENTIFIER)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IFunctionContext is an interface to support dynamic dispatch.
type IFunctionContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	IDENTIFIER() antlr.TerminalNode
	ConfigValues() IConfigValuesContext
	Arguments() IArgumentsContext

	// IsFunctionContext differentiates from other interfaces.
	IsFunctionContext()
}

type FunctionContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyFunctionContext() *FunctionContext {
	var p = new(FunctionContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_function
	return p
}

func InitEmptyFunctionContext(p *FunctionContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_function
}

func (*FunctionContext) IsFunctionContext() {}

func NewFunctionContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *FunctionContext {
	var p = new(FunctionContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_function

	return p
}

func (s *FunctionContext) GetParser() antlr.Parser { return s.parser }

func (s *FunctionContext) IDENTIFIER() antlr.TerminalNode {
	return s.GetToken(ArcParserIDENTIFIER, 0)
}

func (s *FunctionContext) ConfigValues() IConfigValuesContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IConfigValuesContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IConfigValuesContext)
}

func (s *FunctionContext) Arguments() IArgumentsContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IArgumentsContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IArgumentsContext)
}

func (s *FunctionContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *FunctionContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *FunctionContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterFunction(s)
	}
}

func (s *FunctionContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitFunction(s)
	}
}

func (p *ArcParser) Function() (localctx IFunctionContext) {
	localctx = NewFunctionContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 54, ArcParserRULE_function)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(374)
		p.Match(ArcParserIDENTIFIER)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(376)
	p.GetErrorHandler().Sync(p)

	if p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 29, p.GetParserRuleContext()) == 1 {
		{
			p.SetState(375)
			p.ConfigValues()
		}

	} else if p.HasError() { // JIM
		goto errorExit
	}
	p.SetState(379)
	p.GetErrorHandler().Sync(p)

	if p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 30, p.GetParserRuleContext()) == 1 {
		{
			p.SetState(378)
			p.Arguments()
		}

	} else if p.HasError() { // JIM
		goto errorExit
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IConfigValuesContext is an interface to support dynamic dispatch.
type IConfigValuesContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	LBRACE() antlr.TerminalNode
	RBRACE() antlr.TerminalNode
	NamedConfigValues() INamedConfigValuesContext
	AnonymousConfigValues() IAnonymousConfigValuesContext

	// IsConfigValuesContext differentiates from other interfaces.
	IsConfigValuesContext()
}

type ConfigValuesContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyConfigValuesContext() *ConfigValuesContext {
	var p = new(ConfigValuesContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_configValues
	return p
}

func InitEmptyConfigValuesContext(p *ConfigValuesContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_configValues
}

func (*ConfigValuesContext) IsConfigValuesContext() {}

func NewConfigValuesContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ConfigValuesContext {
	var p = new(ConfigValuesContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_configValues

	return p
}

func (s *ConfigValuesContext) GetParser() antlr.Parser { return s.parser }

func (s *ConfigValuesContext) LBRACE() antlr.TerminalNode {
	return s.GetToken(ArcParserLBRACE, 0)
}

func (s *ConfigValuesContext) RBRACE() antlr.TerminalNode {
	return s.GetToken(ArcParserRBRACE, 0)
}

func (s *ConfigValuesContext) NamedConfigValues() INamedConfigValuesContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(INamedConfigValuesContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(INamedConfigValuesContext)
}

func (s *ConfigValuesContext) AnonymousConfigValues() IAnonymousConfigValuesContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IAnonymousConfigValuesContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IAnonymousConfigValuesContext)
}

func (s *ConfigValuesContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ConfigValuesContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ConfigValuesContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterConfigValues(s)
	}
}

func (s *ConfigValuesContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitConfigValues(s)
	}
}

func (p *ArcParser) ConfigValues() (localctx IConfigValuesContext) {
	localctx = NewConfigValuesContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 56, ArcParserRULE_configValues)
	p.SetState(391)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 31, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(381)
			p.Match(ArcParserLBRACE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(382)
			p.Match(ArcParserRBRACE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(383)
			p.Match(ArcParserLBRACE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(384)
			p.NamedConfigValues()
		}
		{
			p.SetState(385)
			p.Match(ArcParserRBRACE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 3:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(387)
			p.Match(ArcParserLBRACE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(388)
			p.AnonymousConfigValues()
		}
		{
			p.SetState(389)
			p.Match(ArcParserRBRACE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case antlr.ATNInvalidAltNumber:
		goto errorExit
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// INamedConfigValuesContext is an interface to support dynamic dispatch.
type INamedConfigValuesContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AllNamedConfigValue() []INamedConfigValueContext
	NamedConfigValue(i int) INamedConfigValueContext
	AllCOMMA() []antlr.TerminalNode
	COMMA(i int) antlr.TerminalNode

	// IsNamedConfigValuesContext differentiates from other interfaces.
	IsNamedConfigValuesContext()
}

type NamedConfigValuesContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyNamedConfigValuesContext() *NamedConfigValuesContext {
	var p = new(NamedConfigValuesContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_namedConfigValues
	return p
}

func InitEmptyNamedConfigValuesContext(p *NamedConfigValuesContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_namedConfigValues
}

func (*NamedConfigValuesContext) IsNamedConfigValuesContext() {}

func NewNamedConfigValuesContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *NamedConfigValuesContext {
	var p = new(NamedConfigValuesContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_namedConfigValues

	return p
}

func (s *NamedConfigValuesContext) GetParser() antlr.Parser { return s.parser }

func (s *NamedConfigValuesContext) AllNamedConfigValue() []INamedConfigValueContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(INamedConfigValueContext); ok {
			len++
		}
	}

	tst := make([]INamedConfigValueContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(INamedConfigValueContext); ok {
			tst[i] = t.(INamedConfigValueContext)
			i++
		}
	}

	return tst
}

func (s *NamedConfigValuesContext) NamedConfigValue(i int) INamedConfigValueContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(INamedConfigValueContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(INamedConfigValueContext)
}

func (s *NamedConfigValuesContext) AllCOMMA() []antlr.TerminalNode {
	return s.GetTokens(ArcParserCOMMA)
}

func (s *NamedConfigValuesContext) COMMA(i int) antlr.TerminalNode {
	return s.GetToken(ArcParserCOMMA, i)
}

func (s *NamedConfigValuesContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *NamedConfigValuesContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *NamedConfigValuesContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterNamedConfigValues(s)
	}
}

func (s *NamedConfigValuesContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitNamedConfigValues(s)
	}
}

func (p *ArcParser) NamedConfigValues() (localctx INamedConfigValuesContext) {
	localctx = NewNamedConfigValuesContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 58, ArcParserRULE_namedConfigValues)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(393)
		p.NamedConfigValue()
	}
	p.SetState(398)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == ArcParserCOMMA {
		{
			p.SetState(394)
			p.Match(ArcParserCOMMA)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(395)
			p.NamedConfigValue()
		}

		p.SetState(400)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// INamedConfigValueContext is an interface to support dynamic dispatch.
type INamedConfigValueContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	IDENTIFIER() antlr.TerminalNode
	ASSIGN() antlr.TerminalNode
	Expression() IExpressionContext

	// IsNamedConfigValueContext differentiates from other interfaces.
	IsNamedConfigValueContext()
}

type NamedConfigValueContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyNamedConfigValueContext() *NamedConfigValueContext {
	var p = new(NamedConfigValueContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_namedConfigValue
	return p
}

func InitEmptyNamedConfigValueContext(p *NamedConfigValueContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_namedConfigValue
}

func (*NamedConfigValueContext) IsNamedConfigValueContext() {}

func NewNamedConfigValueContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *NamedConfigValueContext {
	var p = new(NamedConfigValueContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_namedConfigValue

	return p
}

func (s *NamedConfigValueContext) GetParser() antlr.Parser { return s.parser }

func (s *NamedConfigValueContext) IDENTIFIER() antlr.TerminalNode {
	return s.GetToken(ArcParserIDENTIFIER, 0)
}

func (s *NamedConfigValueContext) ASSIGN() antlr.TerminalNode {
	return s.GetToken(ArcParserASSIGN, 0)
}

func (s *NamedConfigValueContext) Expression() IExpressionContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IExpressionContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IExpressionContext)
}

func (s *NamedConfigValueContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *NamedConfigValueContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *NamedConfigValueContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterNamedConfigValue(s)
	}
}

func (s *NamedConfigValueContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitNamedConfigValue(s)
	}
}

func (p *ArcParser) NamedConfigValue() (localctx INamedConfigValueContext) {
	localctx = NewNamedConfigValueContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 60, ArcParserRULE_namedConfigValue)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(401)
		p.Match(ArcParserIDENTIFIER)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(402)
		p.Match(ArcParserASSIGN)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(403)
		p.Expression()
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IAnonymousConfigValuesContext is an interface to support dynamic dispatch.
type IAnonymousConfigValuesContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AllExpression() []IExpressionContext
	Expression(i int) IExpressionContext
	AllCOMMA() []antlr.TerminalNode
	COMMA(i int) antlr.TerminalNode

	// IsAnonymousConfigValuesContext differentiates from other interfaces.
	IsAnonymousConfigValuesContext()
}

type AnonymousConfigValuesContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyAnonymousConfigValuesContext() *AnonymousConfigValuesContext {
	var p = new(AnonymousConfigValuesContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_anonymousConfigValues
	return p
}

func InitEmptyAnonymousConfigValuesContext(p *AnonymousConfigValuesContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_anonymousConfigValues
}

func (*AnonymousConfigValuesContext) IsAnonymousConfigValuesContext() {}

func NewAnonymousConfigValuesContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *AnonymousConfigValuesContext {
	var p = new(AnonymousConfigValuesContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_anonymousConfigValues

	return p
}

func (s *AnonymousConfigValuesContext) GetParser() antlr.Parser { return s.parser }

func (s *AnonymousConfigValuesContext) AllExpression() []IExpressionContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IExpressionContext); ok {
			len++
		}
	}

	tst := make([]IExpressionContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IExpressionContext); ok {
			tst[i] = t.(IExpressionContext)
			i++
		}
	}

	return tst
}

func (s *AnonymousConfigValuesContext) Expression(i int) IExpressionContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IExpressionContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IExpressionContext)
}

func (s *AnonymousConfigValuesContext) AllCOMMA() []antlr.TerminalNode {
	return s.GetTokens(ArcParserCOMMA)
}

func (s *AnonymousConfigValuesContext) COMMA(i int) antlr.TerminalNode {
	return s.GetToken(ArcParserCOMMA, i)
}

func (s *AnonymousConfigValuesContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *AnonymousConfigValuesContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *AnonymousConfigValuesContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterAnonymousConfigValues(s)
	}
}

func (s *AnonymousConfigValuesContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitAnonymousConfigValues(s)
	}
}

func (p *ArcParser) AnonymousConfigValues() (localctx IAnonymousConfigValuesContext) {
	localctx = NewAnonymousConfigValuesContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 62, ArcParserRULE_anonymousConfigValues)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(405)
		p.Expression()
	}
	p.SetState(410)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == ArcParserCOMMA {
		{
			p.SetState(406)
			p.Match(ArcParserCOMMA)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(407)
			p.Expression()
		}

		p.SetState(412)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IArgumentsContext is an interface to support dynamic dispatch.
type IArgumentsContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	LPAREN() antlr.TerminalNode
	RPAREN() antlr.TerminalNode
	ArgumentList() IArgumentListContext

	// IsArgumentsContext differentiates from other interfaces.
	IsArgumentsContext()
}

type ArgumentsContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyArgumentsContext() *ArgumentsContext {
	var p = new(ArgumentsContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_arguments
	return p
}

func InitEmptyArgumentsContext(p *ArgumentsContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_arguments
}

func (*ArgumentsContext) IsArgumentsContext() {}

func NewArgumentsContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ArgumentsContext {
	var p = new(ArgumentsContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_arguments

	return p
}

func (s *ArgumentsContext) GetParser() antlr.Parser { return s.parser }

func (s *ArgumentsContext) LPAREN() antlr.TerminalNode {
	return s.GetToken(ArcParserLPAREN, 0)
}

func (s *ArgumentsContext) RPAREN() antlr.TerminalNode {
	return s.GetToken(ArcParserRPAREN, 0)
}

func (s *ArgumentsContext) ArgumentList() IArgumentListContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IArgumentListContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IArgumentListContext)
}

func (s *ArgumentsContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ArgumentsContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ArgumentsContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterArguments(s)
	}
}

func (s *ArgumentsContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitArguments(s)
	}
}

func (p *ArcParser) Arguments() (localctx IArgumentsContext) {
	localctx = NewArgumentsContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 64, ArcParserRULE_arguments)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(413)
		p.Match(ArcParserLPAREN)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(415)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&-278378716801796000) != 0 {
		{
			p.SetState(414)
			p.ArgumentList()
		}

	}
	{
		p.SetState(417)
		p.Match(ArcParserRPAREN)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IArgumentListContext is an interface to support dynamic dispatch.
type IArgumentListContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AllExpression() []IExpressionContext
	Expression(i int) IExpressionContext
	AllCOMMA() []antlr.TerminalNode
	COMMA(i int) antlr.TerminalNode

	// IsArgumentListContext differentiates from other interfaces.
	IsArgumentListContext()
}

type ArgumentListContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyArgumentListContext() *ArgumentListContext {
	var p = new(ArgumentListContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_argumentList
	return p
}

func InitEmptyArgumentListContext(p *ArgumentListContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_argumentList
}

func (*ArgumentListContext) IsArgumentListContext() {}

func NewArgumentListContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ArgumentListContext {
	var p = new(ArgumentListContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_argumentList

	return p
}

func (s *ArgumentListContext) GetParser() antlr.Parser { return s.parser }

func (s *ArgumentListContext) AllExpression() []IExpressionContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IExpressionContext); ok {
			len++
		}
	}

	tst := make([]IExpressionContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IExpressionContext); ok {
			tst[i] = t.(IExpressionContext)
			i++
		}
	}

	return tst
}

func (s *ArgumentListContext) Expression(i int) IExpressionContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IExpressionContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IExpressionContext)
}

func (s *ArgumentListContext) AllCOMMA() []antlr.TerminalNode {
	return s.GetTokens(ArcParserCOMMA)
}

func (s *ArgumentListContext) COMMA(i int) antlr.TerminalNode {
	return s.GetToken(ArcParserCOMMA, i)
}

func (s *ArgumentListContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ArgumentListContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ArgumentListContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterArgumentList(s)
	}
}

func (s *ArgumentListContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitArgumentList(s)
	}
}

func (p *ArcParser) ArgumentList() (localctx IArgumentListContext) {
	localctx = NewArgumentListContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 66, ArcParserRULE_argumentList)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(419)
		p.Expression()
	}
	p.SetState(424)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == ArcParserCOMMA {
		{
			p.SetState(420)
			p.Match(ArcParserCOMMA)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(421)
			p.Expression()
		}

		p.SetState(426)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IBlockContext is an interface to support dynamic dispatch.
type IBlockContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	LBRACE() antlr.TerminalNode
	RBRACE() antlr.TerminalNode
	AllStatement() []IStatementContext
	Statement(i int) IStatementContext

	// IsBlockContext differentiates from other interfaces.
	IsBlockContext()
}

type BlockContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyBlockContext() *BlockContext {
	var p = new(BlockContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_block
	return p
}

func InitEmptyBlockContext(p *BlockContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_block
}

func (*BlockContext) IsBlockContext() {}

func NewBlockContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *BlockContext {
	var p = new(BlockContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_block

	return p
}

func (s *BlockContext) GetParser() antlr.Parser { return s.parser }

func (s *BlockContext) LBRACE() antlr.TerminalNode {
	return s.GetToken(ArcParserLBRACE, 0)
}

func (s *BlockContext) RBRACE() antlr.TerminalNode {
	return s.GetToken(ArcParserRBRACE, 0)
}

func (s *BlockContext) AllStatement() []IStatementContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IStatementContext); ok {
			len++
		}
	}

	tst := make([]IStatementContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IStatementContext); ok {
			tst[i] = t.(IStatementContext)
			i++
		}
	}

	return tst
}

func (s *BlockContext) Statement(i int) IStatementContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IStatementContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IStatementContext)
}

func (s *BlockContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *BlockContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *BlockContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterBlock(s)
	}
}

func (s *BlockContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitBlock(s)
	}
}

func (p *ArcParser) Block() (localctx IBlockContext) {
	localctx = NewBlockContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 68, ArcParserRULE_block)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(427)
		p.Match(ArcParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(431)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&-278378716801795980) != 0 {
		{
			p.SetState(428)
			p.Statement()
		}

		p.SetState(433)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(434)
		p.Match(ArcParserRBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IStatementContext is an interface to support dynamic dispatch.
type IStatementContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	VariableDeclaration() IVariableDeclarationContext
	ChannelOperation() IChannelOperationContext
	Assignment() IAssignmentContext
	IfStatement() IIfStatementContext
	ReturnStatement() IReturnStatementContext
	FunctionCall() IFunctionCallContext
	Expression() IExpressionContext

	// IsStatementContext differentiates from other interfaces.
	IsStatementContext()
}

type StatementContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyStatementContext() *StatementContext {
	var p = new(StatementContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_statement
	return p
}

func InitEmptyStatementContext(p *StatementContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_statement
}

func (*StatementContext) IsStatementContext() {}

func NewStatementContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *StatementContext {
	var p = new(StatementContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_statement

	return p
}

func (s *StatementContext) GetParser() antlr.Parser { return s.parser }

func (s *StatementContext) VariableDeclaration() IVariableDeclarationContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IVariableDeclarationContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IVariableDeclarationContext)
}

func (s *StatementContext) ChannelOperation() IChannelOperationContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IChannelOperationContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IChannelOperationContext)
}

func (s *StatementContext) Assignment() IAssignmentContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IAssignmentContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IAssignmentContext)
}

func (s *StatementContext) IfStatement() IIfStatementContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IIfStatementContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IIfStatementContext)
}

func (s *StatementContext) ReturnStatement() IReturnStatementContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IReturnStatementContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IReturnStatementContext)
}

func (s *StatementContext) FunctionCall() IFunctionCallContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IFunctionCallContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IFunctionCallContext)
}

func (s *StatementContext) Expression() IExpressionContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IExpressionContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IExpressionContext)
}

func (s *StatementContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *StatementContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *StatementContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterStatement(s)
	}
}

func (s *StatementContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitStatement(s)
	}
}

func (p *ArcParser) Statement() (localctx IStatementContext) {
	localctx = NewStatementContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 70, ArcParserRULE_statement)
	p.SetState(443)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 37, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(436)
			p.VariableDeclaration()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(437)
			p.ChannelOperation()
		}

	case 3:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(438)
			p.Assignment()
		}

	case 4:
		p.EnterOuterAlt(localctx, 4)
		{
			p.SetState(439)
			p.IfStatement()
		}

	case 5:
		p.EnterOuterAlt(localctx, 5)
		{
			p.SetState(440)
			p.ReturnStatement()
		}

	case 6:
		p.EnterOuterAlt(localctx, 6)
		{
			p.SetState(441)
			p.FunctionCall()
		}

	case 7:
		p.EnterOuterAlt(localctx, 7)
		{
			p.SetState(442)
			p.Expression()
		}

	case antlr.ATNInvalidAltNumber:
		goto errorExit
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IVariableDeclarationContext is an interface to support dynamic dispatch.
type IVariableDeclarationContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	LocalVariable() ILocalVariableContext
	StatefulVariable() IStatefulVariableContext

	// IsVariableDeclarationContext differentiates from other interfaces.
	IsVariableDeclarationContext()
}

type VariableDeclarationContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyVariableDeclarationContext() *VariableDeclarationContext {
	var p = new(VariableDeclarationContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_variableDeclaration
	return p
}

func InitEmptyVariableDeclarationContext(p *VariableDeclarationContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_variableDeclaration
}

func (*VariableDeclarationContext) IsVariableDeclarationContext() {}

func NewVariableDeclarationContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *VariableDeclarationContext {
	var p = new(VariableDeclarationContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_variableDeclaration

	return p
}

func (s *VariableDeclarationContext) GetParser() antlr.Parser { return s.parser }

func (s *VariableDeclarationContext) LocalVariable() ILocalVariableContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ILocalVariableContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ILocalVariableContext)
}

func (s *VariableDeclarationContext) StatefulVariable() IStatefulVariableContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IStatefulVariableContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IStatefulVariableContext)
}

func (s *VariableDeclarationContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *VariableDeclarationContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *VariableDeclarationContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterVariableDeclaration(s)
	}
}

func (s *VariableDeclarationContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitVariableDeclaration(s)
	}
}

func (p *ArcParser) VariableDeclaration() (localctx IVariableDeclarationContext) {
	localctx = NewVariableDeclarationContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 72, ArcParserRULE_variableDeclaration)
	p.SetState(447)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 38, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(445)
			p.LocalVariable()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(446)
			p.StatefulVariable()
		}

	case antlr.ATNInvalidAltNumber:
		goto errorExit
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// ILocalVariableContext is an interface to support dynamic dispatch.
type ILocalVariableContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	IDENTIFIER() antlr.TerminalNode
	DECLARE() antlr.TerminalNode
	Expression() IExpressionContext
	Type_() ITypeContext

	// IsLocalVariableContext differentiates from other interfaces.
	IsLocalVariableContext()
}

type LocalVariableContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyLocalVariableContext() *LocalVariableContext {
	var p = new(LocalVariableContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_localVariable
	return p
}

func InitEmptyLocalVariableContext(p *LocalVariableContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_localVariable
}

func (*LocalVariableContext) IsLocalVariableContext() {}

func NewLocalVariableContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *LocalVariableContext {
	var p = new(LocalVariableContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_localVariable

	return p
}

func (s *LocalVariableContext) GetParser() antlr.Parser { return s.parser }

func (s *LocalVariableContext) IDENTIFIER() antlr.TerminalNode {
	return s.GetToken(ArcParserIDENTIFIER, 0)
}

func (s *LocalVariableContext) DECLARE() antlr.TerminalNode {
	return s.GetToken(ArcParserDECLARE, 0)
}

func (s *LocalVariableContext) Expression() IExpressionContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IExpressionContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IExpressionContext)
}

func (s *LocalVariableContext) Type_() ITypeContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ITypeContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ITypeContext)
}

func (s *LocalVariableContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *LocalVariableContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *LocalVariableContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterLocalVariable(s)
	}
}

func (s *LocalVariableContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitLocalVariable(s)
	}
}

func (p *ArcParser) LocalVariable() (localctx ILocalVariableContext) {
	localctx = NewLocalVariableContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 74, ArcParserRULE_localVariable)
	p.SetState(457)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 39, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(449)
			p.Match(ArcParserIDENTIFIER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(450)
			p.Match(ArcParserDECLARE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(451)
			p.Expression()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(452)
			p.Match(ArcParserIDENTIFIER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(453)
			p.Type_()
		}
		{
			p.SetState(454)
			p.Match(ArcParserDECLARE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(455)
			p.Expression()
		}

	case antlr.ATNInvalidAltNumber:
		goto errorExit
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IStatefulVariableContext is an interface to support dynamic dispatch.
type IStatefulVariableContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	IDENTIFIER() antlr.TerminalNode
	STATE_DECLARE() antlr.TerminalNode
	Expression() IExpressionContext
	Type_() ITypeContext

	// IsStatefulVariableContext differentiates from other interfaces.
	IsStatefulVariableContext()
}

type StatefulVariableContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyStatefulVariableContext() *StatefulVariableContext {
	var p = new(StatefulVariableContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_statefulVariable
	return p
}

func InitEmptyStatefulVariableContext(p *StatefulVariableContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_statefulVariable
}

func (*StatefulVariableContext) IsStatefulVariableContext() {}

func NewStatefulVariableContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *StatefulVariableContext {
	var p = new(StatefulVariableContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_statefulVariable

	return p
}

func (s *StatefulVariableContext) GetParser() antlr.Parser { return s.parser }

func (s *StatefulVariableContext) IDENTIFIER() antlr.TerminalNode {
	return s.GetToken(ArcParserIDENTIFIER, 0)
}

func (s *StatefulVariableContext) STATE_DECLARE() antlr.TerminalNode {
	return s.GetToken(ArcParserSTATE_DECLARE, 0)
}

func (s *StatefulVariableContext) Expression() IExpressionContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IExpressionContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IExpressionContext)
}

func (s *StatefulVariableContext) Type_() ITypeContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ITypeContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ITypeContext)
}

func (s *StatefulVariableContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *StatefulVariableContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *StatefulVariableContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterStatefulVariable(s)
	}
}

func (s *StatefulVariableContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitStatefulVariable(s)
	}
}

func (p *ArcParser) StatefulVariable() (localctx IStatefulVariableContext) {
	localctx = NewStatefulVariableContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 76, ArcParserRULE_statefulVariable)
	p.SetState(467)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 40, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(459)
			p.Match(ArcParserIDENTIFIER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(460)
			p.Match(ArcParserSTATE_DECLARE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(461)
			p.Expression()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(462)
			p.Match(ArcParserIDENTIFIER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(463)
			p.Type_()
		}
		{
			p.SetState(464)
			p.Match(ArcParserSTATE_DECLARE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(465)
			p.Expression()
		}

	case antlr.ATNInvalidAltNumber:
		goto errorExit
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IAssignmentContext is an interface to support dynamic dispatch.
type IAssignmentContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	IDENTIFIER() antlr.TerminalNode
	ASSIGN() antlr.TerminalNode
	Expression() IExpressionContext

	// IsAssignmentContext differentiates from other interfaces.
	IsAssignmentContext()
}

type AssignmentContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyAssignmentContext() *AssignmentContext {
	var p = new(AssignmentContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_assignment
	return p
}

func InitEmptyAssignmentContext(p *AssignmentContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_assignment
}

func (*AssignmentContext) IsAssignmentContext() {}

func NewAssignmentContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *AssignmentContext {
	var p = new(AssignmentContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_assignment

	return p
}

func (s *AssignmentContext) GetParser() antlr.Parser { return s.parser }

func (s *AssignmentContext) IDENTIFIER() antlr.TerminalNode {
	return s.GetToken(ArcParserIDENTIFIER, 0)
}

func (s *AssignmentContext) ASSIGN() antlr.TerminalNode {
	return s.GetToken(ArcParserASSIGN, 0)
}

func (s *AssignmentContext) Expression() IExpressionContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IExpressionContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IExpressionContext)
}

func (s *AssignmentContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *AssignmentContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *AssignmentContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterAssignment(s)
	}
}

func (s *AssignmentContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitAssignment(s)
	}
}

func (p *ArcParser) Assignment() (localctx IAssignmentContext) {
	localctx = NewAssignmentContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 78, ArcParserRULE_assignment)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(469)
		p.Match(ArcParserIDENTIFIER)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(470)
		p.Match(ArcParserASSIGN)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(471)
		p.Expression()
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IChannelOperationContext is an interface to support dynamic dispatch.
type IChannelOperationContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	ChannelWrite() IChannelWriteContext
	ChannelRead() IChannelReadContext

	// IsChannelOperationContext differentiates from other interfaces.
	IsChannelOperationContext()
}

type ChannelOperationContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyChannelOperationContext() *ChannelOperationContext {
	var p = new(ChannelOperationContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_channelOperation
	return p
}

func InitEmptyChannelOperationContext(p *ChannelOperationContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_channelOperation
}

func (*ChannelOperationContext) IsChannelOperationContext() {}

func NewChannelOperationContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ChannelOperationContext {
	var p = new(ChannelOperationContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_channelOperation

	return p
}

func (s *ChannelOperationContext) GetParser() antlr.Parser { return s.parser }

func (s *ChannelOperationContext) ChannelWrite() IChannelWriteContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IChannelWriteContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IChannelWriteContext)
}

func (s *ChannelOperationContext) ChannelRead() IChannelReadContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IChannelReadContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IChannelReadContext)
}

func (s *ChannelOperationContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ChannelOperationContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ChannelOperationContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterChannelOperation(s)
	}
}

func (s *ChannelOperationContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitChannelOperation(s)
	}
}

func (p *ArcParser) ChannelOperation() (localctx IChannelOperationContext) {
	localctx = NewChannelOperationContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 80, ArcParserRULE_channelOperation)
	p.SetState(475)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 41, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(473)
			p.ChannelWrite()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(474)
			p.ChannelRead()
		}

	case antlr.ATNInvalidAltNumber:
		goto errorExit
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IChannelWriteContext is an interface to support dynamic dispatch.
type IChannelWriteContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	Expression() IExpressionContext
	ARROW() antlr.TerminalNode
	IDENTIFIER() antlr.TerminalNode
	RECV() antlr.TerminalNode

	// IsChannelWriteContext differentiates from other interfaces.
	IsChannelWriteContext()
}

type ChannelWriteContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyChannelWriteContext() *ChannelWriteContext {
	var p = new(ChannelWriteContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_channelWrite
	return p
}

func InitEmptyChannelWriteContext(p *ChannelWriteContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_channelWrite
}

func (*ChannelWriteContext) IsChannelWriteContext() {}

func NewChannelWriteContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ChannelWriteContext {
	var p = new(ChannelWriteContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_channelWrite

	return p
}

func (s *ChannelWriteContext) GetParser() antlr.Parser { return s.parser }

func (s *ChannelWriteContext) Expression() IExpressionContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IExpressionContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IExpressionContext)
}

func (s *ChannelWriteContext) ARROW() antlr.TerminalNode {
	return s.GetToken(ArcParserARROW, 0)
}

func (s *ChannelWriteContext) IDENTIFIER() antlr.TerminalNode {
	return s.GetToken(ArcParserIDENTIFIER, 0)
}

func (s *ChannelWriteContext) RECV() antlr.TerminalNode {
	return s.GetToken(ArcParserRECV, 0)
}

func (s *ChannelWriteContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ChannelWriteContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ChannelWriteContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterChannelWrite(s)
	}
}

func (s *ChannelWriteContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitChannelWrite(s)
	}
}

func (p *ArcParser) ChannelWrite() (localctx IChannelWriteContext) {
	localctx = NewChannelWriteContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 82, ArcParserRULE_channelWrite)
	p.SetState(484)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 42, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(477)
			p.Expression()
		}
		{
			p.SetState(478)
			p.Match(ArcParserARROW)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(479)
			p.Match(ArcParserIDENTIFIER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(481)
			p.Match(ArcParserIDENTIFIER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(482)
			p.Match(ArcParserRECV)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(483)
			p.Expression()
		}

	case antlr.ATNInvalidAltNumber:
		goto errorExit
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IChannelReadContext is an interface to support dynamic dispatch.
type IChannelReadContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	BlockingRead() IBlockingReadContext
	NonBlockingRead() INonBlockingReadContext

	// IsChannelReadContext differentiates from other interfaces.
	IsChannelReadContext()
}

type ChannelReadContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyChannelReadContext() *ChannelReadContext {
	var p = new(ChannelReadContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_channelRead
	return p
}

func InitEmptyChannelReadContext(p *ChannelReadContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_channelRead
}

func (*ChannelReadContext) IsChannelReadContext() {}

func NewChannelReadContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ChannelReadContext {
	var p = new(ChannelReadContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_channelRead

	return p
}

func (s *ChannelReadContext) GetParser() antlr.Parser { return s.parser }

func (s *ChannelReadContext) BlockingRead() IBlockingReadContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IBlockingReadContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IBlockingReadContext)
}

func (s *ChannelReadContext) NonBlockingRead() INonBlockingReadContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(INonBlockingReadContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(INonBlockingReadContext)
}

func (s *ChannelReadContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ChannelReadContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ChannelReadContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterChannelRead(s)
	}
}

func (s *ChannelReadContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitChannelRead(s)
	}
}

func (p *ArcParser) ChannelRead() (localctx IChannelReadContext) {
	localctx = NewChannelReadContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 84, ArcParserRULE_channelRead)
	p.SetState(488)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 43, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(486)
			p.BlockingRead()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(487)
			p.NonBlockingRead()
		}

	case antlr.ATNInvalidAltNumber:
		goto errorExit
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IBlockingReadContext is an interface to support dynamic dispatch.
type IBlockingReadContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AllIDENTIFIER() []antlr.TerminalNode
	IDENTIFIER(i int) antlr.TerminalNode
	DECLARE() antlr.TerminalNode
	RECV() antlr.TerminalNode

	// IsBlockingReadContext differentiates from other interfaces.
	IsBlockingReadContext()
}

type BlockingReadContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyBlockingReadContext() *BlockingReadContext {
	var p = new(BlockingReadContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_blockingRead
	return p
}

func InitEmptyBlockingReadContext(p *BlockingReadContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_blockingRead
}

func (*BlockingReadContext) IsBlockingReadContext() {}

func NewBlockingReadContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *BlockingReadContext {
	var p = new(BlockingReadContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_blockingRead

	return p
}

func (s *BlockingReadContext) GetParser() antlr.Parser { return s.parser }

func (s *BlockingReadContext) AllIDENTIFIER() []antlr.TerminalNode {
	return s.GetTokens(ArcParserIDENTIFIER)
}

func (s *BlockingReadContext) IDENTIFIER(i int) antlr.TerminalNode {
	return s.GetToken(ArcParserIDENTIFIER, i)
}

func (s *BlockingReadContext) DECLARE() antlr.TerminalNode {
	return s.GetToken(ArcParserDECLARE, 0)
}

func (s *BlockingReadContext) RECV() antlr.TerminalNode {
	return s.GetToken(ArcParserRECV, 0)
}

func (s *BlockingReadContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *BlockingReadContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *BlockingReadContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterBlockingRead(s)
	}
}

func (s *BlockingReadContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitBlockingRead(s)
	}
}

func (p *ArcParser) BlockingRead() (localctx IBlockingReadContext) {
	localctx = NewBlockingReadContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 86, ArcParserRULE_blockingRead)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(490)
		p.Match(ArcParserIDENTIFIER)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(491)
		p.Match(ArcParserDECLARE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(492)
		p.Match(ArcParserRECV)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(493)
		p.Match(ArcParserIDENTIFIER)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// INonBlockingReadContext is an interface to support dynamic dispatch.
type INonBlockingReadContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AllIDENTIFIER() []antlr.TerminalNode
	IDENTIFIER(i int) antlr.TerminalNode
	DECLARE() antlr.TerminalNode

	// IsNonBlockingReadContext differentiates from other interfaces.
	IsNonBlockingReadContext()
}

type NonBlockingReadContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyNonBlockingReadContext() *NonBlockingReadContext {
	var p = new(NonBlockingReadContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_nonBlockingRead
	return p
}

func InitEmptyNonBlockingReadContext(p *NonBlockingReadContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_nonBlockingRead
}

func (*NonBlockingReadContext) IsNonBlockingReadContext() {}

func NewNonBlockingReadContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *NonBlockingReadContext {
	var p = new(NonBlockingReadContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_nonBlockingRead

	return p
}

func (s *NonBlockingReadContext) GetParser() antlr.Parser { return s.parser }

func (s *NonBlockingReadContext) AllIDENTIFIER() []antlr.TerminalNode {
	return s.GetTokens(ArcParserIDENTIFIER)
}

func (s *NonBlockingReadContext) IDENTIFIER(i int) antlr.TerminalNode {
	return s.GetToken(ArcParserIDENTIFIER, i)
}

func (s *NonBlockingReadContext) DECLARE() antlr.TerminalNode {
	return s.GetToken(ArcParserDECLARE, 0)
}

func (s *NonBlockingReadContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *NonBlockingReadContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *NonBlockingReadContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterNonBlockingRead(s)
	}
}

func (s *NonBlockingReadContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitNonBlockingRead(s)
	}
}

func (p *ArcParser) NonBlockingRead() (localctx INonBlockingReadContext) {
	localctx = NewNonBlockingReadContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 88, ArcParserRULE_nonBlockingRead)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(495)
		p.Match(ArcParserIDENTIFIER)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(496)
		p.Match(ArcParserDECLARE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(497)
		p.Match(ArcParserIDENTIFIER)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IIfStatementContext is an interface to support dynamic dispatch.
type IIfStatementContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	IF() antlr.TerminalNode
	Expression() IExpressionContext
	Block() IBlockContext
	AllElseIfClause() []IElseIfClauseContext
	ElseIfClause(i int) IElseIfClauseContext
	ElseClause() IElseClauseContext

	// IsIfStatementContext differentiates from other interfaces.
	IsIfStatementContext()
}

type IfStatementContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyIfStatementContext() *IfStatementContext {
	var p = new(IfStatementContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_ifStatement
	return p
}

func InitEmptyIfStatementContext(p *IfStatementContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_ifStatement
}

func (*IfStatementContext) IsIfStatementContext() {}

func NewIfStatementContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *IfStatementContext {
	var p = new(IfStatementContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_ifStatement

	return p
}

func (s *IfStatementContext) GetParser() antlr.Parser { return s.parser }

func (s *IfStatementContext) IF() antlr.TerminalNode {
	return s.GetToken(ArcParserIF, 0)
}

func (s *IfStatementContext) Expression() IExpressionContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IExpressionContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IExpressionContext)
}

func (s *IfStatementContext) Block() IBlockContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IBlockContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IBlockContext)
}

func (s *IfStatementContext) AllElseIfClause() []IElseIfClauseContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IElseIfClauseContext); ok {
			len++
		}
	}

	tst := make([]IElseIfClauseContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IElseIfClauseContext); ok {
			tst[i] = t.(IElseIfClauseContext)
			i++
		}
	}

	return tst
}

func (s *IfStatementContext) ElseIfClause(i int) IElseIfClauseContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IElseIfClauseContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IElseIfClauseContext)
}

func (s *IfStatementContext) ElseClause() IElseClauseContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IElseClauseContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IElseClauseContext)
}

func (s *IfStatementContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *IfStatementContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *IfStatementContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterIfStatement(s)
	}
}

func (s *IfStatementContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitIfStatement(s)
	}
}

func (p *ArcParser) IfStatement() (localctx IIfStatementContext) {
	localctx = NewIfStatementContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 90, ArcParserRULE_ifStatement)
	var _la int

	var _alt int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(499)
		p.Match(ArcParserIF)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(500)
		p.Expression()
	}
	{
		p.SetState(501)
		p.Block()
	}
	p.SetState(505)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 44, p.GetParserRuleContext())
	if p.HasError() {
		goto errorExit
	}
	for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
		if _alt == 1 {
			{
				p.SetState(502)
				p.ElseIfClause()
			}

		}
		p.SetState(507)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 44, p.GetParserRuleContext())
		if p.HasError() {
			goto errorExit
		}
	}
	p.SetState(509)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ArcParserELSE {
		{
			p.SetState(508)
			p.ElseClause()
		}

	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IElseIfClauseContext is an interface to support dynamic dispatch.
type IElseIfClauseContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	ELSE() antlr.TerminalNode
	IF() antlr.TerminalNode
	Expression() IExpressionContext
	Block() IBlockContext

	// IsElseIfClauseContext differentiates from other interfaces.
	IsElseIfClauseContext()
}

type ElseIfClauseContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyElseIfClauseContext() *ElseIfClauseContext {
	var p = new(ElseIfClauseContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_elseIfClause
	return p
}

func InitEmptyElseIfClauseContext(p *ElseIfClauseContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_elseIfClause
}

func (*ElseIfClauseContext) IsElseIfClauseContext() {}

func NewElseIfClauseContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ElseIfClauseContext {
	var p = new(ElseIfClauseContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_elseIfClause

	return p
}

func (s *ElseIfClauseContext) GetParser() antlr.Parser { return s.parser }

func (s *ElseIfClauseContext) ELSE() antlr.TerminalNode {
	return s.GetToken(ArcParserELSE, 0)
}

func (s *ElseIfClauseContext) IF() antlr.TerminalNode {
	return s.GetToken(ArcParserIF, 0)
}

func (s *ElseIfClauseContext) Expression() IExpressionContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IExpressionContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IExpressionContext)
}

func (s *ElseIfClauseContext) Block() IBlockContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IBlockContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IBlockContext)
}

func (s *ElseIfClauseContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ElseIfClauseContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ElseIfClauseContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterElseIfClause(s)
	}
}

func (s *ElseIfClauseContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitElseIfClause(s)
	}
}

func (p *ArcParser) ElseIfClause() (localctx IElseIfClauseContext) {
	localctx = NewElseIfClauseContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 92, ArcParserRULE_elseIfClause)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(511)
		p.Match(ArcParserELSE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(512)
		p.Match(ArcParserIF)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(513)
		p.Expression()
	}
	{
		p.SetState(514)
		p.Block()
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IElseClauseContext is an interface to support dynamic dispatch.
type IElseClauseContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	ELSE() antlr.TerminalNode
	Block() IBlockContext

	// IsElseClauseContext differentiates from other interfaces.
	IsElseClauseContext()
}

type ElseClauseContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyElseClauseContext() *ElseClauseContext {
	var p = new(ElseClauseContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_elseClause
	return p
}

func InitEmptyElseClauseContext(p *ElseClauseContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_elseClause
}

func (*ElseClauseContext) IsElseClauseContext() {}

func NewElseClauseContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ElseClauseContext {
	var p = new(ElseClauseContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_elseClause

	return p
}

func (s *ElseClauseContext) GetParser() antlr.Parser { return s.parser }

func (s *ElseClauseContext) ELSE() antlr.TerminalNode {
	return s.GetToken(ArcParserELSE, 0)
}

func (s *ElseClauseContext) Block() IBlockContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IBlockContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IBlockContext)
}

func (s *ElseClauseContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ElseClauseContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ElseClauseContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterElseClause(s)
	}
}

func (s *ElseClauseContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitElseClause(s)
	}
}

func (p *ArcParser) ElseClause() (localctx IElseClauseContext) {
	localctx = NewElseClauseContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 94, ArcParserRULE_elseClause)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(516)
		p.Match(ArcParserELSE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(517)
		p.Block()
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IReturnStatementContext is an interface to support dynamic dispatch.
type IReturnStatementContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	RETURN() antlr.TerminalNode
	Expression() IExpressionContext

	// IsReturnStatementContext differentiates from other interfaces.
	IsReturnStatementContext()
}

type ReturnStatementContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyReturnStatementContext() *ReturnStatementContext {
	var p = new(ReturnStatementContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_returnStatement
	return p
}

func InitEmptyReturnStatementContext(p *ReturnStatementContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_returnStatement
}

func (*ReturnStatementContext) IsReturnStatementContext() {}

func NewReturnStatementContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ReturnStatementContext {
	var p = new(ReturnStatementContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_returnStatement

	return p
}

func (s *ReturnStatementContext) GetParser() antlr.Parser { return s.parser }

func (s *ReturnStatementContext) RETURN() antlr.TerminalNode {
	return s.GetToken(ArcParserRETURN, 0)
}

func (s *ReturnStatementContext) Expression() IExpressionContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IExpressionContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IExpressionContext)
}

func (s *ReturnStatementContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ReturnStatementContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ReturnStatementContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterReturnStatement(s)
	}
}

func (s *ReturnStatementContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitReturnStatement(s)
	}
}

func (p *ArcParser) ReturnStatement() (localctx IReturnStatementContext) {
	localctx = NewReturnStatementContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 96, ArcParserRULE_returnStatement)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(519)
		p.Match(ArcParserRETURN)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(521)
	p.GetErrorHandler().Sync(p)

	if p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 46, p.GetParserRuleContext()) == 1 {
		{
			p.SetState(520)
			p.Expression()
		}

	} else if p.HasError() { // JIM
		goto errorExit
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IFunctionCallContext is an interface to support dynamic dispatch.
type IFunctionCallContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	IDENTIFIER() antlr.TerminalNode
	LPAREN() antlr.TerminalNode
	RPAREN() antlr.TerminalNode
	ArgumentList() IArgumentListContext

	// IsFunctionCallContext differentiates from other interfaces.
	IsFunctionCallContext()
}

type FunctionCallContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyFunctionCallContext() *FunctionCallContext {
	var p = new(FunctionCallContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_functionCall
	return p
}

func InitEmptyFunctionCallContext(p *FunctionCallContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_functionCall
}

func (*FunctionCallContext) IsFunctionCallContext() {}

func NewFunctionCallContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *FunctionCallContext {
	var p = new(FunctionCallContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_functionCall

	return p
}

func (s *FunctionCallContext) GetParser() antlr.Parser { return s.parser }

func (s *FunctionCallContext) IDENTIFIER() antlr.TerminalNode {
	return s.GetToken(ArcParserIDENTIFIER, 0)
}

func (s *FunctionCallContext) LPAREN() antlr.TerminalNode {
	return s.GetToken(ArcParserLPAREN, 0)
}

func (s *FunctionCallContext) RPAREN() antlr.TerminalNode {
	return s.GetToken(ArcParserRPAREN, 0)
}

func (s *FunctionCallContext) ArgumentList() IArgumentListContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IArgumentListContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IArgumentListContext)
}

func (s *FunctionCallContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *FunctionCallContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *FunctionCallContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterFunctionCall(s)
	}
}

func (s *FunctionCallContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitFunctionCall(s)
	}
}

func (p *ArcParser) FunctionCall() (localctx IFunctionCallContext) {
	localctx = NewFunctionCallContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 98, ArcParserRULE_functionCall)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(523)
		p.Match(ArcParserIDENTIFIER)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(524)
		p.Match(ArcParserLPAREN)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(526)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&-278378716801796000) != 0 {
		{
			p.SetState(525)
			p.ArgumentList()
		}

	}
	{
		p.SetState(528)
		p.Match(ArcParserRPAREN)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// ITypeContext is an interface to support dynamic dispatch.
type ITypeContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	PrimitiveType() IPrimitiveTypeContext
	ChannelType() IChannelTypeContext
	SeriesType() ISeriesTypeContext

	// IsTypeContext differentiates from other interfaces.
	IsTypeContext()
}

type TypeContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyTypeContext() *TypeContext {
	var p = new(TypeContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_type
	return p
}

func InitEmptyTypeContext(p *TypeContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_type
}

func (*TypeContext) IsTypeContext() {}

func NewTypeContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *TypeContext {
	var p = new(TypeContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_type

	return p
}

func (s *TypeContext) GetParser() antlr.Parser { return s.parser }

func (s *TypeContext) PrimitiveType() IPrimitiveTypeContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IPrimitiveTypeContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IPrimitiveTypeContext)
}

func (s *TypeContext) ChannelType() IChannelTypeContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IChannelTypeContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IChannelTypeContext)
}

func (s *TypeContext) SeriesType() ISeriesTypeContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ISeriesTypeContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ISeriesTypeContext)
}

func (s *TypeContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *TypeContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *TypeContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterType(s)
	}
}

func (s *TypeContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitType(s)
	}
}

func (p *ArcParser) Type_() (localctx ITypeContext) {
	localctx = NewTypeContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 100, ArcParserRULE_type)
	p.SetState(533)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case ArcParserI8, ArcParserI16, ArcParserI32, ArcParserI64, ArcParserU8, ArcParserU16, ArcParserU32, ArcParserU64, ArcParserF32, ArcParserF64, ArcParserSTR, ArcParserTIMESTAMP, ArcParserTIMESPAN:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(530)
			p.PrimitiveType()
		}

	case ArcParserCHAN, ArcParserRECV_CHAN, ArcParserSEND_CHAN:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(531)
			p.ChannelType()
		}

	case ArcParserSERIES:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(532)
			p.SeriesType()
		}

	default:
		p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
		goto errorExit
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IPrimitiveTypeContext is an interface to support dynamic dispatch.
type IPrimitiveTypeContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	NumericType() INumericTypeContext
	STR() antlr.TerminalNode

	// IsPrimitiveTypeContext differentiates from other interfaces.
	IsPrimitiveTypeContext()
}

type PrimitiveTypeContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyPrimitiveTypeContext() *PrimitiveTypeContext {
	var p = new(PrimitiveTypeContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_primitiveType
	return p
}

func InitEmptyPrimitiveTypeContext(p *PrimitiveTypeContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_primitiveType
}

func (*PrimitiveTypeContext) IsPrimitiveTypeContext() {}

func NewPrimitiveTypeContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *PrimitiveTypeContext {
	var p = new(PrimitiveTypeContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_primitiveType

	return p
}

func (s *PrimitiveTypeContext) GetParser() antlr.Parser { return s.parser }

func (s *PrimitiveTypeContext) NumericType() INumericTypeContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(INumericTypeContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(INumericTypeContext)
}

func (s *PrimitiveTypeContext) STR() antlr.TerminalNode {
	return s.GetToken(ArcParserSTR, 0)
}

func (s *PrimitiveTypeContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *PrimitiveTypeContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *PrimitiveTypeContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterPrimitiveType(s)
	}
}

func (s *PrimitiveTypeContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitPrimitiveType(s)
	}
}

func (p *ArcParser) PrimitiveType() (localctx IPrimitiveTypeContext) {
	localctx = NewPrimitiveTypeContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 102, ArcParserRULE_primitiveType)
	p.SetState(537)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case ArcParserI8, ArcParserI16, ArcParserI32, ArcParserI64, ArcParserU8, ArcParserU16, ArcParserU32, ArcParserU64, ArcParserF32, ArcParserF64, ArcParserTIMESTAMP, ArcParserTIMESPAN:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(535)
			p.NumericType()
		}

	case ArcParserSTR:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(536)
			p.Match(ArcParserSTR)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	default:
		p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
		goto errorExit
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// INumericTypeContext is an interface to support dynamic dispatch.
type INumericTypeContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	IntegerType() IIntegerTypeContext
	FloatType() IFloatTypeContext
	TemporalType() ITemporalTypeContext

	// IsNumericTypeContext differentiates from other interfaces.
	IsNumericTypeContext()
}

type NumericTypeContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyNumericTypeContext() *NumericTypeContext {
	var p = new(NumericTypeContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_numericType
	return p
}

func InitEmptyNumericTypeContext(p *NumericTypeContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_numericType
}

func (*NumericTypeContext) IsNumericTypeContext() {}

func NewNumericTypeContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *NumericTypeContext {
	var p = new(NumericTypeContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_numericType

	return p
}

func (s *NumericTypeContext) GetParser() antlr.Parser { return s.parser }

func (s *NumericTypeContext) IntegerType() IIntegerTypeContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IIntegerTypeContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IIntegerTypeContext)
}

func (s *NumericTypeContext) FloatType() IFloatTypeContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IFloatTypeContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IFloatTypeContext)
}

func (s *NumericTypeContext) TemporalType() ITemporalTypeContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ITemporalTypeContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ITemporalTypeContext)
}

func (s *NumericTypeContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *NumericTypeContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *NumericTypeContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterNumericType(s)
	}
}

func (s *NumericTypeContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitNumericType(s)
	}
}

func (p *ArcParser) NumericType() (localctx INumericTypeContext) {
	localctx = NewNumericTypeContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 104, ArcParserRULE_numericType)
	p.SetState(542)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case ArcParserI8, ArcParserI16, ArcParserI32, ArcParserI64, ArcParserU8, ArcParserU16, ArcParserU32, ArcParserU64:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(539)
			p.IntegerType()
		}

	case ArcParserF32, ArcParserF64:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(540)
			p.FloatType()
		}

	case ArcParserTIMESTAMP, ArcParserTIMESPAN:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(541)
			p.TemporalType()
		}

	default:
		p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
		goto errorExit
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IIntegerTypeContext is an interface to support dynamic dispatch.
type IIntegerTypeContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	I8() antlr.TerminalNode
	I16() antlr.TerminalNode
	I32() antlr.TerminalNode
	I64() antlr.TerminalNode
	U8() antlr.TerminalNode
	U16() antlr.TerminalNode
	U32() antlr.TerminalNode
	U64() antlr.TerminalNode

	// IsIntegerTypeContext differentiates from other interfaces.
	IsIntegerTypeContext()
}

type IntegerTypeContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyIntegerTypeContext() *IntegerTypeContext {
	var p = new(IntegerTypeContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_integerType
	return p
}

func InitEmptyIntegerTypeContext(p *IntegerTypeContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_integerType
}

func (*IntegerTypeContext) IsIntegerTypeContext() {}

func NewIntegerTypeContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *IntegerTypeContext {
	var p = new(IntegerTypeContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_integerType

	return p
}

func (s *IntegerTypeContext) GetParser() antlr.Parser { return s.parser }

func (s *IntegerTypeContext) I8() antlr.TerminalNode {
	return s.GetToken(ArcParserI8, 0)
}

func (s *IntegerTypeContext) I16() antlr.TerminalNode {
	return s.GetToken(ArcParserI16, 0)
}

func (s *IntegerTypeContext) I32() antlr.TerminalNode {
	return s.GetToken(ArcParserI32, 0)
}

func (s *IntegerTypeContext) I64() antlr.TerminalNode {
	return s.GetToken(ArcParserI64, 0)
}

func (s *IntegerTypeContext) U8() antlr.TerminalNode {
	return s.GetToken(ArcParserU8, 0)
}

func (s *IntegerTypeContext) U16() antlr.TerminalNode {
	return s.GetToken(ArcParserU16, 0)
}

func (s *IntegerTypeContext) U32() antlr.TerminalNode {
	return s.GetToken(ArcParserU32, 0)
}

func (s *IntegerTypeContext) U64() antlr.TerminalNode {
	return s.GetToken(ArcParserU64, 0)
}

func (s *IntegerTypeContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *IntegerTypeContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *IntegerTypeContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterIntegerType(s)
	}
}

func (s *IntegerTypeContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitIntegerType(s)
	}
}

func (p *ArcParser) IntegerType() (localctx IIntegerTypeContext) {
	localctx = NewIntegerTypeContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 106, ArcParserRULE_integerType)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(544)
		_la = p.GetTokenStream().LA(1)

		if !((int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&4177920) != 0) {
			p.GetErrorHandler().RecoverInline(p)
		} else {
			p.GetErrorHandler().ReportMatch(p)
			p.Consume()
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IFloatTypeContext is an interface to support dynamic dispatch.
type IFloatTypeContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	F32() antlr.TerminalNode
	F64() antlr.TerminalNode

	// IsFloatTypeContext differentiates from other interfaces.
	IsFloatTypeContext()
}

type FloatTypeContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyFloatTypeContext() *FloatTypeContext {
	var p = new(FloatTypeContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_floatType
	return p
}

func InitEmptyFloatTypeContext(p *FloatTypeContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_floatType
}

func (*FloatTypeContext) IsFloatTypeContext() {}

func NewFloatTypeContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *FloatTypeContext {
	var p = new(FloatTypeContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_floatType

	return p
}

func (s *FloatTypeContext) GetParser() antlr.Parser { return s.parser }

func (s *FloatTypeContext) F32() antlr.TerminalNode {
	return s.GetToken(ArcParserF32, 0)
}

func (s *FloatTypeContext) F64() antlr.TerminalNode {
	return s.GetToken(ArcParserF64, 0)
}

func (s *FloatTypeContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *FloatTypeContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *FloatTypeContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterFloatType(s)
	}
}

func (s *FloatTypeContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitFloatType(s)
	}
}

func (p *ArcParser) FloatType() (localctx IFloatTypeContext) {
	localctx = NewFloatTypeContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 108, ArcParserRULE_floatType)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(546)
		_la = p.GetTokenStream().LA(1)

		if !(_la == ArcParserF32 || _la == ArcParserF64) {
			p.GetErrorHandler().RecoverInline(p)
		} else {
			p.GetErrorHandler().ReportMatch(p)
			p.Consume()
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// ITemporalTypeContext is an interface to support dynamic dispatch.
type ITemporalTypeContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	TIMESTAMP() antlr.TerminalNode
	TIMESPAN() antlr.TerminalNode

	// IsTemporalTypeContext differentiates from other interfaces.
	IsTemporalTypeContext()
}

type TemporalTypeContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyTemporalTypeContext() *TemporalTypeContext {
	var p = new(TemporalTypeContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_temporalType
	return p
}

func InitEmptyTemporalTypeContext(p *TemporalTypeContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_temporalType
}

func (*TemporalTypeContext) IsTemporalTypeContext() {}

func NewTemporalTypeContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *TemporalTypeContext {
	var p = new(TemporalTypeContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_temporalType

	return p
}

func (s *TemporalTypeContext) GetParser() antlr.Parser { return s.parser }

func (s *TemporalTypeContext) TIMESTAMP() antlr.TerminalNode {
	return s.GetToken(ArcParserTIMESTAMP, 0)
}

func (s *TemporalTypeContext) TIMESPAN() antlr.TerminalNode {
	return s.GetToken(ArcParserTIMESPAN, 0)
}

func (s *TemporalTypeContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *TemporalTypeContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *TemporalTypeContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterTemporalType(s)
	}
}

func (s *TemporalTypeContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitTemporalType(s)
	}
}

func (p *ArcParser) TemporalType() (localctx ITemporalTypeContext) {
	localctx = NewTemporalTypeContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 110, ArcParserRULE_temporalType)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(548)
		_la = p.GetTokenStream().LA(1)

		if !(_la == ArcParserTIMESTAMP || _la == ArcParserTIMESPAN) {
			p.GetErrorHandler().RecoverInline(p)
		} else {
			p.GetErrorHandler().ReportMatch(p)
			p.Consume()
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IChannelTypeContext is an interface to support dynamic dispatch.
type IChannelTypeContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	CHAN() antlr.TerminalNode
	RECV_CHAN() antlr.TerminalNode
	SEND_CHAN() antlr.TerminalNode
	PrimitiveType() IPrimitiveTypeContext
	SeriesType() ISeriesTypeContext

	// IsChannelTypeContext differentiates from other interfaces.
	IsChannelTypeContext()
}

type ChannelTypeContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyChannelTypeContext() *ChannelTypeContext {
	var p = new(ChannelTypeContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_channelType
	return p
}

func InitEmptyChannelTypeContext(p *ChannelTypeContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_channelType
}

func (*ChannelTypeContext) IsChannelTypeContext() {}

func NewChannelTypeContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ChannelTypeContext {
	var p = new(ChannelTypeContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_channelType

	return p
}

func (s *ChannelTypeContext) GetParser() antlr.Parser { return s.parser }

func (s *ChannelTypeContext) CHAN() antlr.TerminalNode {
	return s.GetToken(ArcParserCHAN, 0)
}

func (s *ChannelTypeContext) RECV_CHAN() antlr.TerminalNode {
	return s.GetToken(ArcParserRECV_CHAN, 0)
}

func (s *ChannelTypeContext) SEND_CHAN() antlr.TerminalNode {
	return s.GetToken(ArcParserSEND_CHAN, 0)
}

func (s *ChannelTypeContext) PrimitiveType() IPrimitiveTypeContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IPrimitiveTypeContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IPrimitiveTypeContext)
}

func (s *ChannelTypeContext) SeriesType() ISeriesTypeContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ISeriesTypeContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ISeriesTypeContext)
}

func (s *ChannelTypeContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ChannelTypeContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ChannelTypeContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterChannelType(s)
	}
}

func (s *ChannelTypeContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitChannelType(s)
	}
}

func (p *ArcParser) ChannelType() (localctx IChannelTypeContext) {
	localctx = NewChannelTypeContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 112, ArcParserRULE_channelType)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(550)
		_la = p.GetTokenStream().LA(1)

		if !((int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&14336) != 0) {
			p.GetErrorHandler().RecoverInline(p)
		} else {
			p.GetErrorHandler().ReportMatch(p)
			p.Consume()
		}
	}
	p.SetState(553)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case ArcParserI8, ArcParserI16, ArcParserI32, ArcParserI64, ArcParserU8, ArcParserU16, ArcParserU32, ArcParserU64, ArcParserF32, ArcParserF64, ArcParserSTR, ArcParserTIMESTAMP, ArcParserTIMESPAN:
		{
			p.SetState(551)
			p.PrimitiveType()
		}

	case ArcParserSERIES:
		{
			p.SetState(552)
			p.SeriesType()
		}

	default:
		p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
		goto errorExit
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// ISeriesTypeContext is an interface to support dynamic dispatch.
type ISeriesTypeContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	SERIES() antlr.TerminalNode
	PrimitiveType() IPrimitiveTypeContext

	// IsSeriesTypeContext differentiates from other interfaces.
	IsSeriesTypeContext()
}

type SeriesTypeContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptySeriesTypeContext() *SeriesTypeContext {
	var p = new(SeriesTypeContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_seriesType
	return p
}

func InitEmptySeriesTypeContext(p *SeriesTypeContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_seriesType
}

func (*SeriesTypeContext) IsSeriesTypeContext() {}

func NewSeriesTypeContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *SeriesTypeContext {
	var p = new(SeriesTypeContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_seriesType

	return p
}

func (s *SeriesTypeContext) GetParser() antlr.Parser { return s.parser }

func (s *SeriesTypeContext) SERIES() antlr.TerminalNode {
	return s.GetToken(ArcParserSERIES, 0)
}

func (s *SeriesTypeContext) PrimitiveType() IPrimitiveTypeContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IPrimitiveTypeContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IPrimitiveTypeContext)
}

func (s *SeriesTypeContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *SeriesTypeContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *SeriesTypeContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterSeriesType(s)
	}
}

func (s *SeriesTypeContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitSeriesType(s)
	}
}

func (p *ArcParser) SeriesType() (localctx ISeriesTypeContext) {
	localctx = NewSeriesTypeContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 114, ArcParserRULE_seriesType)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(555)
		p.Match(ArcParserSERIES)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(556)
		p.PrimitiveType()
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IExpressionContext is an interface to support dynamic dispatch.
type IExpressionContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	LogicalOrExpression() ILogicalOrExpressionContext

	// IsExpressionContext differentiates from other interfaces.
	IsExpressionContext()
}

type ExpressionContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyExpressionContext() *ExpressionContext {
	var p = new(ExpressionContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_expression
	return p
}

func InitEmptyExpressionContext(p *ExpressionContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_expression
}

func (*ExpressionContext) IsExpressionContext() {}

func NewExpressionContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ExpressionContext {
	var p = new(ExpressionContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_expression

	return p
}

func (s *ExpressionContext) GetParser() antlr.Parser { return s.parser }

func (s *ExpressionContext) LogicalOrExpression() ILogicalOrExpressionContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ILogicalOrExpressionContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ILogicalOrExpressionContext)
}

func (s *ExpressionContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ExpressionContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ExpressionContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterExpression(s)
	}
}

func (s *ExpressionContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitExpression(s)
	}
}

func (p *ArcParser) Expression() (localctx IExpressionContext) {
	localctx = NewExpressionContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 116, ArcParserRULE_expression)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(558)
		p.LogicalOrExpression()
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// ILogicalOrExpressionContext is an interface to support dynamic dispatch.
type ILogicalOrExpressionContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AllLogicalAndExpression() []ILogicalAndExpressionContext
	LogicalAndExpression(i int) ILogicalAndExpressionContext
	AllOR() []antlr.TerminalNode
	OR(i int) antlr.TerminalNode

	// IsLogicalOrExpressionContext differentiates from other interfaces.
	IsLogicalOrExpressionContext()
}

type LogicalOrExpressionContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyLogicalOrExpressionContext() *LogicalOrExpressionContext {
	var p = new(LogicalOrExpressionContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_logicalOrExpression
	return p
}

func InitEmptyLogicalOrExpressionContext(p *LogicalOrExpressionContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_logicalOrExpression
}

func (*LogicalOrExpressionContext) IsLogicalOrExpressionContext() {}

func NewLogicalOrExpressionContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *LogicalOrExpressionContext {
	var p = new(LogicalOrExpressionContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_logicalOrExpression

	return p
}

func (s *LogicalOrExpressionContext) GetParser() antlr.Parser { return s.parser }

func (s *LogicalOrExpressionContext) AllLogicalAndExpression() []ILogicalAndExpressionContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(ILogicalAndExpressionContext); ok {
			len++
		}
	}

	tst := make([]ILogicalAndExpressionContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(ILogicalAndExpressionContext); ok {
			tst[i] = t.(ILogicalAndExpressionContext)
			i++
		}
	}

	return tst
}

func (s *LogicalOrExpressionContext) LogicalAndExpression(i int) ILogicalAndExpressionContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ILogicalAndExpressionContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(ILogicalAndExpressionContext)
}

func (s *LogicalOrExpressionContext) AllOR() []antlr.TerminalNode {
	return s.GetTokens(ArcParserOR)
}

func (s *LogicalOrExpressionContext) OR(i int) antlr.TerminalNode {
	return s.GetToken(ArcParserOR, i)
}

func (s *LogicalOrExpressionContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *LogicalOrExpressionContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *LogicalOrExpressionContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterLogicalOrExpression(s)
	}
}

func (s *LogicalOrExpressionContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitLogicalOrExpression(s)
	}
}

func (p *ArcParser) LogicalOrExpression() (localctx ILogicalOrExpressionContext) {
	localctx = NewLogicalOrExpressionContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 118, ArcParserRULE_logicalOrExpression)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(560)
		p.LogicalAndExpression()
	}
	p.SetState(565)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == ArcParserOR {
		{
			p.SetState(561)
			p.Match(ArcParserOR)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(562)
			p.LogicalAndExpression()
		}

		p.SetState(567)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// ILogicalAndExpressionContext is an interface to support dynamic dispatch.
type ILogicalAndExpressionContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AllEqualityExpression() []IEqualityExpressionContext
	EqualityExpression(i int) IEqualityExpressionContext
	AllAND() []antlr.TerminalNode
	AND(i int) antlr.TerminalNode

	// IsLogicalAndExpressionContext differentiates from other interfaces.
	IsLogicalAndExpressionContext()
}

type LogicalAndExpressionContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyLogicalAndExpressionContext() *LogicalAndExpressionContext {
	var p = new(LogicalAndExpressionContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_logicalAndExpression
	return p
}

func InitEmptyLogicalAndExpressionContext(p *LogicalAndExpressionContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_logicalAndExpression
}

func (*LogicalAndExpressionContext) IsLogicalAndExpressionContext() {}

func NewLogicalAndExpressionContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *LogicalAndExpressionContext {
	var p = new(LogicalAndExpressionContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_logicalAndExpression

	return p
}

func (s *LogicalAndExpressionContext) GetParser() antlr.Parser { return s.parser }

func (s *LogicalAndExpressionContext) AllEqualityExpression() []IEqualityExpressionContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IEqualityExpressionContext); ok {
			len++
		}
	}

	tst := make([]IEqualityExpressionContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IEqualityExpressionContext); ok {
			tst[i] = t.(IEqualityExpressionContext)
			i++
		}
	}

	return tst
}

func (s *LogicalAndExpressionContext) EqualityExpression(i int) IEqualityExpressionContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IEqualityExpressionContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IEqualityExpressionContext)
}

func (s *LogicalAndExpressionContext) AllAND() []antlr.TerminalNode {
	return s.GetTokens(ArcParserAND)
}

func (s *LogicalAndExpressionContext) AND(i int) antlr.TerminalNode {
	return s.GetToken(ArcParserAND, i)
}

func (s *LogicalAndExpressionContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *LogicalAndExpressionContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *LogicalAndExpressionContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterLogicalAndExpression(s)
	}
}

func (s *LogicalAndExpressionContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitLogicalAndExpression(s)
	}
}

func (p *ArcParser) LogicalAndExpression() (localctx ILogicalAndExpressionContext) {
	localctx = NewLogicalAndExpressionContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 120, ArcParserRULE_logicalAndExpression)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(568)
		p.EqualityExpression()
	}
	p.SetState(573)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == ArcParserAND {
		{
			p.SetState(569)
			p.Match(ArcParserAND)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(570)
			p.EqualityExpression()
		}

		p.SetState(575)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IEqualityExpressionContext is an interface to support dynamic dispatch.
type IEqualityExpressionContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AllRelationalExpression() []IRelationalExpressionContext
	RelationalExpression(i int) IRelationalExpressionContext
	AllEQ() []antlr.TerminalNode
	EQ(i int) antlr.TerminalNode
	AllNEQ() []antlr.TerminalNode
	NEQ(i int) antlr.TerminalNode

	// IsEqualityExpressionContext differentiates from other interfaces.
	IsEqualityExpressionContext()
}

type EqualityExpressionContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyEqualityExpressionContext() *EqualityExpressionContext {
	var p = new(EqualityExpressionContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_equalityExpression
	return p
}

func InitEmptyEqualityExpressionContext(p *EqualityExpressionContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_equalityExpression
}

func (*EqualityExpressionContext) IsEqualityExpressionContext() {}

func NewEqualityExpressionContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *EqualityExpressionContext {
	var p = new(EqualityExpressionContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_equalityExpression

	return p
}

func (s *EqualityExpressionContext) GetParser() antlr.Parser { return s.parser }

func (s *EqualityExpressionContext) AllRelationalExpression() []IRelationalExpressionContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IRelationalExpressionContext); ok {
			len++
		}
	}

	tst := make([]IRelationalExpressionContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IRelationalExpressionContext); ok {
			tst[i] = t.(IRelationalExpressionContext)
			i++
		}
	}

	return tst
}

func (s *EqualityExpressionContext) RelationalExpression(i int) IRelationalExpressionContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IRelationalExpressionContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IRelationalExpressionContext)
}

func (s *EqualityExpressionContext) AllEQ() []antlr.TerminalNode {
	return s.GetTokens(ArcParserEQ)
}

func (s *EqualityExpressionContext) EQ(i int) antlr.TerminalNode {
	return s.GetToken(ArcParserEQ, i)
}

func (s *EqualityExpressionContext) AllNEQ() []antlr.TerminalNode {
	return s.GetTokens(ArcParserNEQ)
}

func (s *EqualityExpressionContext) NEQ(i int) antlr.TerminalNode {
	return s.GetToken(ArcParserNEQ, i)
}

func (s *EqualityExpressionContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *EqualityExpressionContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *EqualityExpressionContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterEqualityExpression(s)
	}
}

func (s *EqualityExpressionContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitEqualityExpression(s)
	}
}

func (p *ArcParser) EqualityExpression() (localctx IEqualityExpressionContext) {
	localctx = NewEqualityExpressionContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 122, ArcParserRULE_equalityExpression)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(576)
		p.RelationalExpression()
	}
	p.SetState(581)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == ArcParserEQ || _la == ArcParserNEQ {
		{
			p.SetState(577)
			_la = p.GetTokenStream().LA(1)

			if !(_la == ArcParserEQ || _la == ArcParserNEQ) {
				p.GetErrorHandler().RecoverInline(p)
			} else {
				p.GetErrorHandler().ReportMatch(p)
				p.Consume()
			}
		}
		{
			p.SetState(578)
			p.RelationalExpression()
		}

		p.SetState(583)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IRelationalExpressionContext is an interface to support dynamic dispatch.
type IRelationalExpressionContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AllAdditiveExpression() []IAdditiveExpressionContext
	AdditiveExpression(i int) IAdditiveExpressionContext
	AllLT() []antlr.TerminalNode
	LT(i int) antlr.TerminalNode
	AllGT() []antlr.TerminalNode
	GT(i int) antlr.TerminalNode
	AllLEQ() []antlr.TerminalNode
	LEQ(i int) antlr.TerminalNode
	AllGEQ() []antlr.TerminalNode
	GEQ(i int) antlr.TerminalNode

	// IsRelationalExpressionContext differentiates from other interfaces.
	IsRelationalExpressionContext()
}

type RelationalExpressionContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyRelationalExpressionContext() *RelationalExpressionContext {
	var p = new(RelationalExpressionContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_relationalExpression
	return p
}

func InitEmptyRelationalExpressionContext(p *RelationalExpressionContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_relationalExpression
}

func (*RelationalExpressionContext) IsRelationalExpressionContext() {}

func NewRelationalExpressionContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *RelationalExpressionContext {
	var p = new(RelationalExpressionContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_relationalExpression

	return p
}

func (s *RelationalExpressionContext) GetParser() antlr.Parser { return s.parser }

func (s *RelationalExpressionContext) AllAdditiveExpression() []IAdditiveExpressionContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IAdditiveExpressionContext); ok {
			len++
		}
	}

	tst := make([]IAdditiveExpressionContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IAdditiveExpressionContext); ok {
			tst[i] = t.(IAdditiveExpressionContext)
			i++
		}
	}

	return tst
}

func (s *RelationalExpressionContext) AdditiveExpression(i int) IAdditiveExpressionContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IAdditiveExpressionContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IAdditiveExpressionContext)
}

func (s *RelationalExpressionContext) AllLT() []antlr.TerminalNode {
	return s.GetTokens(ArcParserLT)
}

func (s *RelationalExpressionContext) LT(i int) antlr.TerminalNode {
	return s.GetToken(ArcParserLT, i)
}

func (s *RelationalExpressionContext) AllGT() []antlr.TerminalNode {
	return s.GetTokens(ArcParserGT)
}

func (s *RelationalExpressionContext) GT(i int) antlr.TerminalNode {
	return s.GetToken(ArcParserGT, i)
}

func (s *RelationalExpressionContext) AllLEQ() []antlr.TerminalNode {
	return s.GetTokens(ArcParserLEQ)
}

func (s *RelationalExpressionContext) LEQ(i int) antlr.TerminalNode {
	return s.GetToken(ArcParserLEQ, i)
}

func (s *RelationalExpressionContext) AllGEQ() []antlr.TerminalNode {
	return s.GetTokens(ArcParserGEQ)
}

func (s *RelationalExpressionContext) GEQ(i int) antlr.TerminalNode {
	return s.GetToken(ArcParserGEQ, i)
}

func (s *RelationalExpressionContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *RelationalExpressionContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *RelationalExpressionContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterRelationalExpression(s)
	}
}

func (s *RelationalExpressionContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitRelationalExpression(s)
	}
}

func (p *ArcParser) RelationalExpression() (localctx IRelationalExpressionContext) {
	localctx = NewRelationalExpressionContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 124, ArcParserRULE_relationalExpression)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(584)
		p.AdditiveExpression()
	}
	p.SetState(589)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&65970697666560) != 0 {
		{
			p.SetState(585)
			_la = p.GetTokenStream().LA(1)

			if !((int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&65970697666560) != 0) {
				p.GetErrorHandler().RecoverInline(p)
			} else {
				p.GetErrorHandler().ReportMatch(p)
				p.Consume()
			}
		}
		{
			p.SetState(586)
			p.AdditiveExpression()
		}

		p.SetState(591)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IAdditiveExpressionContext is an interface to support dynamic dispatch.
type IAdditiveExpressionContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AllMultiplicativeExpression() []IMultiplicativeExpressionContext
	MultiplicativeExpression(i int) IMultiplicativeExpressionContext
	AllPLUS() []antlr.TerminalNode
	PLUS(i int) antlr.TerminalNode
	AllMINUS() []antlr.TerminalNode
	MINUS(i int) antlr.TerminalNode

	// IsAdditiveExpressionContext differentiates from other interfaces.
	IsAdditiveExpressionContext()
}

type AdditiveExpressionContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyAdditiveExpressionContext() *AdditiveExpressionContext {
	var p = new(AdditiveExpressionContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_additiveExpression
	return p
}

func InitEmptyAdditiveExpressionContext(p *AdditiveExpressionContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_additiveExpression
}

func (*AdditiveExpressionContext) IsAdditiveExpressionContext() {}

func NewAdditiveExpressionContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *AdditiveExpressionContext {
	var p = new(AdditiveExpressionContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_additiveExpression

	return p
}

func (s *AdditiveExpressionContext) GetParser() antlr.Parser { return s.parser }

func (s *AdditiveExpressionContext) AllMultiplicativeExpression() []IMultiplicativeExpressionContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IMultiplicativeExpressionContext); ok {
			len++
		}
	}

	tst := make([]IMultiplicativeExpressionContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IMultiplicativeExpressionContext); ok {
			tst[i] = t.(IMultiplicativeExpressionContext)
			i++
		}
	}

	return tst
}

func (s *AdditiveExpressionContext) MultiplicativeExpression(i int) IMultiplicativeExpressionContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IMultiplicativeExpressionContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IMultiplicativeExpressionContext)
}

func (s *AdditiveExpressionContext) AllPLUS() []antlr.TerminalNode {
	return s.GetTokens(ArcParserPLUS)
}

func (s *AdditiveExpressionContext) PLUS(i int) antlr.TerminalNode {
	return s.GetToken(ArcParserPLUS, i)
}

func (s *AdditiveExpressionContext) AllMINUS() []antlr.TerminalNode {
	return s.GetTokens(ArcParserMINUS)
}

func (s *AdditiveExpressionContext) MINUS(i int) antlr.TerminalNode {
	return s.GetToken(ArcParserMINUS, i)
}

func (s *AdditiveExpressionContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *AdditiveExpressionContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *AdditiveExpressionContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterAdditiveExpression(s)
	}
}

func (s *AdditiveExpressionContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitAdditiveExpression(s)
	}
}

func (p *ArcParser) AdditiveExpression() (localctx IAdditiveExpressionContext) {
	localctx = NewAdditiveExpressionContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 126, ArcParserRULE_additiveExpression)
	var _la int

	var _alt int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(592)
		p.MultiplicativeExpression()
	}
	p.SetState(597)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 56, p.GetParserRuleContext())
	if p.HasError() {
		goto errorExit
	}
	for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
		if _alt == 1 {
			{
				p.SetState(593)
				_la = p.GetTokenStream().LA(1)

				if !(_la == ArcParserPLUS || _la == ArcParserMINUS) {
					p.GetErrorHandler().RecoverInline(p)
				} else {
					p.GetErrorHandler().ReportMatch(p)
					p.Consume()
				}
			}
			{
				p.SetState(594)
				p.MultiplicativeExpression()
			}

		}
		p.SetState(599)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 56, p.GetParserRuleContext())
		if p.HasError() {
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IMultiplicativeExpressionContext is an interface to support dynamic dispatch.
type IMultiplicativeExpressionContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AllPowerExpression() []IPowerExpressionContext
	PowerExpression(i int) IPowerExpressionContext
	AllSTAR() []antlr.TerminalNode
	STAR(i int) antlr.TerminalNode
	AllSLASH() []antlr.TerminalNode
	SLASH(i int) antlr.TerminalNode
	AllPERCENT() []antlr.TerminalNode
	PERCENT(i int) antlr.TerminalNode

	// IsMultiplicativeExpressionContext differentiates from other interfaces.
	IsMultiplicativeExpressionContext()
}

type MultiplicativeExpressionContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyMultiplicativeExpressionContext() *MultiplicativeExpressionContext {
	var p = new(MultiplicativeExpressionContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_multiplicativeExpression
	return p
}

func InitEmptyMultiplicativeExpressionContext(p *MultiplicativeExpressionContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_multiplicativeExpression
}

func (*MultiplicativeExpressionContext) IsMultiplicativeExpressionContext() {}

func NewMultiplicativeExpressionContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *MultiplicativeExpressionContext {
	var p = new(MultiplicativeExpressionContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_multiplicativeExpression

	return p
}

func (s *MultiplicativeExpressionContext) GetParser() antlr.Parser { return s.parser }

func (s *MultiplicativeExpressionContext) AllPowerExpression() []IPowerExpressionContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IPowerExpressionContext); ok {
			len++
		}
	}

	tst := make([]IPowerExpressionContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IPowerExpressionContext); ok {
			tst[i] = t.(IPowerExpressionContext)
			i++
		}
	}

	return tst
}

func (s *MultiplicativeExpressionContext) PowerExpression(i int) IPowerExpressionContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IPowerExpressionContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IPowerExpressionContext)
}

func (s *MultiplicativeExpressionContext) AllSTAR() []antlr.TerminalNode {
	return s.GetTokens(ArcParserSTAR)
}

func (s *MultiplicativeExpressionContext) STAR(i int) antlr.TerminalNode {
	return s.GetToken(ArcParserSTAR, i)
}

func (s *MultiplicativeExpressionContext) AllSLASH() []antlr.TerminalNode {
	return s.GetTokens(ArcParserSLASH)
}

func (s *MultiplicativeExpressionContext) SLASH(i int) antlr.TerminalNode {
	return s.GetToken(ArcParserSLASH, i)
}

func (s *MultiplicativeExpressionContext) AllPERCENT() []antlr.TerminalNode {
	return s.GetTokens(ArcParserPERCENT)
}

func (s *MultiplicativeExpressionContext) PERCENT(i int) antlr.TerminalNode {
	return s.GetToken(ArcParserPERCENT, i)
}

func (s *MultiplicativeExpressionContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *MultiplicativeExpressionContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *MultiplicativeExpressionContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterMultiplicativeExpression(s)
	}
}

func (s *MultiplicativeExpressionContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitMultiplicativeExpression(s)
	}
}

func (p *ArcParser) MultiplicativeExpression() (localctx IMultiplicativeExpressionContext) {
	localctx = NewMultiplicativeExpressionContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 128, ArcParserRULE_multiplicativeExpression)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(600)
		p.PowerExpression()
	}
	p.SetState(605)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&481036337152) != 0 {
		{
			p.SetState(601)
			_la = p.GetTokenStream().LA(1)

			if !((int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&481036337152) != 0) {
				p.GetErrorHandler().RecoverInline(p)
			} else {
				p.GetErrorHandler().ReportMatch(p)
				p.Consume()
			}
		}
		{
			p.SetState(602)
			p.PowerExpression()
		}

		p.SetState(607)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IPowerExpressionContext is an interface to support dynamic dispatch.
type IPowerExpressionContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	UnaryExpression() IUnaryExpressionContext
	CARET() antlr.TerminalNode
	PowerExpression() IPowerExpressionContext

	// IsPowerExpressionContext differentiates from other interfaces.
	IsPowerExpressionContext()
}

type PowerExpressionContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyPowerExpressionContext() *PowerExpressionContext {
	var p = new(PowerExpressionContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_powerExpression
	return p
}

func InitEmptyPowerExpressionContext(p *PowerExpressionContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_powerExpression
}

func (*PowerExpressionContext) IsPowerExpressionContext() {}

func NewPowerExpressionContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *PowerExpressionContext {
	var p = new(PowerExpressionContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_powerExpression

	return p
}

func (s *PowerExpressionContext) GetParser() antlr.Parser { return s.parser }

func (s *PowerExpressionContext) UnaryExpression() IUnaryExpressionContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IUnaryExpressionContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IUnaryExpressionContext)
}

func (s *PowerExpressionContext) CARET() antlr.TerminalNode {
	return s.GetToken(ArcParserCARET, 0)
}

func (s *PowerExpressionContext) PowerExpression() IPowerExpressionContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IPowerExpressionContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IPowerExpressionContext)
}

func (s *PowerExpressionContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *PowerExpressionContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *PowerExpressionContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterPowerExpression(s)
	}
}

func (s *PowerExpressionContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitPowerExpression(s)
	}
}

func (p *ArcParser) PowerExpression() (localctx IPowerExpressionContext) {
	localctx = NewPowerExpressionContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 130, ArcParserRULE_powerExpression)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(608)
		p.UnaryExpression()
	}
	p.SetState(611)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ArcParserCARET {
		{
			p.SetState(609)
			p.Match(ArcParserCARET)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(610)
			p.PowerExpression()
		}

	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IUnaryExpressionContext is an interface to support dynamic dispatch.
type IUnaryExpressionContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	MINUS() antlr.TerminalNode
	UnaryExpression() IUnaryExpressionContext
	NOT() antlr.TerminalNode
	BlockingReadExpr() IBlockingReadExprContext
	PostfixExpression() IPostfixExpressionContext

	// IsUnaryExpressionContext differentiates from other interfaces.
	IsUnaryExpressionContext()
}

type UnaryExpressionContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyUnaryExpressionContext() *UnaryExpressionContext {
	var p = new(UnaryExpressionContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_unaryExpression
	return p
}

func InitEmptyUnaryExpressionContext(p *UnaryExpressionContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_unaryExpression
}

func (*UnaryExpressionContext) IsUnaryExpressionContext() {}

func NewUnaryExpressionContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *UnaryExpressionContext {
	var p = new(UnaryExpressionContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_unaryExpression

	return p
}

func (s *UnaryExpressionContext) GetParser() antlr.Parser { return s.parser }

func (s *UnaryExpressionContext) MINUS() antlr.TerminalNode {
	return s.GetToken(ArcParserMINUS, 0)
}

func (s *UnaryExpressionContext) UnaryExpression() IUnaryExpressionContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IUnaryExpressionContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IUnaryExpressionContext)
}

func (s *UnaryExpressionContext) NOT() antlr.TerminalNode {
	return s.GetToken(ArcParserNOT, 0)
}

func (s *UnaryExpressionContext) BlockingReadExpr() IBlockingReadExprContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IBlockingReadExprContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IBlockingReadExprContext)
}

func (s *UnaryExpressionContext) PostfixExpression() IPostfixExpressionContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IPostfixExpressionContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IPostfixExpressionContext)
}

func (s *UnaryExpressionContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *UnaryExpressionContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *UnaryExpressionContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterUnaryExpression(s)
	}
}

func (s *UnaryExpressionContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitUnaryExpression(s)
	}
}

func (p *ArcParser) UnaryExpression() (localctx IUnaryExpressionContext) {
	localctx = NewUnaryExpressionContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 132, ArcParserRULE_unaryExpression)
	p.SetState(619)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case ArcParserMINUS:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(613)
			p.Match(ArcParserMINUS)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(614)
			p.UnaryExpression()
		}

	case ArcParserNOT:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(615)
			p.Match(ArcParserNOT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(616)
			p.UnaryExpression()
		}

	case ArcParserRECV:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(617)
			p.BlockingReadExpr()
		}

	case ArcParserNOW, ArcParserLEN, ArcParserCHAN, ArcParserRECV_CHAN, ArcParserSEND_CHAN, ArcParserI8, ArcParserI16, ArcParserI32, ArcParserI64, ArcParserU8, ArcParserU16, ArcParserU32, ArcParserU64, ArcParserF32, ArcParserF64, ArcParserSTR, ArcParserTIMESTAMP, ArcParserTIMESPAN, ArcParserSERIES, ArcParserLPAREN, ArcParserLBRACKET, ArcParserTEMPORAL_LITERAL, ArcParserFREQUENCY_LITERAL, ArcParserINTEGER_LITERAL, ArcParserFLOAT_LITERAL, ArcParserSTR_LITERAL, ArcParserIDENTIFIER:
		p.EnterOuterAlt(localctx, 4)
		{
			p.SetState(618)
			p.PostfixExpression()
		}

	default:
		p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
		goto errorExit
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IBlockingReadExprContext is an interface to support dynamic dispatch.
type IBlockingReadExprContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	RECV() antlr.TerminalNode
	IDENTIFIER() antlr.TerminalNode

	// IsBlockingReadExprContext differentiates from other interfaces.
	IsBlockingReadExprContext()
}

type BlockingReadExprContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyBlockingReadExprContext() *BlockingReadExprContext {
	var p = new(BlockingReadExprContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_blockingReadExpr
	return p
}

func InitEmptyBlockingReadExprContext(p *BlockingReadExprContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_blockingReadExpr
}

func (*BlockingReadExprContext) IsBlockingReadExprContext() {}

func NewBlockingReadExprContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *BlockingReadExprContext {
	var p = new(BlockingReadExprContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_blockingReadExpr

	return p
}

func (s *BlockingReadExprContext) GetParser() antlr.Parser { return s.parser }

func (s *BlockingReadExprContext) RECV() antlr.TerminalNode {
	return s.GetToken(ArcParserRECV, 0)
}

func (s *BlockingReadExprContext) IDENTIFIER() antlr.TerminalNode {
	return s.GetToken(ArcParserIDENTIFIER, 0)
}

func (s *BlockingReadExprContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *BlockingReadExprContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *BlockingReadExprContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterBlockingReadExpr(s)
	}
}

func (s *BlockingReadExprContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitBlockingReadExpr(s)
	}
}

func (p *ArcParser) BlockingReadExpr() (localctx IBlockingReadExprContext) {
	localctx = NewBlockingReadExprContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 134, ArcParserRULE_blockingReadExpr)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(621)
		p.Match(ArcParserRECV)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(622)
		p.Match(ArcParserIDENTIFIER)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IPostfixExpressionContext is an interface to support dynamic dispatch.
type IPostfixExpressionContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	PrimaryExpression() IPrimaryExpressionContext
	AllIndexOrSlice() []IIndexOrSliceContext
	IndexOrSlice(i int) IIndexOrSliceContext
	AllFunctionCallSuffix() []IFunctionCallSuffixContext
	FunctionCallSuffix(i int) IFunctionCallSuffixContext

	// IsPostfixExpressionContext differentiates from other interfaces.
	IsPostfixExpressionContext()
}

type PostfixExpressionContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyPostfixExpressionContext() *PostfixExpressionContext {
	var p = new(PostfixExpressionContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_postfixExpression
	return p
}

func InitEmptyPostfixExpressionContext(p *PostfixExpressionContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_postfixExpression
}

func (*PostfixExpressionContext) IsPostfixExpressionContext() {}

func NewPostfixExpressionContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *PostfixExpressionContext {
	var p = new(PostfixExpressionContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_postfixExpression

	return p
}

func (s *PostfixExpressionContext) GetParser() antlr.Parser { return s.parser }

func (s *PostfixExpressionContext) PrimaryExpression() IPrimaryExpressionContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IPrimaryExpressionContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IPrimaryExpressionContext)
}

func (s *PostfixExpressionContext) AllIndexOrSlice() []IIndexOrSliceContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IIndexOrSliceContext); ok {
			len++
		}
	}

	tst := make([]IIndexOrSliceContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IIndexOrSliceContext); ok {
			tst[i] = t.(IIndexOrSliceContext)
			i++
		}
	}

	return tst
}

func (s *PostfixExpressionContext) IndexOrSlice(i int) IIndexOrSliceContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IIndexOrSliceContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IIndexOrSliceContext)
}

func (s *PostfixExpressionContext) AllFunctionCallSuffix() []IFunctionCallSuffixContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IFunctionCallSuffixContext); ok {
			len++
		}
	}

	tst := make([]IFunctionCallSuffixContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IFunctionCallSuffixContext); ok {
			tst[i] = t.(IFunctionCallSuffixContext)
			i++
		}
	}

	return tst
}

func (s *PostfixExpressionContext) FunctionCallSuffix(i int) IFunctionCallSuffixContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IFunctionCallSuffixContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IFunctionCallSuffixContext)
}

func (s *PostfixExpressionContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *PostfixExpressionContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *PostfixExpressionContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterPostfixExpression(s)
	}
}

func (s *PostfixExpressionContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitPostfixExpression(s)
	}
}

func (p *ArcParser) PostfixExpression() (localctx IPostfixExpressionContext) {
	localctx = NewPostfixExpressionContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 136, ArcParserRULE_postfixExpression)
	var _alt int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(624)
		p.PrimaryExpression()
	}
	p.SetState(629)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 61, p.GetParserRuleContext())
	if p.HasError() {
		goto errorExit
	}
	for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
		if _alt == 1 {
			p.SetState(627)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}

			switch p.GetTokenStream().LA(1) {
			case ArcParserLBRACKET:
				{
					p.SetState(625)
					p.IndexOrSlice()
				}

			case ArcParserLPAREN:
				{
					p.SetState(626)
					p.FunctionCallSuffix()
				}

			default:
				p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
				goto errorExit
			}

		}
		p.SetState(631)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 61, p.GetParserRuleContext())
		if p.HasError() {
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IIndexOrSliceContext is an interface to support dynamic dispatch.
type IIndexOrSliceContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	LBRACKET() antlr.TerminalNode
	AllExpression() []IExpressionContext
	Expression(i int) IExpressionContext
	RBRACKET() antlr.TerminalNode
	COLON() antlr.TerminalNode

	// IsIndexOrSliceContext differentiates from other interfaces.
	IsIndexOrSliceContext()
}

type IndexOrSliceContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyIndexOrSliceContext() *IndexOrSliceContext {
	var p = new(IndexOrSliceContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_indexOrSlice
	return p
}

func InitEmptyIndexOrSliceContext(p *IndexOrSliceContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_indexOrSlice
}

func (*IndexOrSliceContext) IsIndexOrSliceContext() {}

func NewIndexOrSliceContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *IndexOrSliceContext {
	var p = new(IndexOrSliceContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_indexOrSlice

	return p
}

func (s *IndexOrSliceContext) GetParser() antlr.Parser { return s.parser }

func (s *IndexOrSliceContext) LBRACKET() antlr.TerminalNode {
	return s.GetToken(ArcParserLBRACKET, 0)
}

func (s *IndexOrSliceContext) AllExpression() []IExpressionContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IExpressionContext); ok {
			len++
		}
	}

	tst := make([]IExpressionContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IExpressionContext); ok {
			tst[i] = t.(IExpressionContext)
			i++
		}
	}

	return tst
}

func (s *IndexOrSliceContext) Expression(i int) IExpressionContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IExpressionContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IExpressionContext)
}

func (s *IndexOrSliceContext) RBRACKET() antlr.TerminalNode {
	return s.GetToken(ArcParserRBRACKET, 0)
}

func (s *IndexOrSliceContext) COLON() antlr.TerminalNode {
	return s.GetToken(ArcParserCOLON, 0)
}

func (s *IndexOrSliceContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *IndexOrSliceContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *IndexOrSliceContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterIndexOrSlice(s)
	}
}

func (s *IndexOrSliceContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitIndexOrSlice(s)
	}
}

func (p *ArcParser) IndexOrSlice() (localctx IIndexOrSliceContext) {
	localctx = NewIndexOrSliceContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 138, ArcParserRULE_indexOrSlice)
	var _la int

	p.SetState(645)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 64, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(632)
			p.Match(ArcParserLBRACKET)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(633)
			p.Expression()
		}
		{
			p.SetState(634)
			p.Match(ArcParserRBRACKET)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(636)
			p.Match(ArcParserLBRACKET)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		p.SetState(638)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		if (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&-278378716801796000) != 0 {
			{
				p.SetState(637)
				p.Expression()
			}

		}
		{
			p.SetState(640)
			p.Match(ArcParserCOLON)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		p.SetState(642)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		if (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&-278378716801796000) != 0 {
			{
				p.SetState(641)
				p.Expression()
			}

		}
		{
			p.SetState(644)
			p.Match(ArcParserRBRACKET)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case antlr.ATNInvalidAltNumber:
		goto errorExit
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IFunctionCallSuffixContext is an interface to support dynamic dispatch.
type IFunctionCallSuffixContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	LPAREN() antlr.TerminalNode
	RPAREN() antlr.TerminalNode
	ArgumentList() IArgumentListContext

	// IsFunctionCallSuffixContext differentiates from other interfaces.
	IsFunctionCallSuffixContext()
}

type FunctionCallSuffixContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyFunctionCallSuffixContext() *FunctionCallSuffixContext {
	var p = new(FunctionCallSuffixContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_functionCallSuffix
	return p
}

func InitEmptyFunctionCallSuffixContext(p *FunctionCallSuffixContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_functionCallSuffix
}

func (*FunctionCallSuffixContext) IsFunctionCallSuffixContext() {}

func NewFunctionCallSuffixContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *FunctionCallSuffixContext {
	var p = new(FunctionCallSuffixContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_functionCallSuffix

	return p
}

func (s *FunctionCallSuffixContext) GetParser() antlr.Parser { return s.parser }

func (s *FunctionCallSuffixContext) LPAREN() antlr.TerminalNode {
	return s.GetToken(ArcParserLPAREN, 0)
}

func (s *FunctionCallSuffixContext) RPAREN() antlr.TerminalNode {
	return s.GetToken(ArcParserRPAREN, 0)
}

func (s *FunctionCallSuffixContext) ArgumentList() IArgumentListContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IArgumentListContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IArgumentListContext)
}

func (s *FunctionCallSuffixContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *FunctionCallSuffixContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *FunctionCallSuffixContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterFunctionCallSuffix(s)
	}
}

func (s *FunctionCallSuffixContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitFunctionCallSuffix(s)
	}
}

func (p *ArcParser) FunctionCallSuffix() (localctx IFunctionCallSuffixContext) {
	localctx = NewFunctionCallSuffixContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 140, ArcParserRULE_functionCallSuffix)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(647)
		p.Match(ArcParserLPAREN)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(649)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&-278378716801796000) != 0 {
		{
			p.SetState(648)
			p.ArgumentList()
		}

	}
	{
		p.SetState(651)
		p.Match(ArcParserRPAREN)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IPrimaryExpressionContext is an interface to support dynamic dispatch.
type IPrimaryExpressionContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	Literal() ILiteralContext
	IDENTIFIER() antlr.TerminalNode
	LPAREN() antlr.TerminalNode
	Expression() IExpressionContext
	RPAREN() antlr.TerminalNode
	TypeCast() ITypeCastContext
	BuiltinFunction() IBuiltinFunctionContext

	// IsPrimaryExpressionContext differentiates from other interfaces.
	IsPrimaryExpressionContext()
}

type PrimaryExpressionContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyPrimaryExpressionContext() *PrimaryExpressionContext {
	var p = new(PrimaryExpressionContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_primaryExpression
	return p
}

func InitEmptyPrimaryExpressionContext(p *PrimaryExpressionContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_primaryExpression
}

func (*PrimaryExpressionContext) IsPrimaryExpressionContext() {}

func NewPrimaryExpressionContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *PrimaryExpressionContext {
	var p = new(PrimaryExpressionContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_primaryExpression

	return p
}

func (s *PrimaryExpressionContext) GetParser() antlr.Parser { return s.parser }

func (s *PrimaryExpressionContext) Literal() ILiteralContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ILiteralContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ILiteralContext)
}

func (s *PrimaryExpressionContext) IDENTIFIER() antlr.TerminalNode {
	return s.GetToken(ArcParserIDENTIFIER, 0)
}

func (s *PrimaryExpressionContext) LPAREN() antlr.TerminalNode {
	return s.GetToken(ArcParserLPAREN, 0)
}

func (s *PrimaryExpressionContext) Expression() IExpressionContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IExpressionContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IExpressionContext)
}

func (s *PrimaryExpressionContext) RPAREN() antlr.TerminalNode {
	return s.GetToken(ArcParserRPAREN, 0)
}

func (s *PrimaryExpressionContext) TypeCast() ITypeCastContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ITypeCastContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ITypeCastContext)
}

func (s *PrimaryExpressionContext) BuiltinFunction() IBuiltinFunctionContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IBuiltinFunctionContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IBuiltinFunctionContext)
}

func (s *PrimaryExpressionContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *PrimaryExpressionContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *PrimaryExpressionContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterPrimaryExpression(s)
	}
}

func (s *PrimaryExpressionContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitPrimaryExpression(s)
	}
}

func (p *ArcParser) PrimaryExpression() (localctx IPrimaryExpressionContext) {
	localctx = NewPrimaryExpressionContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 142, ArcParserRULE_primaryExpression)
	p.SetState(661)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case ArcParserLBRACKET, ArcParserTEMPORAL_LITERAL, ArcParserFREQUENCY_LITERAL, ArcParserINTEGER_LITERAL, ArcParserFLOAT_LITERAL, ArcParserSTR_LITERAL:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(653)
			p.Literal()
		}

	case ArcParserIDENTIFIER:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(654)
			p.Match(ArcParserIDENTIFIER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case ArcParserLPAREN:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(655)
			p.Match(ArcParserLPAREN)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(656)
			p.Expression()
		}
		{
			p.SetState(657)
			p.Match(ArcParserRPAREN)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case ArcParserCHAN, ArcParserRECV_CHAN, ArcParserSEND_CHAN, ArcParserI8, ArcParserI16, ArcParserI32, ArcParserI64, ArcParserU8, ArcParserU16, ArcParserU32, ArcParserU64, ArcParserF32, ArcParserF64, ArcParserSTR, ArcParserTIMESTAMP, ArcParserTIMESPAN, ArcParserSERIES:
		p.EnterOuterAlt(localctx, 4)
		{
			p.SetState(659)
			p.TypeCast()
		}

	case ArcParserNOW, ArcParserLEN:
		p.EnterOuterAlt(localctx, 5)
		{
			p.SetState(660)
			p.BuiltinFunction()
		}

	default:
		p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
		goto errorExit
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// ITypeCastContext is an interface to support dynamic dispatch.
type ITypeCastContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	Type_() ITypeContext
	LPAREN() antlr.TerminalNode
	Expression() IExpressionContext
	RPAREN() antlr.TerminalNode

	// IsTypeCastContext differentiates from other interfaces.
	IsTypeCastContext()
}

type TypeCastContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyTypeCastContext() *TypeCastContext {
	var p = new(TypeCastContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_typeCast
	return p
}

func InitEmptyTypeCastContext(p *TypeCastContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_typeCast
}

func (*TypeCastContext) IsTypeCastContext() {}

func NewTypeCastContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *TypeCastContext {
	var p = new(TypeCastContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_typeCast

	return p
}

func (s *TypeCastContext) GetParser() antlr.Parser { return s.parser }

func (s *TypeCastContext) Type_() ITypeContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ITypeContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ITypeContext)
}

func (s *TypeCastContext) LPAREN() antlr.TerminalNode {
	return s.GetToken(ArcParserLPAREN, 0)
}

func (s *TypeCastContext) Expression() IExpressionContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IExpressionContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IExpressionContext)
}

func (s *TypeCastContext) RPAREN() antlr.TerminalNode {
	return s.GetToken(ArcParserRPAREN, 0)
}

func (s *TypeCastContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *TypeCastContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *TypeCastContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterTypeCast(s)
	}
}

func (s *TypeCastContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitTypeCast(s)
	}
}

func (p *ArcParser) TypeCast() (localctx ITypeCastContext) {
	localctx = NewTypeCastContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 144, ArcParserRULE_typeCast)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(663)
		p.Type_()
	}
	{
		p.SetState(664)
		p.Match(ArcParserLPAREN)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(665)
		p.Expression()
	}
	{
		p.SetState(666)
		p.Match(ArcParserRPAREN)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IBuiltinFunctionContext is an interface to support dynamic dispatch.
type IBuiltinFunctionContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	LEN() antlr.TerminalNode
	LPAREN() antlr.TerminalNode
	Expression() IExpressionContext
	RPAREN() antlr.TerminalNode
	NOW() antlr.TerminalNode

	// IsBuiltinFunctionContext differentiates from other interfaces.
	IsBuiltinFunctionContext()
}

type BuiltinFunctionContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyBuiltinFunctionContext() *BuiltinFunctionContext {
	var p = new(BuiltinFunctionContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_builtinFunction
	return p
}

func InitEmptyBuiltinFunctionContext(p *BuiltinFunctionContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_builtinFunction
}

func (*BuiltinFunctionContext) IsBuiltinFunctionContext() {}

func NewBuiltinFunctionContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *BuiltinFunctionContext {
	var p = new(BuiltinFunctionContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_builtinFunction

	return p
}

func (s *BuiltinFunctionContext) GetParser() antlr.Parser { return s.parser }

func (s *BuiltinFunctionContext) LEN() antlr.TerminalNode {
	return s.GetToken(ArcParserLEN, 0)
}

func (s *BuiltinFunctionContext) LPAREN() antlr.TerminalNode {
	return s.GetToken(ArcParserLPAREN, 0)
}

func (s *BuiltinFunctionContext) Expression() IExpressionContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IExpressionContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IExpressionContext)
}

func (s *BuiltinFunctionContext) RPAREN() antlr.TerminalNode {
	return s.GetToken(ArcParserRPAREN, 0)
}

func (s *BuiltinFunctionContext) NOW() antlr.TerminalNode {
	return s.GetToken(ArcParserNOW, 0)
}

func (s *BuiltinFunctionContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *BuiltinFunctionContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *BuiltinFunctionContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterBuiltinFunction(s)
	}
}

func (s *BuiltinFunctionContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitBuiltinFunction(s)
	}
}

func (p *ArcParser) BuiltinFunction() (localctx IBuiltinFunctionContext) {
	localctx = NewBuiltinFunctionContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 146, ArcParserRULE_builtinFunction)
	p.SetState(676)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case ArcParserLEN:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(668)
			p.Match(ArcParserLEN)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(669)
			p.Match(ArcParserLPAREN)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(670)
			p.Expression()
		}
		{
			p.SetState(671)
			p.Match(ArcParserRPAREN)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case ArcParserNOW:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(673)
			p.Match(ArcParserNOW)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(674)
			p.Match(ArcParserLPAREN)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(675)
			p.Match(ArcParserRPAREN)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	default:
		p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
		goto errorExit
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// ILiteralContext is an interface to support dynamic dispatch.
type ILiteralContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	NumericLiteral() INumericLiteralContext
	TemporalLiteral() ITemporalLiteralContext
	STR_LITERAL() antlr.TerminalNode
	SeriesLiteral() ISeriesLiteralContext

	// IsLiteralContext differentiates from other interfaces.
	IsLiteralContext()
}

type LiteralContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyLiteralContext() *LiteralContext {
	var p = new(LiteralContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_literal
	return p
}

func InitEmptyLiteralContext(p *LiteralContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_literal
}

func (*LiteralContext) IsLiteralContext() {}

func NewLiteralContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *LiteralContext {
	var p = new(LiteralContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_literal

	return p
}

func (s *LiteralContext) GetParser() antlr.Parser { return s.parser }

func (s *LiteralContext) NumericLiteral() INumericLiteralContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(INumericLiteralContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(INumericLiteralContext)
}

func (s *LiteralContext) TemporalLiteral() ITemporalLiteralContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ITemporalLiteralContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ITemporalLiteralContext)
}

func (s *LiteralContext) STR_LITERAL() antlr.TerminalNode {
	return s.GetToken(ArcParserSTR_LITERAL, 0)
}

func (s *LiteralContext) SeriesLiteral() ISeriesLiteralContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ISeriesLiteralContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ISeriesLiteralContext)
}

func (s *LiteralContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *LiteralContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *LiteralContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterLiteral(s)
	}
}

func (s *LiteralContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitLiteral(s)
	}
}

func (p *ArcParser) Literal() (localctx ILiteralContext) {
	localctx = NewLiteralContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 148, ArcParserRULE_literal)
	p.SetState(682)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case ArcParserINTEGER_LITERAL, ArcParserFLOAT_LITERAL:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(678)
			p.NumericLiteral()
		}

	case ArcParserTEMPORAL_LITERAL, ArcParserFREQUENCY_LITERAL:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(679)
			p.TemporalLiteral()
		}

	case ArcParserSTR_LITERAL:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(680)
			p.Match(ArcParserSTR_LITERAL)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case ArcParserLBRACKET:
		p.EnterOuterAlt(localctx, 4)
		{
			p.SetState(681)
			p.SeriesLiteral()
		}

	default:
		p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
		goto errorExit
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// INumericLiteralContext is an interface to support dynamic dispatch.
type INumericLiteralContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	INTEGER_LITERAL() antlr.TerminalNode
	FLOAT_LITERAL() antlr.TerminalNode

	// IsNumericLiteralContext differentiates from other interfaces.
	IsNumericLiteralContext()
}

type NumericLiteralContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyNumericLiteralContext() *NumericLiteralContext {
	var p = new(NumericLiteralContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_numericLiteral
	return p
}

func InitEmptyNumericLiteralContext(p *NumericLiteralContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_numericLiteral
}

func (*NumericLiteralContext) IsNumericLiteralContext() {}

func NewNumericLiteralContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *NumericLiteralContext {
	var p = new(NumericLiteralContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_numericLiteral

	return p
}

func (s *NumericLiteralContext) GetParser() antlr.Parser { return s.parser }

func (s *NumericLiteralContext) INTEGER_LITERAL() antlr.TerminalNode {
	return s.GetToken(ArcParserINTEGER_LITERAL, 0)
}

func (s *NumericLiteralContext) FLOAT_LITERAL() antlr.TerminalNode {
	return s.GetToken(ArcParserFLOAT_LITERAL, 0)
}

func (s *NumericLiteralContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *NumericLiteralContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *NumericLiteralContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterNumericLiteral(s)
	}
}

func (s *NumericLiteralContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitNumericLiteral(s)
	}
}

func (p *ArcParser) NumericLiteral() (localctx INumericLiteralContext) {
	localctx = NewNumericLiteralContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 150, ArcParserRULE_numericLiteral)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(684)
		_la = p.GetTokenStream().LA(1)

		if !(_la == ArcParserINTEGER_LITERAL || _la == ArcParserFLOAT_LITERAL) {
			p.GetErrorHandler().RecoverInline(p)
		} else {
			p.GetErrorHandler().ReportMatch(p)
			p.Consume()
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// ITemporalLiteralContext is an interface to support dynamic dispatch.
type ITemporalLiteralContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	TEMPORAL_LITERAL() antlr.TerminalNode
	FREQUENCY_LITERAL() antlr.TerminalNode

	// IsTemporalLiteralContext differentiates from other interfaces.
	IsTemporalLiteralContext()
}

type TemporalLiteralContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyTemporalLiteralContext() *TemporalLiteralContext {
	var p = new(TemporalLiteralContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_temporalLiteral
	return p
}

func InitEmptyTemporalLiteralContext(p *TemporalLiteralContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_temporalLiteral
}

func (*TemporalLiteralContext) IsTemporalLiteralContext() {}

func NewTemporalLiteralContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *TemporalLiteralContext {
	var p = new(TemporalLiteralContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_temporalLiteral

	return p
}

func (s *TemporalLiteralContext) GetParser() antlr.Parser { return s.parser }

func (s *TemporalLiteralContext) TEMPORAL_LITERAL() antlr.TerminalNode {
	return s.GetToken(ArcParserTEMPORAL_LITERAL, 0)
}

func (s *TemporalLiteralContext) FREQUENCY_LITERAL() antlr.TerminalNode {
	return s.GetToken(ArcParserFREQUENCY_LITERAL, 0)
}

func (s *TemporalLiteralContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *TemporalLiteralContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *TemporalLiteralContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterTemporalLiteral(s)
	}
}

func (s *TemporalLiteralContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitTemporalLiteral(s)
	}
}

func (p *ArcParser) TemporalLiteral() (localctx ITemporalLiteralContext) {
	localctx = NewTemporalLiteralContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 152, ArcParserRULE_temporalLiteral)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(686)
		_la = p.GetTokenStream().LA(1)

		if !(_la == ArcParserTEMPORAL_LITERAL || _la == ArcParserFREQUENCY_LITERAL) {
			p.GetErrorHandler().RecoverInline(p)
		} else {
			p.GetErrorHandler().ReportMatch(p)
			p.Consume()
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// ISeriesLiteralContext is an interface to support dynamic dispatch.
type ISeriesLiteralContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	LBRACKET() antlr.TerminalNode
	RBRACKET() antlr.TerminalNode
	ExpressionList() IExpressionListContext

	// IsSeriesLiteralContext differentiates from other interfaces.
	IsSeriesLiteralContext()
}

type SeriesLiteralContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptySeriesLiteralContext() *SeriesLiteralContext {
	var p = new(SeriesLiteralContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_seriesLiteral
	return p
}

func InitEmptySeriesLiteralContext(p *SeriesLiteralContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_seriesLiteral
}

func (*SeriesLiteralContext) IsSeriesLiteralContext() {}

func NewSeriesLiteralContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *SeriesLiteralContext {
	var p = new(SeriesLiteralContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_seriesLiteral

	return p
}

func (s *SeriesLiteralContext) GetParser() antlr.Parser { return s.parser }

func (s *SeriesLiteralContext) LBRACKET() antlr.TerminalNode {
	return s.GetToken(ArcParserLBRACKET, 0)
}

func (s *SeriesLiteralContext) RBRACKET() antlr.TerminalNode {
	return s.GetToken(ArcParserRBRACKET, 0)
}

func (s *SeriesLiteralContext) ExpressionList() IExpressionListContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IExpressionListContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IExpressionListContext)
}

func (s *SeriesLiteralContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *SeriesLiteralContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *SeriesLiteralContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterSeriesLiteral(s)
	}
}

func (s *SeriesLiteralContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitSeriesLiteral(s)
	}
}

func (p *ArcParser) SeriesLiteral() (localctx ISeriesLiteralContext) {
	localctx = NewSeriesLiteralContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 154, ArcParserRULE_seriesLiteral)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(688)
		p.Match(ArcParserLBRACKET)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(690)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&-278378716801796000) != 0 {
		{
			p.SetState(689)
			p.ExpressionList()
		}

	}
	{
		p.SetState(692)
		p.Match(ArcParserRBRACKET)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}

// IExpressionListContext is an interface to support dynamic dispatch.
type IExpressionListContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AllExpression() []IExpressionContext
	Expression(i int) IExpressionContext
	AllCOMMA() []antlr.TerminalNode
	COMMA(i int) antlr.TerminalNode

	// IsExpressionListContext differentiates from other interfaces.
	IsExpressionListContext()
}

type ExpressionListContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyExpressionListContext() *ExpressionListContext {
	var p = new(ExpressionListContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_expressionList
	return p
}

func InitEmptyExpressionListContext(p *ExpressionListContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_expressionList
}

func (*ExpressionListContext) IsExpressionListContext() {}

func NewExpressionListContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ExpressionListContext {
	var p = new(ExpressionListContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_expressionList

	return p
}

func (s *ExpressionListContext) GetParser() antlr.Parser { return s.parser }

func (s *ExpressionListContext) AllExpression() []IExpressionContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IExpressionContext); ok {
			len++
		}
	}

	tst := make([]IExpressionContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IExpressionContext); ok {
			tst[i] = t.(IExpressionContext)
			i++
		}
	}

	return tst
}

func (s *ExpressionListContext) Expression(i int) IExpressionContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IExpressionContext); ok {
			if j == i {
				t = ctx.(antlr.RuleContext)
				break
			}
			j++
		}
	}

	if t == nil {
		return nil
	}

	return t.(IExpressionContext)
}

func (s *ExpressionListContext) AllCOMMA() []antlr.TerminalNode {
	return s.GetTokens(ArcParserCOMMA)
}

func (s *ExpressionListContext) COMMA(i int) antlr.TerminalNode {
	return s.GetToken(ArcParserCOMMA, i)
}

func (s *ExpressionListContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ExpressionListContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ExpressionListContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterExpressionList(s)
	}
}

func (s *ExpressionListContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitExpressionList(s)
	}
}

func (p *ArcParser) ExpressionList() (localctx IExpressionListContext) {
	localctx = NewExpressionListContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 156, ArcParserRULE_expressionList)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(694)
		p.Expression()
	}
	p.SetState(699)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == ArcParserCOMMA {
		{
			p.SetState(695)
			p.Match(ArcParserCOMMA)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(696)
			p.Expression()
		}

		p.SetState(701)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
		localctx.SetException(v)
		p.GetErrorHandler().ReportError(p, v)
		p.GetErrorHandler().Recover(p, v)
		p.SetError(nil)
	}
	p.ExitRule()
	return localctx
	goto errorExit // Trick to prevent compiler error if the label is not used
}
