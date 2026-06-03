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
		"'{'", "'}'", "'['", "']'", "'<'", "'>'", "','", "'?'", "'.'", "'='",
		"'@'", "'-'",
	}
	staticData.SymbolicNames = []string{
		"", "STRUCT", "ENUM", "IMPORT", "EXTENDS", "MAP", "ACTION", "LBRACE",
		"RBRACE", "LBRACKET", "RBRACKET", "LT", "GT", "COMMA", "QUESTION", "DOT",
		"EQUALS", "AT", "MINUS", "TRIPLE_STRING_LIT", "STRING_LIT", "FLOAT_LIT",
		"INT_LIT", "BOOL_LIT", "IDENT", "LINE_COMMENT", "BLOCK_COMMENT", "NEWLINE",
		"WS",
	}
	staticData.RuleNames = []string{
		"schema", "nl", "importStmt", "fileDomain", "definition", "structDef",
		"typeRefList", "aliasBody", "typeParams", "typeParam", "structBody",
		"fieldOmit", "domainOmit", "actionDef", "actionBody", "fieldDef", "fieldDefault",
		"arrayDefault", "inlineDomain", "fieldBody", "domain", "domainContent",
		"domainBlock", "typeRef", "arrayModifier", "mapType", "typeArgs", "typeModifiers",
		"qualifiedIdent", "expression", "expressionValue", "enumDef", "enumBody",
		"enumValue", "enumValueBody", "typeDefDef", "typeDefBody",
	}
	staticData.PredictionContextCache = antlr.NewPredictionContextCache()
	staticData.serializedATN = []int32{
		4, 1, 28, 614, 2, 0, 7, 0, 2, 1, 7, 1, 2, 2, 7, 2, 2, 3, 7, 3, 2, 4, 7,
		4, 2, 5, 7, 5, 2, 6, 7, 6, 2, 7, 7, 7, 2, 8, 7, 8, 2, 9, 7, 9, 2, 10, 7,
		10, 2, 11, 7, 11, 2, 12, 7, 12, 2, 13, 7, 13, 2, 14, 7, 14, 2, 15, 7, 15,
		2, 16, 7, 16, 2, 17, 7, 17, 2, 18, 7, 18, 2, 19, 7, 19, 2, 20, 7, 20, 2,
		21, 7, 21, 2, 22, 7, 22, 2, 23, 7, 23, 2, 24, 7, 24, 2, 25, 7, 25, 2, 26,
		7, 26, 2, 27, 7, 27, 2, 28, 7, 28, 2, 29, 7, 29, 2, 30, 7, 30, 2, 31, 7,
		31, 2, 32, 7, 32, 2, 33, 7, 33, 2, 34, 7, 34, 2, 35, 7, 35, 2, 36, 7, 36,
		1, 0, 5, 0, 76, 8, 0, 10, 0, 12, 0, 79, 9, 0, 1, 0, 1, 0, 5, 0, 83, 8,
		0, 10, 0, 12, 0, 86, 9, 0, 5, 0, 88, 8, 0, 10, 0, 12, 0, 91, 9, 0, 1, 0,
		1, 0, 5, 0, 95, 8, 0, 10, 0, 12, 0, 98, 9, 0, 5, 0, 100, 8, 0, 10, 0, 12,
		0, 103, 9, 0, 1, 0, 1, 0, 5, 0, 107, 8, 0, 10, 0, 12, 0, 110, 9, 0, 5,
		0, 112, 8, 0, 10, 0, 12, 0, 115, 9, 0, 1, 0, 1, 0, 1, 1, 1, 1, 1, 2, 1,
		2, 1, 2, 1, 3, 1, 3, 1, 3, 3, 3, 127, 8, 3, 1, 4, 1, 4, 1, 4, 3, 4, 132,
		8, 4, 1, 5, 1, 5, 1, 5, 3, 5, 137, 8, 5, 1, 5, 1, 5, 3, 5, 141, 8, 5, 1,
		5, 5, 5, 144, 8, 5, 10, 5, 12, 5, 147, 9, 5, 1, 5, 1, 5, 5, 5, 151, 8,
		5, 10, 5, 12, 5, 154, 9, 5, 1, 5, 1, 5, 1, 5, 1, 5, 1, 5, 3, 5, 161, 8,
		5, 1, 5, 1, 5, 1, 5, 3, 5, 166, 8, 5, 3, 5, 168, 8, 5, 1, 6, 1, 6, 1, 6,
		5, 6, 173, 8, 6, 10, 6, 12, 6, 176, 9, 6, 1, 6, 5, 6, 179, 8, 6, 10, 6,
		12, 6, 182, 9, 6, 1, 7, 5, 7, 185, 8, 7, 10, 7, 12, 7, 188, 9, 7, 1, 7,
		1, 7, 5, 7, 192, 8, 7, 10, 7, 12, 7, 195, 9, 7, 1, 7, 1, 7, 5, 7, 199,
		8, 7, 10, 7, 12, 7, 202, 9, 7, 5, 7, 204, 8, 7, 10, 7, 12, 7, 207, 9, 7,
		1, 7, 1, 7, 1, 8, 1, 8, 5, 8, 213, 8, 8, 10, 8, 12, 8, 216, 9, 8, 1, 8,
		1, 8, 1, 8, 5, 8, 221, 8, 8, 10, 8, 12, 8, 224, 9, 8, 1, 8, 5, 8, 227,
		8, 8, 10, 8, 12, 8, 230, 9, 8, 1, 8, 5, 8, 233, 8, 8, 10, 8, 12, 8, 236,
		9, 8, 1, 8, 1, 8, 1, 9, 1, 9, 3, 9, 242, 8, 9, 1, 9, 1, 9, 3, 9, 246, 8,
		9, 1, 9, 1, 9, 3, 9, 250, 8, 9, 1, 10, 1, 10, 1, 10, 1, 10, 3, 10, 256,
		8, 10, 1, 10, 5, 10, 259, 8, 10, 10, 10, 12, 10, 262, 9, 10, 5, 10, 264,
		8, 10, 10, 10, 12, 10, 267, 9, 10, 1, 11, 1, 11, 1, 11, 1, 12, 1, 12, 1,
		12, 1, 12, 1, 13, 1, 13, 1, 13, 5, 13, 279, 8, 13, 10, 13, 12, 13, 282,
		9, 13, 1, 13, 1, 13, 5, 13, 286, 8, 13, 10, 13, 12, 13, 289, 9, 13, 1,
		13, 1, 13, 1, 13, 1, 14, 1, 14, 3, 14, 296, 8, 14, 1, 14, 5, 14, 299, 8,
		14, 10, 14, 12, 14, 302, 9, 14, 5, 14, 304, 8, 14, 10, 14, 12, 14, 307,
		9, 14, 1, 15, 1, 15, 1, 15, 3, 15, 312, 8, 15, 1, 15, 1, 15, 3, 15, 316,
		8, 15, 1, 15, 1, 15, 5, 15, 320, 8, 15, 10, 15, 12, 15, 323, 9, 15, 1,
		15, 3, 15, 326, 8, 15, 1, 16, 1, 16, 3, 16, 330, 8, 16, 1, 17, 1, 17, 5,
		17, 334, 8, 17, 10, 17, 12, 17, 337, 9, 17, 1, 17, 1, 17, 1, 17, 5, 17,
		342, 8, 17, 10, 17, 12, 17, 345, 9, 17, 1, 17, 5, 17, 348, 8, 17, 10, 17,
		12, 17, 351, 9, 17, 1, 17, 5, 17, 354, 8, 17, 10, 17, 12, 17, 357, 9, 17,
		3, 17, 359, 8, 17, 1, 17, 1, 17, 1, 18, 1, 18, 1, 18, 3, 18, 366, 8, 18,
		1, 19, 5, 19, 369, 8, 19, 10, 19, 12, 19, 372, 9, 19, 1, 19, 1, 19, 5,
		19, 376, 8, 19, 10, 19, 12, 19, 379, 9, 19, 1, 19, 1, 19, 3, 19, 383, 8,
		19, 1, 19, 5, 19, 386, 8, 19, 10, 19, 12, 19, 389, 9, 19, 5, 19, 391, 8,
		19, 10, 19, 12, 19, 394, 9, 19, 1, 19, 1, 19, 1, 20, 1, 20, 1, 20, 3, 20,
		401, 8, 20, 1, 21, 1, 21, 3, 21, 405, 8, 21, 1, 22, 5, 22, 408, 8, 22,
		10, 22, 12, 22, 411, 9, 22, 1, 22, 1, 22, 5, 22, 415, 8, 22, 10, 22, 12,
		22, 418, 9, 22, 1, 22, 1, 22, 4, 22, 422, 8, 22, 11, 22, 12, 22, 423, 1,
		22, 1, 22, 5, 22, 428, 8, 22, 10, 22, 12, 22, 431, 9, 22, 3, 22, 433, 8,
		22, 1, 22, 5, 22, 436, 8, 22, 10, 22, 12, 22, 439, 9, 22, 1, 22, 1, 22,
		1, 23, 1, 23, 3, 23, 445, 8, 23, 1, 23, 1, 23, 3, 23, 449, 8, 23, 1, 23,
		3, 23, 452, 8, 23, 1, 23, 3, 23, 455, 8, 23, 3, 23, 457, 8, 23, 1, 24,
		1, 24, 1, 24, 1, 24, 1, 24, 3, 24, 464, 8, 24, 1, 25, 1, 25, 1, 25, 1,
		25, 1, 25, 1, 25, 1, 25, 1, 26, 1, 26, 1, 26, 1, 26, 5, 26, 477, 8, 26,
		10, 26, 12, 26, 480, 9, 26, 1, 26, 1, 26, 1, 27, 1, 27, 1, 27, 3, 27, 487,
		8, 27, 1, 28, 1, 28, 1, 28, 3, 28, 492, 8, 28, 1, 29, 1, 29, 5, 29, 496,
		8, 29, 10, 29, 12, 29, 499, 9, 29, 1, 30, 1, 30, 1, 30, 1, 30, 1, 30, 1,
		30, 3, 30, 507, 8, 30, 1, 31, 1, 31, 1, 31, 1, 31, 3, 31, 513, 8, 31, 1,
		31, 5, 31, 516, 8, 31, 10, 31, 12, 31, 519, 9, 31, 1, 31, 1, 31, 5, 31,
		523, 8, 31, 10, 31, 12, 31, 526, 9, 31, 1, 31, 1, 31, 1, 31, 1, 32, 1,
		32, 3, 32, 533, 8, 32, 1, 32, 5, 32, 536, 8, 32, 10, 32, 12, 32, 539, 9,
		32, 5, 32, 541, 8, 32, 10, 32, 12, 32, 544, 9, 32, 1, 33, 1, 33, 1, 33,
		1, 33, 3, 33, 550, 8, 33, 1, 34, 5, 34, 553, 8, 34, 10, 34, 12, 34, 556,
		9, 34, 1, 34, 1, 34, 5, 34, 560, 8, 34, 10, 34, 12, 34, 563, 9, 34, 1,
		34, 1, 34, 5, 34, 567, 8, 34, 10, 34, 12, 34, 570, 9, 34, 5, 34, 572, 8,
		34, 10, 34, 12, 34, 575, 9, 34, 1, 34, 1, 34, 1, 35, 1, 35, 3, 35, 581,
		8, 35, 1, 35, 1, 35, 3, 35, 585, 8, 35, 1, 36, 5, 36, 588, 8, 36, 10, 36,
		12, 36, 591, 9, 36, 1, 36, 1, 36, 5, 36, 595, 8, 36, 10, 36, 12, 36, 598,
		9, 36, 1, 36, 1, 36, 5, 36, 602, 8, 36, 10, 36, 12, 36, 605, 9, 36, 5,
		36, 607, 8, 36, 10, 36, 12, 36, 610, 9, 36, 1, 36, 1, 36, 1, 36, 0, 0,
		37, 0, 2, 4, 6, 8, 10, 12, 14, 16, 18, 20, 22, 24, 26, 28, 30, 32, 34,
		36, 38, 40, 42, 44, 46, 48, 50, 52, 54, 56, 58, 60, 62, 64, 66, 68, 70,
		72, 0, 1, 2, 0, 20, 20, 22, 22, 674, 0, 77, 1, 0, 0, 0, 2, 118, 1, 0, 0,
		0, 4, 120, 1, 0, 0, 0, 6, 123, 1, 0, 0, 0, 8, 131, 1, 0, 0, 0, 10, 167,
		1, 0, 0, 0, 12, 169, 1, 0, 0, 0, 14, 186, 1, 0, 0, 0, 16, 210, 1, 0, 0,
		0, 18, 239, 1, 0, 0, 0, 20, 265, 1, 0, 0, 0, 22, 268, 1, 0, 0, 0, 24, 271,
		1, 0, 0, 0, 26, 275, 1, 0, 0, 0, 28, 305, 1, 0, 0, 0, 30, 308, 1, 0, 0,
		0, 32, 329, 1, 0, 0, 0, 34, 331, 1, 0, 0, 0, 36, 362, 1, 0, 0, 0, 38, 370,
		1, 0, 0, 0, 40, 397, 1, 0, 0, 0, 42, 404, 1, 0, 0, 0, 44, 409, 1, 0, 0,
		0, 46, 456, 1, 0, 0, 0, 48, 463, 1, 0, 0, 0, 50, 465, 1, 0, 0, 0, 52, 472,
		1, 0, 0, 0, 54, 486, 1, 0, 0, 0, 56, 488, 1, 0, 0, 0, 58, 493, 1, 0, 0,
		0, 60, 506, 1, 0, 0, 0, 62, 508, 1, 0, 0, 0, 64, 542, 1, 0, 0, 0, 66, 545,
		1, 0, 0, 0, 68, 554, 1, 0, 0, 0, 70, 578, 1, 0, 0, 0, 72, 589, 1, 0, 0,
		0, 74, 76, 3, 2, 1, 0, 75, 74, 1, 0, 0, 0, 76, 79, 1, 0, 0, 0, 77, 75,
		1, 0, 0, 0, 77, 78, 1, 0, 0, 0, 78, 89, 1, 0, 0, 0, 79, 77, 1, 0, 0, 0,
		80, 84, 3, 4, 2, 0, 81, 83, 3, 2, 1, 0, 82, 81, 1, 0, 0, 0, 83, 86, 1,
		0, 0, 0, 84, 82, 1, 0, 0, 0, 84, 85, 1, 0, 0, 0, 85, 88, 1, 0, 0, 0, 86,
		84, 1, 0, 0, 0, 87, 80, 1, 0, 0, 0, 88, 91, 1, 0, 0, 0, 89, 87, 1, 0, 0,
		0, 89, 90, 1, 0, 0, 0, 90, 101, 1, 0, 0, 0, 91, 89, 1, 0, 0, 0, 92, 96,
		3, 6, 3, 0, 93, 95, 3, 2, 1, 0, 94, 93, 1, 0, 0, 0, 95, 98, 1, 0, 0, 0,
		96, 94, 1, 0, 0, 0, 96, 97, 1, 0, 0, 0, 97, 100, 1, 0, 0, 0, 98, 96, 1,
		0, 0, 0, 99, 92, 1, 0, 0, 0, 100, 103, 1, 0, 0, 0, 101, 99, 1, 0, 0, 0,
		101, 102, 1, 0, 0, 0, 102, 113, 1, 0, 0, 0, 103, 101, 1, 0, 0, 0, 104,
		108, 3, 8, 4, 0, 105, 107, 3, 2, 1, 0, 106, 105, 1, 0, 0, 0, 107, 110,
		1, 0, 0, 0, 108, 106, 1, 0, 0, 0, 108, 109, 1, 0, 0, 0, 109, 112, 1, 0,
		0, 0, 110, 108, 1, 0, 0, 0, 111, 104, 1, 0, 0, 0, 112, 115, 1, 0, 0, 0,
		113, 111, 1, 0, 0, 0, 113, 114, 1, 0, 0, 0, 114, 116, 1, 0, 0, 0, 115,
		113, 1, 0, 0, 0, 116, 117, 5, 0, 0, 1, 117, 1, 1, 0, 0, 0, 118, 119, 5,
		27, 0, 0, 119, 3, 1, 0, 0, 0, 120, 121, 5, 3, 0, 0, 121, 122, 5, 20, 0,
		0, 122, 5, 1, 0, 0, 0, 123, 124, 5, 17, 0, 0, 124, 126, 5, 24, 0, 0, 125,
		127, 3, 42, 21, 0, 126, 125, 1, 0, 0, 0, 126, 127, 1, 0, 0, 0, 127, 7,
		1, 0, 0, 0, 128, 132, 3, 10, 5, 0, 129, 132, 3, 62, 31, 0, 130, 132, 3,
		70, 35, 0, 131, 128, 1, 0, 0, 0, 131, 129, 1, 0, 0, 0, 131, 130, 1, 0,
		0, 0, 132, 9, 1, 0, 0, 0, 133, 134, 5, 24, 0, 0, 134, 136, 5, 1, 0, 0,
		135, 137, 3, 16, 8, 0, 136, 135, 1, 0, 0, 0, 136, 137, 1, 0, 0, 0, 137,
		140, 1, 0, 0, 0, 138, 139, 5, 4, 0, 0, 139, 141, 3, 12, 6, 0, 140, 138,
		1, 0, 0, 0, 140, 141, 1, 0, 0, 0, 141, 145, 1, 0, 0, 0, 142, 144, 3, 2,
		1, 0, 143, 142, 1, 0, 0, 0, 144, 147, 1, 0, 0, 0, 145, 143, 1, 0, 0, 0,
		145, 146, 1, 0, 0, 0, 146, 148, 1, 0, 0, 0, 147, 145, 1, 0, 0, 0, 148,
		152, 5, 7, 0, 0, 149, 151, 3, 2, 1, 0, 150, 149, 1, 0, 0, 0, 151, 154,
		1, 0, 0, 0, 152, 150, 1, 0, 0, 0, 152, 153, 1, 0, 0, 0, 153, 155, 1, 0,
		0, 0, 154, 152, 1, 0, 0, 0, 155, 156, 3, 20, 10, 0, 156, 157, 5, 8, 0,
		0, 157, 168, 1, 0, 0, 0, 158, 160, 5, 24, 0, 0, 159, 161, 3, 16, 8, 0,
		160, 159, 1, 0, 0, 0, 160, 161, 1, 0, 0, 0, 161, 162, 1, 0, 0, 0, 162,
		163, 5, 16, 0, 0, 163, 165, 3, 46, 23, 0, 164, 166, 3, 14, 7, 0, 165, 164,
		1, 0, 0, 0, 165, 166, 1, 0, 0, 0, 166, 168, 1, 0, 0, 0, 167, 133, 1, 0,
		0, 0, 167, 158, 1, 0, 0, 0, 168, 11, 1, 0, 0, 0, 169, 180, 3, 46, 23, 0,
		170, 174, 5, 13, 0, 0, 171, 173, 3, 2, 1, 0, 172, 171, 1, 0, 0, 0, 173,
		176, 1, 0, 0, 0, 174, 172, 1, 0, 0, 0, 174, 175, 1, 0, 0, 0, 175, 177,
		1, 0, 0, 0, 176, 174, 1, 0, 0, 0, 177, 179, 3, 46, 23, 0, 178, 170, 1,
		0, 0, 0, 179, 182, 1, 0, 0, 0, 180, 178, 1, 0, 0, 0, 180, 181, 1, 0, 0,
		0, 181, 13, 1, 0, 0, 0, 182, 180, 1, 0, 0, 0, 183, 185, 3, 2, 1, 0, 184,
		183, 1, 0, 0, 0, 185, 188, 1, 0, 0, 0, 186, 184, 1, 0, 0, 0, 186, 187,
		1, 0, 0, 0, 187, 189, 1, 0, 0, 0, 188, 186, 1, 0, 0, 0, 189, 193, 5, 7,
		0, 0, 190, 192, 3, 2, 1, 0, 191, 190, 1, 0, 0, 0, 192, 195, 1, 0, 0, 0,
		193, 191, 1, 0, 0, 0, 193, 194, 1, 0, 0, 0, 194, 205, 1, 0, 0, 0, 195,
		193, 1, 0, 0, 0, 196, 200, 3, 40, 20, 0, 197, 199, 3, 2, 1, 0, 198, 197,
		1, 0, 0, 0, 199, 202, 1, 0, 0, 0, 200, 198, 1, 0, 0, 0, 200, 201, 1, 0,
		0, 0, 201, 204, 1, 0, 0, 0, 202, 200, 1, 0, 0, 0, 203, 196, 1, 0, 0, 0,
		204, 207, 1, 0, 0, 0, 205, 203, 1, 0, 0, 0, 205, 206, 1, 0, 0, 0, 206,
		208, 1, 0, 0, 0, 207, 205, 1, 0, 0, 0, 208, 209, 5, 8, 0, 0, 209, 15, 1,
		0, 0, 0, 210, 214, 5, 11, 0, 0, 211, 213, 3, 2, 1, 0, 212, 211, 1, 0, 0,
		0, 213, 216, 1, 0, 0, 0, 214, 212, 1, 0, 0, 0, 214, 215, 1, 0, 0, 0, 215,
		217, 1, 0, 0, 0, 216, 214, 1, 0, 0, 0, 217, 228, 3, 18, 9, 0, 218, 222,
		5, 13, 0, 0, 219, 221, 3, 2, 1, 0, 220, 219, 1, 0, 0, 0, 221, 224, 1, 0,
		0, 0, 222, 220, 1, 0, 0, 0, 222, 223, 1, 0, 0, 0, 223, 225, 1, 0, 0, 0,
		224, 222, 1, 0, 0, 0, 225, 227, 3, 18, 9, 0, 226, 218, 1, 0, 0, 0, 227,
		230, 1, 0, 0, 0, 228, 226, 1, 0, 0, 0, 228, 229, 1, 0, 0, 0, 229, 234,
		1, 0, 0, 0, 230, 228, 1, 0, 0, 0, 231, 233, 3, 2, 1, 0, 232, 231, 1, 0,
		0, 0, 233, 236, 1, 0, 0, 0, 234, 232, 1, 0, 0, 0, 234, 235, 1, 0, 0, 0,
		235, 237, 1, 0, 0, 0, 236, 234, 1, 0, 0, 0, 237, 238, 5, 12, 0, 0, 238,
		17, 1, 0, 0, 0, 239, 241, 5, 24, 0, 0, 240, 242, 5, 14, 0, 0, 241, 240,
		1, 0, 0, 0, 241, 242, 1, 0, 0, 0, 242, 245, 1, 0, 0, 0, 243, 244, 5, 4,
		0, 0, 244, 246, 3, 46, 23, 0, 245, 243, 1, 0, 0, 0, 245, 246, 1, 0, 0,
		0, 246, 249, 1, 0, 0, 0, 247, 248, 5, 16, 0, 0, 248, 250, 3, 46, 23, 0,
		249, 247, 1, 0, 0, 0, 249, 250, 1, 0, 0, 0, 250, 19, 1, 0, 0, 0, 251, 256,
		3, 30, 15, 0, 252, 256, 3, 22, 11, 0, 253, 256, 3, 26, 13, 0, 254, 256,
		3, 40, 20, 0, 255, 251, 1, 0, 0, 0, 255, 252, 1, 0, 0, 0, 255, 253, 1,
		0, 0, 0, 255, 254, 1, 0, 0, 0, 256, 260, 1, 0, 0, 0, 257, 259, 3, 2, 1,
		0, 258, 257, 1, 0, 0, 0, 259, 262, 1, 0, 0, 0, 260, 258, 1, 0, 0, 0, 260,
		261, 1, 0, 0, 0, 261, 264, 1, 0, 0, 0, 262, 260, 1, 0, 0, 0, 263, 255,
		1, 0, 0, 0, 264, 267, 1, 0, 0, 0, 265, 263, 1, 0, 0, 0, 265, 266, 1, 0,
		0, 0, 266, 21, 1, 0, 0, 0, 267, 265, 1, 0, 0, 0, 268, 269, 5, 18, 0, 0,
		269, 270, 5, 24, 0, 0, 270, 23, 1, 0, 0, 0, 271, 272, 5, 18, 0, 0, 272,
		273, 5, 17, 0, 0, 273, 274, 5, 24, 0, 0, 274, 25, 1, 0, 0, 0, 275, 276,
		5, 6, 0, 0, 276, 280, 5, 24, 0, 0, 277, 279, 3, 2, 1, 0, 278, 277, 1, 0,
		0, 0, 279, 282, 1, 0, 0, 0, 280, 278, 1, 0, 0, 0, 280, 281, 1, 0, 0, 0,
		281, 283, 1, 0, 0, 0, 282, 280, 1, 0, 0, 0, 283, 287, 5, 7, 0, 0, 284,
		286, 3, 2, 1, 0, 285, 284, 1, 0, 0, 0, 286, 289, 1, 0, 0, 0, 287, 285,
		1, 0, 0, 0, 287, 288, 1, 0, 0, 0, 288, 290, 1, 0, 0, 0, 289, 287, 1, 0,
		0, 0, 290, 291, 3, 28, 14, 0, 291, 292, 5, 8, 0, 0, 292, 27, 1, 0, 0, 0,
		293, 296, 3, 30, 15, 0, 294, 296, 3, 40, 20, 0, 295, 293, 1, 0, 0, 0, 295,
		294, 1, 0, 0, 0, 296, 300, 1, 0, 0, 0, 297, 299, 3, 2, 1, 0, 298, 297,
		1, 0, 0, 0, 299, 302, 1, 0, 0, 0, 300, 298, 1, 0, 0, 0, 300, 301, 1, 0,
		0, 0, 301, 304, 1, 0, 0, 0, 302, 300, 1, 0, 0, 0, 303, 295, 1, 0, 0, 0,
		304, 307, 1, 0, 0, 0, 305, 303, 1, 0, 0, 0, 305, 306, 1, 0, 0, 0, 306,
		29, 1, 0, 0, 0, 307, 305, 1, 0, 0, 0, 308, 311, 5, 24, 0, 0, 309, 312,
		3, 46, 23, 0, 310, 312, 3, 54, 27, 0, 311, 309, 1, 0, 0, 0, 311, 310, 1,
		0, 0, 0, 311, 312, 1, 0, 0, 0, 312, 315, 1, 0, 0, 0, 313, 314, 5, 16, 0,
		0, 314, 316, 3, 32, 16, 0, 315, 313, 1, 0, 0, 0, 315, 316, 1, 0, 0, 0,
		316, 321, 1, 0, 0, 0, 317, 320, 3, 36, 18, 0, 318, 320, 3, 24, 12, 0, 319,
		317, 1, 0, 0, 0, 319, 318, 1, 0, 0, 0, 320, 323, 1, 0, 0, 0, 321, 319,
		1, 0, 0, 0, 321, 322, 1, 0, 0, 0, 322, 325, 1, 0, 0, 0, 323, 321, 1, 0,
		0, 0, 324, 326, 3, 38, 19, 0, 325, 324, 1, 0, 0, 0, 325, 326, 1, 0, 0,
		0, 326, 31, 1, 0, 0, 0, 327, 330, 3, 60, 30, 0, 328, 330, 3, 34, 17, 0,
		329, 327, 1, 0, 0, 0, 329, 328, 1, 0, 0, 0, 330, 33, 1, 0, 0, 0, 331, 335,
		5, 9, 0, 0, 332, 334, 3, 2, 1, 0, 333, 332, 1, 0, 0, 0, 334, 337, 1, 0,
		0, 0, 335, 333, 1, 0, 0, 0, 335, 336, 1, 0, 0, 0, 336, 358, 1, 0, 0, 0,
		337, 335, 1, 0, 0, 0, 338, 349, 3, 60, 30, 0, 339, 343, 5, 13, 0, 0, 340,
		342, 3, 2, 1, 0, 341, 340, 1, 0, 0, 0, 342, 345, 1, 0, 0, 0, 343, 341,
		1, 0, 0, 0, 343, 344, 1, 0, 0, 0, 344, 346, 1, 0, 0, 0, 345, 343, 1, 0,
		0, 0, 346, 348, 3, 60, 30, 0, 347, 339, 1, 0, 0, 0, 348, 351, 1, 0, 0,
		0, 349, 347, 1, 0, 0, 0, 349, 350, 1, 0, 0, 0, 350, 355, 1, 0, 0, 0, 351,
		349, 1, 0, 0, 0, 352, 354, 3, 2, 1, 0, 353, 352, 1, 0, 0, 0, 354, 357,
		1, 0, 0, 0, 355, 353, 1, 0, 0, 0, 355, 356, 1, 0, 0, 0, 356, 359, 1, 0,
		0, 0, 357, 355, 1, 0, 0, 0, 358, 338, 1, 0, 0, 0, 358, 359, 1, 0, 0, 0,
		359, 360, 1, 0, 0, 0, 360, 361, 5, 10, 0, 0, 361, 35, 1, 0, 0, 0, 362,
		363, 5, 17, 0, 0, 363, 365, 5, 24, 0, 0, 364, 366, 3, 42, 21, 0, 365, 364,
		1, 0, 0, 0, 365, 366, 1, 0, 0, 0, 366, 37, 1, 0, 0, 0, 367, 369, 3, 2,
		1, 0, 368, 367, 1, 0, 0, 0, 369, 372, 1, 0, 0, 0, 370, 368, 1, 0, 0, 0,
		370, 371, 1, 0, 0, 0, 371, 373, 1, 0, 0, 0, 372, 370, 1, 0, 0, 0, 373,
		377, 5, 7, 0, 0, 374, 376, 3, 2, 1, 0, 375, 374, 1, 0, 0, 0, 376, 379,
		1, 0, 0, 0, 377, 375, 1, 0, 0, 0, 377, 378, 1, 0, 0, 0, 378, 392, 1, 0,
		0, 0, 379, 377, 1, 0, 0, 0, 380, 383, 3, 40, 20, 0, 381, 383, 3, 24, 12,
		0, 382, 380, 1, 0, 0, 0, 382, 381, 1, 0, 0, 0, 383, 387, 1, 0, 0, 0, 384,
		386, 3, 2, 1, 0, 385, 384, 1, 0, 0, 0, 386, 389, 1, 0, 0, 0, 387, 385,
		1, 0, 0, 0, 387, 388, 1, 0, 0, 0, 388, 391, 1, 0, 0, 0, 389, 387, 1, 0,
		0, 0, 390, 382, 1, 0, 0, 0, 391, 394, 1, 0, 0, 0, 392, 390, 1, 0, 0, 0,
		392, 393, 1, 0, 0, 0, 393, 395, 1, 0, 0, 0, 394, 392, 1, 0, 0, 0, 395,
		396, 5, 8, 0, 0, 396, 39, 1, 0, 0, 0, 397, 398, 5, 17, 0, 0, 398, 400,
		5, 24, 0, 0, 399, 401, 3, 42, 21, 0, 400, 399, 1, 0, 0, 0, 400, 401, 1,
		0, 0, 0, 401, 41, 1, 0, 0, 0, 402, 405, 3, 44, 22, 0, 403, 405, 3, 58,
		29, 0, 404, 402, 1, 0, 0, 0, 404, 403, 1, 0, 0, 0, 405, 43, 1, 0, 0, 0,
		406, 408, 3, 2, 1, 0, 407, 406, 1, 0, 0, 0, 408, 411, 1, 0, 0, 0, 409,
		407, 1, 0, 0, 0, 409, 410, 1, 0, 0, 0, 410, 412, 1, 0, 0, 0, 411, 409,
		1, 0, 0, 0, 412, 416, 5, 7, 0, 0, 413, 415, 3, 2, 1, 0, 414, 413, 1, 0,
		0, 0, 415, 418, 1, 0, 0, 0, 416, 414, 1, 0, 0, 0, 416, 417, 1, 0, 0, 0,
		417, 432, 1, 0, 0, 0, 418, 416, 1, 0, 0, 0, 419, 429, 3, 58, 29, 0, 420,
		422, 3, 2, 1, 0, 421, 420, 1, 0, 0, 0, 422, 423, 1, 0, 0, 0, 423, 421,
		1, 0, 0, 0, 423, 424, 1, 0, 0, 0, 424, 425, 1, 0, 0, 0, 425, 426, 3, 58,
		29, 0, 426, 428, 1, 0, 0, 0, 427, 421, 1, 0, 0, 0, 428, 431, 1, 0, 0, 0,
		429, 427, 1, 0, 0, 0, 429, 430, 1, 0, 0, 0, 430, 433, 1, 0, 0, 0, 431,
		429, 1, 0, 0, 0, 432, 419, 1, 0, 0, 0, 432, 433, 1, 0, 0, 0, 433, 437,
		1, 0, 0, 0, 434, 436, 3, 2, 1, 0, 435, 434, 1, 0, 0, 0, 436, 439, 1, 0,
		0, 0, 437, 435, 1, 0, 0, 0, 437, 438, 1, 0, 0, 0, 438, 440, 1, 0, 0, 0,
		439, 437, 1, 0, 0, 0, 440, 441, 5, 8, 0, 0, 441, 45, 1, 0, 0, 0, 442, 444,
		3, 50, 25, 0, 443, 445, 3, 54, 27, 0, 444, 443, 1, 0, 0, 0, 444, 445, 1,
		0, 0, 0, 445, 457, 1, 0, 0, 0, 446, 448, 3, 56, 28, 0, 447, 449, 3, 52,
		26, 0, 448, 447, 1, 0, 0, 0, 448, 449, 1, 0, 0, 0, 449, 451, 1, 0, 0, 0,
		450, 452, 3, 48, 24, 0, 451, 450, 1, 0, 0, 0, 451, 452, 1, 0, 0, 0, 452,
		454, 1, 0, 0, 0, 453, 455, 3, 54, 27, 0, 454, 453, 1, 0, 0, 0, 454, 455,
		1, 0, 0, 0, 455, 457, 1, 0, 0, 0, 456, 442, 1, 0, 0, 0, 456, 446, 1, 0,
		0, 0, 457, 47, 1, 0, 0, 0, 458, 459, 5, 9, 0, 0, 459, 464, 5, 10, 0, 0,
		460, 461, 5, 9, 0, 0, 461, 462, 5, 22, 0, 0, 462, 464, 5, 10, 0, 0, 463,
		458, 1, 0, 0, 0, 463, 460, 1, 0, 0, 0, 464, 49, 1, 0, 0, 0, 465, 466, 5,
		5, 0, 0, 466, 467, 5, 11, 0, 0, 467, 468, 3, 46, 23, 0, 468, 469, 5, 13,
		0, 0, 469, 470, 3, 46, 23, 0, 470, 471, 5, 12, 0, 0, 471, 51, 1, 0, 0,
		0, 472, 473, 5, 11, 0, 0, 473, 478, 3, 46, 23, 0, 474, 475, 5, 13, 0, 0,
		475, 477, 3, 46, 23, 0, 476, 474, 1, 0, 0, 0, 477, 480, 1, 0, 0, 0, 478,
		476, 1, 0, 0, 0, 478, 479, 1, 0, 0, 0, 479, 481, 1, 0, 0, 0, 480, 478,
		1, 0, 0, 0, 481, 482, 5, 12, 0, 0, 482, 53, 1, 0, 0, 0, 483, 484, 5, 14,
		0, 0, 484, 487, 5, 14, 0, 0, 485, 487, 5, 14, 0, 0, 486, 483, 1, 0, 0,
		0, 486, 485, 1, 0, 0, 0, 487, 55, 1, 0, 0, 0, 488, 491, 5, 24, 0, 0, 489,
		490, 5, 15, 0, 0, 490, 492, 5, 24, 0, 0, 491, 489, 1, 0, 0, 0, 491, 492,
		1, 0, 0, 0, 492, 57, 1, 0, 0, 0, 493, 497, 5, 24, 0, 0, 494, 496, 3, 60,
		30, 0, 495, 494, 1, 0, 0, 0, 496, 499, 1, 0, 0, 0, 497, 495, 1, 0, 0, 0,
		497, 498, 1, 0, 0, 0, 498, 59, 1, 0, 0, 0, 499, 497, 1, 0, 0, 0, 500, 507,
		5, 19, 0, 0, 501, 507, 5, 20, 0, 0, 502, 507, 5, 22, 0, 0, 503, 507, 5,
		21, 0, 0, 504, 507, 5, 23, 0, 0, 505, 507, 3, 56, 28, 0, 506, 500, 1, 0,
		0, 0, 506, 501, 1, 0, 0, 0, 506, 502, 1, 0, 0, 0, 506, 503, 1, 0, 0, 0,
		506, 504, 1, 0, 0, 0, 506, 505, 1, 0, 0, 0, 507, 61, 1, 0, 0, 0, 508, 509,
		5, 24, 0, 0, 509, 512, 5, 2, 0, 0, 510, 511, 5, 4, 0, 0, 511, 513, 3, 12,
		6, 0, 512, 510, 1, 0, 0, 0, 512, 513, 1, 0, 0, 0, 513, 517, 1, 0, 0, 0,
		514, 516, 3, 2, 1, 0, 515, 514, 1, 0, 0, 0, 516, 519, 1, 0, 0, 0, 517,
		515, 1, 0, 0, 0, 517, 518, 1, 0, 0, 0, 518, 520, 1, 0, 0, 0, 519, 517,
		1, 0, 0, 0, 520, 524, 5, 7, 0, 0, 521, 523, 3, 2, 1, 0, 522, 521, 1, 0,
		0, 0, 523, 526, 1, 0, 0, 0, 524, 522, 1, 0, 0, 0, 524, 525, 1, 0, 0, 0,
		525, 527, 1, 0, 0, 0, 526, 524, 1, 0, 0, 0, 527, 528, 3, 64, 32, 0, 528,
		529, 5, 8, 0, 0, 529, 63, 1, 0, 0, 0, 530, 533, 3, 66, 33, 0, 531, 533,
		3, 40, 20, 0, 532, 530, 1, 0, 0, 0, 532, 531, 1, 0, 0, 0, 533, 537, 1,
		0, 0, 0, 534, 536, 3, 2, 1, 0, 535, 534, 1, 0, 0, 0, 536, 539, 1, 0, 0,
		0, 537, 535, 1, 0, 0, 0, 537, 538, 1, 0, 0, 0, 538, 541, 1, 0, 0, 0, 539,
		537, 1, 0, 0, 0, 540, 532, 1, 0, 0, 0, 541, 544, 1, 0, 0, 0, 542, 540,
		1, 0, 0, 0, 542, 543, 1, 0, 0, 0, 543, 65, 1, 0, 0, 0, 544, 542, 1, 0,
		0, 0, 545, 546, 5, 24, 0, 0, 546, 547, 5, 16, 0, 0, 547, 549, 7, 0, 0,
		0, 548, 550, 3, 68, 34, 0, 549, 548, 1, 0, 0, 0, 549, 550, 1, 0, 0, 0,
		550, 67, 1, 0, 0, 0, 551, 553, 3, 2, 1, 0, 552, 551, 1, 0, 0, 0, 553, 556,
		1, 0, 0, 0, 554, 552, 1, 0, 0, 0, 554, 555, 1, 0, 0, 0, 555, 557, 1, 0,
		0, 0, 556, 554, 1, 0, 0, 0, 557, 561, 5, 7, 0, 0, 558, 560, 3, 2, 1, 0,
		559, 558, 1, 0, 0, 0, 560, 563, 1, 0, 0, 0, 561, 559, 1, 0, 0, 0, 561,
		562, 1, 0, 0, 0, 562, 573, 1, 0, 0, 0, 563, 561, 1, 0, 0, 0, 564, 568,
		3, 40, 20, 0, 565, 567, 3, 2, 1, 0, 566, 565, 1, 0, 0, 0, 567, 570, 1,
		0, 0, 0, 568, 566, 1, 0, 0, 0, 568, 569, 1, 0, 0, 0, 569, 572, 1, 0, 0,
		0, 570, 568, 1, 0, 0, 0, 571, 564, 1, 0, 0, 0, 572, 575, 1, 0, 0, 0, 573,
		571, 1, 0, 0, 0, 573, 574, 1, 0, 0, 0, 574, 576, 1, 0, 0, 0, 575, 573,
		1, 0, 0, 0, 576, 577, 5, 8, 0, 0, 577, 69, 1, 0, 0, 0, 578, 580, 5, 24,
		0, 0, 579, 581, 3, 16, 8, 0, 580, 579, 1, 0, 0, 0, 580, 581, 1, 0, 0, 0,
		581, 582, 1, 0, 0, 0, 582, 584, 3, 46, 23, 0, 583, 585, 3, 72, 36, 0, 584,
		583, 1, 0, 0, 0, 584, 585, 1, 0, 0, 0, 585, 71, 1, 0, 0, 0, 586, 588, 3,
		2, 1, 0, 587, 586, 1, 0, 0, 0, 588, 591, 1, 0, 0, 0, 589, 587, 1, 0, 0,
		0, 589, 590, 1, 0, 0, 0, 590, 592, 1, 0, 0, 0, 591, 589, 1, 0, 0, 0, 592,
		596, 5, 7, 0, 0, 593, 595, 3, 2, 1, 0, 594, 593, 1, 0, 0, 0, 595, 598,
		1, 0, 0, 0, 596, 594, 1, 0, 0, 0, 596, 597, 1, 0, 0, 0, 597, 608, 1, 0,
		0, 0, 598, 596, 1, 0, 0, 0, 599, 603, 3, 40, 20, 0, 600, 602, 3, 2, 1,
		0, 601, 600, 1, 0, 0, 0, 602, 605, 1, 0, 0, 0, 603, 601, 1, 0, 0, 0, 603,
		604, 1, 0, 0, 0, 604, 607, 1, 0, 0, 0, 605, 603, 1, 0, 0, 0, 606, 599,
		1, 0, 0, 0, 607, 610, 1, 0, 0, 0, 608, 606, 1, 0, 0, 0, 608, 609, 1, 0,
		0, 0, 609, 611, 1, 0, 0, 0, 610, 608, 1, 0, 0, 0, 611, 612, 5, 8, 0, 0,
		612, 73, 1, 0, 0, 0, 90, 77, 84, 89, 96, 101, 108, 113, 126, 131, 136,
		140, 145, 152, 160, 165, 167, 174, 180, 186, 193, 200, 205, 214, 222, 228,
		234, 241, 245, 249, 255, 260, 265, 280, 287, 295, 300, 305, 311, 315, 319,
		321, 325, 329, 335, 343, 349, 355, 358, 365, 370, 377, 382, 387, 392, 400,
		404, 409, 416, 423, 429, 432, 437, 444, 448, 451, 454, 456, 463, 478, 486,
		491, 497, 506, 512, 517, 524, 532, 537, 542, 549, 554, 561, 568, 573, 580,
		584, 589, 596, 603, 608,
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
	OracleParserLBRACE            = 7
	OracleParserRBRACE            = 8
	OracleParserLBRACKET          = 9
	OracleParserRBRACKET          = 10
	OracleParserLT                = 11
	OracleParserGT                = 12
	OracleParserCOMMA             = 13
	OracleParserQUESTION          = 14
	OracleParserDOT               = 15
	OracleParserEQUALS            = 16
	OracleParserAT                = 17
	OracleParserMINUS             = 18
	OracleParserTRIPLE_STRING_LIT = 19
	OracleParserSTRING_LIT        = 20
	OracleParserFLOAT_LIT         = 21
	OracleParserINT_LIT           = 22
	OracleParserBOOL_LIT          = 23
	OracleParserIDENT             = 24
	OracleParserLINE_COMMENT      = 25
	OracleParserBLOCK_COMMENT     = 26
	OracleParserNEWLINE           = 27
	OracleParserWS                = 28
)

// OracleParser rules.
const (
	OracleParserRULE_schema          = 0
	OracleParserRULE_nl              = 1
	OracleParserRULE_importStmt      = 2
	OracleParserRULE_fileDomain      = 3
	OracleParserRULE_definition      = 4
	OracleParserRULE_structDef       = 5
	OracleParserRULE_typeRefList     = 6
	OracleParserRULE_aliasBody       = 7
	OracleParserRULE_typeParams      = 8
	OracleParserRULE_typeParam       = 9
	OracleParserRULE_structBody      = 10
	OracleParserRULE_fieldOmit       = 11
	OracleParserRULE_domainOmit      = 12
	OracleParserRULE_actionDef       = 13
	OracleParserRULE_actionBody      = 14
	OracleParserRULE_fieldDef        = 15
	OracleParserRULE_fieldDefault    = 16
	OracleParserRULE_arrayDefault    = 17
	OracleParserRULE_inlineDomain    = 18
	OracleParserRULE_fieldBody       = 19
	OracleParserRULE_domain          = 20
	OracleParserRULE_domainContent   = 21
	OracleParserRULE_domainBlock     = 22
	OracleParserRULE_typeRef         = 23
	OracleParserRULE_arrayModifier   = 24
	OracleParserRULE_mapType         = 25
	OracleParserRULE_typeArgs        = 26
	OracleParserRULE_typeModifiers   = 27
	OracleParserRULE_qualifiedIdent  = 28
	OracleParserRULE_expression      = 29
	OracleParserRULE_expressionValue = 30
	OracleParserRULE_enumDef         = 31
	OracleParserRULE_enumBody        = 32
	OracleParserRULE_enumValue       = 33
	OracleParserRULE_enumValueBody   = 34
	OracleParserRULE_typeDefDef      = 35
	OracleParserRULE_typeDefBody     = 36
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
	p.SetState(77)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(74)
			p.Nl()
		}

		p.SetState(79)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	p.SetState(89)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserIMPORT {
		{
			p.SetState(80)
			p.ImportStmt()
		}
		p.SetState(84)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(81)
				p.Nl()
			}

			p.SetState(86)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}

		p.SetState(91)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	p.SetState(101)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserAT {
		{
			p.SetState(92)
			p.FileDomain()
		}
		p.SetState(96)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(93)
				p.Nl()
			}

			p.SetState(98)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}

		p.SetState(103)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	p.SetState(113)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserIDENT {
		{
			p.SetState(104)
			p.Definition()
		}
		p.SetState(108)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(105)
				p.Nl()
			}

			p.SetState(110)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}

		p.SetState(115)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(116)
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
		p.SetState(118)
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
		p.SetState(120)
		p.Match(OracleParserIMPORT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(121)
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
		p.SetState(123)
		p.Match(OracleParserAT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(124)
		p.Match(OracleParserIDENT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(126)
	p.GetErrorHandler().Sync(p)

	if p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 7, p.GetParserRuleContext()) == 1 {
		{
			p.SetState(125)
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
	p.SetState(131)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 8, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(128)
			p.StructDef()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(129)
			p.EnumDef()
		}

	case 3:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(130)
			p.TypeDefDef()
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

	p.SetState(167)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 15, p.GetParserRuleContext()) {
	case 1:
		localctx = NewStructFullContext(p, localctx)
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(133)
			p.Match(OracleParserIDENT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(134)
			p.Match(OracleParserSTRUCT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		p.SetState(136)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		if _la == OracleParserLT {
			{
				p.SetState(135)
				p.TypeParams()
			}

		}
		p.SetState(140)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		if _la == OracleParserEXTENDS {
			{
				p.SetState(138)
				p.Match(OracleParserEXTENDS)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}
			{
				p.SetState(139)
				p.TypeRefList()
			}

		}
		p.SetState(145)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(142)
				p.Nl()
			}

			p.SetState(147)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}
		{
			p.SetState(148)
			p.Match(OracleParserLBRACE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		p.SetState(152)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(149)
				p.Nl()
			}

			p.SetState(154)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}
		{
			p.SetState(155)
			p.StructBody()
		}
		{
			p.SetState(156)
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
			p.SetState(158)
			p.Match(OracleParserIDENT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		p.SetState(160)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		if _la == OracleParserLT {
			{
				p.SetState(159)
				p.TypeParams()
			}

		}
		{
			p.SetState(162)
			p.Match(OracleParserEQUALS)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(163)
			p.TypeRef()
		}
		p.SetState(165)
		p.GetErrorHandler().Sync(p)

		if p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 14, p.GetParserRuleContext()) == 1 {
			{
				p.SetState(164)
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
		p.SetState(169)
		p.TypeRef()
	}
	p.SetState(180)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserCOMMA {
		{
			p.SetState(170)
			p.Match(OracleParserCOMMA)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		p.SetState(174)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(171)
				p.Nl()
			}

			p.SetState(176)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}
		{
			p.SetState(177)
			p.TypeRef()
		}

		p.SetState(182)
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
	p.SetState(186)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(183)
			p.Nl()
		}

		p.SetState(188)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(189)
		p.Match(OracleParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(193)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(190)
			p.Nl()
		}

		p.SetState(195)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	p.SetState(205)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserAT {
		{
			p.SetState(196)
			p.Domain()
		}
		p.SetState(200)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(197)
				p.Nl()
			}

			p.SetState(202)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}

		p.SetState(207)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(208)
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
		p.SetState(210)
		p.Match(OracleParserLT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(214)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(211)
			p.Nl()
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
		p.TypeParam()
	}
	p.SetState(228)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserCOMMA {
		{
			p.SetState(218)
			p.Match(OracleParserCOMMA)
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

		for _la == OracleParserNEWLINE {
			{
				p.SetState(219)
				p.Nl()
			}

			p.SetState(224)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}
		{
			p.SetState(225)
			p.TypeParam()
		}

		p.SetState(230)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	p.SetState(234)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(231)
			p.Nl()
		}

		p.SetState(236)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(237)
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
		p.SetState(239)
		p.Match(OracleParserIDENT)
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

	if _la == OracleParserQUESTION {
		{
			p.SetState(240)
			p.Match(OracleParserQUESTION)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	}
	p.SetState(245)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == OracleParserEXTENDS {
		{
			p.SetState(243)
			p.Match(OracleParserEXTENDS)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(244)
			p.TypeRef()
		}

	}
	p.SetState(249)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == OracleParserEQUALS {
		{
			p.SetState(247)
			p.Match(OracleParserEQUALS)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(248)
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
	p.SetState(265)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&17170496) != 0 {
		p.SetState(255)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}

		switch p.GetTokenStream().LA(1) {
		case OracleParserIDENT:
			{
				p.SetState(251)
				p.FieldDef()
			}

		case OracleParserMINUS:
			{
				p.SetState(252)
				p.FieldOmit()
			}

		case OracleParserACTION:
			{
				p.SetState(253)
				p.ActionDef()
			}

		case OracleParserAT:
			{
				p.SetState(254)
				p.Domain()
			}

		default:
			p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
			goto errorExit
		}
		p.SetState(260)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(257)
				p.Nl()
			}

			p.SetState(262)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}

		p.SetState(267)
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
		p.SetState(268)
		p.Match(OracleParserMINUS)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(269)
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

// IDomainOmitContext is an interface to support dynamic dispatch.
type IDomainOmitContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	MINUS() antlr.TerminalNode
	AT() antlr.TerminalNode
	IDENT() antlr.TerminalNode

	// IsDomainOmitContext differentiates from other interfaces.
	IsDomainOmitContext()
}

type DomainOmitContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyDomainOmitContext() *DomainOmitContext {
	var p = new(DomainOmitContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_domainOmit
	return p
}

func InitEmptyDomainOmitContext(p *DomainOmitContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_domainOmit
}

func (*DomainOmitContext) IsDomainOmitContext() {}

func NewDomainOmitContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *DomainOmitContext {
	var p = new(DomainOmitContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = OracleParserRULE_domainOmit

	return p
}

func (s *DomainOmitContext) GetParser() antlr.Parser { return s.parser }

func (s *DomainOmitContext) MINUS() antlr.TerminalNode {
	return s.GetToken(OracleParserMINUS, 0)
}

func (s *DomainOmitContext) AT() antlr.TerminalNode {
	return s.GetToken(OracleParserAT, 0)
}

func (s *DomainOmitContext) IDENT() antlr.TerminalNode {
	return s.GetToken(OracleParserIDENT, 0)
}

func (s *DomainOmitContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *DomainOmitContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *DomainOmitContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.EnterDomainOmit(s)
	}
}

func (s *DomainOmitContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.ExitDomainOmit(s)
	}
}

func (p *OracleParser) DomainOmit() (localctx IDomainOmitContext) {
	localctx = NewDomainOmitContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 24, OracleParserRULE_domainOmit)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(271)
		p.Match(OracleParserMINUS)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(272)
		p.Match(OracleParserAT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(273)
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
	p.EnterRule(localctx, 26, OracleParserRULE_actionDef)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(275)
		p.Match(OracleParserACTION)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(276)
		p.Match(OracleParserIDENT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(280)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(277)
			p.Nl()
		}

		p.SetState(282)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(283)
		p.Match(OracleParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(287)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(284)
			p.Nl()
		}

		p.SetState(289)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(290)
		p.ActionBody()
	}
	{
		p.SetState(291)
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
	p.EnterRule(localctx, 28, OracleParserRULE_actionBody)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(305)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserAT || _la == OracleParserIDENT {
		p.SetState(295)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}

		switch p.GetTokenStream().LA(1) {
		case OracleParserIDENT:
			{
				p.SetState(293)
				p.FieldDef()
			}

		case OracleParserAT:
			{
				p.SetState(294)
				p.Domain()
			}

		default:
			p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
			goto errorExit
		}
		p.SetState(300)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(297)
				p.Nl()
			}

			p.SetState(302)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}

		p.SetState(307)
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
	TypeModifiers() ITypeModifiersContext
	EQUALS() antlr.TerminalNode
	FieldDefault() IFieldDefaultContext
	AllInlineDomain() []IInlineDomainContext
	InlineDomain(i int) IInlineDomainContext
	AllDomainOmit() []IDomainOmitContext
	DomainOmit(i int) IDomainOmitContext
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

func (s *FieldDefContext) TypeModifiers() ITypeModifiersContext {
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

func (s *FieldDefContext) AllDomainOmit() []IDomainOmitContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IDomainOmitContext); ok {
			len++
		}
	}

	tst := make([]IDomainOmitContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IDomainOmitContext); ok {
			tst[i] = t.(IDomainOmitContext)
			i++
		}
	}

	return tst
}

func (s *FieldDefContext) DomainOmit(i int) IDomainOmitContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IDomainOmitContext); ok {
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

	return t.(IDomainOmitContext)
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
	p.EnterRule(localctx, 30, OracleParserRULE_fieldDef)
	var _la int

	var _alt int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(308)
		p.Match(OracleParserIDENT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(311)
	p.GetErrorHandler().Sync(p)

	if p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 37, p.GetParserRuleContext()) == 1 {
		{
			p.SetState(309)
			p.TypeRef()
		}

	} else if p.HasError() { // JIM
		goto errorExit
	} else if p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 37, p.GetParserRuleContext()) == 2 {
		{
			p.SetState(310)
			p.TypeModifiers()
		}

	} else if p.HasError() { // JIM
		goto errorExit
	}
	p.SetState(315)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == OracleParserEQUALS {
		{
			p.SetState(313)
			p.Match(OracleParserEQUALS)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(314)
			p.FieldDefault()
		}

	}
	p.SetState(321)
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
			p.SetState(319)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}

			switch p.GetTokenStream().LA(1) {
			case OracleParserAT:
				{
					p.SetState(317)
					p.InlineDomain()
				}

			case OracleParserMINUS:
				{
					p.SetState(318)
					p.DomainOmit()
				}

			default:
				p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
				goto errorExit
			}

		}
		p.SetState(323)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 40, p.GetParserRuleContext())
		if p.HasError() {
			goto errorExit
		}
	}
	p.SetState(325)
	p.GetErrorHandler().Sync(p)

	if p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 41, p.GetParserRuleContext()) == 1 {
		{
			p.SetState(324)
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
	p.EnterRule(localctx, 32, OracleParserRULE_fieldDefault)
	p.SetState(329)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case OracleParserTRIPLE_STRING_LIT, OracleParserSTRING_LIT, OracleParserFLOAT_LIT, OracleParserINT_LIT, OracleParserBOOL_LIT, OracleParserIDENT:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(327)
			p.ExpressionValue()
		}

	case OracleParserLBRACKET:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(328)
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
	p.EnterRule(localctx, 34, OracleParserRULE_arrayDefault)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(331)
		p.Match(OracleParserLBRACKET)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(335)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(332)
			p.Nl()
		}

		p.SetState(337)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	p.SetState(358)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&33030144) != 0 {
		{
			p.SetState(338)
			p.ExpressionValue()
		}
		p.SetState(349)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserCOMMA {
			{
				p.SetState(339)
				p.Match(OracleParserCOMMA)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}
			p.SetState(343)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)

			for _la == OracleParserNEWLINE {
				{
					p.SetState(340)
					p.Nl()
				}

				p.SetState(345)
				p.GetErrorHandler().Sync(p)
				if p.HasError() {
					goto errorExit
				}
				_la = p.GetTokenStream().LA(1)
			}
			{
				p.SetState(346)
				p.ExpressionValue()
			}

			p.SetState(351)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}
		p.SetState(355)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(352)
				p.Nl()
			}

			p.SetState(357)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}

	}
	{
		p.SetState(360)
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
	p.EnterRule(localctx, 36, OracleParserRULE_inlineDomain)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(362)
		p.Match(OracleParserAT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(363)
		p.Match(OracleParserIDENT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(365)
	p.GetErrorHandler().Sync(p)

	if p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 48, p.GetParserRuleContext()) == 1 {
		{
			p.SetState(364)
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
	AllDomainOmit() []IDomainOmitContext
	DomainOmit(i int) IDomainOmitContext

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

func (s *FieldBodyContext) AllDomainOmit() []IDomainOmitContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IDomainOmitContext); ok {
			len++
		}
	}

	tst := make([]IDomainOmitContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IDomainOmitContext); ok {
			tst[i] = t.(IDomainOmitContext)
			i++
		}
	}

	return tst
}

func (s *FieldBodyContext) DomainOmit(i int) IDomainOmitContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IDomainOmitContext); ok {
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

	return t.(IDomainOmitContext)
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
	p.EnterRule(localctx, 38, OracleParserRULE_fieldBody)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(370)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(367)
			p.Nl()
		}

		p.SetState(372)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(373)
		p.Match(OracleParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(377)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(374)
			p.Nl()
		}

		p.SetState(379)
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

	for _la == OracleParserAT || _la == OracleParserMINUS {
		p.SetState(382)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}

		switch p.GetTokenStream().LA(1) {
		case OracleParserAT:
			{
				p.SetState(380)
				p.Domain()
			}

		case OracleParserMINUS:
			{
				p.SetState(381)
				p.DomainOmit()
			}

		default:
			p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
			goto errorExit
		}
		p.SetState(387)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(384)
				p.Nl()
			}

			p.SetState(389)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}

		p.SetState(394)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(395)
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
	p.EnterRule(localctx, 40, OracleParserRULE_domain)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(397)
		p.Match(OracleParserAT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(398)
		p.Match(OracleParserIDENT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(400)
	p.GetErrorHandler().Sync(p)

	if p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 54, p.GetParserRuleContext()) == 1 {
		{
			p.SetState(399)
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
	p.EnterRule(localctx, 42, OracleParserRULE_domainContent)
	p.SetState(404)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case OracleParserLBRACE, OracleParserNEWLINE:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(402)
			p.DomainBlock()
		}

	case OracleParserIDENT:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(403)
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
	p.EnterRule(localctx, 44, OracleParserRULE_domainBlock)
	var _la int

	var _alt int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(409)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(406)
			p.Nl()
		}

		p.SetState(411)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(412)
		p.Match(OracleParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(416)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 57, p.GetParserRuleContext())
	if p.HasError() {
		goto errorExit
	}
	for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
		if _alt == 1 {
			{
				p.SetState(413)
				p.Nl()
			}

		}
		p.SetState(418)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 57, p.GetParserRuleContext())
		if p.HasError() {
			goto errorExit
		}
	}
	p.SetState(432)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == OracleParserIDENT {
		{
			p.SetState(419)
			p.Expression()
		}
		p.SetState(429)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 59, p.GetParserRuleContext())
		if p.HasError() {
			goto errorExit
		}
		for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
			if _alt == 1 {
				p.SetState(421)
				p.GetErrorHandler().Sync(p)
				if p.HasError() {
					goto errorExit
				}
				_la = p.GetTokenStream().LA(1)

				for ok := true; ok; ok = _la == OracleParserNEWLINE {
					{
						p.SetState(420)
						p.Nl()
					}

					p.SetState(423)
					p.GetErrorHandler().Sync(p)
					if p.HasError() {
						goto errorExit
					}
					_la = p.GetTokenStream().LA(1)
				}
				{
					p.SetState(425)
					p.Expression()
				}

			}
			p.SetState(431)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 59, p.GetParserRuleContext())
			if p.HasError() {
				goto errorExit
			}
		}

	}
	p.SetState(437)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(434)
			p.Nl()
		}

		p.SetState(439)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(440)
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
	p.EnterRule(localctx, 46, OracleParserRULE_typeRef)
	var _la int

	p.SetState(456)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case OracleParserMAP:
		localctx = NewTypeRefMapContext(p, localctx)
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(442)
			p.MapType()
		}
		p.SetState(444)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		if _la == OracleParserQUESTION {
			{
				p.SetState(443)
				p.TypeModifiers()
			}

		}

	case OracleParserIDENT:
		localctx = NewTypeRefNormalContext(p, localctx)
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(446)
			p.QualifiedIdent()
		}
		p.SetState(448)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		if _la == OracleParserLT {
			{
				p.SetState(447)
				p.TypeArgs()
			}

		}
		p.SetState(451)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		if _la == OracleParserLBRACKET {
			{
				p.SetState(450)
				p.ArrayModifier()
			}

		}
		p.SetState(454)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		if _la == OracleParserQUESTION {
			{
				p.SetState(453)
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
	p.EnterRule(localctx, 48, OracleParserRULE_arrayModifier)
	p.SetState(463)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 67, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
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
			p.Match(OracleParserRBRACKET)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(460)
			p.Match(OracleParserLBRACKET)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(461)
			p.Match(OracleParserINT_LIT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(462)
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
	p.EnterRule(localctx, 50, OracleParserRULE_mapType)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(465)
		p.Match(OracleParserMAP)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(466)
		p.Match(OracleParserLT)
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
		p.Match(OracleParserCOMMA)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(469)
		p.TypeRef()
	}
	{
		p.SetState(470)
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
	p.EnterRule(localctx, 52, OracleParserRULE_typeArgs)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(472)
		p.Match(OracleParserLT)
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

	for _la == OracleParserCOMMA {
		{
			p.SetState(474)
			p.Match(OracleParserCOMMA)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(475)
			p.TypeRef()
		}

		p.SetState(480)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(481)
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
	p.EnterRule(localctx, 54, OracleParserRULE_typeModifiers)
	p.SetState(486)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 69, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(483)
			p.Match(OracleParserQUESTION)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(484)
			p.Match(OracleParserQUESTION)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(485)
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
	p.EnterRule(localctx, 56, OracleParserRULE_qualifiedIdent)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(488)
		p.Match(OracleParserIDENT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(491)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == OracleParserDOT {
		{
			p.SetState(489)
			p.Match(OracleParserDOT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(490)
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
	p.EnterRule(localctx, 58, OracleParserRULE_expression)
	var _alt int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(493)
		p.Match(OracleParserIDENT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(497)
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
			{
				p.SetState(494)
				p.ExpressionValue()
			}

		}
		p.SetState(499)
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
	p.EnterRule(localctx, 60, OracleParserRULE_expressionValue)
	p.SetState(506)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case OracleParserTRIPLE_STRING_LIT:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(500)
			p.Match(OracleParserTRIPLE_STRING_LIT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case OracleParserSTRING_LIT:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(501)
			p.Match(OracleParserSTRING_LIT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case OracleParserINT_LIT:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(502)
			p.Match(OracleParserINT_LIT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case OracleParserFLOAT_LIT:
		p.EnterOuterAlt(localctx, 4)
		{
			p.SetState(503)
			p.Match(OracleParserFLOAT_LIT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case OracleParserBOOL_LIT:
		p.EnterOuterAlt(localctx, 5)
		{
			p.SetState(504)
			p.Match(OracleParserBOOL_LIT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case OracleParserIDENT:
		p.EnterOuterAlt(localctx, 6)
		{
			p.SetState(505)
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
	p.EnterRule(localctx, 62, OracleParserRULE_enumDef)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(508)
		p.Match(OracleParserIDENT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(509)
		p.Match(OracleParserENUM)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(512)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == OracleParserEXTENDS {
		{
			p.SetState(510)
			p.Match(OracleParserEXTENDS)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(511)
			p.TypeRefList()
		}

	}
	p.SetState(517)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(514)
			p.Nl()
		}

		p.SetState(519)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(520)
		p.Match(OracleParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(524)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(521)
			p.Nl()
		}

		p.SetState(526)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(527)
		p.EnumBody()
	}
	{
		p.SetState(528)
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
	p.EnterRule(localctx, 64, OracleParserRULE_enumBody)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(542)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserAT || _la == OracleParserIDENT {
		p.SetState(532)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}

		switch p.GetTokenStream().LA(1) {
		case OracleParserIDENT:
			{
				p.SetState(530)
				p.EnumValue()
			}

		case OracleParserAT:
			{
				p.SetState(531)
				p.Domain()
			}

		default:
			p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
			goto errorExit
		}
		p.SetState(537)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(534)
				p.Nl()
			}

			p.SetState(539)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}

		p.SetState(544)
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
	p.EnterRule(localctx, 66, OracleParserRULE_enumValue)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(545)
		p.Match(OracleParserIDENT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(546)
		p.Match(OracleParserEQUALS)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(547)
		_la = p.GetTokenStream().LA(1)

		if !(_la == OracleParserSTRING_LIT || _la == OracleParserINT_LIT) {
			p.GetErrorHandler().RecoverInline(p)
		} else {
			p.GetErrorHandler().ReportMatch(p)
			p.Consume()
		}
	}
	p.SetState(549)
	p.GetErrorHandler().Sync(p)

	if p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 79, p.GetParserRuleContext()) == 1 {
		{
			p.SetState(548)
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
	p.EnterRule(localctx, 68, OracleParserRULE_enumValueBody)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(554)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(551)
			p.Nl()
		}

		p.SetState(556)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(557)
		p.Match(OracleParserLBRACE)
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

	for _la == OracleParserNEWLINE {
		{
			p.SetState(558)
			p.Nl()
		}

		p.SetState(563)
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

	for _la == OracleParserAT {
		{
			p.SetState(564)
			p.Domain()
		}
		p.SetState(568)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(565)
				p.Nl()
			}

			p.SetState(570)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}

		p.SetState(575)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(576)
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
	p.EnterRule(localctx, 70, OracleParserRULE_typeDefDef)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(578)
		p.Match(OracleParserIDENT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(580)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == OracleParserLT {
		{
			p.SetState(579)
			p.TypeParams()
		}

	}
	{
		p.SetState(582)
		p.TypeRef()
	}
	p.SetState(584)
	p.GetErrorHandler().Sync(p)

	if p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 85, p.GetParserRuleContext()) == 1 {
		{
			p.SetState(583)
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
	p.EnterRule(localctx, 72, OracleParserRULE_typeDefBody)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(589)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(586)
			p.Nl()
		}

		p.SetState(591)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(592)
		p.Match(OracleParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(596)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(593)
			p.Nl()
		}

		p.SetState(598)
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

	for _la == OracleParserAT {
		{
			p.SetState(599)
			p.Domain()
		}
		p.SetState(603)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(600)
				p.Nl()
			}

			p.SetState(605)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}

		p.SetState(610)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(611)
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
