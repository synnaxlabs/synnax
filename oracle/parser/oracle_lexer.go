// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Code generated from OracleLexer.g4 by ANTLR 4.13.2. DO NOT EDIT.

package parser

import (
	"fmt"
	"github.com/antlr4-go/antlr/v4"
	"sync"
	"unicode"
)

// Suppress unused import error
var _ = fmt.Printf
var _ = sync.Once{}
var _ = unicode.IsLetter

type OracleLexer struct {
	*antlr.BaseLexer
	channelNames []string
	modeNames    []string
	// TODO: EOF string
}

var OracleLexerLexerStaticData struct {
	once                   sync.Once
	serializedATN          []int32
	ChannelNames           []string
	ModeNames              []string
	LiteralNames           []string
	SymbolicNames          []string
	RuleNames              []string
	PredictionContextCache *antlr.PredictionContextCache
	atn                    *antlr.ATN
	decisionToDFA          []*antlr.DFA
}

func oraclelexerLexerInit() {
	staticData := &OracleLexerLexerStaticData
	staticData.ChannelNames = []string{
		"DEFAULT_TOKEN_CHANNEL", "HIDDEN",
	}
	staticData.ModeNames = []string{
		"DEFAULT_MODE",
	}
	staticData.LiteralNames = []string{
		"", "'struct'", "'enum'", "'import'", "'extends'", "'map'", "'action'",
		"'union'", "'{'", "'}'", "'['", "']'", "'<'", "'>'", "','", "'?'", "'.'",
		"'='", "'@'", "'-'",
	}
	staticData.SymbolicNames = []string{
		"", "STRUCT", "ENUM", "IMPORT", "EXTENDS", "MAP", "ACTION", "UNION",
		"LBRACE", "RBRACE", "LBRACKET", "RBRACKET", "LT", "GT", "COMMA", "QUESTION",
		"DOT", "EQUALS", "AT", "MINUS", "TRIPLE_STRING_LIT", "STRING_LIT", "DURATION_LIT",
		"FLOAT_LIT", "INT_LIT", "BOOL_LIT", "IDENT", "LINE_COMMENT", "BLOCK_COMMENT",
		"NEWLINE", "WS",
	}
	staticData.RuleNames = []string{
		"STRUCT", "ENUM", "IMPORT", "EXTENDS", "MAP", "ACTION", "UNION", "LBRACE",
		"RBRACE", "LBRACKET", "RBRACKET", "LT", "GT", "COMMA", "QUESTION", "DOT",
		"EQUALS", "AT", "MINUS", "TRIPLE_STRING_LIT", "STRING_LIT", "ESCAPE_SEQUENCE",
		"DURATION_LIT", "DURATION_UNIT", "FLOAT_LIT", "INT_LIT", "BOOL_LIT",
		"IDENT", "LINE_COMMENT", "BLOCK_COMMENT", "NEWLINE", "WS",
	}
	staticData.PredictionContextCache = antlr.NewPredictionContextCache()
	staticData.serializedATN = []int32{
		4, 0, 30, 275, 6, -1, 2, 0, 7, 0, 2, 1, 7, 1, 2, 2, 7, 2, 2, 3, 7, 3, 2,
		4, 7, 4, 2, 5, 7, 5, 2, 6, 7, 6, 2, 7, 7, 7, 2, 8, 7, 8, 2, 9, 7, 9, 2,
		10, 7, 10, 2, 11, 7, 11, 2, 12, 7, 12, 2, 13, 7, 13, 2, 14, 7, 14, 2, 15,
		7, 15, 2, 16, 7, 16, 2, 17, 7, 17, 2, 18, 7, 18, 2, 19, 7, 19, 2, 20, 7,
		20, 2, 21, 7, 21, 2, 22, 7, 22, 2, 23, 7, 23, 2, 24, 7, 24, 2, 25, 7, 25,
		2, 26, 7, 26, 2, 27, 7, 27, 2, 28, 7, 28, 2, 29, 7, 29, 2, 30, 7, 30, 2,
		31, 7, 31, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 1, 1, 1, 1, 1,
		1, 1, 1, 1, 1, 2, 1, 2, 1, 2, 1, 2, 1, 2, 1, 2, 1, 2, 1, 3, 1, 3, 1, 3,
		1, 3, 1, 3, 1, 3, 1, 3, 1, 3, 1, 4, 1, 4, 1, 4, 1, 4, 1, 5, 1, 5, 1, 5,
		1, 5, 1, 5, 1, 5, 1, 5, 1, 6, 1, 6, 1, 6, 1, 6, 1, 6, 1, 6, 1, 7, 1, 7,
		1, 8, 1, 8, 1, 9, 1, 9, 1, 10, 1, 10, 1, 11, 1, 11, 1, 12, 1, 12, 1, 13,
		1, 13, 1, 14, 1, 14, 1, 15, 1, 15, 1, 16, 1, 16, 1, 17, 1, 17, 1, 18, 1,
		18, 1, 19, 1, 19, 1, 19, 1, 19, 1, 19, 5, 19, 139, 8, 19, 10, 19, 12, 19,
		142, 9, 19, 1, 19, 1, 19, 1, 19, 1, 19, 1, 20, 1, 20, 1, 20, 5, 20, 151,
		8, 20, 10, 20, 12, 20, 154, 9, 20, 1, 20, 1, 20, 1, 21, 1, 21, 1, 21, 1,
		21, 1, 21, 1, 21, 1, 21, 1, 21, 1, 21, 3, 21, 167, 8, 21, 1, 22, 3, 22,
		170, 8, 22, 1, 22, 4, 22, 173, 8, 22, 11, 22, 12, 22, 174, 1, 22, 1, 22,
		4, 22, 179, 8, 22, 11, 22, 12, 22, 180, 3, 22, 183, 8, 22, 1, 22, 4, 22,
		186, 8, 22, 11, 22, 12, 22, 187, 1, 23, 1, 23, 1, 23, 1, 23, 1, 23, 1,
		23, 1, 23, 3, 23, 197, 8, 23, 1, 24, 3, 24, 200, 8, 24, 1, 24, 4, 24, 203,
		8, 24, 11, 24, 12, 24, 204, 1, 24, 1, 24, 4, 24, 209, 8, 24, 11, 24, 12,
		24, 210, 1, 25, 3, 25, 214, 8, 25, 1, 25, 4, 25, 217, 8, 25, 11, 25, 12,
		25, 218, 1, 26, 1, 26, 1, 26, 1, 26, 1, 26, 1, 26, 1, 26, 1, 26, 1, 26,
		3, 26, 230, 8, 26, 1, 27, 1, 27, 5, 27, 234, 8, 27, 10, 27, 12, 27, 237,
		9, 27, 1, 28, 1, 28, 1, 28, 1, 28, 5, 28, 243, 8, 28, 10, 28, 12, 28, 246,
		9, 28, 1, 28, 1, 28, 1, 29, 1, 29, 1, 29, 1, 29, 5, 29, 254, 8, 29, 10,
		29, 12, 29, 257, 9, 29, 1, 29, 1, 29, 1, 29, 1, 29, 1, 29, 1, 30, 4, 30,
		265, 8, 30, 11, 30, 12, 30, 266, 1, 31, 4, 31, 270, 8, 31, 11, 31, 12,
		31, 271, 1, 31, 1, 31, 2, 140, 255, 0, 32, 1, 1, 3, 2, 5, 3, 7, 4, 9, 5,
		11, 6, 13, 7, 15, 8, 17, 9, 19, 10, 21, 11, 23, 12, 25, 13, 27, 14, 29,
		15, 31, 16, 33, 17, 35, 18, 37, 19, 39, 20, 41, 21, 43, 0, 45, 22, 47,
		0, 49, 23, 51, 24, 53, 25, 55, 26, 57, 27, 59, 28, 61, 29, 63, 30, 1, 0,
		9, 4, 0, 10, 10, 13, 13, 34, 34, 92, 92, 7, 0, 34, 34, 92, 92, 98, 98,
		102, 102, 110, 110, 114, 114, 116, 116, 3, 0, 48, 57, 65, 70, 97, 102,
		1, 0, 48, 57, 3, 0, 104, 104, 109, 109, 115, 115, 3, 0, 65, 90, 95, 95,
		97, 122, 4, 0, 48, 57, 65, 90, 95, 95, 97, 122, 2, 0, 10, 10, 13, 13, 2,
		0, 9, 9, 32, 32, 295, 0, 1, 1, 0, 0, 0, 0, 3, 1, 0, 0, 0, 0, 5, 1, 0, 0,
		0, 0, 7, 1, 0, 0, 0, 0, 9, 1, 0, 0, 0, 0, 11, 1, 0, 0, 0, 0, 13, 1, 0,
		0, 0, 0, 15, 1, 0, 0, 0, 0, 17, 1, 0, 0, 0, 0, 19, 1, 0, 0, 0, 0, 21, 1,
		0, 0, 0, 0, 23, 1, 0, 0, 0, 0, 25, 1, 0, 0, 0, 0, 27, 1, 0, 0, 0, 0, 29,
		1, 0, 0, 0, 0, 31, 1, 0, 0, 0, 0, 33, 1, 0, 0, 0, 0, 35, 1, 0, 0, 0, 0,
		37, 1, 0, 0, 0, 0, 39, 1, 0, 0, 0, 0, 41, 1, 0, 0, 0, 0, 45, 1, 0, 0, 0,
		0, 49, 1, 0, 0, 0, 0, 51, 1, 0, 0, 0, 0, 53, 1, 0, 0, 0, 0, 55, 1, 0, 0,
		0, 0, 57, 1, 0, 0, 0, 0, 59, 1, 0, 0, 0, 0, 61, 1, 0, 0, 0, 0, 63, 1, 0,
		0, 0, 1, 65, 1, 0, 0, 0, 3, 72, 1, 0, 0, 0, 5, 77, 1, 0, 0, 0, 7, 84, 1,
		0, 0, 0, 9, 92, 1, 0, 0, 0, 11, 96, 1, 0, 0, 0, 13, 103, 1, 0, 0, 0, 15,
		109, 1, 0, 0, 0, 17, 111, 1, 0, 0, 0, 19, 113, 1, 0, 0, 0, 21, 115, 1,
		0, 0, 0, 23, 117, 1, 0, 0, 0, 25, 119, 1, 0, 0, 0, 27, 121, 1, 0, 0, 0,
		29, 123, 1, 0, 0, 0, 31, 125, 1, 0, 0, 0, 33, 127, 1, 0, 0, 0, 35, 129,
		1, 0, 0, 0, 37, 131, 1, 0, 0, 0, 39, 133, 1, 0, 0, 0, 41, 147, 1, 0, 0,
		0, 43, 166, 1, 0, 0, 0, 45, 169, 1, 0, 0, 0, 47, 196, 1, 0, 0, 0, 49, 199,
		1, 0, 0, 0, 51, 213, 1, 0, 0, 0, 53, 229, 1, 0, 0, 0, 55, 231, 1, 0, 0,
		0, 57, 238, 1, 0, 0, 0, 59, 249, 1, 0, 0, 0, 61, 264, 1, 0, 0, 0, 63, 269,
		1, 0, 0, 0, 65, 66, 5, 115, 0, 0, 66, 67, 5, 116, 0, 0, 67, 68, 5, 114,
		0, 0, 68, 69, 5, 117, 0, 0, 69, 70, 5, 99, 0, 0, 70, 71, 5, 116, 0, 0,
		71, 2, 1, 0, 0, 0, 72, 73, 5, 101, 0, 0, 73, 74, 5, 110, 0, 0, 74, 75,
		5, 117, 0, 0, 75, 76, 5, 109, 0, 0, 76, 4, 1, 0, 0, 0, 77, 78, 5, 105,
		0, 0, 78, 79, 5, 109, 0, 0, 79, 80, 5, 112, 0, 0, 80, 81, 5, 111, 0, 0,
		81, 82, 5, 114, 0, 0, 82, 83, 5, 116, 0, 0, 83, 6, 1, 0, 0, 0, 84, 85,
		5, 101, 0, 0, 85, 86, 5, 120, 0, 0, 86, 87, 5, 116, 0, 0, 87, 88, 5, 101,
		0, 0, 88, 89, 5, 110, 0, 0, 89, 90, 5, 100, 0, 0, 90, 91, 5, 115, 0, 0,
		91, 8, 1, 0, 0, 0, 92, 93, 5, 109, 0, 0, 93, 94, 5, 97, 0, 0, 94, 95, 5,
		112, 0, 0, 95, 10, 1, 0, 0, 0, 96, 97, 5, 97, 0, 0, 97, 98, 5, 99, 0, 0,
		98, 99, 5, 116, 0, 0, 99, 100, 5, 105, 0, 0, 100, 101, 5, 111, 0, 0, 101,
		102, 5, 110, 0, 0, 102, 12, 1, 0, 0, 0, 103, 104, 5, 117, 0, 0, 104, 105,
		5, 110, 0, 0, 105, 106, 5, 105, 0, 0, 106, 107, 5, 111, 0, 0, 107, 108,
		5, 110, 0, 0, 108, 14, 1, 0, 0, 0, 109, 110, 5, 123, 0, 0, 110, 16, 1,
		0, 0, 0, 111, 112, 5, 125, 0, 0, 112, 18, 1, 0, 0, 0, 113, 114, 5, 91,
		0, 0, 114, 20, 1, 0, 0, 0, 115, 116, 5, 93, 0, 0, 116, 22, 1, 0, 0, 0,
		117, 118, 5, 60, 0, 0, 118, 24, 1, 0, 0, 0, 119, 120, 5, 62, 0, 0, 120,
		26, 1, 0, 0, 0, 121, 122, 5, 44, 0, 0, 122, 28, 1, 0, 0, 0, 123, 124, 5,
		63, 0, 0, 124, 30, 1, 0, 0, 0, 125, 126, 5, 46, 0, 0, 126, 32, 1, 0, 0,
		0, 127, 128, 5, 61, 0, 0, 128, 34, 1, 0, 0, 0, 129, 130, 5, 64, 0, 0, 130,
		36, 1, 0, 0, 0, 131, 132, 5, 45, 0, 0, 132, 38, 1, 0, 0, 0, 133, 134, 5,
		34, 0, 0, 134, 135, 5, 34, 0, 0, 135, 136, 5, 34, 0, 0, 136, 140, 1, 0,
		0, 0, 137, 139, 9, 0, 0, 0, 138, 137, 1, 0, 0, 0, 139, 142, 1, 0, 0, 0,
		140, 141, 1, 0, 0, 0, 140, 138, 1, 0, 0, 0, 141, 143, 1, 0, 0, 0, 142,
		140, 1, 0, 0, 0, 143, 144, 5, 34, 0, 0, 144, 145, 5, 34, 0, 0, 145, 146,
		5, 34, 0, 0, 146, 40, 1, 0, 0, 0, 147, 152, 5, 34, 0, 0, 148, 151, 8, 0,
		0, 0, 149, 151, 3, 43, 21, 0, 150, 148, 1, 0, 0, 0, 150, 149, 1, 0, 0,
		0, 151, 154, 1, 0, 0, 0, 152, 150, 1, 0, 0, 0, 152, 153, 1, 0, 0, 0, 153,
		155, 1, 0, 0, 0, 154, 152, 1, 0, 0, 0, 155, 156, 5, 34, 0, 0, 156, 42,
		1, 0, 0, 0, 157, 158, 5, 92, 0, 0, 158, 167, 7, 1, 0, 0, 159, 160, 5, 92,
		0, 0, 160, 161, 5, 117, 0, 0, 161, 162, 1, 0, 0, 0, 162, 163, 7, 2, 0,
		0, 163, 164, 7, 2, 0, 0, 164, 165, 7, 2, 0, 0, 165, 167, 7, 2, 0, 0, 166,
		157, 1, 0, 0, 0, 166, 159, 1, 0, 0, 0, 167, 44, 1, 0, 0, 0, 168, 170, 5,
		45, 0, 0, 169, 168, 1, 0, 0, 0, 169, 170, 1, 0, 0, 0, 170, 185, 1, 0, 0,
		0, 171, 173, 7, 3, 0, 0, 172, 171, 1, 0, 0, 0, 173, 174, 1, 0, 0, 0, 174,
		172, 1, 0, 0, 0, 174, 175, 1, 0, 0, 0, 175, 182, 1, 0, 0, 0, 176, 178,
		5, 46, 0, 0, 177, 179, 7, 3, 0, 0, 178, 177, 1, 0, 0, 0, 179, 180, 1, 0,
		0, 0, 180, 178, 1, 0, 0, 0, 180, 181, 1, 0, 0, 0, 181, 183, 1, 0, 0, 0,
		182, 176, 1, 0, 0, 0, 182, 183, 1, 0, 0, 0, 183, 184, 1, 0, 0, 0, 184,
		186, 3, 47, 23, 0, 185, 172, 1, 0, 0, 0, 186, 187, 1, 0, 0, 0, 187, 185,
		1, 0, 0, 0, 187, 188, 1, 0, 0, 0, 188, 46, 1, 0, 0, 0, 189, 190, 5, 110,
		0, 0, 190, 197, 5, 115, 0, 0, 191, 192, 5, 117, 0, 0, 192, 197, 5, 115,
		0, 0, 193, 194, 5, 109, 0, 0, 194, 197, 5, 115, 0, 0, 195, 197, 7, 4, 0,
		0, 196, 189, 1, 0, 0, 0, 196, 191, 1, 0, 0, 0, 196, 193, 1, 0, 0, 0, 196,
		195, 1, 0, 0, 0, 197, 48, 1, 0, 0, 0, 198, 200, 5, 45, 0, 0, 199, 198,
		1, 0, 0, 0, 199, 200, 1, 0, 0, 0, 200, 202, 1, 0, 0, 0, 201, 203, 7, 3,
		0, 0, 202, 201, 1, 0, 0, 0, 203, 204, 1, 0, 0, 0, 204, 202, 1, 0, 0, 0,
		204, 205, 1, 0, 0, 0, 205, 206, 1, 0, 0, 0, 206, 208, 5, 46, 0, 0, 207,
		209, 7, 3, 0, 0, 208, 207, 1, 0, 0, 0, 209, 210, 1, 0, 0, 0, 210, 208,
		1, 0, 0, 0, 210, 211, 1, 0, 0, 0, 211, 50, 1, 0, 0, 0, 212, 214, 5, 45,
		0, 0, 213, 212, 1, 0, 0, 0, 213, 214, 1, 0, 0, 0, 214, 216, 1, 0, 0, 0,
		215, 217, 7, 3, 0, 0, 216, 215, 1, 0, 0, 0, 217, 218, 1, 0, 0, 0, 218,
		216, 1, 0, 0, 0, 218, 219, 1, 0, 0, 0, 219, 52, 1, 0, 0, 0, 220, 221, 5,
		116, 0, 0, 221, 222, 5, 114, 0, 0, 222, 223, 5, 117, 0, 0, 223, 230, 5,
		101, 0, 0, 224, 225, 5, 102, 0, 0, 225, 226, 5, 97, 0, 0, 226, 227, 5,
		108, 0, 0, 227, 228, 5, 115, 0, 0, 228, 230, 5, 101, 0, 0, 229, 220, 1,
		0, 0, 0, 229, 224, 1, 0, 0, 0, 230, 54, 1, 0, 0, 0, 231, 235, 7, 5, 0,
		0, 232, 234, 7, 6, 0, 0, 233, 232, 1, 0, 0, 0, 234, 237, 1, 0, 0, 0, 235,
		233, 1, 0, 0, 0, 235, 236, 1, 0, 0, 0, 236, 56, 1, 0, 0, 0, 237, 235, 1,
		0, 0, 0, 238, 239, 5, 47, 0, 0, 239, 240, 5, 47, 0, 0, 240, 244, 1, 0,
		0, 0, 241, 243, 8, 7, 0, 0, 242, 241, 1, 0, 0, 0, 243, 246, 1, 0, 0, 0,
		244, 242, 1, 0, 0, 0, 244, 245, 1, 0, 0, 0, 245, 247, 1, 0, 0, 0, 246,
		244, 1, 0, 0, 0, 247, 248, 6, 28, 0, 0, 248, 58, 1, 0, 0, 0, 249, 250,
		5, 47, 0, 0, 250, 251, 5, 42, 0, 0, 251, 255, 1, 0, 0, 0, 252, 254, 9,
		0, 0, 0, 253, 252, 1, 0, 0, 0, 254, 257, 1, 0, 0, 0, 255, 256, 1, 0, 0,
		0, 255, 253, 1, 0, 0, 0, 256, 258, 1, 0, 0, 0, 257, 255, 1, 0, 0, 0, 258,
		259, 5, 42, 0, 0, 259, 260, 5, 47, 0, 0, 260, 261, 1, 0, 0, 0, 261, 262,
		6, 29, 0, 0, 262, 60, 1, 0, 0, 0, 263, 265, 7, 7, 0, 0, 264, 263, 1, 0,
		0, 0, 265, 266, 1, 0, 0, 0, 266, 264, 1, 0, 0, 0, 266, 267, 1, 0, 0, 0,
		267, 62, 1, 0, 0, 0, 268, 270, 7, 8, 0, 0, 269, 268, 1, 0, 0, 0, 270, 271,
		1, 0, 0, 0, 271, 269, 1, 0, 0, 0, 271, 272, 1, 0, 0, 0, 272, 273, 1, 0,
		0, 0, 273, 274, 6, 31, 1, 0, 274, 64, 1, 0, 0, 0, 22, 0, 140, 150, 152,
		166, 169, 174, 180, 182, 187, 196, 199, 204, 210, 213, 218, 229, 235, 244,
		255, 266, 271, 2, 0, 1, 0, 6, 0, 0,
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

// OracleLexerInit initializes any static state used to implement OracleLexer. By default the
// static state used to implement the lexer is lazily initialized during the first call to
// NewOracleLexer(). You can call this function if you wish to initialize the static state ahead
// of time.
func OracleLexerInit() {
	staticData := &OracleLexerLexerStaticData
	staticData.once.Do(oraclelexerLexerInit)
}

// NewOracleLexer produces a new lexer instance for the optional input antlr.CharStream.
func NewOracleLexer(input antlr.CharStream) *OracleLexer {
	OracleLexerInit()
	l := new(OracleLexer)
	l.BaseLexer = antlr.NewBaseLexer(input)
	staticData := &OracleLexerLexerStaticData
	l.Interpreter = antlr.NewLexerATNSimulator(l, staticData.atn, staticData.decisionToDFA, staticData.PredictionContextCache)
	l.channelNames = staticData.ChannelNames
	l.modeNames = staticData.ModeNames
	l.RuleNames = staticData.RuleNames
	l.LiteralNames = staticData.LiteralNames
	l.SymbolicNames = staticData.SymbolicNames
	l.GrammarFileName = "OracleLexer.g4"
	// TODO: l.EOF = antlr.TokenEOF

	return l
}

// OracleLexer tokens.
const (
	OracleLexerSTRUCT            = 1
	OracleLexerENUM              = 2
	OracleLexerIMPORT            = 3
	OracleLexerEXTENDS           = 4
	OracleLexerMAP               = 5
	OracleLexerACTION            = 6
	OracleLexerUNION             = 7
	OracleLexerLBRACE            = 8
	OracleLexerRBRACE            = 9
	OracleLexerLBRACKET          = 10
	OracleLexerRBRACKET          = 11
	OracleLexerLT                = 12
	OracleLexerGT                = 13
	OracleLexerCOMMA             = 14
	OracleLexerQUESTION          = 15
	OracleLexerDOT               = 16
	OracleLexerEQUALS            = 17
	OracleLexerAT                = 18
	OracleLexerMINUS             = 19
	OracleLexerTRIPLE_STRING_LIT = 20
	OracleLexerSTRING_LIT        = 21
	OracleLexerDURATION_LIT      = 22
	OracleLexerFLOAT_LIT         = 23
	OracleLexerINT_LIT           = 24
	OracleLexerBOOL_LIT          = 25
	OracleLexerIDENT             = 26
	OracleLexerLINE_COMMENT      = 27
	OracleLexerBLOCK_COMMENT     = 28
	OracleLexerNEWLINE           = 29
	OracleLexerWS                = 30
)
