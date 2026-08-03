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
		"DOT", "EQUALS", "AT", "MINUS", "TRIPLE_STRING_LIT", "STRING_LIT", "FLOAT_LIT",
		"INT_LIT", "BOOL_LIT", "IDENT", "LINE_COMMENT", "BLOCK_COMMENT", "NEWLINE",
		"WS",
	}
	staticData.RuleNames = []string{
		"STRUCT", "ENUM", "IMPORT", "EXTENDS", "MAP", "ACTION", "UNION", "LBRACE",
		"RBRACE", "LBRACKET", "RBRACKET", "LT", "GT", "COMMA", "QUESTION", "DOT",
		"EQUALS", "AT", "MINUS", "TRIPLE_STRING_LIT", "STRING_LIT", "ESCAPE_SEQUENCE",
		"FLOAT_LIT", "INT_LIT", "BOOL_LIT", "IDENT", "LINE_COMMENT", "BLOCK_COMMENT",
		"NEWLINE", "WS",
	}
	staticData.PredictionContextCache = antlr.NewPredictionContextCache()
	staticData.serializedATN = []int32{
		4, 0, 29, 241, 6, -1, 2, 0, 7, 0, 2, 1, 7, 1, 2, 2, 7, 2, 2, 3, 7, 3, 2,
		4, 7, 4, 2, 5, 7, 5, 2, 6, 7, 6, 2, 7, 7, 7, 2, 8, 7, 8, 2, 9, 7, 9, 2,
		10, 7, 10, 2, 11, 7, 11, 2, 12, 7, 12, 2, 13, 7, 13, 2, 14, 7, 14, 2, 15,
		7, 15, 2, 16, 7, 16, 2, 17, 7, 17, 2, 18, 7, 18, 2, 19, 7, 19, 2, 20, 7,
		20, 2, 21, 7, 21, 2, 22, 7, 22, 2, 23, 7, 23, 2, 24, 7, 24, 2, 25, 7, 25,
		2, 26, 7, 26, 2, 27, 7, 27, 2, 28, 7, 28, 2, 29, 7, 29, 1, 0, 1, 0, 1,
		0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 2, 1, 2, 1,
		2, 1, 2, 1, 2, 1, 2, 1, 2, 1, 3, 1, 3, 1, 3, 1, 3, 1, 3, 1, 3, 1, 3, 1,
		3, 1, 4, 1, 4, 1, 4, 1, 4, 1, 5, 1, 5, 1, 5, 1, 5, 1, 5, 1, 5, 1, 5, 1,
		6, 1, 6, 1, 6, 1, 6, 1, 6, 1, 6, 1, 7, 1, 7, 1, 8, 1, 8, 1, 9, 1, 9, 1,
		10, 1, 10, 1, 11, 1, 11, 1, 12, 1, 12, 1, 13, 1, 13, 1, 14, 1, 14, 1, 15,
		1, 15, 1, 16, 1, 16, 1, 17, 1, 17, 1, 18, 1, 18, 1, 19, 1, 19, 1, 19, 1,
		19, 1, 19, 5, 19, 135, 8, 19, 10, 19, 12, 19, 138, 9, 19, 1, 19, 1, 19,
		1, 19, 1, 19, 1, 20, 1, 20, 1, 20, 5, 20, 147, 8, 20, 10, 20, 12, 20, 150,
		9, 20, 1, 20, 1, 20, 1, 21, 1, 21, 1, 21, 1, 21, 1, 21, 1, 21, 1, 21, 1,
		21, 1, 21, 3, 21, 163, 8, 21, 1, 22, 3, 22, 166, 8, 22, 1, 22, 4, 22, 169,
		8, 22, 11, 22, 12, 22, 170, 1, 22, 1, 22, 4, 22, 175, 8, 22, 11, 22, 12,
		22, 176, 1, 23, 3, 23, 180, 8, 23, 1, 23, 4, 23, 183, 8, 23, 11, 23, 12,
		23, 184, 1, 24, 1, 24, 1, 24, 1, 24, 1, 24, 1, 24, 1, 24, 1, 24, 1, 24,
		3, 24, 196, 8, 24, 1, 25, 1, 25, 5, 25, 200, 8, 25, 10, 25, 12, 25, 203,
		9, 25, 1, 26, 1, 26, 1, 26, 1, 26, 5, 26, 209, 8, 26, 10, 26, 12, 26, 212,
		9, 26, 1, 26, 1, 26, 1, 27, 1, 27, 1, 27, 1, 27, 5, 27, 220, 8, 27, 10,
		27, 12, 27, 223, 9, 27, 1, 27, 1, 27, 1, 27, 1, 27, 1, 27, 1, 28, 4, 28,
		231, 8, 28, 11, 28, 12, 28, 232, 1, 29, 4, 29, 236, 8, 29, 11, 29, 12,
		29, 237, 1, 29, 1, 29, 2, 136, 221, 0, 30, 1, 1, 3, 2, 5, 3, 7, 4, 9, 5,
		11, 6, 13, 7, 15, 8, 17, 9, 19, 10, 21, 11, 23, 12, 25, 13, 27, 14, 29,
		15, 31, 16, 33, 17, 35, 18, 37, 19, 39, 20, 41, 21, 43, 0, 45, 22, 47,
		23, 49, 24, 51, 25, 53, 26, 55, 27, 57, 28, 59, 29, 1, 0, 8, 4, 0, 10,
		10, 13, 13, 34, 34, 92, 92, 7, 0, 34, 34, 92, 92, 98, 98, 102, 102, 110,
		110, 114, 114, 116, 116, 3, 0, 48, 57, 65, 70, 97, 102, 1, 0, 48, 57, 3,
		0, 65, 90, 95, 95, 97, 122, 4, 0, 48, 57, 65, 90, 95, 95, 97, 122, 2, 0,
		10, 10, 13, 13, 2, 0, 9, 9, 32, 32, 254, 0, 1, 1, 0, 0, 0, 0, 3, 1, 0,
		0, 0, 0, 5, 1, 0, 0, 0, 0, 7, 1, 0, 0, 0, 0, 9, 1, 0, 0, 0, 0, 11, 1, 0,
		0, 0, 0, 13, 1, 0, 0, 0, 0, 15, 1, 0, 0, 0, 0, 17, 1, 0, 0, 0, 0, 19, 1,
		0, 0, 0, 0, 21, 1, 0, 0, 0, 0, 23, 1, 0, 0, 0, 0, 25, 1, 0, 0, 0, 0, 27,
		1, 0, 0, 0, 0, 29, 1, 0, 0, 0, 0, 31, 1, 0, 0, 0, 0, 33, 1, 0, 0, 0, 0,
		35, 1, 0, 0, 0, 0, 37, 1, 0, 0, 0, 0, 39, 1, 0, 0, 0, 0, 41, 1, 0, 0, 0,
		0, 45, 1, 0, 0, 0, 0, 47, 1, 0, 0, 0, 0, 49, 1, 0, 0, 0, 0, 51, 1, 0, 0,
		0, 0, 53, 1, 0, 0, 0, 0, 55, 1, 0, 0, 0, 0, 57, 1, 0, 0, 0, 0, 59, 1, 0,
		0, 0, 1, 61, 1, 0, 0, 0, 3, 68, 1, 0, 0, 0, 5, 73, 1, 0, 0, 0, 7, 80, 1,
		0, 0, 0, 9, 88, 1, 0, 0, 0, 11, 92, 1, 0, 0, 0, 13, 99, 1, 0, 0, 0, 15,
		105, 1, 0, 0, 0, 17, 107, 1, 0, 0, 0, 19, 109, 1, 0, 0, 0, 21, 111, 1,
		0, 0, 0, 23, 113, 1, 0, 0, 0, 25, 115, 1, 0, 0, 0, 27, 117, 1, 0, 0, 0,
		29, 119, 1, 0, 0, 0, 31, 121, 1, 0, 0, 0, 33, 123, 1, 0, 0, 0, 35, 125,
		1, 0, 0, 0, 37, 127, 1, 0, 0, 0, 39, 129, 1, 0, 0, 0, 41, 143, 1, 0, 0,
		0, 43, 162, 1, 0, 0, 0, 45, 165, 1, 0, 0, 0, 47, 179, 1, 0, 0, 0, 49, 195,
		1, 0, 0, 0, 51, 197, 1, 0, 0, 0, 53, 204, 1, 0, 0, 0, 55, 215, 1, 0, 0,
		0, 57, 230, 1, 0, 0, 0, 59, 235, 1, 0, 0, 0, 61, 62, 5, 115, 0, 0, 62,
		63, 5, 116, 0, 0, 63, 64, 5, 114, 0, 0, 64, 65, 5, 117, 0, 0, 65, 66, 5,
		99, 0, 0, 66, 67, 5, 116, 0, 0, 67, 2, 1, 0, 0, 0, 68, 69, 5, 101, 0, 0,
		69, 70, 5, 110, 0, 0, 70, 71, 5, 117, 0, 0, 71, 72, 5, 109, 0, 0, 72, 4,
		1, 0, 0, 0, 73, 74, 5, 105, 0, 0, 74, 75, 5, 109, 0, 0, 75, 76, 5, 112,
		0, 0, 76, 77, 5, 111, 0, 0, 77, 78, 5, 114, 0, 0, 78, 79, 5, 116, 0, 0,
		79, 6, 1, 0, 0, 0, 80, 81, 5, 101, 0, 0, 81, 82, 5, 120, 0, 0, 82, 83,
		5, 116, 0, 0, 83, 84, 5, 101, 0, 0, 84, 85, 5, 110, 0, 0, 85, 86, 5, 100,
		0, 0, 86, 87, 5, 115, 0, 0, 87, 8, 1, 0, 0, 0, 88, 89, 5, 109, 0, 0, 89,
		90, 5, 97, 0, 0, 90, 91, 5, 112, 0, 0, 91, 10, 1, 0, 0, 0, 92, 93, 5, 97,
		0, 0, 93, 94, 5, 99, 0, 0, 94, 95, 5, 116, 0, 0, 95, 96, 5, 105, 0, 0,
		96, 97, 5, 111, 0, 0, 97, 98, 5, 110, 0, 0, 98, 12, 1, 0, 0, 0, 99, 100,
		5, 117, 0, 0, 100, 101, 5, 110, 0, 0, 101, 102, 5, 105, 0, 0, 102, 103,
		5, 111, 0, 0, 103, 104, 5, 110, 0, 0, 104, 14, 1, 0, 0, 0, 105, 106, 5,
		123, 0, 0, 106, 16, 1, 0, 0, 0, 107, 108, 5, 125, 0, 0, 108, 18, 1, 0,
		0, 0, 109, 110, 5, 91, 0, 0, 110, 20, 1, 0, 0, 0, 111, 112, 5, 93, 0, 0,
		112, 22, 1, 0, 0, 0, 113, 114, 5, 60, 0, 0, 114, 24, 1, 0, 0, 0, 115, 116,
		5, 62, 0, 0, 116, 26, 1, 0, 0, 0, 117, 118, 5, 44, 0, 0, 118, 28, 1, 0,
		0, 0, 119, 120, 5, 63, 0, 0, 120, 30, 1, 0, 0, 0, 121, 122, 5, 46, 0, 0,
		122, 32, 1, 0, 0, 0, 123, 124, 5, 61, 0, 0, 124, 34, 1, 0, 0, 0, 125, 126,
		5, 64, 0, 0, 126, 36, 1, 0, 0, 0, 127, 128, 5, 45, 0, 0, 128, 38, 1, 0,
		0, 0, 129, 130, 5, 34, 0, 0, 130, 131, 5, 34, 0, 0, 131, 132, 5, 34, 0,
		0, 132, 136, 1, 0, 0, 0, 133, 135, 9, 0, 0, 0, 134, 133, 1, 0, 0, 0, 135,
		138, 1, 0, 0, 0, 136, 137, 1, 0, 0, 0, 136, 134, 1, 0, 0, 0, 137, 139,
		1, 0, 0, 0, 138, 136, 1, 0, 0, 0, 139, 140, 5, 34, 0, 0, 140, 141, 5, 34,
		0, 0, 141, 142, 5, 34, 0, 0, 142, 40, 1, 0, 0, 0, 143, 148, 5, 34, 0, 0,
		144, 147, 8, 0, 0, 0, 145, 147, 3, 43, 21, 0, 146, 144, 1, 0, 0, 0, 146,
		145, 1, 0, 0, 0, 147, 150, 1, 0, 0, 0, 148, 146, 1, 0, 0, 0, 148, 149,
		1, 0, 0, 0, 149, 151, 1, 0, 0, 0, 150, 148, 1, 0, 0, 0, 151, 152, 5, 34,
		0, 0, 152, 42, 1, 0, 0, 0, 153, 154, 5, 92, 0, 0, 154, 163, 7, 1, 0, 0,
		155, 156, 5, 92, 0, 0, 156, 157, 5, 117, 0, 0, 157, 158, 1, 0, 0, 0, 158,
		159, 7, 2, 0, 0, 159, 160, 7, 2, 0, 0, 160, 161, 7, 2, 0, 0, 161, 163,
		7, 2, 0, 0, 162, 153, 1, 0, 0, 0, 162, 155, 1, 0, 0, 0, 163, 44, 1, 0,
		0, 0, 164, 166, 5, 45, 0, 0, 165, 164, 1, 0, 0, 0, 165, 166, 1, 0, 0, 0,
		166, 168, 1, 0, 0, 0, 167, 169, 7, 3, 0, 0, 168, 167, 1, 0, 0, 0, 169,
		170, 1, 0, 0, 0, 170, 168, 1, 0, 0, 0, 170, 171, 1, 0, 0, 0, 171, 172,
		1, 0, 0, 0, 172, 174, 5, 46, 0, 0, 173, 175, 7, 3, 0, 0, 174, 173, 1, 0,
		0, 0, 175, 176, 1, 0, 0, 0, 176, 174, 1, 0, 0, 0, 176, 177, 1, 0, 0, 0,
		177, 46, 1, 0, 0, 0, 178, 180, 5, 45, 0, 0, 179, 178, 1, 0, 0, 0, 179,
		180, 1, 0, 0, 0, 180, 182, 1, 0, 0, 0, 181, 183, 7, 3, 0, 0, 182, 181,
		1, 0, 0, 0, 183, 184, 1, 0, 0, 0, 184, 182, 1, 0, 0, 0, 184, 185, 1, 0,
		0, 0, 185, 48, 1, 0, 0, 0, 186, 187, 5, 116, 0, 0, 187, 188, 5, 114, 0,
		0, 188, 189, 5, 117, 0, 0, 189, 196, 5, 101, 0, 0, 190, 191, 5, 102, 0,
		0, 191, 192, 5, 97, 0, 0, 192, 193, 5, 108, 0, 0, 193, 194, 5, 115, 0,
		0, 194, 196, 5, 101, 0, 0, 195, 186, 1, 0, 0, 0, 195, 190, 1, 0, 0, 0,
		196, 50, 1, 0, 0, 0, 197, 201, 7, 4, 0, 0, 198, 200, 7, 5, 0, 0, 199, 198,
		1, 0, 0, 0, 200, 203, 1, 0, 0, 0, 201, 199, 1, 0, 0, 0, 201, 202, 1, 0,
		0, 0, 202, 52, 1, 0, 0, 0, 203, 201, 1, 0, 0, 0, 204, 205, 5, 47, 0, 0,
		205, 206, 5, 47, 0, 0, 206, 210, 1, 0, 0, 0, 207, 209, 8, 6, 0, 0, 208,
		207, 1, 0, 0, 0, 209, 212, 1, 0, 0, 0, 210, 208, 1, 0, 0, 0, 210, 211,
		1, 0, 0, 0, 211, 213, 1, 0, 0, 0, 212, 210, 1, 0, 0, 0, 213, 214, 6, 26,
		0, 0, 214, 54, 1, 0, 0, 0, 215, 216, 5, 47, 0, 0, 216, 217, 5, 42, 0, 0,
		217, 221, 1, 0, 0, 0, 218, 220, 9, 0, 0, 0, 219, 218, 1, 0, 0, 0, 220,
		223, 1, 0, 0, 0, 221, 222, 1, 0, 0, 0, 221, 219, 1, 0, 0, 0, 222, 224,
		1, 0, 0, 0, 223, 221, 1, 0, 0, 0, 224, 225, 5, 42, 0, 0, 225, 226, 5, 47,
		0, 0, 226, 227, 1, 0, 0, 0, 227, 228, 6, 27, 0, 0, 228, 56, 1, 0, 0, 0,
		229, 231, 7, 6, 0, 0, 230, 229, 1, 0, 0, 0, 231, 232, 1, 0, 0, 0, 232,
		230, 1, 0, 0, 0, 232, 233, 1, 0, 0, 0, 233, 58, 1, 0, 0, 0, 234, 236, 7,
		7, 0, 0, 235, 234, 1, 0, 0, 0, 236, 237, 1, 0, 0, 0, 237, 235, 1, 0, 0,
		0, 237, 238, 1, 0, 0, 0, 238, 239, 1, 0, 0, 0, 239, 240, 6, 29, 1, 0, 240,
		60, 1, 0, 0, 0, 16, 0, 136, 146, 148, 162, 165, 170, 176, 179, 184, 195,
		201, 210, 221, 232, 237, 2, 0, 1, 0, 6, 0, 0,
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
	OracleLexerFLOAT_LIT         = 22
	OracleLexerINT_LIT           = 23
	OracleLexerBOOL_LIT          = 24
	OracleLexerIDENT             = 25
	OracleLexerLINE_COMMENT      = 26
	OracleLexerBLOCK_COMMENT     = 27
	OracleLexerNEWLINE           = 28
	OracleLexerWS                = 29
)
