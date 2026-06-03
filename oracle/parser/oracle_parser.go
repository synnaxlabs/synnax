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
		"fieldOmit", "actionDef", "actionBody", "fieldDef", "fieldDefault",
		"arrayDefault", "inlineDomain", "fieldBody", "domain", "domainContent",
		"domainBlock", "typeRef", "arrayModifier", "mapType", "typeArgs", "typeModifiers",
		"qualifiedIdent", "expression", "expressionValue", "enumDef", "enumBody",
		"enumValue", "enumValueBody", "typeDefDef", "typeDefBody",
	}
	staticData.PredictionContextCache = antlr.NewPredictionContextCache()
	staticData.serializedATN = []int32{
		4, 1, 28, 601, 2, 0, 7, 0, 2, 1, 7, 1, 2, 2, 7, 2, 2, 3, 7, 3, 2, 4, 7,
		4, 2, 5, 7, 5, 2, 6, 7, 6, 2, 7, 7, 7, 2, 8, 7, 8, 2, 9, 7, 9, 2, 10, 7,
		10, 2, 11, 7, 11, 2, 12, 7, 12, 2, 13, 7, 13, 2, 14, 7, 14, 2, 15, 7, 15,
		2, 16, 7, 16, 2, 17, 7, 17, 2, 18, 7, 18, 2, 19, 7, 19, 2, 20, 7, 20, 2,
		21, 7, 21, 2, 22, 7, 22, 2, 23, 7, 23, 2, 24, 7, 24, 2, 25, 7, 25, 2, 26,
		7, 26, 2, 27, 7, 27, 2, 28, 7, 28, 2, 29, 7, 29, 2, 30, 7, 30, 2, 31, 7,
		31, 2, 32, 7, 32, 2, 33, 7, 33, 2, 34, 7, 34, 2, 35, 7, 35, 1, 0, 5, 0,
		74, 8, 0, 10, 0, 12, 0, 77, 9, 0, 1, 0, 1, 0, 5, 0, 81, 8, 0, 10, 0, 12,
		0, 84, 9, 0, 5, 0, 86, 8, 0, 10, 0, 12, 0, 89, 9, 0, 1, 0, 1, 0, 5, 0,
		93, 8, 0, 10, 0, 12, 0, 96, 9, 0, 5, 0, 98, 8, 0, 10, 0, 12, 0, 101, 9,
		0, 1, 0, 1, 0, 5, 0, 105, 8, 0, 10, 0, 12, 0, 108, 9, 0, 5, 0, 110, 8,
		0, 10, 0, 12, 0, 113, 9, 0, 1, 0, 1, 0, 1, 1, 1, 1, 1, 2, 1, 2, 1, 2, 1,
		3, 1, 3, 1, 3, 3, 3, 125, 8, 3, 1, 4, 1, 4, 1, 4, 3, 4, 130, 8, 4, 1, 5,
		1, 5, 1, 5, 3, 5, 135, 8, 5, 1, 5, 1, 5, 3, 5, 139, 8, 5, 1, 5, 5, 5, 142,
		8, 5, 10, 5, 12, 5, 145, 9, 5, 1, 5, 1, 5, 5, 5, 149, 8, 5, 10, 5, 12,
		5, 152, 9, 5, 1, 5, 1, 5, 1, 5, 1, 5, 1, 5, 3, 5, 159, 8, 5, 1, 5, 1, 5,
		1, 5, 3, 5, 164, 8, 5, 3, 5, 166, 8, 5, 1, 6, 1, 6, 1, 6, 5, 6, 171, 8,
		6, 10, 6, 12, 6, 174, 9, 6, 1, 6, 5, 6, 177, 8, 6, 10, 6, 12, 6, 180, 9,
		6, 1, 7, 5, 7, 183, 8, 7, 10, 7, 12, 7, 186, 9, 7, 1, 7, 1, 7, 5, 7, 190,
		8, 7, 10, 7, 12, 7, 193, 9, 7, 1, 7, 1, 7, 5, 7, 197, 8, 7, 10, 7, 12,
		7, 200, 9, 7, 5, 7, 202, 8, 7, 10, 7, 12, 7, 205, 9, 7, 1, 7, 1, 7, 1,
		8, 1, 8, 5, 8, 211, 8, 8, 10, 8, 12, 8, 214, 9, 8, 1, 8, 1, 8, 1, 8, 5,
		8, 219, 8, 8, 10, 8, 12, 8, 222, 9, 8, 1, 8, 5, 8, 225, 8, 8, 10, 8, 12,
		8, 228, 9, 8, 1, 8, 5, 8, 231, 8, 8, 10, 8, 12, 8, 234, 9, 8, 1, 8, 1,
		8, 1, 9, 1, 9, 3, 9, 240, 8, 9, 1, 9, 1, 9, 3, 9, 244, 8, 9, 1, 9, 1, 9,
		3, 9, 248, 8, 9, 1, 10, 1, 10, 1, 10, 1, 10, 3, 10, 254, 8, 10, 1, 10,
		5, 10, 257, 8, 10, 10, 10, 12, 10, 260, 9, 10, 5, 10, 262, 8, 10, 10, 10,
		12, 10, 265, 9, 10, 1, 11, 1, 11, 1, 11, 1, 12, 1, 12, 1, 12, 5, 12, 273,
		8, 12, 10, 12, 12, 12, 276, 9, 12, 1, 12, 1, 12, 5, 12, 280, 8, 12, 10,
		12, 12, 12, 283, 9, 12, 1, 12, 1, 12, 1, 12, 1, 13, 1, 13, 3, 13, 290,
		8, 13, 1, 13, 5, 13, 293, 8, 13, 10, 13, 12, 13, 296, 9, 13, 5, 13, 298,
		8, 13, 10, 13, 12, 13, 301, 9, 13, 1, 14, 1, 14, 1, 14, 1, 14, 3, 14, 307,
		8, 14, 1, 14, 5, 14, 310, 8, 14, 10, 14, 12, 14, 313, 9, 14, 1, 14, 3,
		14, 316, 8, 14, 1, 15, 1, 15, 3, 15, 320, 8, 15, 1, 16, 1, 16, 5, 16, 324,
		8, 16, 10, 16, 12, 16, 327, 9, 16, 1, 16, 1, 16, 1, 16, 5, 16, 332, 8,
		16, 10, 16, 12, 16, 335, 9, 16, 1, 16, 5, 16, 338, 8, 16, 10, 16, 12, 16,
		341, 9, 16, 1, 16, 5, 16, 344, 8, 16, 10, 16, 12, 16, 347, 9, 16, 3, 16,
		349, 8, 16, 1, 16, 1, 16, 1, 17, 1, 17, 1, 17, 3, 17, 356, 8, 17, 1, 18,
		5, 18, 359, 8, 18, 10, 18, 12, 18, 362, 9, 18, 1, 18, 1, 18, 5, 18, 366,
		8, 18, 10, 18, 12, 18, 369, 9, 18, 1, 18, 1, 18, 5, 18, 373, 8, 18, 10,
		18, 12, 18, 376, 9, 18, 5, 18, 378, 8, 18, 10, 18, 12, 18, 381, 9, 18,
		1, 18, 1, 18, 1, 19, 1, 19, 1, 19, 3, 19, 388, 8, 19, 1, 20, 1, 20, 3,
		20, 392, 8, 20, 1, 21, 5, 21, 395, 8, 21, 10, 21, 12, 21, 398, 9, 21, 1,
		21, 1, 21, 5, 21, 402, 8, 21, 10, 21, 12, 21, 405, 9, 21, 1, 21, 1, 21,
		4, 21, 409, 8, 21, 11, 21, 12, 21, 410, 1, 21, 1, 21, 5, 21, 415, 8, 21,
		10, 21, 12, 21, 418, 9, 21, 3, 21, 420, 8, 21, 1, 21, 5, 21, 423, 8, 21,
		10, 21, 12, 21, 426, 9, 21, 1, 21, 1, 21, 1, 22, 1, 22, 3, 22, 432, 8,
		22, 1, 22, 1, 22, 3, 22, 436, 8, 22, 1, 22, 3, 22, 439, 8, 22, 1, 22, 3,
		22, 442, 8, 22, 3, 22, 444, 8, 22, 1, 23, 1, 23, 1, 23, 1, 23, 1, 23, 3,
		23, 451, 8, 23, 1, 24, 1, 24, 1, 24, 1, 24, 1, 24, 1, 24, 1, 24, 1, 25,
		1, 25, 1, 25, 1, 25, 5, 25, 464, 8, 25, 10, 25, 12, 25, 467, 9, 25, 1,
		25, 1, 25, 1, 26, 1, 26, 1, 26, 3, 26, 474, 8, 26, 1, 27, 1, 27, 1, 27,
		3, 27, 479, 8, 27, 1, 28, 1, 28, 5, 28, 483, 8, 28, 10, 28, 12, 28, 486,
		9, 28, 1, 29, 1, 29, 1, 29, 1, 29, 1, 29, 1, 29, 3, 29, 494, 8, 29, 1,
		30, 1, 30, 1, 30, 1, 30, 3, 30, 500, 8, 30, 1, 30, 5, 30, 503, 8, 30, 10,
		30, 12, 30, 506, 9, 30, 1, 30, 1, 30, 5, 30, 510, 8, 30, 10, 30, 12, 30,
		513, 9, 30, 1, 30, 1, 30, 1, 30, 1, 31, 1, 31, 3, 31, 520, 8, 31, 1, 31,
		5, 31, 523, 8, 31, 10, 31, 12, 31, 526, 9, 31, 5, 31, 528, 8, 31, 10, 31,
		12, 31, 531, 9, 31, 1, 32, 1, 32, 1, 32, 1, 32, 3, 32, 537, 8, 32, 1, 33,
		5, 33, 540, 8, 33, 10, 33, 12, 33, 543, 9, 33, 1, 33, 1, 33, 5, 33, 547,
		8, 33, 10, 33, 12, 33, 550, 9, 33, 1, 33, 1, 33, 5, 33, 554, 8, 33, 10,
		33, 12, 33, 557, 9, 33, 5, 33, 559, 8, 33, 10, 33, 12, 33, 562, 9, 33,
		1, 33, 1, 33, 1, 34, 1, 34, 3, 34, 568, 8, 34, 1, 34, 1, 34, 3, 34, 572,
		8, 34, 1, 35, 5, 35, 575, 8, 35, 10, 35, 12, 35, 578, 9, 35, 1, 35, 1,
		35, 5, 35, 582, 8, 35, 10, 35, 12, 35, 585, 9, 35, 1, 35, 1, 35, 5, 35,
		589, 8, 35, 10, 35, 12, 35, 592, 9, 35, 5, 35, 594, 8, 35, 10, 35, 12,
		35, 597, 9, 35, 1, 35, 1, 35, 1, 35, 0, 0, 36, 0, 2, 4, 6, 8, 10, 12, 14,
		16, 18, 20, 22, 24, 26, 28, 30, 32, 34, 36, 38, 40, 42, 44, 46, 48, 50,
		52, 54, 56, 58, 60, 62, 64, 66, 68, 70, 0, 1, 2, 0, 20, 20, 22, 22, 658,
		0, 75, 1, 0, 0, 0, 2, 116, 1, 0, 0, 0, 4, 118, 1, 0, 0, 0, 6, 121, 1, 0,
		0, 0, 8, 129, 1, 0, 0, 0, 10, 165, 1, 0, 0, 0, 12, 167, 1, 0, 0, 0, 14,
		184, 1, 0, 0, 0, 16, 208, 1, 0, 0, 0, 18, 237, 1, 0, 0, 0, 20, 263, 1,
		0, 0, 0, 22, 266, 1, 0, 0, 0, 24, 269, 1, 0, 0, 0, 26, 299, 1, 0, 0, 0,
		28, 302, 1, 0, 0, 0, 30, 319, 1, 0, 0, 0, 32, 321, 1, 0, 0, 0, 34, 352,
		1, 0, 0, 0, 36, 360, 1, 0, 0, 0, 38, 384, 1, 0, 0, 0, 40, 391, 1, 0, 0,
		0, 42, 396, 1, 0, 0, 0, 44, 443, 1, 0, 0, 0, 46, 450, 1, 0, 0, 0, 48, 452,
		1, 0, 0, 0, 50, 459, 1, 0, 0, 0, 52, 473, 1, 0, 0, 0, 54, 475, 1, 0, 0,
		0, 56, 480, 1, 0, 0, 0, 58, 493, 1, 0, 0, 0, 60, 495, 1, 0, 0, 0, 62, 529,
		1, 0, 0, 0, 64, 532, 1, 0, 0, 0, 66, 541, 1, 0, 0, 0, 68, 565, 1, 0, 0,
		0, 70, 576, 1, 0, 0, 0, 72, 74, 3, 2, 1, 0, 73, 72, 1, 0, 0, 0, 74, 77,
		1, 0, 0, 0, 75, 73, 1, 0, 0, 0, 75, 76, 1, 0, 0, 0, 76, 87, 1, 0, 0, 0,
		77, 75, 1, 0, 0, 0, 78, 82, 3, 4, 2, 0, 79, 81, 3, 2, 1, 0, 80, 79, 1,
		0, 0, 0, 81, 84, 1, 0, 0, 0, 82, 80, 1, 0, 0, 0, 82, 83, 1, 0, 0, 0, 83,
		86, 1, 0, 0, 0, 84, 82, 1, 0, 0, 0, 85, 78, 1, 0, 0, 0, 86, 89, 1, 0, 0,
		0, 87, 85, 1, 0, 0, 0, 87, 88, 1, 0, 0, 0, 88, 99, 1, 0, 0, 0, 89, 87,
		1, 0, 0, 0, 90, 94, 3, 6, 3, 0, 91, 93, 3, 2, 1, 0, 92, 91, 1, 0, 0, 0,
		93, 96, 1, 0, 0, 0, 94, 92, 1, 0, 0, 0, 94, 95, 1, 0, 0, 0, 95, 98, 1,
		0, 0, 0, 96, 94, 1, 0, 0, 0, 97, 90, 1, 0, 0, 0, 98, 101, 1, 0, 0, 0, 99,
		97, 1, 0, 0, 0, 99, 100, 1, 0, 0, 0, 100, 111, 1, 0, 0, 0, 101, 99, 1,
		0, 0, 0, 102, 106, 3, 8, 4, 0, 103, 105, 3, 2, 1, 0, 104, 103, 1, 0, 0,
		0, 105, 108, 1, 0, 0, 0, 106, 104, 1, 0, 0, 0, 106, 107, 1, 0, 0, 0, 107,
		110, 1, 0, 0, 0, 108, 106, 1, 0, 0, 0, 109, 102, 1, 0, 0, 0, 110, 113,
		1, 0, 0, 0, 111, 109, 1, 0, 0, 0, 111, 112, 1, 0, 0, 0, 112, 114, 1, 0,
		0, 0, 113, 111, 1, 0, 0, 0, 114, 115, 5, 0, 0, 1, 115, 1, 1, 0, 0, 0, 116,
		117, 5, 27, 0, 0, 117, 3, 1, 0, 0, 0, 118, 119, 5, 3, 0, 0, 119, 120, 5,
		20, 0, 0, 120, 5, 1, 0, 0, 0, 121, 122, 5, 17, 0, 0, 122, 124, 5, 24, 0,
		0, 123, 125, 3, 40, 20, 0, 124, 123, 1, 0, 0, 0, 124, 125, 1, 0, 0, 0,
		125, 7, 1, 0, 0, 0, 126, 130, 3, 10, 5, 0, 127, 130, 3, 60, 30, 0, 128,
		130, 3, 68, 34, 0, 129, 126, 1, 0, 0, 0, 129, 127, 1, 0, 0, 0, 129, 128,
		1, 0, 0, 0, 130, 9, 1, 0, 0, 0, 131, 132, 5, 24, 0, 0, 132, 134, 5, 1,
		0, 0, 133, 135, 3, 16, 8, 0, 134, 133, 1, 0, 0, 0, 134, 135, 1, 0, 0, 0,
		135, 138, 1, 0, 0, 0, 136, 137, 5, 4, 0, 0, 137, 139, 3, 12, 6, 0, 138,
		136, 1, 0, 0, 0, 138, 139, 1, 0, 0, 0, 139, 143, 1, 0, 0, 0, 140, 142,
		3, 2, 1, 0, 141, 140, 1, 0, 0, 0, 142, 145, 1, 0, 0, 0, 143, 141, 1, 0,
		0, 0, 143, 144, 1, 0, 0, 0, 144, 146, 1, 0, 0, 0, 145, 143, 1, 0, 0, 0,
		146, 150, 5, 7, 0, 0, 147, 149, 3, 2, 1, 0, 148, 147, 1, 0, 0, 0, 149,
		152, 1, 0, 0, 0, 150, 148, 1, 0, 0, 0, 150, 151, 1, 0, 0, 0, 151, 153,
		1, 0, 0, 0, 152, 150, 1, 0, 0, 0, 153, 154, 3, 20, 10, 0, 154, 155, 5,
		8, 0, 0, 155, 166, 1, 0, 0, 0, 156, 158, 5, 24, 0, 0, 157, 159, 3, 16,
		8, 0, 158, 157, 1, 0, 0, 0, 158, 159, 1, 0, 0, 0, 159, 160, 1, 0, 0, 0,
		160, 161, 5, 16, 0, 0, 161, 163, 3, 44, 22, 0, 162, 164, 3, 14, 7, 0, 163,
		162, 1, 0, 0, 0, 163, 164, 1, 0, 0, 0, 164, 166, 1, 0, 0, 0, 165, 131,
		1, 0, 0, 0, 165, 156, 1, 0, 0, 0, 166, 11, 1, 0, 0, 0, 167, 178, 3, 44,
		22, 0, 168, 172, 5, 13, 0, 0, 169, 171, 3, 2, 1, 0, 170, 169, 1, 0, 0,
		0, 171, 174, 1, 0, 0, 0, 172, 170, 1, 0, 0, 0, 172, 173, 1, 0, 0, 0, 173,
		175, 1, 0, 0, 0, 174, 172, 1, 0, 0, 0, 175, 177, 3, 44, 22, 0, 176, 168,
		1, 0, 0, 0, 177, 180, 1, 0, 0, 0, 178, 176, 1, 0, 0, 0, 178, 179, 1, 0,
		0, 0, 179, 13, 1, 0, 0, 0, 180, 178, 1, 0, 0, 0, 181, 183, 3, 2, 1, 0,
		182, 181, 1, 0, 0, 0, 183, 186, 1, 0, 0, 0, 184, 182, 1, 0, 0, 0, 184,
		185, 1, 0, 0, 0, 185, 187, 1, 0, 0, 0, 186, 184, 1, 0, 0, 0, 187, 191,
		5, 7, 0, 0, 188, 190, 3, 2, 1, 0, 189, 188, 1, 0, 0, 0, 190, 193, 1, 0,
		0, 0, 191, 189, 1, 0, 0, 0, 191, 192, 1, 0, 0, 0, 192, 203, 1, 0, 0, 0,
		193, 191, 1, 0, 0, 0, 194, 198, 3, 38, 19, 0, 195, 197, 3, 2, 1, 0, 196,
		195, 1, 0, 0, 0, 197, 200, 1, 0, 0, 0, 198, 196, 1, 0, 0, 0, 198, 199,
		1, 0, 0, 0, 199, 202, 1, 0, 0, 0, 200, 198, 1, 0, 0, 0, 201, 194, 1, 0,
		0, 0, 202, 205, 1, 0, 0, 0, 203, 201, 1, 0, 0, 0, 203, 204, 1, 0, 0, 0,
		204, 206, 1, 0, 0, 0, 205, 203, 1, 0, 0, 0, 206, 207, 5, 8, 0, 0, 207,
		15, 1, 0, 0, 0, 208, 212, 5, 11, 0, 0, 209, 211, 3, 2, 1, 0, 210, 209,
		1, 0, 0, 0, 211, 214, 1, 0, 0, 0, 212, 210, 1, 0, 0, 0, 212, 213, 1, 0,
		0, 0, 213, 215, 1, 0, 0, 0, 214, 212, 1, 0, 0, 0, 215, 226, 3, 18, 9, 0,
		216, 220, 5, 13, 0, 0, 217, 219, 3, 2, 1, 0, 218, 217, 1, 0, 0, 0, 219,
		222, 1, 0, 0, 0, 220, 218, 1, 0, 0, 0, 220, 221, 1, 0, 0, 0, 221, 223,
		1, 0, 0, 0, 222, 220, 1, 0, 0, 0, 223, 225, 3, 18, 9, 0, 224, 216, 1, 0,
		0, 0, 225, 228, 1, 0, 0, 0, 226, 224, 1, 0, 0, 0, 226, 227, 1, 0, 0, 0,
		227, 232, 1, 0, 0, 0, 228, 226, 1, 0, 0, 0, 229, 231, 3, 2, 1, 0, 230,
		229, 1, 0, 0, 0, 231, 234, 1, 0, 0, 0, 232, 230, 1, 0, 0, 0, 232, 233,
		1, 0, 0, 0, 233, 235, 1, 0, 0, 0, 234, 232, 1, 0, 0, 0, 235, 236, 5, 12,
		0, 0, 236, 17, 1, 0, 0, 0, 237, 239, 5, 24, 0, 0, 238, 240, 5, 14, 0, 0,
		239, 238, 1, 0, 0, 0, 239, 240, 1, 0, 0, 0, 240, 243, 1, 0, 0, 0, 241,
		242, 5, 4, 0, 0, 242, 244, 3, 44, 22, 0, 243, 241, 1, 0, 0, 0, 243, 244,
		1, 0, 0, 0, 244, 247, 1, 0, 0, 0, 245, 246, 5, 16, 0, 0, 246, 248, 3, 44,
		22, 0, 247, 245, 1, 0, 0, 0, 247, 248, 1, 0, 0, 0, 248, 19, 1, 0, 0, 0,
		249, 254, 3, 28, 14, 0, 250, 254, 3, 22, 11, 0, 251, 254, 3, 24, 12, 0,
		252, 254, 3, 38, 19, 0, 253, 249, 1, 0, 0, 0, 253, 250, 1, 0, 0, 0, 253,
		251, 1, 0, 0, 0, 253, 252, 1, 0, 0, 0, 254, 258, 1, 0, 0, 0, 255, 257,
		3, 2, 1, 0, 256, 255, 1, 0, 0, 0, 257, 260, 1, 0, 0, 0, 258, 256, 1, 0,
		0, 0, 258, 259, 1, 0, 0, 0, 259, 262, 1, 0, 0, 0, 260, 258, 1, 0, 0, 0,
		261, 253, 1, 0, 0, 0, 262, 265, 1, 0, 0, 0, 263, 261, 1, 0, 0, 0, 263,
		264, 1, 0, 0, 0, 264, 21, 1, 0, 0, 0, 265, 263, 1, 0, 0, 0, 266, 267, 5,
		18, 0, 0, 267, 268, 5, 24, 0, 0, 268, 23, 1, 0, 0, 0, 269, 270, 5, 6, 0,
		0, 270, 274, 5, 24, 0, 0, 271, 273, 3, 2, 1, 0, 272, 271, 1, 0, 0, 0, 273,
		276, 1, 0, 0, 0, 274, 272, 1, 0, 0, 0, 274, 275, 1, 0, 0, 0, 275, 277,
		1, 0, 0, 0, 276, 274, 1, 0, 0, 0, 277, 281, 5, 7, 0, 0, 278, 280, 3, 2,
		1, 0, 279, 278, 1, 0, 0, 0, 280, 283, 1, 0, 0, 0, 281, 279, 1, 0, 0, 0,
		281, 282, 1, 0, 0, 0, 282, 284, 1, 0, 0, 0, 283, 281, 1, 0, 0, 0, 284,
		285, 3, 26, 13, 0, 285, 286, 5, 8, 0, 0, 286, 25, 1, 0, 0, 0, 287, 290,
		3, 28, 14, 0, 288, 290, 3, 38, 19, 0, 289, 287, 1, 0, 0, 0, 289, 288, 1,
		0, 0, 0, 290, 294, 1, 0, 0, 0, 291, 293, 3, 2, 1, 0, 292, 291, 1, 0, 0,
		0, 293, 296, 1, 0, 0, 0, 294, 292, 1, 0, 0, 0, 294, 295, 1, 0, 0, 0, 295,
		298, 1, 0, 0, 0, 296, 294, 1, 0, 0, 0, 297, 289, 1, 0, 0, 0, 298, 301,
		1, 0, 0, 0, 299, 297, 1, 0, 0, 0, 299, 300, 1, 0, 0, 0, 300, 27, 1, 0,
		0, 0, 301, 299, 1, 0, 0, 0, 302, 303, 5, 24, 0, 0, 303, 306, 3, 44, 22,
		0, 304, 305, 5, 16, 0, 0, 305, 307, 3, 30, 15, 0, 306, 304, 1, 0, 0, 0,
		306, 307, 1, 0, 0, 0, 307, 311, 1, 0, 0, 0, 308, 310, 3, 34, 17, 0, 309,
		308, 1, 0, 0, 0, 310, 313, 1, 0, 0, 0, 311, 309, 1, 0, 0, 0, 311, 312,
		1, 0, 0, 0, 312, 315, 1, 0, 0, 0, 313, 311, 1, 0, 0, 0, 314, 316, 3, 36,
		18, 0, 315, 314, 1, 0, 0, 0, 315, 316, 1, 0, 0, 0, 316, 29, 1, 0, 0, 0,
		317, 320, 3, 58, 29, 0, 318, 320, 3, 32, 16, 0, 319, 317, 1, 0, 0, 0, 319,
		318, 1, 0, 0, 0, 320, 31, 1, 0, 0, 0, 321, 325, 5, 9, 0, 0, 322, 324, 3,
		2, 1, 0, 323, 322, 1, 0, 0, 0, 324, 327, 1, 0, 0, 0, 325, 323, 1, 0, 0,
		0, 325, 326, 1, 0, 0, 0, 326, 348, 1, 0, 0, 0, 327, 325, 1, 0, 0, 0, 328,
		339, 3, 58, 29, 0, 329, 333, 5, 13, 0, 0, 330, 332, 3, 2, 1, 0, 331, 330,
		1, 0, 0, 0, 332, 335, 1, 0, 0, 0, 333, 331, 1, 0, 0, 0, 333, 334, 1, 0,
		0, 0, 334, 336, 1, 0, 0, 0, 335, 333, 1, 0, 0, 0, 336, 338, 3, 58, 29,
		0, 337, 329, 1, 0, 0, 0, 338, 341, 1, 0, 0, 0, 339, 337, 1, 0, 0, 0, 339,
		340, 1, 0, 0, 0, 340, 345, 1, 0, 0, 0, 341, 339, 1, 0, 0, 0, 342, 344,
		3, 2, 1, 0, 343, 342, 1, 0, 0, 0, 344, 347, 1, 0, 0, 0, 345, 343, 1, 0,
		0, 0, 345, 346, 1, 0, 0, 0, 346, 349, 1, 0, 0, 0, 347, 345, 1, 0, 0, 0,
		348, 328, 1, 0, 0, 0, 348, 349, 1, 0, 0, 0, 349, 350, 1, 0, 0, 0, 350,
		351, 5, 10, 0, 0, 351, 33, 1, 0, 0, 0, 352, 353, 5, 17, 0, 0, 353, 355,
		5, 24, 0, 0, 354, 356, 3, 40, 20, 0, 355, 354, 1, 0, 0, 0, 355, 356, 1,
		0, 0, 0, 356, 35, 1, 0, 0, 0, 357, 359, 3, 2, 1, 0, 358, 357, 1, 0, 0,
		0, 359, 362, 1, 0, 0, 0, 360, 358, 1, 0, 0, 0, 360, 361, 1, 0, 0, 0, 361,
		363, 1, 0, 0, 0, 362, 360, 1, 0, 0, 0, 363, 367, 5, 7, 0, 0, 364, 366,
		3, 2, 1, 0, 365, 364, 1, 0, 0, 0, 366, 369, 1, 0, 0, 0, 367, 365, 1, 0,
		0, 0, 367, 368, 1, 0, 0, 0, 368, 379, 1, 0, 0, 0, 369, 367, 1, 0, 0, 0,
		370, 374, 3, 38, 19, 0, 371, 373, 3, 2, 1, 0, 372, 371, 1, 0, 0, 0, 373,
		376, 1, 0, 0, 0, 374, 372, 1, 0, 0, 0, 374, 375, 1, 0, 0, 0, 375, 378,
		1, 0, 0, 0, 376, 374, 1, 0, 0, 0, 377, 370, 1, 0, 0, 0, 378, 381, 1, 0,
		0, 0, 379, 377, 1, 0, 0, 0, 379, 380, 1, 0, 0, 0, 380, 382, 1, 0, 0, 0,
		381, 379, 1, 0, 0, 0, 382, 383, 5, 8, 0, 0, 383, 37, 1, 0, 0, 0, 384, 385,
		5, 17, 0, 0, 385, 387, 5, 24, 0, 0, 386, 388, 3, 40, 20, 0, 387, 386, 1,
		0, 0, 0, 387, 388, 1, 0, 0, 0, 388, 39, 1, 0, 0, 0, 389, 392, 3, 42, 21,
		0, 390, 392, 3, 56, 28, 0, 391, 389, 1, 0, 0, 0, 391, 390, 1, 0, 0, 0,
		392, 41, 1, 0, 0, 0, 393, 395, 3, 2, 1, 0, 394, 393, 1, 0, 0, 0, 395, 398,
		1, 0, 0, 0, 396, 394, 1, 0, 0, 0, 396, 397, 1, 0, 0, 0, 397, 399, 1, 0,
		0, 0, 398, 396, 1, 0, 0, 0, 399, 403, 5, 7, 0, 0, 400, 402, 3, 2, 1, 0,
		401, 400, 1, 0, 0, 0, 402, 405, 1, 0, 0, 0, 403, 401, 1, 0, 0, 0, 403,
		404, 1, 0, 0, 0, 404, 419, 1, 0, 0, 0, 405, 403, 1, 0, 0, 0, 406, 416,
		3, 56, 28, 0, 407, 409, 3, 2, 1, 0, 408, 407, 1, 0, 0, 0, 409, 410, 1,
		0, 0, 0, 410, 408, 1, 0, 0, 0, 410, 411, 1, 0, 0, 0, 411, 412, 1, 0, 0,
		0, 412, 413, 3, 56, 28, 0, 413, 415, 1, 0, 0, 0, 414, 408, 1, 0, 0, 0,
		415, 418, 1, 0, 0, 0, 416, 414, 1, 0, 0, 0, 416, 417, 1, 0, 0, 0, 417,
		420, 1, 0, 0, 0, 418, 416, 1, 0, 0, 0, 419, 406, 1, 0, 0, 0, 419, 420,
		1, 0, 0, 0, 420, 424, 1, 0, 0, 0, 421, 423, 3, 2, 1, 0, 422, 421, 1, 0,
		0, 0, 423, 426, 1, 0, 0, 0, 424, 422, 1, 0, 0, 0, 424, 425, 1, 0, 0, 0,
		425, 427, 1, 0, 0, 0, 426, 424, 1, 0, 0, 0, 427, 428, 5, 8, 0, 0, 428,
		43, 1, 0, 0, 0, 429, 431, 3, 48, 24, 0, 430, 432, 3, 52, 26, 0, 431, 430,
		1, 0, 0, 0, 431, 432, 1, 0, 0, 0, 432, 444, 1, 0, 0, 0, 433, 435, 3, 54,
		27, 0, 434, 436, 3, 50, 25, 0, 435, 434, 1, 0, 0, 0, 435, 436, 1, 0, 0,
		0, 436, 438, 1, 0, 0, 0, 437, 439, 3, 46, 23, 0, 438, 437, 1, 0, 0, 0,
		438, 439, 1, 0, 0, 0, 439, 441, 1, 0, 0, 0, 440, 442, 3, 52, 26, 0, 441,
		440, 1, 0, 0, 0, 441, 442, 1, 0, 0, 0, 442, 444, 1, 0, 0, 0, 443, 429,
		1, 0, 0, 0, 443, 433, 1, 0, 0, 0, 444, 45, 1, 0, 0, 0, 445, 446, 5, 9,
		0, 0, 446, 451, 5, 10, 0, 0, 447, 448, 5, 9, 0, 0, 448, 449, 5, 22, 0,
		0, 449, 451, 5, 10, 0, 0, 450, 445, 1, 0, 0, 0, 450, 447, 1, 0, 0, 0, 451,
		47, 1, 0, 0, 0, 452, 453, 5, 5, 0, 0, 453, 454, 5, 11, 0, 0, 454, 455,
		3, 44, 22, 0, 455, 456, 5, 13, 0, 0, 456, 457, 3, 44, 22, 0, 457, 458,
		5, 12, 0, 0, 458, 49, 1, 0, 0, 0, 459, 460, 5, 11, 0, 0, 460, 465, 3, 44,
		22, 0, 461, 462, 5, 13, 0, 0, 462, 464, 3, 44, 22, 0, 463, 461, 1, 0, 0,
		0, 464, 467, 1, 0, 0, 0, 465, 463, 1, 0, 0, 0, 465, 466, 1, 0, 0, 0, 466,
		468, 1, 0, 0, 0, 467, 465, 1, 0, 0, 0, 468, 469, 5, 12, 0, 0, 469, 51,
		1, 0, 0, 0, 470, 471, 5, 14, 0, 0, 471, 474, 5, 14, 0, 0, 472, 474, 5,
		14, 0, 0, 473, 470, 1, 0, 0, 0, 473, 472, 1, 0, 0, 0, 474, 53, 1, 0, 0,
		0, 475, 478, 5, 24, 0, 0, 476, 477, 5, 15, 0, 0, 477, 479, 5, 24, 0, 0,
		478, 476, 1, 0, 0, 0, 478, 479, 1, 0, 0, 0, 479, 55, 1, 0, 0, 0, 480, 484,
		5, 24, 0, 0, 481, 483, 3, 58, 29, 0, 482, 481, 1, 0, 0, 0, 483, 486, 1,
		0, 0, 0, 484, 482, 1, 0, 0, 0, 484, 485, 1, 0, 0, 0, 485, 57, 1, 0, 0,
		0, 486, 484, 1, 0, 0, 0, 487, 494, 5, 19, 0, 0, 488, 494, 5, 20, 0, 0,
		489, 494, 5, 22, 0, 0, 490, 494, 5, 21, 0, 0, 491, 494, 5, 23, 0, 0, 492,
		494, 3, 54, 27, 0, 493, 487, 1, 0, 0, 0, 493, 488, 1, 0, 0, 0, 493, 489,
		1, 0, 0, 0, 493, 490, 1, 0, 0, 0, 493, 491, 1, 0, 0, 0, 493, 492, 1, 0,
		0, 0, 494, 59, 1, 0, 0, 0, 495, 496, 5, 24, 0, 0, 496, 499, 5, 2, 0, 0,
		497, 498, 5, 4, 0, 0, 498, 500, 3, 12, 6, 0, 499, 497, 1, 0, 0, 0, 499,
		500, 1, 0, 0, 0, 500, 504, 1, 0, 0, 0, 501, 503, 3, 2, 1, 0, 502, 501,
		1, 0, 0, 0, 503, 506, 1, 0, 0, 0, 504, 502, 1, 0, 0, 0, 504, 505, 1, 0,
		0, 0, 505, 507, 1, 0, 0, 0, 506, 504, 1, 0, 0, 0, 507, 511, 5, 7, 0, 0,
		508, 510, 3, 2, 1, 0, 509, 508, 1, 0, 0, 0, 510, 513, 1, 0, 0, 0, 511,
		509, 1, 0, 0, 0, 511, 512, 1, 0, 0, 0, 512, 514, 1, 0, 0, 0, 513, 511,
		1, 0, 0, 0, 514, 515, 3, 62, 31, 0, 515, 516, 5, 8, 0, 0, 516, 61, 1, 0,
		0, 0, 517, 520, 3, 64, 32, 0, 518, 520, 3, 38, 19, 0, 519, 517, 1, 0, 0,
		0, 519, 518, 1, 0, 0, 0, 520, 524, 1, 0, 0, 0, 521, 523, 3, 2, 1, 0, 522,
		521, 1, 0, 0, 0, 523, 526, 1, 0, 0, 0, 524, 522, 1, 0, 0, 0, 524, 525,
		1, 0, 0, 0, 525, 528, 1, 0, 0, 0, 526, 524, 1, 0, 0, 0, 527, 519, 1, 0,
		0, 0, 528, 531, 1, 0, 0, 0, 529, 527, 1, 0, 0, 0, 529, 530, 1, 0, 0, 0,
		530, 63, 1, 0, 0, 0, 531, 529, 1, 0, 0, 0, 532, 533, 5, 24, 0, 0, 533,
		534, 5, 16, 0, 0, 534, 536, 7, 0, 0, 0, 535, 537, 3, 66, 33, 0, 536, 535,
		1, 0, 0, 0, 536, 537, 1, 0, 0, 0, 537, 65, 1, 0, 0, 0, 538, 540, 3, 2,
		1, 0, 539, 538, 1, 0, 0, 0, 540, 543, 1, 0, 0, 0, 541, 539, 1, 0, 0, 0,
		541, 542, 1, 0, 0, 0, 542, 544, 1, 0, 0, 0, 543, 541, 1, 0, 0, 0, 544,
		548, 5, 7, 0, 0, 545, 547, 3, 2, 1, 0, 546, 545, 1, 0, 0, 0, 547, 550,
		1, 0, 0, 0, 548, 546, 1, 0, 0, 0, 548, 549, 1, 0, 0, 0, 549, 560, 1, 0,
		0, 0, 550, 548, 1, 0, 0, 0, 551, 555, 3, 38, 19, 0, 552, 554, 3, 2, 1,
		0, 553, 552, 1, 0, 0, 0, 554, 557, 1, 0, 0, 0, 555, 553, 1, 0, 0, 0, 555,
		556, 1, 0, 0, 0, 556, 559, 1, 0, 0, 0, 557, 555, 1, 0, 0, 0, 558, 551,
		1, 0, 0, 0, 559, 562, 1, 0, 0, 0, 560, 558, 1, 0, 0, 0, 560, 561, 1, 0,
		0, 0, 561, 563, 1, 0, 0, 0, 562, 560, 1, 0, 0, 0, 563, 564, 5, 8, 0, 0,
		564, 67, 1, 0, 0, 0, 565, 567, 5, 24, 0, 0, 566, 568, 3, 16, 8, 0, 567,
		566, 1, 0, 0, 0, 567, 568, 1, 0, 0, 0, 568, 569, 1, 0, 0, 0, 569, 571,
		3, 44, 22, 0, 570, 572, 3, 70, 35, 0, 571, 570, 1, 0, 0, 0, 571, 572, 1,
		0, 0, 0, 572, 69, 1, 0, 0, 0, 573, 575, 3, 2, 1, 0, 574, 573, 1, 0, 0,
		0, 575, 578, 1, 0, 0, 0, 576, 574, 1, 0, 0, 0, 576, 577, 1, 0, 0, 0, 577,
		579, 1, 0, 0, 0, 578, 576, 1, 0, 0, 0, 579, 583, 5, 7, 0, 0, 580, 582,
		3, 2, 1, 0, 581, 580, 1, 0, 0, 0, 582, 585, 1, 0, 0, 0, 583, 581, 1, 0,
		0, 0, 583, 584, 1, 0, 0, 0, 584, 595, 1, 0, 0, 0, 585, 583, 1, 0, 0, 0,
		586, 590, 3, 38, 19, 0, 587, 589, 3, 2, 1, 0, 588, 587, 1, 0, 0, 0, 589,
		592, 1, 0, 0, 0, 590, 588, 1, 0, 0, 0, 590, 591, 1, 0, 0, 0, 591, 594,
		1, 0, 0, 0, 592, 590, 1, 0, 0, 0, 593, 586, 1, 0, 0, 0, 594, 597, 1, 0,
		0, 0, 595, 593, 1, 0, 0, 0, 595, 596, 1, 0, 0, 0, 596, 598, 1, 0, 0, 0,
		597, 595, 1, 0, 0, 0, 598, 599, 5, 8, 0, 0, 599, 71, 1, 0, 0, 0, 87, 75,
		82, 87, 94, 99, 106, 111, 124, 129, 134, 138, 143, 150, 158, 163, 165,
		172, 178, 184, 191, 198, 203, 212, 220, 226, 232, 239, 243, 247, 253, 258,
		263, 274, 281, 289, 294, 299, 306, 311, 315, 319, 325, 333, 339, 345, 348,
		355, 360, 367, 374, 379, 387, 391, 396, 403, 410, 416, 419, 424, 431, 435,
		438, 441, 443, 450, 465, 473, 478, 484, 493, 499, 504, 511, 519, 524, 529,
		536, 541, 548, 555, 560, 567, 571, 576, 583, 590, 595,
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
	OracleParserRULE_actionDef       = 12
	OracleParserRULE_actionBody      = 13
	OracleParserRULE_fieldDef        = 14
	OracleParserRULE_fieldDefault    = 15
	OracleParserRULE_arrayDefault    = 16
	OracleParserRULE_inlineDomain    = 17
	OracleParserRULE_fieldBody       = 18
	OracleParserRULE_domain          = 19
	OracleParserRULE_domainContent   = 20
	OracleParserRULE_domainBlock     = 21
	OracleParserRULE_typeRef         = 22
	OracleParserRULE_arrayModifier   = 23
	OracleParserRULE_mapType         = 24
	OracleParserRULE_typeArgs        = 25
	OracleParserRULE_typeModifiers   = 26
	OracleParserRULE_qualifiedIdent  = 27
	OracleParserRULE_expression      = 28
	OracleParserRULE_expressionValue = 29
	OracleParserRULE_enumDef         = 30
	OracleParserRULE_enumBody        = 31
	OracleParserRULE_enumValue       = 32
	OracleParserRULE_enumValueBody   = 33
	OracleParserRULE_typeDefDef      = 34
	OracleParserRULE_typeDefBody     = 35
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
	p.SetState(75)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(72)
			p.Nl()
		}

		p.SetState(77)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	p.SetState(87)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserIMPORT {
		{
			p.SetState(78)
			p.ImportStmt()
		}
		p.SetState(82)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(79)
				p.Nl()
			}

			p.SetState(84)
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
	}
	p.SetState(99)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserAT {
		{
			p.SetState(90)
			p.FileDomain()
		}
		p.SetState(94)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(91)
				p.Nl()
			}

			p.SetState(96)
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
	}
	p.SetState(111)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserIDENT {
		{
			p.SetState(102)
			p.Definition()
		}
		p.SetState(106)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(103)
				p.Nl()
			}

			p.SetState(108)
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
	}
	{
		p.SetState(114)
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
		p.SetState(116)
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
		p.SetState(118)
		p.Match(OracleParserIMPORT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(119)
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
		p.SetState(121)
		p.Match(OracleParserAT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(122)
		p.Match(OracleParserIDENT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(124)
	p.GetErrorHandler().Sync(p)

	if p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 7, p.GetParserRuleContext()) == 1 {
		{
			p.SetState(123)
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
	p.SetState(129)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 8, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(126)
			p.StructDef()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(127)
			p.EnumDef()
		}

	case 3:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(128)
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

	p.SetState(165)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 15, p.GetParserRuleContext()) {
	case 1:
		localctx = NewStructFullContext(p, localctx)
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(131)
			p.Match(OracleParserIDENT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(132)
			p.Match(OracleParserSTRUCT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		p.SetState(134)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		if _la == OracleParserLT {
			{
				p.SetState(133)
				p.TypeParams()
			}

		}
		p.SetState(138)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		if _la == OracleParserEXTENDS {
			{
				p.SetState(136)
				p.Match(OracleParserEXTENDS)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}
			{
				p.SetState(137)
				p.TypeRefList()
			}

		}
		p.SetState(143)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(140)
				p.Nl()
			}

			p.SetState(145)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}
		{
			p.SetState(146)
			p.Match(OracleParserLBRACE)
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

		for _la == OracleParserNEWLINE {
			{
				p.SetState(147)
				p.Nl()
			}

			p.SetState(152)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}
		{
			p.SetState(153)
			p.StructBody()
		}
		{
			p.SetState(154)
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
			p.SetState(156)
			p.Match(OracleParserIDENT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		p.SetState(158)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		if _la == OracleParserLT {
			{
				p.SetState(157)
				p.TypeParams()
			}

		}
		{
			p.SetState(160)
			p.Match(OracleParserEQUALS)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(161)
			p.TypeRef()
		}
		p.SetState(163)
		p.GetErrorHandler().Sync(p)

		if p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 14, p.GetParserRuleContext()) == 1 {
			{
				p.SetState(162)
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
		p.SetState(167)
		p.TypeRef()
	}
	p.SetState(178)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserCOMMA {
		{
			p.SetState(168)
			p.Match(OracleParserCOMMA)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		p.SetState(172)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(169)
				p.Nl()
			}

			p.SetState(174)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}
		{
			p.SetState(175)
			p.TypeRef()
		}

		p.SetState(180)
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
	p.SetState(184)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(181)
			p.Nl()
		}

		p.SetState(186)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(187)
		p.Match(OracleParserLBRACE)
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

	for _la == OracleParserNEWLINE {
		{
			p.SetState(188)
			p.Nl()
		}

		p.SetState(193)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	p.SetState(203)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserAT {
		{
			p.SetState(194)
			p.Domain()
		}
		p.SetState(198)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(195)
				p.Nl()
			}

			p.SetState(200)
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
	}
	{
		p.SetState(206)
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
		p.SetState(208)
		p.Match(OracleParserLT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(212)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(209)
			p.Nl()
		}

		p.SetState(214)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(215)
		p.TypeParam()
	}
	p.SetState(226)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserCOMMA {
		{
			p.SetState(216)
			p.Match(OracleParserCOMMA)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		p.SetState(220)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(217)
				p.Nl()
			}

			p.SetState(222)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}
		{
			p.SetState(223)
			p.TypeParam()
		}

		p.SetState(228)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	p.SetState(232)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(229)
			p.Nl()
		}

		p.SetState(234)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(235)
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
		p.SetState(237)
		p.Match(OracleParserIDENT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(239)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == OracleParserQUESTION {
		{
			p.SetState(238)
			p.Match(OracleParserQUESTION)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	}
	p.SetState(243)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == OracleParserEXTENDS {
		{
			p.SetState(241)
			p.Match(OracleParserEXTENDS)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(242)
			p.TypeRef()
		}

	}
	p.SetState(247)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == OracleParserEQUALS {
		{
			p.SetState(245)
			p.Match(OracleParserEQUALS)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(246)
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
	p.SetState(263)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&17170496) != 0 {
		p.SetState(253)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}

		switch p.GetTokenStream().LA(1) {
		case OracleParserIDENT:
			{
				p.SetState(249)
				p.FieldDef()
			}

		case OracleParserMINUS:
			{
				p.SetState(250)
				p.FieldOmit()
			}

		case OracleParserACTION:
			{
				p.SetState(251)
				p.ActionDef()
			}

		case OracleParserAT:
			{
				p.SetState(252)
				p.Domain()
			}

		default:
			p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
			goto errorExit
		}
		p.SetState(258)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(255)
				p.Nl()
			}

			p.SetState(260)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}

		p.SetState(265)
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
		p.SetState(266)
		p.Match(OracleParserMINUS)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(267)
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
		p.SetState(269)
		p.Match(OracleParserACTION)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(270)
		p.Match(OracleParserIDENT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(274)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(271)
			p.Nl()
		}

		p.SetState(276)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(277)
		p.Match(OracleParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(281)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(278)
			p.Nl()
		}

		p.SetState(283)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(284)
		p.ActionBody()
	}
	{
		p.SetState(285)
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
	p.SetState(299)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserAT || _la == OracleParserIDENT {
		p.SetState(289)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}

		switch p.GetTokenStream().LA(1) {
		case OracleParserIDENT:
			{
				p.SetState(287)
				p.FieldDef()
			}

		case OracleParserAT:
			{
				p.SetState(288)
				p.Domain()
			}

		default:
			p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
			goto errorExit
		}
		p.SetState(294)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(291)
				p.Nl()
			}

			p.SetState(296)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}

		p.SetState(301)
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
		p.SetState(302)
		p.Match(OracleParserIDENT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(303)
		p.TypeRef()
	}
	p.SetState(306)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == OracleParserEQUALS {
		{
			p.SetState(304)
			p.Match(OracleParserEQUALS)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(305)
			p.FieldDefault()
		}

	}
	p.SetState(311)
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
				p.SetState(308)
				p.InlineDomain()
			}

		}
		p.SetState(313)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 38, p.GetParserRuleContext())
		if p.HasError() {
			goto errorExit
		}
	}
	p.SetState(315)
	p.GetErrorHandler().Sync(p)

	if p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 39, p.GetParserRuleContext()) == 1 {
		{
			p.SetState(314)
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
	p.SetState(319)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case OracleParserTRIPLE_STRING_LIT, OracleParserSTRING_LIT, OracleParserFLOAT_LIT, OracleParserINT_LIT, OracleParserBOOL_LIT, OracleParserIDENT:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(317)
			p.ExpressionValue()
		}

	case OracleParserLBRACKET:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(318)
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
		p.SetState(321)
		p.Match(OracleParserLBRACKET)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(325)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(322)
			p.Nl()
		}

		p.SetState(327)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	p.SetState(348)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&33030144) != 0 {
		{
			p.SetState(328)
			p.ExpressionValue()
		}
		p.SetState(339)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserCOMMA {
			{
				p.SetState(329)
				p.Match(OracleParserCOMMA)
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
			_la = p.GetTokenStream().LA(1)

			for _la == OracleParserNEWLINE {
				{
					p.SetState(330)
					p.Nl()
				}

				p.SetState(335)
				p.GetErrorHandler().Sync(p)
				if p.HasError() {
					goto errorExit
				}
				_la = p.GetTokenStream().LA(1)
			}
			{
				p.SetState(336)
				p.ExpressionValue()
			}

			p.SetState(341)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}
		p.SetState(345)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(342)
				p.Nl()
			}

			p.SetState(347)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}

	}
	{
		p.SetState(350)
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
		p.SetState(352)
		p.Match(OracleParserAT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(353)
		p.Match(OracleParserIDENT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(355)
	p.GetErrorHandler().Sync(p)

	if p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 46, p.GetParserRuleContext()) == 1 {
		{
			p.SetState(354)
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
	p.SetState(360)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(357)
			p.Nl()
		}

		p.SetState(362)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(363)
		p.Match(OracleParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(367)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(364)
			p.Nl()
		}

		p.SetState(369)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	p.SetState(379)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserAT {
		{
			p.SetState(370)
			p.Domain()
		}
		p.SetState(374)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(371)
				p.Nl()
			}

			p.SetState(376)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}

		p.SetState(381)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(382)
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
		p.SetState(384)
		p.Match(OracleParserAT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(385)
		p.Match(OracleParserIDENT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(387)
	p.GetErrorHandler().Sync(p)

	if p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 51, p.GetParserRuleContext()) == 1 {
		{
			p.SetState(386)
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
	p.SetState(391)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case OracleParserLBRACE, OracleParserNEWLINE:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(389)
			p.DomainBlock()
		}

	case OracleParserIDENT:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(390)
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
	p.SetState(396)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(393)
			p.Nl()
		}

		p.SetState(398)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(399)
		p.Match(OracleParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(403)
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
				p.SetState(400)
				p.Nl()
			}

		}
		p.SetState(405)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 54, p.GetParserRuleContext())
		if p.HasError() {
			goto errorExit
		}
	}
	p.SetState(419)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == OracleParserIDENT {
		{
			p.SetState(406)
			p.Expression()
		}
		p.SetState(416)
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
				p.SetState(408)
				p.GetErrorHandler().Sync(p)
				if p.HasError() {
					goto errorExit
				}
				_la = p.GetTokenStream().LA(1)

				for ok := true; ok; ok = _la == OracleParserNEWLINE {
					{
						p.SetState(407)
						p.Nl()
					}

					p.SetState(410)
					p.GetErrorHandler().Sync(p)
					if p.HasError() {
						goto errorExit
					}
					_la = p.GetTokenStream().LA(1)
				}
				{
					p.SetState(412)
					p.Expression()
				}

			}
			p.SetState(418)
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
	p.SetState(424)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(421)
			p.Nl()
		}

		p.SetState(426)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(427)
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

	p.SetState(443)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case OracleParserMAP:
		localctx = NewTypeRefMapContext(p, localctx)
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(429)
			p.MapType()
		}
		p.SetState(431)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		if _la == OracleParserQUESTION {
			{
				p.SetState(430)
				p.TypeModifiers()
			}

		}

	case OracleParserIDENT:
		localctx = NewTypeRefNormalContext(p, localctx)
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(433)
			p.QualifiedIdent()
		}
		p.SetState(435)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		if _la == OracleParserLT {
			{
				p.SetState(434)
				p.TypeArgs()
			}

		}
		p.SetState(438)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		if _la == OracleParserLBRACKET {
			{
				p.SetState(437)
				p.ArrayModifier()
			}

		}
		p.SetState(441)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		if _la == OracleParserQUESTION {
			{
				p.SetState(440)
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
	p.SetState(450)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 64, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(445)
			p.Match(OracleParserLBRACKET)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(446)
			p.Match(OracleParserRBRACKET)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(447)
			p.Match(OracleParserLBRACKET)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(448)
			p.Match(OracleParserINT_LIT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(449)
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
		p.SetState(452)
		p.Match(OracleParserMAP)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(453)
		p.Match(OracleParserLT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(454)
		p.TypeRef()
	}
	{
		p.SetState(455)
		p.Match(OracleParserCOMMA)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(456)
		p.TypeRef()
	}
	{
		p.SetState(457)
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
		p.SetState(459)
		p.Match(OracleParserLT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(460)
		p.TypeRef()
	}
	p.SetState(465)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserCOMMA {
		{
			p.SetState(461)
			p.Match(OracleParserCOMMA)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(462)
			p.TypeRef()
		}

		p.SetState(467)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
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
	p.SetState(473)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 66, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(470)
			p.Match(OracleParserQUESTION)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(471)
			p.Match(OracleParserQUESTION)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(472)
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
		p.SetState(475)
		p.Match(OracleParserIDENT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(478)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == OracleParserDOT {
		{
			p.SetState(476)
			p.Match(OracleParserDOT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(477)
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
		p.SetState(480)
		p.Match(OracleParserIDENT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(484)
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
				p.SetState(481)
				p.ExpressionValue()
			}

		}
		p.SetState(486)
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
	p.SetState(493)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case OracleParserTRIPLE_STRING_LIT:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(487)
			p.Match(OracleParserTRIPLE_STRING_LIT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case OracleParserSTRING_LIT:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(488)
			p.Match(OracleParserSTRING_LIT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case OracleParserINT_LIT:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(489)
			p.Match(OracleParserINT_LIT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case OracleParserFLOAT_LIT:
		p.EnterOuterAlt(localctx, 4)
		{
			p.SetState(490)
			p.Match(OracleParserFLOAT_LIT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case OracleParserBOOL_LIT:
		p.EnterOuterAlt(localctx, 5)
		{
			p.SetState(491)
			p.Match(OracleParserBOOL_LIT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case OracleParserIDENT:
		p.EnterOuterAlt(localctx, 6)
		{
			p.SetState(492)
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
		p.SetState(495)
		p.Match(OracleParserIDENT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(496)
		p.Match(OracleParserENUM)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(499)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == OracleParserEXTENDS {
		{
			p.SetState(497)
			p.Match(OracleParserEXTENDS)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(498)
			p.TypeRefList()
		}

	}
	p.SetState(504)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(501)
			p.Nl()
		}

		p.SetState(506)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(507)
		p.Match(OracleParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(511)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(508)
			p.Nl()
		}

		p.SetState(513)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(514)
		p.EnumBody()
	}
	{
		p.SetState(515)
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
	p.SetState(529)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserAT || _la == OracleParserIDENT {
		p.SetState(519)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}

		switch p.GetTokenStream().LA(1) {
		case OracleParserIDENT:
			{
				p.SetState(517)
				p.EnumValue()
			}

		case OracleParserAT:
			{
				p.SetState(518)
				p.Domain()
			}

		default:
			p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
			goto errorExit
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

		p.SetState(531)
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
		p.SetState(532)
		p.Match(OracleParserIDENT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(533)
		p.Match(OracleParserEQUALS)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(534)
		_la = p.GetTokenStream().LA(1)

		if !(_la == OracleParserSTRING_LIT || _la == OracleParserINT_LIT) {
			p.GetErrorHandler().RecoverInline(p)
		} else {
			p.GetErrorHandler().ReportMatch(p)
			p.Consume()
		}
	}
	p.SetState(536)
	p.GetErrorHandler().Sync(p)

	if p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 76, p.GetParserRuleContext()) == 1 {
		{
			p.SetState(535)
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
	p.SetState(541)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(538)
			p.Nl()
		}

		p.SetState(543)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(544)
		p.Match(OracleParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(548)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(545)
			p.Nl()
		}

		p.SetState(550)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	p.SetState(560)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserAT {
		{
			p.SetState(551)
			p.Domain()
		}
		p.SetState(555)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(552)
				p.Nl()
			}

			p.SetState(557)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}

		p.SetState(562)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(563)
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
		p.SetState(565)
		p.Match(OracleParserIDENT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(567)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == OracleParserLT {
		{
			p.SetState(566)
			p.TypeParams()
		}

	}
	{
		p.SetState(569)
		p.TypeRef()
	}
	p.SetState(571)
	p.GetErrorHandler().Sync(p)

	if p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 82, p.GetParserRuleContext()) == 1 {
		{
			p.SetState(570)
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
	p.SetState(576)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(573)
			p.Nl()
		}

		p.SetState(578)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(579)
		p.Match(OracleParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(583)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(580)
			p.Nl()
		}

		p.SetState(585)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	p.SetState(595)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserAT {
		{
			p.SetState(586)
			p.Domain()
		}
		p.SetState(590)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(587)
				p.Nl()
			}

			p.SetState(592)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}

		p.SetState(597)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(598)
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
