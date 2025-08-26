// Code generated from SlateLexer.g4 by ANTLR 4.13.2. DO NOT EDIT.

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

type SlateLexer struct {
	*antlr.BaseLexer
	channelNames []string
	modeNames    []string
	// TODO: EOF string
}

var SlateLexerLexerStaticData struct {
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

func slatelexerLexerInit() {
	staticData := &SlateLexerLexerStaticData
	staticData.ChannelNames = []string{
		"DEFAULT_TOKEN_CHANNEL", "HIDDEN",
	}
	staticData.ModeNames = []string{
		"DEFAULT_MODE",
	}
	staticData.LiteralNames = []string{
		"", "'func'", "'if'", "'else'", "'return'", "'interval'", "'number'",
		"'chan'", "'void'", "'bool'", "'true'", "'false'", "'->'", "'<-'", "':='",
		"'$='", "'='", "'+'", "'-'", "'*'", "'/'", "'=='", "'!='", "'<'", "'<='",
		"'>'", "'>='", "'&&'", "'||'", "'!'", "'('", "')'", "'{'", "'}'", "'['",
		"']'", "','",
	}
	staticData.SymbolicNames = []string{
		"", "FUNC", "IF", "ELSE", "RETURN", "INTERVAL", "NUMBER", "CHAN", "VOID",
		"BOOL", "TRUE", "FALSE", "CHANNEL_SEND", "CHANNEL_RECV", "LOCAL_ASSIGN",
		"STATE_ASSIGN", "ASSIGN", "PLUS", "MINUS", "MULTIPLY", "DIVIDE", "EQUAL",
		"NOT_EQUAL", "LESS_THAN", "LESS_EQUAL", "GREATER_THAN", "GREATER_EQUAL",
		"AND", "OR", "NOT", "LPAREN", "RPAREN", "LBRACE", "RBRACE", "LBRACKET",
		"RBRACKET", "COMMA", "NUMBER_LITERAL", "STRING", "IDENTIFIER", "LINE_COMMENT",
		"BLOCK_COMMENT", "WS",
	}
	staticData.RuleNames = []string{
		"FUNC", "IF", "ELSE", "RETURN", "INTERVAL", "NUMBER", "CHAN", "VOID",
		"BOOL", "TRUE", "FALSE", "CHANNEL_SEND", "CHANNEL_RECV", "LOCAL_ASSIGN",
		"STATE_ASSIGN", "ASSIGN", "PLUS", "MINUS", "MULTIPLY", "DIVIDE", "EQUAL",
		"NOT_EQUAL", "LESS_THAN", "LESS_EQUAL", "GREATER_THAN", "GREATER_EQUAL",
		"AND", "OR", "NOT", "LPAREN", "RPAREN", "LBRACE", "RBRACE", "LBRACKET",
		"RBRACKET", "COMMA", "NUMBER_LITERAL", "STRING", "IDENTIFIER", "LINE_COMMENT",
		"BLOCK_COMMENT", "WS",
	}
	staticData.PredictionContextCache = antlr.NewPredictionContextCache()
	staticData.serializedATN = []int32{
		4, 0, 42, 270, 6, -1, 2, 0, 7, 0, 2, 1, 7, 1, 2, 2, 7, 2, 2, 3, 7, 3, 2,
		4, 7, 4, 2, 5, 7, 5, 2, 6, 7, 6, 2, 7, 7, 7, 2, 8, 7, 8, 2, 9, 7, 9, 2,
		10, 7, 10, 2, 11, 7, 11, 2, 12, 7, 12, 2, 13, 7, 13, 2, 14, 7, 14, 2, 15,
		7, 15, 2, 16, 7, 16, 2, 17, 7, 17, 2, 18, 7, 18, 2, 19, 7, 19, 2, 20, 7,
		20, 2, 21, 7, 21, 2, 22, 7, 22, 2, 23, 7, 23, 2, 24, 7, 24, 2, 25, 7, 25,
		2, 26, 7, 26, 2, 27, 7, 27, 2, 28, 7, 28, 2, 29, 7, 29, 2, 30, 7, 30, 2,
		31, 7, 31, 2, 32, 7, 32, 2, 33, 7, 33, 2, 34, 7, 34, 2, 35, 7, 35, 2, 36,
		7, 36, 2, 37, 7, 37, 2, 38, 7, 38, 2, 39, 7, 39, 2, 40, 7, 40, 2, 41, 7,
		41, 1, 0, 1, 0, 1, 0, 1, 0, 1, 0, 1, 1, 1, 1, 1, 1, 1, 2, 1, 2, 1, 2, 1,
		2, 1, 2, 1, 3, 1, 3, 1, 3, 1, 3, 1, 3, 1, 3, 1, 3, 1, 4, 1, 4, 1, 4, 1,
		4, 1, 4, 1, 4, 1, 4, 1, 4, 1, 4, 1, 5, 1, 5, 1, 5, 1, 5, 1, 5, 1, 5, 1,
		5, 1, 6, 1, 6, 1, 6, 1, 6, 1, 6, 1, 7, 1, 7, 1, 7, 1, 7, 1, 7, 1, 8, 1,
		8, 1, 8, 1, 8, 1, 8, 1, 9, 1, 9, 1, 9, 1, 9, 1, 9, 1, 10, 1, 10, 1, 10,
		1, 10, 1, 10, 1, 10, 1, 11, 1, 11, 1, 11, 1, 12, 1, 12, 1, 12, 1, 13, 1,
		13, 1, 13, 1, 14, 1, 14, 1, 14, 1, 15, 1, 15, 1, 16, 1, 16, 1, 17, 1, 17,
		1, 18, 1, 18, 1, 19, 1, 19, 1, 20, 1, 20, 1, 20, 1, 21, 1, 21, 1, 21, 1,
		22, 1, 22, 1, 23, 1, 23, 1, 23, 1, 24, 1, 24, 1, 25, 1, 25, 1, 25, 1, 26,
		1, 26, 1, 26, 1, 27, 1, 27, 1, 27, 1, 28, 1, 28, 1, 29, 1, 29, 1, 30, 1,
		30, 1, 31, 1, 31, 1, 32, 1, 32, 1, 33, 1, 33, 1, 34, 1, 34, 1, 35, 1, 35,
		1, 36, 4, 36, 209, 8, 36, 11, 36, 12, 36, 210, 1, 36, 1, 36, 4, 36, 215,
		8, 36, 11, 36, 12, 36, 216, 3, 36, 219, 8, 36, 1, 37, 1, 37, 1, 37, 1,
		37, 5, 37, 225, 8, 37, 10, 37, 12, 37, 228, 9, 37, 1, 37, 1, 37, 1, 38,
		1, 38, 5, 38, 234, 8, 38, 10, 38, 12, 38, 237, 9, 38, 1, 39, 1, 39, 1,
		39, 1, 39, 5, 39, 243, 8, 39, 10, 39, 12, 39, 246, 9, 39, 1, 39, 1, 39,
		1, 40, 1, 40, 1, 40, 1, 40, 5, 40, 254, 8, 40, 10, 40, 12, 40, 257, 9,
		40, 1, 40, 1, 40, 1, 40, 1, 40, 1, 40, 1, 41, 4, 41, 265, 8, 41, 11, 41,
		12, 41, 266, 1, 41, 1, 41, 1, 255, 0, 42, 1, 1, 3, 2, 5, 3, 7, 4, 9, 5,
		11, 6, 13, 7, 15, 8, 17, 9, 19, 10, 21, 11, 23, 12, 25, 13, 27, 14, 29,
		15, 31, 16, 33, 17, 35, 18, 37, 19, 39, 20, 41, 21, 43, 22, 45, 23, 47,
		24, 49, 25, 51, 26, 53, 27, 55, 28, 57, 29, 59, 30, 61, 31, 63, 32, 65,
		33, 67, 34, 69, 35, 71, 36, 73, 37, 75, 38, 77, 39, 79, 40, 81, 41, 83,
		42, 1, 0, 6, 1, 0, 48, 57, 4, 0, 10, 10, 13, 13, 34, 34, 92, 92, 3, 0,
		65, 90, 95, 95, 97, 122, 4, 0, 48, 57, 65, 90, 95, 95, 97, 122, 2, 0, 10,
		10, 13, 13, 3, 0, 9, 10, 13, 13, 32, 32, 278, 0, 1, 1, 0, 0, 0, 0, 3, 1,
		0, 0, 0, 0, 5, 1, 0, 0, 0, 0, 7, 1, 0, 0, 0, 0, 9, 1, 0, 0, 0, 0, 11, 1,
		0, 0, 0, 0, 13, 1, 0, 0, 0, 0, 15, 1, 0, 0, 0, 0, 17, 1, 0, 0, 0, 0, 19,
		1, 0, 0, 0, 0, 21, 1, 0, 0, 0, 0, 23, 1, 0, 0, 0, 0, 25, 1, 0, 0, 0, 0,
		27, 1, 0, 0, 0, 0, 29, 1, 0, 0, 0, 0, 31, 1, 0, 0, 0, 0, 33, 1, 0, 0, 0,
		0, 35, 1, 0, 0, 0, 0, 37, 1, 0, 0, 0, 0, 39, 1, 0, 0, 0, 0, 41, 1, 0, 0,
		0, 0, 43, 1, 0, 0, 0, 0, 45, 1, 0, 0, 0, 0, 47, 1, 0, 0, 0, 0, 49, 1, 0,
		0, 0, 0, 51, 1, 0, 0, 0, 0, 53, 1, 0, 0, 0, 0, 55, 1, 0, 0, 0, 0, 57, 1,
		0, 0, 0, 0, 59, 1, 0, 0, 0, 0, 61, 1, 0, 0, 0, 0, 63, 1, 0, 0, 0, 0, 65,
		1, 0, 0, 0, 0, 67, 1, 0, 0, 0, 0, 69, 1, 0, 0, 0, 0, 71, 1, 0, 0, 0, 0,
		73, 1, 0, 0, 0, 0, 75, 1, 0, 0, 0, 0, 77, 1, 0, 0, 0, 0, 79, 1, 0, 0, 0,
		0, 81, 1, 0, 0, 0, 0, 83, 1, 0, 0, 0, 1, 85, 1, 0, 0, 0, 3, 90, 1, 0, 0,
		0, 5, 93, 1, 0, 0, 0, 7, 98, 1, 0, 0, 0, 9, 105, 1, 0, 0, 0, 11, 114, 1,
		0, 0, 0, 13, 121, 1, 0, 0, 0, 15, 126, 1, 0, 0, 0, 17, 131, 1, 0, 0, 0,
		19, 136, 1, 0, 0, 0, 21, 141, 1, 0, 0, 0, 23, 147, 1, 0, 0, 0, 25, 150,
		1, 0, 0, 0, 27, 153, 1, 0, 0, 0, 29, 156, 1, 0, 0, 0, 31, 159, 1, 0, 0,
		0, 33, 161, 1, 0, 0, 0, 35, 163, 1, 0, 0, 0, 37, 165, 1, 0, 0, 0, 39, 167,
		1, 0, 0, 0, 41, 169, 1, 0, 0, 0, 43, 172, 1, 0, 0, 0, 45, 175, 1, 0, 0,
		0, 47, 177, 1, 0, 0, 0, 49, 180, 1, 0, 0, 0, 51, 182, 1, 0, 0, 0, 53, 185,
		1, 0, 0, 0, 55, 188, 1, 0, 0, 0, 57, 191, 1, 0, 0, 0, 59, 193, 1, 0, 0,
		0, 61, 195, 1, 0, 0, 0, 63, 197, 1, 0, 0, 0, 65, 199, 1, 0, 0, 0, 67, 201,
		1, 0, 0, 0, 69, 203, 1, 0, 0, 0, 71, 205, 1, 0, 0, 0, 73, 208, 1, 0, 0,
		0, 75, 220, 1, 0, 0, 0, 77, 231, 1, 0, 0, 0, 79, 238, 1, 0, 0, 0, 81, 249,
		1, 0, 0, 0, 83, 264, 1, 0, 0, 0, 85, 86, 5, 102, 0, 0, 86, 87, 5, 117,
		0, 0, 87, 88, 5, 110, 0, 0, 88, 89, 5, 99, 0, 0, 89, 2, 1, 0, 0, 0, 90,
		91, 5, 105, 0, 0, 91, 92, 5, 102, 0, 0, 92, 4, 1, 0, 0, 0, 93, 94, 5, 101,
		0, 0, 94, 95, 5, 108, 0, 0, 95, 96, 5, 115, 0, 0, 96, 97, 5, 101, 0, 0,
		97, 6, 1, 0, 0, 0, 98, 99, 5, 114, 0, 0, 99, 100, 5, 101, 0, 0, 100, 101,
		5, 116, 0, 0, 101, 102, 5, 117, 0, 0, 102, 103, 5, 114, 0, 0, 103, 104,
		5, 110, 0, 0, 104, 8, 1, 0, 0, 0, 105, 106, 5, 105, 0, 0, 106, 107, 5,
		110, 0, 0, 107, 108, 5, 116, 0, 0, 108, 109, 5, 101, 0, 0, 109, 110, 5,
		114, 0, 0, 110, 111, 5, 118, 0, 0, 111, 112, 5, 97, 0, 0, 112, 113, 5,
		108, 0, 0, 113, 10, 1, 0, 0, 0, 114, 115, 5, 110, 0, 0, 115, 116, 5, 117,
		0, 0, 116, 117, 5, 109, 0, 0, 117, 118, 5, 98, 0, 0, 118, 119, 5, 101,
		0, 0, 119, 120, 5, 114, 0, 0, 120, 12, 1, 0, 0, 0, 121, 122, 5, 99, 0,
		0, 122, 123, 5, 104, 0, 0, 123, 124, 5, 97, 0, 0, 124, 125, 5, 110, 0,
		0, 125, 14, 1, 0, 0, 0, 126, 127, 5, 118, 0, 0, 127, 128, 5, 111, 0, 0,
		128, 129, 5, 105, 0, 0, 129, 130, 5, 100, 0, 0, 130, 16, 1, 0, 0, 0, 131,
		132, 5, 98, 0, 0, 132, 133, 5, 111, 0, 0, 133, 134, 5, 111, 0, 0, 134,
		135, 5, 108, 0, 0, 135, 18, 1, 0, 0, 0, 136, 137, 5, 116, 0, 0, 137, 138,
		5, 114, 0, 0, 138, 139, 5, 117, 0, 0, 139, 140, 5, 101, 0, 0, 140, 20,
		1, 0, 0, 0, 141, 142, 5, 102, 0, 0, 142, 143, 5, 97, 0, 0, 143, 144, 5,
		108, 0, 0, 144, 145, 5, 115, 0, 0, 145, 146, 5, 101, 0, 0, 146, 22, 1,
		0, 0, 0, 147, 148, 5, 45, 0, 0, 148, 149, 5, 62, 0, 0, 149, 24, 1, 0, 0,
		0, 150, 151, 5, 60, 0, 0, 151, 152, 5, 45, 0, 0, 152, 26, 1, 0, 0, 0, 153,
		154, 5, 58, 0, 0, 154, 155, 5, 61, 0, 0, 155, 28, 1, 0, 0, 0, 156, 157,
		5, 36, 0, 0, 157, 158, 5, 61, 0, 0, 158, 30, 1, 0, 0, 0, 159, 160, 5, 61,
		0, 0, 160, 32, 1, 0, 0, 0, 161, 162, 5, 43, 0, 0, 162, 34, 1, 0, 0, 0,
		163, 164, 5, 45, 0, 0, 164, 36, 1, 0, 0, 0, 165, 166, 5, 42, 0, 0, 166,
		38, 1, 0, 0, 0, 167, 168, 5, 47, 0, 0, 168, 40, 1, 0, 0, 0, 169, 170, 5,
		61, 0, 0, 170, 171, 5, 61, 0, 0, 171, 42, 1, 0, 0, 0, 172, 173, 5, 33,
		0, 0, 173, 174, 5, 61, 0, 0, 174, 44, 1, 0, 0, 0, 175, 176, 5, 60, 0, 0,
		176, 46, 1, 0, 0, 0, 177, 178, 5, 60, 0, 0, 178, 179, 5, 61, 0, 0, 179,
		48, 1, 0, 0, 0, 180, 181, 5, 62, 0, 0, 181, 50, 1, 0, 0, 0, 182, 183, 5,
		62, 0, 0, 183, 184, 5, 61, 0, 0, 184, 52, 1, 0, 0, 0, 185, 186, 5, 38,
		0, 0, 186, 187, 5, 38, 0, 0, 187, 54, 1, 0, 0, 0, 188, 189, 5, 124, 0,
		0, 189, 190, 5, 124, 0, 0, 190, 56, 1, 0, 0, 0, 191, 192, 5, 33, 0, 0,
		192, 58, 1, 0, 0, 0, 193, 194, 5, 40, 0, 0, 194, 60, 1, 0, 0, 0, 195, 196,
		5, 41, 0, 0, 196, 62, 1, 0, 0, 0, 197, 198, 5, 123, 0, 0, 198, 64, 1, 0,
		0, 0, 199, 200, 5, 125, 0, 0, 200, 66, 1, 0, 0, 0, 201, 202, 5, 91, 0,
		0, 202, 68, 1, 0, 0, 0, 203, 204, 5, 93, 0, 0, 204, 70, 1, 0, 0, 0, 205,
		206, 5, 44, 0, 0, 206, 72, 1, 0, 0, 0, 207, 209, 7, 0, 0, 0, 208, 207,
		1, 0, 0, 0, 209, 210, 1, 0, 0, 0, 210, 208, 1, 0, 0, 0, 210, 211, 1, 0,
		0, 0, 211, 218, 1, 0, 0, 0, 212, 214, 5, 46, 0, 0, 213, 215, 7, 0, 0, 0,
		214, 213, 1, 0, 0, 0, 215, 216, 1, 0, 0, 0, 216, 214, 1, 0, 0, 0, 216,
		217, 1, 0, 0, 0, 217, 219, 1, 0, 0, 0, 218, 212, 1, 0, 0, 0, 218, 219,
		1, 0, 0, 0, 219, 74, 1, 0, 0, 0, 220, 226, 5, 34, 0, 0, 221, 225, 8, 1,
		0, 0, 222, 223, 5, 92, 0, 0, 223, 225, 9, 0, 0, 0, 224, 221, 1, 0, 0, 0,
		224, 222, 1, 0, 0, 0, 225, 228, 1, 0, 0, 0, 226, 224, 1, 0, 0, 0, 226,
		227, 1, 0, 0, 0, 227, 229, 1, 0, 0, 0, 228, 226, 1, 0, 0, 0, 229, 230,
		5, 34, 0, 0, 230, 76, 1, 0, 0, 0, 231, 235, 7, 2, 0, 0, 232, 234, 7, 3,
		0, 0, 233, 232, 1, 0, 0, 0, 234, 237, 1, 0, 0, 0, 235, 233, 1, 0, 0, 0,
		235, 236, 1, 0, 0, 0, 236, 78, 1, 0, 0, 0, 237, 235, 1, 0, 0, 0, 238, 239,
		5, 47, 0, 0, 239, 240, 5, 47, 0, 0, 240, 244, 1, 0, 0, 0, 241, 243, 8,
		4, 0, 0, 242, 241, 1, 0, 0, 0, 243, 246, 1, 0, 0, 0, 244, 242, 1, 0, 0,
		0, 244, 245, 1, 0, 0, 0, 245, 247, 1, 0, 0, 0, 246, 244, 1, 0, 0, 0, 247,
		248, 6, 39, 0, 0, 248, 80, 1, 0, 0, 0, 249, 250, 5, 47, 0, 0, 250, 251,
		5, 42, 0, 0, 251, 255, 1, 0, 0, 0, 252, 254, 9, 0, 0, 0, 253, 252, 1, 0,
		0, 0, 254, 257, 1, 0, 0, 0, 255, 256, 1, 0, 0, 0, 255, 253, 1, 0, 0, 0,
		256, 258, 1, 0, 0, 0, 257, 255, 1, 0, 0, 0, 258, 259, 5, 42, 0, 0, 259,
		260, 5, 47, 0, 0, 260, 261, 1, 0, 0, 0, 261, 262, 6, 40, 0, 0, 262, 82,
		1, 0, 0, 0, 263, 265, 7, 5, 0, 0, 264, 263, 1, 0, 0, 0, 265, 266, 1, 0,
		0, 0, 266, 264, 1, 0, 0, 0, 266, 267, 1, 0, 0, 0, 267, 268, 1, 0, 0, 0,
		268, 269, 6, 41, 0, 0, 269, 84, 1, 0, 0, 0, 10, 0, 210, 216, 218, 224,
		226, 235, 244, 255, 266, 1, 6, 0, 0,
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

// SlateLexerInit initializes any static state used to implement SlateLexer. By default the
// static state used to implement the lexer is lazily initialized during the first call to
// NewSlateLexer(). You can call this function if you wish to initialize the static state ahead
// of time.
func SlateLexerInit() {
	staticData := &SlateLexerLexerStaticData
	staticData.once.Do(slatelexerLexerInit)
}

// NewSlateLexer produces a new lexer instance for the optional input antlr.CharStream.
func NewSlateLexer(input antlr.CharStream) *SlateLexer {
	SlateLexerInit()
	l := new(SlateLexer)
	l.BaseLexer = antlr.NewBaseLexer(input)
	staticData := &SlateLexerLexerStaticData
	l.Interpreter = antlr.NewLexerATNSimulator(l, staticData.atn, staticData.decisionToDFA, staticData.PredictionContextCache)
	l.channelNames = staticData.ChannelNames
	l.modeNames = staticData.ModeNames
	l.RuleNames = staticData.RuleNames
	l.LiteralNames = staticData.LiteralNames
	l.SymbolicNames = staticData.SymbolicNames
	l.GrammarFileName = "SlateLexer.g4"
	// TODO: l.EOF = antlr.TokenEOF

	return l
}

// SlateLexer tokens.
const (
	SlateLexerFUNC           = 1
	SlateLexerIF             = 2
	SlateLexerELSE           = 3
	SlateLexerRETURN         = 4
	SlateLexerINTERVAL       = 5
	SlateLexerNUMBER         = 6
	SlateLexerCHAN           = 7
	SlateLexerVOID           = 8
	SlateLexerBOOL           = 9
	SlateLexerTRUE           = 10
	SlateLexerFALSE          = 11
	SlateLexerCHANNEL_SEND   = 12
	SlateLexerCHANNEL_RECV   = 13
	SlateLexerLOCAL_ASSIGN   = 14
	SlateLexerSTATE_ASSIGN   = 15
	SlateLexerASSIGN         = 16
	SlateLexerPLUS           = 17
	SlateLexerMINUS          = 18
	SlateLexerMULTIPLY       = 19
	SlateLexerDIVIDE         = 20
	SlateLexerEQUAL          = 21
	SlateLexerNOT_EQUAL      = 22
	SlateLexerLESS_THAN      = 23
	SlateLexerLESS_EQUAL     = 24
	SlateLexerGREATER_THAN   = 25
	SlateLexerGREATER_EQUAL  = 26
	SlateLexerAND            = 27
	SlateLexerOR             = 28
	SlateLexerNOT            = 29
	SlateLexerLPAREN         = 30
	SlateLexerRPAREN         = 31
	SlateLexerLBRACE         = 32
	SlateLexerRBRACE         = 33
	SlateLexerLBRACKET       = 34
	SlateLexerRBRACKET       = 35
	SlateLexerCOMMA          = 36
	SlateLexerNUMBER_LITERAL = 37
	SlateLexerSTRING         = 38
	SlateLexerIDENTIFIER     = 39
	SlateLexerLINE_COMMENT   = 40
	SlateLexerBLOCK_COMMENT  = 41
	SlateLexerWS             = 42
)
