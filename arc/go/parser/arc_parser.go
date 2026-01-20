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
		"", "'func'", "'if'", "'else'", "'return'", "'sequence'", "'stage'",
		"'next'", "'chan'", "'i8'", "'i16'", "'i32'", "'i64'", "'u8'", "'u16'",
		"'u32'", "'u64'", "'f32'", "'f64'", "'str'", "'series'", "'->'", "':='",
		"'$='", "'=>'", "'='", "'+='", "'-='", "'*='", "'/='", "'%='", "'+'",
		"'-'", "'*'", "'/'", "'%'", "'^'", "'=='", "'!='", "'<'", "'>'", "'<='",
		"'>='", "'and'", "'or'", "'not'", "'('", "')'", "'{'", "'}'", "'['",
		"']'", "','", "':'",
	}
	staticData.SymbolicNames = []string{
		"", "FUNC", "IF", "ELSE", "RETURN", "SEQUENCE", "STAGE", "NEXT", "CHAN",
		"I8", "I16", "I32", "I64", "U8", "U16", "U32", "U64", "F32", "F64",
		"STR", "SERIES", "ARROW", "DECLARE", "STATE_DECLARE", "TRANSITION",
		"ASSIGN", "PLUS_ASSIGN", "MINUS_ASSIGN", "STAR_ASSIGN", "SLASH_ASSIGN",
		"PERCENT_ASSIGN", "PLUS", "MINUS", "STAR", "SLASH", "PERCENT", "CARET",
		"EQ", "NEQ", "LT", "GT", "LEQ", "GEQ", "AND", "OR", "NOT", "LPAREN",
		"RPAREN", "LBRACE", "RBRACE", "LBRACKET", "RBRACKET", "COMMA", "COLON",
		"INTEGER_LITERAL", "FLOAT_LITERAL", "STR_LITERAL", "IDENTIFIER", "SINGLE_LINE_COMMENT",
		"MULTI_LINE_COMMENT", "WS",
	}
	staticData.RuleNames = []string{
		"program", "topLevelItem", "functionDeclaration", "inputList", "input",
		"outputType", "multiOutputBlock", "namedOutput", "configBlock", "config",
		"sequenceDeclaration", "stageDeclaration", "stageBody", "stageItem",
		"flowStatement", "flowOperator", "routingTable", "routingEntry", "flowNode",
		"identifier", "function", "configValues", "namedConfigValues", "namedConfigValue",
		"anonymousConfigValues", "arguments", "argumentList", "block", "statement",
		"variableDeclaration", "localVariable", "statefulVariable", "assignment",
		"compoundOp", "ifStatement", "elseIfClause", "elseClause", "returnStatement",
		"functionCall", "type", "unitSuffix", "primitiveType", "numericType",
		"integerType", "floatType", "channelType", "seriesType", "expression",
		"logicalOrExpression", "logicalAndExpression", "equalityExpression",
		"relationalExpression", "additiveExpression", "multiplicativeExpression",
		"powerExpression", "unaryExpression", "postfixExpression", "indexOrSlice",
		"functionCallSuffix", "primaryExpression", "typeCast", "literal", "numericLiteral",
		"seriesLiteral", "expressionList",
	}
	staticData.PredictionContextCache = antlr.NewPredictionContextCache()
	staticData.serializedATN = []int32{
		4, 1, 60, 614, 2, 0, 7, 0, 2, 1, 7, 1, 2, 2, 7, 2, 2, 3, 7, 3, 2, 4, 7,
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
		63, 7, 63, 2, 64, 7, 64, 1, 0, 5, 0, 132, 8, 0, 10, 0, 12, 0, 135, 9, 0,
		1, 0, 1, 0, 1, 1, 1, 1, 1, 1, 3, 1, 142, 8, 1, 1, 2, 1, 2, 1, 2, 3, 2,
		147, 8, 2, 1, 2, 1, 2, 3, 2, 151, 8, 2, 1, 2, 1, 2, 3, 2, 155, 8, 2, 1,
		2, 1, 2, 1, 3, 1, 3, 1, 3, 5, 3, 162, 8, 3, 10, 3, 12, 3, 165, 9, 3, 1,
		3, 3, 3, 168, 8, 3, 1, 4, 1, 4, 1, 4, 1, 4, 3, 4, 174, 8, 4, 1, 5, 1, 5,
		1, 5, 1, 5, 3, 5, 180, 8, 5, 1, 6, 1, 6, 1, 6, 1, 6, 5, 6, 186, 8, 6, 10,
		6, 12, 6, 189, 9, 6, 1, 6, 3, 6, 192, 8, 6, 1, 6, 1, 6, 1, 7, 1, 7, 1,
		7, 1, 8, 1, 8, 5, 8, 201, 8, 8, 10, 8, 12, 8, 204, 9, 8, 1, 8, 1, 8, 1,
		9, 1, 9, 1, 9, 1, 10, 1, 10, 1, 10, 1, 10, 5, 10, 215, 8, 10, 10, 10, 12,
		10, 218, 9, 10, 1, 10, 1, 10, 1, 11, 1, 11, 1, 11, 1, 11, 1, 12, 1, 12,
		1, 12, 1, 12, 5, 12, 230, 8, 12, 10, 12, 12, 12, 233, 9, 12, 1, 12, 3,
		12, 236, 8, 12, 3, 12, 238, 8, 12, 1, 12, 1, 12, 1, 13, 1, 13, 1, 14, 1,
		14, 3, 14, 246, 8, 14, 1, 14, 1, 14, 1, 14, 3, 14, 251, 8, 14, 4, 14, 253,
		8, 14, 11, 14, 12, 14, 254, 1, 15, 1, 15, 1, 16, 1, 16, 1, 16, 1, 16, 5,
		16, 263, 8, 16, 10, 16, 12, 16, 266, 9, 16, 1, 16, 3, 16, 269, 8, 16, 1,
		16, 1, 16, 1, 17, 1, 17, 1, 17, 1, 17, 1, 17, 5, 17, 278, 8, 17, 10, 17,
		12, 17, 281, 9, 17, 1, 17, 1, 17, 3, 17, 285, 8, 17, 1, 18, 1, 18, 1, 18,
		1, 18, 3, 18, 291, 8, 18, 1, 19, 1, 19, 1, 20, 1, 20, 1, 20, 1, 21, 1,
		21, 1, 21, 1, 21, 1, 21, 1, 21, 1, 21, 1, 21, 1, 21, 1, 21, 3, 21, 308,
		8, 21, 1, 22, 1, 22, 1, 22, 5, 22, 313, 8, 22, 10, 22, 12, 22, 316, 9,
		22, 1, 22, 3, 22, 319, 8, 22, 1, 23, 1, 23, 1, 23, 1, 23, 1, 24, 1, 24,
		1, 24, 5, 24, 328, 8, 24, 10, 24, 12, 24, 331, 9, 24, 1, 24, 3, 24, 334,
		8, 24, 1, 25, 1, 25, 3, 25, 338, 8, 25, 1, 25, 1, 25, 1, 26, 1, 26, 1,
		26, 5, 26, 345, 8, 26, 10, 26, 12, 26, 348, 9, 26, 1, 26, 3, 26, 351, 8,
		26, 1, 27, 1, 27, 5, 27, 355, 8, 27, 10, 27, 12, 27, 358, 9, 27, 1, 27,
		1, 27, 1, 28, 1, 28, 1, 28, 1, 28, 1, 28, 1, 28, 3, 28, 368, 8, 28, 1,
		29, 1, 29, 3, 29, 372, 8, 29, 1, 30, 1, 30, 1, 30, 1, 30, 1, 30, 1, 30,
		1, 30, 1, 30, 3, 30, 382, 8, 30, 1, 31, 1, 31, 1, 31, 1, 31, 1, 31, 1,
		31, 1, 31, 1, 31, 3, 31, 392, 8, 31, 1, 32, 1, 32, 1, 32, 1, 32, 1, 32,
		1, 32, 1, 32, 1, 32, 1, 32, 1, 32, 1, 32, 1, 32, 1, 32, 1, 32, 1, 32, 1,
		32, 1, 32, 3, 32, 411, 8, 32, 1, 33, 1, 33, 1, 34, 1, 34, 1, 34, 1, 34,
		5, 34, 419, 8, 34, 10, 34, 12, 34, 422, 9, 34, 1, 34, 3, 34, 425, 8, 34,
		1, 35, 1, 35, 1, 35, 1, 35, 1, 35, 1, 36, 1, 36, 1, 36, 1, 37, 1, 37, 3,
		37, 437, 8, 37, 1, 38, 1, 38, 1, 38, 3, 38, 442, 8, 38, 1, 38, 1, 38, 1,
		39, 1, 39, 3, 39, 448, 8, 39, 1, 39, 1, 39, 3, 39, 452, 8, 39, 1, 40, 1,
		40, 1, 41, 1, 41, 3, 41, 458, 8, 41, 1, 42, 1, 42, 3, 42, 462, 8, 42, 1,
		43, 1, 43, 1, 44, 1, 44, 1, 45, 1, 45, 1, 45, 3, 45, 471, 8, 45, 1, 45,
		1, 45, 3, 45, 475, 8, 45, 1, 46, 1, 46, 1, 46, 3, 46, 480, 8, 46, 1, 47,
		1, 47, 1, 48, 1, 48, 1, 48, 5, 48, 487, 8, 48, 10, 48, 12, 48, 490, 9,
		48, 1, 49, 1, 49, 1, 49, 5, 49, 495, 8, 49, 10, 49, 12, 49, 498, 9, 49,
		1, 50, 1, 50, 1, 50, 5, 50, 503, 8, 50, 10, 50, 12, 50, 506, 9, 50, 1,
		51, 1, 51, 1, 51, 5, 51, 511, 8, 51, 10, 51, 12, 51, 514, 9, 51, 1, 52,
		1, 52, 1, 52, 5, 52, 519, 8, 52, 10, 52, 12, 52, 522, 9, 52, 1, 53, 1,
		53, 1, 53, 5, 53, 527, 8, 53, 10, 53, 12, 53, 530, 9, 53, 1, 54, 1, 54,
		1, 54, 3, 54, 535, 8, 54, 1, 55, 1, 55, 1, 55, 1, 55, 1, 55, 3, 55, 542,
		8, 55, 1, 56, 1, 56, 1, 56, 5, 56, 547, 8, 56, 10, 56, 12, 56, 550, 9,
		56, 1, 57, 1, 57, 1, 57, 1, 57, 1, 57, 1, 57, 3, 57, 558, 8, 57, 1, 57,
		1, 57, 3, 57, 562, 8, 57, 1, 57, 3, 57, 565, 8, 57, 1, 58, 1, 58, 3, 58,
		569, 8, 58, 1, 58, 1, 58, 1, 59, 1, 59, 1, 59, 1, 59, 1, 59, 1, 59, 1,
		59, 3, 59, 580, 8, 59, 1, 60, 1, 60, 1, 60, 1, 60, 1, 60, 1, 61, 1, 61,
		1, 61, 3, 61, 590, 8, 61, 1, 62, 1, 62, 1, 62, 3, 62, 595, 8, 62, 1, 63,
		1, 63, 3, 63, 599, 8, 63, 1, 63, 1, 63, 1, 64, 1, 64, 1, 64, 5, 64, 606,
		8, 64, 10, 64, 12, 64, 609, 9, 64, 1, 64, 3, 64, 612, 8, 64, 1, 64, 0,
		0, 65, 0, 2, 4, 6, 8, 10, 12, 14, 16, 18, 20, 22, 24, 26, 28, 30, 32, 34,
		36, 38, 40, 42, 44, 46, 48, 50, 52, 54, 56, 58, 60, 62, 64, 66, 68, 70,
		72, 74, 76, 78, 80, 82, 84, 86, 88, 90, 92, 94, 96, 98, 100, 102, 104,
		106, 108, 110, 112, 114, 116, 118, 120, 122, 124, 126, 128, 0, 9, 2, 0,
		21, 21, 24, 24, 1, 0, 26, 30, 1, 0, 9, 16, 1, 0, 17, 18, 1, 0, 37, 38,
		1, 0, 39, 42, 1, 0, 31, 32, 1, 0, 33, 35, 1, 0, 54, 55, 633, 0, 133, 1,
		0, 0, 0, 2, 141, 1, 0, 0, 0, 4, 143, 1, 0, 0, 0, 6, 158, 1, 0, 0, 0, 8,
		169, 1, 0, 0, 0, 10, 179, 1, 0, 0, 0, 12, 181, 1, 0, 0, 0, 14, 195, 1,
		0, 0, 0, 16, 198, 1, 0, 0, 0, 18, 207, 1, 0, 0, 0, 20, 210, 1, 0, 0, 0,
		22, 221, 1, 0, 0, 0, 24, 225, 1, 0, 0, 0, 26, 241, 1, 0, 0, 0, 28, 245,
		1, 0, 0, 0, 30, 256, 1, 0, 0, 0, 32, 258, 1, 0, 0, 0, 34, 272, 1, 0, 0,
		0, 36, 290, 1, 0, 0, 0, 38, 292, 1, 0, 0, 0, 40, 294, 1, 0, 0, 0, 42, 307,
		1, 0, 0, 0, 44, 309, 1, 0, 0, 0, 46, 320, 1, 0, 0, 0, 48, 324, 1, 0, 0,
		0, 50, 335, 1, 0, 0, 0, 52, 341, 1, 0, 0, 0, 54, 352, 1, 0, 0, 0, 56, 367,
		1, 0, 0, 0, 58, 371, 1, 0, 0, 0, 60, 381, 1, 0, 0, 0, 62, 391, 1, 0, 0,
		0, 64, 410, 1, 0, 0, 0, 66, 412, 1, 0, 0, 0, 68, 414, 1, 0, 0, 0, 70, 426,
		1, 0, 0, 0, 72, 431, 1, 0, 0, 0, 74, 434, 1, 0, 0, 0, 76, 438, 1, 0, 0,
		0, 78, 451, 1, 0, 0, 0, 80, 453, 1, 0, 0, 0, 82, 457, 1, 0, 0, 0, 84, 461,
		1, 0, 0, 0, 86, 463, 1, 0, 0, 0, 88, 465, 1, 0, 0, 0, 90, 474, 1, 0, 0,
		0, 92, 476, 1, 0, 0, 0, 94, 481, 1, 0, 0, 0, 96, 483, 1, 0, 0, 0, 98, 491,
		1, 0, 0, 0, 100, 499, 1, 0, 0, 0, 102, 507, 1, 0, 0, 0, 104, 515, 1, 0,
		0, 0, 106, 523, 1, 0, 0, 0, 108, 531, 1, 0, 0, 0, 110, 541, 1, 0, 0, 0,
		112, 543, 1, 0, 0, 0, 114, 564, 1, 0, 0, 0, 116, 566, 1, 0, 0, 0, 118,
		579, 1, 0, 0, 0, 120, 581, 1, 0, 0, 0, 122, 589, 1, 0, 0, 0, 124, 591,
		1, 0, 0, 0, 126, 596, 1, 0, 0, 0, 128, 602, 1, 0, 0, 0, 130, 132, 3, 2,
		1, 0, 131, 130, 1, 0, 0, 0, 132, 135, 1, 0, 0, 0, 133, 131, 1, 0, 0, 0,
		133, 134, 1, 0, 0, 0, 134, 136, 1, 0, 0, 0, 135, 133, 1, 0, 0, 0, 136,
		137, 5, 0, 0, 1, 137, 1, 1, 0, 0, 0, 138, 142, 3, 4, 2, 0, 139, 142, 3,
		28, 14, 0, 140, 142, 3, 20, 10, 0, 141, 138, 1, 0, 0, 0, 141, 139, 1, 0,
		0, 0, 141, 140, 1, 0, 0, 0, 142, 3, 1, 0, 0, 0, 143, 144, 5, 1, 0, 0, 144,
		146, 5, 57, 0, 0, 145, 147, 3, 16, 8, 0, 146, 145, 1, 0, 0, 0, 146, 147,
		1, 0, 0, 0, 147, 148, 1, 0, 0, 0, 148, 150, 5, 46, 0, 0, 149, 151, 3, 6,
		3, 0, 150, 149, 1, 0, 0, 0, 150, 151, 1, 0, 0, 0, 151, 152, 1, 0, 0, 0,
		152, 154, 5, 47, 0, 0, 153, 155, 3, 10, 5, 0, 154, 153, 1, 0, 0, 0, 154,
		155, 1, 0, 0, 0, 155, 156, 1, 0, 0, 0, 156, 157, 3, 54, 27, 0, 157, 5,
		1, 0, 0, 0, 158, 163, 3, 8, 4, 0, 159, 160, 5, 52, 0, 0, 160, 162, 3, 8,
		4, 0, 161, 159, 1, 0, 0, 0, 162, 165, 1, 0, 0, 0, 163, 161, 1, 0, 0, 0,
		163, 164, 1, 0, 0, 0, 164, 167, 1, 0, 0, 0, 165, 163, 1, 0, 0, 0, 166,
		168, 5, 52, 0, 0, 167, 166, 1, 0, 0, 0, 167, 168, 1, 0, 0, 0, 168, 7, 1,
		0, 0, 0, 169, 170, 5, 57, 0, 0, 170, 173, 3, 78, 39, 0, 171, 172, 5, 25,
		0, 0, 172, 174, 3, 122, 61, 0, 173, 171, 1, 0, 0, 0, 173, 174, 1, 0, 0,
		0, 174, 9, 1, 0, 0, 0, 175, 180, 3, 78, 39, 0, 176, 177, 5, 57, 0, 0, 177,
		180, 3, 78, 39, 0, 178, 180, 3, 12, 6, 0, 179, 175, 1, 0, 0, 0, 179, 176,
		1, 0, 0, 0, 179, 178, 1, 0, 0, 0, 180, 11, 1, 0, 0, 0, 181, 182, 5, 46,
		0, 0, 182, 187, 3, 14, 7, 0, 183, 184, 5, 52, 0, 0, 184, 186, 3, 14, 7,
		0, 185, 183, 1, 0, 0, 0, 186, 189, 1, 0, 0, 0, 187, 185, 1, 0, 0, 0, 187,
		188, 1, 0, 0, 0, 188, 191, 1, 0, 0, 0, 189, 187, 1, 0, 0, 0, 190, 192,
		5, 52, 0, 0, 191, 190, 1, 0, 0, 0, 191, 192, 1, 0, 0, 0, 192, 193, 1, 0,
		0, 0, 193, 194, 5, 47, 0, 0, 194, 13, 1, 0, 0, 0, 195, 196, 5, 57, 0, 0,
		196, 197, 3, 78, 39, 0, 197, 15, 1, 0, 0, 0, 198, 202, 5, 48, 0, 0, 199,
		201, 3, 18, 9, 0, 200, 199, 1, 0, 0, 0, 201, 204, 1, 0, 0, 0, 202, 200,
		1, 0, 0, 0, 202, 203, 1, 0, 0, 0, 203, 205, 1, 0, 0, 0, 204, 202, 1, 0,
		0, 0, 205, 206, 5, 49, 0, 0, 206, 17, 1, 0, 0, 0, 207, 208, 5, 57, 0, 0,
		208, 209, 3, 78, 39, 0, 209, 19, 1, 0, 0, 0, 210, 211, 5, 5, 0, 0, 211,
		212, 5, 57, 0, 0, 212, 216, 5, 48, 0, 0, 213, 215, 3, 22, 11, 0, 214, 213,
		1, 0, 0, 0, 215, 218, 1, 0, 0, 0, 216, 214, 1, 0, 0, 0, 216, 217, 1, 0,
		0, 0, 217, 219, 1, 0, 0, 0, 218, 216, 1, 0, 0, 0, 219, 220, 5, 49, 0, 0,
		220, 21, 1, 0, 0, 0, 221, 222, 5, 6, 0, 0, 222, 223, 5, 57, 0, 0, 223,
		224, 3, 24, 12, 0, 224, 23, 1, 0, 0, 0, 225, 237, 5, 48, 0, 0, 226, 231,
		3, 26, 13, 0, 227, 228, 5, 52, 0, 0, 228, 230, 3, 26, 13, 0, 229, 227,
		1, 0, 0, 0, 230, 233, 1, 0, 0, 0, 231, 229, 1, 0, 0, 0, 231, 232, 1, 0,
		0, 0, 232, 235, 1, 0, 0, 0, 233, 231, 1, 0, 0, 0, 234, 236, 5, 52, 0, 0,
		235, 234, 1, 0, 0, 0, 235, 236, 1, 0, 0, 0, 236, 238, 1, 0, 0, 0, 237,
		226, 1, 0, 0, 0, 237, 238, 1, 0, 0, 0, 238, 239, 1, 0, 0, 0, 239, 240,
		5, 49, 0, 0, 240, 25, 1, 0, 0, 0, 241, 242, 3, 28, 14, 0, 242, 27, 1, 0,
		0, 0, 243, 246, 3, 32, 16, 0, 244, 246, 3, 36, 18, 0, 245, 243, 1, 0, 0,
		0, 245, 244, 1, 0, 0, 0, 246, 252, 1, 0, 0, 0, 247, 250, 3, 30, 15, 0,
		248, 251, 3, 32, 16, 0, 249, 251, 3, 36, 18, 0, 250, 248, 1, 0, 0, 0, 250,
		249, 1, 0, 0, 0, 251, 253, 1, 0, 0, 0, 252, 247, 1, 0, 0, 0, 253, 254,
		1, 0, 0, 0, 254, 252, 1, 0, 0, 0, 254, 255, 1, 0, 0, 0, 255, 29, 1, 0,
		0, 0, 256, 257, 7, 0, 0, 0, 257, 31, 1, 0, 0, 0, 258, 259, 5, 48, 0, 0,
		259, 264, 3, 34, 17, 0, 260, 261, 5, 52, 0, 0, 261, 263, 3, 34, 17, 0,
		262, 260, 1, 0, 0, 0, 263, 266, 1, 0, 0, 0, 264, 262, 1, 0, 0, 0, 264,
		265, 1, 0, 0, 0, 265, 268, 1, 0, 0, 0, 266, 264, 1, 0, 0, 0, 267, 269,
		5, 52, 0, 0, 268, 267, 1, 0, 0, 0, 268, 269, 1, 0, 0, 0, 269, 270, 1, 0,
		0, 0, 270, 271, 5, 49, 0, 0, 271, 33, 1, 0, 0, 0, 272, 273, 5, 57, 0, 0,
		273, 274, 5, 53, 0, 0, 274, 279, 3, 36, 18, 0, 275, 276, 5, 21, 0, 0, 276,
		278, 3, 36, 18, 0, 277, 275, 1, 0, 0, 0, 278, 281, 1, 0, 0, 0, 279, 277,
		1, 0, 0, 0, 279, 280, 1, 0, 0, 0, 280, 284, 1, 0, 0, 0, 281, 279, 1, 0,
		0, 0, 282, 283, 5, 53, 0, 0, 283, 285, 5, 57, 0, 0, 284, 282, 1, 0, 0,
		0, 284, 285, 1, 0, 0, 0, 285, 35, 1, 0, 0, 0, 286, 291, 3, 38, 19, 0, 287,
		291, 3, 40, 20, 0, 288, 291, 3, 94, 47, 0, 289, 291, 5, 7, 0, 0, 290, 286,
		1, 0, 0, 0, 290, 287, 1, 0, 0, 0, 290, 288, 1, 0, 0, 0, 290, 289, 1, 0,
		0, 0, 291, 37, 1, 0, 0, 0, 292, 293, 5, 57, 0, 0, 293, 39, 1, 0, 0, 0,
		294, 295, 5, 57, 0, 0, 295, 296, 3, 42, 21, 0, 296, 41, 1, 0, 0, 0, 297,
		298, 5, 48, 0, 0, 298, 308, 5, 49, 0, 0, 299, 300, 5, 48, 0, 0, 300, 301,
		3, 44, 22, 0, 301, 302, 5, 49, 0, 0, 302, 308, 1, 0, 0, 0, 303, 304, 5,
		48, 0, 0, 304, 305, 3, 48, 24, 0, 305, 306, 5, 49, 0, 0, 306, 308, 1, 0,
		0, 0, 307, 297, 1, 0, 0, 0, 307, 299, 1, 0, 0, 0, 307, 303, 1, 0, 0, 0,
		308, 43, 1, 0, 0, 0, 309, 314, 3, 46, 23, 0, 310, 311, 5, 52, 0, 0, 311,
		313, 3, 46, 23, 0, 312, 310, 1, 0, 0, 0, 313, 316, 1, 0, 0, 0, 314, 312,
		1, 0, 0, 0, 314, 315, 1, 0, 0, 0, 315, 318, 1, 0, 0, 0, 316, 314, 1, 0,
		0, 0, 317, 319, 5, 52, 0, 0, 318, 317, 1, 0, 0, 0, 318, 319, 1, 0, 0, 0,
		319, 45, 1, 0, 0, 0, 320, 321, 5, 57, 0, 0, 321, 322, 5, 25, 0, 0, 322,
		323, 3, 94, 47, 0, 323, 47, 1, 0, 0, 0, 324, 329, 3, 94, 47, 0, 325, 326,
		5, 52, 0, 0, 326, 328, 3, 94, 47, 0, 327, 325, 1, 0, 0, 0, 328, 331, 1,
		0, 0, 0, 329, 327, 1, 0, 0, 0, 329, 330, 1, 0, 0, 0, 330, 333, 1, 0, 0,
		0, 331, 329, 1, 0, 0, 0, 332, 334, 5, 52, 0, 0, 333, 332, 1, 0, 0, 0, 333,
		334, 1, 0, 0, 0, 334, 49, 1, 0, 0, 0, 335, 337, 5, 46, 0, 0, 336, 338,
		3, 52, 26, 0, 337, 336, 1, 0, 0, 0, 337, 338, 1, 0, 0, 0, 338, 339, 1,
		0, 0, 0, 339, 340, 5, 47, 0, 0, 340, 51, 1, 0, 0, 0, 341, 346, 3, 94, 47,
		0, 342, 343, 5, 52, 0, 0, 343, 345, 3, 94, 47, 0, 344, 342, 1, 0, 0, 0,
		345, 348, 1, 0, 0, 0, 346, 344, 1, 0, 0, 0, 346, 347, 1, 0, 0, 0, 347,
		350, 1, 0, 0, 0, 348, 346, 1, 0, 0, 0, 349, 351, 5, 52, 0, 0, 350, 349,
		1, 0, 0, 0, 350, 351, 1, 0, 0, 0, 351, 53, 1, 0, 0, 0, 352, 356, 5, 48,
		0, 0, 353, 355, 3, 56, 28, 0, 354, 353, 1, 0, 0, 0, 355, 358, 1, 0, 0,
		0, 356, 354, 1, 0, 0, 0, 356, 357, 1, 0, 0, 0, 357, 359, 1, 0, 0, 0, 358,
		356, 1, 0, 0, 0, 359, 360, 5, 49, 0, 0, 360, 55, 1, 0, 0, 0, 361, 368,
		3, 58, 29, 0, 362, 368, 3, 64, 32, 0, 363, 368, 3, 68, 34, 0, 364, 368,
		3, 74, 37, 0, 365, 368, 3, 76, 38, 0, 366, 368, 3, 94, 47, 0, 367, 361,
		1, 0, 0, 0, 367, 362, 1, 0, 0, 0, 367, 363, 1, 0, 0, 0, 367, 364, 1, 0,
		0, 0, 367, 365, 1, 0, 0, 0, 367, 366, 1, 0, 0, 0, 368, 57, 1, 0, 0, 0,
		369, 372, 3, 60, 30, 0, 370, 372, 3, 62, 31, 0, 371, 369, 1, 0, 0, 0, 371,
		370, 1, 0, 0, 0, 372, 59, 1, 0, 0, 0, 373, 374, 5, 57, 0, 0, 374, 375,
		5, 22, 0, 0, 375, 382, 3, 94, 47, 0, 376, 377, 5, 57, 0, 0, 377, 378, 3,
		78, 39, 0, 378, 379, 5, 22, 0, 0, 379, 380, 3, 94, 47, 0, 380, 382, 1,
		0, 0, 0, 381, 373, 1, 0, 0, 0, 381, 376, 1, 0, 0, 0, 382, 61, 1, 0, 0,
		0, 383, 384, 5, 57, 0, 0, 384, 385, 5, 23, 0, 0, 385, 392, 3, 94, 47, 0,
		386, 387, 5, 57, 0, 0, 387, 388, 3, 78, 39, 0, 388, 389, 5, 23, 0, 0, 389,
		390, 3, 94, 47, 0, 390, 392, 1, 0, 0, 0, 391, 383, 1, 0, 0, 0, 391, 386,
		1, 0, 0, 0, 392, 63, 1, 0, 0, 0, 393, 394, 5, 57, 0, 0, 394, 395, 5, 25,
		0, 0, 395, 411, 3, 94, 47, 0, 396, 397, 5, 57, 0, 0, 397, 398, 3, 114,
		57, 0, 398, 399, 5, 25, 0, 0, 399, 400, 3, 94, 47, 0, 400, 411, 1, 0, 0,
		0, 401, 402, 5, 57, 0, 0, 402, 403, 3, 66, 33, 0, 403, 404, 3, 94, 47,
		0, 404, 411, 1, 0, 0, 0, 405, 406, 5, 57, 0, 0, 406, 407, 3, 114, 57, 0,
		407, 408, 3, 66, 33, 0, 408, 409, 3, 94, 47, 0, 409, 411, 1, 0, 0, 0, 410,
		393, 1, 0, 0, 0, 410, 396, 1, 0, 0, 0, 410, 401, 1, 0, 0, 0, 410, 405,
		1, 0, 0, 0, 411, 65, 1, 0, 0, 0, 412, 413, 7, 1, 0, 0, 413, 67, 1, 0, 0,
		0, 414, 415, 5, 2, 0, 0, 415, 416, 3, 94, 47, 0, 416, 420, 3, 54, 27, 0,
		417, 419, 3, 70, 35, 0, 418, 417, 1, 0, 0, 0, 419, 422, 1, 0, 0, 0, 420,
		418, 1, 0, 0, 0, 420, 421, 1, 0, 0, 0, 421, 424, 1, 0, 0, 0, 422, 420,
		1, 0, 0, 0, 423, 425, 3, 72, 36, 0, 424, 423, 1, 0, 0, 0, 424, 425, 1,
		0, 0, 0, 425, 69, 1, 0, 0, 0, 426, 427, 5, 3, 0, 0, 427, 428, 5, 2, 0,
		0, 428, 429, 3, 94, 47, 0, 429, 430, 3, 54, 27, 0, 430, 71, 1, 0, 0, 0,
		431, 432, 5, 3, 0, 0, 432, 433, 3, 54, 27, 0, 433, 73, 1, 0, 0, 0, 434,
		436, 5, 4, 0, 0, 435, 437, 3, 94, 47, 0, 436, 435, 1, 0, 0, 0, 436, 437,
		1, 0, 0, 0, 437, 75, 1, 0, 0, 0, 438, 439, 5, 57, 0, 0, 439, 441, 5, 46,
		0, 0, 440, 442, 3, 52, 26, 0, 441, 440, 1, 0, 0, 0, 441, 442, 1, 0, 0,
		0, 442, 443, 1, 0, 0, 0, 443, 444, 5, 47, 0, 0, 444, 77, 1, 0, 0, 0, 445,
		447, 3, 82, 41, 0, 446, 448, 3, 80, 40, 0, 447, 446, 1, 0, 0, 0, 447, 448,
		1, 0, 0, 0, 448, 452, 1, 0, 0, 0, 449, 452, 3, 90, 45, 0, 450, 452, 3,
		92, 46, 0, 451, 445, 1, 0, 0, 0, 451, 449, 1, 0, 0, 0, 451, 450, 1, 0,
		0, 0, 452, 79, 1, 0, 0, 0, 453, 454, 5, 57, 0, 0, 454, 81, 1, 0, 0, 0,
		455, 458, 3, 84, 42, 0, 456, 458, 5, 19, 0, 0, 457, 455, 1, 0, 0, 0, 457,
		456, 1, 0, 0, 0, 458, 83, 1, 0, 0, 0, 459, 462, 3, 86, 43, 0, 460, 462,
		3, 88, 44, 0, 461, 459, 1, 0, 0, 0, 461, 460, 1, 0, 0, 0, 462, 85, 1, 0,
		0, 0, 463, 464, 7, 2, 0, 0, 464, 87, 1, 0, 0, 0, 465, 466, 7, 3, 0, 0,
		466, 89, 1, 0, 0, 0, 467, 468, 5, 8, 0, 0, 468, 470, 3, 82, 41, 0, 469,
		471, 3, 80, 40, 0, 470, 469, 1, 0, 0, 0, 470, 471, 1, 0, 0, 0, 471, 475,
		1, 0, 0, 0, 472, 473, 5, 8, 0, 0, 473, 475, 3, 92, 46, 0, 474, 467, 1,
		0, 0, 0, 474, 472, 1, 0, 0, 0, 475, 91, 1, 0, 0, 0, 476, 477, 5, 20, 0,
		0, 477, 479, 3, 82, 41, 0, 478, 480, 3, 80, 40, 0, 479, 478, 1, 0, 0, 0,
		479, 480, 1, 0, 0, 0, 480, 93, 1, 0, 0, 0, 481, 482, 3, 96, 48, 0, 482,
		95, 1, 0, 0, 0, 483, 488, 3, 98, 49, 0, 484, 485, 5, 44, 0, 0, 485, 487,
		3, 98, 49, 0, 486, 484, 1, 0, 0, 0, 487, 490, 1, 0, 0, 0, 488, 486, 1,
		0, 0, 0, 488, 489, 1, 0, 0, 0, 489, 97, 1, 0, 0, 0, 490, 488, 1, 0, 0,
		0, 491, 496, 3, 100, 50, 0, 492, 493, 5, 43, 0, 0, 493, 495, 3, 100, 50,
		0, 494, 492, 1, 0, 0, 0, 495, 498, 1, 0, 0, 0, 496, 494, 1, 0, 0, 0, 496,
		497, 1, 0, 0, 0, 497, 99, 1, 0, 0, 0, 498, 496, 1, 0, 0, 0, 499, 504, 3,
		102, 51, 0, 500, 501, 7, 4, 0, 0, 501, 503, 3, 102, 51, 0, 502, 500, 1,
		0, 0, 0, 503, 506, 1, 0, 0, 0, 504, 502, 1, 0, 0, 0, 504, 505, 1, 0, 0,
		0, 505, 101, 1, 0, 0, 0, 506, 504, 1, 0, 0, 0, 507, 512, 3, 104, 52, 0,
		508, 509, 7, 5, 0, 0, 509, 511, 3, 104, 52, 0, 510, 508, 1, 0, 0, 0, 511,
		514, 1, 0, 0, 0, 512, 510, 1, 0, 0, 0, 512, 513, 1, 0, 0, 0, 513, 103,
		1, 0, 0, 0, 514, 512, 1, 0, 0, 0, 515, 520, 3, 106, 53, 0, 516, 517, 7,
		6, 0, 0, 517, 519, 3, 106, 53, 0, 518, 516, 1, 0, 0, 0, 519, 522, 1, 0,
		0, 0, 520, 518, 1, 0, 0, 0, 520, 521, 1, 0, 0, 0, 521, 105, 1, 0, 0, 0,
		522, 520, 1, 0, 0, 0, 523, 528, 3, 108, 54, 0, 524, 525, 7, 7, 0, 0, 525,
		527, 3, 108, 54, 0, 526, 524, 1, 0, 0, 0, 527, 530, 1, 0, 0, 0, 528, 526,
		1, 0, 0, 0, 528, 529, 1, 0, 0, 0, 529, 107, 1, 0, 0, 0, 530, 528, 1, 0,
		0, 0, 531, 534, 3, 110, 55, 0, 532, 533, 5, 36, 0, 0, 533, 535, 3, 108,
		54, 0, 534, 532, 1, 0, 0, 0, 534, 535, 1, 0, 0, 0, 535, 109, 1, 0, 0, 0,
		536, 537, 5, 32, 0, 0, 537, 542, 3, 110, 55, 0, 538, 539, 5, 45, 0, 0,
		539, 542, 3, 110, 55, 0, 540, 542, 3, 112, 56, 0, 541, 536, 1, 0, 0, 0,
		541, 538, 1, 0, 0, 0, 541, 540, 1, 0, 0, 0, 542, 111, 1, 0, 0, 0, 543,
		548, 3, 118, 59, 0, 544, 547, 3, 114, 57, 0, 545, 547, 3, 116, 58, 0, 546,
		544, 1, 0, 0, 0, 546, 545, 1, 0, 0, 0, 547, 550, 1, 0, 0, 0, 548, 546,
		1, 0, 0, 0, 548, 549, 1, 0, 0, 0, 549, 113, 1, 0, 0, 0, 550, 548, 1, 0,
		0, 0, 551, 552, 5, 50, 0, 0, 552, 553, 3, 94, 47, 0, 553, 554, 5, 51, 0,
		0, 554, 565, 1, 0, 0, 0, 555, 557, 5, 50, 0, 0, 556, 558, 3, 94, 47, 0,
		557, 556, 1, 0, 0, 0, 557, 558, 1, 0, 0, 0, 558, 559, 1, 0, 0, 0, 559,
		561, 5, 53, 0, 0, 560, 562, 3, 94, 47, 0, 561, 560, 1, 0, 0, 0, 561, 562,
		1, 0, 0, 0, 562, 563, 1, 0, 0, 0, 563, 565, 5, 51, 0, 0, 564, 551, 1, 0,
		0, 0, 564, 555, 1, 0, 0, 0, 565, 115, 1, 0, 0, 0, 566, 568, 5, 46, 0, 0,
		567, 569, 3, 52, 26, 0, 568, 567, 1, 0, 0, 0, 568, 569, 1, 0, 0, 0, 569,
		570, 1, 0, 0, 0, 570, 571, 5, 47, 0, 0, 571, 117, 1, 0, 0, 0, 572, 580,
		3, 122, 61, 0, 573, 580, 5, 57, 0, 0, 574, 575, 5, 46, 0, 0, 575, 576,
		3, 94, 47, 0, 576, 577, 5, 47, 0, 0, 577, 580, 1, 0, 0, 0, 578, 580, 3,
		120, 60, 0, 579, 572, 1, 0, 0, 0, 579, 573, 1, 0, 0, 0, 579, 574, 1, 0,
		0, 0, 579, 578, 1, 0, 0, 0, 580, 119, 1, 0, 0, 0, 581, 582, 3, 78, 39,
		0, 582, 583, 5, 46, 0, 0, 583, 584, 3, 94, 47, 0, 584, 585, 5, 47, 0, 0,
		585, 121, 1, 0, 0, 0, 586, 590, 3, 124, 62, 0, 587, 590, 5, 56, 0, 0, 588,
		590, 3, 126, 63, 0, 589, 586, 1, 0, 0, 0, 589, 587, 1, 0, 0, 0, 589, 588,
		1, 0, 0, 0, 590, 123, 1, 0, 0, 0, 591, 594, 7, 8, 0, 0, 592, 593, 4, 62,
		0, 0, 593, 595, 5, 57, 0, 0, 594, 592, 1, 0, 0, 0, 594, 595, 1, 0, 0, 0,
		595, 125, 1, 0, 0, 0, 596, 598, 5, 50, 0, 0, 597, 599, 3, 128, 64, 0, 598,
		597, 1, 0, 0, 0, 598, 599, 1, 0, 0, 0, 599, 600, 1, 0, 0, 0, 600, 601,
		5, 51, 0, 0, 601, 127, 1, 0, 0, 0, 602, 607, 3, 94, 47, 0, 603, 604, 5,
		52, 0, 0, 604, 606, 3, 94, 47, 0, 605, 603, 1, 0, 0, 0, 606, 609, 1, 0,
		0, 0, 607, 605, 1, 0, 0, 0, 607, 608, 1, 0, 0, 0, 608, 611, 1, 0, 0, 0,
		609, 607, 1, 0, 0, 0, 610, 612, 5, 52, 0, 0, 611, 610, 1, 0, 0, 0, 611,
		612, 1, 0, 0, 0, 612, 129, 1, 0, 0, 0, 69, 133, 141, 146, 150, 154, 163,
		167, 173, 179, 187, 191, 202, 216, 231, 235, 237, 245, 250, 254, 264, 268,
		279, 284, 290, 307, 314, 318, 329, 333, 337, 346, 350, 356, 367, 371, 381,
		391, 410, 420, 424, 436, 441, 447, 451, 457, 461, 470, 474, 479, 488, 496,
		504, 512, 520, 528, 534, 541, 546, 548, 557, 561, 564, 568, 579, 589, 594,
		598, 607, 611,
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
	ArcParserSEQUENCE            = 5
	ArcParserSTAGE               = 6
	ArcParserNEXT                = 7
	ArcParserCHAN                = 8
	ArcParserI8                  = 9
	ArcParserI16                 = 10
	ArcParserI32                 = 11
	ArcParserI64                 = 12
	ArcParserU8                  = 13
	ArcParserU16                 = 14
	ArcParserU32                 = 15
	ArcParserU64                 = 16
	ArcParserF32                 = 17
	ArcParserF64                 = 18
	ArcParserSTR                 = 19
	ArcParserSERIES              = 20
	ArcParserARROW               = 21
	ArcParserDECLARE             = 22
	ArcParserSTATE_DECLARE       = 23
	ArcParserTRANSITION          = 24
	ArcParserASSIGN              = 25
	ArcParserPLUS_ASSIGN         = 26
	ArcParserMINUS_ASSIGN        = 27
	ArcParserSTAR_ASSIGN         = 28
	ArcParserSLASH_ASSIGN        = 29
	ArcParserPERCENT_ASSIGN      = 30
	ArcParserPLUS                = 31
	ArcParserMINUS               = 32
	ArcParserSTAR                = 33
	ArcParserSLASH               = 34
	ArcParserPERCENT             = 35
	ArcParserCARET               = 36
	ArcParserEQ                  = 37
	ArcParserNEQ                 = 38
	ArcParserLT                  = 39
	ArcParserGT                  = 40
	ArcParserLEQ                 = 41
	ArcParserGEQ                 = 42
	ArcParserAND                 = 43
	ArcParserOR                  = 44
	ArcParserNOT                 = 45
	ArcParserLPAREN              = 46
	ArcParserRPAREN              = 47
	ArcParserLBRACE              = 48
	ArcParserRBRACE              = 49
	ArcParserLBRACKET            = 50
	ArcParserRBRACKET            = 51
	ArcParserCOMMA               = 52
	ArcParserCOLON               = 53
	ArcParserINTEGER_LITERAL     = 54
	ArcParserFLOAT_LITERAL       = 55
	ArcParserSTR_LITERAL         = 56
	ArcParserIDENTIFIER          = 57
	ArcParserSINGLE_LINE_COMMENT = 58
	ArcParserMULTI_LINE_COMMENT  = 59
	ArcParserWS                  = 60
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
	ArcParserRULE_stageDeclaration         = 11
	ArcParserRULE_stageBody                = 12
	ArcParserRULE_stageItem                = 13
	ArcParserRULE_flowStatement            = 14
	ArcParserRULE_flowOperator             = 15
	ArcParserRULE_routingTable             = 16
	ArcParserRULE_routingEntry             = 17
	ArcParserRULE_flowNode                 = 18
	ArcParserRULE_identifier               = 19
	ArcParserRULE_function                 = 20
	ArcParserRULE_configValues             = 21
	ArcParserRULE_namedConfigValues        = 22
	ArcParserRULE_namedConfigValue         = 23
	ArcParserRULE_anonymousConfigValues    = 24
	ArcParserRULE_arguments                = 25
	ArcParserRULE_argumentList             = 26
	ArcParserRULE_block                    = 27
	ArcParserRULE_statement                = 28
	ArcParserRULE_variableDeclaration      = 29
	ArcParserRULE_localVariable            = 30
	ArcParserRULE_statefulVariable         = 31
	ArcParserRULE_assignment               = 32
	ArcParserRULE_compoundOp               = 33
	ArcParserRULE_ifStatement              = 34
	ArcParserRULE_elseIfClause             = 35
	ArcParserRULE_elseClause               = 36
	ArcParserRULE_returnStatement          = 37
	ArcParserRULE_functionCall             = 38
	ArcParserRULE_type                     = 39
	ArcParserRULE_unitSuffix               = 40
	ArcParserRULE_primitiveType            = 41
	ArcParserRULE_numericType              = 42
	ArcParserRULE_integerType              = 43
	ArcParserRULE_floatType                = 44
	ArcParserRULE_channelType              = 45
	ArcParserRULE_seriesType               = 46
	ArcParserRULE_expression               = 47
	ArcParserRULE_logicalOrExpression      = 48
	ArcParserRULE_logicalAndExpression     = 49
	ArcParserRULE_equalityExpression       = 50
	ArcParserRULE_relationalExpression     = 51
	ArcParserRULE_additiveExpression       = 52
	ArcParserRULE_multiplicativeExpression = 53
	ArcParserRULE_powerExpression          = 54
	ArcParserRULE_unaryExpression          = 55
	ArcParserRULE_postfixExpression        = 56
	ArcParserRULE_indexOrSlice             = 57
	ArcParserRULE_functionCallSuffix       = 58
	ArcParserRULE_primaryExpression        = 59
	ArcParserRULE_typeCast                 = 60
	ArcParserRULE_literal                  = 61
	ArcParserRULE_numericLiteral           = 62
	ArcParserRULE_seriesLiteral            = 63
	ArcParserRULE_expressionList           = 64
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
	p.SetState(133)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&271728909939113890) != 0 {
		{
			p.SetState(130)
			p.TopLevelItem()
		}

		p.SetState(135)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(136)
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
	p.SetState(141)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case ArcParserFUNC:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(138)
			p.FunctionDeclaration()
		}

	case ArcParserNEXT, ArcParserCHAN, ArcParserI8, ArcParserI16, ArcParserI32, ArcParserI64, ArcParserU8, ArcParserU16, ArcParserU32, ArcParserU64, ArcParserF32, ArcParserF64, ArcParserSTR, ArcParserSERIES, ArcParserMINUS, ArcParserNOT, ArcParserLPAREN, ArcParserLBRACE, ArcParserLBRACKET, ArcParserINTEGER_LITERAL, ArcParserFLOAT_LITERAL, ArcParserSTR_LITERAL, ArcParserIDENTIFIER:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(139)
			p.FlowStatement()
		}

	case ArcParserSEQUENCE:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(140)
			p.SequenceDeclaration()
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
		p.SetState(143)
		p.Match(ArcParserFUNC)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(144)
		p.Match(ArcParserIDENTIFIER)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(146)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ArcParserLBRACE {
		{
			p.SetState(145)
			p.ConfigBlock()
		}

	}
	{
		p.SetState(148)
		p.Match(ArcParserLPAREN)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(150)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ArcParserIDENTIFIER {
		{
			p.SetState(149)
			p.InputList()
		}

	}
	{
		p.SetState(152)
		p.Match(ArcParserRPAREN)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(154)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&144185556822130432) != 0 {
		{
			p.SetState(153)
			p.OutputType()
		}

	}
	{
		p.SetState(156)
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

	var _alt int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(158)
		p.Input()
	}
	p.SetState(163)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 5, p.GetParserRuleContext())
	if p.HasError() {
		goto errorExit
	}
	for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
		if _alt == 1 {
			{
				p.SetState(159)
				p.Match(ArcParserCOMMA)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}
			{
				p.SetState(160)
				p.Input()
			}

		}
		p.SetState(165)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 5, p.GetParserRuleContext())
		if p.HasError() {
			goto errorExit
		}
	}
	p.SetState(167)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ArcParserCOMMA {
		{
			p.SetState(166)
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

func (p *ArcParser) Input() (localctx IInputContext) {
	localctx = NewInputContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 8, ArcParserRULE_input)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(169)
		p.Match(ArcParserIDENTIFIER)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(170)
		p.Type_()
	}
	p.SetState(173)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ArcParserASSIGN {
		{
			p.SetState(171)
			p.Match(ArcParserASSIGN)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(172)
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
	p.SetState(179)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case ArcParserCHAN, ArcParserI8, ArcParserI16, ArcParserI32, ArcParserI64, ArcParserU8, ArcParserU16, ArcParserU32, ArcParserU64, ArcParserF32, ArcParserF64, ArcParserSTR, ArcParserSERIES:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(175)
			p.Type_()
		}

	case ArcParserIDENTIFIER:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(176)
			p.Match(ArcParserIDENTIFIER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(177)
			p.Type_()
		}

	case ArcParserLPAREN:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(178)
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

	var _alt int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(181)
		p.Match(ArcParserLPAREN)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(182)
		p.NamedOutput()
	}
	p.SetState(187)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 9, p.GetParserRuleContext())
	if p.HasError() {
		goto errorExit
	}
	for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
		if _alt == 1 {
			{
				p.SetState(183)
				p.Match(ArcParserCOMMA)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}
			{
				p.SetState(184)
				p.NamedOutput()
			}

		}
		p.SetState(189)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 9, p.GetParserRuleContext())
		if p.HasError() {
			goto errorExit
		}
	}
	p.SetState(191)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ArcParserCOMMA {
		{
			p.SetState(190)
			p.Match(ArcParserCOMMA)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	}
	{
		p.SetState(193)
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
		p.SetState(198)
		p.Match(ArcParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(202)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == ArcParserIDENTIFIER {
		{
			p.SetState(199)
			p.Config()
		}

		p.SetState(204)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(205)
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
		p.SetState(207)
		p.Match(ArcParserIDENTIFIER)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(208)
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
	RBRACE() antlr.TerminalNode
	AllStageDeclaration() []IStageDeclarationContext
	StageDeclaration(i int) IStageDeclarationContext

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

func (s *SequenceDeclarationContext) RBRACE() antlr.TerminalNode {
	return s.GetToken(ArcParserRBRACE, 0)
}

func (s *SequenceDeclarationContext) AllStageDeclaration() []IStageDeclarationContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IStageDeclarationContext); ok {
			len++
		}
	}

	tst := make([]IStageDeclarationContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IStageDeclarationContext); ok {
			tst[i] = t.(IStageDeclarationContext)
			i++
		}
	}

	return tst
}

func (s *SequenceDeclarationContext) StageDeclaration(i int) IStageDeclarationContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IStageDeclarationContext); ok {
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

	return t.(IStageDeclarationContext)
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
		p.SetState(210)
		p.Match(ArcParserSEQUENCE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(211)
		p.Match(ArcParserIDENTIFIER)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(212)
		p.Match(ArcParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(216)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == ArcParserSTAGE {
		{
			p.SetState(213)
			p.StageDeclaration()
		}

		p.SetState(218)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(219)
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
	p.EnterRule(localctx, 22, ArcParserRULE_stageDeclaration)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(221)
		p.Match(ArcParserSTAGE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(222)
		p.Match(ArcParserIDENTIFIER)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(223)
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

func (p *ArcParser) StageBody() (localctx IStageBodyContext) {
	localctx = NewStageBodyContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 24, ArcParserRULE_stageBody)
	var _la int

	var _alt int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(225)
		p.Match(ArcParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(237)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&271728909939113856) != 0 {
		{
			p.SetState(226)
			p.StageItem()
		}
		p.SetState(231)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 13, p.GetParserRuleContext())
		if p.HasError() {
			goto errorExit
		}
		for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
			if _alt == 1 {
				{
					p.SetState(227)
					p.Match(ArcParserCOMMA)
					if p.HasError() {
						// Recognition error - abort rule
						goto errorExit
					}
				}
				{
					p.SetState(228)
					p.StageItem()
				}

			}
			p.SetState(233)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 13, p.GetParserRuleContext())
			if p.HasError() {
				goto errorExit
			}
		}
		p.SetState(235)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		if _la == ArcParserCOMMA {
			{
				p.SetState(234)
				p.Match(ArcParserCOMMA)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}

		}

	}
	{
		p.SetState(239)
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
	p.EnterRule(localctx, 26, ArcParserRULE_stageItem)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(241)
		p.FlowStatement()
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

func (p *ArcParser) FlowStatement() (localctx IFlowStatementContext) {
	localctx = NewFlowStatementContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 28, ArcParserRULE_flowStatement)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(245)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case ArcParserLBRACE:
		{
			p.SetState(243)
			p.RoutingTable()
		}

	case ArcParserNEXT, ArcParserCHAN, ArcParserI8, ArcParserI16, ArcParserI32, ArcParserI64, ArcParserU8, ArcParserU16, ArcParserU32, ArcParserU64, ArcParserF32, ArcParserF64, ArcParserSTR, ArcParserSERIES, ArcParserMINUS, ArcParserNOT, ArcParserLPAREN, ArcParserLBRACKET, ArcParserINTEGER_LITERAL, ArcParserFLOAT_LITERAL, ArcParserSTR_LITERAL, ArcParserIDENTIFIER:
		{
			p.SetState(244)
			p.FlowNode()
		}

	default:
		p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
		goto errorExit
	}
	p.SetState(252)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for ok := true; ok; ok = _la == ArcParserARROW || _la == ArcParserTRANSITION {
		{
			p.SetState(247)
			p.FlowOperator()
		}
		p.SetState(250)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}

		switch p.GetTokenStream().LA(1) {
		case ArcParserLBRACE:
			{
				p.SetState(248)
				p.RoutingTable()
			}

		case ArcParserNEXT, ArcParserCHAN, ArcParserI8, ArcParserI16, ArcParserI32, ArcParserI64, ArcParserU8, ArcParserU16, ArcParserU32, ArcParserU64, ArcParserF32, ArcParserF64, ArcParserSTR, ArcParserSERIES, ArcParserMINUS, ArcParserNOT, ArcParserLPAREN, ArcParserLBRACKET, ArcParserINTEGER_LITERAL, ArcParserFLOAT_LITERAL, ArcParserSTR_LITERAL, ArcParserIDENTIFIER:
			{
				p.SetState(249)
				p.FlowNode()
			}

		default:
			p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
			goto errorExit
		}

		p.SetState(254)
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

func (p *ArcParser) FlowOperator() (localctx IFlowOperatorContext) {
	localctx = NewFlowOperatorContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 30, ArcParserRULE_flowOperator)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(256)
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

func (p *ArcParser) RoutingTable() (localctx IRoutingTableContext) {
	localctx = NewRoutingTableContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 32, ArcParserRULE_routingTable)
	var _la int

	var _alt int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(258)
		p.Match(ArcParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(259)
		p.RoutingEntry()
	}
	p.SetState(264)
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
				p.SetState(260)
				p.Match(ArcParserCOMMA)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}
			{
				p.SetState(261)
				p.RoutingEntry()
			}

		}
		p.SetState(266)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 19, p.GetParserRuleContext())
		if p.HasError() {
			goto errorExit
		}
	}
	p.SetState(268)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ArcParserCOMMA {
		{
			p.SetState(267)
			p.Match(ArcParserCOMMA)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	}
	{
		p.SetState(270)
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
	p.EnterRule(localctx, 34, ArcParserRULE_routingEntry)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(272)
		p.Match(ArcParserIDENTIFIER)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(273)
		p.Match(ArcParserCOLON)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(274)
		p.FlowNode()
	}
	p.SetState(279)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == ArcParserARROW {
		{
			p.SetState(275)
			p.Match(ArcParserARROW)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(276)
			p.FlowNode()
		}

		p.SetState(281)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	p.SetState(284)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ArcParserCOLON {
		{
			p.SetState(282)
			p.Match(ArcParserCOLON)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(283)
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

func (p *ArcParser) FlowNode() (localctx IFlowNodeContext) {
	localctx = NewFlowNodeContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 36, ArcParserRULE_flowNode)
	p.SetState(290)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 23, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(286)
			p.Identifier()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(287)
			p.Function()
		}

	case 3:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(288)
			p.Expression()
		}

	case 4:
		p.EnterOuterAlt(localctx, 4)
		{
			p.SetState(289)
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

func (p *ArcParser) Identifier() (localctx IIdentifierContext) {
	localctx = NewIdentifierContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 38, ArcParserRULE_identifier)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(292)
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
	p.EnterRule(localctx, 40, ArcParserRULE_function)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(294)
		p.Match(ArcParserIDENTIFIER)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(295)
		p.ConfigValues()
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
	p.EnterRule(localctx, 42, ArcParserRULE_configValues)
	p.SetState(307)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 24, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(297)
			p.Match(ArcParserLBRACE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(298)
			p.Match(ArcParserRBRACE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(299)
			p.Match(ArcParserLBRACE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(300)
			p.NamedConfigValues()
		}
		{
			p.SetState(301)
			p.Match(ArcParserRBRACE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 3:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(303)
			p.Match(ArcParserLBRACE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(304)
			p.AnonymousConfigValues()
		}
		{
			p.SetState(305)
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
	p.EnterRule(localctx, 44, ArcParserRULE_namedConfigValues)
	var _la int

	var _alt int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(309)
		p.NamedConfigValue()
	}
	p.SetState(314)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 25, p.GetParserRuleContext())
	if p.HasError() {
		goto errorExit
	}
	for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
		if _alt == 1 {
			{
				p.SetState(310)
				p.Match(ArcParserCOMMA)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}
			{
				p.SetState(311)
				p.NamedConfigValue()
			}

		}
		p.SetState(316)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 25, p.GetParserRuleContext())
		if p.HasError() {
			goto errorExit
		}
	}
	p.SetState(318)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ArcParserCOMMA {
		{
			p.SetState(317)
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
	p.EnterRule(localctx, 46, ArcParserRULE_namedConfigValue)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(320)
		p.Match(ArcParserIDENTIFIER)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(321)
		p.Match(ArcParserASSIGN)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(322)
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
	p.EnterRule(localctx, 48, ArcParserRULE_anonymousConfigValues)
	var _la int

	var _alt int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(324)
		p.Expression()
	}
	p.SetState(329)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 27, p.GetParserRuleContext())
	if p.HasError() {
		goto errorExit
	}
	for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
		if _alt == 1 {
			{
				p.SetState(325)
				p.Match(ArcParserCOMMA)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}
			{
				p.SetState(326)
				p.Expression()
			}

		}
		p.SetState(331)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 27, p.GetParserRuleContext())
		if p.HasError() {
			goto errorExit
		}
	}
	p.SetState(333)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ArcParserCOMMA {
		{
			p.SetState(332)
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
	p.EnterRule(localctx, 50, ArcParserRULE_arguments)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(335)
		p.Match(ArcParserLPAREN)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(337)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&271447434962403072) != 0 {
		{
			p.SetState(336)
			p.ArgumentList()
		}

	}
	{
		p.SetState(339)
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
	p.EnterRule(localctx, 52, ArcParserRULE_argumentList)
	var _la int

	var _alt int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(341)
		p.Expression()
	}
	p.SetState(346)
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
			{
				p.SetState(342)
				p.Match(ArcParserCOMMA)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}
			{
				p.SetState(343)
				p.Expression()
			}

		}
		p.SetState(348)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 30, p.GetParserRuleContext())
		if p.HasError() {
			goto errorExit
		}
	}
	p.SetState(350)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ArcParserCOMMA {
		{
			p.SetState(349)
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
	p.EnterRule(localctx, 54, ArcParserRULE_block)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(352)
		p.Match(ArcParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(356)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&271447434962403092) != 0 {
		{
			p.SetState(353)
			p.Statement()
		}

		p.SetState(358)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(359)
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
	p.EnterRule(localctx, 56, ArcParserRULE_statement)
	p.SetState(367)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 33, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(361)
			p.VariableDeclaration()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(362)
			p.Assignment()
		}

	case 3:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(363)
			p.IfStatement()
		}

	case 4:
		p.EnterOuterAlt(localctx, 4)
		{
			p.SetState(364)
			p.ReturnStatement()
		}

	case 5:
		p.EnterOuterAlt(localctx, 5)
		{
			p.SetState(365)
			p.FunctionCall()
		}

	case 6:
		p.EnterOuterAlt(localctx, 6)
		{
			p.SetState(366)
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
	p.EnterRule(localctx, 58, ArcParserRULE_variableDeclaration)
	p.SetState(371)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 34, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(369)
			p.LocalVariable()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(370)
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
	p.EnterRule(localctx, 60, ArcParserRULE_localVariable)
	p.SetState(381)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 35, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(373)
			p.Match(ArcParserIDENTIFIER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(374)
			p.Match(ArcParserDECLARE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(375)
			p.Expression()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(376)
			p.Match(ArcParserIDENTIFIER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(377)
			p.Type_()
		}
		{
			p.SetState(378)
			p.Match(ArcParserDECLARE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(379)
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
	p.EnterRule(localctx, 62, ArcParserRULE_statefulVariable)
	p.SetState(391)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 36, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(383)
			p.Match(ArcParserIDENTIFIER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(384)
			p.Match(ArcParserSTATE_DECLARE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(385)
			p.Expression()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(386)
			p.Match(ArcParserIDENTIFIER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(387)
			p.Type_()
		}
		{
			p.SetState(388)
			p.Match(ArcParserSTATE_DECLARE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(389)
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

func (p *ArcParser) Assignment() (localctx IAssignmentContext) {
	localctx = NewAssignmentContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 64, ArcParserRULE_assignment)
	p.SetState(410)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 37, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(393)
			p.Match(ArcParserIDENTIFIER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(394)
			p.Match(ArcParserASSIGN)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(395)
			p.Expression()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(396)
			p.Match(ArcParserIDENTIFIER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(397)
			p.IndexOrSlice()
		}
		{
			p.SetState(398)
			p.Match(ArcParserASSIGN)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(399)
			p.Expression()
		}

	case 3:
		p.EnterOuterAlt(localctx, 3)
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
			p.CompoundOp()
		}
		{
			p.SetState(403)
			p.Expression()
		}

	case 4:
		p.EnterOuterAlt(localctx, 4)
		{
			p.SetState(405)
			p.Match(ArcParserIDENTIFIER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(406)
			p.IndexOrSlice()
		}
		{
			p.SetState(407)
			p.CompoundOp()
		}
		{
			p.SetState(408)
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

func (p *ArcParser) CompoundOp() (localctx ICompoundOpContext) {
	localctx = NewCompoundOpContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 66, ArcParserRULE_compoundOp)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(412)
		_la = p.GetTokenStream().LA(1)

		if !((int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&2080374784) != 0) {
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

func (p *ArcParser) IfStatement() (localctx IIfStatementContext) {
	localctx = NewIfStatementContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 68, ArcParserRULE_ifStatement)
	var _la int

	var _alt int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(414)
		p.Match(ArcParserIF)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(415)
		p.Expression()
	}
	{
		p.SetState(416)
		p.Block()
	}
	p.SetState(420)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 38, p.GetParserRuleContext())
	if p.HasError() {
		goto errorExit
	}
	for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
		if _alt == 1 {
			{
				p.SetState(417)
				p.ElseIfClause()
			}

		}
		p.SetState(422)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 38, p.GetParserRuleContext())
		if p.HasError() {
			goto errorExit
		}
	}
	p.SetState(424)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ArcParserELSE {
		{
			p.SetState(423)
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
	p.EnterRule(localctx, 70, ArcParserRULE_elseIfClause)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(426)
		p.Match(ArcParserELSE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(427)
		p.Match(ArcParserIF)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(428)
		p.Expression()
	}
	{
		p.SetState(429)
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
	p.EnterRule(localctx, 72, ArcParserRULE_elseClause)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(431)
		p.Match(ArcParserELSE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(432)
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
	p.EnterRule(localctx, 74, ArcParserRULE_returnStatement)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(434)
		p.Match(ArcParserRETURN)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(436)
	p.GetErrorHandler().Sync(p)

	if p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 40, p.GetParserRuleContext()) == 1 {
		{
			p.SetState(435)
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
	p.EnterRule(localctx, 76, ArcParserRULE_functionCall)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(438)
		p.Match(ArcParserIDENTIFIER)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(439)
		p.Match(ArcParserLPAREN)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(441)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&271447434962403072) != 0 {
		{
			p.SetState(440)
			p.ArgumentList()
		}

	}
	{
		p.SetState(443)
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

func (p *ArcParser) Type_() (localctx ITypeContext) {
	localctx = NewTypeContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 78, ArcParserRULE_type)
	p.SetState(451)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case ArcParserI8, ArcParserI16, ArcParserI32, ArcParserI64, ArcParserU8, ArcParserU16, ArcParserU32, ArcParserU64, ArcParserF32, ArcParserF64, ArcParserSTR:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(445)
			p.PrimitiveType()
		}
		p.SetState(447)
		p.GetErrorHandler().Sync(p)

		if p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 42, p.GetParserRuleContext()) == 1 {
			{
				p.SetState(446)
				p.UnitSuffix()
			}

		} else if p.HasError() { // JIM
			goto errorExit
		}

	case ArcParserCHAN:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(449)
			p.ChannelType()
		}

	case ArcParserSERIES:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(450)
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

func (p *ArcParser) UnitSuffix() (localctx IUnitSuffixContext) {
	localctx = NewUnitSuffixContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 80, ArcParserRULE_unitSuffix)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(453)
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

func (p *ArcParser) PrimitiveType() (localctx IPrimitiveTypeContext) {
	localctx = NewPrimitiveTypeContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 82, ArcParserRULE_primitiveType)
	p.SetState(457)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case ArcParserI8, ArcParserI16, ArcParserI32, ArcParserI64, ArcParserU8, ArcParserU16, ArcParserU32, ArcParserU64, ArcParserF32, ArcParserF64:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(455)
			p.NumericType()
		}

	case ArcParserSTR:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(456)
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

func (p *ArcParser) NumericType() (localctx INumericTypeContext) {
	localctx = NewNumericTypeContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 84, ArcParserRULE_numericType)
	p.SetState(461)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case ArcParserI8, ArcParserI16, ArcParserI32, ArcParserI64, ArcParserU8, ArcParserU16, ArcParserU32, ArcParserU64:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(459)
			p.IntegerType()
		}

	case ArcParserF32, ArcParserF64:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(460)
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

func (p *ArcParser) IntegerType() (localctx IIntegerTypeContext) {
	localctx = NewIntegerTypeContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 86, ArcParserRULE_integerType)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(463)
		_la = p.GetTokenStream().LA(1)

		if !((int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&130560) != 0) {
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
	p.EnterRule(localctx, 88, ArcParserRULE_floatType)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(465)
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

func (p *ArcParser) ChannelType() (localctx IChannelTypeContext) {
	localctx = NewChannelTypeContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 90, ArcParserRULE_channelType)
	p.SetState(474)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 47, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(467)
			p.Match(ArcParserCHAN)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(468)
			p.PrimitiveType()
		}
		p.SetState(470)
		p.GetErrorHandler().Sync(p)

		if p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 46, p.GetParserRuleContext()) == 1 {
			{
				p.SetState(469)
				p.UnitSuffix()
			}

		} else if p.HasError() { // JIM
			goto errorExit
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(472)
			p.Match(ArcParserCHAN)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(473)
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

func (p *ArcParser) SeriesType() (localctx ISeriesTypeContext) {
	localctx = NewSeriesTypeContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 92, ArcParserRULE_seriesType)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(476)
		p.Match(ArcParserSERIES)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(477)
		p.PrimitiveType()
	}
	p.SetState(479)
	p.GetErrorHandler().Sync(p)

	if p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 48, p.GetParserRuleContext()) == 1 {
		{
			p.SetState(478)
			p.UnitSuffix()
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
	p.EnterRule(localctx, 94, ArcParserRULE_expression)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(481)
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
	p.EnterRule(localctx, 96, ArcParserRULE_logicalOrExpression)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(483)
		p.LogicalAndExpression()
	}
	p.SetState(488)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == ArcParserOR {
		{
			p.SetState(484)
			p.Match(ArcParserOR)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(485)
			p.LogicalAndExpression()
		}

		p.SetState(490)
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
	p.EnterRule(localctx, 98, ArcParserRULE_logicalAndExpression)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(491)
		p.EqualityExpression()
	}
	p.SetState(496)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == ArcParserAND {
		{
			p.SetState(492)
			p.Match(ArcParserAND)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(493)
			p.EqualityExpression()
		}

		p.SetState(498)
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
	p.EnterRule(localctx, 100, ArcParserRULE_equalityExpression)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(499)
		p.RelationalExpression()
	}
	p.SetState(504)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == ArcParserEQ || _la == ArcParserNEQ {
		{
			p.SetState(500)
			_la = p.GetTokenStream().LA(1)

			if !(_la == ArcParserEQ || _la == ArcParserNEQ) {
				p.GetErrorHandler().RecoverInline(p)
			} else {
				p.GetErrorHandler().ReportMatch(p)
				p.Consume()
			}
		}
		{
			p.SetState(501)
			p.RelationalExpression()
		}

		p.SetState(506)
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
	p.EnterRule(localctx, 102, ArcParserRULE_relationalExpression)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(507)
		p.AdditiveExpression()
	}
	p.SetState(512)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&8246337208320) != 0 {
		{
			p.SetState(508)
			_la = p.GetTokenStream().LA(1)

			if !((int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&8246337208320) != 0) {
				p.GetErrorHandler().RecoverInline(p)
			} else {
				p.GetErrorHandler().ReportMatch(p)
				p.Consume()
			}
		}
		{
			p.SetState(509)
			p.AdditiveExpression()
		}

		p.SetState(514)
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
	p.EnterRule(localctx, 104, ArcParserRULE_additiveExpression)
	var _la int

	var _alt int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(515)
		p.MultiplicativeExpression()
	}
	p.SetState(520)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 53, p.GetParserRuleContext())
	if p.HasError() {
		goto errorExit
	}
	for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
		if _alt == 1 {
			{
				p.SetState(516)
				_la = p.GetTokenStream().LA(1)

				if !(_la == ArcParserPLUS || _la == ArcParserMINUS) {
					p.GetErrorHandler().RecoverInline(p)
				} else {
					p.GetErrorHandler().ReportMatch(p)
					p.Consume()
				}
			}
			{
				p.SetState(517)
				p.MultiplicativeExpression()
			}

		}
		p.SetState(522)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 53, p.GetParserRuleContext())
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
	p.EnterRule(localctx, 106, ArcParserRULE_multiplicativeExpression)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(523)
		p.PowerExpression()
	}
	p.SetState(528)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&60129542144) != 0 {
		{
			p.SetState(524)
			_la = p.GetTokenStream().LA(1)

			if !((int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&60129542144) != 0) {
				p.GetErrorHandler().RecoverInline(p)
			} else {
				p.GetErrorHandler().ReportMatch(p)
				p.Consume()
			}
		}
		{
			p.SetState(525)
			p.PowerExpression()
		}

		p.SetState(530)
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
	p.EnterRule(localctx, 108, ArcParserRULE_powerExpression)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(531)
		p.UnaryExpression()
	}
	p.SetState(534)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ArcParserCARET {
		{
			p.SetState(532)
			p.Match(ArcParserCARET)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(533)
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

func (p *ArcParser) UnaryExpression() (localctx IUnaryExpressionContext) {
	localctx = NewUnaryExpressionContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 110, ArcParserRULE_unaryExpression)
	p.SetState(541)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case ArcParserMINUS:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(536)
			p.Match(ArcParserMINUS)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(537)
			p.UnaryExpression()
		}

	case ArcParserNOT:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(538)
			p.Match(ArcParserNOT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(539)
			p.UnaryExpression()
		}

	case ArcParserCHAN, ArcParserI8, ArcParserI16, ArcParserI32, ArcParserI64, ArcParserU8, ArcParserU16, ArcParserU32, ArcParserU64, ArcParserF32, ArcParserF64, ArcParserSTR, ArcParserSERIES, ArcParserLPAREN, ArcParserLBRACKET, ArcParserINTEGER_LITERAL, ArcParserFLOAT_LITERAL, ArcParserSTR_LITERAL, ArcParserIDENTIFIER:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(540)
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
	p.EnterRule(localctx, 112, ArcParserRULE_postfixExpression)
	var _alt int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(543)
		p.PrimaryExpression()
	}
	p.SetState(548)
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
			p.SetState(546)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}

			switch p.GetTokenStream().LA(1) {
			case ArcParserLBRACKET:
				{
					p.SetState(544)
					p.IndexOrSlice()
				}

			case ArcParserLPAREN:
				{
					p.SetState(545)
					p.FunctionCallSuffix()
				}

			default:
				p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
				goto errorExit
			}

		}
		p.SetState(550)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 58, p.GetParserRuleContext())
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
	p.EnterRule(localctx, 114, ArcParserRULE_indexOrSlice)
	var _la int

	p.SetState(564)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 61, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(551)
			p.Match(ArcParserLBRACKET)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(552)
			p.Expression()
		}
		{
			p.SetState(553)
			p.Match(ArcParserRBRACKET)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(555)
			p.Match(ArcParserLBRACKET)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		p.SetState(557)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		if (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&271447434962403072) != 0 {
			{
				p.SetState(556)
				p.Expression()
			}

		}
		{
			p.SetState(559)
			p.Match(ArcParserCOLON)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		p.SetState(561)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		if (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&271447434962403072) != 0 {
			{
				p.SetState(560)
				p.Expression()
			}

		}
		{
			p.SetState(563)
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
	p.EnterRule(localctx, 116, ArcParserRULE_functionCallSuffix)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(566)
		p.Match(ArcParserLPAREN)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(568)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&271447434962403072) != 0 {
		{
			p.SetState(567)
			p.ArgumentList()
		}

	}
	{
		p.SetState(570)
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
	p.EnterRule(localctx, 118, ArcParserRULE_primaryExpression)
	p.SetState(579)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case ArcParserLBRACKET, ArcParserINTEGER_LITERAL, ArcParserFLOAT_LITERAL, ArcParserSTR_LITERAL:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(572)
			p.Literal()
		}

	case ArcParserIDENTIFIER:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(573)
			p.Match(ArcParserIDENTIFIER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case ArcParserLPAREN:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(574)
			p.Match(ArcParserLPAREN)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(575)
			p.Expression()
		}
		{
			p.SetState(576)
			p.Match(ArcParserRPAREN)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case ArcParserCHAN, ArcParserI8, ArcParserI16, ArcParserI32, ArcParserI64, ArcParserU8, ArcParserU16, ArcParserU32, ArcParserU64, ArcParserF32, ArcParserF64, ArcParserSTR, ArcParserSERIES:
		p.EnterOuterAlt(localctx, 4)
		{
			p.SetState(578)
			p.TypeCast()
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
	p.EnterRule(localctx, 120, ArcParserRULE_typeCast)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(581)
		p.Type_()
	}
	{
		p.SetState(582)
		p.Match(ArcParserLPAREN)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(583)
		p.Expression()
	}
	{
		p.SetState(584)
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
	p.EnterRule(localctx, 122, ArcParserRULE_literal)
	p.SetState(589)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case ArcParserINTEGER_LITERAL, ArcParserFLOAT_LITERAL:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(586)
			p.NumericLiteral()
		}

	case ArcParserSTR_LITERAL:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(587)
			p.Match(ArcParserSTR_LITERAL)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case ArcParserLBRACKET:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(588)
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

func (p *ArcParser) NumericLiteral() (localctx INumericLiteralContext) {
	localctx = NewNumericLiteralContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 124, ArcParserRULE_numericLiteral)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(591)
		_la = p.GetTokenStream().LA(1)

		if !(_la == ArcParserINTEGER_LITERAL || _la == ArcParserFLOAT_LITERAL) {
			p.GetErrorHandler().RecoverInline(p)
		} else {
			p.GetErrorHandler().ReportMatch(p)
			p.Consume()
		}
	}
	p.SetState(594)
	p.GetErrorHandler().Sync(p)

	if p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 65, p.GetParserRuleContext()) == 1 {
		p.SetState(592)

		if !(p.TokensAdjacent(p.GetTokenStream().LT(-1), p.GetTokenStream().LT(1))) {
			p.SetError(antlr.NewFailedPredicateException(p, "p.TokensAdjacent(p.GetTokenStream().LT(-1), p.GetTokenStream().LT(1))", ""))
			goto errorExit
		}
		{
			p.SetState(593)
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

func (p *ArcParser) SeriesLiteral() (localctx ISeriesLiteralContext) {
	localctx = NewSeriesLiteralContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 126, ArcParserRULE_seriesLiteral)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(596)
		p.Match(ArcParserLBRACKET)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(598)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&271447434962403072) != 0 {
		{
			p.SetState(597)
			p.ExpressionList()
		}

	}
	{
		p.SetState(600)
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
	p.EnterRule(localctx, 128, ArcParserRULE_expressionList)
	var _la int

	var _alt int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(602)
		p.Expression()
	}
	p.SetState(607)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 67, p.GetParserRuleContext())
	if p.HasError() {
		goto errorExit
	}
	for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
		if _alt == 1 {
			{
				p.SetState(603)
				p.Match(ArcParserCOMMA)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}
			{
				p.SetState(604)
				p.Expression()
			}

		}
		p.SetState(609)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 67, p.GetParserRuleContext())
		if p.HasError() {
			goto errorExit
		}
	}
	p.SetState(611)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ArcParserCOMMA {
		{
			p.SetState(610)
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
	case 62:
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
