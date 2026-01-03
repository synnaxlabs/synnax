// Copyright 2025 Synnax Labs, Inc.
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
		"", "'struct'", "'enum'", "'import'", "'extends'", "'map'", "'{'", "'}'",
		"'['", "']'", "'<'", "'>'", "','", "'?'", "'.'", "'='", "'@'", "'-'",
	}
	staticData.SymbolicNames = []string{
		"", "STRUCT", "ENUM", "IMPORT", "EXTENDS", "MAP", "LBRACE", "RBRACE",
		"LBRACKET", "RBRACKET", "LT", "GT", "COMMA", "QUESTION", "DOT", "EQUALS",
		"AT", "MINUS", "STRING_LIT", "FLOAT_LIT", "INT_LIT", "BOOL_LIT", "IDENT",
		"LINE_COMMENT", "BLOCK_COMMENT", "NEWLINE", "WS",
	}
	staticData.RuleNames = []string{
		"schema", "nl", "importStmt", "fileDomain", "definition", "structDef",
		"aliasBody", "typeParams", "typeParam", "structBody", "fieldOmit", "fieldDef",
		"inlineDomain", "fieldBody", "domain", "domainContent", "domainBlock",
		"typeRef", "mapType", "typeArgs", "typeModifiers", "qualifiedIdent",
		"expression", "expressionValue", "enumDef", "enumBody", "enumValue",
		"typeDefDef", "typeDefBody",
	}
	staticData.PredictionContextCache = antlr.NewPredictionContextCache()
	staticData.serializedATN = []int32{
		4, 1, 26, 460, 2, 0, 7, 0, 2, 1, 7, 1, 2, 2, 7, 2, 2, 3, 7, 3, 2, 4, 7,
		4, 2, 5, 7, 5, 2, 6, 7, 6, 2, 7, 7, 7, 2, 8, 7, 8, 2, 9, 7, 9, 2, 10, 7,
		10, 2, 11, 7, 11, 2, 12, 7, 12, 2, 13, 7, 13, 2, 14, 7, 14, 2, 15, 7, 15,
		2, 16, 7, 16, 2, 17, 7, 17, 2, 18, 7, 18, 2, 19, 7, 19, 2, 20, 7, 20, 2,
		21, 7, 21, 2, 22, 7, 22, 2, 23, 7, 23, 2, 24, 7, 24, 2, 25, 7, 25, 2, 26,
		7, 26, 2, 27, 7, 27, 2, 28, 7, 28, 1, 0, 5, 0, 60, 8, 0, 10, 0, 12, 0,
		63, 9, 0, 1, 0, 1, 0, 5, 0, 67, 8, 0, 10, 0, 12, 0, 70, 9, 0, 5, 0, 72,
		8, 0, 10, 0, 12, 0, 75, 9, 0, 1, 0, 1, 0, 5, 0, 79, 8, 0, 10, 0, 12, 0,
		82, 9, 0, 5, 0, 84, 8, 0, 10, 0, 12, 0, 87, 9, 0, 1, 0, 1, 0, 5, 0, 91,
		8, 0, 10, 0, 12, 0, 94, 9, 0, 5, 0, 96, 8, 0, 10, 0, 12, 0, 99, 9, 0, 1,
		0, 1, 0, 1, 1, 1, 1, 1, 2, 1, 2, 1, 2, 1, 3, 1, 3, 1, 3, 3, 3, 111, 8,
		3, 1, 4, 1, 4, 1, 4, 3, 4, 116, 8, 4, 1, 5, 1, 5, 1, 5, 3, 5, 121, 8, 5,
		1, 5, 1, 5, 3, 5, 125, 8, 5, 1, 5, 5, 5, 128, 8, 5, 10, 5, 12, 5, 131,
		9, 5, 1, 5, 1, 5, 5, 5, 135, 8, 5, 10, 5, 12, 5, 138, 9, 5, 1, 5, 1, 5,
		1, 5, 1, 5, 1, 5, 3, 5, 145, 8, 5, 1, 5, 1, 5, 1, 5, 3, 5, 150, 8, 5, 3,
		5, 152, 8, 5, 1, 6, 5, 6, 155, 8, 6, 10, 6, 12, 6, 158, 9, 6, 1, 6, 1,
		6, 5, 6, 162, 8, 6, 10, 6, 12, 6, 165, 9, 6, 1, 6, 1, 6, 5, 6, 169, 8,
		6, 10, 6, 12, 6, 172, 9, 6, 5, 6, 174, 8, 6, 10, 6, 12, 6, 177, 9, 6, 1,
		6, 1, 6, 1, 7, 1, 7, 5, 7, 183, 8, 7, 10, 7, 12, 7, 186, 9, 7, 1, 7, 1,
		7, 1, 7, 5, 7, 191, 8, 7, 10, 7, 12, 7, 194, 9, 7, 1, 7, 5, 7, 197, 8,
		7, 10, 7, 12, 7, 200, 9, 7, 1, 7, 5, 7, 203, 8, 7, 10, 7, 12, 7, 206, 9,
		7, 1, 7, 1, 7, 1, 8, 1, 8, 3, 8, 212, 8, 8, 1, 8, 1, 8, 3, 8, 216, 8, 8,
		1, 8, 1, 8, 3, 8, 220, 8, 8, 1, 9, 1, 9, 1, 9, 3, 9, 225, 8, 9, 1, 9, 5,
		9, 228, 8, 9, 10, 9, 12, 9, 231, 9, 9, 5, 9, 233, 8, 9, 10, 9, 12, 9, 236,
		9, 9, 1, 10, 1, 10, 1, 10, 1, 11, 1, 11, 1, 11, 5, 11, 244, 8, 11, 10,
		11, 12, 11, 247, 9, 11, 1, 11, 3, 11, 250, 8, 11, 1, 12, 1, 12, 1, 12,
		3, 12, 255, 8, 12, 1, 13, 5, 13, 258, 8, 13, 10, 13, 12, 13, 261, 9, 13,
		1, 13, 1, 13, 5, 13, 265, 8, 13, 10, 13, 12, 13, 268, 9, 13, 1, 13, 1,
		13, 5, 13, 272, 8, 13, 10, 13, 12, 13, 275, 9, 13, 5, 13, 277, 8, 13, 10,
		13, 12, 13, 280, 9, 13, 1, 13, 1, 13, 1, 14, 1, 14, 1, 14, 3, 14, 287,
		8, 14, 1, 15, 1, 15, 3, 15, 291, 8, 15, 1, 16, 5, 16, 294, 8, 16, 10, 16,
		12, 16, 297, 9, 16, 1, 16, 1, 16, 5, 16, 301, 8, 16, 10, 16, 12, 16, 304,
		9, 16, 1, 16, 1, 16, 4, 16, 308, 8, 16, 11, 16, 12, 16, 309, 1, 16, 1,
		16, 5, 16, 314, 8, 16, 10, 16, 12, 16, 317, 9, 16, 3, 16, 319, 8, 16, 1,
		16, 5, 16, 322, 8, 16, 10, 16, 12, 16, 325, 9, 16, 1, 16, 1, 16, 1, 17,
		1, 17, 3, 17, 331, 8, 17, 1, 17, 1, 17, 3, 17, 335, 8, 17, 1, 17, 1, 17,
		3, 17, 339, 8, 17, 1, 17, 3, 17, 342, 8, 17, 3, 17, 344, 8, 17, 1, 18,
		1, 18, 1, 18, 1, 18, 1, 18, 1, 18, 1, 18, 1, 19, 1, 19, 1, 19, 1, 19, 5,
		19, 357, 8, 19, 10, 19, 12, 19, 360, 9, 19, 1, 19, 1, 19, 1, 20, 1, 20,
		1, 20, 3, 20, 367, 8, 20, 1, 21, 1, 21, 1, 21, 3, 21, 372, 8, 21, 1, 22,
		1, 22, 5, 22, 376, 8, 22, 10, 22, 12, 22, 379, 9, 22, 1, 23, 1, 23, 1,
		23, 1, 23, 1, 23, 3, 23, 386, 8, 23, 1, 24, 1, 24, 1, 24, 5, 24, 391, 8,
		24, 10, 24, 12, 24, 394, 9, 24, 1, 24, 1, 24, 5, 24, 398, 8, 24, 10, 24,
		12, 24, 401, 9, 24, 1, 24, 1, 24, 1, 24, 1, 25, 1, 25, 3, 25, 408, 8, 25,
		1, 25, 5, 25, 411, 8, 25, 10, 25, 12, 25, 414, 9, 25, 5, 25, 416, 8, 25,
		10, 25, 12, 25, 419, 9, 25, 1, 26, 1, 26, 1, 26, 1, 26, 1, 27, 1, 27, 3,
		27, 427, 8, 27, 1, 27, 1, 27, 3, 27, 431, 8, 27, 1, 28, 5, 28, 434, 8,
		28, 10, 28, 12, 28, 437, 9, 28, 1, 28, 1, 28, 5, 28, 441, 8, 28, 10, 28,
		12, 28, 444, 9, 28, 1, 28, 1, 28, 5, 28, 448, 8, 28, 10, 28, 12, 28, 451,
		9, 28, 5, 28, 453, 8, 28, 10, 28, 12, 28, 456, 9, 28, 1, 28, 1, 28, 1,
		28, 0, 0, 29, 0, 2, 4, 6, 8, 10, 12, 14, 16, 18, 20, 22, 24, 26, 28, 30,
		32, 34, 36, 38, 40, 42, 44, 46, 48, 50, 52, 54, 56, 0, 1, 2, 0, 18, 18,
		20, 20, 501, 0, 61, 1, 0, 0, 0, 2, 102, 1, 0, 0, 0, 4, 104, 1, 0, 0, 0,
		6, 107, 1, 0, 0, 0, 8, 115, 1, 0, 0, 0, 10, 151, 1, 0, 0, 0, 12, 156, 1,
		0, 0, 0, 14, 180, 1, 0, 0, 0, 16, 209, 1, 0, 0, 0, 18, 234, 1, 0, 0, 0,
		20, 237, 1, 0, 0, 0, 22, 240, 1, 0, 0, 0, 24, 251, 1, 0, 0, 0, 26, 259,
		1, 0, 0, 0, 28, 283, 1, 0, 0, 0, 30, 290, 1, 0, 0, 0, 32, 295, 1, 0, 0,
		0, 34, 343, 1, 0, 0, 0, 36, 345, 1, 0, 0, 0, 38, 352, 1, 0, 0, 0, 40, 366,
		1, 0, 0, 0, 42, 368, 1, 0, 0, 0, 44, 373, 1, 0, 0, 0, 46, 385, 1, 0, 0,
		0, 48, 387, 1, 0, 0, 0, 50, 417, 1, 0, 0, 0, 52, 420, 1, 0, 0, 0, 54, 424,
		1, 0, 0, 0, 56, 435, 1, 0, 0, 0, 58, 60, 3, 2, 1, 0, 59, 58, 1, 0, 0, 0,
		60, 63, 1, 0, 0, 0, 61, 59, 1, 0, 0, 0, 61, 62, 1, 0, 0, 0, 62, 73, 1,
		0, 0, 0, 63, 61, 1, 0, 0, 0, 64, 68, 3, 4, 2, 0, 65, 67, 3, 2, 1, 0, 66,
		65, 1, 0, 0, 0, 67, 70, 1, 0, 0, 0, 68, 66, 1, 0, 0, 0, 68, 69, 1, 0, 0,
		0, 69, 72, 1, 0, 0, 0, 70, 68, 1, 0, 0, 0, 71, 64, 1, 0, 0, 0, 72, 75,
		1, 0, 0, 0, 73, 71, 1, 0, 0, 0, 73, 74, 1, 0, 0, 0, 74, 85, 1, 0, 0, 0,
		75, 73, 1, 0, 0, 0, 76, 80, 3, 6, 3, 0, 77, 79, 3, 2, 1, 0, 78, 77, 1,
		0, 0, 0, 79, 82, 1, 0, 0, 0, 80, 78, 1, 0, 0, 0, 80, 81, 1, 0, 0, 0, 81,
		84, 1, 0, 0, 0, 82, 80, 1, 0, 0, 0, 83, 76, 1, 0, 0, 0, 84, 87, 1, 0, 0,
		0, 85, 83, 1, 0, 0, 0, 85, 86, 1, 0, 0, 0, 86, 97, 1, 0, 0, 0, 87, 85,
		1, 0, 0, 0, 88, 92, 3, 8, 4, 0, 89, 91, 3, 2, 1, 0, 90, 89, 1, 0, 0, 0,
		91, 94, 1, 0, 0, 0, 92, 90, 1, 0, 0, 0, 92, 93, 1, 0, 0, 0, 93, 96, 1,
		0, 0, 0, 94, 92, 1, 0, 0, 0, 95, 88, 1, 0, 0, 0, 96, 99, 1, 0, 0, 0, 97,
		95, 1, 0, 0, 0, 97, 98, 1, 0, 0, 0, 98, 100, 1, 0, 0, 0, 99, 97, 1, 0,
		0, 0, 100, 101, 5, 0, 0, 1, 101, 1, 1, 0, 0, 0, 102, 103, 5, 25, 0, 0,
		103, 3, 1, 0, 0, 0, 104, 105, 5, 3, 0, 0, 105, 106, 5, 18, 0, 0, 106, 5,
		1, 0, 0, 0, 107, 108, 5, 16, 0, 0, 108, 110, 5, 22, 0, 0, 109, 111, 3,
		30, 15, 0, 110, 109, 1, 0, 0, 0, 110, 111, 1, 0, 0, 0, 111, 7, 1, 0, 0,
		0, 112, 116, 3, 10, 5, 0, 113, 116, 3, 48, 24, 0, 114, 116, 3, 54, 27,
		0, 115, 112, 1, 0, 0, 0, 115, 113, 1, 0, 0, 0, 115, 114, 1, 0, 0, 0, 116,
		9, 1, 0, 0, 0, 117, 118, 5, 22, 0, 0, 118, 120, 5, 1, 0, 0, 119, 121, 3,
		14, 7, 0, 120, 119, 1, 0, 0, 0, 120, 121, 1, 0, 0, 0, 121, 124, 1, 0, 0,
		0, 122, 123, 5, 4, 0, 0, 123, 125, 3, 34, 17, 0, 124, 122, 1, 0, 0, 0,
		124, 125, 1, 0, 0, 0, 125, 129, 1, 0, 0, 0, 126, 128, 3, 2, 1, 0, 127,
		126, 1, 0, 0, 0, 128, 131, 1, 0, 0, 0, 129, 127, 1, 0, 0, 0, 129, 130,
		1, 0, 0, 0, 130, 132, 1, 0, 0, 0, 131, 129, 1, 0, 0, 0, 132, 136, 5, 6,
		0, 0, 133, 135, 3, 2, 1, 0, 134, 133, 1, 0, 0, 0, 135, 138, 1, 0, 0, 0,
		136, 134, 1, 0, 0, 0, 136, 137, 1, 0, 0, 0, 137, 139, 1, 0, 0, 0, 138,
		136, 1, 0, 0, 0, 139, 140, 3, 18, 9, 0, 140, 141, 5, 7, 0, 0, 141, 152,
		1, 0, 0, 0, 142, 144, 5, 22, 0, 0, 143, 145, 3, 14, 7, 0, 144, 143, 1,
		0, 0, 0, 144, 145, 1, 0, 0, 0, 145, 146, 1, 0, 0, 0, 146, 147, 5, 15, 0,
		0, 147, 149, 3, 34, 17, 0, 148, 150, 3, 12, 6, 0, 149, 148, 1, 0, 0, 0,
		149, 150, 1, 0, 0, 0, 150, 152, 1, 0, 0, 0, 151, 117, 1, 0, 0, 0, 151,
		142, 1, 0, 0, 0, 152, 11, 1, 0, 0, 0, 153, 155, 3, 2, 1, 0, 154, 153, 1,
		0, 0, 0, 155, 158, 1, 0, 0, 0, 156, 154, 1, 0, 0, 0, 156, 157, 1, 0, 0,
		0, 157, 159, 1, 0, 0, 0, 158, 156, 1, 0, 0, 0, 159, 163, 5, 6, 0, 0, 160,
		162, 3, 2, 1, 0, 161, 160, 1, 0, 0, 0, 162, 165, 1, 0, 0, 0, 163, 161,
		1, 0, 0, 0, 163, 164, 1, 0, 0, 0, 164, 175, 1, 0, 0, 0, 165, 163, 1, 0,
		0, 0, 166, 170, 3, 28, 14, 0, 167, 169, 3, 2, 1, 0, 168, 167, 1, 0, 0,
		0, 169, 172, 1, 0, 0, 0, 170, 168, 1, 0, 0, 0, 170, 171, 1, 0, 0, 0, 171,
		174, 1, 0, 0, 0, 172, 170, 1, 0, 0, 0, 173, 166, 1, 0, 0, 0, 174, 177,
		1, 0, 0, 0, 175, 173, 1, 0, 0, 0, 175, 176, 1, 0, 0, 0, 176, 178, 1, 0,
		0, 0, 177, 175, 1, 0, 0, 0, 178, 179, 5, 7, 0, 0, 179, 13, 1, 0, 0, 0,
		180, 184, 5, 10, 0, 0, 181, 183, 3, 2, 1, 0, 182, 181, 1, 0, 0, 0, 183,
		186, 1, 0, 0, 0, 184, 182, 1, 0, 0, 0, 184, 185, 1, 0, 0, 0, 185, 187,
		1, 0, 0, 0, 186, 184, 1, 0, 0, 0, 187, 198, 3, 16, 8, 0, 188, 192, 5, 12,
		0, 0, 189, 191, 3, 2, 1, 0, 190, 189, 1, 0, 0, 0, 191, 194, 1, 0, 0, 0,
		192, 190, 1, 0, 0, 0, 192, 193, 1, 0, 0, 0, 193, 195, 1, 0, 0, 0, 194,
		192, 1, 0, 0, 0, 195, 197, 3, 16, 8, 0, 196, 188, 1, 0, 0, 0, 197, 200,
		1, 0, 0, 0, 198, 196, 1, 0, 0, 0, 198, 199, 1, 0, 0, 0, 199, 204, 1, 0,
		0, 0, 200, 198, 1, 0, 0, 0, 201, 203, 3, 2, 1, 0, 202, 201, 1, 0, 0, 0,
		203, 206, 1, 0, 0, 0, 204, 202, 1, 0, 0, 0, 204, 205, 1, 0, 0, 0, 205,
		207, 1, 0, 0, 0, 206, 204, 1, 0, 0, 0, 207, 208, 5, 11, 0, 0, 208, 15,
		1, 0, 0, 0, 209, 211, 5, 22, 0, 0, 210, 212, 5, 13, 0, 0, 211, 210, 1,
		0, 0, 0, 211, 212, 1, 0, 0, 0, 212, 215, 1, 0, 0, 0, 213, 214, 5, 4, 0,
		0, 214, 216, 3, 34, 17, 0, 215, 213, 1, 0, 0, 0, 215, 216, 1, 0, 0, 0,
		216, 219, 1, 0, 0, 0, 217, 218, 5, 15, 0, 0, 218, 220, 3, 34, 17, 0, 219,
		217, 1, 0, 0, 0, 219, 220, 1, 0, 0, 0, 220, 17, 1, 0, 0, 0, 221, 225, 3,
		22, 11, 0, 222, 225, 3, 20, 10, 0, 223, 225, 3, 28, 14, 0, 224, 221, 1,
		0, 0, 0, 224, 222, 1, 0, 0, 0, 224, 223, 1, 0, 0, 0, 225, 229, 1, 0, 0,
		0, 226, 228, 3, 2, 1, 0, 227, 226, 1, 0, 0, 0, 228, 231, 1, 0, 0, 0, 229,
		227, 1, 0, 0, 0, 229, 230, 1, 0, 0, 0, 230, 233, 1, 0, 0, 0, 231, 229,
		1, 0, 0, 0, 232, 224, 1, 0, 0, 0, 233, 236, 1, 0, 0, 0, 234, 232, 1, 0,
		0, 0, 234, 235, 1, 0, 0, 0, 235, 19, 1, 0, 0, 0, 236, 234, 1, 0, 0, 0,
		237, 238, 5, 17, 0, 0, 238, 239, 5, 22, 0, 0, 239, 21, 1, 0, 0, 0, 240,
		241, 5, 22, 0, 0, 241, 245, 3, 34, 17, 0, 242, 244, 3, 24, 12, 0, 243,
		242, 1, 0, 0, 0, 244, 247, 1, 0, 0, 0, 245, 243, 1, 0, 0, 0, 245, 246,
		1, 0, 0, 0, 246, 249, 1, 0, 0, 0, 247, 245, 1, 0, 0, 0, 248, 250, 3, 26,
		13, 0, 249, 248, 1, 0, 0, 0, 249, 250, 1, 0, 0, 0, 250, 23, 1, 0, 0, 0,
		251, 252, 5, 16, 0, 0, 252, 254, 5, 22, 0, 0, 253, 255, 3, 30, 15, 0, 254,
		253, 1, 0, 0, 0, 254, 255, 1, 0, 0, 0, 255, 25, 1, 0, 0, 0, 256, 258, 3,
		2, 1, 0, 257, 256, 1, 0, 0, 0, 258, 261, 1, 0, 0, 0, 259, 257, 1, 0, 0,
		0, 259, 260, 1, 0, 0, 0, 260, 262, 1, 0, 0, 0, 261, 259, 1, 0, 0, 0, 262,
		266, 5, 6, 0, 0, 263, 265, 3, 2, 1, 0, 264, 263, 1, 0, 0, 0, 265, 268,
		1, 0, 0, 0, 266, 264, 1, 0, 0, 0, 266, 267, 1, 0, 0, 0, 267, 278, 1, 0,
		0, 0, 268, 266, 1, 0, 0, 0, 269, 273, 3, 28, 14, 0, 270, 272, 3, 2, 1,
		0, 271, 270, 1, 0, 0, 0, 272, 275, 1, 0, 0, 0, 273, 271, 1, 0, 0, 0, 273,
		274, 1, 0, 0, 0, 274, 277, 1, 0, 0, 0, 275, 273, 1, 0, 0, 0, 276, 269,
		1, 0, 0, 0, 277, 280, 1, 0, 0, 0, 278, 276, 1, 0, 0, 0, 278, 279, 1, 0,
		0, 0, 279, 281, 1, 0, 0, 0, 280, 278, 1, 0, 0, 0, 281, 282, 5, 7, 0, 0,
		282, 27, 1, 0, 0, 0, 283, 284, 5, 16, 0, 0, 284, 286, 5, 22, 0, 0, 285,
		287, 3, 30, 15, 0, 286, 285, 1, 0, 0, 0, 286, 287, 1, 0, 0, 0, 287, 29,
		1, 0, 0, 0, 288, 291, 3, 32, 16, 0, 289, 291, 3, 44, 22, 0, 290, 288, 1,
		0, 0, 0, 290, 289, 1, 0, 0, 0, 291, 31, 1, 0, 0, 0, 292, 294, 3, 2, 1,
		0, 293, 292, 1, 0, 0, 0, 294, 297, 1, 0, 0, 0, 295, 293, 1, 0, 0, 0, 295,
		296, 1, 0, 0, 0, 296, 298, 1, 0, 0, 0, 297, 295, 1, 0, 0, 0, 298, 302,
		5, 6, 0, 0, 299, 301, 3, 2, 1, 0, 300, 299, 1, 0, 0, 0, 301, 304, 1, 0,
		0, 0, 302, 300, 1, 0, 0, 0, 302, 303, 1, 0, 0, 0, 303, 318, 1, 0, 0, 0,
		304, 302, 1, 0, 0, 0, 305, 315, 3, 44, 22, 0, 306, 308, 3, 2, 1, 0, 307,
		306, 1, 0, 0, 0, 308, 309, 1, 0, 0, 0, 309, 307, 1, 0, 0, 0, 309, 310,
		1, 0, 0, 0, 310, 311, 1, 0, 0, 0, 311, 312, 3, 44, 22, 0, 312, 314, 1,
		0, 0, 0, 313, 307, 1, 0, 0, 0, 314, 317, 1, 0, 0, 0, 315, 313, 1, 0, 0,
		0, 315, 316, 1, 0, 0, 0, 316, 319, 1, 0, 0, 0, 317, 315, 1, 0, 0, 0, 318,
		305, 1, 0, 0, 0, 318, 319, 1, 0, 0, 0, 319, 323, 1, 0, 0, 0, 320, 322,
		3, 2, 1, 0, 321, 320, 1, 0, 0, 0, 322, 325, 1, 0, 0, 0, 323, 321, 1, 0,
		0, 0, 323, 324, 1, 0, 0, 0, 324, 326, 1, 0, 0, 0, 325, 323, 1, 0, 0, 0,
		326, 327, 5, 7, 0, 0, 327, 33, 1, 0, 0, 0, 328, 330, 3, 36, 18, 0, 329,
		331, 3, 40, 20, 0, 330, 329, 1, 0, 0, 0, 330, 331, 1, 0, 0, 0, 331, 344,
		1, 0, 0, 0, 332, 334, 3, 42, 21, 0, 333, 335, 3, 38, 19, 0, 334, 333, 1,
		0, 0, 0, 334, 335, 1, 0, 0, 0, 335, 338, 1, 0, 0, 0, 336, 337, 5, 8, 0,
		0, 337, 339, 5, 9, 0, 0, 338, 336, 1, 0, 0, 0, 338, 339, 1, 0, 0, 0, 339,
		341, 1, 0, 0, 0, 340, 342, 3, 40, 20, 0, 341, 340, 1, 0, 0, 0, 341, 342,
		1, 0, 0, 0, 342, 344, 1, 0, 0, 0, 343, 328, 1, 0, 0, 0, 343, 332, 1, 0,
		0, 0, 344, 35, 1, 0, 0, 0, 345, 346, 5, 5, 0, 0, 346, 347, 5, 10, 0, 0,
		347, 348, 3, 34, 17, 0, 348, 349, 5, 12, 0, 0, 349, 350, 3, 34, 17, 0,
		350, 351, 5, 11, 0, 0, 351, 37, 1, 0, 0, 0, 352, 353, 5, 10, 0, 0, 353,
		358, 3, 34, 17, 0, 354, 355, 5, 12, 0, 0, 355, 357, 3, 34, 17, 0, 356,
		354, 1, 0, 0, 0, 357, 360, 1, 0, 0, 0, 358, 356, 1, 0, 0, 0, 358, 359,
		1, 0, 0, 0, 359, 361, 1, 0, 0, 0, 360, 358, 1, 0, 0, 0, 361, 362, 5, 11,
		0, 0, 362, 39, 1, 0, 0, 0, 363, 364, 5, 13, 0, 0, 364, 367, 5, 13, 0, 0,
		365, 367, 5, 13, 0, 0, 366, 363, 1, 0, 0, 0, 366, 365, 1, 0, 0, 0, 367,
		41, 1, 0, 0, 0, 368, 371, 5, 22, 0, 0, 369, 370, 5, 14, 0, 0, 370, 372,
		5, 22, 0, 0, 371, 369, 1, 0, 0, 0, 371, 372, 1, 0, 0, 0, 372, 43, 1, 0,
		0, 0, 373, 377, 5, 22, 0, 0, 374, 376, 3, 46, 23, 0, 375, 374, 1, 0, 0,
		0, 376, 379, 1, 0, 0, 0, 377, 375, 1, 0, 0, 0, 377, 378, 1, 0, 0, 0, 378,
		45, 1, 0, 0, 0, 379, 377, 1, 0, 0, 0, 380, 386, 5, 18, 0, 0, 381, 386,
		5, 20, 0, 0, 382, 386, 5, 19, 0, 0, 383, 386, 5, 21, 0, 0, 384, 386, 3,
		42, 21, 0, 385, 380, 1, 0, 0, 0, 385, 381, 1, 0, 0, 0, 385, 382, 1, 0,
		0, 0, 385, 383, 1, 0, 0, 0, 385, 384, 1, 0, 0, 0, 386, 47, 1, 0, 0, 0,
		387, 388, 5, 22, 0, 0, 388, 392, 5, 2, 0, 0, 389, 391, 3, 2, 1, 0, 390,
		389, 1, 0, 0, 0, 391, 394, 1, 0, 0, 0, 392, 390, 1, 0, 0, 0, 392, 393,
		1, 0, 0, 0, 393, 395, 1, 0, 0, 0, 394, 392, 1, 0, 0, 0, 395, 399, 5, 6,
		0, 0, 396, 398, 3, 2, 1, 0, 397, 396, 1, 0, 0, 0, 398, 401, 1, 0, 0, 0,
		399, 397, 1, 0, 0, 0, 399, 400, 1, 0, 0, 0, 400, 402, 1, 0, 0, 0, 401,
		399, 1, 0, 0, 0, 402, 403, 3, 50, 25, 0, 403, 404, 5, 7, 0, 0, 404, 49,
		1, 0, 0, 0, 405, 408, 3, 52, 26, 0, 406, 408, 3, 28, 14, 0, 407, 405, 1,
		0, 0, 0, 407, 406, 1, 0, 0, 0, 408, 412, 1, 0, 0, 0, 409, 411, 3, 2, 1,
		0, 410, 409, 1, 0, 0, 0, 411, 414, 1, 0, 0, 0, 412, 410, 1, 0, 0, 0, 412,
		413, 1, 0, 0, 0, 413, 416, 1, 0, 0, 0, 414, 412, 1, 0, 0, 0, 415, 407,
		1, 0, 0, 0, 416, 419, 1, 0, 0, 0, 417, 415, 1, 0, 0, 0, 417, 418, 1, 0,
		0, 0, 418, 51, 1, 0, 0, 0, 419, 417, 1, 0, 0, 0, 420, 421, 5, 22, 0, 0,
		421, 422, 5, 15, 0, 0, 422, 423, 7, 0, 0, 0, 423, 53, 1, 0, 0, 0, 424,
		426, 5, 22, 0, 0, 425, 427, 3, 14, 7, 0, 426, 425, 1, 0, 0, 0, 426, 427,
		1, 0, 0, 0, 427, 428, 1, 0, 0, 0, 428, 430, 3, 34, 17, 0, 429, 431, 3,
		56, 28, 0, 430, 429, 1, 0, 0, 0, 430, 431, 1, 0, 0, 0, 431, 55, 1, 0, 0,
		0, 432, 434, 3, 2, 1, 0, 433, 432, 1, 0, 0, 0, 434, 437, 1, 0, 0, 0, 435,
		433, 1, 0, 0, 0, 435, 436, 1, 0, 0, 0, 436, 438, 1, 0, 0, 0, 437, 435,
		1, 0, 0, 0, 438, 442, 5, 6, 0, 0, 439, 441, 3, 2, 1, 0, 440, 439, 1, 0,
		0, 0, 441, 444, 1, 0, 0, 0, 442, 440, 1, 0, 0, 0, 442, 443, 1, 0, 0, 0,
		443, 454, 1, 0, 0, 0, 444, 442, 1, 0, 0, 0, 445, 449, 3, 28, 14, 0, 446,
		448, 3, 2, 1, 0, 447, 446, 1, 0, 0, 0, 448, 451, 1, 0, 0, 0, 449, 447,
		1, 0, 0, 0, 449, 450, 1, 0, 0, 0, 450, 453, 1, 0, 0, 0, 451, 449, 1, 0,
		0, 0, 452, 445, 1, 0, 0, 0, 453, 456, 1, 0, 0, 0, 454, 452, 1, 0, 0, 0,
		454, 455, 1, 0, 0, 0, 455, 457, 1, 0, 0, 0, 456, 454, 1, 0, 0, 0, 457,
		458, 5, 7, 0, 0, 458, 57, 1, 0, 0, 0, 66, 61, 68, 73, 80, 85, 92, 97, 110,
		115, 120, 124, 129, 136, 144, 149, 151, 156, 163, 170, 175, 184, 192, 198,
		204, 211, 215, 219, 224, 229, 234, 245, 249, 254, 259, 266, 273, 278, 286,
		290, 295, 302, 309, 315, 318, 323, 330, 334, 338, 341, 343, 358, 366, 371,
		377, 385, 392, 399, 407, 412, 417, 426, 430, 435, 442, 449, 454,
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
	OracleParserEOF           = antlr.TokenEOF
	OracleParserSTRUCT        = 1
	OracleParserENUM          = 2
	OracleParserIMPORT        = 3
	OracleParserEXTENDS       = 4
	OracleParserMAP           = 5
	OracleParserLBRACE        = 6
	OracleParserRBRACE        = 7
	OracleParserLBRACKET      = 8
	OracleParserRBRACKET      = 9
	OracleParserLT            = 10
	OracleParserGT            = 11
	OracleParserCOMMA         = 12
	OracleParserQUESTION      = 13
	OracleParserDOT           = 14
	OracleParserEQUALS        = 15
	OracleParserAT            = 16
	OracleParserMINUS         = 17
	OracleParserSTRING_LIT    = 18
	OracleParserFLOAT_LIT     = 19
	OracleParserINT_LIT       = 20
	OracleParserBOOL_LIT      = 21
	OracleParserIDENT         = 22
	OracleParserLINE_COMMENT  = 23
	OracleParserBLOCK_COMMENT = 24
	OracleParserNEWLINE       = 25
	OracleParserWS            = 26
)

// OracleParser rules.
const (
	OracleParserRULE_schema          = 0
	OracleParserRULE_nl              = 1
	OracleParserRULE_importStmt      = 2
	OracleParserRULE_fileDomain      = 3
	OracleParserRULE_definition      = 4
	OracleParserRULE_structDef       = 5
	OracleParserRULE_aliasBody       = 6
	OracleParserRULE_typeParams      = 7
	OracleParserRULE_typeParam       = 8
	OracleParserRULE_structBody      = 9
	OracleParserRULE_fieldOmit       = 10
	OracleParserRULE_fieldDef        = 11
	OracleParserRULE_inlineDomain    = 12
	OracleParserRULE_fieldBody       = 13
	OracleParserRULE_domain          = 14
	OracleParserRULE_domainContent   = 15
	OracleParserRULE_domainBlock     = 16
	OracleParserRULE_typeRef         = 17
	OracleParserRULE_mapType         = 18
	OracleParserRULE_typeArgs        = 19
	OracleParserRULE_typeModifiers   = 20
	OracleParserRULE_qualifiedIdent  = 21
	OracleParserRULE_expression      = 22
	OracleParserRULE_expressionValue = 23
	OracleParserRULE_enumDef         = 24
	OracleParserRULE_enumBody        = 25
	OracleParserRULE_enumValue       = 26
	OracleParserRULE_typeDefDef      = 27
	OracleParserRULE_typeDefBody     = 28
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
	p.SetState(61)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(58)
			p.Nl()
		}

		p.SetState(63)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	p.SetState(73)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserIMPORT {
		{
			p.SetState(64)
			p.ImportStmt()
		}
		p.SetState(68)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(65)
				p.Nl()
			}

			p.SetState(70)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}

		p.SetState(75)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	p.SetState(85)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserAT {
		{
			p.SetState(76)
			p.FileDomain()
		}
		p.SetState(80)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(77)
				p.Nl()
			}

			p.SetState(82)
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
	}
	p.SetState(97)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserIDENT {
		{
			p.SetState(88)
			p.Definition()
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
	{
		p.SetState(100)
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
		p.SetState(102)
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
		p.SetState(104)
		p.Match(OracleParserIMPORT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(105)
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
		p.SetState(107)
		p.Match(OracleParserAT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(108)
		p.Match(OracleParserIDENT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(110)
	p.GetErrorHandler().Sync(p)

	if p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 7, p.GetParserRuleContext()) == 1 {
		{
			p.SetState(109)
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
	p.SetState(115)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 8, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(112)
			p.StructDef()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(113)
			p.EnumDef()
		}

	case 3:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(114)
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

func (s *StructFullContext) TypeRef() ITypeRefContext {
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

	p.SetState(151)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 15, p.GetParserRuleContext()) {
	case 1:
		localctx = NewStructFullContext(p, localctx)
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(117)
			p.Match(OracleParserIDENT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(118)
			p.Match(OracleParserSTRUCT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		p.SetState(120)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		if _la == OracleParserLT {
			{
				p.SetState(119)
				p.TypeParams()
			}

		}
		p.SetState(124)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		if _la == OracleParserEXTENDS {
			{
				p.SetState(122)
				p.Match(OracleParserEXTENDS)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}
			{
				p.SetState(123)
				p.TypeRef()
			}

		}
		p.SetState(129)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(126)
				p.Nl()
			}

			p.SetState(131)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}
		{
			p.SetState(132)
			p.Match(OracleParserLBRACE)
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

		for _la == OracleParserNEWLINE {
			{
				p.SetState(133)
				p.Nl()
			}

			p.SetState(138)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}
		{
			p.SetState(139)
			p.StructBody()
		}
		{
			p.SetState(140)
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
			p.SetState(142)
			p.Match(OracleParserIDENT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		p.SetState(144)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		if _la == OracleParserLT {
			{
				p.SetState(143)
				p.TypeParams()
			}

		}
		{
			p.SetState(146)
			p.Match(OracleParserEQUALS)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(147)
			p.TypeRef()
		}
		p.SetState(149)
		p.GetErrorHandler().Sync(p)

		if p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 14, p.GetParserRuleContext()) == 1 {
			{
				p.SetState(148)
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
	p.EnterRule(localctx, 12, OracleParserRULE_aliasBody)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(156)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(153)
			p.Nl()
		}

		p.SetState(158)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(159)
		p.Match(OracleParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(163)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(160)
			p.Nl()
		}

		p.SetState(165)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	p.SetState(175)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserAT {
		{
			p.SetState(166)
			p.Domain()
		}
		p.SetState(170)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(167)
				p.Nl()
			}

			p.SetState(172)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}

		p.SetState(177)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(178)
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
	p.EnterRule(localctx, 14, OracleParserRULE_typeParams)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(180)
		p.Match(OracleParserLT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
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
		p.TypeParam()
	}
	p.SetState(198)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserCOMMA {
		{
			p.SetState(188)
			p.Match(OracleParserCOMMA)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		p.SetState(192)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(189)
				p.Nl()
			}

			p.SetState(194)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}
		{
			p.SetState(195)
			p.TypeParam()
		}

		p.SetState(200)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	p.SetState(204)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(201)
			p.Nl()
		}

		p.SetState(206)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(207)
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
	p.EnterRule(localctx, 16, OracleParserRULE_typeParam)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(209)
		p.Match(OracleParserIDENT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(211)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == OracleParserQUESTION {
		{
			p.SetState(210)
			p.Match(OracleParserQUESTION)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	}
	p.SetState(215)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == OracleParserEXTENDS {
		{
			p.SetState(213)
			p.Match(OracleParserEXTENDS)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(214)
			p.TypeRef()
		}

	}
	p.SetState(219)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == OracleParserEQUALS {
		{
			p.SetState(217)
			p.Match(OracleParserEQUALS)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(218)
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
	p.EnterRule(localctx, 18, OracleParserRULE_structBody)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(234)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&4390912) != 0 {
		p.SetState(224)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}

		switch p.GetTokenStream().LA(1) {
		case OracleParserIDENT:
			{
				p.SetState(221)
				p.FieldDef()
			}

		case OracleParserMINUS:
			{
				p.SetState(222)
				p.FieldOmit()
			}

		case OracleParserAT:
			{
				p.SetState(223)
				p.Domain()
			}

		default:
			p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
			goto errorExit
		}
		p.SetState(229)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(226)
				p.Nl()
			}

			p.SetState(231)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}

		p.SetState(236)
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
	p.EnterRule(localctx, 20, OracleParserRULE_fieldOmit)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(237)
		p.Match(OracleParserMINUS)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(238)
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

// IFieldDefContext is an interface to support dynamic dispatch.
type IFieldDefContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	IDENT() antlr.TerminalNode
	TypeRef() ITypeRefContext
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
	p.EnterRule(localctx, 22, OracleParserRULE_fieldDef)
	var _alt int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(240)
		p.Match(OracleParserIDENT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(241)
		p.TypeRef()
	}
	p.SetState(245)
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
				p.SetState(242)
				p.InlineDomain()
			}

		}
		p.SetState(247)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 30, p.GetParserRuleContext())
		if p.HasError() {
			goto errorExit
		}
	}
	p.SetState(249)
	p.GetErrorHandler().Sync(p)

	if p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 31, p.GetParserRuleContext()) == 1 {
		{
			p.SetState(248)
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
	p.EnterRule(localctx, 24, OracleParserRULE_inlineDomain)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(251)
		p.Match(OracleParserAT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(252)
		p.Match(OracleParserIDENT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(254)
	p.GetErrorHandler().Sync(p)

	if p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 32, p.GetParserRuleContext()) == 1 {
		{
			p.SetState(253)
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
	p.EnterRule(localctx, 26, OracleParserRULE_fieldBody)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(259)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(256)
			p.Nl()
		}

		p.SetState(261)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(262)
		p.Match(OracleParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(266)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(263)
			p.Nl()
		}

		p.SetState(268)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	p.SetState(278)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserAT {
		{
			p.SetState(269)
			p.Domain()
		}
		p.SetState(273)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(270)
				p.Nl()
			}

			p.SetState(275)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}

		p.SetState(280)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(281)
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
	p.EnterRule(localctx, 28, OracleParserRULE_domain)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(283)
		p.Match(OracleParserAT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(284)
		p.Match(OracleParserIDENT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(286)
	p.GetErrorHandler().Sync(p)

	if p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 37, p.GetParserRuleContext()) == 1 {
		{
			p.SetState(285)
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
	p.EnterRule(localctx, 30, OracleParserRULE_domainContent)
	p.SetState(290)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case OracleParserLBRACE, OracleParserNEWLINE:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(288)
			p.DomainBlock()
		}

	case OracleParserIDENT:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(289)
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
	p.EnterRule(localctx, 32, OracleParserRULE_domainBlock)
	var _la int

	var _alt int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(295)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(292)
			p.Nl()
		}

		p.SetState(297)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(298)
		p.Match(OracleParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(302)
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
				p.SetState(299)
				p.Nl()
			}

		}
		p.SetState(304)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 40, p.GetParserRuleContext())
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

	if _la == OracleParserIDENT {
		{
			p.SetState(305)
			p.Expression()
		}
		p.SetState(315)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 42, p.GetParserRuleContext())
		if p.HasError() {
			goto errorExit
		}
		for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
			if _alt == 1 {
				p.SetState(307)
				p.GetErrorHandler().Sync(p)
				if p.HasError() {
					goto errorExit
				}
				_la = p.GetTokenStream().LA(1)

				for ok := true; ok; ok = _la == OracleParserNEWLINE {
					{
						p.SetState(306)
						p.Nl()
					}

					p.SetState(309)
					p.GetErrorHandler().Sync(p)
					if p.HasError() {
						goto errorExit
					}
					_la = p.GetTokenStream().LA(1)
				}
				{
					p.SetState(311)
					p.Expression()
				}

			}
			p.SetState(317)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 42, p.GetParserRuleContext())
			if p.HasError() {
				goto errorExit
			}
		}

	}
	p.SetState(323)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(320)
			p.Nl()
		}

		p.SetState(325)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(326)
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

func (s *TypeRefNormalContext) LBRACKET() antlr.TerminalNode {
	return s.GetToken(OracleParserLBRACKET, 0)
}

func (s *TypeRefNormalContext) RBRACKET() antlr.TerminalNode {
	return s.GetToken(OracleParserRBRACKET, 0)
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
	p.EnterRule(localctx, 34, OracleParserRULE_typeRef)
	var _la int

	p.SetState(343)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case OracleParserMAP:
		localctx = NewTypeRefMapContext(p, localctx)
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(328)
			p.MapType()
		}
		p.SetState(330)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		if _la == OracleParserQUESTION {
			{
				p.SetState(329)
				p.TypeModifiers()
			}

		}

	case OracleParserIDENT:
		localctx = NewTypeRefNormalContext(p, localctx)
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(332)
			p.QualifiedIdent()
		}
		p.SetState(334)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		if _la == OracleParserLT {
			{
				p.SetState(333)
				p.TypeArgs()
			}

		}
		p.SetState(338)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		if _la == OracleParserLBRACKET {
			{
				p.SetState(336)
				p.Match(OracleParserLBRACKET)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}
			{
				p.SetState(337)
				p.Match(OracleParserRBRACKET)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}

		}
		p.SetState(341)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		if _la == OracleParserQUESTION {
			{
				p.SetState(340)
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
	p.EnterRule(localctx, 36, OracleParserRULE_mapType)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(345)
		p.Match(OracleParserMAP)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(346)
		p.Match(OracleParserLT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(347)
		p.TypeRef()
	}
	{
		p.SetState(348)
		p.Match(OracleParserCOMMA)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(349)
		p.TypeRef()
	}
	{
		p.SetState(350)
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
	p.EnterRule(localctx, 38, OracleParserRULE_typeArgs)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(352)
		p.Match(OracleParserLT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(353)
		p.TypeRef()
	}
	p.SetState(358)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserCOMMA {
		{
			p.SetState(354)
			p.Match(OracleParserCOMMA)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(355)
			p.TypeRef()
		}

		p.SetState(360)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(361)
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
	p.EnterRule(localctx, 40, OracleParserRULE_typeModifiers)
	p.SetState(366)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 51, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(363)
			p.Match(OracleParserQUESTION)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(364)
			p.Match(OracleParserQUESTION)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(365)
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
	p.EnterRule(localctx, 42, OracleParserRULE_qualifiedIdent)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(368)
		p.Match(OracleParserIDENT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(371)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == OracleParserDOT {
		{
			p.SetState(369)
			p.Match(OracleParserDOT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(370)
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
	p.EnterRule(localctx, 44, OracleParserRULE_expression)
	var _alt int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(373)
		p.Match(OracleParserIDENT)
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
	_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 53, p.GetParserRuleContext())
	if p.HasError() {
		goto errorExit
	}
	for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
		if _alt == 1 {
			{
				p.SetState(374)
				p.ExpressionValue()
			}

		}
		p.SetState(379)
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

// IExpressionValueContext is an interface to support dynamic dispatch.
type IExpressionValueContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
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
	p.EnterRule(localctx, 46, OracleParserRULE_expressionValue)
	p.SetState(385)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case OracleParserSTRING_LIT:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(380)
			p.Match(OracleParserSTRING_LIT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case OracleParserINT_LIT:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(381)
			p.Match(OracleParserINT_LIT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case OracleParserFLOAT_LIT:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(382)
			p.Match(OracleParserFLOAT_LIT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case OracleParserBOOL_LIT:
		p.EnterOuterAlt(localctx, 4)
		{
			p.SetState(383)
			p.Match(OracleParserBOOL_LIT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case OracleParserIDENT:
		p.EnterOuterAlt(localctx, 5)
		{
			p.SetState(384)
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
	p.EnterRule(localctx, 48, OracleParserRULE_enumDef)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(387)
		p.Match(OracleParserIDENT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(388)
		p.Match(OracleParserENUM)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(392)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(389)
			p.Nl()
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
		p.Match(OracleParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(399)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(396)
			p.Nl()
		}

		p.SetState(401)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(402)
		p.EnumBody()
	}
	{
		p.SetState(403)
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
	p.EnterRule(localctx, 50, OracleParserRULE_enumBody)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(417)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserAT || _la == OracleParserIDENT {
		p.SetState(407)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}

		switch p.GetTokenStream().LA(1) {
		case OracleParserIDENT:
			{
				p.SetState(405)
				p.EnumValue()
			}

		case OracleParserAT:
			{
				p.SetState(406)
				p.Domain()
			}

		default:
			p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
			goto errorExit
		}
		p.SetState(412)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(409)
				p.Nl()
			}

			p.SetState(414)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}

		p.SetState(419)
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
	p.EnterRule(localctx, 52, OracleParserRULE_enumValue)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(420)
		p.Match(OracleParserIDENT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(421)
		p.Match(OracleParserEQUALS)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(422)
		_la = p.GetTokenStream().LA(1)

		if !(_la == OracleParserSTRING_LIT || _la == OracleParserINT_LIT) {
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
	p.EnterRule(localctx, 54, OracleParserRULE_typeDefDef)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(424)
		p.Match(OracleParserIDENT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(426)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == OracleParserLT {
		{
			p.SetState(425)
			p.TypeParams()
		}

	}
	{
		p.SetState(428)
		p.TypeRef()
	}
	p.SetState(430)
	p.GetErrorHandler().Sync(p)

	if p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 61, p.GetParserRuleContext()) == 1 {
		{
			p.SetState(429)
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
	p.EnterRule(localctx, 56, OracleParserRULE_typeDefBody)
	var _la int

	p.EnterOuterAlt(localctx, 1)
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
		p.Match(OracleParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(442)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(439)
			p.Nl()
		}

		p.SetState(444)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	p.SetState(454)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserAT {
		{
			p.SetState(445)
			p.Domain()
		}
		p.SetState(449)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(446)
				p.Nl()
			}

			p.SetState(451)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}

		p.SetState(456)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(457)
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
