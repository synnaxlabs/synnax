// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Code generated from OracleParser.g4 by ANTLR 4.13.2. DO NOT EDIT.

package parser // OracleParser
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

type OracleParser struct {
	*antlr.BaseParser
}

var OracleParserParserStaticData struct {
	once                   sync.Once
	serializedATN          []int32
	LiteralNames           []string
	SymbolicNames          []string
	RuleNames              []string
	PredictionContextCache *antlr.PredictionContextCache
	atn                    *antlr.ATN
	decisionToDFA          []*antlr.DFA
}

func oracleparserParserInit() {
	staticData := &OracleParserParserStaticData
	staticData.LiteralNames = []string{
		"", "'struct'", "'enum'", "'import'", "'extends'", "'map'", "'action'",
		"'union'", "'{'", "'}'", "'['", "']'", "'<'", "'>'", "','", "'?'", "'.'",
		"'='", "'@'", "'-'",
	}
	staticData.SymbolicNames = []string{
		"", "STRUCT", "ENUM", "IMPORT", "EXTENDS", "MAP", "ACTION", "UNION",
		"LBRACE", "RBRACE", "LBRACKET", "RBRACKET", "LT", "GT", "COMMA", "QUESTION",
		"DOT", "EQUALS", "AT", "MINUS", "TRIPLE_STRING_LIT", "STRING_LIT", "FLOAT_LIT",
		"INT_LIT", "BOOL_LIT", "IDENT", "LINE_COMMENT", "BLOCK_COMMENT", "NEWLINE",
		"WS",
	}
	staticData.RuleNames = []string{
		"schema", "nl", "importStmt", "fileDomain", "definition", "structDef",
		"typeRefList", "aliasBody", "typeParams", "typeParam", "structBody",
		"fieldOmit", "actionDef", "actionBody", "fieldDef", "fieldDefault",
		"arrayDefault", "inlineDomain", "fieldBody", "domain", "domainContent",
		"domainBlock", "typeRef", "arrayModifier", "mapType", "typeArgs", "typeModifiers",
		"qualifiedIdent", "expression", "expressionValue", "enumDef", "enumBody",
		"enumValue", "enumValueBody", "typeDefDef", "typeDefBody", "unionDef",
		"unionBody", "unionVariant", "variantName", "unionVariantBody",
	}
	staticData.PredictionContextCache = antlr.NewPredictionContextCache()
	staticData.serializedATN = []int32{
		4, 1, 29, 685, 2, 0, 7, 0, 2, 1, 7, 1, 2, 2, 7, 2, 2, 3, 7, 3, 2, 4, 7,
		4, 2, 5, 7, 5, 2, 6, 7, 6, 2, 7, 7, 7, 2, 8, 7, 8, 2, 9, 7, 9, 2, 10, 7,
		10, 2, 11, 7, 11, 2, 12, 7, 12, 2, 13, 7, 13, 2, 14, 7, 14, 2, 15, 7, 15,
		2, 16, 7, 16, 2, 17, 7, 17, 2, 18, 7, 18, 2, 19, 7, 19, 2, 20, 7, 20, 2,
		21, 7, 21, 2, 22, 7, 22, 2, 23, 7, 23, 2, 24, 7, 24, 2, 25, 7, 25, 2, 26,
		7, 26, 2, 27, 7, 27, 2, 28, 7, 28, 2, 29, 7, 29, 2, 30, 7, 30, 2, 31, 7,
		31, 2, 32, 7, 32, 2, 33, 7, 33, 2, 34, 7, 34, 2, 35, 7, 35, 2, 36, 7, 36,
		2, 37, 7, 37, 2, 38, 7, 38, 2, 39, 7, 39, 2, 40, 7, 40, 1, 0, 5, 0, 84,
		8, 0, 10, 0, 12, 0, 87, 9, 0, 1, 0, 1, 0, 5, 0, 91, 8, 0, 10, 0, 12, 0,
		94, 9, 0, 5, 0, 96, 8, 0, 10, 0, 12, 0, 99, 9, 0, 1, 0, 1, 0, 5, 0, 103,
		8, 0, 10, 0, 12, 0, 106, 9, 0, 5, 0, 108, 8, 0, 10, 0, 12, 0, 111, 9, 0,
		1, 0, 1, 0, 5, 0, 115, 8, 0, 10, 0, 12, 0, 118, 9, 0, 5, 0, 120, 8, 0,
		10, 0, 12, 0, 123, 9, 0, 1, 0, 1, 0, 1, 1, 1, 1, 1, 2, 1, 2, 1, 2, 1, 3,
		1, 3, 1, 3, 3, 3, 135, 8, 3, 1, 4, 1, 4, 1, 4, 1, 4, 3, 4, 141, 8, 4, 1,
		5, 1, 5, 1, 5, 3, 5, 146, 8, 5, 1, 5, 1, 5, 3, 5, 150, 8, 5, 1, 5, 5, 5,
		153, 8, 5, 10, 5, 12, 5, 156, 9, 5, 1, 5, 1, 5, 5, 5, 160, 8, 5, 10, 5,
		12, 5, 163, 9, 5, 1, 5, 1, 5, 1, 5, 1, 5, 1, 5, 3, 5, 170, 8, 5, 1, 5,
		1, 5, 1, 5, 3, 5, 175, 8, 5, 3, 5, 177, 8, 5, 1, 6, 1, 6, 1, 6, 5, 6, 182,
		8, 6, 10, 6, 12, 6, 185, 9, 6, 1, 6, 5, 6, 188, 8, 6, 10, 6, 12, 6, 191,
		9, 6, 1, 7, 5, 7, 194, 8, 7, 10, 7, 12, 7, 197, 9, 7, 1, 7, 1, 7, 5, 7,
		201, 8, 7, 10, 7, 12, 7, 204, 9, 7, 1, 7, 1, 7, 5, 7, 208, 8, 7, 10, 7,
		12, 7, 211, 9, 7, 5, 7, 213, 8, 7, 10, 7, 12, 7, 216, 9, 7, 1, 7, 1, 7,
		1, 8, 1, 8, 5, 8, 222, 8, 8, 10, 8, 12, 8, 225, 9, 8, 1, 8, 1, 8, 1, 8,
		5, 8, 230, 8, 8, 10, 8, 12, 8, 233, 9, 8, 1, 8, 5, 8, 236, 8, 8, 10, 8,
		12, 8, 239, 9, 8, 1, 8, 5, 8, 242, 8, 8, 10, 8, 12, 8, 245, 9, 8, 1, 8,
		1, 8, 1, 9, 1, 9, 3, 9, 251, 8, 9, 1, 9, 1, 9, 3, 9, 255, 8, 9, 1, 9, 1,
		9, 3, 9, 259, 8, 9, 1, 10, 1, 10, 1, 10, 1, 10, 3, 10, 265, 8, 10, 1, 10,
		5, 10, 268, 8, 10, 10, 10, 12, 10, 271, 9, 10, 5, 10, 273, 8, 10, 10, 10,
		12, 10, 276, 9, 10, 1, 11, 1, 11, 1, 11, 1, 12, 1, 12, 1, 12, 5, 12, 284,
		8, 12, 10, 12, 12, 12, 287, 9, 12, 1, 12, 1, 12, 5, 12, 291, 8, 12, 10,
		12, 12, 12, 294, 9, 12, 1, 12, 1, 12, 1, 12, 1, 13, 1, 13, 3, 13, 301,
		8, 13, 1, 13, 5, 13, 304, 8, 13, 10, 13, 12, 13, 307, 9, 13, 5, 13, 309,
		8, 13, 10, 13, 12, 13, 312, 9, 13, 1, 14, 1, 14, 1, 14, 1, 14, 3, 14, 318,
		8, 14, 1, 14, 5, 14, 321, 8, 14, 10, 14, 12, 14, 324, 9, 14, 1, 14, 3,
		14, 327, 8, 14, 1, 15, 1, 15, 3, 15, 331, 8, 15, 1, 16, 1, 16, 5, 16, 335,
		8, 16, 10, 16, 12, 16, 338, 9, 16, 1, 16, 1, 16, 1, 16, 5, 16, 343, 8,
		16, 10, 16, 12, 16, 346, 9, 16, 1, 16, 5, 16, 349, 8, 16, 10, 16, 12, 16,
		352, 9, 16, 1, 16, 5, 16, 355, 8, 16, 10, 16, 12, 16, 358, 9, 16, 3, 16,
		360, 8, 16, 1, 16, 1, 16, 1, 17, 1, 17, 1, 17, 3, 17, 367, 8, 17, 1, 18,
		5, 18, 370, 8, 18, 10, 18, 12, 18, 373, 9, 18, 1, 18, 1, 18, 5, 18, 377,
		8, 18, 10, 18, 12, 18, 380, 9, 18, 1, 18, 1, 18, 5, 18, 384, 8, 18, 10,
		18, 12, 18, 387, 9, 18, 5, 18, 389, 8, 18, 10, 18, 12, 18, 392, 9, 18,
		1, 18, 1, 18, 1, 19, 1, 19, 1, 19, 3, 19, 399, 8, 19, 1, 20, 1, 20, 3,
		20, 403, 8, 20, 1, 21, 5, 21, 406, 8, 21, 10, 21, 12, 21, 409, 9, 21, 1,
		21, 1, 21, 5, 21, 413, 8, 21, 10, 21, 12, 21, 416, 9, 21, 1, 21, 1, 21,
		4, 21, 420, 8, 21, 11, 21, 12, 21, 421, 1, 21, 1, 21, 5, 21, 426, 8, 21,
		10, 21, 12, 21, 429, 9, 21, 3, 21, 431, 8, 21, 1, 21, 5, 21, 434, 8, 21,
		10, 21, 12, 21, 437, 9, 21, 1, 21, 1, 21, 1, 22, 1, 22, 3, 22, 443, 8,
		22, 1, 22, 1, 22, 3, 22, 447, 8, 22, 1, 22, 3, 22, 450, 8, 22, 1, 22, 3,
		22, 453, 8, 22, 3, 22, 455, 8, 22, 1, 23, 1, 23, 1, 23, 1, 23, 1, 23, 3,
		23, 462, 8, 23, 1, 24, 1, 24, 1, 24, 1, 24, 1, 24, 1, 24, 1, 24, 1, 25,
		1, 25, 1, 25, 1, 25, 5, 25, 475, 8, 25, 10, 25, 12, 25, 478, 9, 25, 1,
		25, 1, 25, 1, 26, 1, 26, 1, 26, 3, 26, 485, 8, 26, 1, 27, 1, 27, 1, 27,
		3, 27, 490, 8, 27, 1, 28, 1, 28, 5, 28, 494, 8, 28, 10, 28, 12, 28, 497,
		9, 28, 1, 29, 1, 29, 1, 29, 1, 29, 1, 29, 1, 29, 3, 29, 505, 8, 29, 1,
		30, 1, 30, 1, 30, 1, 30, 3, 30, 511, 8, 30, 1, 30, 5, 30, 514, 8, 30, 10,
		30, 12, 30, 517, 9, 30, 1, 30, 1, 30, 5, 30, 521, 8, 30, 10, 30, 12, 30,
		524, 9, 30, 1, 30, 1, 30, 1, 30, 1, 31, 1, 31, 3, 31, 531, 8, 31, 1, 31,
		5, 31, 534, 8, 31, 10, 31, 12, 31, 537, 9, 31, 5, 31, 539, 8, 31, 10, 31,
		12, 31, 542, 9, 31, 1, 32, 1, 32, 1, 32, 1, 32, 3, 32, 548, 8, 32, 1, 33,
		5, 33, 551, 8, 33, 10, 33, 12, 33, 554, 9, 33, 1, 33, 1, 33, 5, 33, 558,
		8, 33, 10, 33, 12, 33, 561, 9, 33, 1, 33, 1, 33, 5, 33, 565, 8, 33, 10,
		33, 12, 33, 568, 9, 33, 5, 33, 570, 8, 33, 10, 33, 12, 33, 573, 9, 33,
		1, 33, 1, 33, 1, 34, 1, 34, 3, 34, 579, 8, 34, 1, 34, 1, 34, 3, 34, 583,
		8, 34, 1, 35, 5, 35, 586, 8, 35, 10, 35, 12, 35, 589, 9, 35, 1, 35, 1,
		35, 5, 35, 593, 8, 35, 10, 35, 12, 35, 596, 9, 35, 1, 35, 1, 35, 5, 35,
		600, 8, 35, 10, 35, 12, 35, 603, 9, 35, 5, 35, 605, 8, 35, 10, 35, 12,
		35, 608, 9, 35, 1, 35, 1, 35, 1, 36, 1, 36, 1, 36, 1, 36, 1, 36, 1, 36,
		3, 36, 618, 8, 36, 1, 36, 5, 36, 621, 8, 36, 10, 36, 12, 36, 624, 9, 36,
		1, 36, 1, 36, 5, 36, 628, 8, 36, 10, 36, 12, 36, 631, 9, 36, 1, 36, 1,
		36, 1, 36, 1, 37, 1, 37, 3, 37, 638, 8, 37, 1, 37, 5, 37, 641, 8, 37, 10,
		37, 12, 37, 644, 9, 37, 5, 37, 646, 8, 37, 10, 37, 12, 37, 649, 9, 37,
		1, 38, 1, 38, 1, 38, 3, 38, 654, 8, 38, 1, 39, 1, 39, 1, 40, 5, 40, 659,
		8, 40, 10, 40, 12, 40, 662, 9, 40, 1, 40, 1, 40, 5, 40, 666, 8, 40, 10,
		40, 12, 40, 669, 9, 40, 1, 40, 1, 40, 5, 40, 673, 8, 40, 10, 40, 12, 40,
		676, 9, 40, 5, 40, 678, 8, 40, 10, 40, 12, 40, 681, 9, 40, 1, 40, 1, 40,
		1, 40, 0, 0, 41, 0, 2, 4, 6, 8, 10, 12, 14, 16, 18, 20, 22, 24, 26, 28,
		30, 32, 34, 36, 38, 40, 42, 44, 46, 48, 50, 52, 54, 56, 58, 60, 62, 64,
		66, 68, 70, 72, 74, 76, 78, 80, 0, 2, 2, 0, 21, 21, 23, 23, 2, 0, 1, 7,
		25, 25, 749, 0, 85, 1, 0, 0, 0, 2, 126, 1, 0, 0, 0, 4, 128, 1, 0, 0, 0,
		6, 131, 1, 0, 0, 0, 8, 140, 1, 0, 0, 0, 10, 176, 1, 0, 0, 0, 12, 178, 1,
		0, 0, 0, 14, 195, 1, 0, 0, 0, 16, 219, 1, 0, 0, 0, 18, 248, 1, 0, 0, 0,
		20, 274, 1, 0, 0, 0, 22, 277, 1, 0, 0, 0, 24, 280, 1, 0, 0, 0, 26, 310,
		1, 0, 0, 0, 28, 313, 1, 0, 0, 0, 30, 330, 1, 0, 0, 0, 32, 332, 1, 0, 0,
		0, 34, 363, 1, 0, 0, 0, 36, 371, 1, 0, 0, 0, 38, 395, 1, 0, 0, 0, 40, 402,
		1, 0, 0, 0, 42, 407, 1, 0, 0, 0, 44, 454, 1, 0, 0, 0, 46, 461, 1, 0, 0,
		0, 48, 463, 1, 0, 0, 0, 50, 470, 1, 0, 0, 0, 52, 484, 1, 0, 0, 0, 54, 486,
		1, 0, 0, 0, 56, 491, 1, 0, 0, 0, 58, 504, 1, 0, 0, 0, 60, 506, 1, 0, 0,
		0, 62, 540, 1, 0, 0, 0, 64, 543, 1, 0, 0, 0, 66, 552, 1, 0, 0, 0, 68, 576,
		1, 0, 0, 0, 70, 587, 1, 0, 0, 0, 72, 611, 1, 0, 0, 0, 74, 647, 1, 0, 0,
		0, 76, 650, 1, 0, 0, 0, 78, 655, 1, 0, 0, 0, 80, 660, 1, 0, 0, 0, 82, 84,
		3, 2, 1, 0, 83, 82, 1, 0, 0, 0, 84, 87, 1, 0, 0, 0, 85, 83, 1, 0, 0, 0,
		85, 86, 1, 0, 0, 0, 86, 97, 1, 0, 0, 0, 87, 85, 1, 0, 0, 0, 88, 92, 3,
		4, 2, 0, 89, 91, 3, 2, 1, 0, 90, 89, 1, 0, 0, 0, 91, 94, 1, 0, 0, 0, 92,
		90, 1, 0, 0, 0, 92, 93, 1, 0, 0, 0, 93, 96, 1, 0, 0, 0, 94, 92, 1, 0, 0,
		0, 95, 88, 1, 0, 0, 0, 96, 99, 1, 0, 0, 0, 97, 95, 1, 0, 0, 0, 97, 98,
		1, 0, 0, 0, 98, 109, 1, 0, 0, 0, 99, 97, 1, 0, 0, 0, 100, 104, 3, 6, 3,
		0, 101, 103, 3, 2, 1, 0, 102, 101, 1, 0, 0, 0, 103, 106, 1, 0, 0, 0, 104,
		102, 1, 0, 0, 0, 104, 105, 1, 0, 0, 0, 105, 108, 1, 0, 0, 0, 106, 104,
		1, 0, 0, 0, 107, 100, 1, 0, 0, 0, 108, 111, 1, 0, 0, 0, 109, 107, 1, 0,
		0, 0, 109, 110, 1, 0, 0, 0, 110, 121, 1, 0, 0, 0, 111, 109, 1, 0, 0, 0,
		112, 116, 3, 8, 4, 0, 113, 115, 3, 2, 1, 0, 114, 113, 1, 0, 0, 0, 115,
		118, 1, 0, 0, 0, 116, 114, 1, 0, 0, 0, 116, 117, 1, 0, 0, 0, 117, 120,
		1, 0, 0, 0, 118, 116, 1, 0, 0, 0, 119, 112, 1, 0, 0, 0, 120, 123, 1, 0,
		0, 0, 121, 119, 1, 0, 0, 0, 121, 122, 1, 0, 0, 0, 122, 124, 1, 0, 0, 0,
		123, 121, 1, 0, 0, 0, 124, 125, 5, 0, 0, 1, 125, 1, 1, 0, 0, 0, 126, 127,
		5, 28, 0, 0, 127, 3, 1, 0, 0, 0, 128, 129, 5, 3, 0, 0, 129, 130, 5, 21,
		0, 0, 130, 5, 1, 0, 0, 0, 131, 132, 5, 18, 0, 0, 132, 134, 5, 25, 0, 0,
		133, 135, 3, 40, 20, 0, 134, 133, 1, 0, 0, 0, 134, 135, 1, 0, 0, 0, 135,
		7, 1, 0, 0, 0, 136, 141, 3, 10, 5, 0, 137, 141, 3, 60, 30, 0, 138, 141,
		3, 68, 34, 0, 139, 141, 3, 72, 36, 0, 140, 136, 1, 0, 0, 0, 140, 137, 1,
		0, 0, 0, 140, 138, 1, 0, 0, 0, 140, 139, 1, 0, 0, 0, 141, 9, 1, 0, 0, 0,
		142, 143, 5, 25, 0, 0, 143, 145, 5, 1, 0, 0, 144, 146, 3, 16, 8, 0, 145,
		144, 1, 0, 0, 0, 145, 146, 1, 0, 0, 0, 146, 149, 1, 0, 0, 0, 147, 148,
		5, 4, 0, 0, 148, 150, 3, 12, 6, 0, 149, 147, 1, 0, 0, 0, 149, 150, 1, 0,
		0, 0, 150, 154, 1, 0, 0, 0, 151, 153, 3, 2, 1, 0, 152, 151, 1, 0, 0, 0,
		153, 156, 1, 0, 0, 0, 154, 152, 1, 0, 0, 0, 154, 155, 1, 0, 0, 0, 155,
		157, 1, 0, 0, 0, 156, 154, 1, 0, 0, 0, 157, 161, 5, 8, 0, 0, 158, 160,
		3, 2, 1, 0, 159, 158, 1, 0, 0, 0, 160, 163, 1, 0, 0, 0, 161, 159, 1, 0,
		0, 0, 161, 162, 1, 0, 0, 0, 162, 164, 1, 0, 0, 0, 163, 161, 1, 0, 0, 0,
		164, 165, 3, 20, 10, 0, 165, 166, 5, 9, 0, 0, 166, 177, 1, 0, 0, 0, 167,
		169, 5, 25, 0, 0, 168, 170, 3, 16, 8, 0, 169, 168, 1, 0, 0, 0, 169, 170,
		1, 0, 0, 0, 170, 171, 1, 0, 0, 0, 171, 172, 5, 17, 0, 0, 172, 174, 3, 44,
		22, 0, 173, 175, 3, 14, 7, 0, 174, 173, 1, 0, 0, 0, 174, 175, 1, 0, 0,
		0, 175, 177, 1, 0, 0, 0, 176, 142, 1, 0, 0, 0, 176, 167, 1, 0, 0, 0, 177,
		11, 1, 0, 0, 0, 178, 189, 3, 44, 22, 0, 179, 183, 5, 14, 0, 0, 180, 182,
		3, 2, 1, 0, 181, 180, 1, 0, 0, 0, 182, 185, 1, 0, 0, 0, 183, 181, 1, 0,
		0, 0, 183, 184, 1, 0, 0, 0, 184, 186, 1, 0, 0, 0, 185, 183, 1, 0, 0, 0,
		186, 188, 3, 44, 22, 0, 187, 179, 1, 0, 0, 0, 188, 191, 1, 0, 0, 0, 189,
		187, 1, 0, 0, 0, 189, 190, 1, 0, 0, 0, 190, 13, 1, 0, 0, 0, 191, 189, 1,
		0, 0, 0, 192, 194, 3, 2, 1, 0, 193, 192, 1, 0, 0, 0, 194, 197, 1, 0, 0,
		0, 195, 193, 1, 0, 0, 0, 195, 196, 1, 0, 0, 0, 196, 198, 1, 0, 0, 0, 197,
		195, 1, 0, 0, 0, 198, 202, 5, 8, 0, 0, 199, 201, 3, 2, 1, 0, 200, 199,
		1, 0, 0, 0, 201, 204, 1, 0, 0, 0, 202, 200, 1, 0, 0, 0, 202, 203, 1, 0,
		0, 0, 203, 214, 1, 0, 0, 0, 204, 202, 1, 0, 0, 0, 205, 209, 3, 38, 19,
		0, 206, 208, 3, 2, 1, 0, 207, 206, 1, 0, 0, 0, 208, 211, 1, 0, 0, 0, 209,
		207, 1, 0, 0, 0, 209, 210, 1, 0, 0, 0, 210, 213, 1, 0, 0, 0, 211, 209,
		1, 0, 0, 0, 212, 205, 1, 0, 0, 0, 213, 216, 1, 0, 0, 0, 214, 212, 1, 0,
		0, 0, 214, 215, 1, 0, 0, 0, 215, 217, 1, 0, 0, 0, 216, 214, 1, 0, 0, 0,
		217, 218, 5, 9, 0, 0, 218, 15, 1, 0, 0, 0, 219, 223, 5, 12, 0, 0, 220,
		222, 3, 2, 1, 0, 221, 220, 1, 0, 0, 0, 222, 225, 1, 0, 0, 0, 223, 221,
		1, 0, 0, 0, 223, 224, 1, 0, 0, 0, 224, 226, 1, 0, 0, 0, 225, 223, 1, 0,
		0, 0, 226, 237, 3, 18, 9, 0, 227, 231, 5, 14, 0, 0, 228, 230, 3, 2, 1,
		0, 229, 228, 1, 0, 0, 0, 230, 233, 1, 0, 0, 0, 231, 229, 1, 0, 0, 0, 231,
		232, 1, 0, 0, 0, 232, 234, 1, 0, 0, 0, 233, 231, 1, 0, 0, 0, 234, 236,
		3, 18, 9, 0, 235, 227, 1, 0, 0, 0, 236, 239, 1, 0, 0, 0, 237, 235, 1, 0,
		0, 0, 237, 238, 1, 0, 0, 0, 238, 243, 1, 0, 0, 0, 239, 237, 1, 0, 0, 0,
		240, 242, 3, 2, 1, 0, 241, 240, 1, 0, 0, 0, 242, 245, 1, 0, 0, 0, 243,
		241, 1, 0, 0, 0, 243, 244, 1, 0, 0, 0, 244, 246, 1, 0, 0, 0, 245, 243,
		1, 0, 0, 0, 246, 247, 5, 13, 0, 0, 247, 17, 1, 0, 0, 0, 248, 250, 5, 25,
		0, 0, 249, 251, 5, 15, 0, 0, 250, 249, 1, 0, 0, 0, 250, 251, 1, 0, 0, 0,
		251, 254, 1, 0, 0, 0, 252, 253, 5, 4, 0, 0, 253, 255, 3, 44, 22, 0, 254,
		252, 1, 0, 0, 0, 254, 255, 1, 0, 0, 0, 255, 258, 1, 0, 0, 0, 256, 257,
		5, 17, 0, 0, 257, 259, 3, 44, 22, 0, 258, 256, 1, 0, 0, 0, 258, 259, 1,
		0, 0, 0, 259, 19, 1, 0, 0, 0, 260, 265, 3, 28, 14, 0, 261, 265, 3, 22,
		11, 0, 262, 265, 3, 24, 12, 0, 263, 265, 3, 38, 19, 0, 264, 260, 1, 0,
		0, 0, 264, 261, 1, 0, 0, 0, 264, 262, 1, 0, 0, 0, 264, 263, 1, 0, 0, 0,
		265, 269, 1, 0, 0, 0, 266, 268, 3, 2, 1, 0, 267, 266, 1, 0, 0, 0, 268,
		271, 1, 0, 0, 0, 269, 267, 1, 0, 0, 0, 269, 270, 1, 0, 0, 0, 270, 273,
		1, 0, 0, 0, 271, 269, 1, 0, 0, 0, 272, 264, 1, 0, 0, 0, 273, 276, 1, 0,
		0, 0, 274, 272, 1, 0, 0, 0, 274, 275, 1, 0, 0, 0, 275, 21, 1, 0, 0, 0,
		276, 274, 1, 0, 0, 0, 277, 278, 5, 19, 0, 0, 278, 279, 5, 25, 0, 0, 279,
		23, 1, 0, 0, 0, 280, 281, 5, 6, 0, 0, 281, 285, 5, 25, 0, 0, 282, 284,
		3, 2, 1, 0, 283, 282, 1, 0, 0, 0, 284, 287, 1, 0, 0, 0, 285, 283, 1, 0,
		0, 0, 285, 286, 1, 0, 0, 0, 286, 288, 1, 0, 0, 0, 287, 285, 1, 0, 0, 0,
		288, 292, 5, 8, 0, 0, 289, 291, 3, 2, 1, 0, 290, 289, 1, 0, 0, 0, 291,
		294, 1, 0, 0, 0, 292, 290, 1, 0, 0, 0, 292, 293, 1, 0, 0, 0, 293, 295,
		1, 0, 0, 0, 294, 292, 1, 0, 0, 0, 295, 296, 3, 26, 13, 0, 296, 297, 5,
		9, 0, 0, 297, 25, 1, 0, 0, 0, 298, 301, 3, 28, 14, 0, 299, 301, 3, 38,
		19, 0, 300, 298, 1, 0, 0, 0, 300, 299, 1, 0, 0, 0, 301, 305, 1, 0, 0, 0,
		302, 304, 3, 2, 1, 0, 303, 302, 1, 0, 0, 0, 304, 307, 1, 0, 0, 0, 305,
		303, 1, 0, 0, 0, 305, 306, 1, 0, 0, 0, 306, 309, 1, 0, 0, 0, 307, 305,
		1, 0, 0, 0, 308, 300, 1, 0, 0, 0, 309, 312, 1, 0, 0, 0, 310, 308, 1, 0,
		0, 0, 310, 311, 1, 0, 0, 0, 311, 27, 1, 0, 0, 0, 312, 310, 1, 0, 0, 0,
		313, 314, 5, 25, 0, 0, 314, 317, 3, 44, 22, 0, 315, 316, 5, 17, 0, 0, 316,
		318, 3, 30, 15, 0, 317, 315, 1, 0, 0, 0, 317, 318, 1, 0, 0, 0, 318, 322,
		1, 0, 0, 0, 319, 321, 3, 34, 17, 0, 320, 319, 1, 0, 0, 0, 321, 324, 1,
		0, 0, 0, 322, 320, 1, 0, 0, 0, 322, 323, 1, 0, 0, 0, 323, 326, 1, 0, 0,
		0, 324, 322, 1, 0, 0, 0, 325, 327, 3, 36, 18, 0, 326, 325, 1, 0, 0, 0,
		326, 327, 1, 0, 0, 0, 327, 29, 1, 0, 0, 0, 328, 331, 3, 58, 29, 0, 329,
		331, 3, 32, 16, 0, 330, 328, 1, 0, 0, 0, 330, 329, 1, 0, 0, 0, 331, 31,
		1, 0, 0, 0, 332, 336, 5, 10, 0, 0, 333, 335, 3, 2, 1, 0, 334, 333, 1, 0,
		0, 0, 335, 338, 1, 0, 0, 0, 336, 334, 1, 0, 0, 0, 336, 337, 1, 0, 0, 0,
		337, 359, 1, 0, 0, 0, 338, 336, 1, 0, 0, 0, 339, 350, 3, 58, 29, 0, 340,
		344, 5, 14, 0, 0, 341, 343, 3, 2, 1, 0, 342, 341, 1, 0, 0, 0, 343, 346,
		1, 0, 0, 0, 344, 342, 1, 0, 0, 0, 344, 345, 1, 0, 0, 0, 345, 347, 1, 0,
		0, 0, 346, 344, 1, 0, 0, 0, 347, 349, 3, 58, 29, 0, 348, 340, 1, 0, 0,
		0, 349, 352, 1, 0, 0, 0, 350, 348, 1, 0, 0, 0, 350, 351, 1, 0, 0, 0, 351,
		356, 1, 0, 0, 0, 352, 350, 1, 0, 0, 0, 353, 355, 3, 2, 1, 0, 354, 353,
		1, 0, 0, 0, 355, 358, 1, 0, 0, 0, 356, 354, 1, 0, 0, 0, 356, 357, 1, 0,
		0, 0, 357, 360, 1, 0, 0, 0, 358, 356, 1, 0, 0, 0, 359, 339, 1, 0, 0, 0,
		359, 360, 1, 0, 0, 0, 360, 361, 1, 0, 0, 0, 361, 362, 5, 11, 0, 0, 362,
		33, 1, 0, 0, 0, 363, 364, 5, 18, 0, 0, 364, 366, 5, 25, 0, 0, 365, 367,
		3, 40, 20, 0, 366, 365, 1, 0, 0, 0, 366, 367, 1, 0, 0, 0, 367, 35, 1, 0,
		0, 0, 368, 370, 3, 2, 1, 0, 369, 368, 1, 0, 0, 0, 370, 373, 1, 0, 0, 0,
		371, 369, 1, 0, 0, 0, 371, 372, 1, 0, 0, 0, 372, 374, 1, 0, 0, 0, 373,
		371, 1, 0, 0, 0, 374, 378, 5, 8, 0, 0, 375, 377, 3, 2, 1, 0, 376, 375,
		1, 0, 0, 0, 377, 380, 1, 0, 0, 0, 378, 376, 1, 0, 0, 0, 378, 379, 1, 0,
		0, 0, 379, 390, 1, 0, 0, 0, 380, 378, 1, 0, 0, 0, 381, 385, 3, 38, 19,
		0, 382, 384, 3, 2, 1, 0, 383, 382, 1, 0, 0, 0, 384, 387, 1, 0, 0, 0, 385,
		383, 1, 0, 0, 0, 385, 386, 1, 0, 0, 0, 386, 389, 1, 0, 0, 0, 387, 385,
		1, 0, 0, 0, 388, 381, 1, 0, 0, 0, 389, 392, 1, 0, 0, 0, 390, 388, 1, 0,
		0, 0, 390, 391, 1, 0, 0, 0, 391, 393, 1, 0, 0, 0, 392, 390, 1, 0, 0, 0,
		393, 394, 5, 9, 0, 0, 394, 37, 1, 0, 0, 0, 395, 396, 5, 18, 0, 0, 396,
		398, 5, 25, 0, 0, 397, 399, 3, 40, 20, 0, 398, 397, 1, 0, 0, 0, 398, 399,
		1, 0, 0, 0, 399, 39, 1, 0, 0, 0, 400, 403, 3, 42, 21, 0, 401, 403, 3, 56,
		28, 0, 402, 400, 1, 0, 0, 0, 402, 401, 1, 0, 0, 0, 403, 41, 1, 0, 0, 0,
		404, 406, 3, 2, 1, 0, 405, 404, 1, 0, 0, 0, 406, 409, 1, 0, 0, 0, 407,
		405, 1, 0, 0, 0, 407, 408, 1, 0, 0, 0, 408, 410, 1, 0, 0, 0, 409, 407,
		1, 0, 0, 0, 410, 414, 5, 8, 0, 0, 411, 413, 3, 2, 1, 0, 412, 411, 1, 0,
		0, 0, 413, 416, 1, 0, 0, 0, 414, 412, 1, 0, 0, 0, 414, 415, 1, 0, 0, 0,
		415, 430, 1, 0, 0, 0, 416, 414, 1, 0, 0, 0, 417, 427, 3, 56, 28, 0, 418,
		420, 3, 2, 1, 0, 419, 418, 1, 0, 0, 0, 420, 421, 1, 0, 0, 0, 421, 419,
		1, 0, 0, 0, 421, 422, 1, 0, 0, 0, 422, 423, 1, 0, 0, 0, 423, 424, 3, 56,
		28, 0, 424, 426, 1, 0, 0, 0, 425, 419, 1, 0, 0, 0, 426, 429, 1, 0, 0, 0,
		427, 425, 1, 0, 0, 0, 427, 428, 1, 0, 0, 0, 428, 431, 1, 0, 0, 0, 429,
		427, 1, 0, 0, 0, 430, 417, 1, 0, 0, 0, 430, 431, 1, 0, 0, 0, 431, 435,
		1, 0, 0, 0, 432, 434, 3, 2, 1, 0, 433, 432, 1, 0, 0, 0, 434, 437, 1, 0,
		0, 0, 435, 433, 1, 0, 0, 0, 435, 436, 1, 0, 0, 0, 436, 438, 1, 0, 0, 0,
		437, 435, 1, 0, 0, 0, 438, 439, 5, 9, 0, 0, 439, 43, 1, 0, 0, 0, 440, 442,
		3, 48, 24, 0, 441, 443, 3, 52, 26, 0, 442, 441, 1, 0, 0, 0, 442, 443, 1,
		0, 0, 0, 443, 455, 1, 0, 0, 0, 444, 446, 3, 54, 27, 0, 445, 447, 3, 50,
		25, 0, 446, 445, 1, 0, 0, 0, 446, 447, 1, 0, 0, 0, 447, 449, 1, 0, 0, 0,
		448, 450, 3, 46, 23, 0, 449, 448, 1, 0, 0, 0, 449, 450, 1, 0, 0, 0, 450,
		452, 1, 0, 0, 0, 451, 453, 3, 52, 26, 0, 452, 451, 1, 0, 0, 0, 452, 453,
		1, 0, 0, 0, 453, 455, 1, 0, 0, 0, 454, 440, 1, 0, 0, 0, 454, 444, 1, 0,
		0, 0, 455, 45, 1, 0, 0, 0, 456, 457, 5, 10, 0, 0, 457, 462, 5, 11, 0, 0,
		458, 459, 5, 10, 0, 0, 459, 460, 5, 23, 0, 0, 460, 462, 5, 11, 0, 0, 461,
		456, 1, 0, 0, 0, 461, 458, 1, 0, 0, 0, 462, 47, 1, 0, 0, 0, 463, 464, 5,
		5, 0, 0, 464, 465, 5, 12, 0, 0, 465, 466, 3, 44, 22, 0, 466, 467, 5, 14,
		0, 0, 467, 468, 3, 44, 22, 0, 468, 469, 5, 13, 0, 0, 469, 49, 1, 0, 0,
		0, 470, 471, 5, 12, 0, 0, 471, 476, 3, 44, 22, 0, 472, 473, 5, 14, 0, 0,
		473, 475, 3, 44, 22, 0, 474, 472, 1, 0, 0, 0, 475, 478, 1, 0, 0, 0, 476,
		474, 1, 0, 0, 0, 476, 477, 1, 0, 0, 0, 477, 479, 1, 0, 0, 0, 478, 476,
		1, 0, 0, 0, 479, 480, 5, 13, 0, 0, 480, 51, 1, 0, 0, 0, 481, 482, 5, 15,
		0, 0, 482, 485, 5, 15, 0, 0, 483, 485, 5, 15, 0, 0, 484, 481, 1, 0, 0,
		0, 484, 483, 1, 0, 0, 0, 485, 53, 1, 0, 0, 0, 486, 489, 5, 25, 0, 0, 487,
		488, 5, 16, 0, 0, 488, 490, 5, 25, 0, 0, 489, 487, 1, 0, 0, 0, 489, 490,
		1, 0, 0, 0, 490, 55, 1, 0, 0, 0, 491, 495, 5, 25, 0, 0, 492, 494, 3, 58,
		29, 0, 493, 492, 1, 0, 0, 0, 494, 497, 1, 0, 0, 0, 495, 493, 1, 0, 0, 0,
		495, 496, 1, 0, 0, 0, 496, 57, 1, 0, 0, 0, 497, 495, 1, 0, 0, 0, 498, 505,
		5, 20, 0, 0, 499, 505, 5, 21, 0, 0, 500, 505, 5, 23, 0, 0, 501, 505, 5,
		22, 0, 0, 502, 505, 5, 24, 0, 0, 503, 505, 3, 54, 27, 0, 504, 498, 1, 0,
		0, 0, 504, 499, 1, 0, 0, 0, 504, 500, 1, 0, 0, 0, 504, 501, 1, 0, 0, 0,
		504, 502, 1, 0, 0, 0, 504, 503, 1, 0, 0, 0, 505, 59, 1, 0, 0, 0, 506, 507,
		5, 25, 0, 0, 507, 510, 5, 2, 0, 0, 508, 509, 5, 4, 0, 0, 509, 511, 3, 12,
		6, 0, 510, 508, 1, 0, 0, 0, 510, 511, 1, 0, 0, 0, 511, 515, 1, 0, 0, 0,
		512, 514, 3, 2, 1, 0, 513, 512, 1, 0, 0, 0, 514, 517, 1, 0, 0, 0, 515,
		513, 1, 0, 0, 0, 515, 516, 1, 0, 0, 0, 516, 518, 1, 0, 0, 0, 517, 515,
		1, 0, 0, 0, 518, 522, 5, 8, 0, 0, 519, 521, 3, 2, 1, 0, 520, 519, 1, 0,
		0, 0, 521, 524, 1, 0, 0, 0, 522, 520, 1, 0, 0, 0, 522, 523, 1, 0, 0, 0,
		523, 525, 1, 0, 0, 0, 524, 522, 1, 0, 0, 0, 525, 526, 3, 62, 31, 0, 526,
		527, 5, 9, 0, 0, 527, 61, 1, 0, 0, 0, 528, 531, 3, 64, 32, 0, 529, 531,
		3, 38, 19, 0, 530, 528, 1, 0, 0, 0, 530, 529, 1, 0, 0, 0, 531, 535, 1,
		0, 0, 0, 532, 534, 3, 2, 1, 0, 533, 532, 1, 0, 0, 0, 534, 537, 1, 0, 0,
		0, 535, 533, 1, 0, 0, 0, 535, 536, 1, 0, 0, 0, 536, 539, 1, 0, 0, 0, 537,
		535, 1, 0, 0, 0, 538, 530, 1, 0, 0, 0, 539, 542, 1, 0, 0, 0, 540, 538,
		1, 0, 0, 0, 540, 541, 1, 0, 0, 0, 541, 63, 1, 0, 0, 0, 542, 540, 1, 0,
		0, 0, 543, 544, 5, 25, 0, 0, 544, 545, 5, 17, 0, 0, 545, 547, 7, 0, 0,
		0, 546, 548, 3, 66, 33, 0, 547, 546, 1, 0, 0, 0, 547, 548, 1, 0, 0, 0,
		548, 65, 1, 0, 0, 0, 549, 551, 3, 2, 1, 0, 550, 549, 1, 0, 0, 0, 551, 554,
		1, 0, 0, 0, 552, 550, 1, 0, 0, 0, 552, 553, 1, 0, 0, 0, 553, 555, 1, 0,
		0, 0, 554, 552, 1, 0, 0, 0, 555, 559, 5, 8, 0, 0, 556, 558, 3, 2, 1, 0,
		557, 556, 1, 0, 0, 0, 558, 561, 1, 0, 0, 0, 559, 557, 1, 0, 0, 0, 559,
		560, 1, 0, 0, 0, 560, 571, 1, 0, 0, 0, 561, 559, 1, 0, 0, 0, 562, 566,
		3, 38, 19, 0, 563, 565, 3, 2, 1, 0, 564, 563, 1, 0, 0, 0, 565, 568, 1,
		0, 0, 0, 566, 564, 1, 0, 0, 0, 566, 567, 1, 0, 0, 0, 567, 570, 1, 0, 0,
		0, 568, 566, 1, 0, 0, 0, 569, 562, 1, 0, 0, 0, 570, 573, 1, 0, 0, 0, 571,
		569, 1, 0, 0, 0, 571, 572, 1, 0, 0, 0, 572, 574, 1, 0, 0, 0, 573, 571,
		1, 0, 0, 0, 574, 575, 5, 9, 0, 0, 575, 67, 1, 0, 0, 0, 576, 578, 5, 25,
		0, 0, 577, 579, 3, 16, 8, 0, 578, 577, 1, 0, 0, 0, 578, 579, 1, 0, 0, 0,
		579, 580, 1, 0, 0, 0, 580, 582, 3, 44, 22, 0, 581, 583, 3, 70, 35, 0, 582,
		581, 1, 0, 0, 0, 582, 583, 1, 0, 0, 0, 583, 69, 1, 0, 0, 0, 584, 586, 3,
		2, 1, 0, 585, 584, 1, 0, 0, 0, 586, 589, 1, 0, 0, 0, 587, 585, 1, 0, 0,
		0, 587, 588, 1, 0, 0, 0, 588, 590, 1, 0, 0, 0, 589, 587, 1, 0, 0, 0, 590,
		594, 5, 8, 0, 0, 591, 593, 3, 2, 1, 0, 592, 591, 1, 0, 0, 0, 593, 596,
		1, 0, 0, 0, 594, 592, 1, 0, 0, 0, 594, 595, 1, 0, 0, 0, 595, 606, 1, 0,
		0, 0, 596, 594, 1, 0, 0, 0, 597, 601, 3, 38, 19, 0, 598, 600, 3, 2, 1,
		0, 599, 598, 1, 0, 0, 0, 600, 603, 1, 0, 0, 0, 601, 599, 1, 0, 0, 0, 601,
		602, 1, 0, 0, 0, 602, 605, 1, 0, 0, 0, 603, 601, 1, 0, 0, 0, 604, 597,
		1, 0, 0, 0, 605, 608, 1, 0, 0, 0, 606, 604, 1, 0, 0, 0, 606, 607, 1, 0,
		0, 0, 607, 609, 1, 0, 0, 0, 608, 606, 1, 0, 0, 0, 609, 610, 5, 9, 0, 0,
		610, 71, 1, 0, 0, 0, 611, 612, 5, 25, 0, 0, 612, 613, 5, 7, 0, 0, 613,
		614, 5, 25, 0, 0, 614, 617, 5, 25, 0, 0, 615, 616, 5, 4, 0, 0, 616, 618,
		3, 12, 6, 0, 617, 615, 1, 0, 0, 0, 617, 618, 1, 0, 0, 0, 618, 622, 1, 0,
		0, 0, 619, 621, 3, 2, 1, 0, 620, 619, 1, 0, 0, 0, 621, 624, 1, 0, 0, 0,
		622, 620, 1, 0, 0, 0, 622, 623, 1, 0, 0, 0, 623, 625, 1, 0, 0, 0, 624,
		622, 1, 0, 0, 0, 625, 629, 5, 8, 0, 0, 626, 628, 3, 2, 1, 0, 627, 626,
		1, 0, 0, 0, 628, 631, 1, 0, 0, 0, 629, 627, 1, 0, 0, 0, 629, 630, 1, 0,
		0, 0, 630, 632, 1, 0, 0, 0, 631, 629, 1, 0, 0, 0, 632, 633, 3, 74, 37,
		0, 633, 634, 5, 9, 0, 0, 634, 73, 1, 0, 0, 0, 635, 638, 3, 76, 38, 0, 636,
		638, 3, 38, 19, 0, 637, 635, 1, 0, 0, 0, 637, 636, 1, 0, 0, 0, 638, 642,
		1, 0, 0, 0, 639, 641, 3, 2, 1, 0, 640, 639, 1, 0, 0, 0, 641, 644, 1, 0,
		0, 0, 642, 640, 1, 0, 0, 0, 642, 643, 1, 0, 0, 0, 643, 646, 1, 0, 0, 0,
		644, 642, 1, 0, 0, 0, 645, 637, 1, 0, 0, 0, 646, 649, 1, 0, 0, 0, 647,
		645, 1, 0, 0, 0, 647, 648, 1, 0, 0, 0, 648, 75, 1, 0, 0, 0, 649, 647, 1,
		0, 0, 0, 650, 651, 3, 78, 39, 0, 651, 653, 3, 44, 22, 0, 652, 654, 3, 80,
		40, 0, 653, 652, 1, 0, 0, 0, 653, 654, 1, 0, 0, 0, 654, 77, 1, 0, 0, 0,
		655, 656, 7, 1, 0, 0, 656, 79, 1, 0, 0, 0, 657, 659, 3, 2, 1, 0, 658, 657,
		1, 0, 0, 0, 659, 662, 1, 0, 0, 0, 660, 658, 1, 0, 0, 0, 660, 661, 1, 0,
		0, 0, 661, 663, 1, 0, 0, 0, 662, 660, 1, 0, 0, 0, 663, 667, 5, 8, 0, 0,
		664, 666, 3, 2, 1, 0, 665, 664, 1, 0, 0, 0, 666, 669, 1, 0, 0, 0, 667,
		665, 1, 0, 0, 0, 667, 668, 1, 0, 0, 0, 668, 679, 1, 0, 0, 0, 669, 667,
		1, 0, 0, 0, 670, 674, 3, 38, 19, 0, 671, 673, 3, 2, 1, 0, 672, 671, 1,
		0, 0, 0, 673, 676, 1, 0, 0, 0, 674, 672, 1, 0, 0, 0, 674, 675, 1, 0, 0,
		0, 675, 678, 1, 0, 0, 0, 676, 674, 1, 0, 0, 0, 677, 670, 1, 0, 0, 0, 678,
		681, 1, 0, 0, 0, 679, 677, 1, 0, 0, 0, 679, 680, 1, 0, 0, 0, 680, 682,
		1, 0, 0, 0, 681, 679, 1, 0, 0, 0, 682, 683, 5, 9, 0, 0, 683, 81, 1, 0,
		0, 0, 98, 85, 92, 97, 104, 109, 116, 121, 134, 140, 145, 149, 154, 161,
		169, 174, 176, 183, 189, 195, 202, 209, 214, 223, 231, 237, 243, 250, 254,
		258, 264, 269, 274, 285, 292, 300, 305, 310, 317, 322, 326, 330, 336, 344,
		350, 356, 359, 366, 371, 378, 385, 390, 398, 402, 407, 414, 421, 427, 430,
		435, 442, 446, 449, 452, 454, 461, 476, 484, 489, 495, 504, 510, 515, 522,
		530, 535, 540, 547, 552, 559, 566, 571, 578, 582, 587, 594, 601, 606, 617,
		622, 629, 637, 642, 647, 653, 660, 667, 674, 679,
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

// OracleParserInit initializes any static state used to implement OracleParser. By default the
// static state used to implement the parser is lazily initialized during the first call to
// NewOracleParser(). You can call this function if you wish to initialize the static state ahead
// of time.
func OracleParserInit() {
	staticData := &OracleParserParserStaticData
	staticData.once.Do(oracleparserParserInit)
}

// NewOracleParser produces a new parser instance for the optional input antlr.TokenStream.
func NewOracleParser(input antlr.TokenStream) *OracleParser {
	OracleParserInit()
	this := new(OracleParser)
	this.BaseParser = antlr.NewBaseParser(input)
	staticData := &OracleParserParserStaticData
	this.Interpreter = antlr.NewParserATNSimulator(this, staticData.atn, staticData.decisionToDFA, staticData.PredictionContextCache)
	this.RuleNames = staticData.RuleNames
	this.LiteralNames = staticData.LiteralNames
	this.SymbolicNames = staticData.SymbolicNames
	this.GrammarFileName = "OracleParser.g4"

	return this
}

// OracleParser tokens.
const (
	OracleParserEOF               = antlr.TokenEOF
	OracleParserSTRUCT            = 1
	OracleParserENUM              = 2
	OracleParserIMPORT            = 3
	OracleParserEXTENDS           = 4
	OracleParserMAP               = 5
	OracleParserACTION            = 6
	OracleParserUNION             = 7
	OracleParserLBRACE            = 8
	OracleParserRBRACE            = 9
	OracleParserLBRACKET          = 10
	OracleParserRBRACKET          = 11
	OracleParserLT                = 12
	OracleParserGT                = 13
	OracleParserCOMMA             = 14
	OracleParserQUESTION          = 15
	OracleParserDOT               = 16
	OracleParserEQUALS            = 17
	OracleParserAT                = 18
	OracleParserMINUS             = 19
	OracleParserTRIPLE_STRING_LIT = 20
	OracleParserSTRING_LIT        = 21
	OracleParserFLOAT_LIT         = 22
	OracleParserINT_LIT           = 23
	OracleParserBOOL_LIT          = 24
	OracleParserIDENT             = 25
	OracleParserLINE_COMMENT      = 26
	OracleParserBLOCK_COMMENT     = 27
	OracleParserNEWLINE           = 28
	OracleParserWS                = 29
)

// OracleParser rules.
const (
	OracleParserRULE_schema           = 0
	OracleParserRULE_nl               = 1
	OracleParserRULE_importStmt       = 2
	OracleParserRULE_fileDomain       = 3
	OracleParserRULE_definition       = 4
	OracleParserRULE_structDef        = 5
	OracleParserRULE_typeRefList      = 6
	OracleParserRULE_aliasBody        = 7
	OracleParserRULE_typeParams       = 8
	OracleParserRULE_typeParam        = 9
	OracleParserRULE_structBody       = 10
	OracleParserRULE_fieldOmit        = 11
	OracleParserRULE_actionDef        = 12
	OracleParserRULE_actionBody       = 13
	OracleParserRULE_fieldDef         = 14
	OracleParserRULE_fieldDefault     = 15
	OracleParserRULE_arrayDefault     = 16
	OracleParserRULE_inlineDomain     = 17
	OracleParserRULE_fieldBody        = 18
	OracleParserRULE_domain           = 19
	OracleParserRULE_domainContent    = 20
	OracleParserRULE_domainBlock      = 21
	OracleParserRULE_typeRef          = 22
	OracleParserRULE_arrayModifier    = 23
	OracleParserRULE_mapType          = 24
	OracleParserRULE_typeArgs         = 25
	OracleParserRULE_typeModifiers    = 26
	OracleParserRULE_qualifiedIdent   = 27
	OracleParserRULE_expression       = 28
	OracleParserRULE_expressionValue  = 29
	OracleParserRULE_enumDef          = 30
	OracleParserRULE_enumBody         = 31
	OracleParserRULE_enumValue        = 32
	OracleParserRULE_enumValueBody    = 33
	OracleParserRULE_typeDefDef       = 34
	OracleParserRULE_typeDefBody      = 35
	OracleParserRULE_unionDef         = 36
	OracleParserRULE_unionBody        = 37
	OracleParserRULE_unionVariant     = 38
	OracleParserRULE_variantName      = 39
	OracleParserRULE_unionVariantBody = 40
)

// ISchemaContext is an interface to support dynamic dispatch.
type ISchemaContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	EOF() antlr.TerminalNode
	AllNl() []INlContext
	Nl(i int) INlContext
	AllImportStmt() []IImportStmtContext
	ImportStmt(i int) IImportStmtContext
	AllFileDomain() []IFileDomainContext
	FileDomain(i int) IFileDomainContext
	AllDefinition() []IDefinitionContext
	Definition(i int) IDefinitionContext

	// IsSchemaContext differentiates from other interfaces.
	IsSchemaContext()
}

type SchemaContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptySchemaContext() *SchemaContext {
	var p = new(SchemaContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_schema
	return p
}

func InitEmptySchemaContext(p *SchemaContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_schema
}

func (*SchemaContext) IsSchemaContext() {}

func NewSchemaContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *SchemaContext {
	var p = new(SchemaContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = OracleParserRULE_schema

	return p
}

func (s *SchemaContext) GetParser() antlr.Parser { return s.parser }

func (s *SchemaContext) EOF() antlr.TerminalNode {
	return s.GetToken(OracleParserEOF, 0)
}

func (s *SchemaContext) AllNl() []INlContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(INlContext); ok {
			len++
		}
	}

	tst := make([]INlContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(INlContext); ok {
			tst[i] = t.(INlContext)
			i++
		}
	}

	return tst
}

func (s *SchemaContext) Nl(i int) INlContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(INlContext); ok {
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

	return t.(INlContext)
}

func (s *SchemaContext) AllImportStmt() []IImportStmtContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IImportStmtContext); ok {
			len++
		}
	}

	tst := make([]IImportStmtContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IImportStmtContext); ok {
			tst[i] = t.(IImportStmtContext)
			i++
		}
	}

	return tst
}

func (s *SchemaContext) ImportStmt(i int) IImportStmtContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IImportStmtContext); ok {
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

	return t.(IImportStmtContext)
}

func (s *SchemaContext) AllFileDomain() []IFileDomainContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IFileDomainContext); ok {
			len++
		}
	}

	tst := make([]IFileDomainContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IFileDomainContext); ok {
			tst[i] = t.(IFileDomainContext)
			i++
		}
	}

	return tst
}

func (s *SchemaContext) FileDomain(i int) IFileDomainContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IFileDomainContext); ok {
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

	return t.(IFileDomainContext)
}

func (s *SchemaContext) AllDefinition() []IDefinitionContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IDefinitionContext); ok {
			len++
		}
	}

	tst := make([]IDefinitionContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IDefinitionContext); ok {
			tst[i] = t.(IDefinitionContext)
			i++
		}
	}

	return tst
}

func (s *SchemaContext) Definition(i int) IDefinitionContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IDefinitionContext); ok {
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

	return t.(IDefinitionContext)
}

func (s *SchemaContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *SchemaContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *SchemaContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.EnterSchema(s)
	}
}

func (s *SchemaContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.ExitSchema(s)
	}
}

func (p *OracleParser) Schema() (localctx ISchemaContext) {
	localctx = NewSchemaContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 0, OracleParserRULE_schema)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(85)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(82)
			p.Nl()
		}

		p.SetState(87)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	p.SetState(97)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserIMPORT {
		{
			p.SetState(88)
			p.ImportStmt()
		}
		p.SetState(92)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(89)
				p.Nl()
			}

			p.SetState(94)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}

		p.SetState(99)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	p.SetState(109)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserAT {
		{
			p.SetState(100)
			p.FileDomain()
		}
		p.SetState(104)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(101)
				p.Nl()
			}

			p.SetState(106)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}

		p.SetState(111)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	p.SetState(121)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserIDENT {
		{
			p.SetState(112)
			p.Definition()
		}
		p.SetState(116)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(113)
				p.Nl()
			}

			p.SetState(118)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}

		p.SetState(123)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(124)
		p.Match(OracleParserEOF)
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

// INlContext is an interface to support dynamic dispatch.
type INlContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	NEWLINE() antlr.TerminalNode

	// IsNlContext differentiates from other interfaces.
	IsNlContext()
}

type NlContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyNlContext() *NlContext {
	var p = new(NlContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_nl
	return p
}

func InitEmptyNlContext(p *NlContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_nl
}

func (*NlContext) IsNlContext() {}

func NewNlContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *NlContext {
	var p = new(NlContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = OracleParserRULE_nl

	return p
}

func (s *NlContext) GetParser() antlr.Parser { return s.parser }

func (s *NlContext) NEWLINE() antlr.TerminalNode {
	return s.GetToken(OracleParserNEWLINE, 0)
}

func (s *NlContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *NlContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *NlContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.EnterNl(s)
	}
}

func (s *NlContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.ExitNl(s)
	}
}

func (p *OracleParser) Nl() (localctx INlContext) {
	localctx = NewNlContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 2, OracleParserRULE_nl)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(126)
		p.Match(OracleParserNEWLINE)
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

// IImportStmtContext is an interface to support dynamic dispatch.
type IImportStmtContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	IMPORT() antlr.TerminalNode
	STRING_LIT() antlr.TerminalNode

	// IsImportStmtContext differentiates from other interfaces.
	IsImportStmtContext()
}

type ImportStmtContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyImportStmtContext() *ImportStmtContext {
	var p = new(ImportStmtContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_importStmt
	return p
}

func InitEmptyImportStmtContext(p *ImportStmtContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_importStmt
}

func (*ImportStmtContext) IsImportStmtContext() {}

func NewImportStmtContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ImportStmtContext {
	var p = new(ImportStmtContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = OracleParserRULE_importStmt

	return p
}

func (s *ImportStmtContext) GetParser() antlr.Parser { return s.parser }

func (s *ImportStmtContext) IMPORT() antlr.TerminalNode {
	return s.GetToken(OracleParserIMPORT, 0)
}

func (s *ImportStmtContext) STRING_LIT() antlr.TerminalNode {
	return s.GetToken(OracleParserSTRING_LIT, 0)
}

func (s *ImportStmtContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ImportStmtContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ImportStmtContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.EnterImportStmt(s)
	}
}

func (s *ImportStmtContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.ExitImportStmt(s)
	}
}

func (p *OracleParser) ImportStmt() (localctx IImportStmtContext) {
	localctx = NewImportStmtContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 4, OracleParserRULE_importStmt)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(128)
		p.Match(OracleParserIMPORT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(129)
		p.Match(OracleParserSTRING_LIT)
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

// IFileDomainContext is an interface to support dynamic dispatch.
type IFileDomainContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AT() antlr.TerminalNode
	IDENT() antlr.TerminalNode
	DomainContent() IDomainContentContext

	// IsFileDomainContext differentiates from other interfaces.
	IsFileDomainContext()
}

type FileDomainContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyFileDomainContext() *FileDomainContext {
	var p = new(FileDomainContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_fileDomain
	return p
}

func InitEmptyFileDomainContext(p *FileDomainContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_fileDomain
}

func (*FileDomainContext) IsFileDomainContext() {}

func NewFileDomainContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *FileDomainContext {
	var p = new(FileDomainContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = OracleParserRULE_fileDomain

	return p
}

func (s *FileDomainContext) GetParser() antlr.Parser { return s.parser }

func (s *FileDomainContext) AT() antlr.TerminalNode {
	return s.GetToken(OracleParserAT, 0)
}

func (s *FileDomainContext) IDENT() antlr.TerminalNode {
	return s.GetToken(OracleParserIDENT, 0)
}

func (s *FileDomainContext) DomainContent() IDomainContentContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IDomainContentContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IDomainContentContext)
}

func (s *FileDomainContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *FileDomainContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *FileDomainContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.EnterFileDomain(s)
	}
}

func (s *FileDomainContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.ExitFileDomain(s)
	}
}

func (p *OracleParser) FileDomain() (localctx IFileDomainContext) {
	localctx = NewFileDomainContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 6, OracleParserRULE_fileDomain)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(131)
		p.Match(OracleParserAT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(132)
		p.Match(OracleParserIDENT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(134)
	p.GetErrorHandler().Sync(p)

	if p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 7, p.GetParserRuleContext()) == 1 {
		{
			p.SetState(133)
			p.DomainContent()
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

// IDefinitionContext is an interface to support dynamic dispatch.
type IDefinitionContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	StructDef() IStructDefContext
	EnumDef() IEnumDefContext
	TypeDefDef() ITypeDefDefContext
	UnionDef() IUnionDefContext

	// IsDefinitionContext differentiates from other interfaces.
	IsDefinitionContext()
}

type DefinitionContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyDefinitionContext() *DefinitionContext {
	var p = new(DefinitionContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_definition
	return p
}

func InitEmptyDefinitionContext(p *DefinitionContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_definition
}

func (*DefinitionContext) IsDefinitionContext() {}

func NewDefinitionContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *DefinitionContext {
	var p = new(DefinitionContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = OracleParserRULE_definition

	return p
}

func (s *DefinitionContext) GetParser() antlr.Parser { return s.parser }

func (s *DefinitionContext) StructDef() IStructDefContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IStructDefContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IStructDefContext)
}

func (s *DefinitionContext) EnumDef() IEnumDefContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IEnumDefContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IEnumDefContext)
}

func (s *DefinitionContext) TypeDefDef() ITypeDefDefContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ITypeDefDefContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ITypeDefDefContext)
}

func (s *DefinitionContext) UnionDef() IUnionDefContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IUnionDefContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IUnionDefContext)
}

func (s *DefinitionContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *DefinitionContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *DefinitionContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.EnterDefinition(s)
	}
}

func (s *DefinitionContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.ExitDefinition(s)
	}
}

func (p *OracleParser) Definition() (localctx IDefinitionContext) {
	localctx = NewDefinitionContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 8, OracleParserRULE_definition)
	p.SetState(140)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 8, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(136)
			p.StructDef()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(137)
			p.EnumDef()
		}

	case 3:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(138)
			p.TypeDefDef()
		}

	case 4:
		p.EnterOuterAlt(localctx, 4)
		{
			p.SetState(139)
			p.UnionDef()
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

// IStructDefContext is an interface to support dynamic dispatch.
type IStructDefContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser
	// IsStructDefContext differentiates from other interfaces.
	IsStructDefContext()
}

type StructDefContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyStructDefContext() *StructDefContext {
	var p = new(StructDefContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_structDef
	return p
}

func InitEmptyStructDefContext(p *StructDefContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_structDef
}

func (*StructDefContext) IsStructDefContext() {}

func NewStructDefContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *StructDefContext {
	var p = new(StructDefContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = OracleParserRULE_structDef

	return p
}

func (s *StructDefContext) GetParser() antlr.Parser { return s.parser }

func (s *StructDefContext) CopyAll(ctx *StructDefContext) {
	s.CopyFrom(&ctx.BaseParserRuleContext)
}

func (s *StructDefContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *StructDefContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

type StructFullContext struct {
	StructDefContext
}

func NewStructFullContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *StructFullContext {
	var p = new(StructFullContext)

	InitEmptyStructDefContext(&p.StructDefContext)
	p.parser = parser
	p.CopyAll(ctx.(*StructDefContext))

	return p
}

func (s *StructFullContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *StructFullContext) IDENT() antlr.TerminalNode {
	return s.GetToken(OracleParserIDENT, 0)
}

func (s *StructFullContext) STRUCT() antlr.TerminalNode {
	return s.GetToken(OracleParserSTRUCT, 0)
}

func (s *StructFullContext) LBRACE() antlr.TerminalNode {
	return s.GetToken(OracleParserLBRACE, 0)
}

func (s *StructFullContext) StructBody() IStructBodyContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IStructBodyContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IStructBodyContext)
}

func (s *StructFullContext) RBRACE() antlr.TerminalNode {
	return s.GetToken(OracleParserRBRACE, 0)
}

func (s *StructFullContext) TypeParams() ITypeParamsContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ITypeParamsContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ITypeParamsContext)
}

func (s *StructFullContext) EXTENDS() antlr.TerminalNode {
	return s.GetToken(OracleParserEXTENDS, 0)
}

func (s *StructFullContext) TypeRefList() ITypeRefListContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ITypeRefListContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ITypeRefListContext)
}

func (s *StructFullContext) AllNl() []INlContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(INlContext); ok {
			len++
		}
	}

	tst := make([]INlContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(INlContext); ok {
			tst[i] = t.(INlContext)
			i++
		}
	}

	return tst
}

func (s *StructFullContext) Nl(i int) INlContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(INlContext); ok {
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

	return t.(INlContext)
}

func (s *StructFullContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.EnterStructFull(s)
	}
}

func (s *StructFullContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.ExitStructFull(s)
	}
}

type StructAliasContext struct {
	StructDefContext
}

func NewStructAliasContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *StructAliasContext {
	var p = new(StructAliasContext)

	InitEmptyStructDefContext(&p.StructDefContext)
	p.parser = parser
	p.CopyAll(ctx.(*StructDefContext))

	return p
}

func (s *StructAliasContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *StructAliasContext) IDENT() antlr.TerminalNode {
	return s.GetToken(OracleParserIDENT, 0)
}

func (s *StructAliasContext) EQUALS() antlr.TerminalNode {
	return s.GetToken(OracleParserEQUALS, 0)
}

func (s *StructAliasContext) TypeRef() ITypeRefContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ITypeRefContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ITypeRefContext)
}

func (s *StructAliasContext) TypeParams() ITypeParamsContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ITypeParamsContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ITypeParamsContext)
}

func (s *StructAliasContext) AliasBody() IAliasBodyContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IAliasBodyContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IAliasBodyContext)
}

func (s *StructAliasContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.EnterStructAlias(s)
	}
}

func (s *StructAliasContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.ExitStructAlias(s)
	}
}

func (p *OracleParser) StructDef() (localctx IStructDefContext) {
	localctx = NewStructDefContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 10, OracleParserRULE_structDef)
	var _la int

	p.SetState(176)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 15, p.GetParserRuleContext()) {
	case 1:
		localctx = NewStructFullContext(p, localctx)
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(142)
			p.Match(OracleParserIDENT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(143)
			p.Match(OracleParserSTRUCT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		p.SetState(145)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		if _la == OracleParserLT {
			{
				p.SetState(144)
				p.TypeParams()
			}

		}
		p.SetState(149)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		if _la == OracleParserEXTENDS {
			{
				p.SetState(147)
				p.Match(OracleParserEXTENDS)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}
			{
				p.SetState(148)
				p.TypeRefList()
			}

		}
		p.SetState(154)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(151)
				p.Nl()
			}

			p.SetState(156)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}
		{
			p.SetState(157)
			p.Match(OracleParserLBRACE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		p.SetState(161)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(158)
				p.Nl()
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
			p.StructBody()
		}
		{
			p.SetState(165)
			p.Match(OracleParserRBRACE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 2:
		localctx = NewStructAliasContext(p, localctx)
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(167)
			p.Match(OracleParserIDENT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		p.SetState(169)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		if _la == OracleParserLT {
			{
				p.SetState(168)
				p.TypeParams()
			}

		}
		{
			p.SetState(171)
			p.Match(OracleParserEQUALS)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(172)
			p.TypeRef()
		}
		p.SetState(174)
		p.GetErrorHandler().Sync(p)

		if p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 14, p.GetParserRuleContext()) == 1 {
			{
				p.SetState(173)
				p.AliasBody()
			}

		} else if p.HasError() { // JIM
			goto errorExit
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

// ITypeRefListContext is an interface to support dynamic dispatch.
type ITypeRefListContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AllTypeRef() []ITypeRefContext
	TypeRef(i int) ITypeRefContext
	AllCOMMA() []antlr.TerminalNode
	COMMA(i int) antlr.TerminalNode
	AllNl() []INlContext
	Nl(i int) INlContext

	// IsTypeRefListContext differentiates from other interfaces.
	IsTypeRefListContext()
}

type TypeRefListContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyTypeRefListContext() *TypeRefListContext {
	var p = new(TypeRefListContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_typeRefList
	return p
}

func InitEmptyTypeRefListContext(p *TypeRefListContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_typeRefList
}

func (*TypeRefListContext) IsTypeRefListContext() {}

func NewTypeRefListContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *TypeRefListContext {
	var p = new(TypeRefListContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = OracleParserRULE_typeRefList

	return p
}

func (s *TypeRefListContext) GetParser() antlr.Parser { return s.parser }

func (s *TypeRefListContext) AllTypeRef() []ITypeRefContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(ITypeRefContext); ok {
			len++
		}
	}

	tst := make([]ITypeRefContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(ITypeRefContext); ok {
			tst[i] = t.(ITypeRefContext)
			i++
		}
	}

	return tst
}

func (s *TypeRefListContext) TypeRef(i int) ITypeRefContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ITypeRefContext); ok {
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

	return t.(ITypeRefContext)
}

func (s *TypeRefListContext) AllCOMMA() []antlr.TerminalNode {
	return s.GetTokens(OracleParserCOMMA)
}

func (s *TypeRefListContext) COMMA(i int) antlr.TerminalNode {
	return s.GetToken(OracleParserCOMMA, i)
}

func (s *TypeRefListContext) AllNl() []INlContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(INlContext); ok {
			len++
		}
	}

	tst := make([]INlContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(INlContext); ok {
			tst[i] = t.(INlContext)
			i++
		}
	}

	return tst
}

func (s *TypeRefListContext) Nl(i int) INlContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(INlContext); ok {
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

	return t.(INlContext)
}

func (s *TypeRefListContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *TypeRefListContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *TypeRefListContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.EnterTypeRefList(s)
	}
}

func (s *TypeRefListContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.ExitTypeRefList(s)
	}
}

func (p *OracleParser) TypeRefList() (localctx ITypeRefListContext) {
	localctx = NewTypeRefListContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 12, OracleParserRULE_typeRefList)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(178)
		p.TypeRef()
	}
	p.SetState(189)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserCOMMA {
		{
			p.SetState(179)
			p.Match(OracleParserCOMMA)
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

		for _la == OracleParserNEWLINE {
			{
				p.SetState(180)
				p.Nl()
			}

			p.SetState(185)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}
		{
			p.SetState(186)
			p.TypeRef()
		}

		p.SetState(191)
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

// IAliasBodyContext is an interface to support dynamic dispatch.
type IAliasBodyContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	LBRACE() antlr.TerminalNode
	RBRACE() antlr.TerminalNode
	AllNl() []INlContext
	Nl(i int) INlContext
	AllDomain() []IDomainContext
	Domain(i int) IDomainContext

	// IsAliasBodyContext differentiates from other interfaces.
	IsAliasBodyContext()
}

type AliasBodyContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyAliasBodyContext() *AliasBodyContext {
	var p = new(AliasBodyContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_aliasBody
	return p
}

func InitEmptyAliasBodyContext(p *AliasBodyContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_aliasBody
}

func (*AliasBodyContext) IsAliasBodyContext() {}

func NewAliasBodyContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *AliasBodyContext {
	var p = new(AliasBodyContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = OracleParserRULE_aliasBody

	return p
}

func (s *AliasBodyContext) GetParser() antlr.Parser { return s.parser }

func (s *AliasBodyContext) LBRACE() antlr.TerminalNode {
	return s.GetToken(OracleParserLBRACE, 0)
}

func (s *AliasBodyContext) RBRACE() antlr.TerminalNode {
	return s.GetToken(OracleParserRBRACE, 0)
}

func (s *AliasBodyContext) AllNl() []INlContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(INlContext); ok {
			len++
		}
	}

	tst := make([]INlContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(INlContext); ok {
			tst[i] = t.(INlContext)
			i++
		}
	}

	return tst
}

func (s *AliasBodyContext) Nl(i int) INlContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(INlContext); ok {
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

	return t.(INlContext)
}

func (s *AliasBodyContext) AllDomain() []IDomainContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IDomainContext); ok {
			len++
		}
	}

	tst := make([]IDomainContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IDomainContext); ok {
			tst[i] = t.(IDomainContext)
			i++
		}
	}

	return tst
}

func (s *AliasBodyContext) Domain(i int) IDomainContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IDomainContext); ok {
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

	return t.(IDomainContext)
}

func (s *AliasBodyContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *AliasBodyContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *AliasBodyContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.EnterAliasBody(s)
	}
}

func (s *AliasBodyContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.ExitAliasBody(s)
	}
}

func (p *OracleParser) AliasBody() (localctx IAliasBodyContext) {
	localctx = NewAliasBodyContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 14, OracleParserRULE_aliasBody)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(195)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(192)
			p.Nl()
		}

		p.SetState(197)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(198)
		p.Match(OracleParserLBRACE)
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

	for _la == OracleParserNEWLINE {
		{
			p.SetState(199)
			p.Nl()
		}

		p.SetState(204)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	p.SetState(214)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserAT {
		{
			p.SetState(205)
			p.Domain()
		}
		p.SetState(209)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(206)
				p.Nl()
			}

			p.SetState(211)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}

		p.SetState(216)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(217)
		p.Match(OracleParserRBRACE)
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

// ITypeParamsContext is an interface to support dynamic dispatch.
type ITypeParamsContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	LT() antlr.TerminalNode
	AllTypeParam() []ITypeParamContext
	TypeParam(i int) ITypeParamContext
	GT() antlr.TerminalNode
	AllNl() []INlContext
	Nl(i int) INlContext
	AllCOMMA() []antlr.TerminalNode
	COMMA(i int) antlr.TerminalNode

	// IsTypeParamsContext differentiates from other interfaces.
	IsTypeParamsContext()
}

type TypeParamsContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyTypeParamsContext() *TypeParamsContext {
	var p = new(TypeParamsContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_typeParams
	return p
}

func InitEmptyTypeParamsContext(p *TypeParamsContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_typeParams
}

func (*TypeParamsContext) IsTypeParamsContext() {}

func NewTypeParamsContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *TypeParamsContext {
	var p = new(TypeParamsContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = OracleParserRULE_typeParams

	return p
}

func (s *TypeParamsContext) GetParser() antlr.Parser { return s.parser }

func (s *TypeParamsContext) LT() antlr.TerminalNode {
	return s.GetToken(OracleParserLT, 0)
}

func (s *TypeParamsContext) AllTypeParam() []ITypeParamContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(ITypeParamContext); ok {
			len++
		}
	}

	tst := make([]ITypeParamContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(ITypeParamContext); ok {
			tst[i] = t.(ITypeParamContext)
			i++
		}
	}

	return tst
}

func (s *TypeParamsContext) TypeParam(i int) ITypeParamContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ITypeParamContext); ok {
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

	return t.(ITypeParamContext)
}

func (s *TypeParamsContext) GT() antlr.TerminalNode {
	return s.GetToken(OracleParserGT, 0)
}

func (s *TypeParamsContext) AllNl() []INlContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(INlContext); ok {
			len++
		}
	}

	tst := make([]INlContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(INlContext); ok {
			tst[i] = t.(INlContext)
			i++
		}
	}

	return tst
}

func (s *TypeParamsContext) Nl(i int) INlContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(INlContext); ok {
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

	return t.(INlContext)
}

func (s *TypeParamsContext) AllCOMMA() []antlr.TerminalNode {
	return s.GetTokens(OracleParserCOMMA)
}

func (s *TypeParamsContext) COMMA(i int) antlr.TerminalNode {
	return s.GetToken(OracleParserCOMMA, i)
}

func (s *TypeParamsContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *TypeParamsContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *TypeParamsContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.EnterTypeParams(s)
	}
}

func (s *TypeParamsContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.ExitTypeParams(s)
	}
}

func (p *OracleParser) TypeParams() (localctx ITypeParamsContext) {
	localctx = NewTypeParamsContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 16, OracleParserRULE_typeParams)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(219)
		p.Match(OracleParserLT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(223)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(220)
			p.Nl()
		}

		p.SetState(225)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(226)
		p.TypeParam()
	}
	p.SetState(237)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserCOMMA {
		{
			p.SetState(227)
			p.Match(OracleParserCOMMA)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		p.SetState(231)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(228)
				p.Nl()
			}

			p.SetState(233)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}
		{
			p.SetState(234)
			p.TypeParam()
		}

		p.SetState(239)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	p.SetState(243)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(240)
			p.Nl()
		}

		p.SetState(245)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(246)
		p.Match(OracleParserGT)
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

// ITypeParamContext is an interface to support dynamic dispatch.
type ITypeParamContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	IDENT() antlr.TerminalNode
	QUESTION() antlr.TerminalNode
	EXTENDS() antlr.TerminalNode
	AllTypeRef() []ITypeRefContext
	TypeRef(i int) ITypeRefContext
	EQUALS() antlr.TerminalNode

	// IsTypeParamContext differentiates from other interfaces.
	IsTypeParamContext()
}

type TypeParamContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyTypeParamContext() *TypeParamContext {
	var p = new(TypeParamContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_typeParam
	return p
}

func InitEmptyTypeParamContext(p *TypeParamContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_typeParam
}

func (*TypeParamContext) IsTypeParamContext() {}

func NewTypeParamContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *TypeParamContext {
	var p = new(TypeParamContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = OracleParserRULE_typeParam

	return p
}

func (s *TypeParamContext) GetParser() antlr.Parser { return s.parser }

func (s *TypeParamContext) IDENT() antlr.TerminalNode {
	return s.GetToken(OracleParserIDENT, 0)
}

func (s *TypeParamContext) QUESTION() antlr.TerminalNode {
	return s.GetToken(OracleParserQUESTION, 0)
}

func (s *TypeParamContext) EXTENDS() antlr.TerminalNode {
	return s.GetToken(OracleParserEXTENDS, 0)
}

func (s *TypeParamContext) AllTypeRef() []ITypeRefContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(ITypeRefContext); ok {
			len++
		}
	}

	tst := make([]ITypeRefContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(ITypeRefContext); ok {
			tst[i] = t.(ITypeRefContext)
			i++
		}
	}

	return tst
}

func (s *TypeParamContext) TypeRef(i int) ITypeRefContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ITypeRefContext); ok {
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

	return t.(ITypeRefContext)
}

func (s *TypeParamContext) EQUALS() antlr.TerminalNode {
	return s.GetToken(OracleParserEQUALS, 0)
}

func (s *TypeParamContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *TypeParamContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *TypeParamContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.EnterTypeParam(s)
	}
}

func (s *TypeParamContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.ExitTypeParam(s)
	}
}

func (p *OracleParser) TypeParam() (localctx ITypeParamContext) {
	localctx = NewTypeParamContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 18, OracleParserRULE_typeParam)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(248)
		p.Match(OracleParserIDENT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(250)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == OracleParserQUESTION {
		{
			p.SetState(249)
			p.Match(OracleParserQUESTION)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	}
	p.SetState(254)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == OracleParserEXTENDS {
		{
			p.SetState(252)
			p.Match(OracleParserEXTENDS)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(253)
			p.TypeRef()
		}

	}
	p.SetState(258)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == OracleParserEQUALS {
		{
			p.SetState(256)
			p.Match(OracleParserEQUALS)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(257)
			p.TypeRef()
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

// IStructBodyContext is an interface to support dynamic dispatch.
type IStructBodyContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AllFieldDef() []IFieldDefContext
	FieldDef(i int) IFieldDefContext
	AllFieldOmit() []IFieldOmitContext
	FieldOmit(i int) IFieldOmitContext
	AllActionDef() []IActionDefContext
	ActionDef(i int) IActionDefContext
	AllDomain() []IDomainContext
	Domain(i int) IDomainContext
	AllNl() []INlContext
	Nl(i int) INlContext

	// IsStructBodyContext differentiates from other interfaces.
	IsStructBodyContext()
}

type StructBodyContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyStructBodyContext() *StructBodyContext {
	var p = new(StructBodyContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_structBody
	return p
}

func InitEmptyStructBodyContext(p *StructBodyContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_structBody
}

func (*StructBodyContext) IsStructBodyContext() {}

func NewStructBodyContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *StructBodyContext {
	var p = new(StructBodyContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = OracleParserRULE_structBody

	return p
}

func (s *StructBodyContext) GetParser() antlr.Parser { return s.parser }

func (s *StructBodyContext) AllFieldDef() []IFieldDefContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IFieldDefContext); ok {
			len++
		}
	}

	tst := make([]IFieldDefContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IFieldDefContext); ok {
			tst[i] = t.(IFieldDefContext)
			i++
		}
	}

	return tst
}

func (s *StructBodyContext) FieldDef(i int) IFieldDefContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IFieldDefContext); ok {
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

	return t.(IFieldDefContext)
}

func (s *StructBodyContext) AllFieldOmit() []IFieldOmitContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IFieldOmitContext); ok {
			len++
		}
	}

	tst := make([]IFieldOmitContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IFieldOmitContext); ok {
			tst[i] = t.(IFieldOmitContext)
			i++
		}
	}

	return tst
}

func (s *StructBodyContext) FieldOmit(i int) IFieldOmitContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IFieldOmitContext); ok {
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

	return t.(IFieldOmitContext)
}

func (s *StructBodyContext) AllActionDef() []IActionDefContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IActionDefContext); ok {
			len++
		}
	}

	tst := make([]IActionDefContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IActionDefContext); ok {
			tst[i] = t.(IActionDefContext)
			i++
		}
	}

	return tst
}

func (s *StructBodyContext) ActionDef(i int) IActionDefContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IActionDefContext); ok {
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

	return t.(IActionDefContext)
}

func (s *StructBodyContext) AllDomain() []IDomainContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IDomainContext); ok {
			len++
		}
	}

	tst := make([]IDomainContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IDomainContext); ok {
			tst[i] = t.(IDomainContext)
			i++
		}
	}

	return tst
}

func (s *StructBodyContext) Domain(i int) IDomainContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IDomainContext); ok {
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

	return t.(IDomainContext)
}

func (s *StructBodyContext) AllNl() []INlContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(INlContext); ok {
			len++
		}
	}

	tst := make([]INlContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(INlContext); ok {
			tst[i] = t.(INlContext)
			i++
		}
	}

	return tst
}

func (s *StructBodyContext) Nl(i int) INlContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(INlContext); ok {
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

	return t.(INlContext)
}

func (s *StructBodyContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *StructBodyContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *StructBodyContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.EnterStructBody(s)
	}
}

func (s *StructBodyContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.ExitStructBody(s)
	}
}

func (p *OracleParser) StructBody() (localctx IStructBodyContext) {
	localctx = NewStructBodyContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 20, OracleParserRULE_structBody)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(274)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&34340928) != 0 {
		p.SetState(264)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}

		switch p.GetTokenStream().LA(1) {
		case OracleParserIDENT:
			{
				p.SetState(260)
				p.FieldDef()
			}

		case OracleParserMINUS:
			{
				p.SetState(261)
				p.FieldOmit()
			}

		case OracleParserACTION:
			{
				p.SetState(262)
				p.ActionDef()
			}

		case OracleParserAT:
			{
				p.SetState(263)
				p.Domain()
			}

		default:
			p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
			goto errorExit
		}
		p.SetState(269)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(266)
				p.Nl()
			}

			p.SetState(271)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}

		p.SetState(276)
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

// IFieldOmitContext is an interface to support dynamic dispatch.
type IFieldOmitContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	MINUS() antlr.TerminalNode
	IDENT() antlr.TerminalNode

	// IsFieldOmitContext differentiates from other interfaces.
	IsFieldOmitContext()
}

type FieldOmitContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyFieldOmitContext() *FieldOmitContext {
	var p = new(FieldOmitContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_fieldOmit
	return p
}

func InitEmptyFieldOmitContext(p *FieldOmitContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_fieldOmit
}

func (*FieldOmitContext) IsFieldOmitContext() {}

func NewFieldOmitContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *FieldOmitContext {
	var p = new(FieldOmitContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = OracleParserRULE_fieldOmit

	return p
}

func (s *FieldOmitContext) GetParser() antlr.Parser { return s.parser }

func (s *FieldOmitContext) MINUS() antlr.TerminalNode {
	return s.GetToken(OracleParserMINUS, 0)
}

func (s *FieldOmitContext) IDENT() antlr.TerminalNode {
	return s.GetToken(OracleParserIDENT, 0)
}

func (s *FieldOmitContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *FieldOmitContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *FieldOmitContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.EnterFieldOmit(s)
	}
}

func (s *FieldOmitContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.ExitFieldOmit(s)
	}
}

func (p *OracleParser) FieldOmit() (localctx IFieldOmitContext) {
	localctx = NewFieldOmitContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 22, OracleParserRULE_fieldOmit)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(277)
		p.Match(OracleParserMINUS)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(278)
		p.Match(OracleParserIDENT)
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

// IActionDefContext is an interface to support dynamic dispatch.
type IActionDefContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	ACTION() antlr.TerminalNode
	IDENT() antlr.TerminalNode
	LBRACE() antlr.TerminalNode
	ActionBody() IActionBodyContext
	RBRACE() antlr.TerminalNode
	AllNl() []INlContext
	Nl(i int) INlContext

	// IsActionDefContext differentiates from other interfaces.
	IsActionDefContext()
}

type ActionDefContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyActionDefContext() *ActionDefContext {
	var p = new(ActionDefContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_actionDef
	return p
}

func InitEmptyActionDefContext(p *ActionDefContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_actionDef
}

func (*ActionDefContext) IsActionDefContext() {}

func NewActionDefContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ActionDefContext {
	var p = new(ActionDefContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = OracleParserRULE_actionDef

	return p
}

func (s *ActionDefContext) GetParser() antlr.Parser { return s.parser }

func (s *ActionDefContext) ACTION() antlr.TerminalNode {
	return s.GetToken(OracleParserACTION, 0)
}

func (s *ActionDefContext) IDENT() antlr.TerminalNode {
	return s.GetToken(OracleParserIDENT, 0)
}

func (s *ActionDefContext) LBRACE() antlr.TerminalNode {
	return s.GetToken(OracleParserLBRACE, 0)
}

func (s *ActionDefContext) ActionBody() IActionBodyContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IActionBodyContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IActionBodyContext)
}

func (s *ActionDefContext) RBRACE() antlr.TerminalNode {
	return s.GetToken(OracleParserRBRACE, 0)
}

func (s *ActionDefContext) AllNl() []INlContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(INlContext); ok {
			len++
		}
	}

	tst := make([]INlContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(INlContext); ok {
			tst[i] = t.(INlContext)
			i++
		}
	}

	return tst
}

func (s *ActionDefContext) Nl(i int) INlContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(INlContext); ok {
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

	return t.(INlContext)
}

func (s *ActionDefContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ActionDefContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ActionDefContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.EnterActionDef(s)
	}
}

func (s *ActionDefContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.ExitActionDef(s)
	}
}

func (p *OracleParser) ActionDef() (localctx IActionDefContext) {
	localctx = NewActionDefContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 24, OracleParserRULE_actionDef)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(280)
		p.Match(OracleParserACTION)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(281)
		p.Match(OracleParserIDENT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(285)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(282)
			p.Nl()
		}

		p.SetState(287)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(288)
		p.Match(OracleParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(292)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(289)
			p.Nl()
		}

		p.SetState(294)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(295)
		p.ActionBody()
	}
	{
		p.SetState(296)
		p.Match(OracleParserRBRACE)
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

// IActionBodyContext is an interface to support dynamic dispatch.
type IActionBodyContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AllFieldDef() []IFieldDefContext
	FieldDef(i int) IFieldDefContext
	AllDomain() []IDomainContext
	Domain(i int) IDomainContext
	AllNl() []INlContext
	Nl(i int) INlContext

	// IsActionBodyContext differentiates from other interfaces.
	IsActionBodyContext()
}

type ActionBodyContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyActionBodyContext() *ActionBodyContext {
	var p = new(ActionBodyContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_actionBody
	return p
}

func InitEmptyActionBodyContext(p *ActionBodyContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_actionBody
}

func (*ActionBodyContext) IsActionBodyContext() {}

func NewActionBodyContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ActionBodyContext {
	var p = new(ActionBodyContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = OracleParserRULE_actionBody

	return p
}

func (s *ActionBodyContext) GetParser() antlr.Parser { return s.parser }

func (s *ActionBodyContext) AllFieldDef() []IFieldDefContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IFieldDefContext); ok {
			len++
		}
	}

	tst := make([]IFieldDefContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IFieldDefContext); ok {
			tst[i] = t.(IFieldDefContext)
			i++
		}
	}

	return tst
}

func (s *ActionBodyContext) FieldDef(i int) IFieldDefContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IFieldDefContext); ok {
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

	return t.(IFieldDefContext)
}

func (s *ActionBodyContext) AllDomain() []IDomainContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IDomainContext); ok {
			len++
		}
	}

	tst := make([]IDomainContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IDomainContext); ok {
			tst[i] = t.(IDomainContext)
			i++
		}
	}

	return tst
}

func (s *ActionBodyContext) Domain(i int) IDomainContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IDomainContext); ok {
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

	return t.(IDomainContext)
}

func (s *ActionBodyContext) AllNl() []INlContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(INlContext); ok {
			len++
		}
	}

	tst := make([]INlContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(INlContext); ok {
			tst[i] = t.(INlContext)
			i++
		}
	}

	return tst
}

func (s *ActionBodyContext) Nl(i int) INlContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(INlContext); ok {
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

	return t.(INlContext)
}

func (s *ActionBodyContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ActionBodyContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ActionBodyContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.EnterActionBody(s)
	}
}

func (s *ActionBodyContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.ExitActionBody(s)
	}
}

func (p *OracleParser) ActionBody() (localctx IActionBodyContext) {
	localctx = NewActionBodyContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 26, OracleParserRULE_actionBody)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(310)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserAT || _la == OracleParserIDENT {
		p.SetState(300)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}

		switch p.GetTokenStream().LA(1) {
		case OracleParserIDENT:
			{
				p.SetState(298)
				p.FieldDef()
			}

		case OracleParserAT:
			{
				p.SetState(299)
				p.Domain()
			}

		default:
			p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
			goto errorExit
		}
		p.SetState(305)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(302)
				p.Nl()
			}

			p.SetState(307)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}

		p.SetState(312)
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

// IFieldDefContext is an interface to support dynamic dispatch.
type IFieldDefContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	IDENT() antlr.TerminalNode
	TypeRef() ITypeRefContext
	EQUALS() antlr.TerminalNode
	FieldDefault() IFieldDefaultContext
	AllInlineDomain() []IInlineDomainContext
	InlineDomain(i int) IInlineDomainContext
	FieldBody() IFieldBodyContext

	// IsFieldDefContext differentiates from other interfaces.
	IsFieldDefContext()
}

type FieldDefContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyFieldDefContext() *FieldDefContext {
	var p = new(FieldDefContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_fieldDef
	return p
}

func InitEmptyFieldDefContext(p *FieldDefContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_fieldDef
}

func (*FieldDefContext) IsFieldDefContext() {}

func NewFieldDefContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *FieldDefContext {
	var p = new(FieldDefContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = OracleParserRULE_fieldDef

	return p
}

func (s *FieldDefContext) GetParser() antlr.Parser { return s.parser }

func (s *FieldDefContext) IDENT() antlr.TerminalNode {
	return s.GetToken(OracleParserIDENT, 0)
}

func (s *FieldDefContext) TypeRef() ITypeRefContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ITypeRefContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ITypeRefContext)
}

func (s *FieldDefContext) EQUALS() antlr.TerminalNode {
	return s.GetToken(OracleParserEQUALS, 0)
}

func (s *FieldDefContext) FieldDefault() IFieldDefaultContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IFieldDefaultContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IFieldDefaultContext)
}

func (s *FieldDefContext) AllInlineDomain() []IInlineDomainContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IInlineDomainContext); ok {
			len++
		}
	}

	tst := make([]IInlineDomainContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IInlineDomainContext); ok {
			tst[i] = t.(IInlineDomainContext)
			i++
		}
	}

	return tst
}

func (s *FieldDefContext) InlineDomain(i int) IInlineDomainContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IInlineDomainContext); ok {
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

	return t.(IInlineDomainContext)
}

func (s *FieldDefContext) FieldBody() IFieldBodyContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IFieldBodyContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IFieldBodyContext)
}

func (s *FieldDefContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *FieldDefContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *FieldDefContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.EnterFieldDef(s)
	}
}

func (s *FieldDefContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.ExitFieldDef(s)
	}
}

func (p *OracleParser) FieldDef() (localctx IFieldDefContext) {
	localctx = NewFieldDefContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 28, OracleParserRULE_fieldDef)
	var _la int

	var _alt int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(313)
		p.Match(OracleParserIDENT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(314)
		p.TypeRef()
	}
	p.SetState(317)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == OracleParserEQUALS {
		{
			p.SetState(315)
			p.Match(OracleParserEQUALS)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(316)
			p.FieldDefault()
		}

	}
	p.SetState(322)
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
				p.SetState(319)
				p.InlineDomain()
			}

		}
		p.SetState(324)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 38, p.GetParserRuleContext())
		if p.HasError() {
			goto errorExit
		}
	}
	p.SetState(326)
	p.GetErrorHandler().Sync(p)

	if p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 39, p.GetParserRuleContext()) == 1 {
		{
			p.SetState(325)
			p.FieldBody()
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

// IFieldDefaultContext is an interface to support dynamic dispatch.
type IFieldDefaultContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	ExpressionValue() IExpressionValueContext
	ArrayDefault() IArrayDefaultContext

	// IsFieldDefaultContext differentiates from other interfaces.
	IsFieldDefaultContext()
}

type FieldDefaultContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyFieldDefaultContext() *FieldDefaultContext {
	var p = new(FieldDefaultContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_fieldDefault
	return p
}

func InitEmptyFieldDefaultContext(p *FieldDefaultContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_fieldDefault
}

func (*FieldDefaultContext) IsFieldDefaultContext() {}

func NewFieldDefaultContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *FieldDefaultContext {
	var p = new(FieldDefaultContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = OracleParserRULE_fieldDefault

	return p
}

func (s *FieldDefaultContext) GetParser() antlr.Parser { return s.parser }

func (s *FieldDefaultContext) ExpressionValue() IExpressionValueContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IExpressionValueContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IExpressionValueContext)
}

func (s *FieldDefaultContext) ArrayDefault() IArrayDefaultContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IArrayDefaultContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IArrayDefaultContext)
}

func (s *FieldDefaultContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *FieldDefaultContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *FieldDefaultContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.EnterFieldDefault(s)
	}
}

func (s *FieldDefaultContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.ExitFieldDefault(s)
	}
}

func (p *OracleParser) FieldDefault() (localctx IFieldDefaultContext) {
	localctx = NewFieldDefaultContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 30, OracleParserRULE_fieldDefault)
	p.SetState(330)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case OracleParserTRIPLE_STRING_LIT, OracleParserSTRING_LIT, OracleParserFLOAT_LIT, OracleParserINT_LIT, OracleParserBOOL_LIT, OracleParserIDENT:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(328)
			p.ExpressionValue()
		}

	case OracleParserLBRACKET:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(329)
			p.ArrayDefault()
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

// IArrayDefaultContext is an interface to support dynamic dispatch.
type IArrayDefaultContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	LBRACKET() antlr.TerminalNode
	RBRACKET() antlr.TerminalNode
	AllNl() []INlContext
	Nl(i int) INlContext
	AllExpressionValue() []IExpressionValueContext
	ExpressionValue(i int) IExpressionValueContext
	AllCOMMA() []antlr.TerminalNode
	COMMA(i int) antlr.TerminalNode

	// IsArrayDefaultContext differentiates from other interfaces.
	IsArrayDefaultContext()
}

type ArrayDefaultContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyArrayDefaultContext() *ArrayDefaultContext {
	var p = new(ArrayDefaultContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_arrayDefault
	return p
}

func InitEmptyArrayDefaultContext(p *ArrayDefaultContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_arrayDefault
}

func (*ArrayDefaultContext) IsArrayDefaultContext() {}

func NewArrayDefaultContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ArrayDefaultContext {
	var p = new(ArrayDefaultContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = OracleParserRULE_arrayDefault

	return p
}

func (s *ArrayDefaultContext) GetParser() antlr.Parser { return s.parser }

func (s *ArrayDefaultContext) LBRACKET() antlr.TerminalNode {
	return s.GetToken(OracleParserLBRACKET, 0)
}

func (s *ArrayDefaultContext) RBRACKET() antlr.TerminalNode {
	return s.GetToken(OracleParserRBRACKET, 0)
}

func (s *ArrayDefaultContext) AllNl() []INlContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(INlContext); ok {
			len++
		}
	}

	tst := make([]INlContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(INlContext); ok {
			tst[i] = t.(INlContext)
			i++
		}
	}

	return tst
}

func (s *ArrayDefaultContext) Nl(i int) INlContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(INlContext); ok {
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

	return t.(INlContext)
}

func (s *ArrayDefaultContext) AllExpressionValue() []IExpressionValueContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IExpressionValueContext); ok {
			len++
		}
	}

	tst := make([]IExpressionValueContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IExpressionValueContext); ok {
			tst[i] = t.(IExpressionValueContext)
			i++
		}
	}

	return tst
}

func (s *ArrayDefaultContext) ExpressionValue(i int) IExpressionValueContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IExpressionValueContext); ok {
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

	return t.(IExpressionValueContext)
}

func (s *ArrayDefaultContext) AllCOMMA() []antlr.TerminalNode {
	return s.GetTokens(OracleParserCOMMA)
}

func (s *ArrayDefaultContext) COMMA(i int) antlr.TerminalNode {
	return s.GetToken(OracleParserCOMMA, i)
}

func (s *ArrayDefaultContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ArrayDefaultContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ArrayDefaultContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.EnterArrayDefault(s)
	}
}

func (s *ArrayDefaultContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.ExitArrayDefault(s)
	}
}

func (p *OracleParser) ArrayDefault() (localctx IArrayDefaultContext) {
	localctx = NewArrayDefaultContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 32, OracleParserRULE_arrayDefault)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(332)
		p.Match(OracleParserLBRACKET)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(336)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(333)
			p.Nl()
		}

		p.SetState(338)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	p.SetState(359)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&66060288) != 0 {
		{
			p.SetState(339)
			p.ExpressionValue()
		}
		p.SetState(350)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserCOMMA {
			{
				p.SetState(340)
				p.Match(OracleParserCOMMA)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}
			p.SetState(344)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)

			for _la == OracleParserNEWLINE {
				{
					p.SetState(341)
					p.Nl()
				}

				p.SetState(346)
				p.GetErrorHandler().Sync(p)
				if p.HasError() {
					goto errorExit
				}
				_la = p.GetTokenStream().LA(1)
			}
			{
				p.SetState(347)
				p.ExpressionValue()
			}

			p.SetState(352)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}
		p.SetState(356)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(353)
				p.Nl()
			}

			p.SetState(358)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}

	}
	{
		p.SetState(361)
		p.Match(OracleParserRBRACKET)
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

// IInlineDomainContext is an interface to support dynamic dispatch.
type IInlineDomainContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AT() antlr.TerminalNode
	IDENT() antlr.TerminalNode
	DomainContent() IDomainContentContext

	// IsInlineDomainContext differentiates from other interfaces.
	IsInlineDomainContext()
}

type InlineDomainContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyInlineDomainContext() *InlineDomainContext {
	var p = new(InlineDomainContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_inlineDomain
	return p
}

func InitEmptyInlineDomainContext(p *InlineDomainContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_inlineDomain
}

func (*InlineDomainContext) IsInlineDomainContext() {}

func NewInlineDomainContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *InlineDomainContext {
	var p = new(InlineDomainContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = OracleParserRULE_inlineDomain

	return p
}

func (s *InlineDomainContext) GetParser() antlr.Parser { return s.parser }

func (s *InlineDomainContext) AT() antlr.TerminalNode {
	return s.GetToken(OracleParserAT, 0)
}

func (s *InlineDomainContext) IDENT() antlr.TerminalNode {
	return s.GetToken(OracleParserIDENT, 0)
}

func (s *InlineDomainContext) DomainContent() IDomainContentContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IDomainContentContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IDomainContentContext)
}

func (s *InlineDomainContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *InlineDomainContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *InlineDomainContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.EnterInlineDomain(s)
	}
}

func (s *InlineDomainContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.ExitInlineDomain(s)
	}
}

func (p *OracleParser) InlineDomain() (localctx IInlineDomainContext) {
	localctx = NewInlineDomainContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 34, OracleParserRULE_inlineDomain)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(363)
		p.Match(OracleParserAT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(364)
		p.Match(OracleParserIDENT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(366)
	p.GetErrorHandler().Sync(p)

	if p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 46, p.GetParserRuleContext()) == 1 {
		{
			p.SetState(365)
			p.DomainContent()
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

// IFieldBodyContext is an interface to support dynamic dispatch.
type IFieldBodyContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	LBRACE() antlr.TerminalNode
	RBRACE() antlr.TerminalNode
	AllNl() []INlContext
	Nl(i int) INlContext
	AllDomain() []IDomainContext
	Domain(i int) IDomainContext

	// IsFieldBodyContext differentiates from other interfaces.
	IsFieldBodyContext()
}

type FieldBodyContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyFieldBodyContext() *FieldBodyContext {
	var p = new(FieldBodyContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_fieldBody
	return p
}

func InitEmptyFieldBodyContext(p *FieldBodyContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_fieldBody
}

func (*FieldBodyContext) IsFieldBodyContext() {}

func NewFieldBodyContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *FieldBodyContext {
	var p = new(FieldBodyContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = OracleParserRULE_fieldBody

	return p
}

func (s *FieldBodyContext) GetParser() antlr.Parser { return s.parser }

func (s *FieldBodyContext) LBRACE() antlr.TerminalNode {
	return s.GetToken(OracleParserLBRACE, 0)
}

func (s *FieldBodyContext) RBRACE() antlr.TerminalNode {
	return s.GetToken(OracleParserRBRACE, 0)
}

func (s *FieldBodyContext) AllNl() []INlContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(INlContext); ok {
			len++
		}
	}

	tst := make([]INlContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(INlContext); ok {
			tst[i] = t.(INlContext)
			i++
		}
	}

	return tst
}

func (s *FieldBodyContext) Nl(i int) INlContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(INlContext); ok {
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

	return t.(INlContext)
}

func (s *FieldBodyContext) AllDomain() []IDomainContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IDomainContext); ok {
			len++
		}
	}

	tst := make([]IDomainContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IDomainContext); ok {
			tst[i] = t.(IDomainContext)
			i++
		}
	}

	return tst
}

func (s *FieldBodyContext) Domain(i int) IDomainContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IDomainContext); ok {
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

	return t.(IDomainContext)
}

func (s *FieldBodyContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *FieldBodyContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *FieldBodyContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.EnterFieldBody(s)
	}
}

func (s *FieldBodyContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.ExitFieldBody(s)
	}
}

func (p *OracleParser) FieldBody() (localctx IFieldBodyContext) {
	localctx = NewFieldBodyContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 36, OracleParserRULE_fieldBody)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(371)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(368)
			p.Nl()
		}

		p.SetState(373)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(374)
		p.Match(OracleParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(378)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(375)
			p.Nl()
		}

		p.SetState(380)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	p.SetState(390)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserAT {
		{
			p.SetState(381)
			p.Domain()
		}
		p.SetState(385)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(382)
				p.Nl()
			}

			p.SetState(387)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}

		p.SetState(392)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(393)
		p.Match(OracleParserRBRACE)
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

// IDomainContext is an interface to support dynamic dispatch.
type IDomainContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AT() antlr.TerminalNode
	IDENT() antlr.TerminalNode
	DomainContent() IDomainContentContext

	// IsDomainContext differentiates from other interfaces.
	IsDomainContext()
}

type DomainContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyDomainContext() *DomainContext {
	var p = new(DomainContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_domain
	return p
}

func InitEmptyDomainContext(p *DomainContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_domain
}

func (*DomainContext) IsDomainContext() {}

func NewDomainContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *DomainContext {
	var p = new(DomainContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = OracleParserRULE_domain

	return p
}

func (s *DomainContext) GetParser() antlr.Parser { return s.parser }

func (s *DomainContext) AT() antlr.TerminalNode {
	return s.GetToken(OracleParserAT, 0)
}

func (s *DomainContext) IDENT() antlr.TerminalNode {
	return s.GetToken(OracleParserIDENT, 0)
}

func (s *DomainContext) DomainContent() IDomainContentContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IDomainContentContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IDomainContentContext)
}

func (s *DomainContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *DomainContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *DomainContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.EnterDomain(s)
	}
}

func (s *DomainContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.ExitDomain(s)
	}
}

func (p *OracleParser) Domain() (localctx IDomainContext) {
	localctx = NewDomainContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 38, OracleParserRULE_domain)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(395)
		p.Match(OracleParserAT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(396)
		p.Match(OracleParserIDENT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(398)
	p.GetErrorHandler().Sync(p)

	if p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 51, p.GetParserRuleContext()) == 1 {
		{
			p.SetState(397)
			p.DomainContent()
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

// IDomainContentContext is an interface to support dynamic dispatch.
type IDomainContentContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	DomainBlock() IDomainBlockContext
	Expression() IExpressionContext

	// IsDomainContentContext differentiates from other interfaces.
	IsDomainContentContext()
}

type DomainContentContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyDomainContentContext() *DomainContentContext {
	var p = new(DomainContentContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_domainContent
	return p
}

func InitEmptyDomainContentContext(p *DomainContentContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_domainContent
}

func (*DomainContentContext) IsDomainContentContext() {}

func NewDomainContentContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *DomainContentContext {
	var p = new(DomainContentContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = OracleParserRULE_domainContent

	return p
}

func (s *DomainContentContext) GetParser() antlr.Parser { return s.parser }

func (s *DomainContentContext) DomainBlock() IDomainBlockContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IDomainBlockContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IDomainBlockContext)
}

func (s *DomainContentContext) Expression() IExpressionContext {
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

func (s *DomainContentContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *DomainContentContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *DomainContentContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.EnterDomainContent(s)
	}
}

func (s *DomainContentContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.ExitDomainContent(s)
	}
}

func (p *OracleParser) DomainContent() (localctx IDomainContentContext) {
	localctx = NewDomainContentContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 40, OracleParserRULE_domainContent)
	p.SetState(402)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case OracleParserLBRACE, OracleParserNEWLINE:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(400)
			p.DomainBlock()
		}

	case OracleParserIDENT:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(401)
			p.Expression()
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

// IDomainBlockContext is an interface to support dynamic dispatch.
type IDomainBlockContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	LBRACE() antlr.TerminalNode
	RBRACE() antlr.TerminalNode
	AllNl() []INlContext
	Nl(i int) INlContext
	AllExpression() []IExpressionContext
	Expression(i int) IExpressionContext

	// IsDomainBlockContext differentiates from other interfaces.
	IsDomainBlockContext()
}

type DomainBlockContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyDomainBlockContext() *DomainBlockContext {
	var p = new(DomainBlockContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_domainBlock
	return p
}

func InitEmptyDomainBlockContext(p *DomainBlockContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_domainBlock
}

func (*DomainBlockContext) IsDomainBlockContext() {}

func NewDomainBlockContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *DomainBlockContext {
	var p = new(DomainBlockContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = OracleParserRULE_domainBlock

	return p
}

func (s *DomainBlockContext) GetParser() antlr.Parser { return s.parser }

func (s *DomainBlockContext) LBRACE() antlr.TerminalNode {
	return s.GetToken(OracleParserLBRACE, 0)
}

func (s *DomainBlockContext) RBRACE() antlr.TerminalNode {
	return s.GetToken(OracleParserRBRACE, 0)
}

func (s *DomainBlockContext) AllNl() []INlContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(INlContext); ok {
			len++
		}
	}

	tst := make([]INlContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(INlContext); ok {
			tst[i] = t.(INlContext)
			i++
		}
	}

	return tst
}

func (s *DomainBlockContext) Nl(i int) INlContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(INlContext); ok {
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

	return t.(INlContext)
}

func (s *DomainBlockContext) AllExpression() []IExpressionContext {
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

func (s *DomainBlockContext) Expression(i int) IExpressionContext {
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

func (s *DomainBlockContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *DomainBlockContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *DomainBlockContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.EnterDomainBlock(s)
	}
}

func (s *DomainBlockContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.ExitDomainBlock(s)
	}
}

func (p *OracleParser) DomainBlock() (localctx IDomainBlockContext) {
	localctx = NewDomainBlockContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 42, OracleParserRULE_domainBlock)
	var _la int

	var _alt int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(407)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(404)
			p.Nl()
		}

		p.SetState(409)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(410)
		p.Match(OracleParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(414)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 54, p.GetParserRuleContext())
	if p.HasError() {
		goto errorExit
	}
	for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
		if _alt == 1 {
			{
				p.SetState(411)
				p.Nl()
			}

		}
		p.SetState(416)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 54, p.GetParserRuleContext())
		if p.HasError() {
			goto errorExit
		}
	}
	p.SetState(430)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == OracleParserIDENT {
		{
			p.SetState(417)
			p.Expression()
		}
		p.SetState(427)
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
				p.SetState(419)
				p.GetErrorHandler().Sync(p)
				if p.HasError() {
					goto errorExit
				}
				_la = p.GetTokenStream().LA(1)

				for ok := true; ok; ok = _la == OracleParserNEWLINE {
					{
						p.SetState(418)
						p.Nl()
					}

					p.SetState(421)
					p.GetErrorHandler().Sync(p)
					if p.HasError() {
						goto errorExit
					}
					_la = p.GetTokenStream().LA(1)
				}
				{
					p.SetState(423)
					p.Expression()
				}

			}
			p.SetState(429)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 56, p.GetParserRuleContext())
			if p.HasError() {
				goto errorExit
			}
		}

	}
	p.SetState(435)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(432)
			p.Nl()
		}

		p.SetState(437)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(438)
		p.Match(OracleParserRBRACE)
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

// ITypeRefContext is an interface to support dynamic dispatch.
type ITypeRefContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser
	// IsTypeRefContext differentiates from other interfaces.
	IsTypeRefContext()
}

type TypeRefContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyTypeRefContext() *TypeRefContext {
	var p = new(TypeRefContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_typeRef
	return p
}

func InitEmptyTypeRefContext(p *TypeRefContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_typeRef
}

func (*TypeRefContext) IsTypeRefContext() {}

func NewTypeRefContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *TypeRefContext {
	var p = new(TypeRefContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = OracleParserRULE_typeRef

	return p
}

func (s *TypeRefContext) GetParser() antlr.Parser { return s.parser }

func (s *TypeRefContext) CopyAll(ctx *TypeRefContext) {
	s.CopyFrom(&ctx.BaseParserRuleContext)
}

func (s *TypeRefContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *TypeRefContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

type TypeRefMapContext struct {
	TypeRefContext
}

func NewTypeRefMapContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *TypeRefMapContext {
	var p = new(TypeRefMapContext)

	InitEmptyTypeRefContext(&p.TypeRefContext)
	p.parser = parser
	p.CopyAll(ctx.(*TypeRefContext))

	return p
}

func (s *TypeRefMapContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *TypeRefMapContext) MapType() IMapTypeContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IMapTypeContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IMapTypeContext)
}

func (s *TypeRefMapContext) TypeModifiers() ITypeModifiersContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ITypeModifiersContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ITypeModifiersContext)
}

func (s *TypeRefMapContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.EnterTypeRefMap(s)
	}
}

func (s *TypeRefMapContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.ExitTypeRefMap(s)
	}
}

type TypeRefNormalContext struct {
	TypeRefContext
}

func NewTypeRefNormalContext(parser antlr.Parser, ctx antlr.ParserRuleContext) *TypeRefNormalContext {
	var p = new(TypeRefNormalContext)

	InitEmptyTypeRefContext(&p.TypeRefContext)
	p.parser = parser
	p.CopyAll(ctx.(*TypeRefContext))

	return p
}

func (s *TypeRefNormalContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *TypeRefNormalContext) QualifiedIdent() IQualifiedIdentContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IQualifiedIdentContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IQualifiedIdentContext)
}

func (s *TypeRefNormalContext) TypeArgs() ITypeArgsContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ITypeArgsContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ITypeArgsContext)
}

func (s *TypeRefNormalContext) ArrayModifier() IArrayModifierContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IArrayModifierContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IArrayModifierContext)
}

func (s *TypeRefNormalContext) TypeModifiers() ITypeModifiersContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ITypeModifiersContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ITypeModifiersContext)
}

func (s *TypeRefNormalContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.EnterTypeRefNormal(s)
	}
}

func (s *TypeRefNormalContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.ExitTypeRefNormal(s)
	}
}

func (p *OracleParser) TypeRef() (localctx ITypeRefContext) {
	localctx = NewTypeRefContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 44, OracleParserRULE_typeRef)
	var _la int

	p.SetState(454)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case OracleParserMAP:
		localctx = NewTypeRefMapContext(p, localctx)
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(440)
			p.MapType()
		}
		p.SetState(442)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		if _la == OracleParserQUESTION {
			{
				p.SetState(441)
				p.TypeModifiers()
			}

		}

	case OracleParserIDENT:
		localctx = NewTypeRefNormalContext(p, localctx)
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(444)
			p.QualifiedIdent()
		}
		p.SetState(446)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		if _la == OracleParserLT {
			{
				p.SetState(445)
				p.TypeArgs()
			}

		}
		p.SetState(449)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		if _la == OracleParserLBRACKET {
			{
				p.SetState(448)
				p.ArrayModifier()
			}

		}
		p.SetState(452)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		if _la == OracleParserQUESTION {
			{
				p.SetState(451)
				p.TypeModifiers()
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

// IArrayModifierContext is an interface to support dynamic dispatch.
type IArrayModifierContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	LBRACKET() antlr.TerminalNode
	RBRACKET() antlr.TerminalNode
	INT_LIT() antlr.TerminalNode

	// IsArrayModifierContext differentiates from other interfaces.
	IsArrayModifierContext()
}

type ArrayModifierContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyArrayModifierContext() *ArrayModifierContext {
	var p = new(ArrayModifierContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_arrayModifier
	return p
}

func InitEmptyArrayModifierContext(p *ArrayModifierContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_arrayModifier
}

func (*ArrayModifierContext) IsArrayModifierContext() {}

func NewArrayModifierContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ArrayModifierContext {
	var p = new(ArrayModifierContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = OracleParserRULE_arrayModifier

	return p
}

func (s *ArrayModifierContext) GetParser() antlr.Parser { return s.parser }

func (s *ArrayModifierContext) LBRACKET() antlr.TerminalNode {
	return s.GetToken(OracleParserLBRACKET, 0)
}

func (s *ArrayModifierContext) RBRACKET() antlr.TerminalNode {
	return s.GetToken(OracleParserRBRACKET, 0)
}

func (s *ArrayModifierContext) INT_LIT() antlr.TerminalNode {
	return s.GetToken(OracleParserINT_LIT, 0)
}

func (s *ArrayModifierContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ArrayModifierContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ArrayModifierContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.EnterArrayModifier(s)
	}
}

func (s *ArrayModifierContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.ExitArrayModifier(s)
	}
}

func (p *OracleParser) ArrayModifier() (localctx IArrayModifierContext) {
	localctx = NewArrayModifierContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 46, OracleParserRULE_arrayModifier)
	p.SetState(461)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 64, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(456)
			p.Match(OracleParserLBRACKET)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(457)
			p.Match(OracleParserRBRACKET)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(458)
			p.Match(OracleParserLBRACKET)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(459)
			p.Match(OracleParserINT_LIT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(460)
			p.Match(OracleParserRBRACKET)
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

// IMapTypeContext is an interface to support dynamic dispatch.
type IMapTypeContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	MAP() antlr.TerminalNode
	LT() antlr.TerminalNode
	AllTypeRef() []ITypeRefContext
	TypeRef(i int) ITypeRefContext
	COMMA() antlr.TerminalNode
	GT() antlr.TerminalNode

	// IsMapTypeContext differentiates from other interfaces.
	IsMapTypeContext()
}

type MapTypeContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyMapTypeContext() *MapTypeContext {
	var p = new(MapTypeContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_mapType
	return p
}

func InitEmptyMapTypeContext(p *MapTypeContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_mapType
}

func (*MapTypeContext) IsMapTypeContext() {}

func NewMapTypeContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *MapTypeContext {
	var p = new(MapTypeContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = OracleParserRULE_mapType

	return p
}

func (s *MapTypeContext) GetParser() antlr.Parser { return s.parser }

func (s *MapTypeContext) MAP() antlr.TerminalNode {
	return s.GetToken(OracleParserMAP, 0)
}

func (s *MapTypeContext) LT() antlr.TerminalNode {
	return s.GetToken(OracleParserLT, 0)
}

func (s *MapTypeContext) AllTypeRef() []ITypeRefContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(ITypeRefContext); ok {
			len++
		}
	}

	tst := make([]ITypeRefContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(ITypeRefContext); ok {
			tst[i] = t.(ITypeRefContext)
			i++
		}
	}

	return tst
}

func (s *MapTypeContext) TypeRef(i int) ITypeRefContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ITypeRefContext); ok {
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

	return t.(ITypeRefContext)
}

func (s *MapTypeContext) COMMA() antlr.TerminalNode {
	return s.GetToken(OracleParserCOMMA, 0)
}

func (s *MapTypeContext) GT() antlr.TerminalNode {
	return s.GetToken(OracleParserGT, 0)
}

func (s *MapTypeContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *MapTypeContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *MapTypeContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.EnterMapType(s)
	}
}

func (s *MapTypeContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.ExitMapType(s)
	}
}

func (p *OracleParser) MapType() (localctx IMapTypeContext) {
	localctx = NewMapTypeContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 48, OracleParserRULE_mapType)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(463)
		p.Match(OracleParserMAP)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(464)
		p.Match(OracleParserLT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(465)
		p.TypeRef()
	}
	{
		p.SetState(466)
		p.Match(OracleParserCOMMA)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(467)
		p.TypeRef()
	}
	{
		p.SetState(468)
		p.Match(OracleParserGT)
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

// ITypeArgsContext is an interface to support dynamic dispatch.
type ITypeArgsContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	LT() antlr.TerminalNode
	AllTypeRef() []ITypeRefContext
	TypeRef(i int) ITypeRefContext
	GT() antlr.TerminalNode
	AllCOMMA() []antlr.TerminalNode
	COMMA(i int) antlr.TerminalNode

	// IsTypeArgsContext differentiates from other interfaces.
	IsTypeArgsContext()
}

type TypeArgsContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyTypeArgsContext() *TypeArgsContext {
	var p = new(TypeArgsContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_typeArgs
	return p
}

func InitEmptyTypeArgsContext(p *TypeArgsContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_typeArgs
}

func (*TypeArgsContext) IsTypeArgsContext() {}

func NewTypeArgsContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *TypeArgsContext {
	var p = new(TypeArgsContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = OracleParserRULE_typeArgs

	return p
}

func (s *TypeArgsContext) GetParser() antlr.Parser { return s.parser }

func (s *TypeArgsContext) LT() antlr.TerminalNode {
	return s.GetToken(OracleParserLT, 0)
}

func (s *TypeArgsContext) AllTypeRef() []ITypeRefContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(ITypeRefContext); ok {
			len++
		}
	}

	tst := make([]ITypeRefContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(ITypeRefContext); ok {
			tst[i] = t.(ITypeRefContext)
			i++
		}
	}

	return tst
}

func (s *TypeArgsContext) TypeRef(i int) ITypeRefContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ITypeRefContext); ok {
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

	return t.(ITypeRefContext)
}

func (s *TypeArgsContext) GT() antlr.TerminalNode {
	return s.GetToken(OracleParserGT, 0)
}

func (s *TypeArgsContext) AllCOMMA() []antlr.TerminalNode {
	return s.GetTokens(OracleParserCOMMA)
}

func (s *TypeArgsContext) COMMA(i int) antlr.TerminalNode {
	return s.GetToken(OracleParserCOMMA, i)
}

func (s *TypeArgsContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *TypeArgsContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *TypeArgsContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.EnterTypeArgs(s)
	}
}

func (s *TypeArgsContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.ExitTypeArgs(s)
	}
}

func (p *OracleParser) TypeArgs() (localctx ITypeArgsContext) {
	localctx = NewTypeArgsContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 50, OracleParserRULE_typeArgs)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(470)
		p.Match(OracleParserLT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(471)
		p.TypeRef()
	}
	p.SetState(476)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserCOMMA {
		{
			p.SetState(472)
			p.Match(OracleParserCOMMA)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(473)
			p.TypeRef()
		}

		p.SetState(478)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(479)
		p.Match(OracleParserGT)
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

// ITypeModifiersContext is an interface to support dynamic dispatch.
type ITypeModifiersContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AllQUESTION() []antlr.TerminalNode
	QUESTION(i int) antlr.TerminalNode

	// IsTypeModifiersContext differentiates from other interfaces.
	IsTypeModifiersContext()
}

type TypeModifiersContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyTypeModifiersContext() *TypeModifiersContext {
	var p = new(TypeModifiersContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_typeModifiers
	return p
}

func InitEmptyTypeModifiersContext(p *TypeModifiersContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_typeModifiers
}

func (*TypeModifiersContext) IsTypeModifiersContext() {}

func NewTypeModifiersContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *TypeModifiersContext {
	var p = new(TypeModifiersContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = OracleParserRULE_typeModifiers

	return p
}

func (s *TypeModifiersContext) GetParser() antlr.Parser { return s.parser }

func (s *TypeModifiersContext) AllQUESTION() []antlr.TerminalNode {
	return s.GetTokens(OracleParserQUESTION)
}

func (s *TypeModifiersContext) QUESTION(i int) antlr.TerminalNode {
	return s.GetToken(OracleParserQUESTION, i)
}

func (s *TypeModifiersContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *TypeModifiersContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *TypeModifiersContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.EnterTypeModifiers(s)
	}
}

func (s *TypeModifiersContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.ExitTypeModifiers(s)
	}
}

func (p *OracleParser) TypeModifiers() (localctx ITypeModifiersContext) {
	localctx = NewTypeModifiersContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 52, OracleParserRULE_typeModifiers)
	p.SetState(484)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 66, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(481)
			p.Match(OracleParserQUESTION)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(482)
			p.Match(OracleParserQUESTION)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(483)
			p.Match(OracleParserQUESTION)
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

// IQualifiedIdentContext is an interface to support dynamic dispatch.
type IQualifiedIdentContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AllIDENT() []antlr.TerminalNode
	IDENT(i int) antlr.TerminalNode
	DOT() antlr.TerminalNode

	// IsQualifiedIdentContext differentiates from other interfaces.
	IsQualifiedIdentContext()
}

type QualifiedIdentContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyQualifiedIdentContext() *QualifiedIdentContext {
	var p = new(QualifiedIdentContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_qualifiedIdent
	return p
}

func InitEmptyQualifiedIdentContext(p *QualifiedIdentContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_qualifiedIdent
}

func (*QualifiedIdentContext) IsQualifiedIdentContext() {}

func NewQualifiedIdentContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *QualifiedIdentContext {
	var p = new(QualifiedIdentContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = OracleParserRULE_qualifiedIdent

	return p
}

func (s *QualifiedIdentContext) GetParser() antlr.Parser { return s.parser }

func (s *QualifiedIdentContext) AllIDENT() []antlr.TerminalNode {
	return s.GetTokens(OracleParserIDENT)
}

func (s *QualifiedIdentContext) IDENT(i int) antlr.TerminalNode {
	return s.GetToken(OracleParserIDENT, i)
}

func (s *QualifiedIdentContext) DOT() antlr.TerminalNode {
	return s.GetToken(OracleParserDOT, 0)
}

func (s *QualifiedIdentContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *QualifiedIdentContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *QualifiedIdentContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.EnterQualifiedIdent(s)
	}
}

func (s *QualifiedIdentContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.ExitQualifiedIdent(s)
	}
}

func (p *OracleParser) QualifiedIdent() (localctx IQualifiedIdentContext) {
	localctx = NewQualifiedIdentContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 54, OracleParserRULE_qualifiedIdent)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(486)
		p.Match(OracleParserIDENT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(489)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == OracleParserDOT {
		{
			p.SetState(487)
			p.Match(OracleParserDOT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(488)
			p.Match(OracleParserIDENT)
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

// IExpressionContext is an interface to support dynamic dispatch.
type IExpressionContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	IDENT() antlr.TerminalNode
	AllExpressionValue() []IExpressionValueContext
	ExpressionValue(i int) IExpressionValueContext

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
	p.RuleIndex = OracleParserRULE_expression
	return p
}

func InitEmptyExpressionContext(p *ExpressionContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_expression
}

func (*ExpressionContext) IsExpressionContext() {}

func NewExpressionContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ExpressionContext {
	var p = new(ExpressionContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = OracleParserRULE_expression

	return p
}

func (s *ExpressionContext) GetParser() antlr.Parser { return s.parser }

func (s *ExpressionContext) IDENT() antlr.TerminalNode {
	return s.GetToken(OracleParserIDENT, 0)
}

func (s *ExpressionContext) AllExpressionValue() []IExpressionValueContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IExpressionValueContext); ok {
			len++
		}
	}

	tst := make([]IExpressionValueContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IExpressionValueContext); ok {
			tst[i] = t.(IExpressionValueContext)
			i++
		}
	}

	return tst
}

func (s *ExpressionContext) ExpressionValue(i int) IExpressionValueContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IExpressionValueContext); ok {
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

	return t.(IExpressionValueContext)
}

func (s *ExpressionContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ExpressionContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ExpressionContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.EnterExpression(s)
	}
}

func (s *ExpressionContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.ExitExpression(s)
	}
}

func (p *OracleParser) Expression() (localctx IExpressionContext) {
	localctx = NewExpressionContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 56, OracleParserRULE_expression)
	var _alt int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(491)
		p.Match(OracleParserIDENT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(495)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 68, p.GetParserRuleContext())
	if p.HasError() {
		goto errorExit
	}
	for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
		if _alt == 1 {
			{
				p.SetState(492)
				p.ExpressionValue()
			}

		}
		p.SetState(497)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 68, p.GetParserRuleContext())
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

// IExpressionValueContext is an interface to support dynamic dispatch.
type IExpressionValueContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	TRIPLE_STRING_LIT() antlr.TerminalNode
	STRING_LIT() antlr.TerminalNode
	INT_LIT() antlr.TerminalNode
	FLOAT_LIT() antlr.TerminalNode
	BOOL_LIT() antlr.TerminalNode
	QualifiedIdent() IQualifiedIdentContext

	// IsExpressionValueContext differentiates from other interfaces.
	IsExpressionValueContext()
}

type ExpressionValueContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyExpressionValueContext() *ExpressionValueContext {
	var p = new(ExpressionValueContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_expressionValue
	return p
}

func InitEmptyExpressionValueContext(p *ExpressionValueContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_expressionValue
}

func (*ExpressionValueContext) IsExpressionValueContext() {}

func NewExpressionValueContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ExpressionValueContext {
	var p = new(ExpressionValueContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = OracleParserRULE_expressionValue

	return p
}

func (s *ExpressionValueContext) GetParser() antlr.Parser { return s.parser }

func (s *ExpressionValueContext) TRIPLE_STRING_LIT() antlr.TerminalNode {
	return s.GetToken(OracleParserTRIPLE_STRING_LIT, 0)
}

func (s *ExpressionValueContext) STRING_LIT() antlr.TerminalNode {
	return s.GetToken(OracleParserSTRING_LIT, 0)
}

func (s *ExpressionValueContext) INT_LIT() antlr.TerminalNode {
	return s.GetToken(OracleParserINT_LIT, 0)
}

func (s *ExpressionValueContext) FLOAT_LIT() antlr.TerminalNode {
	return s.GetToken(OracleParserFLOAT_LIT, 0)
}

func (s *ExpressionValueContext) BOOL_LIT() antlr.TerminalNode {
	return s.GetToken(OracleParserBOOL_LIT, 0)
}

func (s *ExpressionValueContext) QualifiedIdent() IQualifiedIdentContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IQualifiedIdentContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IQualifiedIdentContext)
}

func (s *ExpressionValueContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ExpressionValueContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ExpressionValueContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.EnterExpressionValue(s)
	}
}

func (s *ExpressionValueContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.ExitExpressionValue(s)
	}
}

func (p *OracleParser) ExpressionValue() (localctx IExpressionValueContext) {
	localctx = NewExpressionValueContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 58, OracleParserRULE_expressionValue)
	p.SetState(504)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case OracleParserTRIPLE_STRING_LIT:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(498)
			p.Match(OracleParserTRIPLE_STRING_LIT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case OracleParserSTRING_LIT:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(499)
			p.Match(OracleParserSTRING_LIT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case OracleParserINT_LIT:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(500)
			p.Match(OracleParserINT_LIT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case OracleParserFLOAT_LIT:
		p.EnterOuterAlt(localctx, 4)
		{
			p.SetState(501)
			p.Match(OracleParserFLOAT_LIT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case OracleParserBOOL_LIT:
		p.EnterOuterAlt(localctx, 5)
		{
			p.SetState(502)
			p.Match(OracleParserBOOL_LIT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case OracleParserIDENT:
		p.EnterOuterAlt(localctx, 6)
		{
			p.SetState(503)
			p.QualifiedIdent()
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

// IEnumDefContext is an interface to support dynamic dispatch.
type IEnumDefContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	IDENT() antlr.TerminalNode
	ENUM() antlr.TerminalNode
	LBRACE() antlr.TerminalNode
	EnumBody() IEnumBodyContext
	RBRACE() antlr.TerminalNode
	EXTENDS() antlr.TerminalNode
	TypeRefList() ITypeRefListContext
	AllNl() []INlContext
	Nl(i int) INlContext

	// IsEnumDefContext differentiates from other interfaces.
	IsEnumDefContext()
}

type EnumDefContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyEnumDefContext() *EnumDefContext {
	var p = new(EnumDefContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_enumDef
	return p
}

func InitEmptyEnumDefContext(p *EnumDefContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_enumDef
}

func (*EnumDefContext) IsEnumDefContext() {}

func NewEnumDefContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *EnumDefContext {
	var p = new(EnumDefContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = OracleParserRULE_enumDef

	return p
}

func (s *EnumDefContext) GetParser() antlr.Parser { return s.parser }

func (s *EnumDefContext) IDENT() antlr.TerminalNode {
	return s.GetToken(OracleParserIDENT, 0)
}

func (s *EnumDefContext) ENUM() antlr.TerminalNode {
	return s.GetToken(OracleParserENUM, 0)
}

func (s *EnumDefContext) LBRACE() antlr.TerminalNode {
	return s.GetToken(OracleParserLBRACE, 0)
}

func (s *EnumDefContext) EnumBody() IEnumBodyContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IEnumBodyContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IEnumBodyContext)
}

func (s *EnumDefContext) RBRACE() antlr.TerminalNode {
	return s.GetToken(OracleParserRBRACE, 0)
}

func (s *EnumDefContext) EXTENDS() antlr.TerminalNode {
	return s.GetToken(OracleParserEXTENDS, 0)
}

func (s *EnumDefContext) TypeRefList() ITypeRefListContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ITypeRefListContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ITypeRefListContext)
}

func (s *EnumDefContext) AllNl() []INlContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(INlContext); ok {
			len++
		}
	}

	tst := make([]INlContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(INlContext); ok {
			tst[i] = t.(INlContext)
			i++
		}
	}

	return tst
}

func (s *EnumDefContext) Nl(i int) INlContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(INlContext); ok {
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

	return t.(INlContext)
}

func (s *EnumDefContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *EnumDefContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *EnumDefContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.EnterEnumDef(s)
	}
}

func (s *EnumDefContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.ExitEnumDef(s)
	}
}

func (p *OracleParser) EnumDef() (localctx IEnumDefContext) {
	localctx = NewEnumDefContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 60, OracleParserRULE_enumDef)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(506)
		p.Match(OracleParserIDENT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(507)
		p.Match(OracleParserENUM)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(510)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == OracleParserEXTENDS {
		{
			p.SetState(508)
			p.Match(OracleParserEXTENDS)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(509)
			p.TypeRefList()
		}

	}
	p.SetState(515)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(512)
			p.Nl()
		}

		p.SetState(517)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(518)
		p.Match(OracleParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(522)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(519)
			p.Nl()
		}

		p.SetState(524)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(525)
		p.EnumBody()
	}
	{
		p.SetState(526)
		p.Match(OracleParserRBRACE)
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

// IEnumBodyContext is an interface to support dynamic dispatch.
type IEnumBodyContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AllEnumValue() []IEnumValueContext
	EnumValue(i int) IEnumValueContext
	AllDomain() []IDomainContext
	Domain(i int) IDomainContext
	AllNl() []INlContext
	Nl(i int) INlContext

	// IsEnumBodyContext differentiates from other interfaces.
	IsEnumBodyContext()
}

type EnumBodyContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyEnumBodyContext() *EnumBodyContext {
	var p = new(EnumBodyContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_enumBody
	return p
}

func InitEmptyEnumBodyContext(p *EnumBodyContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_enumBody
}

func (*EnumBodyContext) IsEnumBodyContext() {}

func NewEnumBodyContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *EnumBodyContext {
	var p = new(EnumBodyContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = OracleParserRULE_enumBody

	return p
}

func (s *EnumBodyContext) GetParser() antlr.Parser { return s.parser }

func (s *EnumBodyContext) AllEnumValue() []IEnumValueContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IEnumValueContext); ok {
			len++
		}
	}

	tst := make([]IEnumValueContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IEnumValueContext); ok {
			tst[i] = t.(IEnumValueContext)
			i++
		}
	}

	return tst
}

func (s *EnumBodyContext) EnumValue(i int) IEnumValueContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IEnumValueContext); ok {
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

	return t.(IEnumValueContext)
}

func (s *EnumBodyContext) AllDomain() []IDomainContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IDomainContext); ok {
			len++
		}
	}

	tst := make([]IDomainContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IDomainContext); ok {
			tst[i] = t.(IDomainContext)
			i++
		}
	}

	return tst
}

func (s *EnumBodyContext) Domain(i int) IDomainContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IDomainContext); ok {
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

	return t.(IDomainContext)
}

func (s *EnumBodyContext) AllNl() []INlContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(INlContext); ok {
			len++
		}
	}

	tst := make([]INlContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(INlContext); ok {
			tst[i] = t.(INlContext)
			i++
		}
	}

	return tst
}

func (s *EnumBodyContext) Nl(i int) INlContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(INlContext); ok {
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

	return t.(INlContext)
}

func (s *EnumBodyContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *EnumBodyContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *EnumBodyContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.EnterEnumBody(s)
	}
}

func (s *EnumBodyContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.ExitEnumBody(s)
	}
}

func (p *OracleParser) EnumBody() (localctx IEnumBodyContext) {
	localctx = NewEnumBodyContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 62, OracleParserRULE_enumBody)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(540)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserAT || _la == OracleParserIDENT {
		p.SetState(530)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}

		switch p.GetTokenStream().LA(1) {
		case OracleParserIDENT:
			{
				p.SetState(528)
				p.EnumValue()
			}

		case OracleParserAT:
			{
				p.SetState(529)
				p.Domain()
			}

		default:
			p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
			goto errorExit
		}
		p.SetState(535)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(532)
				p.Nl()
			}

			p.SetState(537)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}

		p.SetState(542)
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

// IEnumValueContext is an interface to support dynamic dispatch.
type IEnumValueContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	IDENT() antlr.TerminalNode
	EQUALS() antlr.TerminalNode
	INT_LIT() antlr.TerminalNode
	STRING_LIT() antlr.TerminalNode
	EnumValueBody() IEnumValueBodyContext

	// IsEnumValueContext differentiates from other interfaces.
	IsEnumValueContext()
}

type EnumValueContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyEnumValueContext() *EnumValueContext {
	var p = new(EnumValueContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_enumValue
	return p
}

func InitEmptyEnumValueContext(p *EnumValueContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_enumValue
}

func (*EnumValueContext) IsEnumValueContext() {}

func NewEnumValueContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *EnumValueContext {
	var p = new(EnumValueContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = OracleParserRULE_enumValue

	return p
}

func (s *EnumValueContext) GetParser() antlr.Parser { return s.parser }

func (s *EnumValueContext) IDENT() antlr.TerminalNode {
	return s.GetToken(OracleParserIDENT, 0)
}

func (s *EnumValueContext) EQUALS() antlr.TerminalNode {
	return s.GetToken(OracleParserEQUALS, 0)
}

func (s *EnumValueContext) INT_LIT() antlr.TerminalNode {
	return s.GetToken(OracleParserINT_LIT, 0)
}

func (s *EnumValueContext) STRING_LIT() antlr.TerminalNode {
	return s.GetToken(OracleParserSTRING_LIT, 0)
}

func (s *EnumValueContext) EnumValueBody() IEnumValueBodyContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IEnumValueBodyContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IEnumValueBodyContext)
}

func (s *EnumValueContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *EnumValueContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *EnumValueContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.EnterEnumValue(s)
	}
}

func (s *EnumValueContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.ExitEnumValue(s)
	}
}

func (p *OracleParser) EnumValue() (localctx IEnumValueContext) {
	localctx = NewEnumValueContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 64, OracleParserRULE_enumValue)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(543)
		p.Match(OracleParserIDENT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(544)
		p.Match(OracleParserEQUALS)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(545)
		_la = p.GetTokenStream().LA(1)

		if !(_la == OracleParserSTRING_LIT || _la == OracleParserINT_LIT) {
			p.GetErrorHandler().RecoverInline(p)
		} else {
			p.GetErrorHandler().ReportMatch(p)
			p.Consume()
		}
	}
	p.SetState(547)
	p.GetErrorHandler().Sync(p)

	if p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 76, p.GetParserRuleContext()) == 1 {
		{
			p.SetState(546)
			p.EnumValueBody()
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

// IEnumValueBodyContext is an interface to support dynamic dispatch.
type IEnumValueBodyContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	LBRACE() antlr.TerminalNode
	RBRACE() antlr.TerminalNode
	AllNl() []INlContext
	Nl(i int) INlContext
	AllDomain() []IDomainContext
	Domain(i int) IDomainContext

	// IsEnumValueBodyContext differentiates from other interfaces.
	IsEnumValueBodyContext()
}

type EnumValueBodyContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyEnumValueBodyContext() *EnumValueBodyContext {
	var p = new(EnumValueBodyContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_enumValueBody
	return p
}

func InitEmptyEnumValueBodyContext(p *EnumValueBodyContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_enumValueBody
}

func (*EnumValueBodyContext) IsEnumValueBodyContext() {}

func NewEnumValueBodyContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *EnumValueBodyContext {
	var p = new(EnumValueBodyContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = OracleParserRULE_enumValueBody

	return p
}

func (s *EnumValueBodyContext) GetParser() antlr.Parser { return s.parser }

func (s *EnumValueBodyContext) LBRACE() antlr.TerminalNode {
	return s.GetToken(OracleParserLBRACE, 0)
}

func (s *EnumValueBodyContext) RBRACE() antlr.TerminalNode {
	return s.GetToken(OracleParserRBRACE, 0)
}

func (s *EnumValueBodyContext) AllNl() []INlContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(INlContext); ok {
			len++
		}
	}

	tst := make([]INlContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(INlContext); ok {
			tst[i] = t.(INlContext)
			i++
		}
	}

	return tst
}

func (s *EnumValueBodyContext) Nl(i int) INlContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(INlContext); ok {
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

	return t.(INlContext)
}

func (s *EnumValueBodyContext) AllDomain() []IDomainContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IDomainContext); ok {
			len++
		}
	}

	tst := make([]IDomainContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IDomainContext); ok {
			tst[i] = t.(IDomainContext)
			i++
		}
	}

	return tst
}

func (s *EnumValueBodyContext) Domain(i int) IDomainContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IDomainContext); ok {
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

	return t.(IDomainContext)
}

func (s *EnumValueBodyContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *EnumValueBodyContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *EnumValueBodyContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.EnterEnumValueBody(s)
	}
}

func (s *EnumValueBodyContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.ExitEnumValueBody(s)
	}
}

func (p *OracleParser) EnumValueBody() (localctx IEnumValueBodyContext) {
	localctx = NewEnumValueBodyContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 66, OracleParserRULE_enumValueBody)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(552)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(549)
			p.Nl()
		}

		p.SetState(554)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(555)
		p.Match(OracleParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(559)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(556)
			p.Nl()
		}

		p.SetState(561)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	p.SetState(571)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserAT {
		{
			p.SetState(562)
			p.Domain()
		}
		p.SetState(566)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(563)
				p.Nl()
			}

			p.SetState(568)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}

		p.SetState(573)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(574)
		p.Match(OracleParserRBRACE)
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

// ITypeDefDefContext is an interface to support dynamic dispatch.
type ITypeDefDefContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	IDENT() antlr.TerminalNode
	TypeRef() ITypeRefContext
	TypeParams() ITypeParamsContext
	TypeDefBody() ITypeDefBodyContext

	// IsTypeDefDefContext differentiates from other interfaces.
	IsTypeDefDefContext()
}

type TypeDefDefContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyTypeDefDefContext() *TypeDefDefContext {
	var p = new(TypeDefDefContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_typeDefDef
	return p
}

func InitEmptyTypeDefDefContext(p *TypeDefDefContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_typeDefDef
}

func (*TypeDefDefContext) IsTypeDefDefContext() {}

func NewTypeDefDefContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *TypeDefDefContext {
	var p = new(TypeDefDefContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = OracleParserRULE_typeDefDef

	return p
}

func (s *TypeDefDefContext) GetParser() antlr.Parser { return s.parser }

func (s *TypeDefDefContext) IDENT() antlr.TerminalNode {
	return s.GetToken(OracleParserIDENT, 0)
}

func (s *TypeDefDefContext) TypeRef() ITypeRefContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ITypeRefContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ITypeRefContext)
}

func (s *TypeDefDefContext) TypeParams() ITypeParamsContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ITypeParamsContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ITypeParamsContext)
}

func (s *TypeDefDefContext) TypeDefBody() ITypeDefBodyContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ITypeDefBodyContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ITypeDefBodyContext)
}

func (s *TypeDefDefContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *TypeDefDefContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *TypeDefDefContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.EnterTypeDefDef(s)
	}
}

func (s *TypeDefDefContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.ExitTypeDefDef(s)
	}
}

func (p *OracleParser) TypeDefDef() (localctx ITypeDefDefContext) {
	localctx = NewTypeDefDefContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 68, OracleParserRULE_typeDefDef)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(576)
		p.Match(OracleParserIDENT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(578)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == OracleParserLT {
		{
			p.SetState(577)
			p.TypeParams()
		}

	}
	{
		p.SetState(580)
		p.TypeRef()
	}
	p.SetState(582)
	p.GetErrorHandler().Sync(p)

	if p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 82, p.GetParserRuleContext()) == 1 {
		{
			p.SetState(581)
			p.TypeDefBody()
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

// ITypeDefBodyContext is an interface to support dynamic dispatch.
type ITypeDefBodyContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	LBRACE() antlr.TerminalNode
	RBRACE() antlr.TerminalNode
	AllNl() []INlContext
	Nl(i int) INlContext
	AllDomain() []IDomainContext
	Domain(i int) IDomainContext

	// IsTypeDefBodyContext differentiates from other interfaces.
	IsTypeDefBodyContext()
}

type TypeDefBodyContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyTypeDefBodyContext() *TypeDefBodyContext {
	var p = new(TypeDefBodyContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_typeDefBody
	return p
}

func InitEmptyTypeDefBodyContext(p *TypeDefBodyContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_typeDefBody
}

func (*TypeDefBodyContext) IsTypeDefBodyContext() {}

func NewTypeDefBodyContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *TypeDefBodyContext {
	var p = new(TypeDefBodyContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = OracleParserRULE_typeDefBody

	return p
}

func (s *TypeDefBodyContext) GetParser() antlr.Parser { return s.parser }

func (s *TypeDefBodyContext) LBRACE() antlr.TerminalNode {
	return s.GetToken(OracleParserLBRACE, 0)
}

func (s *TypeDefBodyContext) RBRACE() antlr.TerminalNode {
	return s.GetToken(OracleParserRBRACE, 0)
}

func (s *TypeDefBodyContext) AllNl() []INlContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(INlContext); ok {
			len++
		}
	}

	tst := make([]INlContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(INlContext); ok {
			tst[i] = t.(INlContext)
			i++
		}
	}

	return tst
}

func (s *TypeDefBodyContext) Nl(i int) INlContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(INlContext); ok {
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

	return t.(INlContext)
}

func (s *TypeDefBodyContext) AllDomain() []IDomainContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IDomainContext); ok {
			len++
		}
	}

	tst := make([]IDomainContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IDomainContext); ok {
			tst[i] = t.(IDomainContext)
			i++
		}
	}

	return tst
}

func (s *TypeDefBodyContext) Domain(i int) IDomainContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IDomainContext); ok {
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

	return t.(IDomainContext)
}

func (s *TypeDefBodyContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *TypeDefBodyContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *TypeDefBodyContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.EnterTypeDefBody(s)
	}
}

func (s *TypeDefBodyContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.ExitTypeDefBody(s)
	}
}

func (p *OracleParser) TypeDefBody() (localctx ITypeDefBodyContext) {
	localctx = NewTypeDefBodyContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 70, OracleParserRULE_typeDefBody)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(587)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(584)
			p.Nl()
		}

		p.SetState(589)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(590)
		p.Match(OracleParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(594)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(591)
			p.Nl()
		}

		p.SetState(596)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	p.SetState(606)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserAT {
		{
			p.SetState(597)
			p.Domain()
		}
		p.SetState(601)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(598)
				p.Nl()
			}

			p.SetState(603)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}

		p.SetState(608)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(609)
		p.Match(OracleParserRBRACE)
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

// IUnionDefContext is an interface to support dynamic dispatch.
type IUnionDefContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AllIDENT() []antlr.TerminalNode
	IDENT(i int) antlr.TerminalNode
	UNION() antlr.TerminalNode
	LBRACE() antlr.TerminalNode
	UnionBody() IUnionBodyContext
	RBRACE() antlr.TerminalNode
	EXTENDS() antlr.TerminalNode
	TypeRefList() ITypeRefListContext
	AllNl() []INlContext
	Nl(i int) INlContext

	// IsUnionDefContext differentiates from other interfaces.
	IsUnionDefContext()
}

type UnionDefContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyUnionDefContext() *UnionDefContext {
	var p = new(UnionDefContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_unionDef
	return p
}

func InitEmptyUnionDefContext(p *UnionDefContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_unionDef
}

func (*UnionDefContext) IsUnionDefContext() {}

func NewUnionDefContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *UnionDefContext {
	var p = new(UnionDefContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = OracleParserRULE_unionDef

	return p
}

func (s *UnionDefContext) GetParser() antlr.Parser { return s.parser }

func (s *UnionDefContext) AllIDENT() []antlr.TerminalNode {
	return s.GetTokens(OracleParserIDENT)
}

func (s *UnionDefContext) IDENT(i int) antlr.TerminalNode {
	return s.GetToken(OracleParserIDENT, i)
}

func (s *UnionDefContext) UNION() antlr.TerminalNode {
	return s.GetToken(OracleParserUNION, 0)
}

func (s *UnionDefContext) LBRACE() antlr.TerminalNode {
	return s.GetToken(OracleParserLBRACE, 0)
}

func (s *UnionDefContext) UnionBody() IUnionBodyContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IUnionBodyContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IUnionBodyContext)
}

func (s *UnionDefContext) RBRACE() antlr.TerminalNode {
	return s.GetToken(OracleParserRBRACE, 0)
}

func (s *UnionDefContext) EXTENDS() antlr.TerminalNode {
	return s.GetToken(OracleParserEXTENDS, 0)
}

func (s *UnionDefContext) TypeRefList() ITypeRefListContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ITypeRefListContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ITypeRefListContext)
}

func (s *UnionDefContext) AllNl() []INlContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(INlContext); ok {
			len++
		}
	}

	tst := make([]INlContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(INlContext); ok {
			tst[i] = t.(INlContext)
			i++
		}
	}

	return tst
}

func (s *UnionDefContext) Nl(i int) INlContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(INlContext); ok {
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

	return t.(INlContext)
}

func (s *UnionDefContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *UnionDefContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *UnionDefContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.EnterUnionDef(s)
	}
}

func (s *UnionDefContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.ExitUnionDef(s)
	}
}

func (p *OracleParser) UnionDef() (localctx IUnionDefContext) {
	localctx = NewUnionDefContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 72, OracleParserRULE_unionDef)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(611)
		p.Match(OracleParserIDENT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(612)
		p.Match(OracleParserUNION)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(613)
		p.Match(OracleParserIDENT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(614)
		p.Match(OracleParserIDENT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(617)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == OracleParserEXTENDS {
		{
			p.SetState(615)
			p.Match(OracleParserEXTENDS)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(616)
			p.TypeRefList()
		}

	}
	p.SetState(622)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(619)
			p.Nl()
		}

		p.SetState(624)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(625)
		p.Match(OracleParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(629)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(626)
			p.Nl()
		}

		p.SetState(631)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(632)
		p.UnionBody()
	}
	{
		p.SetState(633)
		p.Match(OracleParserRBRACE)
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

// IUnionBodyContext is an interface to support dynamic dispatch.
type IUnionBodyContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	AllUnionVariant() []IUnionVariantContext
	UnionVariant(i int) IUnionVariantContext
	AllDomain() []IDomainContext
	Domain(i int) IDomainContext
	AllNl() []INlContext
	Nl(i int) INlContext

	// IsUnionBodyContext differentiates from other interfaces.
	IsUnionBodyContext()
}

type UnionBodyContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyUnionBodyContext() *UnionBodyContext {
	var p = new(UnionBodyContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_unionBody
	return p
}

func InitEmptyUnionBodyContext(p *UnionBodyContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_unionBody
}

func (*UnionBodyContext) IsUnionBodyContext() {}

func NewUnionBodyContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *UnionBodyContext {
	var p = new(UnionBodyContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = OracleParserRULE_unionBody

	return p
}

func (s *UnionBodyContext) GetParser() antlr.Parser { return s.parser }

func (s *UnionBodyContext) AllUnionVariant() []IUnionVariantContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IUnionVariantContext); ok {
			len++
		}
	}

	tst := make([]IUnionVariantContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IUnionVariantContext); ok {
			tst[i] = t.(IUnionVariantContext)
			i++
		}
	}

	return tst
}

func (s *UnionBodyContext) UnionVariant(i int) IUnionVariantContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IUnionVariantContext); ok {
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

	return t.(IUnionVariantContext)
}

func (s *UnionBodyContext) AllDomain() []IDomainContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IDomainContext); ok {
			len++
		}
	}

	tst := make([]IDomainContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IDomainContext); ok {
			tst[i] = t.(IDomainContext)
			i++
		}
	}

	return tst
}

func (s *UnionBodyContext) Domain(i int) IDomainContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IDomainContext); ok {
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

	return t.(IDomainContext)
}

func (s *UnionBodyContext) AllNl() []INlContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(INlContext); ok {
			len++
		}
	}

	tst := make([]INlContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(INlContext); ok {
			tst[i] = t.(INlContext)
			i++
		}
	}

	return tst
}

func (s *UnionBodyContext) Nl(i int) INlContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(INlContext); ok {
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

	return t.(INlContext)
}

func (s *UnionBodyContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *UnionBodyContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *UnionBodyContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.EnterUnionBody(s)
	}
}

func (s *UnionBodyContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.ExitUnionBody(s)
	}
}

func (p *OracleParser) UnionBody() (localctx IUnionBodyContext) {
	localctx = NewUnionBodyContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 74, OracleParserRULE_unionBody)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(647)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&33816830) != 0 {
		p.SetState(637)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}

		switch p.GetTokenStream().LA(1) {
		case OracleParserSTRUCT, OracleParserENUM, OracleParserIMPORT, OracleParserEXTENDS, OracleParserMAP, OracleParserACTION, OracleParserUNION, OracleParserIDENT:
			{
				p.SetState(635)
				p.UnionVariant()
			}

		case OracleParserAT:
			{
				p.SetState(636)
				p.Domain()
			}

		default:
			p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
			goto errorExit
		}
		p.SetState(642)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(639)
				p.Nl()
			}

			p.SetState(644)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}

		p.SetState(649)
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

// IUnionVariantContext is an interface to support dynamic dispatch.
type IUnionVariantContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	VariantName() IVariantNameContext
	TypeRef() ITypeRefContext
	UnionVariantBody() IUnionVariantBodyContext

	// IsUnionVariantContext differentiates from other interfaces.
	IsUnionVariantContext()
}

type UnionVariantContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyUnionVariantContext() *UnionVariantContext {
	var p = new(UnionVariantContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_unionVariant
	return p
}

func InitEmptyUnionVariantContext(p *UnionVariantContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_unionVariant
}

func (*UnionVariantContext) IsUnionVariantContext() {}

func NewUnionVariantContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *UnionVariantContext {
	var p = new(UnionVariantContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = OracleParserRULE_unionVariant

	return p
}

func (s *UnionVariantContext) GetParser() antlr.Parser { return s.parser }

func (s *UnionVariantContext) VariantName() IVariantNameContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IVariantNameContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IVariantNameContext)
}

func (s *UnionVariantContext) TypeRef() ITypeRefContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(ITypeRefContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(ITypeRefContext)
}

func (s *UnionVariantContext) UnionVariantBody() IUnionVariantBodyContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IUnionVariantBodyContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IUnionVariantBodyContext)
}

func (s *UnionVariantContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *UnionVariantContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *UnionVariantContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.EnterUnionVariant(s)
	}
}

func (s *UnionVariantContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.ExitUnionVariant(s)
	}
}

func (p *OracleParser) UnionVariant() (localctx IUnionVariantContext) {
	localctx = NewUnionVariantContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 76, OracleParserRULE_unionVariant)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(650)
		p.VariantName()
	}
	{
		p.SetState(651)
		p.TypeRef()
	}
	p.SetState(653)
	p.GetErrorHandler().Sync(p)

	if p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 93, p.GetParserRuleContext()) == 1 {
		{
			p.SetState(652)
			p.UnionVariantBody()
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

// IVariantNameContext is an interface to support dynamic dispatch.
type IVariantNameContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	IDENT() antlr.TerminalNode
	MAP() antlr.TerminalNode
	UNION() antlr.TerminalNode
	STRUCT() antlr.TerminalNode
	ENUM() antlr.TerminalNode
	EXTENDS() antlr.TerminalNode
	IMPORT() antlr.TerminalNode
	ACTION() antlr.TerminalNode

	// IsVariantNameContext differentiates from other interfaces.
	IsVariantNameContext()
}

type VariantNameContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyVariantNameContext() *VariantNameContext {
	var p = new(VariantNameContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_variantName
	return p
}

func InitEmptyVariantNameContext(p *VariantNameContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_variantName
}

func (*VariantNameContext) IsVariantNameContext() {}

func NewVariantNameContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *VariantNameContext {
	var p = new(VariantNameContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = OracleParserRULE_variantName

	return p
}

func (s *VariantNameContext) GetParser() antlr.Parser { return s.parser }

func (s *VariantNameContext) IDENT() antlr.TerminalNode {
	return s.GetToken(OracleParserIDENT, 0)
}

func (s *VariantNameContext) MAP() antlr.TerminalNode {
	return s.GetToken(OracleParserMAP, 0)
}

func (s *VariantNameContext) UNION() antlr.TerminalNode {
	return s.GetToken(OracleParserUNION, 0)
}

func (s *VariantNameContext) STRUCT() antlr.TerminalNode {
	return s.GetToken(OracleParserSTRUCT, 0)
}

func (s *VariantNameContext) ENUM() antlr.TerminalNode {
	return s.GetToken(OracleParserENUM, 0)
}

func (s *VariantNameContext) EXTENDS() antlr.TerminalNode {
	return s.GetToken(OracleParserEXTENDS, 0)
}

func (s *VariantNameContext) IMPORT() antlr.TerminalNode {
	return s.GetToken(OracleParserIMPORT, 0)
}

func (s *VariantNameContext) ACTION() antlr.TerminalNode {
	return s.GetToken(OracleParserACTION, 0)
}

func (s *VariantNameContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *VariantNameContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *VariantNameContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.EnterVariantName(s)
	}
}

func (s *VariantNameContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.ExitVariantName(s)
	}
}

func (p *OracleParser) VariantName() (localctx IVariantNameContext) {
	localctx = NewVariantNameContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 78, OracleParserRULE_variantName)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(655)
		_la = p.GetTokenStream().LA(1)

		if !((int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&33554686) != 0) {
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

// IUnionVariantBodyContext is an interface to support dynamic dispatch.
type IUnionVariantBodyContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	LBRACE() antlr.TerminalNode
	RBRACE() antlr.TerminalNode
	AllNl() []INlContext
	Nl(i int) INlContext
	AllDomain() []IDomainContext
	Domain(i int) IDomainContext

	// IsUnionVariantBodyContext differentiates from other interfaces.
	IsUnionVariantBodyContext()
}

type UnionVariantBodyContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyUnionVariantBodyContext() *UnionVariantBodyContext {
	var p = new(UnionVariantBodyContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_unionVariantBody
	return p
}

func InitEmptyUnionVariantBodyContext(p *UnionVariantBodyContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_unionVariantBody
}

func (*UnionVariantBodyContext) IsUnionVariantBodyContext() {}

func NewUnionVariantBodyContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *UnionVariantBodyContext {
	var p = new(UnionVariantBodyContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = OracleParserRULE_unionVariantBody

	return p
}

func (s *UnionVariantBodyContext) GetParser() antlr.Parser { return s.parser }

func (s *UnionVariantBodyContext) LBRACE() antlr.TerminalNode {
	return s.GetToken(OracleParserLBRACE, 0)
}

func (s *UnionVariantBodyContext) RBRACE() antlr.TerminalNode {
	return s.GetToken(OracleParserRBRACE, 0)
}

func (s *UnionVariantBodyContext) AllNl() []INlContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(INlContext); ok {
			len++
		}
	}

	tst := make([]INlContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(INlContext); ok {
			tst[i] = t.(INlContext)
			i++
		}
	}

	return tst
}

func (s *UnionVariantBodyContext) Nl(i int) INlContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(INlContext); ok {
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

	return t.(INlContext)
}

func (s *UnionVariantBodyContext) AllDomain() []IDomainContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IDomainContext); ok {
			len++
		}
	}

	tst := make([]IDomainContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IDomainContext); ok {
			tst[i] = t.(IDomainContext)
			i++
		}
	}

	return tst
}

func (s *UnionVariantBodyContext) Domain(i int) IDomainContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IDomainContext); ok {
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

	return t.(IDomainContext)
}

func (s *UnionVariantBodyContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *UnionVariantBodyContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *UnionVariantBodyContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.EnterUnionVariantBody(s)
	}
}

func (s *UnionVariantBodyContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.ExitUnionVariantBody(s)
	}
}

func (p *OracleParser) UnionVariantBody() (localctx IUnionVariantBodyContext) {
	localctx = NewUnionVariantBodyContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 80, OracleParserRULE_unionVariantBody)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(660)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(657)
			p.Nl()
		}

		p.SetState(662)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(663)
		p.Match(OracleParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(667)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(664)
			p.Nl()
		}

		p.SetState(669)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	p.SetState(679)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserAT {
		{
			p.SetState(670)
			p.Domain()
		}
		p.SetState(674)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(671)
				p.Nl()
			}

			p.SetState(676)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}

		p.SetState(681)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(682)
		p.Match(OracleParserRBRACE)
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
