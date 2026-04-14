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
		"'sequence'", "'stage'", "'next'", "'chan'", "'authority'", "'i8'",
		"'i16'", "'i32'", "'i64'", "'u8'", "'u16'", "'u32'", "'u64'", "'f32'",
		"'f64'", "'str'", "'series'", "'->'", "':='", "'$='", "'=>'", "'='",
		"'+='", "'-='", "'*='", "'/='", "'%='", "'+'", "'-'", "'*'", "'/'",
		"'%'", "'^'", "'=='", "'!='", "'<'", "'>'", "'<='", "'>='", "'and'",
		"'or'", "'not'", "'('", "')'", "'{'", "'}'", "'['", "']'", "','", "':'",
		"'.'",
	}
	staticData.SymbolicNames = []string{
		"", "FUNC", "IF", "ELSE", "RETURN", "FOR", "BREAK", "CONTINUE", "SEQUENCE",
		"STAGE", "NEXT", "CHAN", "AUTHORITY", "I8", "I16", "I32", "I64", "U8",
		"U16", "U32", "U64", "F32", "F64", "STR", "SERIES", "ARROW", "DECLARE",
		"STATE_DECLARE", "TRANSITION", "ASSIGN", "PLUS_ASSIGN", "MINUS_ASSIGN",
		"STAR_ASSIGN", "SLASH_ASSIGN", "PERCENT_ASSIGN", "PLUS", "MINUS", "STAR",
		"SLASH", "PERCENT", "CARET", "EQ", "NEQ", "LT", "GT", "LEQ", "GEQ",
		"AND", "OR", "NOT", "LPAREN", "RPAREN", "LBRACE", "RBRACE", "LBRACKET",
		"RBRACKET", "COMMA", "COLON", "DOT", "INTEGER_LITERAL", "FLOAT_LITERAL",
		"STR_LITERAL", "IDENTIFIER", "SINGLE_LINE_COMMENT", "MULTI_LINE_COMMENT",
		"WS",
	}
	staticData.RuleNames = []string{
		"program", "topLevelItem", "authorityBlock", "authorityEntry", "functionDeclaration",
		"inputList", "input", "outputType", "multiOutputBlock", "namedOutput",
		"configBlock", "configList", "config", "sequenceDeclaration", "sequenceItem",
		"stageDeclaration", "stageBody", "stageItem", "singleInvocation", "globalConstant",
		"flowStatement", "flowOperator", "routingTable", "routingEntry", "flowNode",
		"identifier", "function", "qualifiedIdentifier", "configValues", "namedConfigValues",
		"namedConfigValue", "anonymousConfigValues", "arguments", "argumentList",
		"block", "statement", "variableDeclaration", "localVariable", "statefulVariable",
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
		4, 1, 65, 720, 2, 0, 7, 0, 2, 1, 7, 1, 2, 2, 7, 2, 2, 3, 7, 3, 2, 4, 7,
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
		73, 2, 74, 7, 74, 1, 0, 5, 0, 152, 8, 0, 10, 0, 12, 0, 155, 9, 0, 1, 0,
		1, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 3, 1, 165, 8, 1, 1, 2, 1, 2,
		1, 2, 1, 2, 1, 2, 5, 2, 172, 8, 2, 10, 2, 12, 2, 175, 9, 2, 1, 2, 3, 2,
		178, 8, 2, 1, 3, 1, 3, 1, 3, 3, 3, 183, 8, 3, 1, 4, 1, 4, 1, 4, 3, 4, 188,
		8, 4, 1, 4, 1, 4, 3, 4, 192, 8, 4, 1, 4, 1, 4, 3, 4, 196, 8, 4, 1, 4, 1,
		4, 1, 5, 1, 5, 1, 5, 5, 5, 203, 8, 5, 10, 5, 12, 5, 206, 9, 5, 1, 5, 3,
		5, 209, 8, 5, 1, 6, 1, 6, 1, 6, 1, 6, 3, 6, 215, 8, 6, 1, 7, 1, 7, 1, 7,
		1, 7, 3, 7, 221, 8, 7, 1, 8, 1, 8, 1, 8, 1, 8, 5, 8, 227, 8, 8, 10, 8,
		12, 8, 230, 9, 8, 1, 8, 3, 8, 233, 8, 8, 1, 8, 1, 8, 1, 9, 1, 9, 1, 9,
		1, 10, 1, 10, 3, 10, 242, 8, 10, 1, 10, 1, 10, 1, 11, 1, 11, 1, 11, 5,
		11, 249, 8, 11, 10, 11, 12, 11, 252, 9, 11, 1, 11, 3, 11, 255, 8, 11, 1,
		12, 1, 12, 1, 12, 1, 12, 3, 12, 261, 8, 12, 1, 13, 1, 13, 3, 13, 265, 8,
		13, 1, 13, 1, 13, 5, 13, 269, 8, 13, 10, 13, 12, 13, 272, 9, 13, 1, 13,
		1, 13, 1, 14, 1, 14, 1, 14, 1, 14, 3, 14, 280, 8, 14, 1, 15, 1, 15, 3,
		15, 284, 8, 15, 1, 15, 1, 15, 1, 16, 1, 16, 1, 16, 1, 16, 5, 16, 292, 8,
		16, 10, 16, 12, 16, 295, 9, 16, 1, 16, 3, 16, 298, 8, 16, 3, 16, 300, 8,
		16, 1, 16, 1, 16, 1, 17, 1, 17, 1, 17, 3, 17, 307, 8, 17, 1, 18, 1, 18,
		3, 18, 311, 8, 18, 1, 19, 1, 19, 1, 19, 1, 19, 1, 19, 1, 19, 1, 19, 1,
		19, 3, 19, 321, 8, 19, 1, 20, 1, 20, 3, 20, 325, 8, 20, 1, 20, 1, 20, 1,
		20, 3, 20, 330, 8, 20, 4, 20, 332, 8, 20, 11, 20, 12, 20, 333, 1, 21, 1,
		21, 1, 22, 1, 22, 1, 22, 1, 22, 5, 22, 342, 8, 22, 10, 22, 12, 22, 345,
		9, 22, 1, 22, 3, 22, 348, 8, 22, 1, 22, 1, 22, 1, 23, 1, 23, 1, 23, 1,
		23, 1, 23, 5, 23, 357, 8, 23, 10, 23, 12, 23, 360, 9, 23, 1, 23, 1, 23,
		3, 23, 364, 8, 23, 1, 24, 1, 24, 1, 24, 1, 24, 3, 24, 370, 8, 24, 1, 25,
		1, 25, 1, 26, 1, 26, 1, 26, 1, 26, 1, 26, 3, 26, 379, 8, 26, 1, 27, 1,
		27, 1, 27, 1, 27, 1, 28, 1, 28, 1, 28, 1, 28, 1, 28, 1, 28, 1, 28, 1, 28,
		1, 28, 1, 28, 3, 28, 395, 8, 28, 1, 29, 1, 29, 1, 29, 5, 29, 400, 8, 29,
		10, 29, 12, 29, 403, 9, 29, 1, 29, 3, 29, 406, 8, 29, 1, 30, 1, 30, 1,
		30, 1, 30, 1, 31, 1, 31, 1, 31, 5, 31, 415, 8, 31, 10, 31, 12, 31, 418,
		9, 31, 1, 31, 3, 31, 421, 8, 31, 1, 32, 1, 32, 3, 32, 425, 8, 32, 1, 32,
		1, 32, 1, 33, 1, 33, 1, 33, 5, 33, 432, 8, 33, 10, 33, 12, 33, 435, 9,
		33, 1, 33, 3, 33, 438, 8, 33, 1, 34, 1, 34, 5, 34, 442, 8, 34, 10, 34,
		12, 34, 445, 9, 34, 1, 34, 1, 34, 1, 35, 1, 35, 1, 35, 1, 35, 1, 35, 1,
		35, 1, 35, 1, 35, 3, 35, 457, 8, 35, 1, 36, 1, 36, 3, 36, 461, 8, 36, 1,
		37, 1, 37, 1, 37, 1, 37, 1, 37, 1, 37, 1, 37, 1, 37, 3, 37, 471, 8, 37,
		1, 38, 1, 38, 1, 38, 1, 38, 1, 38, 1, 38, 1, 38, 1, 38, 3, 38, 481, 8,
		38, 1, 39, 1, 39, 1, 39, 1, 39, 1, 39, 1, 39, 1, 39, 1, 39, 1, 39, 1, 39,
		1, 39, 1, 39, 1, 39, 1, 39, 1, 39, 1, 39, 1, 39, 3, 39, 500, 8, 39, 1,
		40, 1, 40, 1, 41, 1, 41, 1, 41, 1, 41, 5, 41, 508, 8, 41, 10, 41, 12, 41,
		511, 9, 41, 1, 41, 3, 41, 514, 8, 41, 1, 42, 1, 42, 1, 42, 1, 42, 1, 42,
		1, 43, 1, 43, 1, 43, 1, 44, 1, 44, 1, 44, 1, 44, 1, 45, 1, 45, 1, 45, 1,
		45, 1, 45, 1, 45, 1, 45, 1, 45, 1, 45, 1, 45, 3, 45, 538, 8, 45, 1, 46,
		1, 46, 1, 47, 1, 47, 1, 48, 1, 48, 3, 48, 546, 8, 48, 1, 49, 1, 49, 3,
		49, 550, 8, 49, 1, 49, 1, 49, 3, 49, 554, 8, 49, 1, 50, 1, 50, 1, 51, 1,
		51, 3, 51, 560, 8, 51, 1, 52, 1, 52, 3, 52, 564, 8, 52, 1, 53, 1, 53, 1,
		54, 1, 54, 1, 55, 1, 55, 1, 55, 3, 55, 573, 8, 55, 1, 55, 1, 55, 3, 55,
		577, 8, 55, 1, 56, 1, 56, 1, 56, 3, 56, 582, 8, 56, 1, 57, 1, 57, 1, 58,
		1, 58, 1, 58, 5, 58, 589, 8, 58, 10, 58, 12, 58, 592, 9, 58, 1, 59, 1,
		59, 1, 59, 5, 59, 597, 8, 59, 10, 59, 12, 59, 600, 9, 59, 1, 60, 1, 60,
		1, 60, 5, 60, 605, 8, 60, 10, 60, 12, 60, 608, 9, 60, 1, 61, 1, 61, 1,
		61, 5, 61, 613, 8, 61, 10, 61, 12, 61, 616, 9, 61, 1, 62, 1, 62, 1, 62,
		5, 62, 621, 8, 62, 10, 62, 12, 62, 624, 9, 62, 1, 63, 1, 63, 1, 63, 5,
		63, 629, 8, 63, 10, 63, 12, 63, 632, 9, 63, 1, 64, 1, 64, 1, 64, 3, 64,
		637, 8, 64, 1, 65, 1, 65, 1, 65, 1, 65, 1, 65, 3, 65, 644, 8, 65, 1, 66,
		1, 66, 1, 66, 5, 66, 649, 8, 66, 10, 66, 12, 66, 652, 9, 66, 1, 67, 1,
		67, 1, 67, 1, 67, 1, 67, 1, 67, 3, 67, 660, 8, 67, 1, 67, 1, 67, 3, 67,
		664, 8, 67, 1, 67, 3, 67, 667, 8, 67, 1, 68, 1, 68, 3, 68, 671, 8, 68,
		1, 68, 1, 68, 1, 69, 1, 69, 1, 69, 1, 69, 1, 69, 1, 69, 1, 69, 1, 69, 3,
		69, 683, 8, 69, 1, 70, 1, 70, 1, 70, 1, 70, 1, 70, 1, 71, 1, 71, 1, 71,
		3, 71, 693, 8, 71, 1, 72, 3, 72, 696, 8, 72, 1, 72, 1, 72, 1, 72, 3, 72,
		701, 8, 72, 1, 73, 1, 73, 3, 73, 705, 8, 73, 1, 73, 1, 73, 1, 74, 1, 74,
		1, 74, 5, 74, 712, 8, 74, 10, 74, 12, 74, 715, 9, 74, 1, 74, 3, 74, 718,
		8, 74, 1, 74, 0, 0, 75, 0, 2, 4, 6, 8, 10, 12, 14, 16, 18, 20, 22, 24,
		26, 28, 30, 32, 34, 36, 38, 40, 42, 44, 46, 48, 50, 52, 54, 56, 58, 60,
		62, 64, 66, 68, 70, 72, 74, 76, 78, 80, 82, 84, 86, 88, 90, 92, 94, 96,
		98, 100, 102, 104, 106, 108, 110, 112, 114, 116, 118, 120, 122, 124, 126,
		128, 130, 132, 134, 136, 138, 140, 142, 144, 146, 148, 0, 9, 2, 0, 25,
		25, 28, 28, 1, 0, 30, 34, 1, 0, 13, 20, 1, 0, 21, 22, 1, 0, 41, 42, 1,
		0, 43, 46, 1, 0, 35, 36, 1, 0, 37, 39, 1, 0, 59, 60, 754, 0, 153, 1, 0,
		0, 0, 2, 164, 1, 0, 0, 0, 4, 177, 1, 0, 0, 0, 6, 182, 1, 0, 0, 0, 8, 184,
		1, 0, 0, 0, 10, 199, 1, 0, 0, 0, 12, 210, 1, 0, 0, 0, 14, 220, 1, 0, 0,
		0, 16, 222, 1, 0, 0, 0, 18, 236, 1, 0, 0, 0, 20, 239, 1, 0, 0, 0, 22, 245,
		1, 0, 0, 0, 24, 256, 1, 0, 0, 0, 26, 262, 1, 0, 0, 0, 28, 279, 1, 0, 0,
		0, 30, 281, 1, 0, 0, 0, 32, 287, 1, 0, 0, 0, 34, 306, 1, 0, 0, 0, 36, 310,
		1, 0, 0, 0, 38, 320, 1, 0, 0, 0, 40, 324, 1, 0, 0, 0, 42, 335, 1, 0, 0,
		0, 44, 337, 1, 0, 0, 0, 46, 351, 1, 0, 0, 0, 48, 369, 1, 0, 0, 0, 50, 371,
		1, 0, 0, 0, 52, 378, 1, 0, 0, 0, 54, 380, 1, 0, 0, 0, 56, 394, 1, 0, 0,
		0, 58, 396, 1, 0, 0, 0, 60, 407, 1, 0, 0, 0, 62, 411, 1, 0, 0, 0, 64, 422,
		1, 0, 0, 0, 66, 428, 1, 0, 0, 0, 68, 439, 1, 0, 0, 0, 70, 456, 1, 0, 0,
		0, 72, 460, 1, 0, 0, 0, 74, 470, 1, 0, 0, 0, 76, 480, 1, 0, 0, 0, 78, 499,
		1, 0, 0, 0, 80, 501, 1, 0, 0, 0, 82, 503, 1, 0, 0, 0, 84, 515, 1, 0, 0,
		0, 86, 520, 1, 0, 0, 0, 88, 523, 1, 0, 0, 0, 90, 537, 1, 0, 0, 0, 92, 539,
		1, 0, 0, 0, 94, 541, 1, 0, 0, 0, 96, 543, 1, 0, 0, 0, 98, 553, 1, 0, 0,
		0, 100, 555, 1, 0, 0, 0, 102, 559, 1, 0, 0, 0, 104, 563, 1, 0, 0, 0, 106,
		565, 1, 0, 0, 0, 108, 567, 1, 0, 0, 0, 110, 576, 1, 0, 0, 0, 112, 578,
		1, 0, 0, 0, 114, 583, 1, 0, 0, 0, 116, 585, 1, 0, 0, 0, 118, 593, 1, 0,
		0, 0, 120, 601, 1, 0, 0, 0, 122, 609, 1, 0, 0, 0, 124, 617, 1, 0, 0, 0,
		126, 625, 1, 0, 0, 0, 128, 633, 1, 0, 0, 0, 130, 643, 1, 0, 0, 0, 132,
		645, 1, 0, 0, 0, 134, 666, 1, 0, 0, 0, 136, 668, 1, 0, 0, 0, 138, 682,
		1, 0, 0, 0, 140, 684, 1, 0, 0, 0, 142, 692, 1, 0, 0, 0, 144, 695, 1, 0,
		0, 0, 146, 702, 1, 0, 0, 0, 148, 708, 1, 0, 0, 0, 150, 152, 3, 2, 1, 0,
		151, 150, 1, 0, 0, 0, 152, 155, 1, 0, 0, 0, 153, 151, 1, 0, 0, 0, 153,
		154, 1, 0, 0, 0, 154, 156, 1, 0, 0, 0, 155, 153, 1, 0, 0, 0, 156, 157,
		5, 0, 0, 1, 157, 1, 1, 0, 0, 0, 158, 165, 3, 4, 2, 0, 159, 165, 3, 8, 4,
		0, 160, 165, 3, 40, 20, 0, 161, 165, 3, 26, 13, 0, 162, 165, 3, 30, 15,
		0, 163, 165, 3, 38, 19, 0, 164, 158, 1, 0, 0, 0, 164, 159, 1, 0, 0, 0,
		164, 160, 1, 0, 0, 0, 164, 161, 1, 0, 0, 0, 164, 162, 1, 0, 0, 0, 164,
		163, 1, 0, 0, 0, 165, 3, 1, 0, 0, 0, 166, 167, 5, 12, 0, 0, 167, 178, 5,
		59, 0, 0, 168, 169, 5, 12, 0, 0, 169, 173, 5, 50, 0, 0, 170, 172, 3, 6,
		3, 0, 171, 170, 1, 0, 0, 0, 172, 175, 1, 0, 0, 0, 173, 171, 1, 0, 0, 0,
		173, 174, 1, 0, 0, 0, 174, 176, 1, 0, 0, 0, 175, 173, 1, 0, 0, 0, 176,
		178, 5, 51, 0, 0, 177, 166, 1, 0, 0, 0, 177, 168, 1, 0, 0, 0, 178, 5, 1,
		0, 0, 0, 179, 183, 5, 59, 0, 0, 180, 181, 5, 62, 0, 0, 181, 183, 5, 59,
		0, 0, 182, 179, 1, 0, 0, 0, 182, 180, 1, 0, 0, 0, 183, 7, 1, 0, 0, 0, 184,
		185, 5, 1, 0, 0, 185, 187, 5, 62, 0, 0, 186, 188, 3, 20, 10, 0, 187, 186,
		1, 0, 0, 0, 187, 188, 1, 0, 0, 0, 188, 189, 1, 0, 0, 0, 189, 191, 5, 50,
		0, 0, 190, 192, 3, 10, 5, 0, 191, 190, 1, 0, 0, 0, 191, 192, 1, 0, 0, 0,
		192, 193, 1, 0, 0, 0, 193, 195, 5, 51, 0, 0, 194, 196, 3, 14, 7, 0, 195,
		194, 1, 0, 0, 0, 195, 196, 1, 0, 0, 0, 196, 197, 1, 0, 0, 0, 197, 198,
		3, 68, 34, 0, 198, 9, 1, 0, 0, 0, 199, 204, 3, 12, 6, 0, 200, 201, 5, 56,
		0, 0, 201, 203, 3, 12, 6, 0, 202, 200, 1, 0, 0, 0, 203, 206, 1, 0, 0, 0,
		204, 202, 1, 0, 0, 0, 204, 205, 1, 0, 0, 0, 205, 208, 1, 0, 0, 0, 206,
		204, 1, 0, 0, 0, 207, 209, 5, 56, 0, 0, 208, 207, 1, 0, 0, 0, 208, 209,
		1, 0, 0, 0, 209, 11, 1, 0, 0, 0, 210, 211, 5, 62, 0, 0, 211, 214, 3, 98,
		49, 0, 212, 213, 5, 29, 0, 0, 213, 215, 3, 142, 71, 0, 214, 212, 1, 0,
		0, 0, 214, 215, 1, 0, 0, 0, 215, 13, 1, 0, 0, 0, 216, 221, 3, 98, 49, 0,
		217, 218, 5, 62, 0, 0, 218, 221, 3, 98, 49, 0, 219, 221, 3, 16, 8, 0, 220,
		216, 1, 0, 0, 0, 220, 217, 1, 0, 0, 0, 220, 219, 1, 0, 0, 0, 221, 15, 1,
		0, 0, 0, 222, 223, 5, 50, 0, 0, 223, 228, 3, 18, 9, 0, 224, 225, 5, 56,
		0, 0, 225, 227, 3, 18, 9, 0, 226, 224, 1, 0, 0, 0, 227, 230, 1, 0, 0, 0,
		228, 226, 1, 0, 0, 0, 228, 229, 1, 0, 0, 0, 229, 232, 1, 0, 0, 0, 230,
		228, 1, 0, 0, 0, 231, 233, 5, 56, 0, 0, 232, 231, 1, 0, 0, 0, 232, 233,
		1, 0, 0, 0, 233, 234, 1, 0, 0, 0, 234, 235, 5, 51, 0, 0, 235, 17, 1, 0,
		0, 0, 236, 237, 5, 62, 0, 0, 237, 238, 3, 98, 49, 0, 238, 19, 1, 0, 0,
		0, 239, 241, 5, 52, 0, 0, 240, 242, 3, 22, 11, 0, 241, 240, 1, 0, 0, 0,
		241, 242, 1, 0, 0, 0, 242, 243, 1, 0, 0, 0, 243, 244, 5, 53, 0, 0, 244,
		21, 1, 0, 0, 0, 245, 250, 3, 24, 12, 0, 246, 247, 5, 56, 0, 0, 247, 249,
		3, 24, 12, 0, 248, 246, 1, 0, 0, 0, 249, 252, 1, 0, 0, 0, 250, 248, 1,
		0, 0, 0, 250, 251, 1, 0, 0, 0, 251, 254, 1, 0, 0, 0, 252, 250, 1, 0, 0,
		0, 253, 255, 5, 56, 0, 0, 254, 253, 1, 0, 0, 0, 254, 255, 1, 0, 0, 0, 255,
		23, 1, 0, 0, 0, 256, 257, 5, 62, 0, 0, 257, 260, 3, 98, 49, 0, 258, 259,
		5, 29, 0, 0, 259, 261, 3, 142, 71, 0, 260, 258, 1, 0, 0, 0, 260, 261, 1,
		0, 0, 0, 261, 25, 1, 0, 0, 0, 262, 264, 5, 8, 0, 0, 263, 265, 5, 62, 0,
		0, 264, 263, 1, 0, 0, 0, 264, 265, 1, 0, 0, 0, 265, 266, 1, 0, 0, 0, 266,
		270, 5, 52, 0, 0, 267, 269, 3, 28, 14, 0, 268, 267, 1, 0, 0, 0, 269, 272,
		1, 0, 0, 0, 270, 268, 1, 0, 0, 0, 270, 271, 1, 0, 0, 0, 271, 273, 1, 0,
		0, 0, 272, 270, 1, 0, 0, 0, 273, 274, 5, 53, 0, 0, 274, 27, 1, 0, 0, 0,
		275, 280, 3, 30, 15, 0, 276, 280, 3, 26, 13, 0, 277, 280, 3, 40, 20, 0,
		278, 280, 3, 36, 18, 0, 279, 275, 1, 0, 0, 0, 279, 276, 1, 0, 0, 0, 279,
		277, 1, 0, 0, 0, 279, 278, 1, 0, 0, 0, 280, 29, 1, 0, 0, 0, 281, 283, 5,
		9, 0, 0, 282, 284, 5, 62, 0, 0, 283, 282, 1, 0, 0, 0, 283, 284, 1, 0, 0,
		0, 284, 285, 1, 0, 0, 0, 285, 286, 3, 32, 16, 0, 286, 31, 1, 0, 0, 0, 287,
		299, 5, 52, 0, 0, 288, 293, 3, 34, 17, 0, 289, 290, 5, 56, 0, 0, 290, 292,
		3, 34, 17, 0, 291, 289, 1, 0, 0, 0, 292, 295, 1, 0, 0, 0, 293, 291, 1,
		0, 0, 0, 293, 294, 1, 0, 0, 0, 294, 297, 1, 0, 0, 0, 295, 293, 1, 0, 0,
		0, 296, 298, 5, 56, 0, 0, 297, 296, 1, 0, 0, 0, 297, 298, 1, 0, 0, 0, 298,
		300, 1, 0, 0, 0, 299, 288, 1, 0, 0, 0, 299, 300, 1, 0, 0, 0, 300, 301,
		1, 0, 0, 0, 301, 302, 5, 53, 0, 0, 302, 33, 1, 0, 0, 0, 303, 307, 3, 40,
		20, 0, 304, 307, 3, 36, 18, 0, 305, 307, 3, 26, 13, 0, 306, 303, 1, 0,
		0, 0, 306, 304, 1, 0, 0, 0, 306, 305, 1, 0, 0, 0, 307, 35, 1, 0, 0, 0,
		308, 311, 3, 52, 26, 0, 309, 311, 3, 114, 57, 0, 310, 308, 1, 0, 0, 0,
		310, 309, 1, 0, 0, 0, 311, 37, 1, 0, 0, 0, 312, 313, 5, 62, 0, 0, 313,
		314, 5, 26, 0, 0, 314, 321, 3, 142, 71, 0, 315, 316, 5, 62, 0, 0, 316,
		317, 3, 98, 49, 0, 317, 318, 5, 26, 0, 0, 318, 319, 3, 142, 71, 0, 319,
		321, 1, 0, 0, 0, 320, 312, 1, 0, 0, 0, 320, 315, 1, 0, 0, 0, 321, 39, 1,
		0, 0, 0, 322, 325, 3, 44, 22, 0, 323, 325, 3, 48, 24, 0, 324, 322, 1, 0,
		0, 0, 324, 323, 1, 0, 0, 0, 325, 331, 1, 0, 0, 0, 326, 329, 3, 42, 21,
		0, 327, 330, 3, 44, 22, 0, 328, 330, 3, 48, 24, 0, 329, 327, 1, 0, 0, 0,
		329, 328, 1, 0, 0, 0, 330, 332, 1, 0, 0, 0, 331, 326, 1, 0, 0, 0, 332,
		333, 1, 0, 0, 0, 333, 331, 1, 0, 0, 0, 333, 334, 1, 0, 0, 0, 334, 41, 1,
		0, 0, 0, 335, 336, 7, 0, 0, 0, 336, 43, 1, 0, 0, 0, 337, 338, 5, 52, 0,
		0, 338, 343, 3, 46, 23, 0, 339, 340, 5, 56, 0, 0, 340, 342, 3, 46, 23,
		0, 341, 339, 1, 0, 0, 0, 342, 345, 1, 0, 0, 0, 343, 341, 1, 0, 0, 0, 343,
		344, 1, 0, 0, 0, 344, 347, 1, 0, 0, 0, 345, 343, 1, 0, 0, 0, 346, 348,
		5, 56, 0, 0, 347, 346, 1, 0, 0, 0, 347, 348, 1, 0, 0, 0, 348, 349, 1, 0,
		0, 0, 349, 350, 5, 53, 0, 0, 350, 45, 1, 0, 0, 0, 351, 352, 5, 62, 0, 0,
		352, 353, 5, 57, 0, 0, 353, 358, 3, 48, 24, 0, 354, 355, 5, 25, 0, 0, 355,
		357, 3, 48, 24, 0, 356, 354, 1, 0, 0, 0, 357, 360, 1, 0, 0, 0, 358, 356,
		1, 0, 0, 0, 358, 359, 1, 0, 0, 0, 359, 363, 1, 0, 0, 0, 360, 358, 1, 0,
		0, 0, 361, 362, 5, 57, 0, 0, 362, 364, 5, 62, 0, 0, 363, 361, 1, 0, 0,
		0, 363, 364, 1, 0, 0, 0, 364, 47, 1, 0, 0, 0, 365, 370, 3, 50, 25, 0, 366,
		370, 3, 52, 26, 0, 367, 370, 3, 114, 57, 0, 368, 370, 5, 10, 0, 0, 369,
		365, 1, 0, 0, 0, 369, 366, 1, 0, 0, 0, 369, 367, 1, 0, 0, 0, 369, 368,
		1, 0, 0, 0, 370, 49, 1, 0, 0, 0, 371, 372, 5, 62, 0, 0, 372, 51, 1, 0,
		0, 0, 373, 374, 3, 54, 27, 0, 374, 375, 3, 56, 28, 0, 375, 379, 1, 0, 0,
		0, 376, 377, 5, 62, 0, 0, 377, 379, 3, 56, 28, 0, 378, 373, 1, 0, 0, 0,
		378, 376, 1, 0, 0, 0, 379, 53, 1, 0, 0, 0, 380, 381, 5, 62, 0, 0, 381,
		382, 5, 58, 0, 0, 382, 383, 5, 62, 0, 0, 383, 55, 1, 0, 0, 0, 384, 385,
		5, 52, 0, 0, 385, 395, 5, 53, 0, 0, 386, 387, 5, 52, 0, 0, 387, 388, 3,
		58, 29, 0, 388, 389, 5, 53, 0, 0, 389, 395, 1, 0, 0, 0, 390, 391, 5, 52,
		0, 0, 391, 392, 3, 62, 31, 0, 392, 393, 5, 53, 0, 0, 393, 395, 1, 0, 0,
		0, 394, 384, 1, 0, 0, 0, 394, 386, 1, 0, 0, 0, 394, 390, 1, 0, 0, 0, 395,
		57, 1, 0, 0, 0, 396, 401, 3, 60, 30, 0, 397, 398, 5, 56, 0, 0, 398, 400,
		3, 60, 30, 0, 399, 397, 1, 0, 0, 0, 400, 403, 1, 0, 0, 0, 401, 399, 1,
		0, 0, 0, 401, 402, 1, 0, 0, 0, 402, 405, 1, 0, 0, 0, 403, 401, 1, 0, 0,
		0, 404, 406, 5, 56, 0, 0, 405, 404, 1, 0, 0, 0, 405, 406, 1, 0, 0, 0, 406,
		59, 1, 0, 0, 0, 407, 408, 5, 62, 0, 0, 408, 409, 5, 29, 0, 0, 409, 410,
		3, 114, 57, 0, 410, 61, 1, 0, 0, 0, 411, 416, 3, 114, 57, 0, 412, 413,
		5, 56, 0, 0, 413, 415, 3, 114, 57, 0, 414, 412, 1, 0, 0, 0, 415, 418, 1,
		0, 0, 0, 416, 414, 1, 0, 0, 0, 416, 417, 1, 0, 0, 0, 417, 420, 1, 0, 0,
		0, 418, 416, 1, 0, 0, 0, 419, 421, 5, 56, 0, 0, 420, 419, 1, 0, 0, 0, 420,
		421, 1, 0, 0, 0, 421, 63, 1, 0, 0, 0, 422, 424, 5, 50, 0, 0, 423, 425,
		3, 66, 33, 0, 424, 423, 1, 0, 0, 0, 424, 425, 1, 0, 0, 0, 425, 426, 1,
		0, 0, 0, 426, 427, 5, 51, 0, 0, 427, 65, 1, 0, 0, 0, 428, 433, 3, 114,
		57, 0, 429, 430, 5, 56, 0, 0, 430, 432, 3, 114, 57, 0, 431, 429, 1, 0,
		0, 0, 432, 435, 1, 0, 0, 0, 433, 431, 1, 0, 0, 0, 433, 434, 1, 0, 0, 0,
		434, 437, 1, 0, 0, 0, 435, 433, 1, 0, 0, 0, 436, 438, 5, 56, 0, 0, 437,
		436, 1, 0, 0, 0, 437, 438, 1, 0, 0, 0, 438, 67, 1, 0, 0, 0, 439, 443, 5,
		52, 0, 0, 440, 442, 3, 70, 35, 0, 441, 440, 1, 0, 0, 0, 442, 445, 1, 0,
		0, 0, 443, 441, 1, 0, 0, 0, 443, 444, 1, 0, 0, 0, 444, 446, 1, 0, 0, 0,
		445, 443, 1, 0, 0, 0, 446, 447, 5, 53, 0, 0, 447, 69, 1, 0, 0, 0, 448,
		457, 3, 72, 36, 0, 449, 457, 3, 78, 39, 0, 450, 457, 3, 82, 41, 0, 451,
		457, 3, 88, 44, 0, 452, 457, 3, 92, 46, 0, 453, 457, 3, 94, 47, 0, 454,
		457, 3, 96, 48, 0, 455, 457, 3, 114, 57, 0, 456, 448, 1, 0, 0, 0, 456,
		449, 1, 0, 0, 0, 456, 450, 1, 0, 0, 0, 456, 451, 1, 0, 0, 0, 456, 452,
		1, 0, 0, 0, 456, 453, 1, 0, 0, 0, 456, 454, 1, 0, 0, 0, 456, 455, 1, 0,
		0, 0, 457, 71, 1, 0, 0, 0, 458, 461, 3, 74, 37, 0, 459, 461, 3, 76, 38,
		0, 460, 458, 1, 0, 0, 0, 460, 459, 1, 0, 0, 0, 461, 73, 1, 0, 0, 0, 462,
		463, 5, 62, 0, 0, 463, 464, 5, 26, 0, 0, 464, 471, 3, 114, 57, 0, 465,
		466, 5, 62, 0, 0, 466, 467, 3, 98, 49, 0, 467, 468, 5, 26, 0, 0, 468, 469,
		3, 114, 57, 0, 469, 471, 1, 0, 0, 0, 470, 462, 1, 0, 0, 0, 470, 465, 1,
		0, 0, 0, 471, 75, 1, 0, 0, 0, 472, 473, 5, 62, 0, 0, 473, 474, 5, 27, 0,
		0, 474, 481, 3, 114, 57, 0, 475, 476, 5, 62, 0, 0, 476, 477, 3, 98, 49,
		0, 477, 478, 5, 27, 0, 0, 478, 479, 3, 114, 57, 0, 479, 481, 1, 0, 0, 0,
		480, 472, 1, 0, 0, 0, 480, 475, 1, 0, 0, 0, 481, 77, 1, 0, 0, 0, 482, 483,
		5, 62, 0, 0, 483, 484, 5, 29, 0, 0, 484, 500, 3, 114, 57, 0, 485, 486,
		5, 62, 0, 0, 486, 487, 3, 134, 67, 0, 487, 488, 5, 29, 0, 0, 488, 489,
		3, 114, 57, 0, 489, 500, 1, 0, 0, 0, 490, 491, 5, 62, 0, 0, 491, 492, 3,
		80, 40, 0, 492, 493, 3, 114, 57, 0, 493, 500, 1, 0, 0, 0, 494, 495, 5,
		62, 0, 0, 495, 496, 3, 134, 67, 0, 496, 497, 3, 80, 40, 0, 497, 498, 3,
		114, 57, 0, 498, 500, 1, 0, 0, 0, 499, 482, 1, 0, 0, 0, 499, 485, 1, 0,
		0, 0, 499, 490, 1, 0, 0, 0, 499, 494, 1, 0, 0, 0, 500, 79, 1, 0, 0, 0,
		501, 502, 7, 1, 0, 0, 502, 81, 1, 0, 0, 0, 503, 504, 5, 2, 0, 0, 504, 505,
		3, 114, 57, 0, 505, 509, 3, 68, 34, 0, 506, 508, 3, 84, 42, 0, 507, 506,
		1, 0, 0, 0, 508, 511, 1, 0, 0, 0, 509, 507, 1, 0, 0, 0, 509, 510, 1, 0,
		0, 0, 510, 513, 1, 0, 0, 0, 511, 509, 1, 0, 0, 0, 512, 514, 3, 86, 43,
		0, 513, 512, 1, 0, 0, 0, 513, 514, 1, 0, 0, 0, 514, 83, 1, 0, 0, 0, 515,
		516, 5, 3, 0, 0, 516, 517, 5, 2, 0, 0, 517, 518, 3, 114, 57, 0, 518, 519,
		3, 68, 34, 0, 519, 85, 1, 0, 0, 0, 520, 521, 5, 3, 0, 0, 521, 522, 3, 68,
		34, 0, 522, 87, 1, 0, 0, 0, 523, 524, 5, 5, 0, 0, 524, 525, 3, 90, 45,
		0, 525, 526, 3, 68, 34, 0, 526, 89, 1, 0, 0, 0, 527, 528, 5, 62, 0, 0,
		528, 529, 5, 56, 0, 0, 529, 530, 5, 62, 0, 0, 530, 531, 5, 26, 0, 0, 531,
		538, 3, 114, 57, 0, 532, 533, 5, 62, 0, 0, 533, 534, 5, 26, 0, 0, 534,
		538, 3, 114, 57, 0, 535, 538, 3, 114, 57, 0, 536, 538, 1, 0, 0, 0, 537,
		527, 1, 0, 0, 0, 537, 532, 1, 0, 0, 0, 537, 535, 1, 0, 0, 0, 537, 536,
		1, 0, 0, 0, 538, 91, 1, 0, 0, 0, 539, 540, 5, 6, 0, 0, 540, 93, 1, 0, 0,
		0, 541, 542, 5, 7, 0, 0, 542, 95, 1, 0, 0, 0, 543, 545, 5, 4, 0, 0, 544,
		546, 3, 114, 57, 0, 545, 544, 1, 0, 0, 0, 545, 546, 1, 0, 0, 0, 546, 97,
		1, 0, 0, 0, 547, 549, 3, 102, 51, 0, 548, 550, 3, 100, 50, 0, 549, 548,
		1, 0, 0, 0, 549, 550, 1, 0, 0, 0, 550, 554, 1, 0, 0, 0, 551, 554, 3, 110,
		55, 0, 552, 554, 3, 112, 56, 0, 553, 547, 1, 0, 0, 0, 553, 551, 1, 0, 0,
		0, 553, 552, 1, 0, 0, 0, 554, 99, 1, 0, 0, 0, 555, 556, 5, 62, 0, 0, 556,
		101, 1, 0, 0, 0, 557, 560, 3, 104, 52, 0, 558, 560, 5, 23, 0, 0, 559, 557,
		1, 0, 0, 0, 559, 558, 1, 0, 0, 0, 560, 103, 1, 0, 0, 0, 561, 564, 3, 106,
		53, 0, 562, 564, 3, 108, 54, 0, 563, 561, 1, 0, 0, 0, 563, 562, 1, 0, 0,
		0, 564, 105, 1, 0, 0, 0, 565, 566, 7, 2, 0, 0, 566, 107, 1, 0, 0, 0, 567,
		568, 7, 3, 0, 0, 568, 109, 1, 0, 0, 0, 569, 570, 5, 11, 0, 0, 570, 572,
		3, 102, 51, 0, 571, 573, 3, 100, 50, 0, 572, 571, 1, 0, 0, 0, 572, 573,
		1, 0, 0, 0, 573, 577, 1, 0, 0, 0, 574, 575, 5, 11, 0, 0, 575, 577, 3, 112,
		56, 0, 576, 569, 1, 0, 0, 0, 576, 574, 1, 0, 0, 0, 577, 111, 1, 0, 0, 0,
		578, 579, 5, 24, 0, 0, 579, 581, 3, 102, 51, 0, 580, 582, 3, 100, 50, 0,
		581, 580, 1, 0, 0, 0, 581, 582, 1, 0, 0, 0, 582, 113, 1, 0, 0, 0, 583,
		584, 3, 116, 58, 0, 584, 115, 1, 0, 0, 0, 585, 590, 3, 118, 59, 0, 586,
		587, 5, 48, 0, 0, 587, 589, 3, 118, 59, 0, 588, 586, 1, 0, 0, 0, 589, 592,
		1, 0, 0, 0, 590, 588, 1, 0, 0, 0, 590, 591, 1, 0, 0, 0, 591, 117, 1, 0,
		0, 0, 592, 590, 1, 0, 0, 0, 593, 598, 3, 120, 60, 0, 594, 595, 5, 47, 0,
		0, 595, 597, 3, 120, 60, 0, 596, 594, 1, 0, 0, 0, 597, 600, 1, 0, 0, 0,
		598, 596, 1, 0, 0, 0, 598, 599, 1, 0, 0, 0, 599, 119, 1, 0, 0, 0, 600,
		598, 1, 0, 0, 0, 601, 606, 3, 122, 61, 0, 602, 603, 7, 4, 0, 0, 603, 605,
		3, 122, 61, 0, 604, 602, 1, 0, 0, 0, 605, 608, 1, 0, 0, 0, 606, 604, 1,
		0, 0, 0, 606, 607, 1, 0, 0, 0, 607, 121, 1, 0, 0, 0, 608, 606, 1, 0, 0,
		0, 609, 614, 3, 124, 62, 0, 610, 611, 7, 5, 0, 0, 611, 613, 3, 124, 62,
		0, 612, 610, 1, 0, 0, 0, 613, 616, 1, 0, 0, 0, 614, 612, 1, 0, 0, 0, 614,
		615, 1, 0, 0, 0, 615, 123, 1, 0, 0, 0, 616, 614, 1, 0, 0, 0, 617, 622,
		3, 126, 63, 0, 618, 619, 7, 6, 0, 0, 619, 621, 3, 126, 63, 0, 620, 618,
		1, 0, 0, 0, 621, 624, 1, 0, 0, 0, 622, 620, 1, 0, 0, 0, 622, 623, 1, 0,
		0, 0, 623, 125, 1, 0, 0, 0, 624, 622, 1, 0, 0, 0, 625, 630, 3, 128, 64,
		0, 626, 627, 7, 7, 0, 0, 627, 629, 3, 128, 64, 0, 628, 626, 1, 0, 0, 0,
		629, 632, 1, 0, 0, 0, 630, 628, 1, 0, 0, 0, 630, 631, 1, 0, 0, 0, 631,
		127, 1, 0, 0, 0, 632, 630, 1, 0, 0, 0, 633, 636, 3, 130, 65, 0, 634, 635,
		5, 40, 0, 0, 635, 637, 3, 128, 64, 0, 636, 634, 1, 0, 0, 0, 636, 637, 1,
		0, 0, 0, 637, 129, 1, 0, 0, 0, 638, 639, 5, 36, 0, 0, 639, 644, 3, 130,
		65, 0, 640, 641, 5, 49, 0, 0, 641, 644, 3, 130, 65, 0, 642, 644, 3, 132,
		66, 0, 643, 638, 1, 0, 0, 0, 643, 640, 1, 0, 0, 0, 643, 642, 1, 0, 0, 0,
		644, 131, 1, 0, 0, 0, 645, 650, 3, 138, 69, 0, 646, 649, 3, 134, 67, 0,
		647, 649, 3, 136, 68, 0, 648, 646, 1, 0, 0, 0, 648, 647, 1, 0, 0, 0, 649,
		652, 1, 0, 0, 0, 650, 648, 1, 0, 0, 0, 650, 651, 1, 0, 0, 0, 651, 133,
		1, 0, 0, 0, 652, 650, 1, 0, 0, 0, 653, 654, 5, 54, 0, 0, 654, 655, 3, 114,
		57, 0, 655, 656, 5, 55, 0, 0, 656, 667, 1, 0, 0, 0, 657, 659, 5, 54, 0,
		0, 658, 660, 3, 114, 57, 0, 659, 658, 1, 0, 0, 0, 659, 660, 1, 0, 0, 0,
		660, 661, 1, 0, 0, 0, 661, 663, 5, 57, 0, 0, 662, 664, 3, 114, 57, 0, 663,
		662, 1, 0, 0, 0, 663, 664, 1, 0, 0, 0, 664, 665, 1, 0, 0, 0, 665, 667,
		5, 55, 0, 0, 666, 653, 1, 0, 0, 0, 666, 657, 1, 0, 0, 0, 667, 135, 1, 0,
		0, 0, 668, 670, 5, 50, 0, 0, 669, 671, 3, 66, 33, 0, 670, 669, 1, 0, 0,
		0, 670, 671, 1, 0, 0, 0, 671, 672, 1, 0, 0, 0, 672, 673, 5, 51, 0, 0, 673,
		137, 1, 0, 0, 0, 674, 683, 3, 142, 71, 0, 675, 683, 3, 54, 27, 0, 676,
		683, 5, 62, 0, 0, 677, 678, 5, 50, 0, 0, 678, 679, 3, 114, 57, 0, 679,
		680, 5, 51, 0, 0, 680, 683, 1, 0, 0, 0, 681, 683, 3, 140, 70, 0, 682, 674,
		1, 0, 0, 0, 682, 675, 1, 0, 0, 0, 682, 676, 1, 0, 0, 0, 682, 677, 1, 0,
		0, 0, 682, 681, 1, 0, 0, 0, 683, 139, 1, 0, 0, 0, 684, 685, 3, 98, 49,
		0, 685, 686, 5, 50, 0, 0, 686, 687, 3, 114, 57, 0, 687, 688, 5, 51, 0,
		0, 688, 141, 1, 0, 0, 0, 689, 693, 3, 144, 72, 0, 690, 693, 5, 61, 0, 0,
		691, 693, 3, 146, 73, 0, 692, 689, 1, 0, 0, 0, 692, 690, 1, 0, 0, 0, 692,
		691, 1, 0, 0, 0, 693, 143, 1, 0, 0, 0, 694, 696, 5, 36, 0, 0, 695, 694,
		1, 0, 0, 0, 695, 696, 1, 0, 0, 0, 696, 697, 1, 0, 0, 0, 697, 700, 7, 8,
		0, 0, 698, 699, 4, 72, 0, 0, 699, 701, 5, 62, 0, 0, 700, 698, 1, 0, 0,
		0, 700, 701, 1, 0, 0, 0, 701, 145, 1, 0, 0, 0, 702, 704, 5, 54, 0, 0, 703,
		705, 3, 148, 74, 0, 704, 703, 1, 0, 0, 0, 704, 705, 1, 0, 0, 0, 705, 706,
		1, 0, 0, 0, 706, 707, 5, 55, 0, 0, 707, 147, 1, 0, 0, 0, 708, 713, 3, 114,
		57, 0, 709, 710, 5, 56, 0, 0, 710, 712, 3, 114, 57, 0, 711, 709, 1, 0,
		0, 0, 712, 715, 1, 0, 0, 0, 713, 711, 1, 0, 0, 0, 713, 714, 1, 0, 0, 0,
		714, 717, 1, 0, 0, 0, 715, 713, 1, 0, 0, 0, 716, 718, 5, 56, 0, 0, 717,
		716, 1, 0, 0, 0, 717, 718, 1, 0, 0, 0, 718, 149, 1, 0, 0, 0, 83, 153, 164,
		173, 177, 182, 187, 191, 195, 204, 208, 214, 220, 228, 232, 241, 250, 254,
		260, 264, 270, 279, 283, 293, 297, 299, 306, 310, 320, 324, 329, 333, 343,
		347, 358, 363, 369, 378, 394, 401, 405, 416, 420, 424, 433, 437, 443, 456,
		460, 470, 480, 499, 509, 513, 537, 545, 549, 553, 559, 563, 572, 576, 581,
		590, 598, 606, 614, 622, 630, 636, 643, 648, 650, 659, 663, 666, 670, 682,
		692, 695, 700, 704, 713, 717,
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
	ArcParserSEQUENCE            = 8
	ArcParserSTAGE               = 9
	ArcParserNEXT                = 10
	ArcParserCHAN                = 11
	ArcParserAUTHORITY           = 12
	ArcParserI8                  = 13
	ArcParserI16                 = 14
	ArcParserI32                 = 15
	ArcParserI64                 = 16
	ArcParserU8                  = 17
	ArcParserU16                 = 18
	ArcParserU32                 = 19
	ArcParserU64                 = 20
	ArcParserF32                 = 21
	ArcParserF64                 = 22
	ArcParserSTR                 = 23
	ArcParserSERIES              = 24
	ArcParserARROW               = 25
	ArcParserDECLARE             = 26
	ArcParserSTATE_DECLARE       = 27
	ArcParserTRANSITION          = 28
	ArcParserASSIGN              = 29
	ArcParserPLUS_ASSIGN         = 30
	ArcParserMINUS_ASSIGN        = 31
	ArcParserSTAR_ASSIGN         = 32
	ArcParserSLASH_ASSIGN        = 33
	ArcParserPERCENT_ASSIGN      = 34
	ArcParserPLUS                = 35
	ArcParserMINUS               = 36
	ArcParserSTAR                = 37
	ArcParserSLASH               = 38
	ArcParserPERCENT             = 39
	ArcParserCARET               = 40
	ArcParserEQ                  = 41
	ArcParserNEQ                 = 42
	ArcParserLT                  = 43
	ArcParserGT                  = 44
	ArcParserLEQ                 = 45
	ArcParserGEQ                 = 46
	ArcParserAND                 = 47
	ArcParserOR                  = 48
	ArcParserNOT                 = 49
	ArcParserLPAREN              = 50
	ArcParserRPAREN              = 51
	ArcParserLBRACE              = 52
	ArcParserRBRACE              = 53
	ArcParserLBRACKET            = 54
	ArcParserRBRACKET            = 55
	ArcParserCOMMA               = 56
	ArcParserCOLON               = 57
	ArcParserDOT                 = 58
	ArcParserINTEGER_LITERAL     = 59
	ArcParserFLOAT_LITERAL       = 60
	ArcParserSTR_LITERAL         = 61
	ArcParserIDENTIFIER          = 62
	ArcParserSINGLE_LINE_COMMENT = 63
	ArcParserMULTI_LINE_COMMENT  = 64
	ArcParserWS                  = 65
)

// ArcParser rules.
const (
	ArcParserRULE_program                  = 0
	ArcParserRULE_topLevelItem             = 1
	ArcParserRULE_authorityBlock           = 2
	ArcParserRULE_authorityEntry           = 3
	ArcParserRULE_functionDeclaration      = 4
	ArcParserRULE_inputList                = 5
	ArcParserRULE_input                    = 6
	ArcParserRULE_outputType               = 7
	ArcParserRULE_multiOutputBlock         = 8
	ArcParserRULE_namedOutput              = 9
	ArcParserRULE_configBlock              = 10
	ArcParserRULE_configList               = 11
	ArcParserRULE_config                   = 12
	ArcParserRULE_sequenceDeclaration      = 13
	ArcParserRULE_sequenceItem             = 14
	ArcParserRULE_stageDeclaration         = 15
	ArcParserRULE_stageBody                = 16
	ArcParserRULE_stageItem                = 17
	ArcParserRULE_singleInvocation         = 18
	ArcParserRULE_globalConstant           = 19
	ArcParserRULE_flowStatement            = 20
	ArcParserRULE_flowOperator             = 21
	ArcParserRULE_routingTable             = 22
	ArcParserRULE_routingEntry             = 23
	ArcParserRULE_flowNode                 = 24
	ArcParserRULE_identifier               = 25
	ArcParserRULE_function                 = 26
	ArcParserRULE_qualifiedIdentifier      = 27
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
	ArcParserRULE_compoundOp               = 40
	ArcParserRULE_ifStatement              = 41
	ArcParserRULE_elseIfClause             = 42
	ArcParserRULE_elseClause               = 43
	ArcParserRULE_forStatement             = 44
	ArcParserRULE_forClause                = 45
	ArcParserRULE_breakStatement           = 46
	ArcParserRULE_continueStatement        = 47
	ArcParserRULE_returnStatement          = 48
	ArcParserRULE_type                     = 49
	ArcParserRULE_unitSuffix               = 50
	ArcParserRULE_primitiveType            = 51
	ArcParserRULE_numericType              = 52
	ArcParserRULE_integerType              = 53
	ArcParserRULE_floatType                = 54
	ArcParserRULE_channelType              = 55
	ArcParserRULE_seriesType               = 56
	ArcParserRULE_expression               = 57
	ArcParserRULE_logicalOrExpression      = 58
	ArcParserRULE_logicalAndExpression     = 59
	ArcParserRULE_equalityExpression       = 60
	ArcParserRULE_relationalExpression     = 61
	ArcParserRULE_additiveExpression       = 62
	ArcParserRULE_multiplicativeExpression = 63
	ArcParserRULE_powerExpression          = 64
	ArcParserRULE_unaryExpression          = 65
	ArcParserRULE_postfixExpression        = 66
	ArcParserRULE_indexOrSlice             = 67
	ArcParserRULE_functionCallSuffix       = 68
	ArcParserRULE_primaryExpression        = 69
	ArcParserRULE_typeCast                 = 70
	ArcParserRULE_literal                  = 71
	ArcParserRULE_numericLiteral           = 72
	ArcParserRULE_seriesLiteral            = 73
	ArcParserRULE_expressionList           = 74
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
	p.SetState(153)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&8671118201301499650) != 0 {
		{
			p.SetState(150)
			p.TopLevelItem()
		}

		p.SetState(155)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(156)
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
	p.SetState(164)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 1, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(158)
			p.AuthorityBlock()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(159)
			p.FunctionDeclaration()
		}

	case 3:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(160)
			p.FlowStatement()
		}

	case 4:
		p.EnterOuterAlt(localctx, 4)
		{
			p.SetState(161)
			p.SequenceDeclaration()
		}

	case 5:
		p.EnterOuterAlt(localctx, 5)
		{
			p.SetState(162)
			p.StageDeclaration()
		}

	case 6:
		p.EnterOuterAlt(localctx, 6)
		{
			p.SetState(163)
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
	p.EnterRule(localctx, 4, ArcParserRULE_authorityBlock)
	var _la int

	p.SetState(177)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 3, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(166)
			p.Match(ArcParserAUTHORITY)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(167)
			p.Match(ArcParserINTEGER_LITERAL)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(168)
			p.Match(ArcParserAUTHORITY)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(169)
			p.Match(ArcParserLPAREN)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		p.SetState(173)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == ArcParserINTEGER_LITERAL || _la == ArcParserIDENTIFIER {
			{
				p.SetState(170)
				p.AuthorityEntry()
			}

			p.SetState(175)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}
		{
			p.SetState(176)
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
	p.EnterRule(localctx, 6, ArcParserRULE_authorityEntry)
	p.SetState(182)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case ArcParserINTEGER_LITERAL:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(179)
			p.Match(ArcParserINTEGER_LITERAL)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case ArcParserIDENTIFIER:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(180)
			p.Match(ArcParserIDENTIFIER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(181)
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
	p.EnterRule(localctx, 8, ArcParserRULE_functionDeclaration)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(184)
		p.Match(ArcParserFUNC)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(185)
		p.Match(ArcParserIDENTIFIER)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(187)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ArcParserLBRACE {
		{
			p.SetState(186)
			p.ConfigBlock()
		}

	}
	{
		p.SetState(189)
		p.Match(ArcParserLPAREN)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(191)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ArcParserIDENTIFIER {
		{
			p.SetState(190)
			p.InputList()
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
	p.SetState(195)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&4612811918367778816) != 0 {
		{
			p.SetState(194)
			p.OutputType()
		}

	}
	{
		p.SetState(197)
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
	p.EnterRule(localctx, 10, ArcParserRULE_inputList)
	var _la int

	var _alt int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(199)
		p.Input()
	}
	p.SetState(204)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 8, p.GetParserRuleContext())
	if p.HasError() {
		goto errorExit
	}
	for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
		if _alt == 1 {
			{
				p.SetState(200)
				p.Match(ArcParserCOMMA)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}
			{
				p.SetState(201)
				p.Input()
			}

		}
		p.SetState(206)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 8, p.GetParserRuleContext())
		if p.HasError() {
			goto errorExit
		}
	}
	p.SetState(208)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ArcParserCOMMA {
		{
			p.SetState(207)
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
	p.EnterRule(localctx, 12, ArcParserRULE_input)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(210)
		p.Match(ArcParserIDENTIFIER)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(211)
		p.Type_()
	}
	p.SetState(214)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ArcParserASSIGN {
		{
			p.SetState(212)
			p.Match(ArcParserASSIGN)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(213)
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
	p.EnterRule(localctx, 14, ArcParserRULE_outputType)
	p.SetState(220)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case ArcParserCHAN, ArcParserI8, ArcParserI16, ArcParserI32, ArcParserI64, ArcParserU8, ArcParserU16, ArcParserU32, ArcParserU64, ArcParserF32, ArcParserF64, ArcParserSTR, ArcParserSERIES:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(216)
			p.Type_()
		}

	case ArcParserIDENTIFIER:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(217)
			p.Match(ArcParserIDENTIFIER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(218)
			p.Type_()
		}

	case ArcParserLPAREN:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(219)
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
	p.EnterRule(localctx, 16, ArcParserRULE_multiOutputBlock)
	var _la int

	var _alt int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(222)
		p.Match(ArcParserLPAREN)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(223)
		p.NamedOutput()
	}
	p.SetState(228)
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
				p.SetState(224)
				p.Match(ArcParserCOMMA)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}
			{
				p.SetState(225)
				p.NamedOutput()
			}

		}
		p.SetState(230)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 12, p.GetParserRuleContext())
		if p.HasError() {
			goto errorExit
		}
	}
	p.SetState(232)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ArcParserCOMMA {
		{
			p.SetState(231)
			p.Match(ArcParserCOMMA)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	}
	{
		p.SetState(234)
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
	p.EnterRule(localctx, 18, ArcParserRULE_namedOutput)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(236)
		p.Match(ArcParserIDENTIFIER)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(237)
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
	ConfigList() IConfigListContext

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

func (s *ConfigBlockContext) ConfigList() IConfigListContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IConfigListContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IConfigListContext)
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

func (s *ConfigBlockContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitConfigBlock(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) ConfigBlock() (localctx IConfigBlockContext) {
	localctx = NewConfigBlockContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 20, ArcParserRULE_configBlock)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(239)
		p.Match(ArcParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(241)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ArcParserIDENTIFIER {
		{
			p.SetState(240)
			p.ConfigList()
		}

	}
	{
		p.SetState(243)
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

// IConfigListContext is an interface to support dynamic dispatch.
type IConfigListContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AllConfig() []IConfigContext
	Config(i int) IConfigContext
	AllCOMMA() []antlr.TerminalNode
	COMMA(i int) antlr.TerminalNode

	// IsConfigListContext differentiates from other interfaces.
	IsConfigListContext()
}

type ConfigListContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyConfigListContext() *ConfigListContext {
	var p = new(ConfigListContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_configList
	return p
}

func InitEmptyConfigListContext(p *ConfigListContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = ArcParserRULE_configList
}

func (*ConfigListContext) IsConfigListContext() {}

func NewConfigListContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ConfigListContext {
	var p = new(ConfigListContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = ArcParserRULE_configList

	return p
}

func (s *ConfigListContext) GetParser() antlr.Parser { return s.parser }

func (s *ConfigListContext) AllConfig() []IConfigContext {
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

func (s *ConfigListContext) Config(i int) IConfigContext {
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

func (s *ConfigListContext) AllCOMMA() []antlr.TerminalNode {
	return s.GetTokens(ArcParserCOMMA)
}

func (s *ConfigListContext) COMMA(i int) antlr.TerminalNode {
	return s.GetToken(ArcParserCOMMA, i)
}

func (s *ConfigListContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ConfigListContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ConfigListContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.EnterConfigList(s)
	}
}

func (s *ConfigListContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(ArcParserListener); ok {
		listenerT.ExitConfigList(s)
	}
}

func (s *ConfigListContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitConfigList(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) ConfigList() (localctx IConfigListContext) {
	localctx = NewConfigListContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 22, ArcParserRULE_configList)
	var _la int

	var _alt int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(245)
		p.Config()
	}
	p.SetState(250)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 15, p.GetParserRuleContext())
	if p.HasError() {
		goto errorExit
	}
	for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
		if _alt == 1 {
			{
				p.SetState(246)
				p.Match(ArcParserCOMMA)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}
			{
				p.SetState(247)
				p.Config()
			}

		}
		p.SetState(252)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 15, p.GetParserRuleContext())
		if p.HasError() {
			goto errorExit
		}
	}
	p.SetState(254)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ArcParserCOMMA {
		{
			p.SetState(253)
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

// IConfigContext is an interface to support dynamic dispatch.
type IConfigContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	IDENTIFIER() antlr.TerminalNode
	Type_() ITypeContext
	ASSIGN() antlr.TerminalNode
	Literal() ILiteralContext

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

func (s *ConfigContext) ASSIGN() antlr.TerminalNode {
	return s.GetToken(ArcParserASSIGN, 0)
}

func (s *ConfigContext) Literal() ILiteralContext {
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

func (s *ConfigContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitConfig(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) Config() (localctx IConfigContext) {
	localctx = NewConfigContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 24, ArcParserRULE_config)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(256)
		p.Match(ArcParserIDENTIFIER)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(257)
		p.Type_()
	}
	p.SetState(260)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ArcParserASSIGN {
		{
			p.SetState(258)
			p.Match(ArcParserASSIGN)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(259)
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
	p.EnterRule(localctx, 26, ArcParserRULE_sequenceDeclaration)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(262)
		p.Match(ArcParserSEQUENCE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(264)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ArcParserIDENTIFIER {
		{
			p.SetState(263)
			p.Match(ArcParserIDENTIFIER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	}
	{
		p.SetState(266)
		p.Match(ArcParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(270)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&8671118201301495552) != 0 {
		{
			p.SetState(267)
			p.SequenceItem()
		}

		p.SetState(272)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(273)
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
	p.EnterRule(localctx, 28, ArcParserRULE_sequenceItem)
	p.SetState(279)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 20, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(275)
			p.StageDeclaration()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(276)
			p.SequenceDeclaration()
		}

	case 3:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(277)
			p.FlowStatement()
		}

	case 4:
		p.EnterOuterAlt(localctx, 4)
		{
			p.SetState(278)
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
	p.EnterRule(localctx, 30, ArcParserRULE_stageDeclaration)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(281)
		p.Match(ArcParserSTAGE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(283)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ArcParserIDENTIFIER {
		{
			p.SetState(282)
			p.Match(ArcParserIDENTIFIER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	}
	{
		p.SetState(285)
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
	p.EnterRule(localctx, 32, ArcParserRULE_stageBody)
	var _la int

	var _alt int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(287)
		p.Match(ArcParserLBRACE)
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

	if (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&8671118201301495040) != 0 {
		{
			p.SetState(288)
			p.StageItem()
		}
		p.SetState(293)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 22, p.GetParserRuleContext())
		if p.HasError() {
			goto errorExit
		}
		for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
			if _alt == 1 {
				{
					p.SetState(289)
					p.Match(ArcParserCOMMA)
					if p.HasError() {
						// Recognition error - abort rule
						goto errorExit
					}
				}
				{
					p.SetState(290)
					p.StageItem()
				}

			}
			p.SetState(295)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 22, p.GetParserRuleContext())
			if p.HasError() {
				goto errorExit
			}
		}
		p.SetState(297)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		if _la == ArcParserCOMMA {
			{
				p.SetState(296)
				p.Match(ArcParserCOMMA)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}

		}

	}
	{
		p.SetState(301)
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
	p.EnterRule(localctx, 34, ArcParserRULE_stageItem)
	p.SetState(306)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 25, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(303)
			p.FlowStatement()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(304)
			p.SingleInvocation()
		}

	case 3:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(305)
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
	p.EnterRule(localctx, 36, ArcParserRULE_singleInvocation)
	p.SetState(310)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 26, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(308)
			p.Function()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(309)
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
	p.EnterRule(localctx, 38, ArcParserRULE_globalConstant)
	p.SetState(320)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 27, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(312)
			p.Match(ArcParserIDENTIFIER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(313)
			p.Match(ArcParserDECLARE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(314)
			p.Literal()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(315)
			p.Match(ArcParserIDENTIFIER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(316)
			p.Type_()
		}
		{
			p.SetState(317)
			p.Match(ArcParserDECLARE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(318)
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
	p.EnterRule(localctx, 40, ArcParserRULE_flowStatement)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(324)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case ArcParserLBRACE:
		{
			p.SetState(322)
			p.RoutingTable()
		}

	case ArcParserNEXT, ArcParserCHAN, ArcParserI8, ArcParserI16, ArcParserI32, ArcParserI64, ArcParserU8, ArcParserU16, ArcParserU32, ArcParserU64, ArcParserF32, ArcParserF64, ArcParserSTR, ArcParserSERIES, ArcParserMINUS, ArcParserNOT, ArcParserLPAREN, ArcParserLBRACKET, ArcParserINTEGER_LITERAL, ArcParserFLOAT_LITERAL, ArcParserSTR_LITERAL, ArcParserIDENTIFIER:
		{
			p.SetState(323)
			p.FlowNode()
		}

	default:
		p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
		goto errorExit
	}
	p.SetState(331)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for ok := true; ok; ok = _la == ArcParserARROW || _la == ArcParserTRANSITION {
		{
			p.SetState(326)
			p.FlowOperator()
		}
		p.SetState(329)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}

		switch p.GetTokenStream().LA(1) {
		case ArcParserLBRACE:
			{
				p.SetState(327)
				p.RoutingTable()
			}

		case ArcParserNEXT, ArcParserCHAN, ArcParserI8, ArcParserI16, ArcParserI32, ArcParserI64, ArcParserU8, ArcParserU16, ArcParserU32, ArcParserU64, ArcParserF32, ArcParserF64, ArcParserSTR, ArcParserSERIES, ArcParserMINUS, ArcParserNOT, ArcParserLPAREN, ArcParserLBRACKET, ArcParserINTEGER_LITERAL, ArcParserFLOAT_LITERAL, ArcParserSTR_LITERAL, ArcParserIDENTIFIER:
			{
				p.SetState(328)
				p.FlowNode()
			}

		default:
			p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
			goto errorExit
		}

		p.SetState(333)
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
	p.EnterRule(localctx, 42, ArcParserRULE_flowOperator)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(335)
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
	p.EnterRule(localctx, 44, ArcParserRULE_routingTable)
	var _la int

	var _alt int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(337)
		p.Match(ArcParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(338)
		p.RoutingEntry()
	}
	p.SetState(343)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 31, p.GetParserRuleContext())
	if p.HasError() {
		goto errorExit
	}
	for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
		if _alt == 1 {
			{
				p.SetState(339)
				p.Match(ArcParserCOMMA)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}
			{
				p.SetState(340)
				p.RoutingEntry()
			}

		}
		p.SetState(345)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 31, p.GetParserRuleContext())
		if p.HasError() {
			goto errorExit
		}
	}
	p.SetState(347)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ArcParserCOMMA {
		{
			p.SetState(346)
			p.Match(ArcParserCOMMA)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	}
	{
		p.SetState(349)
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
	p.EnterRule(localctx, 46, ArcParserRULE_routingEntry)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(351)
		p.Match(ArcParserIDENTIFIER)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(352)
		p.Match(ArcParserCOLON)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(353)
		p.FlowNode()
	}
	p.SetState(358)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == ArcParserARROW {
		{
			p.SetState(354)
			p.Match(ArcParserARROW)
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
	}
	p.SetState(363)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ArcParserCOLON {
		{
			p.SetState(361)
			p.Match(ArcParserCOLON)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(362)
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
	p.EnterRule(localctx, 48, ArcParserRULE_flowNode)
	p.SetState(369)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 35, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(365)
			p.Identifier()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(366)
			p.Function()
		}

	case 3:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(367)
			p.Expression()
		}

	case 4:
		p.EnterOuterAlt(localctx, 4)
		{
			p.SetState(368)
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
	p.EnterRule(localctx, 50, ArcParserRULE_identifier)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(371)
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
	ConfigValues() IConfigValuesContext
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
	p.EnterRule(localctx, 52, ArcParserRULE_function)
	p.SetState(378)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 36, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(373)
			p.QualifiedIdentifier()
		}
		{
			p.SetState(374)
			p.ConfigValues()
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
			p.ConfigValues()
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
	p.EnterRule(localctx, 54, ArcParserRULE_qualifiedIdentifier)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(380)
		p.Match(ArcParserIDENTIFIER)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(381)
		p.Match(ArcParserDOT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(382)
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

func (s *ConfigValuesContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitConfigValues(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) ConfigValues() (localctx IConfigValuesContext) {
	localctx = NewConfigValuesContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 56, ArcParserRULE_configValues)
	p.SetState(394)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 37, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(384)
			p.Match(ArcParserLBRACE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(385)
			p.Match(ArcParserRBRACE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(386)
			p.Match(ArcParserLBRACE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(387)
			p.NamedConfigValues()
		}
		{
			p.SetState(388)
			p.Match(ArcParserRBRACE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 3:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(390)
			p.Match(ArcParserLBRACE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(391)
			p.AnonymousConfigValues()
		}
		{
			p.SetState(392)
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

func (s *NamedConfigValuesContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitNamedConfigValues(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) NamedConfigValues() (localctx INamedConfigValuesContext) {
	localctx = NewNamedConfigValuesContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 58, ArcParserRULE_namedConfigValues)
	var _la int

	var _alt int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(396)
		p.NamedConfigValue()
	}
	p.SetState(401)
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
				p.SetState(397)
				p.Match(ArcParserCOMMA)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}
			{
				p.SetState(398)
				p.NamedConfigValue()
			}

		}
		p.SetState(403)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 38, p.GetParserRuleContext())
		if p.HasError() {
			goto errorExit
		}
	}
	p.SetState(405)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ArcParserCOMMA {
		{
			p.SetState(404)
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

func (s *NamedConfigValueContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitNamedConfigValue(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) NamedConfigValue() (localctx INamedConfigValueContext) {
	localctx = NewNamedConfigValueContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 60, ArcParserRULE_namedConfigValue)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(407)
		p.Match(ArcParserIDENTIFIER)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(408)
		p.Match(ArcParserASSIGN)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(409)
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

func (s *AnonymousConfigValuesContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitAnonymousConfigValues(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) AnonymousConfigValues() (localctx IAnonymousConfigValuesContext) {
	localctx = NewAnonymousConfigValuesContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 62, ArcParserRULE_anonymousConfigValues)
	var _la int

	var _alt int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(411)
		p.Expression()
	}
	p.SetState(416)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 40, p.GetParserRuleContext())
	if p.HasError() {
		goto errorExit
	}
	for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
		if _alt == 1 {
			{
				p.SetState(412)
				p.Match(ArcParserCOMMA)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}
			{
				p.SetState(413)
				p.Expression()
			}

		}
		p.SetState(418)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 40, p.GetParserRuleContext())
		if p.HasError() {
			goto errorExit
		}
	}
	p.SetState(420)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ArcParserCOMMA {
		{
			p.SetState(419)
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

func (s *ArgumentsContext) Accept(visitor antlr.ParseTreeVisitor) interface{} {
	switch t := visitor.(type) {
	case ArcParserVisitor:
		return t.VisitArguments(s)

	default:
		return t.VisitChildren(s)
	}
}

func (p *ArcParser) Arguments() (localctx IArgumentsContext) {
	localctx = NewArgumentsContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 64, ArcParserRULE_arguments)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(422)
		p.Match(ArcParserLPAREN)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(424)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&8666614601674123264) != 0 {
		{
			p.SetState(423)
			p.ArgumentList()
		}

	}
	{
		p.SetState(426)
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
	p.EnterRule(localctx, 66, ArcParserRULE_argumentList)
	var _la int

	var _alt int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(428)
		p.Expression()
	}
	p.SetState(433)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 43, p.GetParserRuleContext())
	if p.HasError() {
		goto errorExit
	}
	for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
		if _alt == 1 {
			{
				p.SetState(429)
				p.Match(ArcParserCOMMA)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}
			{
				p.SetState(430)
				p.Expression()
			}

		}
		p.SetState(435)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 43, p.GetParserRuleContext())
		if p.HasError() {
			goto errorExit
		}
	}
	p.SetState(437)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ArcParserCOMMA {
		{
			p.SetState(436)
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
	p.EnterRule(localctx, 68, ArcParserRULE_block)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(439)
		p.Match(ArcParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(443)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&8666614601674123508) != 0 {
		{
			p.SetState(440)
			p.Statement()
		}

		p.SetState(445)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(446)
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
	p.EnterRule(localctx, 70, ArcParserRULE_statement)
	p.SetState(456)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 46, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(448)
			p.VariableDeclaration()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(449)
			p.Assignment()
		}

	case 3:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(450)
			p.IfStatement()
		}

	case 4:
		p.EnterOuterAlt(localctx, 4)
		{
			p.SetState(451)
			p.ForStatement()
		}

	case 5:
		p.EnterOuterAlt(localctx, 5)
		{
			p.SetState(452)
			p.BreakStatement()
		}

	case 6:
		p.EnterOuterAlt(localctx, 6)
		{
			p.SetState(453)
			p.ContinueStatement()
		}

	case 7:
		p.EnterOuterAlt(localctx, 7)
		{
			p.SetState(454)
			p.ReturnStatement()
		}

	case 8:
		p.EnterOuterAlt(localctx, 8)
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
	p.EnterRule(localctx, 72, ArcParserRULE_variableDeclaration)
	p.SetState(460)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 47, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(458)
			p.LocalVariable()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(459)
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
	p.EnterRule(localctx, 74, ArcParserRULE_localVariable)
	p.SetState(470)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 48, p.GetParserRuleContext()) {
	case 1:
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
			p.Match(ArcParserDECLARE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(464)
			p.Expression()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(465)
			p.Match(ArcParserIDENTIFIER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(466)
			p.Type_()
		}
		{
			p.SetState(467)
			p.Match(ArcParserDECLARE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(468)
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
	p.EnterRule(localctx, 76, ArcParserRULE_statefulVariable)
	p.SetState(480)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 49, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(472)
			p.Match(ArcParserIDENTIFIER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(473)
			p.Match(ArcParserSTATE_DECLARE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(474)
			p.Expression()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(475)
			p.Match(ArcParserIDENTIFIER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(476)
			p.Type_()
		}
		{
			p.SetState(477)
			p.Match(ArcParserSTATE_DECLARE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(478)
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
	p.EnterRule(localctx, 78, ArcParserRULE_assignment)
	p.SetState(499)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 50, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(482)
			p.Match(ArcParserIDENTIFIER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(483)
			p.Match(ArcParserASSIGN)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(484)
			p.Expression()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(485)
			p.Match(ArcParserIDENTIFIER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(486)
			p.IndexOrSlice()
		}
		{
			p.SetState(487)
			p.Match(ArcParserASSIGN)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(488)
			p.Expression()
		}

	case 3:
		p.EnterOuterAlt(localctx, 3)
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
			p.CompoundOp()
		}
		{
			p.SetState(492)
			p.Expression()
		}

	case 4:
		p.EnterOuterAlt(localctx, 4)
		{
			p.SetState(494)
			p.Match(ArcParserIDENTIFIER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(495)
			p.IndexOrSlice()
		}
		{
			p.SetState(496)
			p.CompoundOp()
		}
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
	p.EnterRule(localctx, 80, ArcParserRULE_compoundOp)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(501)
		_la = p.GetTokenStream().LA(1)

		if !((int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&33285996544) != 0) {
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
	p.EnterRule(localctx, 82, ArcParserRULE_ifStatement)
	var _la int

	var _alt int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(503)
		p.Match(ArcParserIF)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(504)
		p.Expression()
	}
	{
		p.SetState(505)
		p.Block()
	}
	p.SetState(509)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 51, p.GetParserRuleContext())
	if p.HasError() {
		goto errorExit
	}
	for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
		if _alt == 1 {
			{
				p.SetState(506)
				p.ElseIfClause()
			}

		}
		p.SetState(511)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 51, p.GetParserRuleContext())
		if p.HasError() {
			goto errorExit
		}
	}
	p.SetState(513)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ArcParserELSE {
		{
			p.SetState(512)
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
	p.EnterRule(localctx, 84, ArcParserRULE_elseIfClause)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(515)
		p.Match(ArcParserELSE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(516)
		p.Match(ArcParserIF)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(517)
		p.Expression()
	}
	{
		p.SetState(518)
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
	p.EnterRule(localctx, 86, ArcParserRULE_elseClause)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(520)
		p.Match(ArcParserELSE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(521)
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
	p.EnterRule(localctx, 88, ArcParserRULE_forStatement)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(523)
		p.Match(ArcParserFOR)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(524)
		p.ForClause()
	}
	{
		p.SetState(525)
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
	p.EnterRule(localctx, 90, ArcParserRULE_forClause)
	p.SetState(537)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 53, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
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
			p.Match(ArcParserCOMMA)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(529)
			p.Match(ArcParserIDENTIFIER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(530)
			p.Match(ArcParserDECLARE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(531)
			p.Expression()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
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
			p.Match(ArcParserDECLARE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(534)
			p.Expression()
		}

	case 3:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(535)
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
	p.EnterRule(localctx, 92, ArcParserRULE_breakStatement)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(539)
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
	p.EnterRule(localctx, 94, ArcParserRULE_continueStatement)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(541)
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
	p.EnterRule(localctx, 96, ArcParserRULE_returnStatement)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(543)
		p.Match(ArcParserRETURN)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(545)
	p.GetErrorHandler().Sync(p)

	if p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 54, p.GetParserRuleContext()) == 1 {
		{
			p.SetState(544)
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
	p.EnterRule(localctx, 98, ArcParserRULE_type)
	var _la int

	p.SetState(553)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case ArcParserI8, ArcParserI16, ArcParserI32, ArcParserI64, ArcParserU8, ArcParserU16, ArcParserU32, ArcParserU64, ArcParserF32, ArcParserF64, ArcParserSTR:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(547)
			p.PrimitiveType()
		}
		p.SetState(549)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		if _la == ArcParserIDENTIFIER {
			{
				p.SetState(548)
				p.UnitSuffix()
			}

		}

	case ArcParserCHAN:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(551)
			p.ChannelType()
		}

	case ArcParserSERIES:
		p.EnterOuterAlt(localctx, 3)
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
	p.EnterRule(localctx, 100, ArcParserRULE_unitSuffix)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(555)
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
	p.EnterRule(localctx, 102, ArcParserRULE_primitiveType)
	p.SetState(559)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case ArcParserI8, ArcParserI16, ArcParserI32, ArcParserI64, ArcParserU8, ArcParserU16, ArcParserU32, ArcParserU64, ArcParserF32, ArcParserF64:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(557)
			p.NumericType()
		}

	case ArcParserSTR:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(558)
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
	p.EnterRule(localctx, 104, ArcParserRULE_numericType)
	p.SetState(563)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case ArcParserI8, ArcParserI16, ArcParserI32, ArcParserI64, ArcParserU8, ArcParserU16, ArcParserU32, ArcParserU64:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(561)
			p.IntegerType()
		}

	case ArcParserF32, ArcParserF64:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(562)
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
	p.EnterRule(localctx, 106, ArcParserRULE_integerType)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(565)
		_la = p.GetTokenStream().LA(1)

		if !((int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&2088960) != 0) {
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
	p.EnterRule(localctx, 108, ArcParserRULE_floatType)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(567)
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
	p.EnterRule(localctx, 110, ArcParserRULE_channelType)
	var _la int

	p.SetState(576)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 60, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(569)
			p.Match(ArcParserCHAN)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(570)
			p.PrimitiveType()
		}
		p.SetState(572)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		if _la == ArcParserIDENTIFIER {
			{
				p.SetState(571)
				p.UnitSuffix()
			}

		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(574)
			p.Match(ArcParserCHAN)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(575)
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
	p.EnterRule(localctx, 112, ArcParserRULE_seriesType)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(578)
		p.Match(ArcParserSERIES)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(579)
		p.PrimitiveType()
	}
	p.SetState(581)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ArcParserIDENTIFIER {
		{
			p.SetState(580)
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
	p.EnterRule(localctx, 114, ArcParserRULE_expression)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(583)
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
	p.EnterRule(localctx, 116, ArcParserRULE_logicalOrExpression)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(585)
		p.LogicalAndExpression()
	}
	p.SetState(590)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == ArcParserOR {
		{
			p.SetState(586)
			p.Match(ArcParserOR)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(587)
			p.LogicalAndExpression()
		}

		p.SetState(592)
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
	p.EnterRule(localctx, 118, ArcParserRULE_logicalAndExpression)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(593)
		p.EqualityExpression()
	}
	p.SetState(598)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == ArcParserAND {
		{
			p.SetState(594)
			p.Match(ArcParserAND)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(595)
			p.EqualityExpression()
		}

		p.SetState(600)
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
	p.EnterRule(localctx, 120, ArcParserRULE_equalityExpression)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(601)
		p.RelationalExpression()
	}
	p.SetState(606)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == ArcParserEQ || _la == ArcParserNEQ {
		{
			p.SetState(602)
			_la = p.GetTokenStream().LA(1)

			if !(_la == ArcParserEQ || _la == ArcParserNEQ) {
				p.GetErrorHandler().RecoverInline(p)
			} else {
				p.GetErrorHandler().ReportMatch(p)
				p.Consume()
			}
		}
		{
			p.SetState(603)
			p.RelationalExpression()
		}

		p.SetState(608)
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
	p.EnterRule(localctx, 122, ArcParserRULE_relationalExpression)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(609)
		p.AdditiveExpression()
	}
	p.SetState(614)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&131941395333120) != 0 {
		{
			p.SetState(610)
			_la = p.GetTokenStream().LA(1)

			if !((int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&131941395333120) != 0) {
				p.GetErrorHandler().RecoverInline(p)
			} else {
				p.GetErrorHandler().ReportMatch(p)
				p.Consume()
			}
		}
		{
			p.SetState(611)
			p.AdditiveExpression()
		}

		p.SetState(616)
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
	p.EnterRule(localctx, 124, ArcParserRULE_additiveExpression)
	var _la int

	var _alt int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(617)
		p.MultiplicativeExpression()
	}
	p.SetState(622)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 66, p.GetParserRuleContext())
	if p.HasError() {
		goto errorExit
	}
	for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
		if _alt == 1 {
			{
				p.SetState(618)
				_la = p.GetTokenStream().LA(1)

				if !(_la == ArcParserPLUS || _la == ArcParserMINUS) {
					p.GetErrorHandler().RecoverInline(p)
				} else {
					p.GetErrorHandler().ReportMatch(p)
					p.Consume()
				}
			}
			{
				p.SetState(619)
				p.MultiplicativeExpression()
			}

		}
		p.SetState(624)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 66, p.GetParserRuleContext())
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
	p.EnterRule(localctx, 126, ArcParserRULE_multiplicativeExpression)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(625)
		p.PowerExpression()
	}
	p.SetState(630)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&962072674304) != 0 {
		{
			p.SetState(626)
			_la = p.GetTokenStream().LA(1)

			if !((int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&962072674304) != 0) {
				p.GetErrorHandler().RecoverInline(p)
			} else {
				p.GetErrorHandler().ReportMatch(p)
				p.Consume()
			}
		}
		{
			p.SetState(627)
			p.PowerExpression()
		}

		p.SetState(632)
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
	p.EnterRule(localctx, 128, ArcParserRULE_powerExpression)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(633)
		p.UnaryExpression()
	}
	p.SetState(636)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ArcParserCARET {
		{
			p.SetState(634)
			p.Match(ArcParserCARET)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(635)
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
	p.EnterRule(localctx, 130, ArcParserRULE_unaryExpression)
	p.SetState(643)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 69, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(638)
			p.Match(ArcParserMINUS)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(639)
			p.UnaryExpression()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(640)
			p.Match(ArcParserNOT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(641)
			p.UnaryExpression()
		}

	case 3:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(642)
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
	p.EnterRule(localctx, 132, ArcParserRULE_postfixExpression)
	var _alt int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(645)
		p.PrimaryExpression()
	}
	p.SetState(650)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 71, p.GetParserRuleContext())
	if p.HasError() {
		goto errorExit
	}
	for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
		if _alt == 1 {
			p.SetState(648)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}

			switch p.GetTokenStream().LA(1) {
			case ArcParserLBRACKET:
				{
					p.SetState(646)
					p.IndexOrSlice()
				}

			case ArcParserLPAREN:
				{
					p.SetState(647)
					p.FunctionCallSuffix()
				}

			default:
				p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
				goto errorExit
			}

		}
		p.SetState(652)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 71, p.GetParserRuleContext())
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
	p.EnterRule(localctx, 134, ArcParserRULE_indexOrSlice)
	var _la int

	p.SetState(666)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 74, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(653)
			p.Match(ArcParserLBRACKET)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(654)
			p.Expression()
		}
		{
			p.SetState(655)
			p.Match(ArcParserRBRACKET)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(657)
			p.Match(ArcParserLBRACKET)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		p.SetState(659)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		if (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&8666614601674123264) != 0 {
			{
				p.SetState(658)
				p.Expression()
			}

		}
		{
			p.SetState(661)
			p.Match(ArcParserCOLON)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		p.SetState(663)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		if (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&8666614601674123264) != 0 {
			{
				p.SetState(662)
				p.Expression()
			}

		}
		{
			p.SetState(665)
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
	p.EnterRule(localctx, 136, ArcParserRULE_functionCallSuffix)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(668)
		p.Match(ArcParserLPAREN)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(670)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&8666614601674123264) != 0 {
		{
			p.SetState(669)
			p.ArgumentList()
		}

	}
	{
		p.SetState(672)
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
	p.EnterRule(localctx, 138, ArcParserRULE_primaryExpression)
	p.SetState(682)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 76, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(674)
			p.Literal()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(675)
			p.QualifiedIdentifier()
		}

	case 3:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(676)
			p.Match(ArcParserIDENTIFIER)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 4:
		p.EnterOuterAlt(localctx, 4)
		{
			p.SetState(677)
			p.Match(ArcParserLPAREN)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(678)
			p.Expression()
		}
		{
			p.SetState(679)
			p.Match(ArcParserRPAREN)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 5:
		p.EnterOuterAlt(localctx, 5)
		{
			p.SetState(681)
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
	p.EnterRule(localctx, 140, ArcParserRULE_typeCast)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(684)
		p.Type_()
	}
	{
		p.SetState(685)
		p.Match(ArcParserLPAREN)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(686)
		p.Expression()
	}
	{
		p.SetState(687)
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
	p.EnterRule(localctx, 142, ArcParserRULE_literal)
	p.SetState(692)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case ArcParserMINUS, ArcParserINTEGER_LITERAL, ArcParserFLOAT_LITERAL:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(689)
			p.NumericLiteral()
		}

	case ArcParserSTR_LITERAL:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(690)
			p.Match(ArcParserSTR_LITERAL)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case ArcParserLBRACKET:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(691)
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
	p.EnterRule(localctx, 144, ArcParserRULE_numericLiteral)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(695)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ArcParserMINUS {
		{
			p.SetState(694)
			p.Match(ArcParserMINUS)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	}
	{
		p.SetState(697)
		_la = p.GetTokenStream().LA(1)

		if !(_la == ArcParserINTEGER_LITERAL || _la == ArcParserFLOAT_LITERAL) {
			p.GetErrorHandler().RecoverInline(p)
		} else {
			p.GetErrorHandler().ReportMatch(p)
			p.Consume()
		}
	}
	p.SetState(700)
	p.GetErrorHandler().Sync(p)

	if p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 79, p.GetParserRuleContext()) == 1 {
		p.SetState(698)

		if !(p.TokensAdjacent(p.GetTokenStream().LT(-1), p.GetTokenStream().LT(1))) {
			p.SetError(antlr.NewFailedPredicateException(p, "p.TokensAdjacent(p.GetTokenStream().LT(-1), p.GetTokenStream().LT(1))", ""))
			goto errorExit
		}
		{
			p.SetState(699)
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
	p.EnterRule(localctx, 146, ArcParserRULE_seriesLiteral)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(702)
		p.Match(ArcParserLBRACKET)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(704)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&8666614601674123264) != 0 {
		{
			p.SetState(703)
			p.ExpressionList()
		}

	}
	{
		p.SetState(706)
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
	p.EnterRule(localctx, 148, ArcParserRULE_expressionList)
	var _la int

	var _alt int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(708)
		p.Expression()
	}
	p.SetState(713)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 81, p.GetParserRuleContext())
	if p.HasError() {
		goto errorExit
	}
	for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
		if _alt == 1 {
			{
				p.SetState(709)
				p.Match(ArcParserCOMMA)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}
			{
				p.SetState(710)
				p.Expression()
			}

		}
		p.SetState(715)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 81, p.GetParserRuleContext())
		if p.HasError() {
			goto errorExit
		}
	}
	p.SetState(717)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == ArcParserCOMMA {
		{
			p.SetState(716)
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
	case 72:
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
