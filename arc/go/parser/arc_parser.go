// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

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
		"", "'func'", "'if'", "'else'", "'return'", "'for'", "'break'", "'continue'",
		"'import'", "'as'", "'sequence'", "'stage'", "'next'", "'chan'", "'authority'",
		"'i8'", "'i16'", "'i32'", "'i64'", "'u8'", "'u16'", "'u32'", "'u64'",
		"'f32'", "'f64'", "'str'", "'series'", "'->'", "':='", "'$='", "'=>'",
		"'='", "'+='", "'-='", "'*='", "'/='", "'%='", "'+'", "'-'", "'*'",
		"'/'", "'%'", "'^'", "'=='", "'!='", "'<'", "'>'", "'<='", "'>='", "'and'",
		"'or'", "'not'", "'('", "')'", "'{'", "'}'", "'['", "']'", "','", "':'",
		"'.'",
	}
	staticData.SymbolicNames = []string{
		"", "FUNC", "IF", "ELSE", "RETURN", "FOR", "BREAK", "CONTINUE", "IMPORT",
		"AS", "SEQUENCE", "STAGE", "NEXT", "CHAN", "AUTHORITY", "I8", "I16",
		"I32", "I64", "U8", "U16", "U32", "U64", "F32", "F64", "STR", "SERIES",
		"ARROW", "DECLARE", "STATE_DECLARE", "TRANSITION", "ASSIGN", "PLUS_ASSIGN",
		"MINUS_ASSIGN", "STAR_ASSIGN", "SLASH_ASSIGN", "PERCENT_ASSIGN", "PLUS",
		"MINUS", "STAR", "SLASH", "PERCENT", "CARET", "EQ", "NEQ", "LT", "GT",
		"LEQ", "GEQ", "AND", "OR", "NOT", "LPAREN", "RPAREN", "LBRACE", "RBRACE",
		"LBRACKET", "RBRACKET", "COMMA", "COLON", "DOT", "INTEGER_LITERAL",
		"FLOAT_LITERAL", "STR_LITERAL_MULTI", "STR_LITERAL", "IDENTIFIER", "SINGLE_LINE_COMMENT",
		"MULTI_LINE_COMMENT", "WS",
	}
	staticData.RuleNames = []string{
		"program", "topLevelItem", "importStatement", "importItem", "importPath",
		"importPathHead", "authorityBlock", "authorityEntry", "functionDeclaration",
		"triggerList", "trigger", "outputType", "multiOutputBlock", "namedOutput",
		"inputBlock", "inputList", "input", "sequenceDeclaration", "sequenceItem",
		"stageDeclaration", "stageBody", "stageItem", "singleInvocation", "globalConstant",
		"flowStatement", "flowOperator", "routingTable", "routingEntry", "flowNode",
		"identifier", "function", "qualifiedIdentifier", "inputValues", "namedInputValues",
		"namedInputValue", "anonymousInputValues", "argumentList", "block",
		"statement", "variableDeclaration", "localVariable", "statefulVariable",
		"assignment", "compoundOp", "ifStatement", "elseIfClause", "elseClause",
		"forStatement", "forClause", "breakStatement", "continueStatement",
		"returnStatement", "type", "unitSuffix", "primitiveType", "numericType",
		"integerType", "floatType", "channelType", "seriesType", "expression",
		"logicalOrExpression", "logicalAndExpression", "equalityExpression",
		"relationalExpression", "additiveExpression", "multiplicativeExpression",
		"powerExpression", "unaryExpression", "postfixExpression", "indexOrSlice",
		"functionCallSuffix", "primaryExpression", "typeCast", "literal", "numericLiteral",
		"seriesLiteral", "expressionList",
	}
	staticData.PredictionContextCache = antlr.NewPredictionContextCache()
	staticData.serializedATN = []int32{
		4, 1, 68, 763, 2, 0, 7, 0, 2, 1, 7, 1, 2, 2, 7, 2, 2, 3, 7, 3, 2, 4, 7,
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
		73, 2, 74, 7, 74, 2, 75, 7, 75, 2, 76, 7, 76, 2, 77, 7, 77, 1, 0, 5, 0,
		158, 8, 0, 10, 0, 12, 0, 161, 9, 0, 1, 0, 1, 0, 1, 1, 1, 1, 1, 1, 1, 1,
		1, 1, 1, 1, 1, 1, 3, 1, 172, 8, 1, 1, 2, 1, 2, 1, 2, 1, 2, 1, 2, 5, 2,
		179, 8, 2, 10, 2, 12, 2, 182, 9, 2, 1, 2, 3, 2, 185, 8, 2, 1, 3, 1, 3,
		1, 3, 3, 3, 190, 8, 3, 1, 4, 1, 4, 1, 4, 5, 4, 195, 8, 4, 10, 4, 12, 4,
		198, 9, 4, 1, 5, 1, 5, 1, 6, 1, 6, 1, 6, 1, 6, 1, 6, 5, 6, 207, 8, 6, 10,
		6, 12, 6, 210, 9, 6, 1, 6, 3, 6, 213, 8, 6, 1, 7, 1, 7, 1, 7, 3, 7, 218,
		8, 7, 1, 8, 1, 8, 1, 8, 3, 8, 223, 8, 8, 1, 8, 1, 8, 3, 8, 227, 8, 8, 1,
		8, 1, 8, 3, 8, 231, 8, 8, 1, 8, 1, 8, 1, 9, 1, 9, 1, 9, 5, 9, 238, 8, 9,
		10, 9, 12, 9, 241, 9, 9, 1, 9, 3, 9, 244, 8, 9, 1, 10, 1, 10, 1, 10, 1,
		10, 3, 10, 250, 8, 10, 1, 11, 1, 11, 1, 11, 1, 11, 3, 11, 256, 8, 11, 1,
		12, 1, 12, 1, 12, 1, 12, 5, 12, 262, 8, 12, 10, 12, 12, 12, 265, 9, 12,
		1, 12, 3, 12, 268, 8, 12, 1, 12, 1, 12, 1, 13, 1, 13, 1, 13, 1, 14, 1,
		14, 3, 14, 277, 8, 14, 1, 14, 1, 14, 1, 15, 1, 15, 1, 15, 5, 15, 284, 8,
		15, 10, 15, 12, 15, 287, 9, 15, 1, 15, 3, 15, 290, 8, 15, 1, 16, 1, 16,
		1, 16, 1, 16, 3, 16, 296, 8, 16, 1, 17, 1, 17, 3, 17, 300, 8, 17, 1, 17,
		1, 17, 1, 17, 3, 17, 305, 8, 17, 1, 17, 5, 17, 308, 8, 17, 10, 17, 12,
		17, 311, 9, 17, 1, 17, 3, 17, 314, 8, 17, 3, 17, 316, 8, 17, 1, 17, 1,
		17, 1, 18, 1, 18, 1, 18, 1, 18, 3, 18, 324, 8, 18, 1, 19, 1, 19, 3, 19,
		328, 8, 19, 1, 19, 1, 19, 1, 20, 1, 20, 1, 20, 3, 20, 335, 8, 20, 1, 20,
		5, 20, 338, 8, 20, 10, 20, 12, 20, 341, 9, 20, 1, 20, 3, 20, 344, 8, 20,
		3, 20, 346, 8, 20, 1, 20, 1, 20, 1, 21, 1, 21, 1, 21, 3, 21, 353, 8, 21,
		1, 22, 1, 22, 3, 22, 357, 8, 22, 1, 23, 1, 23, 1, 23, 1, 23, 1, 23, 1,
		23, 1, 23, 1, 23, 3, 23, 367, 8, 23, 1, 24, 1, 24, 3, 24, 371, 8, 24, 1,
		24, 1, 24, 1, 24, 3, 24, 376, 8, 24, 4, 24, 378, 8, 24, 11, 24, 12, 24,
		379, 1, 25, 1, 25, 1, 26, 1, 26, 1, 26, 1, 26, 5, 26, 388, 8, 26, 10, 26,
		12, 26, 391, 9, 26, 1, 26, 3, 26, 394, 8, 26, 1, 26, 1, 26, 1, 27, 1, 27,
		1, 27, 1, 27, 1, 27, 5, 27, 403, 8, 27, 10, 27, 12, 27, 406, 9, 27, 1,
		27, 1, 27, 3, 27, 410, 8, 27, 1, 28, 1, 28, 1, 28, 1, 28, 1, 28, 1, 28,
		3, 28, 418, 8, 28, 1, 29, 1, 29, 1, 30, 1, 30, 1, 30, 1, 30, 1, 30, 3,
		30, 427, 8, 30, 1, 31, 1, 31, 1, 31, 1, 31, 1, 31, 1, 31, 1, 31, 1, 31,
		1, 31, 3, 31, 438, 8, 31, 1, 32, 1, 32, 1, 32, 1, 32, 1, 32, 1, 32, 1,
		32, 1, 32, 1, 32, 1, 32, 3, 32, 450, 8, 32, 1, 33, 1, 33, 1, 33, 5, 33,
		455, 8, 33, 10, 33, 12, 33, 458, 9, 33, 1, 33, 3, 33, 461, 8, 33, 1, 34,
		1, 34, 1, 34, 1, 34, 1, 35, 1, 35, 1, 35, 5, 35, 470, 8, 35, 10, 35, 12,
		35, 473, 9, 35, 1, 35, 3, 35, 476, 8, 35, 1, 36, 1, 36, 3, 36, 480, 8,
		36, 1, 37, 1, 37, 5, 37, 484, 8, 37, 10, 37, 12, 37, 487, 9, 37, 1, 37,
		1, 37, 1, 38, 1, 38, 1, 38, 1, 38, 1, 38, 1, 38, 1, 38, 1, 38, 3, 38, 499,
		8, 38, 1, 39, 1, 39, 3, 39, 503, 8, 39, 1, 40, 1, 40, 1, 40, 1, 40, 1,
		40, 1, 40, 1, 40, 1, 40, 3, 40, 513, 8, 40, 1, 41, 1, 41, 1, 41, 1, 41,
		1, 41, 1, 41, 1, 41, 1, 41, 3, 41, 523, 8, 41, 1, 42, 1, 42, 1, 42, 1,
		42, 1, 42, 1, 42, 1, 42, 1, 42, 1, 42, 1, 42, 1, 42, 1, 42, 1, 42, 1, 42,
		1, 42, 1, 42, 1, 42, 3, 42, 542, 8, 42, 1, 43, 1, 43, 1, 44, 1, 44, 1,
		44, 1, 44, 5, 44, 550, 8, 44, 10, 44, 12, 44, 553, 9, 44, 1, 44, 3, 44,
		556, 8, 44, 1, 45, 1, 45, 1, 45, 1, 45, 1, 45, 1, 46, 1, 46, 1, 46, 1,
		47, 1, 47, 1, 47, 1, 47, 1, 48, 1, 48, 1, 48, 1, 48, 1, 48, 1, 48, 1, 48,
		1, 48, 1, 48, 1, 48, 3, 48, 580, 8, 48, 1, 49, 1, 49, 1, 50, 1, 50, 1,
		51, 1, 51, 3, 51, 588, 8, 51, 1, 52, 1, 52, 3, 52, 592, 8, 52, 1, 52, 1,
		52, 3, 52, 596, 8, 52, 1, 53, 1, 53, 1, 54, 1, 54, 3, 54, 602, 8, 54, 1,
		55, 1, 55, 3, 55, 606, 8, 55, 1, 56, 1, 56, 1, 57, 1, 57, 1, 58, 1, 58,
		1, 58, 3, 58, 615, 8, 58, 1, 58, 1, 58, 3, 58, 619, 8, 58, 1, 59, 1, 59,
		1, 59, 3, 59, 624, 8, 59, 1, 60, 1, 60, 1, 61, 1, 61, 1, 61, 5, 61, 631,
		8, 61, 10, 61, 12, 61, 634, 9, 61, 1, 62, 1, 62, 1, 62, 5, 62, 639, 8,
		62, 10, 62, 12, 62, 642, 9, 62, 1, 63, 1, 63, 1, 63, 5, 63, 647, 8, 63,
		10, 63, 12, 63, 650, 9, 63, 1, 64, 1, 64, 1, 64, 5, 64, 655, 8, 64, 10,
		64, 12, 64, 658, 9, 64, 1, 65, 1, 65, 1, 65, 5, 65, 663, 8, 65, 10, 65,
		12, 65, 666, 9, 65, 1, 66, 1, 66, 1, 66, 5, 66, 671, 8, 66, 10, 66, 12,
		66, 674, 9, 66, 1, 67, 1, 67, 1, 67, 3, 67, 679, 8, 67, 1, 68, 1, 68, 1,
		68, 1, 68, 1, 68, 3, 68, 686, 8, 68, 1, 69, 1, 69, 1, 69, 5, 69, 691, 8,
		69, 10, 69, 12, 69, 694, 9, 69, 1, 70, 1, 70, 1, 70, 1, 70, 1, 70, 1, 70,
		3, 70, 702, 8, 70, 1, 70, 1, 70, 3, 70, 706, 8, 70, 1, 70, 3, 70, 709,
		8, 70, 1, 71, 1, 71, 3, 71, 713, 8, 71, 1, 71, 1, 71, 1, 72, 1, 72, 1,
		72, 1, 72, 1, 72, 1, 72, 1, 72, 1, 72, 3, 72, 725, 8, 72, 1, 73, 1, 73,
		1, 73, 1, 73, 1, 73, 1, 74, 1, 74, 1, 74, 1, 74, 3, 74, 736, 8, 74, 1,
		75, 3, 75, 739, 8, 75, 1, 75, 1, 75, 1, 75, 3, 75, 744, 8, 75, 1, 76, 1,
		76, 3, 76, 748, 8, 76, 1, 76, 1, 76, 1, 77, 1, 77, 1, 77, 5, 77, 755, 8,
		77, 10, 77, 12, 77, 758, 9, 77, 1, 77, 3, 77, 761, 8, 77, 1, 77, 0, 0,
		78, 0, 2, 4, 6, 8, 10, 12, 14, 16, 18, 20, 22, 24, 26, 28, 30, 32, 34,
		36, 38, 40, 42, 44, 46, 48, 50, 52, 54, 56, 58, 60, 62, 64, 66, 68, 70,
		72, 74, 76, 78, 80, 82, 84, 86, 88, 90, 92, 94, 96, 98, 100, 102, 104,
		106, 108, 110, 112, 114, 116, 118, 120, 122, 124, 126, 128, 130, 132, 134,
		136, 138, 140, 142, 144, 146, 148, 150, 152, 154, 0, 10, 2, 0, 14, 14,
		65, 65, 2, 0, 27, 27, 30, 30, 1, 0, 32, 36, 1, 0, 15, 22, 1, 0, 23, 24,
		1, 0, 43, 44, 1, 0, 45, 48, 1, 0, 37, 38, 1, 0, 39, 41, 1, 0, 61, 62, 806,
		0, 159, 1, 0, 0, 0, 2, 171, 1, 0, 0, 0, 4, 184, 1, 0, 0, 0, 6, 186, 1,
		0, 0, 0, 8, 191, 1, 0, 0, 0, 10, 199, 1, 0, 0, 0, 12, 212, 1, 0, 0, 0,
		14, 217, 1, 0, 0, 0, 16, 219, 1, 0, 0, 0, 18, 234, 1, 0, 0, 0, 20, 245,
		1, 0, 0, 0, 22, 255, 1, 0, 0, 0, 24, 257, 1, 0, 0, 0, 26, 271, 1, 0, 0,
		0, 28, 274, 1, 0, 0, 0, 30, 280, 1, 0, 0, 0, 32, 291, 1, 0, 0, 0, 34, 297,
		1, 0, 0, 0, 36, 323, 1, 0, 0, 0, 38, 325, 1, 0, 0, 0, 40, 331, 1, 0, 0,
		0, 42, 352, 1, 0, 0, 0, 44, 356, 1, 0, 0, 0, 46, 366, 1, 0, 0, 0, 48, 370,
		1, 0, 0, 0, 50, 381, 1, 0, 0, 0, 52, 383, 1, 0, 0, 0, 54, 397, 1, 0, 0,
		0, 56, 417, 1, 0, 0, 0, 58, 419, 1, 0, 0, 0, 60, 426, 1, 0, 0, 0, 62, 437,
		1, 0, 0, 0, 64, 449, 1, 0, 0, 0, 66, 451, 1, 0, 0, 0, 68, 462, 1, 0, 0,
		0, 70, 466, 1, 0, 0, 0, 72, 479, 1, 0, 0, 0, 74, 481, 1, 0, 0, 0, 76, 498,
		1, 0, 0, 0, 78, 502, 1, 0, 0, 0, 80, 512, 1, 0, 0, 0, 82, 522, 1, 0, 0,
		0, 84, 541, 1, 0, 0, 0, 86, 543, 1, 0, 0, 0, 88, 545, 1, 0, 0, 0, 90, 557,
		1, 0, 0, 0, 92, 562, 1, 0, 0, 0, 94, 565, 1, 0, 0, 0, 96, 579, 1, 0, 0,
		0, 98, 581, 1, 0, 0, 0, 100, 583, 1, 0, 0, 0, 102, 585, 1, 0, 0, 0, 104,
		595, 1, 0, 0, 0, 106, 597, 1, 0, 0, 0, 108, 601, 1, 0, 0, 0, 110, 605,
		1, 0, 0, 0, 112, 607, 1, 0, 0, 0, 114, 609, 1, 0, 0, 0, 116, 618, 1, 0,
		0, 0, 118, 620, 1, 0, 0, 0, 120, 625, 1, 0, 0, 0, 122, 627, 1, 0, 0, 0,
		124, 635, 1, 0, 0, 0, 126, 643, 1, 0, 0, 0, 128, 651, 1, 0, 0, 0, 130,
		659, 1, 0, 0, 0, 132, 667, 1, 0, 0, 0, 134, 675, 1, 0, 0, 0, 136, 685,
		1, 0, 0, 0, 138, 687, 1, 0, 0, 0, 140, 708, 1, 0, 0, 0, 142, 710, 1, 0,
		0, 0, 144, 724, 1, 0, 0, 0, 146, 726, 1, 0, 0, 0, 148, 735, 1, 0, 0, 0,
		150, 738, 1, 0, 0, 0, 152, 745, 1, 0, 0, 0, 154, 751, 1, 0, 0, 0, 156,
		158, 3, 2, 1, 0, 157, 156, 1, 0, 0, 0, 158, 161, 1, 0, 0, 0, 159, 157,
		1, 0, 0, 0, 159, 160, 1, 0, 0, 0, 160, 162, 1, 0, 0, 0, 161, 159, 1, 0,
		0, 0, 162, 163, 5, 0, 0, 1, 163, 1, 1, 0, 0, 0, 164, 172, 3, 4, 2, 0, 165,
		172, 3, 12, 6, 0, 166, 172, 3, 16, 8, 0, 167, 172, 3, 48, 24, 0, 168, 172,
		3, 34, 17, 0, 169, 172, 3, 38, 19, 0, 170, 172, 3, 46, 23, 0, 171, 164,
		1, 0, 0, 0, 171, 165, 1, 0, 0, 0, 171, 166, 1, 0, 0, 0, 171, 167, 1, 0,
		0, 0, 171, 168, 1, 0, 0, 0, 171, 169, 1, 0, 0, 0, 171, 170, 1, 0, 0, 0,
		172, 3, 1, 0, 0, 0, 173, 174, 5, 8, 0, 0, 174, 185, 3, 6, 3, 0, 175, 176,
		5, 8, 0, 0, 176, 180, 5, 52, 0, 0, 177, 179, 3, 6, 3, 0, 178, 177, 1, 0,
		0, 0, 179, 182, 1, 0, 0, 0, 180, 178, 1, 0, 0, 0, 180, 181, 1, 0, 0, 0,
		181, 183, 1, 0, 0, 0, 182, 180, 1, 0, 0, 0, 183, 185, 5, 53, 0, 0, 184,
		173, 1, 0, 0, 0, 184, 175, 1, 0, 0, 0, 185, 5, 1, 0, 0, 0, 186, 189, 3,
		8, 4, 0, 187, 188, 5, 9, 0, 0, 188, 190, 5, 65, 0, 0, 189, 187, 1, 0, 0,
		0, 189, 190, 1, 0, 0, 0, 190, 7, 1, 0, 0, 0, 191, 196, 3, 10, 5, 0, 192,
		193, 5, 60, 0, 0, 193, 195, 5, 65, 0, 0, 194, 192, 1, 0, 0, 0, 195, 198,
		1, 0, 0, 0, 196, 194, 1, 0, 0, 0, 196, 197, 1, 0, 0, 0, 197, 9, 1, 0, 0,
		0, 198, 196, 1, 0, 0, 0, 199, 200, 7, 0, 0, 0, 200, 11, 1, 0, 0, 0, 201,
		202, 5, 14, 0, 0, 202, 213, 5, 61, 0, 0, 203, 204, 5, 14, 0, 0, 204, 208,
		5, 52, 0, 0, 205, 207, 3, 14, 7, 0, 206, 205, 1, 0, 0, 0, 207, 210, 1,
		0, 0, 0, 208, 206, 1, 0, 0, 0, 208, 209, 1, 0, 0, 0, 209, 211, 1, 0, 0,
		0, 210, 208, 1, 0, 0, 0, 211, 213, 5, 53, 0, 0, 212, 201, 1, 0, 0, 0, 212,
		203, 1, 0, 0, 0, 213, 13, 1, 0, 0, 0, 214, 218, 5, 61, 0, 0, 215, 216,
		5, 65, 0, 0, 216, 218, 5, 61, 0, 0, 217, 214, 1, 0, 0, 0, 217, 215, 1,
		0, 0, 0, 218, 15, 1, 0, 0, 0, 219, 220, 5, 1, 0, 0, 220, 222, 5, 65, 0,
		0, 221, 223, 3, 28, 14, 0, 222, 221, 1, 0, 0, 0, 222, 223, 1, 0, 0, 0,
		223, 224, 1, 0, 0, 0, 224, 226, 5, 52, 0, 0, 225, 227, 3, 18, 9, 0, 226,
		225, 1, 0, 0, 0, 226, 227, 1, 0, 0, 0, 227, 228, 1, 0, 0, 0, 228, 230,
		5, 53, 0, 0, 229, 231, 3, 22, 11, 0, 230, 229, 1, 0, 0, 0, 230, 231, 1,
		0, 0, 0, 231, 232, 1, 0, 0, 0, 232, 233, 3, 74, 37, 0, 233, 17, 1, 0, 0,
		0, 234, 239, 3, 20, 10, 0, 235, 236, 5, 58, 0, 0, 236, 238, 3, 20, 10,
		0, 237, 235, 1, 0, 0, 0, 238, 241, 1, 0, 0, 0, 239, 237, 1, 0, 0, 0, 239,
		240, 1, 0, 0, 0, 240, 243, 1, 0, 0, 0, 241, 239, 1, 0, 0, 0, 242, 244,
		5, 58, 0, 0, 243, 242, 1, 0, 0, 0, 243, 244, 1, 0, 0, 0, 244, 19, 1, 0,
		0, 0, 245, 246, 5, 65, 0, 0, 246, 249, 3, 104, 52, 0, 247, 248, 5, 31,
		0, 0, 248, 250, 3, 148, 74, 0, 249, 247, 1, 0, 0, 0, 249, 250, 1, 0, 0,
		0, 250, 21, 1, 0, 0, 0, 251, 256, 3, 104, 52, 0, 252, 253, 5, 65, 0, 0,
		253, 256, 3, 104, 52, 0, 254, 256, 3, 24, 12, 0, 255, 251, 1, 0, 0, 0,
		255, 252, 1, 0, 0, 0, 255, 254, 1, 0, 0, 0, 256, 23, 1, 0, 0, 0, 257, 258,
		5, 52, 0, 0, 258, 263, 3, 26, 13, 0, 259, 260, 5, 58, 0, 0, 260, 262, 3,
		26, 13, 0, 261, 259, 1, 0, 0, 0, 262, 265, 1, 0, 0, 0, 263, 261, 1, 0,
		0, 0, 263, 264, 1, 0, 0, 0, 264, 267, 1, 0, 0, 0, 265, 263, 1, 0, 0, 0,
		266, 268, 5, 58, 0, 0, 267, 266, 1, 0, 0, 0, 267, 268, 1, 0, 0, 0, 268,
		269, 1, 0, 0, 0, 269, 270, 5, 53, 0, 0, 270, 25, 1, 0, 0, 0, 271, 272,
		5, 65, 0, 0, 272, 273, 3, 104, 52, 0, 273, 27, 1, 0, 0, 0, 274, 276, 5,
		54, 0, 0, 275, 277, 3, 30, 15, 0, 276, 275, 1, 0, 0, 0, 276, 277, 1, 0,
		0, 0, 277, 278, 1, 0, 0, 0, 278, 279, 5, 55, 0, 0, 279, 29, 1, 0, 0, 0,
		280, 285, 3, 32, 16, 0, 281, 282, 5, 58, 0, 0, 282, 284, 3, 32, 16, 0,
		283, 281, 1, 0, 0, 0, 284, 287, 1, 0, 0, 0, 285, 283, 1, 0, 0, 0, 285,
		286, 1, 0, 0, 0, 286, 289, 1, 0, 0, 0, 287, 285, 1, 0, 0, 0, 288, 290,
		5, 58, 0, 0, 289, 288, 1, 0, 0, 0, 289, 290, 1, 0, 0, 0, 290, 31, 1, 0,
		0, 0, 291, 292, 5, 65, 0, 0, 292, 295, 3, 104, 52, 0, 293, 294, 5, 31,
		0, 0, 294, 296, 3, 148, 74, 0, 295, 293, 1, 0, 0, 0, 295, 296, 1, 0, 0,
		0, 296, 33, 1, 0, 0, 0, 297, 299, 5, 10, 0, 0, 298, 300, 5, 65, 0, 0, 299,
		298, 1, 0, 0, 0, 299, 300, 1, 0, 0, 0, 300, 301, 1, 0, 0, 0, 301, 315,
		5, 54, 0, 0, 302, 309, 3, 36, 18, 0, 303, 305, 5, 58, 0, 0, 304, 303, 1,
		0, 0, 0, 304, 305, 1, 0, 0, 0, 305, 306, 1, 0, 0, 0, 306, 308, 3, 36, 18,
		0, 307, 304, 1, 0, 0, 0, 308, 311, 1, 0, 0, 0, 309, 307, 1, 0, 0, 0, 309,
		310, 1, 0, 0, 0, 310, 313, 1, 0, 0, 0, 311, 309, 1, 0, 0, 0, 312, 314,
		5, 58, 0, 0, 313, 312, 1, 0, 0, 0, 313, 314, 1, 0, 0, 0, 314, 316, 1, 0,
		0, 0, 315, 302, 1, 0, 0, 0, 315, 316, 1, 0, 0, 0, 316, 317, 1, 0, 0, 0,
		317, 318, 5, 55, 0, 0, 318, 35, 1, 0, 0, 0, 319, 324, 3, 38, 19, 0, 320,
		324, 3, 34, 17, 0, 321, 324, 3, 48, 24, 0, 322, 324, 3, 44, 22, 0, 323,
		319, 1, 0, 0, 0, 323, 320, 1, 0, 0, 0, 323, 321, 1, 0, 0, 0, 323, 322,
		1, 0, 0, 0, 324, 37, 1, 0, 0, 0, 325, 327, 5, 11, 0, 0, 326, 328, 5, 65,
		0, 0, 327, 326, 1, 0, 0, 0, 327, 328, 1, 0, 0, 0, 328, 329, 1, 0, 0, 0,
		329, 330, 3, 40, 20, 0, 330, 39, 1, 0, 0, 0, 331, 345, 5, 54, 0, 0, 332,
		339, 3, 42, 21, 0, 333, 335, 5, 58, 0, 0, 334, 333, 1, 0, 0, 0, 334, 335,
		1, 0, 0, 0, 335, 336, 1, 0, 0, 0, 336, 338, 3, 42, 21, 0, 337, 334, 1,
		0, 0, 0, 338, 341, 1, 0, 0, 0, 339, 337, 1, 0, 0, 0, 339, 340, 1, 0, 0,
		0, 340, 343, 1, 0, 0, 0, 341, 339, 1, 0, 0, 0, 342, 344, 5, 58, 0, 0, 343,
		342, 1, 0, 0, 0, 343, 344, 1, 0, 0, 0, 344, 346, 1, 0, 0, 0, 345, 332,
		1, 0, 0, 0, 345, 346, 1, 0, 0, 0, 346, 347, 1, 0, 0, 0, 347, 348, 5, 55,
		0, 0, 348, 41, 1, 0, 0, 0, 349, 353, 3, 48, 24, 0, 350, 353, 3, 44, 22,
		0, 351, 353, 3, 34, 17, 0, 352, 349, 1, 0, 0, 0, 352, 350, 1, 0, 0, 0,
		352, 351, 1, 0, 0, 0, 353, 43, 1, 0, 0, 0, 354, 357, 3, 60, 30, 0, 355,
		357, 3, 120, 60, 0, 356, 354, 1, 0, 0, 0, 356, 355, 1, 0, 0, 0, 357, 45,
		1, 0, 0, 0, 358, 359, 5, 65, 0, 0, 359, 360, 5, 28, 0, 0, 360, 367, 3,
		148, 74, 0, 361, 362, 5, 65, 0, 0, 362, 363, 3, 104, 52, 0, 363, 364, 5,
		28, 0, 0, 364, 365, 3, 148, 74, 0, 365, 367, 1, 0, 0, 0, 366, 358, 1, 0,
		0, 0, 366, 361, 1, 0, 0, 0, 367, 47, 1, 0, 0, 0, 368, 371, 3, 52, 26, 0,
		369, 371, 3, 56, 28, 0, 370, 368, 1, 0, 0, 0, 370, 369, 1, 0, 0, 0, 371,
		377, 1, 0, 0, 0, 372, 375, 3, 50, 25, 0, 373, 376, 3, 52, 26, 0, 374, 376,
		3, 56, 28, 0, 375, 373, 1, 0, 0, 0, 375, 374, 1, 0, 0, 0, 376, 378, 1,
		0, 0, 0, 377, 372, 1, 0, 0, 0, 378, 379, 1, 0, 0, 0, 379, 377, 1, 0, 0,
		0, 379, 380, 1, 0, 0, 0, 380, 49, 1, 0, 0, 0, 381, 382, 7, 1, 0, 0, 382,
		51, 1, 0, 0, 0, 383, 384, 5, 54, 0, 0, 384, 389, 3, 54, 27, 0, 385, 386,
		5, 58, 0, 0, 386, 388, 3, 54, 27, 0, 387, 385, 1, 0, 0, 0, 388, 391, 1,
		0, 0, 0, 389, 387, 1, 0, 0, 0, 389, 390, 1, 0, 0, 0, 390, 393, 1, 0, 0,
		0, 391, 389, 1, 0, 0, 0, 392, 394, 5, 58, 0, 0, 393, 392, 1, 0, 0, 0, 393,
		394, 1, 0, 0, 0, 394, 395, 1, 0, 0, 0, 395, 396, 5, 55, 0, 0, 396, 53,
		1, 0, 0, 0, 397, 398, 5, 65, 0, 0, 398, 399, 5, 59, 0, 0, 399, 404, 3,
		56, 28, 0, 400, 401, 5, 27, 0, 0, 401, 403, 3, 56, 28, 0, 402, 400, 1,
		0, 0, 0, 403, 406, 1, 0, 0, 0, 404, 402, 1, 0, 0, 0, 404, 405, 1, 0, 0,
		0, 405, 409, 1, 0, 0, 0, 406, 404, 1, 0, 0, 0, 407, 408, 5, 59, 0, 0, 408,
		410, 5, 65, 0, 0, 409, 407, 1, 0, 0, 0, 409, 410, 1, 0, 0, 0, 410, 55,
		1, 0, 0, 0, 411, 418, 3, 58, 29, 0, 412, 418, 3, 60, 30, 0, 413, 418, 3,
		120, 60, 0, 414, 418, 3, 38, 19, 0, 415, 418, 3, 34, 17, 0, 416, 418, 5,
		12, 0, 0, 417, 411, 1, 0, 0, 0, 417, 412, 1, 0, 0, 0, 417, 413, 1, 0, 0,
		0, 417, 414, 1, 0, 0, 0, 417, 415, 1, 0, 0, 0, 417, 416, 1, 0, 0, 0, 418,
		57, 1, 0, 0, 0, 419, 420, 5, 65, 0, 0, 420, 59, 1, 0, 0, 0, 421, 422, 3,
		62, 31, 0, 422, 423, 3, 64, 32, 0, 423, 427, 1, 0, 0, 0, 424, 425, 5, 65,
		0, 0, 425, 427, 3, 64, 32, 0, 426, 421, 1, 0, 0, 0, 426, 424, 1, 0, 0,
		0, 427, 61, 1, 0, 0, 0, 428, 429, 5, 65, 0, 0, 429, 430, 5, 60, 0, 0, 430,
		438, 5, 65, 0, 0, 431, 432, 5, 65, 0, 0, 432, 433, 5, 60, 0, 0, 433, 438,
		5, 5, 0, 0, 434, 435, 5, 14, 0, 0, 435, 436, 5, 60, 0, 0, 436, 438, 5,
		65, 0, 0, 437, 428, 1, 0, 0, 0, 437, 431, 1, 0, 0, 0, 437, 434, 1, 0, 0,
		0, 438, 63, 1, 0, 0, 0, 439, 440, 5, 54, 0, 0, 440, 450, 5, 55, 0, 0, 441,
		442, 5, 54, 0, 0, 442, 443, 3, 66, 33, 0, 443, 444, 5, 55, 0, 0, 444, 450,
		1, 0, 0, 0, 445, 446, 5, 54, 0, 0, 446, 447, 3, 70, 35, 0, 447, 448, 5,
		55, 0, 0, 448, 450, 1, 0, 0, 0, 449, 439, 1, 0, 0, 0, 449, 441, 1, 0, 0,
		0, 449, 445, 1, 0, 0, 0, 450, 65, 1, 0, 0, 0, 451, 456, 3, 68, 34, 0, 452,
		453, 5, 58, 0, 0, 453, 455, 3, 68, 34, 0, 454, 452, 1, 0, 0, 0, 455, 458,
		1, 0, 0, 0, 456, 454, 1, 0, 0, 0, 456, 457, 1, 0, 0, 0, 457, 460, 1, 0,
		0, 0, 458, 456, 1, 0, 0, 0, 459, 461, 5, 58, 0, 0, 460, 459, 1, 0, 0, 0,
		460, 461, 1, 0, 0, 0, 461, 67, 1, 0, 0, 0, 462, 463, 5, 65, 0, 0, 463,
		464, 5, 31, 0, 0, 464, 465, 3, 120, 60, 0, 465, 69, 1, 0, 0, 0, 466, 471,
		3, 120, 60, 0, 467, 468, 5, 58, 0, 0, 468, 470, 3, 120, 60, 0, 469, 467,
		1, 0, 0, 0, 470, 473, 1, 0, 0, 0, 471, 469, 1, 0, 0, 0, 471, 472, 1, 0,
		0, 0, 472, 475, 1, 0, 0, 0, 473, 471, 1, 0, 0, 0, 474, 476, 5, 58, 0, 0,
		475, 474, 1, 0, 0, 0, 475, 476, 1, 0, 0, 0, 476, 71, 1, 0, 0, 0, 477, 480,
		3, 66, 33, 0, 478, 480, 3, 70, 35, 0, 479, 477, 1, 0, 0, 0, 479, 478, 1,
		0, 0, 0, 480, 73, 1, 0, 0, 0, 481, 485, 5, 54, 0, 0, 482, 484, 3, 76, 38,
		0, 483, 482, 1, 0, 0, 0, 484, 487, 1, 0, 0, 0, 485, 483, 1, 0, 0, 0, 485,
		486, 1, 0, 0, 0, 486, 488, 1, 0, 0, 0, 487, 485, 1, 0, 0, 0, 488, 489,
		5, 55, 0, 0, 489, 75, 1, 0, 0, 0, 490, 499, 3, 78, 39, 0, 491, 499, 3,
		84, 42, 0, 492, 499, 3, 88, 44, 0, 493, 499, 3, 94, 47, 0, 494, 499, 3,
		98, 49, 0, 495, 499, 3, 100, 50, 0, 496, 499, 3, 102, 51, 0, 497, 499,
		3, 120, 60, 0, 498, 490, 1, 0, 0, 0, 498, 491, 1, 0, 0, 0, 498, 492, 1,
		0, 0, 0, 498, 493, 1, 0, 0, 0, 498, 494, 1, 0, 0, 0, 498, 495, 1, 0, 0,
		0, 498, 496, 1, 0, 0, 0, 498, 497, 1, 0, 0, 0, 499, 77, 1, 0, 0, 0, 500,
		503, 3, 80, 40, 0, 501, 503, 3, 82, 41, 0, 502, 500, 1, 0, 0, 0, 502, 501,
		1, 0, 0, 0, 503, 79, 1, 0, 0, 0, 504, 505, 5, 65, 0, 0, 505, 506, 5, 28,
		0, 0, 506, 513, 3, 120, 60, 0, 507, 508, 5, 65, 0, 0, 508, 509, 3, 104,
		52, 0, 509, 510, 5, 28, 0, 0, 510, 511, 3, 120, 60, 0, 511, 513, 1, 0,
		0, 0, 512, 504, 1, 0, 0, 0, 512, 507, 1, 0, 0, 0, 513, 81, 1, 0, 0, 0,
		514, 515, 5, 65, 0, 0, 515, 516, 5, 29, 0, 0, 516, 523, 3, 120, 60, 0,
		517, 518, 5, 65, 0, 0, 518, 519, 3, 104, 52, 0, 519, 520, 5, 29, 0, 0,
		520, 521, 3, 120, 60, 0, 521, 523, 1, 0, 0, 0, 522, 514, 1, 0, 0, 0, 522,
		517, 1, 0, 0, 0, 523, 83, 1, 0, 0, 0, 524, 525, 5, 65, 0, 0, 525, 526,
		5, 31, 0, 0, 526, 542, 3, 120, 60, 0, 527, 528, 5, 65, 0, 0, 528, 529,
		3, 140, 70, 0, 529, 530, 5, 31, 0, 0, 530, 531, 3, 120, 60, 0, 531, 542,
		1, 0, 0, 0, 532, 533, 5, 65, 0, 0, 533, 534, 3, 86, 43, 0, 534, 535, 3,
		120, 60, 0, 535, 542, 1, 0, 0, 0, 536, 537, 5, 65, 0, 0, 537, 538, 3, 140,
		70, 0, 538, 539, 3, 86, 43, 0, 539, 540, 3, 120, 60, 0, 540, 542, 1, 0,
		0, 0, 541, 524, 1, 0, 0, 0, 541, 527, 1, 0, 0, 0, 541, 532, 1, 0, 0, 0,
		541, 536, 1, 0, 0, 0, 542, 85, 1, 0, 0, 0, 543, 544, 7, 2, 0, 0, 544, 87,
		1, 0, 0, 0, 545, 546, 5, 2, 0, 0, 546, 547, 3, 120, 60, 0, 547, 551, 3,
		74, 37, 0, 548, 550, 3, 90, 45, 0, 549, 548, 1, 0, 0, 0, 550, 553, 1, 0,
		0, 0, 551, 549, 1, 0, 0, 0, 551, 552, 1, 0, 0, 0, 552, 555, 1, 0, 0, 0,
		553, 551, 1, 0, 0, 0, 554, 556, 3, 92, 46, 0, 555, 554, 1, 0, 0, 0, 555,
		556, 1, 0, 0, 0, 556, 89, 1, 0, 0, 0, 557, 558, 5, 3, 0, 0, 558, 559, 5,
		2, 0, 0, 559, 560, 3, 120, 60, 0, 560, 561, 3, 74, 37, 0, 561, 91, 1, 0,
		0, 0, 562, 563, 5, 3, 0, 0, 563, 564, 3, 74, 37, 0, 564, 93, 1, 0, 0, 0,
		565, 566, 5, 5, 0, 0, 566, 567, 3, 96, 48, 0, 567, 568, 3, 74, 37, 0, 568,
		95, 1, 0, 0, 0, 569, 570, 5, 65, 0, 0, 570, 571, 5, 58, 0, 0, 571, 572,
		5, 65, 0, 0, 572, 573, 5, 28, 0, 0, 573, 580, 3, 120, 60, 0, 574, 575,
		5, 65, 0, 0, 575, 576, 5, 28, 0, 0, 576, 580, 3, 120, 60, 0, 577, 580,
		3, 120, 60, 0, 578, 580, 1, 0, 0, 0, 579, 569, 1, 0, 0, 0, 579, 574, 1,
		0, 0, 0, 579, 577, 1, 0, 0, 0, 579, 578, 1, 0, 0, 0, 580, 97, 1, 0, 0,
		0, 581, 582, 5, 6, 0, 0, 582, 99, 1, 0, 0, 0, 583, 584, 5, 7, 0, 0, 584,
		101, 1, 0, 0, 0, 585, 587, 5, 4, 0, 0, 586, 588, 3, 120, 60, 0, 587, 586,
		1, 0, 0, 0, 587, 588, 1, 0, 0, 0, 588, 103, 1, 0, 0, 0, 589, 591, 3, 108,
		54, 0, 590, 592, 3, 106, 53, 0, 591, 590, 1, 0, 0, 0, 591, 592, 1, 0, 0,
		0, 592, 596, 1, 0, 0, 0, 593, 596, 3, 116, 58, 0, 594, 596, 3, 118, 59,
		0, 595, 589, 1, 0, 0, 0, 595, 593, 1, 0, 0, 0, 595, 594, 1, 0, 0, 0, 596,
		105, 1, 0, 0, 0, 597, 598, 5, 65, 0, 0, 598, 107, 1, 0, 0, 0, 599, 602,
		3, 110, 55, 0, 600, 602, 5, 25, 0, 0, 601, 599, 1, 0, 0, 0, 601, 600, 1,
		0, 0, 0, 602, 109, 1, 0, 0, 0, 603, 606, 3, 112, 56, 0, 604, 606, 3, 114,
		57, 0, 605, 603, 1, 0, 0, 0, 605, 604, 1, 0, 0, 0, 606, 111, 1, 0, 0, 0,
		607, 608, 7, 3, 0, 0, 608, 113, 1, 0, 0, 0, 609, 610, 7, 4, 0, 0, 610,
		115, 1, 0, 0, 0, 611, 612, 5, 13, 0, 0, 612, 614, 3, 108, 54, 0, 613, 615,
		3, 106, 53, 0, 614, 613, 1, 0, 0, 0, 614, 615, 1, 0, 0, 0, 615, 619, 1,
		0, 0, 0, 616, 617, 5, 13, 0, 0, 617, 619, 3, 118, 59, 0, 618, 611, 1, 0,
		0, 0, 618, 616, 1, 0, 0, 0, 619, 117, 1, 0, 0, 0, 620, 621, 5, 26, 0, 0,
		621, 623, 3, 108, 54, 0, 622, 624, 3, 106, 53, 0, 623, 622, 1, 0, 0, 0,
		623, 624, 1, 0, 0, 0, 624, 119, 1, 0, 0, 0, 625, 626, 3, 122, 61, 0, 626,
		121, 1, 0, 0, 0, 627, 632, 3, 124, 62, 0, 628, 629, 5, 50, 0, 0, 629, 631,
		3, 124, 62, 0, 630, 628, 1, 0, 0, 0, 631, 634, 1, 0, 0, 0, 632, 630, 1,
		0, 0, 0, 632, 633, 1, 0, 0, 0, 633, 123, 1, 0, 0, 0, 634, 632, 1, 0, 0,
		0, 635, 640, 3, 126, 63, 0, 636, 637, 5, 49, 0, 0, 637, 639, 3, 126, 63,
		0, 638, 636, 1, 0, 0, 0, 639, 642, 1, 0, 0, 0, 640, 638, 1, 0, 0, 0, 640,
		641, 1, 0, 0, 0, 641, 125, 1, 0, 0, 0, 642, 640, 1, 0, 0, 0, 643, 648,
		3, 128, 64, 0, 644, 645, 7, 5, 0, 0, 645, 647, 3, 128, 64, 0, 646, 644,
		1, 0, 0, 0, 647, 650, 1, 0, 0, 0, 648, 646, 1, 0, 0, 0, 648, 649, 1, 0,
		0, 0, 649, 127, 1, 0, 0, 0, 650, 648, 1, 0, 0, 0, 651, 656, 3, 130, 65,
		0, 652, 653, 7, 6, 0, 0, 653, 655, 3, 130, 65, 0, 654, 652, 1, 0, 0, 0,
		655, 658, 1, 0, 0, 0, 656, 654, 1, 0, 0, 0, 656, 657, 1, 0, 0, 0, 657,
		129, 1, 0, 0, 0, 658, 656, 1, 0, 0, 0, 659, 664, 3, 132, 66, 0, 660, 661,
		7, 7, 0, 0, 661, 663, 3, 132, 66, 0, 662, 660, 1, 0, 0, 0, 663, 666, 1,
		0, 0, 0, 664, 662, 1, 0, 0, 0, 664, 665, 1, 0, 0, 0, 665, 131, 1, 0, 0,
		0, 666, 664, 1, 0, 0, 0, 667, 672, 3, 134, 67, 0, 668, 669, 7, 8, 0, 0,
		669, 671, 3, 134, 67, 0, 670, 668, 1, 0, 0, 0, 671, 674, 1, 0, 0, 0, 672,
		670, 1, 0, 0, 0, 672, 673, 1, 0, 0, 0, 673, 133, 1, 0, 0, 0, 674, 672,
		1, 0, 0, 0, 675, 678, 3, 136, 68, 0, 676, 677, 5, 42, 0, 0, 677, 679, 3,
		134, 67, 0, 678, 676, 1, 0, 0, 0, 678, 679, 1, 0, 0, 0, 679, 135, 1, 0,
		0, 0, 680, 681, 5, 38, 0, 0, 681, 686, 3, 136, 68, 0, 682, 683, 5, 51,
		0, 0, 683, 686, 3, 136, 68, 0, 684, 686, 3, 138, 69, 0, 685, 680, 1, 0,
		0, 0, 685, 682, 1, 0, 0, 0, 685, 684, 1, 0, 0, 0, 686, 137, 1, 0, 0, 0,
		687, 692, 3, 144, 72, 0, 688, 691, 3, 140, 70, 0, 689, 691, 3, 142, 71,
		0, 690, 688, 1, 0, 0, 0, 690, 689, 1, 0, 0, 0, 691, 694, 1, 0, 0, 0, 692,
		690, 1, 0, 0, 0, 692, 693, 1, 0, 0, 0, 693, 139, 1, 0, 0, 0, 694, 692,
		1, 0, 0, 0, 695, 696, 5, 56, 0, 0, 696, 697, 3, 120, 60, 0, 697, 698, 5,
		57, 0, 0, 698, 709, 1, 0, 0, 0, 699, 701, 5, 56, 0, 0, 700, 702, 3, 120,
		60, 0, 701, 700, 1, 0, 0, 0, 701, 702, 1, 0, 0, 0, 702, 703, 1, 0, 0, 0,
		703, 705, 5, 59, 0, 0, 704, 706, 3, 120, 60, 0, 705, 704, 1, 0, 0, 0, 705,
		706, 1, 0, 0, 0, 706, 707, 1, 0, 0, 0, 707, 709, 5, 57, 0, 0, 708, 695,
		1, 0, 0, 0, 708, 699, 1, 0, 0, 0, 709, 141, 1, 0, 0, 0, 710, 712, 5, 52,
		0, 0, 711, 713, 3, 72, 36, 0, 712, 711, 1, 0, 0, 0, 712, 713, 1, 0, 0,
		0, 713, 714, 1, 0, 0, 0, 714, 715, 5, 53, 0, 0, 715, 143, 1, 0, 0, 0, 716,
		725, 3, 148, 74, 0, 717, 725, 3, 62, 31, 0, 718, 725, 5, 65, 0, 0, 719,
		720, 5, 52, 0, 0, 720, 721, 3, 120, 60, 0, 721, 722, 5, 53, 0, 0, 722,
		725, 1, 0, 0, 0, 723, 725, 3, 146, 73, 0, 724, 716, 1, 0, 0, 0, 724, 717,
		1, 0, 0, 0, 724, 718, 1, 0, 0, 0, 724, 719, 1, 0, 0, 0, 724, 723, 1, 0,
		0, 0, 725, 145, 1, 0, 0, 0, 726, 727, 3, 104, 52, 0, 727, 728, 5, 52, 0,
		0, 728, 729, 3, 120, 60, 0, 729, 730, 5, 53, 0, 0, 730, 147, 1, 0, 0, 0,
		731, 736, 3, 150, 75, 0, 732, 736, 5, 64, 0, 0, 733, 736, 5, 63, 0, 0,
		734, 736, 3, 152, 76, 0, 735, 731, 1, 0, 0, 0, 735, 732, 1, 0, 0, 0, 735,
		733, 1, 0, 0, 0, 735, 734, 1, 0, 0, 0, 736, 149, 1, 0, 0, 0, 737, 739,
		5, 38, 0, 0, 738, 737, 1, 0, 0, 0, 738, 739, 1, 0, 0, 0, 739, 740, 1, 0,
		0, 0, 740, 743, 7, 9, 0, 0, 741, 742, 4, 75, 0, 0, 742, 744, 5, 65, 0,
		0, 743, 741, 1, 0, 0, 0, 743, 744, 1, 0, 0, 0, 744, 151, 1, 0, 0, 0, 745,
		747, 5, 56, 0, 0, 746, 748, 3, 154, 77, 0, 747, 746, 1, 0, 0, 0, 747, 748,
		1, 0, 0, 0, 748, 749, 1, 0, 0, 0, 749, 750, 5, 57, 0, 0, 750, 153, 1, 0,
		0, 0, 751, 756, 3, 120, 60, 0, 752, 753, 5, 58, 0, 0, 753, 755, 3, 120,
		60, 0, 754, 752, 1, 0, 0, 0, 755, 758, 1, 0, 0, 0, 756, 754, 1, 0, 0, 0,
		756, 757, 1, 0, 0, 0, 757, 760, 1, 0, 0, 0, 758, 756, 1, 0, 0, 0, 759,
		761, 5, 58, 0, 0, 760, 759, 1, 0, 0, 0, 760, 761, 1, 0, 0, 0, 761, 155,
		1, 0, 0, 0, 90, 159, 171, 180, 184, 189, 196, 208, 212, 217, 222, 226,
		230, 239, 243, 249, 255, 263, 267, 276, 285, 289, 295, 299, 304, 309, 313,
		315, 323, 327, 334, 339, 343, 345, 352, 356, 366, 370, 375, 379, 389, 393,
		404, 409, 417, 426, 437, 449, 456, 460, 471, 475, 479, 485, 498, 502, 512,
		522, 541, 551, 555, 579, 587, 591, 595, 601, 605, 614, 618, 623, 632, 640,
		648, 656, 664, 672, 678, 685, 690, 692, 701, 705, 708, 712, 724, 735, 738,
		743, 747, 756, 760,
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
	ArcParserFOR                 = 5
	ArcParserBREAK               = 6
	ArcParserCONTINUE            = 7
	ArcParserIMPORT              = 8
	ArcParserAS                  = 9
	ArcParserSEQUENCE            = 10
	ArcParserSTAGE               = 11
	ArcParserNEXT                = 12
	ArcParserCHAN                = 13
	ArcParserAUTHORITY           = 14
	ArcParserI8                  = 15
	ArcParserI16                 = 16
	ArcParserI32                 = 17
	ArcParserI64                 = 18
	ArcParserU8                  = 19
	ArcParserU16                 = 20
	ArcParserU32                 = 21
	ArcParserU64                 = 22
	ArcParserF32                 = 23
	ArcParserF64                 = 24
	ArcParserSTR                 = 25
	ArcParserSERIES              = 26
	ArcParserARROW               = 27
	ArcParserDECLARE             = 28
	ArcParserSTATE_DECLARE       = 29
	ArcParserTRANSITION          = 30
	ArcParserASSIGN              = 31
	ArcParserPLUS_ASSIGN         = 32
	ArcParserMINUS_ASSIGN        = 33
	ArcParserSTAR_ASSIGN         = 34
	ArcParserSLASH_ASSIGN        = 35
	ArcParserPERCENT_ASSIGN      = 36
	ArcParserPLUS                = 37
	ArcParserMINUS               = 38
	ArcParserSTAR                = 39
	ArcParserSLASH               = 40
	ArcParserPERCENT             = 41
	ArcParserCARET               = 42
	ArcParserEQ                  = 43
	ArcParserNEQ                 = 44
	ArcParserLT                  = 45
	ArcParserGT                  = 46
	ArcParserLEQ                 = 47
	ArcParserGEQ                 = 48
	ArcParserAND                 = 49
	ArcParserOR                  = 50
	ArcParserNOT                 = 51
	ArcParserLPAREN              = 52
	ArcParserRPAREN              = 53
	ArcParserLBRACE              = 54
	ArcParserRBRACE              = 55
	ArcParserLBRACKET            = 56
	ArcParserRBRACKET            = 57
	ArcParserCOMMA               = 58
	ArcParserCOLON               = 59
	ArcParserDOT                 = 60
	ArcParserINTEGER_LITERAL     = 61
	ArcParserFLOAT_LITERAL       = 62
	ArcParserSTR_LITERAL_MULTI   = 63
	ArcParserSTR_LITERAL         = 64
	ArcParserIDENTIFIER          = 65
	ArcParserSINGLE_LINE_COMMENT = 66
	ArcParserMULTI_LINE_COMMENT  = 67
	ArcParserWS                  = 68
)

// ArcParser rules.
const (
	ArcParserRULE_program                  = 0
	ArcParserRULE_topLevelItem             = 1
	ArcParserRULE_importStatement          = 2
	ArcParserRULE_importItem               = 3
	ArcParserRULE_importPath               = 4
	ArcParserRULE_importPathHead           = 5
	ArcParserRULE_authorityBlock           = 6
	ArcParserRULE_authorityEntry           = 7
	ArcParserRULE_functionDeclaration      = 8
	ArcParserRULE_triggerList              = 9
	ArcParserRULE_trigger                  = 10
	ArcParserRULE_outputType               = 11
	ArcParserRULE_multiOutputBlock         = 12
	ArcParserRULE_namedOutput              = 13
	ArcParserRULE_inputBlock               = 14
	ArcParserRULE_inputList                = 15
	ArcParserRULE_input                    = 16
	ArcParserRULE_sequenceDeclaration      = 17
	ArcParserRULE_sequenceItem             = 18
	ArcParserRULE_stageDeclaration         = 19
	ArcParserRULE_stageBody                = 20
	ArcParserRULE_stageItem                = 21
	ArcParserRULE_singleInvocation         = 22
	ArcParserRULE_globalConstant           = 23
	ArcParserRULE_flowStatement            = 24
	ArcParserRULE_flowOperator             = 25
	ArcParserRULE_routingTable             = 26
	ArcParserRULE_routingEntry             = 27
	ArcParserRULE_flowNode                 = 28
	ArcParserRULE_identifier               = 29
	ArcParserRULE_function                 = 30
	ArcParserRULE_qualifiedIdentifier      = 31
	ArcParserRULE_inputValues              = 32
	ArcParserRULE_namedInputValues         = 33
	ArcParserRULE_namedInputValue          = 34
	ArcParserRULE_anonymousInputValues     = 35
	ArcParserRULE_argumentList             = 36
	ArcParserRULE_block                    = 37
	ArcParserRULE_statement                = 38
	ArcParserRULE_variableDeclaration      = 39
	ArcParserRULE_localVariable            = 40
	ArcParserRULE_statefulVariable         = 41
	ArcParserRULE_assignment               = 42
	ArcParserRULE_compoundOp               = 43
	ArcParserRULE_ifStatement              = 44
	ArcParserRULE_elseIfClause             = 45
	ArcParserRULE_elseClause               = 46
	ArcParserRULE_forStatement             = 47
	ArcParserRULE_forClause                = 48
	ArcParserRULE_breakStatement           = 49
	ArcParserRULE_continueStatement        = 50
	ArcParserRULE_returnStatement          = 51
	ArcParserRULE_type                     = 52
	ArcParserRULE_unitSuffix               = 53
	ArcParserRULE_primitiveType            = 54
	ArcParserRULE_numericType              = 55
	ArcParserRULE_integerType              = 56
	ArcParserRULE_floatType                = 57
	ArcParserRULE_channelType              = 58
	ArcParserRULE_seriesType               = 59
	ArcParserRULE_expression               = 60
	ArcParserRULE_logicalOrExpression      = 61
	ArcParserRULE_logicalAndExpression     = 62
	ArcParserRULE_equalityExpression       = 63
	ArcParserRULE_relationalExpression     = 64
	ArcParserRULE_additiveExpression       = 65
	ArcParserRULE_multiplicativeExpression = 66
	ArcParserRULE_powerExpression          = 67
	ArcParserRULE_unaryExpression          = 68
	ArcParserRULE_postfixExpression        = 69
	ArcParserRULE_indexOrSlice             = 70
	ArcParserRULE_functionCallSuffix       = 71
	ArcParserRULE_primaryExpression        = 72
	ArcParserRULE_typeCast                 = 73
	ArcParserRULE_literal                  = 74
	ArcParserRULE_numericLiteral           = 75
	ArcParserRULE_seriesLiteral            = 76
	ArcParserRULE_expressionList           = 77
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

func (s *ProgramContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitProgram(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) Program() (localctx IProgramContext) {
	localctx = NewProgramContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 0, ArcParserRULE_program)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(159)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for ((int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&-2209015342213104382) != 0) || _la == ArcParserSTR_LITERAL || _la == ArcParserIDENTIFIER {
		{
			p.SetState(156)
			p.TopLevelItem()
		}

		p.SetState(161)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(162)
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
	ImportStatement() IImportStatementContext
	AuthorityBlock() IAuthorityBlockContext
	FunctionDeclaration() IFunctionDeclarationContext
	FlowStatement() IFlowStatementContext
	SequenceDeclaration() ISequenceDeclarationContext
	StageDeclaration() IStageDeclarationContext
	GlobalConstant() IGlobalConstantContext

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

func (s *TopLevelItemContext) ImportStatement() IImportStatementContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IImportStatementContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IImportStatementContext)
}

func (s *TopLevelItemContext) AuthorityBlock() IAuthorityBlockContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IAuthorityBlockContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IAuthorityBlockContext)
}

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

func (s *TopLevelItemContext) GlobalConstant() IGlobalConstantContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IGlobalConstantContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IGlobalConstantContext)
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

func (s *TopLevelItemContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitTopLevelItem(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) TopLevelItem() (localctx ITopLevelItemContext) {
	localctx = NewTopLevelItemContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 2, ArcParserRULE_topLevelItem)
	p.SetState(171)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 1, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(164)
			p.ImportStatement()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(165)
			p.AuthorityBlock()
		}

	case 3:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(166)
			p.FunctionDeclaration()
		}

	case 4:
		p.EnterOuterAlt(localctx, 4)
		{
			p.SetState(167)
			p.FlowStatement()
		}

	case 5:
		p.EnterOuterAlt(localctx, 5)
		{
			p.SetState(168)
			p.SequenceDeclaration()
		}

	case 6:
		p.EnterOuterAlt(localctx, 6)
		{
			p.SetState(169)
			p.StageDeclaration()
		}

	case 7:
		p.EnterOuterAlt(localctx, 7)
		{
			p.SetState(170)
			p.GlobalConstant()
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

// IImportStatementContext is an interface to support dynamic dispatch.
type IImportStatementContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	IMPORT() antlr.TerminalNode
	AllImportItem() []IImportItemContext
	ImportItem(i int) IImportItemContext
	LPAREN() antlr.TerminalNode
	RPAREN() antlr.TerminalNode

	// IsImportStatementContext differentiates from other interfaces.
	IsImportStatementContext()
}

type ImportStatementContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyImportStatementContext() *ImportStatementContext {
	var p = new(ImportStatementContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_importStatement
	return p
}

func InitEmptyImportStatementContext(p *ImportStatementContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_importStatement
}

func (*ImportStatementContext) IsImportStatementContext() {}

func NewImportStatementContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ImportStatementContext {
	var p = new(ImportStatementContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_importStatement

	return p
}

func (s *ImportStatementContext) GetParser() antlr.Parser { return s.parser }

func (s *ImportStatementContext) IMPORT() antlr.TerminalNode {
	return s.GetToken(ArcParserIMPORT, 0)
}

func (s *ImportStatementContext) AllImportItem() []IImportItemContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IImportItemContext); ok {
			len++
		}
	}

	tst := make([]IImportItemContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IImportItemContext); ok {
			tst[i] = t.(IImportItemContext)
			i++
		}
	}

	return tst
}

func (s *ImportStatementContext) ImportItem(i int) IImportItemContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IImportItemContext); ok {
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

	return t.(IImportItemContext)
}

func (s *ImportStatementContext) LPAREN() antlr.TerminalNode {
	return s.GetToken(ArcParserLPAREN, 0)
}

func (s *ImportStatementContext) RPAREN() antlr.TerminalNode {
	return s.GetToken(ArcParserRPAREN, 0)
}

func (s *ImportStatementContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ImportStatementContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ImportStatementContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterImportStatement(s)
	}
}

func (s *ImportStatementContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitImportStatement(s)
	}
}

func (s *ImportStatementContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitImportStatement(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) ImportStatement() (localctx IImportStatementContext) {
	localctx = NewImportStatementContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 4, ArcParserRULE_importStatement)
	var _la int

	p.SetState(184)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 3, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(173)
			p.Match(ArcParserIMPORT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(174)
			p.ImportItem()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(175)
			p.Match(ArcParserIMPORT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(176)
			p.Match(ArcParserLPAREN)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		p.SetState(180)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == ArcParserAUTHORITY || _la == ArcParserIDENTIFIER {
			{
				p.SetState(177)
				p.ImportItem()
			}

			p.SetState(182)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}
		{
			p.SetState(183)
			p.Match(ArcParserRPAREN)
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

// IImportItemContext is an interface to support dynamic dispatch.
type IImportItemContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	ImportPath() IImportPathContext
	AS() antlr.TerminalNode
	IDENTIFIER() antlr.TerminalNode

	// IsImportItemContext differentiates from other interfaces.
	IsImportItemContext()
}

type ImportItemContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyImportItemContext() *ImportItemContext {
	var p = new(ImportItemContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_importItem
	return p
}

func InitEmptyImportItemContext(p *ImportItemContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_importItem
}

func (*ImportItemContext) IsImportItemContext() {}

func NewImportItemContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ImportItemContext {
	var p = new(ImportItemContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_importItem

	return p
}

func (s *ImportItemContext) GetParser() antlr.Parser { return s.parser }

func (s *ImportItemContext) ImportPath() IImportPathContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IImportPathContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IImportPathContext)
}

func (s *ImportItemContext) AS() antlr.TerminalNode {
	return s.GetToken(ArcParserAS, 0)
}

func (s *ImportItemContext) IDENTIFIER() antlr.TerminalNode {
	return s.GetToken(ArcParserIDENTIFIER, 0)
}

func (s *ImportItemContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ImportItemContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ImportItemContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterImportItem(s)
	}
}

func (s *ImportItemContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitImportItem(s)
	}
}

func (s *ImportItemContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitImportItem(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) ImportItem() (localctx IImportItemContext) {
	localctx = NewImportItemContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 6, ArcParserRULE_importItem)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(186)
		p.ImportPath()
	}
	p.SetState(189)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ArcParserAS {
		{
			p.SetState(187)
			p.Match(ArcParserAS)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(188)
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

// IImportPathContext is an interface to support dynamic dispatch.
type IImportPathContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	ImportPathHead() IImportPathHeadContext
	AllDOT() []antlr.TerminalNode
	DOT(i int) antlr.TerminalNode
	AllIDENTIFIER() []antlr.TerminalNode
	IDENTIFIER(i int) antlr.TerminalNode

	// IsImportPathContext differentiates from other interfaces.
	IsImportPathContext()
}

type ImportPathContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyImportPathContext() *ImportPathContext {
	var p = new(ImportPathContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_importPath
	return p
}

func InitEmptyImportPathContext(p *ImportPathContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_importPath
}

func (*ImportPathContext) IsImportPathContext() {}

func NewImportPathContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ImportPathContext {
	var p = new(ImportPathContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_importPath

	return p
}

func (s *ImportPathContext) GetParser() antlr.Parser { return s.parser }

func (s *ImportPathContext) ImportPathHead() IImportPathHeadContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IImportPathHeadContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IImportPathHeadContext)
}

func (s *ImportPathContext) AllDOT() []antlr.TerminalNode {
	return s.GetTokens(ArcParserDOT)
}

func (s *ImportPathContext) DOT(i int) antlr.TerminalNode {
	return s.GetToken(ArcParserDOT, i)
}

func (s *ImportPathContext) AllIDENTIFIER() []antlr.TerminalNode {
	return s.GetTokens(ArcParserIDENTIFIER)
}

func (s *ImportPathContext) IDENTIFIER(i int) antlr.TerminalNode {
	return s.GetToken(ArcParserIDENTIFIER, i)
}

func (s *ImportPathContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ImportPathContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ImportPathContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterImportPath(s)
	}
}

func (s *ImportPathContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitImportPath(s)
	}
}

func (s *ImportPathContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitImportPath(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) ImportPath() (localctx IImportPathContext) {
	localctx = NewImportPathContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 8, ArcParserRULE_importPath)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(191)
		p.ImportPathHead()
	}
	p.SetState(196)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == ArcParserDOT {
		{
			p.SetState(192)
			p.Match(ArcParserDOT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(193)
			p.Match(ArcParserIDENTIFIER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

		p.SetState(198)
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

// IImportPathHeadContext is an interface to support dynamic dispatch.
type IImportPathHeadContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	IDENTIFIER() antlr.TerminalNode
	AUTHORITY() antlr.TerminalNode

	// IsImportPathHeadContext differentiates from other interfaces.
	IsImportPathHeadContext()
}

type ImportPathHeadContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyImportPathHeadContext() *ImportPathHeadContext {
	var p = new(ImportPathHeadContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_importPathHead
	return p
}

func InitEmptyImportPathHeadContext(p *ImportPathHeadContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_importPathHead
}

func (*ImportPathHeadContext) IsImportPathHeadContext() {}

func NewImportPathHeadContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ImportPathHeadContext {
	var p = new(ImportPathHeadContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_importPathHead

	return p
}

func (s *ImportPathHeadContext) GetParser() antlr.Parser { return s.parser }

func (s *ImportPathHeadContext) IDENTIFIER() antlr.TerminalNode {
	return s.GetToken(ArcParserIDENTIFIER, 0)
}

func (s *ImportPathHeadContext) AUTHORITY() antlr.TerminalNode {
	return s.GetToken(ArcParserAUTHORITY, 0)
}

func (s *ImportPathHeadContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ImportPathHeadContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ImportPathHeadContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterImportPathHead(s)
	}
}

func (s *ImportPathHeadContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitImportPathHead(s)
	}
}

func (s *ImportPathHeadContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitImportPathHead(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) ImportPathHead() (localctx IImportPathHeadContext) {
	localctx = NewImportPathHeadContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 10, ArcParserRULE_importPathHead)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(199)
		_la = p.GetTokenStream().LA(1)

		if !(_la == ArcParserAUTHORITY || _la == ArcParserIDENTIFIER) {
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

// IAuthorityBlockContext is an interface to support dynamic dispatch.
type IAuthorityBlockContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AUTHORITY() antlr.TerminalNode
	INTEGER_LITERAL() antlr.TerminalNode
	LPAREN() antlr.TerminalNode
	RPAREN() antlr.TerminalNode
	AllAuthorityEntry() []IAuthorityEntryContext
	AuthorityEntry(i int) IAuthorityEntryContext

	// IsAuthorityBlockContext differentiates from other interfaces.
	IsAuthorityBlockContext()
}

type AuthorityBlockContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyAuthorityBlockContext() *AuthorityBlockContext {
	var p = new(AuthorityBlockContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_authorityBlock
	return p
}

func InitEmptyAuthorityBlockContext(p *AuthorityBlockContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_authorityBlock
}

func (*AuthorityBlockContext) IsAuthorityBlockContext() {}

func NewAuthorityBlockContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *AuthorityBlockContext {
	var p = new(AuthorityBlockContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_authorityBlock

	return p
}

func (s *AuthorityBlockContext) GetParser() antlr.Parser { return s.parser }

func (s *AuthorityBlockContext) AUTHORITY() antlr.TerminalNode {
	return s.GetToken(ArcParserAUTHORITY, 0)
}

func (s *AuthorityBlockContext) INTEGER_LITERAL() antlr.TerminalNode {
	return s.GetToken(ArcParserINTEGER_LITERAL, 0)
}

func (s *AuthorityBlockContext) LPAREN() antlr.TerminalNode {
	return s.GetToken(ArcParserLPAREN, 0)
}

func (s *AuthorityBlockContext) RPAREN() antlr.TerminalNode {
	return s.GetToken(ArcParserRPAREN, 0)
}

func (s *AuthorityBlockContext) AllAuthorityEntry() []IAuthorityEntryContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IAuthorityEntryContext); ok {
			len++
		}
	}

	tst := make([]IAuthorityEntryContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IAuthorityEntryContext); ok {
			tst[i] = t.(IAuthorityEntryContext)
			i++
		}
	}

	return tst
}

func (s *AuthorityBlockContext) AuthorityEntry(i int) IAuthorityEntryContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IAuthorityEntryContext); ok {
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

	return t.(IAuthorityEntryContext)
}

func (s *AuthorityBlockContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *AuthorityBlockContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *AuthorityBlockContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterAuthorityBlock(s)
	}
}

func (s *AuthorityBlockContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitAuthorityBlock(s)
	}
}

func (s *AuthorityBlockContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitAuthorityBlock(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) AuthorityBlock() (localctx IAuthorityBlockContext) {
	localctx = NewAuthorityBlockContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 12, ArcParserRULE_authorityBlock)
	var _la int

	p.SetState(212)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 7, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(201)
			p.Match(ArcParserAUTHORITY)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(202)
			p.Match(ArcParserINTEGER_LITERAL)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(203)
			p.Match(ArcParserAUTHORITY)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(204)
			p.Match(ArcParserLPAREN)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		p.SetState(208)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == ArcParserINTEGER_LITERAL || _la == ArcParserIDENTIFIER {
			{
				p.SetState(205)
				p.AuthorityEntry()
			}

			p.SetState(210)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}
		{
			p.SetState(211)
			p.Match(ArcParserRPAREN)
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

// IAuthorityEntryContext is an interface to support dynamic dispatch.
type IAuthorityEntryContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	INTEGER_LITERAL() antlr.TerminalNode
	IDENTIFIER() antlr.TerminalNode

	// IsAuthorityEntryContext differentiates from other interfaces.
	IsAuthorityEntryContext()
}

type AuthorityEntryContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyAuthorityEntryContext() *AuthorityEntryContext {
	var p = new(AuthorityEntryContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_authorityEntry
	return p
}

func InitEmptyAuthorityEntryContext(p *AuthorityEntryContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_authorityEntry
}

func (*AuthorityEntryContext) IsAuthorityEntryContext() {}

func NewAuthorityEntryContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *AuthorityEntryContext {
	var p = new(AuthorityEntryContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_authorityEntry

	return p
}

func (s *AuthorityEntryContext) GetParser() antlr.Parser { return s.parser }

func (s *AuthorityEntryContext) INTEGER_LITERAL() antlr.TerminalNode {
	return s.GetToken(ArcParserINTEGER_LITERAL, 0)
}

func (s *AuthorityEntryContext) IDENTIFIER() antlr.TerminalNode {
	return s.GetToken(ArcParserIDENTIFIER, 0)
}

func (s *AuthorityEntryContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *AuthorityEntryContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *AuthorityEntryContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterAuthorityEntry(s)
	}
}

func (s *AuthorityEntryContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitAuthorityEntry(s)
	}
}

func (s *AuthorityEntryContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitAuthorityEntry(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) AuthorityEntry() (localctx IAuthorityEntryContext) {
	localctx = NewAuthorityEntryContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 14, ArcParserRULE_authorityEntry)
	p.SetState(217)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case ArcParserINTEGER_LITERAL:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(214)
			p.Match(ArcParserINTEGER_LITERAL)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case ArcParserIDENTIFIER:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(215)
			p.Match(ArcParserIDENTIFIER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(216)
			p.Match(ArcParserINTEGER_LITERAL)
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
	InputBlock() IInputBlockContext
	TriggerList() ITriggerListContext
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

func (s *FunctionDeclarationContext) InputBlock() IInputBlockContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IInputBlockContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IInputBlockContext)
}

func (s *FunctionDeclarationContext) TriggerList() ITriggerListContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ITriggerListContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ITriggerListContext)
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

func (s *FunctionDeclarationContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitFunctionDeclaration(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) FunctionDeclaration() (localctx IFunctionDeclarationContext) {
	localctx = NewFunctionDeclarationContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 16, ArcParserRULE_functionDeclaration)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(219)
		p.Match(ArcParserFUNC)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(220)
		p.Match(ArcParserIDENTIFIER)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(222)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ArcParserLBRACE {
		{
			p.SetState(221)
			p.InputBlock()
		}

	}
	{
		p.SetState(224)
		p.Match(ArcParserLPAREN)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(226)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ArcParserIDENTIFIER {
		{
			p.SetState(225)
			p.TriggerList()
		}

	}
	{
		p.SetState(228)
		p.Match(ArcParserRPAREN)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(230)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if (int64((_la-13)) & ^0x3f) == 0 && ((int64(1)<<(_la-13))&4504149383200765) != 0 {
		{
			p.SetState(229)
			p.OutputType()
		}

	}
	{
		p.SetState(232)
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

// ITriggerListContext is an interface to support dynamic dispatch.
type ITriggerListContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AllTrigger() []ITriggerContext
	Trigger(i int) ITriggerContext
	AllCOMMA() []antlr.TerminalNode
	COMMA(i int) antlr.TerminalNode

	// IsTriggerListContext differentiates from other interfaces.
	IsTriggerListContext()
}

type TriggerListContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyTriggerListContext() *TriggerListContext {
	var p = new(TriggerListContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_triggerList
	return p
}

func InitEmptyTriggerListContext(p *TriggerListContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_triggerList
}

func (*TriggerListContext) IsTriggerListContext() {}

func NewTriggerListContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *TriggerListContext {
	var p = new(TriggerListContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_triggerList

	return p
}

func (s *TriggerListContext) GetParser() antlr.Parser { return s.parser }

func (s *TriggerListContext) AllTrigger() []ITriggerContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(ITriggerContext); ok {
			len++
		}
	}

	tst := make([]ITriggerContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(ITriggerContext); ok {
			tst[i] = t.(ITriggerContext)
			i++
		}
	}

	return tst
}

func (s *TriggerListContext) Trigger(i int) ITriggerContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ITriggerContext); ok {
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

	return t.(ITriggerContext)
}

func (s *TriggerListContext) AllCOMMA() []antlr.TerminalNode {
	return s.GetTokens(ArcParserCOMMA)
}

func (s *TriggerListContext) COMMA(i int) antlr.TerminalNode {
	return s.GetToken(ArcParserCOMMA, i)
}

func (s *TriggerListContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *TriggerListContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *TriggerListContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterTriggerList(s)
	}
}

func (s *TriggerListContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitTriggerList(s)
	}
}

func (s *TriggerListContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitTriggerList(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) TriggerList() (localctx ITriggerListContext) {
	localctx = NewTriggerListContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 18, ArcParserRULE_triggerList)
	var _la int

	var _alt int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(234)
		p.Trigger()
	}
	p.SetState(239)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 12, p.GetParserRuleContext())
	if p.HasError() {
		goto errorExit
	}
	for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
		if _alt == 1 {
			{
				p.SetState(235)
				p.Match(ArcParserCOMMA)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}
			{
				p.SetState(236)
				p.Trigger()
			}

		}
		p.SetState(241)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 12, p.GetParserRuleContext())
		if p.HasError() {
			goto errorExit
		}
	}
	p.SetState(243)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ArcParserCOMMA {
		{
			p.SetState(242)
			p.Match(ArcParserCOMMA)
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

// ITriggerContext is an interface to support dynamic dispatch.
type ITriggerContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	IDENTIFIER() antlr.TerminalNode
	Type_() ITypeContext
	ASSIGN() antlr.TerminalNode
	Literal() ILiteralContext

	// IsTriggerContext differentiates from other interfaces.
	IsTriggerContext()
}

type TriggerContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyTriggerContext() *TriggerContext {
	var p = new(TriggerContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_trigger
	return p
}

func InitEmptyTriggerContext(p *TriggerContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_trigger
}

func (*TriggerContext) IsTriggerContext() {}

func NewTriggerContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *TriggerContext {
	var p = new(TriggerContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_trigger

	return p
}

func (s *TriggerContext) GetParser() antlr.Parser { return s.parser }

func (s *TriggerContext) IDENTIFIER() antlr.TerminalNode {
	return s.GetToken(ArcParserIDENTIFIER, 0)
}

func (s *TriggerContext) Type_() ITypeContext {
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

func (s *TriggerContext) ASSIGN() antlr.TerminalNode {
	return s.GetToken(ArcParserASSIGN, 0)
}

func (s *TriggerContext) Literal() ILiteralContext {
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

func (s *TriggerContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *TriggerContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *TriggerContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterTrigger(s)
	}
}

func (s *TriggerContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitTrigger(s)
	}
}

func (s *TriggerContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitTrigger(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) Trigger() (localctx ITriggerContext) {
	localctx = NewTriggerContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 20, ArcParserRULE_trigger)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(245)
		p.Match(ArcParserIDENTIFIER)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(246)
		p.Type_()
	}
	p.SetState(249)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ArcParserASSIGN {
		{
			p.SetState(247)
			p.Match(ArcParserASSIGN)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(248)
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

func (s *OutputTypeContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitOutputType(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) OutputType() (localctx IOutputTypeContext) {
	localctx = NewOutputTypeContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 22, ArcParserRULE_outputType)
	p.SetState(255)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case ArcParserCHAN, ArcParserI8, ArcParserI16, ArcParserI32, ArcParserI64, ArcParserU8, ArcParserU16, ArcParserU32, ArcParserU64, ArcParserF32, ArcParserF64, ArcParserSTR, ArcParserSERIES:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(251)
			p.Type_()
		}

	case ArcParserIDENTIFIER:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(252)
			p.Match(ArcParserIDENTIFIER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(253)
			p.Type_()
		}

	case ArcParserLPAREN:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(254)
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

func (s *MultiOutputBlockContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitMultiOutputBlock(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) MultiOutputBlock() (localctx IMultiOutputBlockContext) {
	localctx = NewMultiOutputBlockContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 24, ArcParserRULE_multiOutputBlock)
	var _la int

	var _alt int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(257)
		p.Match(ArcParserLPAREN)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(258)
		p.NamedOutput()
	}
	p.SetState(263)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 16, p.GetParserRuleContext())
	if p.HasError() {
		goto errorExit
	}
	for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
		if _alt == 1 {
			{
				p.SetState(259)
				p.Match(ArcParserCOMMA)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}
			{
				p.SetState(260)
				p.NamedOutput()
			}

		}
		p.SetState(265)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 16, p.GetParserRuleContext())
		if p.HasError() {
			goto errorExit
		}
	}
	p.SetState(267)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ArcParserCOMMA {
		{
			p.SetState(266)
			p.Match(ArcParserCOMMA)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	}
	{
		p.SetState(269)
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

func (s *NamedOutputContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitNamedOutput(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) NamedOutput() (localctx INamedOutputContext) {
	localctx = NewNamedOutputContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 26, ArcParserRULE_namedOutput)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(271)
		p.Match(ArcParserIDENTIFIER)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(272)
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

// IInputBlockContext is an interface to support dynamic dispatch.
type IInputBlockContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	LBRACE() antlr.TerminalNode
	RBRACE() antlr.TerminalNode
	InputList() IInputListContext

	// IsInputBlockContext differentiates from other interfaces.
	IsInputBlockContext()
}

type InputBlockContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyInputBlockContext() *InputBlockContext {
	var p = new(InputBlockContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_inputBlock
	return p
}

func InitEmptyInputBlockContext(p *InputBlockContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_inputBlock
}

func (*InputBlockContext) IsInputBlockContext() {}

func NewInputBlockContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *InputBlockContext {
	var p = new(InputBlockContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_inputBlock

	return p
}

func (s *InputBlockContext) GetParser() antlr.Parser { return s.parser }

func (s *InputBlockContext) LBRACE() antlr.TerminalNode {
	return s.GetToken(ArcParserLBRACE, 0)
}

func (s *InputBlockContext) RBRACE() antlr.TerminalNode {
	return s.GetToken(ArcParserRBRACE, 0)
}

func (s *InputBlockContext) InputList() IInputListContext {
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

func (s *InputBlockContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *InputBlockContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *InputBlockContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterInputBlock(s)
	}
}

func (s *InputBlockContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitInputBlock(s)
	}
}

func (s *InputBlockContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitInputBlock(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) InputBlock() (localctx IInputBlockContext) {
	localctx = NewInputBlockContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 28, ArcParserRULE_inputBlock)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(274)
		p.Match(ArcParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(276)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ArcParserIDENTIFIER {
		{
			p.SetState(275)
			p.InputList()
		}

	}
	{
		p.SetState(278)
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

func (s *InputListContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitInputList(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) InputList() (localctx IInputListContext) {
	localctx = NewInputListContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 30, ArcParserRULE_inputList)
	var _la int

	var _alt int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(280)
		p.Input()
	}
	p.SetState(285)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 19, p.GetParserRuleContext())
	if p.HasError() {
		goto errorExit
	}
	for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
		if _alt == 1 {
			{
				p.SetState(281)
				p.Match(ArcParserCOMMA)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}
			{
				p.SetState(282)
				p.Input()
			}

		}
		p.SetState(287)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 19, p.GetParserRuleContext())
		if p.HasError() {
			goto errorExit
		}
	}
	p.SetState(289)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ArcParserCOMMA {
		{
			p.SetState(288)
			p.Match(ArcParserCOMMA)
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

func (s *InputContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitInput(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) Input() (localctx IInputContext) {
	localctx = NewInputContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 32, ArcParserRULE_input)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(291)
		p.Match(ArcParserIDENTIFIER)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(292)
		p.Type_()
	}
	p.SetState(295)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ArcParserASSIGN {
		{
			p.SetState(293)
			p.Match(ArcParserASSIGN)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(294)
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

// ISequenceDeclarationContext is an interface to support dynamic dispatch.
type ISequenceDeclarationContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	SEQUENCE() antlr.TerminalNode
	LBRACE() antlr.TerminalNode
	RBRACE() antlr.TerminalNode
	IDENTIFIER() antlr.TerminalNode
	AllSequenceItem() []ISequenceItemContext
	SequenceItem(i int) ISequenceItemContext
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

func (s *SequenceDeclarationContext) LBRACE() antlr.TerminalNode {
	return s.GetToken(ArcParserLBRACE, 0)
}

func (s *SequenceDeclarationContext) RBRACE() antlr.TerminalNode {
	return s.GetToken(ArcParserRBRACE, 0)
}

func (s *SequenceDeclarationContext) IDENTIFIER() antlr.TerminalNode {
	return s.GetToken(ArcParserIDENTIFIER, 0)
}

func (s *SequenceDeclarationContext) AllSequenceItem() []ISequenceItemContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(ISequenceItemContext); ok {
			len++
		}
	}

	tst := make([]ISequenceItemContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(ISequenceItemContext); ok {
			tst[i] = t.(ISequenceItemContext)
			i++
		}
	}

	return tst
}

func (s *SequenceDeclarationContext) SequenceItem(i int) ISequenceItemContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ISequenceItemContext); ok {
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

	return t.(ISequenceItemContext)
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

func (s *SequenceDeclarationContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitSequenceDeclaration(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) SequenceDeclaration() (localctx ISequenceDeclarationContext) {
	localctx = NewSequenceDeclarationContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 34, ArcParserRULE_sequenceDeclaration)
	var _la int

	var _alt int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(297)
		p.Match(ArcParserSEQUENCE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(299)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ArcParserIDENTIFIER {
		{
			p.SetState(298)
			p.Match(ArcParserIDENTIFIER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	}
	{
		p.SetState(301)
		p.Match(ArcParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(315)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if (int64((_la-10)) & ^0x3f) == 0 && ((int64(1)<<(_la-10))&69900352492797951) != 0 {
		{
			p.SetState(302)
			p.SequenceItem()
		}
		p.SetState(309)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 24, p.GetParserRuleContext())
		if p.HasError() {
			goto errorExit
		}
		for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
			if _alt == 1 {
				p.SetState(304)
				p.GetErrorHandler().Sync(p)
				if p.HasError() {
					goto errorExit
				}
				_la = p.GetTokenStream().LA(1)

				if _la == ArcParserCOMMA {
					{
						p.SetState(303)
						p.Match(ArcParserCOMMA)
						if p.HasError() {
							// Recognition error - abort rule
							goto errorExit
						}
					}

				}
				{
					p.SetState(306)
					p.SequenceItem()
				}

			}
			p.SetState(311)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 24, p.GetParserRuleContext())
			if p.HasError() {
				goto errorExit
			}
		}
		p.SetState(313)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		if _la == ArcParserCOMMA {
			{
				p.SetState(312)
				p.Match(ArcParserCOMMA)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}

		}

	}
	{
		p.SetState(317)
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

// ISequenceItemContext is an interface to support dynamic dispatch.
type ISequenceItemContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	StageDeclaration() IStageDeclarationContext
	SequenceDeclaration() ISequenceDeclarationContext
	FlowStatement() IFlowStatementContext
	SingleInvocation() ISingleInvocationContext

	// IsSequenceItemContext differentiates from other interfaces.
	IsSequenceItemContext()
}

type SequenceItemContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptySequenceItemContext() *SequenceItemContext {
	var p = new(SequenceItemContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_sequenceItem
	return p
}

func InitEmptySequenceItemContext(p *SequenceItemContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_sequenceItem
}

func (*SequenceItemContext) IsSequenceItemContext() {}

func NewSequenceItemContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *SequenceItemContext {
	var p = new(SequenceItemContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_sequenceItem

	return p
}

func (s *SequenceItemContext) GetParser() antlr.Parser { return s.parser }

func (s *SequenceItemContext) StageDeclaration() IStageDeclarationContext {
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

func (s *SequenceItemContext) SequenceDeclaration() ISequenceDeclarationContext {
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

func (s *SequenceItemContext) FlowStatement() IFlowStatementContext {
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

func (s *SequenceItemContext) SingleInvocation() ISingleInvocationContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ISingleInvocationContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ISingleInvocationContext)
}

func (s *SequenceItemContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *SequenceItemContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *SequenceItemContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterSequenceItem(s)
	}
}

func (s *SequenceItemContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitSequenceItem(s)
	}
}

func (s *SequenceItemContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitSequenceItem(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) SequenceItem() (localctx ISequenceItemContext) {
	localctx = NewSequenceItemContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 36, ArcParserRULE_sequenceItem)
	p.SetState(323)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 27, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(319)
			p.StageDeclaration()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(320)
			p.SequenceDeclaration()
		}

	case 3:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(321)
			p.FlowStatement()
		}

	case 4:
		p.EnterOuterAlt(localctx, 4)
		{
			p.SetState(322)
			p.SingleInvocation()
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
	StageBody() IStageBodyContext
	IDENTIFIER() antlr.TerminalNode

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

func (s *StageDeclarationContext) IDENTIFIER() antlr.TerminalNode {
	return s.GetToken(ArcParserIDENTIFIER, 0)
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

func (s *StageDeclarationContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitStageDeclaration(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) StageDeclaration() (localctx IStageDeclarationContext) {
	localctx = NewStageDeclarationContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 38, ArcParserRULE_stageDeclaration)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(325)
		p.Match(ArcParserSTAGE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(327)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ArcParserIDENTIFIER {
		{
			p.SetState(326)
			p.Match(ArcParserIDENTIFIER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	}
	{
		p.SetState(329)
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
	AllCOMMA() []antlr.TerminalNode
	COMMA(i int) antlr.TerminalNode

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

func (s *StageBodyContext) AllCOMMA() []antlr.TerminalNode {
	return s.GetTokens(ArcParserCOMMA)
}

func (s *StageBodyContext) COMMA(i int) antlr.TerminalNode {
	return s.GetToken(ArcParserCOMMA, i)
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

func (s *StageBodyContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitStageBody(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) StageBody() (localctx IStageBodyContext) {
	localctx = NewStageBodyContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 40, ArcParserRULE_stageBody)
	var _la int

	var _alt int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(331)
		p.Match(ArcParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(345)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if (int64((_la-10)) & ^0x3f) == 0 && ((int64(1)<<(_la-10))&69900352492797951) != 0 {
		{
			p.SetState(332)
			p.StageItem()
		}
		p.SetState(339)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 30, p.GetParserRuleContext())
		if p.HasError() {
			goto errorExit
		}
		for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
			if _alt == 1 {
				p.SetState(334)
				p.GetErrorHandler().Sync(p)
				if p.HasError() {
					goto errorExit
				}
				_la = p.GetTokenStream().LA(1)

				if _la == ArcParserCOMMA {
					{
						p.SetState(333)
						p.Match(ArcParserCOMMA)
						if p.HasError() {
							// Recognition error - abort rule
							goto errorExit
						}
					}

				}
				{
					p.SetState(336)
					p.StageItem()
				}

			}
			p.SetState(341)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 30, p.GetParserRuleContext())
			if p.HasError() {
				goto errorExit
			}
		}
		p.SetState(343)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		if _la == ArcParserCOMMA {
			{
				p.SetState(342)
				p.Match(ArcParserCOMMA)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}

		}

	}
	{
		p.SetState(347)
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
	FlowStatement() IFlowStatementContext
	SingleInvocation() ISingleInvocationContext
	SequenceDeclaration() ISequenceDeclarationContext

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

func (s *StageItemContext) FlowStatement() IFlowStatementContext {
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

func (s *StageItemContext) SingleInvocation() ISingleInvocationContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ISingleInvocationContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ISingleInvocationContext)
}

func (s *StageItemContext) SequenceDeclaration() ISequenceDeclarationContext {
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

func (s *StageItemContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitStageItem(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) StageItem() (localctx IStageItemContext) {
	localctx = NewStageItemContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 42, ArcParserRULE_stageItem)
	p.SetState(352)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 33, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(349)
			p.FlowStatement()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(350)
			p.SingleInvocation()
		}

	case 3:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(351)
			p.SequenceDeclaration()
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

// ISingleInvocationContext is an interface to support dynamic dispatch.
type ISingleInvocationContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	Function() IFunctionContext
	Expression() IExpressionContext

	// IsSingleInvocationContext differentiates from other interfaces.
	IsSingleInvocationContext()
}

type SingleInvocationContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptySingleInvocationContext() *SingleInvocationContext {
	var p = new(SingleInvocationContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_singleInvocation
	return p
}

func InitEmptySingleInvocationContext(p *SingleInvocationContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_singleInvocation
}

func (*SingleInvocationContext) IsSingleInvocationContext() {}

func NewSingleInvocationContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *SingleInvocationContext {
	var p = new(SingleInvocationContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_singleInvocation

	return p
}

func (s *SingleInvocationContext) GetParser() antlr.Parser { return s.parser }

func (s *SingleInvocationContext) Function() IFunctionContext {
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

func (s *SingleInvocationContext) Expression() IExpressionContext {
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

func (s *SingleInvocationContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *SingleInvocationContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *SingleInvocationContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterSingleInvocation(s)
	}
}

func (s *SingleInvocationContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitSingleInvocation(s)
	}
}

func (s *SingleInvocationContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitSingleInvocation(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) SingleInvocation() (localctx ISingleInvocationContext) {
	localctx = NewSingleInvocationContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 44, ArcParserRULE_singleInvocation)
	p.SetState(356)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 34, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(354)
			p.Function()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(355)
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

// IGlobalConstantContext is an interface to support dynamic dispatch.
type IGlobalConstantContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	IDENTIFIER() antlr.TerminalNode
	DECLARE() antlr.TerminalNode
	Literal() ILiteralContext
	Type_() ITypeContext

	// IsGlobalConstantContext differentiates from other interfaces.
	IsGlobalConstantContext()
}

type GlobalConstantContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyGlobalConstantContext() *GlobalConstantContext {
	var p = new(GlobalConstantContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_globalConstant
	return p
}

func InitEmptyGlobalConstantContext(p *GlobalConstantContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_globalConstant
}

func (*GlobalConstantContext) IsGlobalConstantContext() {}

func NewGlobalConstantContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *GlobalConstantContext {
	var p = new(GlobalConstantContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_globalConstant

	return p
}

func (s *GlobalConstantContext) GetParser() antlr.Parser { return s.parser }

func (s *GlobalConstantContext) IDENTIFIER() antlr.TerminalNode {
	return s.GetToken(ArcParserIDENTIFIER, 0)
}

func (s *GlobalConstantContext) DECLARE() antlr.TerminalNode {
	return s.GetToken(ArcParserDECLARE, 0)
}

func (s *GlobalConstantContext) Literal() ILiteralContext {
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

func (s *GlobalConstantContext) Type_() ITypeContext {
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

func (s *GlobalConstantContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *GlobalConstantContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *GlobalConstantContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterGlobalConstant(s)
	}
}

func (s *GlobalConstantContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitGlobalConstant(s)
	}
}

func (s *GlobalConstantContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitGlobalConstant(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) GlobalConstant() (localctx IGlobalConstantContext) {
	localctx = NewGlobalConstantContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 46, ArcParserRULE_globalConstant)
	p.SetState(366)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 35, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(358)
			p.Match(ArcParserIDENTIFIER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(359)
			p.Match(ArcParserDECLARE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(360)
			p.Literal()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(361)
			p.Match(ArcParserIDENTIFIER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(362)
			p.Type_()
		}
		{
			p.SetState(363)
			p.Match(ArcParserDECLARE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(364)
			p.Literal()
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
	AllFlowOperator() []IFlowOperatorContext
	FlowOperator(i int) IFlowOperatorContext

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

func (s *FlowStatementContext) AllFlowOperator() []IFlowOperatorContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IFlowOperatorContext); ok {
			len++
		}
	}

	tst := make([]IFlowOperatorContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IFlowOperatorContext); ok {
			tst[i] = t.(IFlowOperatorContext)
			i++
		}
	}

	return tst
}

func (s *FlowStatementContext) FlowOperator(i int) IFlowOperatorContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IFlowOperatorContext); ok {
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

	return t.(IFlowOperatorContext)
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

func (s *FlowStatementContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitFlowStatement(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) FlowStatement() (localctx IFlowStatementContext) {
	localctx = NewFlowStatementContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 48, ArcParserRULE_flowStatement)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(370)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case ArcParserLBRACE:
		{
			p.SetState(368)
			p.RoutingTable()
		}

	case ArcParserSEQUENCE, ArcParserSTAGE, ArcParserNEXT, ArcParserCHAN, ArcParserAUTHORITY, ArcParserI8, ArcParserI16, ArcParserI32, ArcParserI64, ArcParserU8, ArcParserU16, ArcParserU32, ArcParserU64, ArcParserF32, ArcParserF64, ArcParserSTR, ArcParserSERIES, ArcParserMINUS, ArcParserNOT, ArcParserLPAREN, ArcParserLBRACKET, ArcParserINTEGER_LITERAL, ArcParserFLOAT_LITERAL, ArcParserSTR_LITERAL_MULTI, ArcParserSTR_LITERAL, ArcParserIDENTIFIER:
		{
			p.SetState(369)
			p.FlowNode()
		}

	default:
		p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
		goto errorExit
	}
	p.SetState(377)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for ok := true; ok; ok = _la == ArcParserARROW || _la == ArcParserTRANSITION {
		{
			p.SetState(372)
			p.FlowOperator()
		}
		p.SetState(375)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}

		switch p.GetTokenStream().LA(1) {
		case ArcParserLBRACE:
			{
				p.SetState(373)
				p.RoutingTable()
			}

		case ArcParserSEQUENCE, ArcParserSTAGE, ArcParserNEXT, ArcParserCHAN, ArcParserAUTHORITY, ArcParserI8, ArcParserI16, ArcParserI32, ArcParserI64, ArcParserU8, ArcParserU16, ArcParserU32, ArcParserU64, ArcParserF32, ArcParserF64, ArcParserSTR, ArcParserSERIES, ArcParserMINUS, ArcParserNOT, ArcParserLPAREN, ArcParserLBRACKET, ArcParserINTEGER_LITERAL, ArcParserFLOAT_LITERAL, ArcParserSTR_LITERAL_MULTI, ArcParserSTR_LITERAL, ArcParserIDENTIFIER:
			{
				p.SetState(374)
				p.FlowNode()
			}

		default:
			p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
			goto errorExit
		}

		p.SetState(379)
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

// IFlowOperatorContext is an interface to support dynamic dispatch.
type IFlowOperatorContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	ARROW() antlr.TerminalNode
	TRANSITION() antlr.TerminalNode

	// IsFlowOperatorContext differentiates from other interfaces.
	IsFlowOperatorContext()
}

type FlowOperatorContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyFlowOperatorContext() *FlowOperatorContext {
	var p = new(FlowOperatorContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_flowOperator
	return p
}

func InitEmptyFlowOperatorContext(p *FlowOperatorContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_flowOperator
}

func (*FlowOperatorContext) IsFlowOperatorContext() {}

func NewFlowOperatorContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *FlowOperatorContext {
	var p = new(FlowOperatorContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_flowOperator

	return p
}

func (s *FlowOperatorContext) GetParser() antlr.Parser { return s.parser }

func (s *FlowOperatorContext) ARROW() antlr.TerminalNode {
	return s.GetToken(ArcParserARROW, 0)
}

func (s *FlowOperatorContext) TRANSITION() antlr.TerminalNode {
	return s.GetToken(ArcParserTRANSITION, 0)
}

func (s *FlowOperatorContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *FlowOperatorContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *FlowOperatorContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterFlowOperator(s)
	}
}

func (s *FlowOperatorContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitFlowOperator(s)
	}
}

func (s *FlowOperatorContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitFlowOperator(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) FlowOperator() (localctx IFlowOperatorContext) {
	localctx = NewFlowOperatorContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 50, ArcParserRULE_flowOperator)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(381)
		_la = p.GetTokenStream().LA(1)

		if !(_la == ArcParserARROW || _la == ArcParserTRANSITION) {
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

func (s *RoutingTableContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitRoutingTable(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) RoutingTable() (localctx IRoutingTableContext) {
	localctx = NewRoutingTableContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 52, ArcParserRULE_routingTable)
	var _la int

	var _alt int

	p.EnterOuterAlt(localctx, 1)
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
		p.RoutingEntry()
	}
	p.SetState(389)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 39, p.GetParserRuleContext())
	if p.HasError() {
		goto errorExit
	}
	for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
		if _alt == 1 {
			{
				p.SetState(385)
				p.Match(ArcParserCOMMA)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}
			{
				p.SetState(386)
				p.RoutingEntry()
			}

		}
		p.SetState(391)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 39, p.GetParserRuleContext())
		if p.HasError() {
			goto errorExit
		}
	}
	p.SetState(393)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ArcParserCOMMA {
		{
			p.SetState(392)
			p.Match(ArcParserCOMMA)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	}
	{
		p.SetState(395)
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

func (s *RoutingEntryContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitRoutingEntry(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) RoutingEntry() (localctx IRoutingEntryContext) {
	localctx = NewRoutingEntryContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 54, ArcParserRULE_routingEntry)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(397)
		p.Match(ArcParserIDENTIFIER)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(398)
		p.Match(ArcParserCOLON)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(399)
		p.FlowNode()
	}
	p.SetState(404)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == ArcParserARROW {
		{
			p.SetState(400)
			p.Match(ArcParserARROW)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(401)
			p.FlowNode()
		}

		p.SetState(406)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	p.SetState(409)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ArcParserCOLON {
		{
			p.SetState(407)
			p.Match(ArcParserCOLON)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(408)
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
	Identifier() IIdentifierContext
	Function() IFunctionContext
	Expression() IExpressionContext
	StageDeclaration() IStageDeclarationContext
	SequenceDeclaration() ISequenceDeclarationContext
	NEXT() antlr.TerminalNode

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

func (s *FlowNodeContext) Identifier() IIdentifierContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IIdentifierContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IIdentifierContext)
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

func (s *FlowNodeContext) StageDeclaration() IStageDeclarationContext {
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

func (s *FlowNodeContext) SequenceDeclaration() ISequenceDeclarationContext {
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

func (s *FlowNodeContext) NEXT() antlr.TerminalNode {
	return s.GetToken(ArcParserNEXT, 0)
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

func (s *FlowNodeContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitFlowNode(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) FlowNode() (localctx IFlowNodeContext) {
	localctx = NewFlowNodeContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 56, ArcParserRULE_flowNode)
	p.SetState(417)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 43, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(411)
			p.Identifier()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(412)
			p.Function()
		}

	case 3:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(413)
			p.Expression()
		}

	case 4:
		p.EnterOuterAlt(localctx, 4)
		{
			p.SetState(414)
			p.StageDeclaration()
		}

	case 5:
		p.EnterOuterAlt(localctx, 5)
		{
			p.SetState(415)
			p.SequenceDeclaration()
		}

	case 6:
		p.EnterOuterAlt(localctx, 6)
		{
			p.SetState(416)
			p.Match(ArcParserNEXT)
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

// IIdentifierContext is an interface to support dynamic dispatch.
type IIdentifierContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	IDENTIFIER() antlr.TerminalNode

	// IsIdentifierContext differentiates from other interfaces.
	IsIdentifierContext()
}

type IdentifierContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyIdentifierContext() *IdentifierContext {
	var p = new(IdentifierContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_identifier
	return p
}

func InitEmptyIdentifierContext(p *IdentifierContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_identifier
}

func (*IdentifierContext) IsIdentifierContext() {}

func NewIdentifierContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *IdentifierContext {
	var p = new(IdentifierContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_identifier

	return p
}

func (s *IdentifierContext) GetParser() antlr.Parser { return s.parser }

func (s *IdentifierContext) IDENTIFIER() antlr.TerminalNode {
	return s.GetToken(ArcParserIDENTIFIER, 0)
}

func (s *IdentifierContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *IdentifierContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *IdentifierContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterIdentifier(s)
	}
}

func (s *IdentifierContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitIdentifier(s)
	}
}

func (s *IdentifierContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitIdentifier(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) Identifier() (localctx IIdentifierContext) {
	localctx = NewIdentifierContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 58, ArcParserRULE_identifier)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(419)
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
	QualifiedIdentifier() IQualifiedIdentifierContext
	InputValues() IInputValuesContext
	IDENTIFIER() antlr.TerminalNode

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

func (s *FunctionContext) QualifiedIdentifier() IQualifiedIdentifierContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IQualifiedIdentifierContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IQualifiedIdentifierContext)
}

func (s *FunctionContext) InputValues() IInputValuesContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IInputValuesContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IInputValuesContext)
}

func (s *FunctionContext) IDENTIFIER() antlr.TerminalNode {
	return s.GetToken(ArcParserIDENTIFIER, 0)
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

func (s *FunctionContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitFunction(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) Function() (localctx IFunctionContext) {
	localctx = NewFunctionContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 60, ArcParserRULE_function)
	p.SetState(426)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 44, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(421)
			p.QualifiedIdentifier()
		}
		{
			p.SetState(422)
			p.InputValues()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(424)
			p.Match(ArcParserIDENTIFIER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(425)
			p.InputValues()
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

// IQualifiedIdentifierContext is an interface to support dynamic dispatch.
type IQualifiedIdentifierContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AllIDENTIFIER() []antlr.TerminalNode
	IDENTIFIER(i int) antlr.TerminalNode
	DOT() antlr.TerminalNode
	FOR() antlr.TerminalNode
	AUTHORITY() antlr.TerminalNode

	// IsQualifiedIdentifierContext differentiates from other interfaces.
	IsQualifiedIdentifierContext()
}

type QualifiedIdentifierContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyQualifiedIdentifierContext() *QualifiedIdentifierContext {
	var p = new(QualifiedIdentifierContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_qualifiedIdentifier
	return p
}

func InitEmptyQualifiedIdentifierContext(p *QualifiedIdentifierContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_qualifiedIdentifier
}

func (*QualifiedIdentifierContext) IsQualifiedIdentifierContext() {}

func NewQualifiedIdentifierContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *QualifiedIdentifierContext {
	var p = new(QualifiedIdentifierContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_qualifiedIdentifier

	return p
}

func (s *QualifiedIdentifierContext) GetParser() antlr.Parser { return s.parser }

func (s *QualifiedIdentifierContext) AllIDENTIFIER() []antlr.TerminalNode {
	return s.GetTokens(ArcParserIDENTIFIER)
}

func (s *QualifiedIdentifierContext) IDENTIFIER(i int) antlr.TerminalNode {
	return s.GetToken(ArcParserIDENTIFIER, i)
}

func (s *QualifiedIdentifierContext) DOT() antlr.TerminalNode {
	return s.GetToken(ArcParserDOT, 0)
}

func (s *QualifiedIdentifierContext) FOR() antlr.TerminalNode {
	return s.GetToken(ArcParserFOR, 0)
}

func (s *QualifiedIdentifierContext) AUTHORITY() antlr.TerminalNode {
	return s.GetToken(ArcParserAUTHORITY, 0)
}

func (s *QualifiedIdentifierContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *QualifiedIdentifierContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *QualifiedIdentifierContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterQualifiedIdentifier(s)
	}
}

func (s *QualifiedIdentifierContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitQualifiedIdentifier(s)
	}
}

func (s *QualifiedIdentifierContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitQualifiedIdentifier(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) QualifiedIdentifier() (localctx IQualifiedIdentifierContext) {
	localctx = NewQualifiedIdentifierContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 62, ArcParserRULE_qualifiedIdentifier)
	p.SetState(437)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 45, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(428)
			p.Match(ArcParserIDENTIFIER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(429)
			p.Match(ArcParserDOT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(430)
			p.Match(ArcParserIDENTIFIER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(431)
			p.Match(ArcParserIDENTIFIER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(432)
			p.Match(ArcParserDOT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(433)
			p.Match(ArcParserFOR)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 3:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(434)
			p.Match(ArcParserAUTHORITY)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(435)
			p.Match(ArcParserDOT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(436)
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

// IInputValuesContext is an interface to support dynamic dispatch.
type IInputValuesContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	LBRACE() antlr.TerminalNode
	RBRACE() antlr.TerminalNode
	NamedInputValues() INamedInputValuesContext
	AnonymousInputValues() IAnonymousInputValuesContext

	// IsInputValuesContext differentiates from other interfaces.
	IsInputValuesContext()
}

type InputValuesContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyInputValuesContext() *InputValuesContext {
	var p = new(InputValuesContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_inputValues
	return p
}

func InitEmptyInputValuesContext(p *InputValuesContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_inputValues
}

func (*InputValuesContext) IsInputValuesContext() {}

func NewInputValuesContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *InputValuesContext {
	var p = new(InputValuesContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_inputValues

	return p
}

func (s *InputValuesContext) GetParser() antlr.Parser { return s.parser }

func (s *InputValuesContext) LBRACE() antlr.TerminalNode {
	return s.GetToken(ArcParserLBRACE, 0)
}

func (s *InputValuesContext) RBRACE() antlr.TerminalNode {
	return s.GetToken(ArcParserRBRACE, 0)
}

func (s *InputValuesContext) NamedInputValues() INamedInputValuesContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(INamedInputValuesContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(INamedInputValuesContext)
}

func (s *InputValuesContext) AnonymousInputValues() IAnonymousInputValuesContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IAnonymousInputValuesContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IAnonymousInputValuesContext)
}

func (s *InputValuesContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *InputValuesContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *InputValuesContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterInputValues(s)
	}
}

func (s *InputValuesContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitInputValues(s)
	}
}

func (s *InputValuesContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitInputValues(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) InputValues() (localctx IInputValuesContext) {
	localctx = NewInputValuesContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 64, ArcParserRULE_inputValues)
	p.SetState(449)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 46, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(439)
			p.Match(ArcParserLBRACE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(440)
			p.Match(ArcParserRBRACE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(441)
			p.Match(ArcParserLBRACE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(442)
			p.NamedInputValues()
		}
		{
			p.SetState(443)
			p.Match(ArcParserRBRACE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 3:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(445)
			p.Match(ArcParserLBRACE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(446)
			p.AnonymousInputValues()
		}
		{
			p.SetState(447)
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

// INamedInputValuesContext is an interface to support dynamic dispatch.
type INamedInputValuesContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AllNamedInputValue() []INamedInputValueContext
	NamedInputValue(i int) INamedInputValueContext
	AllCOMMA() []antlr.TerminalNode
	COMMA(i int) antlr.TerminalNode

	// IsNamedInputValuesContext differentiates from other interfaces.
	IsNamedInputValuesContext()
}

type NamedInputValuesContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyNamedInputValuesContext() *NamedInputValuesContext {
	var p = new(NamedInputValuesContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_namedInputValues
	return p
}

func InitEmptyNamedInputValuesContext(p *NamedInputValuesContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_namedInputValues
}

func (*NamedInputValuesContext) IsNamedInputValuesContext() {}

func NewNamedInputValuesContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *NamedInputValuesContext {
	var p = new(NamedInputValuesContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_namedInputValues

	return p
}

func (s *NamedInputValuesContext) GetParser() antlr.Parser { return s.parser }

func (s *NamedInputValuesContext) AllNamedInputValue() []INamedInputValueContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(INamedInputValueContext); ok {
			len++
		}
	}

	tst := make([]INamedInputValueContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(INamedInputValueContext); ok {
			tst[i] = t.(INamedInputValueContext)
			i++
		}
	}

	return tst
}

func (s *NamedInputValuesContext) NamedInputValue(i int) INamedInputValueContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(INamedInputValueContext); ok {
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

	return t.(INamedInputValueContext)
}

func (s *NamedInputValuesContext) AllCOMMA() []antlr.TerminalNode {
	return s.GetTokens(ArcParserCOMMA)
}

func (s *NamedInputValuesContext) COMMA(i int) antlr.TerminalNode {
	return s.GetToken(ArcParserCOMMA, i)
}

func (s *NamedInputValuesContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *NamedInputValuesContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *NamedInputValuesContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterNamedInputValues(s)
	}
}

func (s *NamedInputValuesContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitNamedInputValues(s)
	}
}

func (s *NamedInputValuesContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitNamedInputValues(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) NamedInputValues() (localctx INamedInputValuesContext) {
	localctx = NewNamedInputValuesContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 66, ArcParserRULE_namedInputValues)
	var _la int

	var _alt int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(451)
		p.NamedInputValue()
	}
	p.SetState(456)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 47, p.GetParserRuleContext())
	if p.HasError() {
		goto errorExit
	}
	for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
		if _alt == 1 {
			{
				p.SetState(452)
				p.Match(ArcParserCOMMA)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}
			{
				p.SetState(453)
				p.NamedInputValue()
			}

		}
		p.SetState(458)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 47, p.GetParserRuleContext())
		if p.HasError() {
			goto errorExit
		}
	}
	p.SetState(460)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ArcParserCOMMA {
		{
			p.SetState(459)
			p.Match(ArcParserCOMMA)
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

// INamedInputValueContext is an interface to support dynamic dispatch.
type INamedInputValueContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	IDENTIFIER() antlr.TerminalNode
	ASSIGN() antlr.TerminalNode
	Expression() IExpressionContext

	// IsNamedInputValueContext differentiates from other interfaces.
	IsNamedInputValueContext()
}

type NamedInputValueContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyNamedInputValueContext() *NamedInputValueContext {
	var p = new(NamedInputValueContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_namedInputValue
	return p
}

func InitEmptyNamedInputValueContext(p *NamedInputValueContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_namedInputValue
}

func (*NamedInputValueContext) IsNamedInputValueContext() {}

func NewNamedInputValueContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *NamedInputValueContext {
	var p = new(NamedInputValueContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_namedInputValue

	return p
}

func (s *NamedInputValueContext) GetParser() antlr.Parser { return s.parser }

func (s *NamedInputValueContext) IDENTIFIER() antlr.TerminalNode {
	return s.GetToken(ArcParserIDENTIFIER, 0)
}

func (s *NamedInputValueContext) ASSIGN() antlr.TerminalNode {
	return s.GetToken(ArcParserASSIGN, 0)
}

func (s *NamedInputValueContext) Expression() IExpressionContext {
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

func (s *NamedInputValueContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *NamedInputValueContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *NamedInputValueContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterNamedInputValue(s)
	}
}

func (s *NamedInputValueContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitNamedInputValue(s)
	}
}

func (s *NamedInputValueContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitNamedInputValue(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) NamedInputValue() (localctx INamedInputValueContext) {
	localctx = NewNamedInputValueContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 68, ArcParserRULE_namedInputValue)
	p.EnterOuterAlt(localctx, 1)
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
		p.Match(ArcParserASSIGN)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(464)
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

// IAnonymousInputValuesContext is an interface to support dynamic dispatch.
type IAnonymousInputValuesContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AllExpression() []IExpressionContext
	Expression(i int) IExpressionContext
	AllCOMMA() []antlr.TerminalNode
	COMMA(i int) antlr.TerminalNode

	// IsAnonymousInputValuesContext differentiates from other interfaces.
	IsAnonymousInputValuesContext()
}

type AnonymousInputValuesContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyAnonymousInputValuesContext() *AnonymousInputValuesContext {
	var p = new(AnonymousInputValuesContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_anonymousInputValues
	return p
}

func InitEmptyAnonymousInputValuesContext(p *AnonymousInputValuesContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_anonymousInputValues
}

func (*AnonymousInputValuesContext) IsAnonymousInputValuesContext() {}

func NewAnonymousInputValuesContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *AnonymousInputValuesContext {
	var p = new(AnonymousInputValuesContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_anonymousInputValues

	return p
}

func (s *AnonymousInputValuesContext) GetParser() antlr.Parser { return s.parser }

func (s *AnonymousInputValuesContext) AllExpression() []IExpressionContext {
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

func (s *AnonymousInputValuesContext) Expression(i int) IExpressionContext {
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

func (s *AnonymousInputValuesContext) AllCOMMA() []antlr.TerminalNode {
	return s.GetTokens(ArcParserCOMMA)
}

func (s *AnonymousInputValuesContext) COMMA(i int) antlr.TerminalNode {
	return s.GetToken(ArcParserCOMMA, i)
}

func (s *AnonymousInputValuesContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *AnonymousInputValuesContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *AnonymousInputValuesContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterAnonymousInputValues(s)
	}
}

func (s *AnonymousInputValuesContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitAnonymousInputValues(s)
	}
}

func (s *AnonymousInputValuesContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitAnonymousInputValues(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) AnonymousInputValues() (localctx IAnonymousInputValuesContext) {
	localctx = NewAnonymousInputValuesContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 70, ArcParserRULE_anonymousInputValues)
	var _la int

	var _alt int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(466)
		p.Expression()
	}
	p.SetState(471)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 49, p.GetParserRuleContext())
	if p.HasError() {
		goto errorExit
	}
	for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
		if _alt == 1 {
			{
				p.SetState(467)
				p.Match(ArcParserCOMMA)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}
			{
				p.SetState(468)
				p.Expression()
			}

		}
		p.SetState(473)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 49, p.GetParserRuleContext())
		if p.HasError() {
			goto errorExit
		}
	}
	p.SetState(475)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ArcParserCOMMA {
		{
			p.SetState(474)
			p.Match(ArcParserCOMMA)
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

// IArgumentListContext is an interface to support dynamic dispatch.
type IArgumentListContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	NamedInputValues() INamedInputValuesContext
	AnonymousInputValues() IAnonymousInputValuesContext

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

func (s *ArgumentListContext) NamedInputValues() INamedInputValuesContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(INamedInputValuesContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(INamedInputValuesContext)
}

func (s *ArgumentListContext) AnonymousInputValues() IAnonymousInputValuesContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IAnonymousInputValuesContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IAnonymousInputValuesContext)
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

func (s *ArgumentListContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitArgumentList(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) ArgumentList() (localctx IArgumentListContext) {
	localctx = NewArgumentListContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 72, ArcParserRULE_argumentList)
	p.SetState(479)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 51, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(477)
			p.NamedInputValues()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(478)
			p.AnonymousInputValues()
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

func (s *BlockContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitBlock(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) Block() (localctx IBlockContext) {
	localctx = NewBlockContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 74, ArcParserRULE_block)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(481)
		p.Match(ArcParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(485)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for (int64((_la-2)) & ^0x3f) == 0 && ((int64(1)<<(_la-2))&-556757435180648387) != 0 {
		{
			p.SetState(482)
			p.Statement()
		}

		p.SetState(487)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(488)
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
	Assignment() IAssignmentContext
	IfStatement() IIfStatementContext
	ForStatement() IForStatementContext
	BreakStatement() IBreakStatementContext
	ContinueStatement() IContinueStatementContext
	ReturnStatement() IReturnStatementContext
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

func (s *StatementContext) ForStatement() IForStatementContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IForStatementContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IForStatementContext)
}

func (s *StatementContext) BreakStatement() IBreakStatementContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IBreakStatementContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IBreakStatementContext)
}

func (s *StatementContext) ContinueStatement() IContinueStatementContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IContinueStatementContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IContinueStatementContext)
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

func (s *StatementContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitStatement(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) Statement() (localctx IStatementContext) {
	localctx = NewStatementContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 76, ArcParserRULE_statement)
	p.SetState(498)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 53, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(490)
			p.VariableDeclaration()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(491)
			p.Assignment()
		}

	case 3:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(492)
			p.IfStatement()
		}

	case 4:
		p.EnterOuterAlt(localctx, 4)
		{
			p.SetState(493)
			p.ForStatement()
		}

	case 5:
		p.EnterOuterAlt(localctx, 5)
		{
			p.SetState(494)
			p.BreakStatement()
		}

	case 6:
		p.EnterOuterAlt(localctx, 6)
		{
			p.SetState(495)
			p.ContinueStatement()
		}

	case 7:
		p.EnterOuterAlt(localctx, 7)
		{
			p.SetState(496)
			p.ReturnStatement()
		}

	case 8:
		p.EnterOuterAlt(localctx, 8)
		{
			p.SetState(497)
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

func (s *VariableDeclarationContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitVariableDeclaration(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) VariableDeclaration() (localctx IVariableDeclarationContext) {
	localctx = NewVariableDeclarationContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 78, ArcParserRULE_variableDeclaration)
	p.SetState(502)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 54, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(500)
			p.LocalVariable()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(501)
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

func (s *LocalVariableContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitLocalVariable(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) LocalVariable() (localctx ILocalVariableContext) {
	localctx = NewLocalVariableContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 80, ArcParserRULE_localVariable)
	p.SetState(512)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 55, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(504)
			p.Match(ArcParserIDENTIFIER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(505)
			p.Match(ArcParserDECLARE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(506)
			p.Expression()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(507)
			p.Match(ArcParserIDENTIFIER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(508)
			p.Type_()
		}
		{
			p.SetState(509)
			p.Match(ArcParserDECLARE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(510)
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

func (s *StatefulVariableContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitStatefulVariable(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) StatefulVariable() (localctx IStatefulVariableContext) {
	localctx = NewStatefulVariableContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 82, ArcParserRULE_statefulVariable)
	p.SetState(522)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 56, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(514)
			p.Match(ArcParserIDENTIFIER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(515)
			p.Match(ArcParserSTATE_DECLARE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(516)
			p.Expression()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(517)
			p.Match(ArcParserIDENTIFIER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(518)
			p.Type_()
		}
		{
			p.SetState(519)
			p.Match(ArcParserSTATE_DECLARE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(520)
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
	IndexOrSlice() IIndexOrSliceContext
	CompoundOp() ICompoundOpContext

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

func (s *AssignmentContext) IndexOrSlice() IIndexOrSliceContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IIndexOrSliceContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IIndexOrSliceContext)
}

func (s *AssignmentContext) CompoundOp() ICompoundOpContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ICompoundOpContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ICompoundOpContext)
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

func (s *AssignmentContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitAssignment(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) Assignment() (localctx IAssignmentContext) {
	localctx = NewAssignmentContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 84, ArcParserRULE_assignment)
	p.SetState(541)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 57, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(524)
			p.Match(ArcParserIDENTIFIER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(525)
			p.Match(ArcParserASSIGN)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(526)
			p.Expression()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(527)
			p.Match(ArcParserIDENTIFIER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(528)
			p.IndexOrSlice()
		}
		{
			p.SetState(529)
			p.Match(ArcParserASSIGN)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(530)
			p.Expression()
		}

	case 3:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(532)
			p.Match(ArcParserIDENTIFIER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(533)
			p.CompoundOp()
		}
		{
			p.SetState(534)
			p.Expression()
		}

	case 4:
		p.EnterOuterAlt(localctx, 4)
		{
			p.SetState(536)
			p.Match(ArcParserIDENTIFIER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(537)
			p.IndexOrSlice()
		}
		{
			p.SetState(538)
			p.CompoundOp()
		}
		{
			p.SetState(539)
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

// ICompoundOpContext is an interface to support dynamic dispatch.
type ICompoundOpContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	PLUS_ASSIGN() antlr.TerminalNode
	MINUS_ASSIGN() antlr.TerminalNode
	STAR_ASSIGN() antlr.TerminalNode
	SLASH_ASSIGN() antlr.TerminalNode
	PERCENT_ASSIGN() antlr.TerminalNode

	// IsCompoundOpContext differentiates from other interfaces.
	IsCompoundOpContext()
}

type CompoundOpContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyCompoundOpContext() *CompoundOpContext {
	var p = new(CompoundOpContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_compoundOp
	return p
}

func InitEmptyCompoundOpContext(p *CompoundOpContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_compoundOp
}

func (*CompoundOpContext) IsCompoundOpContext() {}

func NewCompoundOpContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *CompoundOpContext {
	var p = new(CompoundOpContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_compoundOp

	return p
}

func (s *CompoundOpContext) GetParser() antlr.Parser { return s.parser }

func (s *CompoundOpContext) PLUS_ASSIGN() antlr.TerminalNode {
	return s.GetToken(ArcParserPLUS_ASSIGN, 0)
}

func (s *CompoundOpContext) MINUS_ASSIGN() antlr.TerminalNode {
	return s.GetToken(ArcParserMINUS_ASSIGN, 0)
}

func (s *CompoundOpContext) STAR_ASSIGN() antlr.TerminalNode {
	return s.GetToken(ArcParserSTAR_ASSIGN, 0)
}

func (s *CompoundOpContext) SLASH_ASSIGN() antlr.TerminalNode {
	return s.GetToken(ArcParserSLASH_ASSIGN, 0)
}

func (s *CompoundOpContext) PERCENT_ASSIGN() antlr.TerminalNode {
	return s.GetToken(ArcParserPERCENT_ASSIGN, 0)
}

func (s *CompoundOpContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *CompoundOpContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *CompoundOpContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterCompoundOp(s)
	}
}

func (s *CompoundOpContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitCompoundOp(s)
	}
}

func (s *CompoundOpContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitCompoundOp(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) CompoundOp() (localctx ICompoundOpContext) {
	localctx = NewCompoundOpContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 86, ArcParserRULE_compoundOp)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(543)
		_la = p.GetTokenStream().LA(1)

		if !((int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&133143986176) != 0) {
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

func (s *IfStatementContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitIfStatement(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) IfStatement() (localctx IIfStatementContext) {
	localctx = NewIfStatementContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 88, ArcParserRULE_ifStatement)
	var _la int

	var _alt int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(545)
		p.Match(ArcParserIF)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(546)
		p.Expression()
	}
	{
		p.SetState(547)
		p.Block()
	}
	p.SetState(551)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 58, p.GetParserRuleContext())
	if p.HasError() {
		goto errorExit
	}
	for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
		if _alt == 1 {
			{
				p.SetState(548)
				p.ElseIfClause()
			}

		}
		p.SetState(553)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 58, p.GetParserRuleContext())
		if p.HasError() {
			goto errorExit
		}
	}
	p.SetState(555)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ArcParserELSE {
		{
			p.SetState(554)
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

func (s *ElseIfClauseContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitElseIfClause(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) ElseIfClause() (localctx IElseIfClauseContext) {
	localctx = NewElseIfClauseContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 90, ArcParserRULE_elseIfClause)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(557)
		p.Match(ArcParserELSE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(558)
		p.Match(ArcParserIF)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(559)
		p.Expression()
	}
	{
		p.SetState(560)
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

func (s *ElseClauseContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitElseClause(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) ElseClause() (localctx IElseClauseContext) {
	localctx = NewElseClauseContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 92, ArcParserRULE_elseClause)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(562)
		p.Match(ArcParserELSE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(563)
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

// IForStatementContext is an interface to support dynamic dispatch.
type IForStatementContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	FOR() antlr.TerminalNode
	ForClause() IForClauseContext
	Block() IBlockContext

	// IsForStatementContext differentiates from other interfaces.
	IsForStatementContext()
}

type ForStatementContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyForStatementContext() *ForStatementContext {
	var p = new(ForStatementContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_forStatement
	return p
}

func InitEmptyForStatementContext(p *ForStatementContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_forStatement
}

func (*ForStatementContext) IsForStatementContext() {}

func NewForStatementContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ForStatementContext {
	var p = new(ForStatementContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_forStatement

	return p
}

func (s *ForStatementContext) GetParser() antlr.Parser { return s.parser }

func (s *ForStatementContext) FOR() antlr.TerminalNode {
	return s.GetToken(ArcParserFOR, 0)
}

func (s *ForStatementContext) ForClause() IForClauseContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IForClauseContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IForClauseContext)
}

func (s *ForStatementContext) Block() IBlockContext {
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

func (s *ForStatementContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ForStatementContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ForStatementContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterForStatement(s)
	}
}

func (s *ForStatementContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitForStatement(s)
	}
}

func (s *ForStatementContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitForStatement(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) ForStatement() (localctx IForStatementContext) {
	localctx = NewForStatementContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 94, ArcParserRULE_forStatement)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(565)
		p.Match(ArcParserFOR)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(566)
		p.ForClause()
	}
	{
		p.SetState(567)
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

// IForClauseContext is an interface to support dynamic dispatch.
type IForClauseContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AllIDENTIFIER() []antlr.TerminalNode
	IDENTIFIER(i int) antlr.TerminalNode
	COMMA() antlr.TerminalNode
	DECLARE() antlr.TerminalNode
	Expression() IExpressionContext

	// IsForClauseContext differentiates from other interfaces.
	IsForClauseContext()
}

type ForClauseContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyForClauseContext() *ForClauseContext {
	var p = new(ForClauseContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_forClause
	return p
}

func InitEmptyForClauseContext(p *ForClauseContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_forClause
}

func (*ForClauseContext) IsForClauseContext() {}

func NewForClauseContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ForClauseContext {
	var p = new(ForClauseContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_forClause

	return p
}

func (s *ForClauseContext) GetParser() antlr.Parser { return s.parser }

func (s *ForClauseContext) AllIDENTIFIER() []antlr.TerminalNode {
	return s.GetTokens(ArcParserIDENTIFIER)
}

func (s *ForClauseContext) IDENTIFIER(i int) antlr.TerminalNode {
	return s.GetToken(ArcParserIDENTIFIER, i)
}

func (s *ForClauseContext) COMMA() antlr.TerminalNode {
	return s.GetToken(ArcParserCOMMA, 0)
}

func (s *ForClauseContext) DECLARE() antlr.TerminalNode {
	return s.GetToken(ArcParserDECLARE, 0)
}

func (s *ForClauseContext) Expression() IExpressionContext {
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

func (s *ForClauseContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ForClauseContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ForClauseContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterForClause(s)
	}
}

func (s *ForClauseContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitForClause(s)
	}
}

func (s *ForClauseContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitForClause(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) ForClause() (localctx IForClauseContext) {
	localctx = NewForClauseContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 96, ArcParserRULE_forClause)
	p.SetState(579)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 60, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(569)
			p.Match(ArcParserIDENTIFIER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(570)
			p.Match(ArcParserCOMMA)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(571)
			p.Match(ArcParserIDENTIFIER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(572)
			p.Match(ArcParserDECLARE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(573)
			p.Expression()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(574)
			p.Match(ArcParserIDENTIFIER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(575)
			p.Match(ArcParserDECLARE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(576)
			p.Expression()
		}

	case 3:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(577)
			p.Expression()
		}

	case 4:
		p.EnterOuterAlt(localctx, 4)

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

// IBreakStatementContext is an interface to support dynamic dispatch.
type IBreakStatementContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	BREAK() antlr.TerminalNode

	// IsBreakStatementContext differentiates from other interfaces.
	IsBreakStatementContext()
}

type BreakStatementContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyBreakStatementContext() *BreakStatementContext {
	var p = new(BreakStatementContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_breakStatement
	return p
}

func InitEmptyBreakStatementContext(p *BreakStatementContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_breakStatement
}

func (*BreakStatementContext) IsBreakStatementContext() {}

func NewBreakStatementContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *BreakStatementContext {
	var p = new(BreakStatementContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_breakStatement

	return p
}

func (s *BreakStatementContext) GetParser() antlr.Parser { return s.parser }

func (s *BreakStatementContext) BREAK() antlr.TerminalNode {
	return s.GetToken(ArcParserBREAK, 0)
}

func (s *BreakStatementContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *BreakStatementContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *BreakStatementContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterBreakStatement(s)
	}
}

func (s *BreakStatementContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitBreakStatement(s)
	}
}

func (s *BreakStatementContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitBreakStatement(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) BreakStatement() (localctx IBreakStatementContext) {
	localctx = NewBreakStatementContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 98, ArcParserRULE_breakStatement)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(581)
		p.Match(ArcParserBREAK)
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

// IContinueStatementContext is an interface to support dynamic dispatch.
type IContinueStatementContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	CONTINUE() antlr.TerminalNode

	// IsContinueStatementContext differentiates from other interfaces.
	IsContinueStatementContext()
}

type ContinueStatementContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyContinueStatementContext() *ContinueStatementContext {
	var p = new(ContinueStatementContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_continueStatement
	return p
}

func InitEmptyContinueStatementContext(p *ContinueStatementContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_continueStatement
}

func (*ContinueStatementContext) IsContinueStatementContext() {}

func NewContinueStatementContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ContinueStatementContext {
	var p = new(ContinueStatementContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_continueStatement

	return p
}

func (s *ContinueStatementContext) GetParser() antlr.Parser { return s.parser }

func (s *ContinueStatementContext) CONTINUE() antlr.TerminalNode {
	return s.GetToken(ArcParserCONTINUE, 0)
}

func (s *ContinueStatementContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ContinueStatementContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ContinueStatementContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterContinueStatement(s)
	}
}

func (s *ContinueStatementContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitContinueStatement(s)
	}
}

func (s *ContinueStatementContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitContinueStatement(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) ContinueStatement() (localctx IContinueStatementContext) {
	localctx = NewContinueStatementContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 100, ArcParserRULE_continueStatement)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(583)
		p.Match(ArcParserCONTINUE)
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

func (s *ReturnStatementContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitReturnStatement(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) ReturnStatement() (localctx IReturnStatementContext) {
	localctx = NewReturnStatementContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 102, ArcParserRULE_returnStatement)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(585)
		p.Match(ArcParserRETURN)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(587)
	p.GetErrorHandler().Sync(p)

	if p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 61, p.GetParserRuleContext()) == 1 {
		{
			p.SetState(586)
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

// ITypeContext is an interface to support dynamic dispatch.
type ITypeContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	PrimitiveType() IPrimitiveTypeContext
	UnitSuffix() IUnitSuffixContext
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

func (s *TypeContext) UnitSuffix() IUnitSuffixContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IUnitSuffixContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IUnitSuffixContext)
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

func (s *TypeContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitType(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) Type_() (localctx ITypeContext) {
	localctx = NewTypeContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 104, ArcParserRULE_type)
	var _la int

	p.SetState(595)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case ArcParserI8, ArcParserI16, ArcParserI32, ArcParserI64, ArcParserU8, ArcParserU16, ArcParserU32, ArcParserU64, ArcParserF32, ArcParserF64, ArcParserSTR:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(589)
			p.PrimitiveType()
		}
		p.SetState(591)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		if _la == ArcParserIDENTIFIER {
			{
				p.SetState(590)
				p.UnitSuffix()
			}

		}

	case ArcParserCHAN:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(593)
			p.ChannelType()
		}

	case ArcParserSERIES:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(594)
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

// IUnitSuffixContext is an interface to support dynamic dispatch.
type IUnitSuffixContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	IDENTIFIER() antlr.TerminalNode

	// IsUnitSuffixContext differentiates from other interfaces.
	IsUnitSuffixContext()
}

type UnitSuffixContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyUnitSuffixContext() *UnitSuffixContext {
	var p = new(UnitSuffixContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_unitSuffix
	return p
}

func InitEmptyUnitSuffixContext(p *UnitSuffixContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_unitSuffix
}

func (*UnitSuffixContext) IsUnitSuffixContext() {}

func NewUnitSuffixContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *UnitSuffixContext {
	var p = new(UnitSuffixContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_unitSuffix

	return p
}

func (s *UnitSuffixContext) GetParser() antlr.Parser { return s.parser }

func (s *UnitSuffixContext) IDENTIFIER() antlr.TerminalNode {
	return s.GetToken(ArcParserIDENTIFIER, 0)
}

func (s *UnitSuffixContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *UnitSuffixContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *UnitSuffixContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterUnitSuffix(s)
	}
}

func (s *UnitSuffixContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitUnitSuffix(s)
	}
}

func (s *UnitSuffixContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitUnitSuffix(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) UnitSuffix() (localctx IUnitSuffixContext) {
	localctx = NewUnitSuffixContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 106, ArcParserRULE_unitSuffix)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(597)
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

func (s *PrimitiveTypeContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitPrimitiveType(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) PrimitiveType() (localctx IPrimitiveTypeContext) {
	localctx = NewPrimitiveTypeContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 108, ArcParserRULE_primitiveType)
	p.SetState(601)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case ArcParserI8, ArcParserI16, ArcParserI32, ArcParserI64, ArcParserU8, ArcParserU16, ArcParserU32, ArcParserU64, ArcParserF32, ArcParserF64:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(599)
			p.NumericType()
		}

	case ArcParserSTR:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(600)
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

func (s *NumericTypeContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitNumericType(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) NumericType() (localctx INumericTypeContext) {
	localctx = NewNumericTypeContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 110, ArcParserRULE_numericType)
	p.SetState(605)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case ArcParserI8, ArcParserI16, ArcParserI32, ArcParserI64, ArcParserU8, ArcParserU16, ArcParserU32, ArcParserU64:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(603)
			p.IntegerType()
		}

	case ArcParserF32, ArcParserF64:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(604)
			p.FloatType()
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

func (s *IntegerTypeContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitIntegerType(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) IntegerType() (localctx IIntegerTypeContext) {
	localctx = NewIntegerTypeContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 112, ArcParserRULE_integerType)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(607)
		_la = p.GetTokenStream().LA(1)

		if !((int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&8355840) != 0) {
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

func (s *FloatTypeContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitFloatType(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) FloatType() (localctx IFloatTypeContext) {
	localctx = NewFloatTypeContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 114, ArcParserRULE_floatType)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(609)
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

// IChannelTypeContext is an interface to support dynamic dispatch.
type IChannelTypeContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	CHAN() antlr.TerminalNode
	PrimitiveType() IPrimitiveTypeContext
	UnitSuffix() IUnitSuffixContext
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

func (s *ChannelTypeContext) UnitSuffix() IUnitSuffixContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IUnitSuffixContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IUnitSuffixContext)
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

func (s *ChannelTypeContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitChannelType(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) ChannelType() (localctx IChannelTypeContext) {
	localctx = NewChannelTypeContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 116, ArcParserRULE_channelType)
	var _la int

	p.SetState(618)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 67, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(611)
			p.Match(ArcParserCHAN)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(612)
			p.PrimitiveType()
		}
		p.SetState(614)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		if _la == ArcParserIDENTIFIER {
			{
				p.SetState(613)
				p.UnitSuffix()
			}

		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(616)
			p.Match(ArcParserCHAN)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(617)
			p.SeriesType()
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

// ISeriesTypeContext is an interface to support dynamic dispatch.
type ISeriesTypeContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	SERIES() antlr.TerminalNode
	PrimitiveType() IPrimitiveTypeContext
	UnitSuffix() IUnitSuffixContext

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

func (s *SeriesTypeContext) UnitSuffix() IUnitSuffixContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IUnitSuffixContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IUnitSuffixContext)
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

func (s *SeriesTypeContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitSeriesType(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) SeriesType() (localctx ISeriesTypeContext) {
	localctx = NewSeriesTypeContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 118, ArcParserRULE_seriesType)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(620)
		p.Match(ArcParserSERIES)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(621)
		p.PrimitiveType()
	}
	p.SetState(623)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ArcParserIDENTIFIER {
		{
			p.SetState(622)
			p.UnitSuffix()
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

func (s *ExpressionContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitExpression(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) Expression() (localctx IExpressionContext) {
	localctx = NewExpressionContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 120, ArcParserRULE_expression)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(625)
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

func (s *LogicalOrExpressionContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitLogicalOrExpression(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) LogicalOrExpression() (localctx ILogicalOrExpressionContext) {
	localctx = NewLogicalOrExpressionContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 122, ArcParserRULE_logicalOrExpression)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(627)
		p.LogicalAndExpression()
	}
	p.SetState(632)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == ArcParserOR {
		{
			p.SetState(628)
			p.Match(ArcParserOR)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(629)
			p.LogicalAndExpression()
		}

		p.SetState(634)
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

func (s *LogicalAndExpressionContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitLogicalAndExpression(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) LogicalAndExpression() (localctx ILogicalAndExpressionContext) {
	localctx = NewLogicalAndExpressionContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 124, ArcParserRULE_logicalAndExpression)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(635)
		p.EqualityExpression()
	}
	p.SetState(640)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == ArcParserAND {
		{
			p.SetState(636)
			p.Match(ArcParserAND)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(637)
			p.EqualityExpression()
		}

		p.SetState(642)
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

func (s *EqualityExpressionContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitEqualityExpression(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) EqualityExpression() (localctx IEqualityExpressionContext) {
	localctx = NewEqualityExpressionContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 126, ArcParserRULE_equalityExpression)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(643)
		p.RelationalExpression()
	}
	p.SetState(648)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == ArcParserEQ || _la == ArcParserNEQ {
		{
			p.SetState(644)
			_la = p.GetTokenStream().LA(1)

			if !(_la == ArcParserEQ || _la == ArcParserNEQ) {
				p.GetErrorHandler().RecoverInline(p)
			} else {
				p.GetErrorHandler().ReportMatch(p)
				p.Consume()
			}
		}
		{
			p.SetState(645)
			p.RelationalExpression()
		}

		p.SetState(650)
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

func (s *RelationalExpressionContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitRelationalExpression(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) RelationalExpression() (localctx IRelationalExpressionContext) {
	localctx = NewRelationalExpressionContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 128, ArcParserRULE_relationalExpression)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(651)
		p.AdditiveExpression()
	}
	p.SetState(656)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&527765581332480) != 0 {
		{
			p.SetState(652)
			_la = p.GetTokenStream().LA(1)

			if !((int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&527765581332480) != 0) {
				p.GetErrorHandler().RecoverInline(p)
			} else {
				p.GetErrorHandler().ReportMatch(p)
				p.Consume()
			}
		}
		{
			p.SetState(653)
			p.AdditiveExpression()
		}

		p.SetState(658)
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

func (s *AdditiveExpressionContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitAdditiveExpression(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) AdditiveExpression() (localctx IAdditiveExpressionContext) {
	localctx = NewAdditiveExpressionContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 130, ArcParserRULE_additiveExpression)
	var _la int

	var _alt int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(659)
		p.MultiplicativeExpression()
	}
	p.SetState(664)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 73, p.GetParserRuleContext())
	if p.HasError() {
		goto errorExit
	}
	for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
		if _alt == 1 {
			{
				p.SetState(660)
				_la = p.GetTokenStream().LA(1)

				if !(_la == ArcParserPLUS || _la == ArcParserMINUS) {
					p.GetErrorHandler().RecoverInline(p)
				} else {
					p.GetErrorHandler().ReportMatch(p)
					p.Consume()
				}
			}
			{
				p.SetState(661)
				p.MultiplicativeExpression()
			}

		}
		p.SetState(666)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 73, p.GetParserRuleContext())
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

func (s *MultiplicativeExpressionContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitMultiplicativeExpression(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) MultiplicativeExpression() (localctx IMultiplicativeExpressionContext) {
	localctx = NewMultiplicativeExpressionContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 132, ArcParserRULE_multiplicativeExpression)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(667)
		p.PowerExpression()
	}
	p.SetState(672)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&3848290697216) != 0 {
		{
			p.SetState(668)
			_la = p.GetTokenStream().LA(1)

			if !((int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&3848290697216) != 0) {
				p.GetErrorHandler().RecoverInline(p)
			} else {
				p.GetErrorHandler().ReportMatch(p)
				p.Consume()
			}
		}
		{
			p.SetState(669)
			p.PowerExpression()
		}

		p.SetState(674)
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

func (s *PowerExpressionContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitPowerExpression(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) PowerExpression() (localctx IPowerExpressionContext) {
	localctx = NewPowerExpressionContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 134, ArcParserRULE_powerExpression)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(675)
		p.UnaryExpression()
	}
	p.SetState(678)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ArcParserCARET {
		{
			p.SetState(676)
			p.Match(ArcParserCARET)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(677)
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

func (s *UnaryExpressionContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitUnaryExpression(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) UnaryExpression() (localctx IUnaryExpressionContext) {
	localctx = NewUnaryExpressionContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 136, ArcParserRULE_unaryExpression)
	p.SetState(685)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 76, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(680)
			p.Match(ArcParserMINUS)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(681)
			p.UnaryExpression()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(682)
			p.Match(ArcParserNOT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(683)
			p.UnaryExpression()
		}

	case 3:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(684)
			p.PostfixExpression()
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

func (s *PostfixExpressionContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitPostfixExpression(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) PostfixExpression() (localctx IPostfixExpressionContext) {
	localctx = NewPostfixExpressionContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 138, ArcParserRULE_postfixExpression)
	var _alt int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(687)
		p.PrimaryExpression()
	}
	p.SetState(692)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 78, p.GetParserRuleContext())
	if p.HasError() {
		goto errorExit
	}
	for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
		if _alt == 1 {
			p.SetState(690)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}

			switch p.GetTokenStream().LA(1) {
			case ArcParserLBRACKET:
				{
					p.SetState(688)
					p.IndexOrSlice()
				}

			case ArcParserLPAREN:
				{
					p.SetState(689)
					p.FunctionCallSuffix()
				}

			default:
				p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
				goto errorExit
			}

		}
		p.SetState(694)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 78, p.GetParserRuleContext())
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

func (s *IndexOrSliceContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitIndexOrSlice(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) IndexOrSlice() (localctx IIndexOrSliceContext) {
	localctx = NewIndexOrSliceContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 140, ArcParserRULE_indexOrSlice)
	var _la int

	p.SetState(708)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 81, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(695)
			p.Match(ArcParserLBRACKET)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(696)
			p.Expression()
		}
		{
			p.SetState(697)
			p.Match(ArcParserRBRACKET)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(699)
			p.Match(ArcParserLBRACKET)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		p.SetState(701)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		if (int64((_la-13)) & ^0x3f) == 0 && ((int64(1)<<(_la-13))&8735345038344191) != 0 {
			{
				p.SetState(700)
				p.Expression()
			}

		}
		{
			p.SetState(703)
			p.Match(ArcParserCOLON)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		p.SetState(705)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		if (int64((_la-13)) & ^0x3f) == 0 && ((int64(1)<<(_la-13))&8735345038344191) != 0 {
			{
				p.SetState(704)
				p.Expression()
			}

		}
		{
			p.SetState(707)
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

func (s *FunctionCallSuffixContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitFunctionCallSuffix(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) FunctionCallSuffix() (localctx IFunctionCallSuffixContext) {
	localctx = NewFunctionCallSuffixContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 142, ArcParserRULE_functionCallSuffix)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(710)
		p.Match(ArcParserLPAREN)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(712)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if (int64((_la-13)) & ^0x3f) == 0 && ((int64(1)<<(_la-13))&8735345038344191) != 0 {
		{
			p.SetState(711)
			p.ArgumentList()
		}

	}
	{
		p.SetState(714)
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
	QualifiedIdentifier() IQualifiedIdentifierContext
	IDENTIFIER() antlr.TerminalNode
	LPAREN() antlr.TerminalNode
	Expression() IExpressionContext
	RPAREN() antlr.TerminalNode
	TypeCast() ITypeCastContext

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

func (s *PrimaryExpressionContext) QualifiedIdentifier() IQualifiedIdentifierContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IQualifiedIdentifierContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IQualifiedIdentifierContext)
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

func (s *PrimaryExpressionContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitPrimaryExpression(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) PrimaryExpression() (localctx IPrimaryExpressionContext) {
	localctx = NewPrimaryExpressionContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 144, ArcParserRULE_primaryExpression)
	p.SetState(724)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 83, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(716)
			p.Literal()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(717)
			p.QualifiedIdentifier()
		}

	case 3:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(718)
			p.Match(ArcParserIDENTIFIER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 4:
		p.EnterOuterAlt(localctx, 4)
		{
			p.SetState(719)
			p.Match(ArcParserLPAREN)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(720)
			p.Expression()
		}
		{
			p.SetState(721)
			p.Match(ArcParserRPAREN)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 5:
		p.EnterOuterAlt(localctx, 5)
		{
			p.SetState(723)
			p.TypeCast()
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

func (s *TypeCastContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitTypeCast(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) TypeCast() (localctx ITypeCastContext) {
	localctx = NewTypeCastContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 146, ArcParserRULE_typeCast)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(726)
		p.Type_()
	}
	{
		p.SetState(727)
		p.Match(ArcParserLPAREN)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(728)
		p.Expression()
	}
	{
		p.SetState(729)
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

// ILiteralContext is an interface to support dynamic dispatch.
type ILiteralContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	NumericLiteral() INumericLiteralContext
	STR_LITERAL() antlr.TerminalNode
	STR_LITERAL_MULTI() antlr.TerminalNode
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

func (s *LiteralContext) STR_LITERAL() antlr.TerminalNode {
	return s.GetToken(ArcParserSTR_LITERAL, 0)
}

func (s *LiteralContext) STR_LITERAL_MULTI() antlr.TerminalNode {
	return s.GetToken(ArcParserSTR_LITERAL_MULTI, 0)
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

func (s *LiteralContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitLiteral(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) Literal() (localctx ILiteralContext) {
	localctx = NewLiteralContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 148, ArcParserRULE_literal)
	p.SetState(735)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case ArcParserMINUS, ArcParserINTEGER_LITERAL, ArcParserFLOAT_LITERAL:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(731)
			p.NumericLiteral()
		}

	case ArcParserSTR_LITERAL:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(732)
			p.Match(ArcParserSTR_LITERAL)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case ArcParserSTR_LITERAL_MULTI:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(733)
			p.Match(ArcParserSTR_LITERAL_MULTI)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case ArcParserLBRACKET:
		p.EnterOuterAlt(localctx, 4)
		{
			p.SetState(734)
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
	MINUS() antlr.TerminalNode
	IDENTIFIER() antlr.TerminalNode

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

func (s *NumericLiteralContext) MINUS() antlr.TerminalNode {
	return s.GetToken(ArcParserMINUS, 0)
}

func (s *NumericLiteralContext) IDENTIFIER() antlr.TerminalNode {
	return s.GetToken(ArcParserIDENTIFIER, 0)
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

func (s *NumericLiteralContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitNumericLiteral(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) NumericLiteral() (localctx INumericLiteralContext) {
	localctx = NewNumericLiteralContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 150, ArcParserRULE_numericLiteral)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(738)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ArcParserMINUS {
		{
			p.SetState(737)
			p.Match(ArcParserMINUS)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	}
	{
		p.SetState(740)
		_la = p.GetTokenStream().LA(1)

		if !(_la == ArcParserINTEGER_LITERAL || _la == ArcParserFLOAT_LITERAL) {
			p.GetErrorHandler().RecoverInline(p)
		} else {
			p.GetErrorHandler().ReportMatch(p)
			p.Consume()
		}
	}
	p.SetState(743)
	p.GetErrorHandler().Sync(p)

	if p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 86, p.GetParserRuleContext()) == 1 {
		p.SetState(741)

		if !(p.TokensAdjacent(p.GetTokenStream().LT(-1), p.GetTokenStream().LT(1))) {
			p.SetError(antlr.NewFailedPredicateException(p, "p.TokensAdjacent(p.GetTokenStream().LT(-1), p.GetTokenStream().LT(1))", ""))
			goto errorExit
		}
		{
			p.SetState(742)
			p.Match(ArcParserIDENTIFIER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
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

func (s *SeriesLiteralContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitSeriesLiteral(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) SeriesLiteral() (localctx ISeriesLiteralContext) {
	localctx = NewSeriesLiteralContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 152, ArcParserRULE_seriesLiteral)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(745)
		p.Match(ArcParserLBRACKET)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(747)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if (int64((_la-13)) & ^0x3f) == 0 && ((int64(1)<<(_la-13))&8735345038344191) != 0 {
		{
			p.SetState(746)
			p.ExpressionList()
		}

	}
	{
		p.SetState(749)
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

func (s *ExpressionListContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitExpressionList(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) ExpressionList() (localctx IExpressionListContext) {
	localctx = NewExpressionListContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 154, ArcParserRULE_expressionList)
	var _la int

	var _alt int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(751)
		p.Expression()
	}
	p.SetState(756)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 88, p.GetParserRuleContext())
	if p.HasError() {
		goto errorExit
	}
	for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
		if _alt == 1 {
			{
				p.SetState(752)
				p.Match(ArcParserCOMMA)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}
			{
				p.SetState(753)
				p.Expression()
			}

		}
		p.SetState(758)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 88, p.GetParserRuleContext())
		if p.HasError() {
			goto errorExit
		}
	}
	p.SetState(760)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ArcParserCOMMA {
		{
			p.SetState(759)
			p.Match(ArcParserCOMMA)
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

func (p *ArcParser) Sempred(localctx antlr.RuleContext, ruleIndex, predIndex int) bool {
	switch ruleIndex {
	case 75:
		var t *NumericLiteralContext = nil
		if localctx != nil {
			t = localctx.(*NumericLiteralContext)
		}
		return p.NumericLiteral_Sempred(t, predIndex)

	default:
		panic("No predicate with index: " + fmt.Sprint(ruleIndex))
	}
}

func (p *ArcParser) NumericLiteral_Sempred(localctx antlr.RuleContext, predIndex int) bool {
	switch predIndex {
	case 0:
		return p.TokensAdjacent(p.GetTokenStream().LT(-1), p.GetTokenStream().LT(1))

	default:
		panic("No predicate with index: " + fmt.Sprint(predIndex))
	}
}
