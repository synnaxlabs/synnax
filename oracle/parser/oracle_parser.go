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
		"defaultValue", "arrayDefault", "structDefault", "structFieldDefault",
		"inlineDomain", "fieldBody", "domain", "domainContent", "domainBlock",
		"typeRef", "arrayModifier", "mapType", "typeArgs", "typeModifiers",
		"qualifiedIdent", "expression", "expressionValue", "objectLiteral",
		"objectField", "enumDef", "enumBody", "enumValue", "enumValueBody",
		"typeDefDef", "typeDefBody",
	}
	staticData.PredictionContextCache = antlr.NewPredictionContextCache()
	staticData.serializedATN = []int32{
		4, 1, 28, 703, 2, 0, 7, 0, 2, 1, 7, 1, 2, 2, 7, 2, 2, 3, 7, 3, 2, 4, 7,
		4, 2, 5, 7, 5, 2, 6, 7, 6, 2, 7, 7, 7, 2, 8, 7, 8, 2, 9, 7, 9, 2, 10, 7,
		10, 2, 11, 7, 11, 2, 12, 7, 12, 2, 13, 7, 13, 2, 14, 7, 14, 2, 15, 7, 15,
		2, 16, 7, 16, 2, 17, 7, 17, 2, 18, 7, 18, 2, 19, 7, 19, 2, 20, 7, 20, 2,
		21, 7, 21, 2, 22, 7, 22, 2, 23, 7, 23, 2, 24, 7, 24, 2, 25, 7, 25, 2, 26,
		7, 26, 2, 27, 7, 27, 2, 28, 7, 28, 2, 29, 7, 29, 2, 30, 7, 30, 2, 31, 7,
		31, 2, 32, 7, 32, 2, 33, 7, 33, 2, 34, 7, 34, 2, 35, 7, 35, 2, 36, 7, 36,
		2, 37, 7, 37, 2, 38, 7, 38, 2, 39, 7, 39, 2, 40, 7, 40, 2, 41, 7, 41, 1,
		0, 5, 0, 86, 8, 0, 10, 0, 12, 0, 89, 9, 0, 1, 0, 1, 0, 5, 0, 93, 8, 0,
		10, 0, 12, 0, 96, 9, 0, 5, 0, 98, 8, 0, 10, 0, 12, 0, 101, 9, 0, 1, 0,
		1, 0, 5, 0, 105, 8, 0, 10, 0, 12, 0, 108, 9, 0, 5, 0, 110, 8, 0, 10, 0,
		12, 0, 113, 9, 0, 1, 0, 1, 0, 5, 0, 117, 8, 0, 10, 0, 12, 0, 120, 9, 0,
		5, 0, 122, 8, 0, 10, 0, 12, 0, 125, 9, 0, 1, 0, 1, 0, 1, 1, 1, 1, 1, 2,
		1, 2, 1, 2, 1, 3, 1, 3, 1, 3, 3, 3, 137, 8, 3, 1, 4, 1, 4, 1, 4, 3, 4,
		142, 8, 4, 1, 5, 1, 5, 1, 5, 3, 5, 147, 8, 5, 1, 5, 1, 5, 3, 5, 151, 8,
		5, 1, 5, 5, 5, 154, 8, 5, 10, 5, 12, 5, 157, 9, 5, 1, 5, 1, 5, 5, 5, 161,
		8, 5, 10, 5, 12, 5, 164, 9, 5, 1, 5, 1, 5, 1, 5, 1, 5, 1, 5, 3, 5, 171,
		8, 5, 1, 5, 1, 5, 1, 5, 3, 5, 176, 8, 5, 3, 5, 178, 8, 5, 1, 6, 1, 6, 1,
		6, 5, 6, 183, 8, 6, 10, 6, 12, 6, 186, 9, 6, 1, 6, 5, 6, 189, 8, 6, 10,
		6, 12, 6, 192, 9, 6, 1, 7, 5, 7, 195, 8, 7, 10, 7, 12, 7, 198, 9, 7, 1,
		7, 1, 7, 5, 7, 202, 8, 7, 10, 7, 12, 7, 205, 9, 7, 1, 7, 1, 7, 5, 7, 209,
		8, 7, 10, 7, 12, 7, 212, 9, 7, 5, 7, 214, 8, 7, 10, 7, 12, 7, 217, 9, 7,
		1, 7, 1, 7, 1, 8, 1, 8, 5, 8, 223, 8, 8, 10, 8, 12, 8, 226, 9, 8, 1, 8,
		1, 8, 1, 8, 5, 8, 231, 8, 8, 10, 8, 12, 8, 234, 9, 8, 1, 8, 5, 8, 237,
		8, 8, 10, 8, 12, 8, 240, 9, 8, 1, 8, 5, 8, 243, 8, 8, 10, 8, 12, 8, 246,
		9, 8, 1, 8, 1, 8, 1, 9, 1, 9, 3, 9, 252, 8, 9, 1, 9, 1, 9, 3, 9, 256, 8,
		9, 1, 9, 1, 9, 3, 9, 260, 8, 9, 1, 10, 1, 10, 1, 10, 1, 10, 3, 10, 266,
		8, 10, 1, 10, 5, 10, 269, 8, 10, 10, 10, 12, 10, 272, 9, 10, 5, 10, 274,
		8, 10, 10, 10, 12, 10, 277, 9, 10, 1, 11, 1, 11, 1, 11, 1, 12, 1, 12, 1,
		12, 1, 12, 1, 13, 1, 13, 1, 13, 5, 13, 289, 8, 13, 10, 13, 12, 13, 292,
		9, 13, 1, 13, 1, 13, 5, 13, 296, 8, 13, 10, 13, 12, 13, 299, 9, 13, 1,
		13, 1, 13, 1, 13, 1, 14, 1, 14, 3, 14, 306, 8, 14, 1, 14, 5, 14, 309, 8,
		14, 10, 14, 12, 14, 312, 9, 14, 5, 14, 314, 8, 14, 10, 14, 12, 14, 317,
		9, 14, 1, 15, 1, 15, 1, 15, 3, 15, 322, 8, 15, 1, 15, 1, 15, 3, 15, 326,
		8, 15, 1, 15, 1, 15, 5, 15, 330, 8, 15, 10, 15, 12, 15, 333, 9, 15, 1,
		15, 3, 15, 336, 8, 15, 1, 16, 1, 16, 1, 16, 3, 16, 341, 8, 16, 1, 17, 1,
		17, 1, 17, 3, 17, 346, 8, 17, 1, 18, 1, 18, 5, 18, 350, 8, 18, 10, 18,
		12, 18, 353, 9, 18, 1, 18, 1, 18, 1, 18, 5, 18, 358, 8, 18, 10, 18, 12,
		18, 361, 9, 18, 1, 18, 5, 18, 364, 8, 18, 10, 18, 12, 18, 367, 9, 18, 1,
		18, 5, 18, 370, 8, 18, 10, 18, 12, 18, 373, 9, 18, 3, 18, 375, 8, 18, 1,
		18, 1, 18, 1, 19, 1, 19, 5, 19, 381, 8, 19, 10, 19, 12, 19, 384, 9, 19,
		1, 19, 1, 19, 1, 19, 5, 19, 389, 8, 19, 10, 19, 12, 19, 392, 9, 19, 1,
		19, 5, 19, 395, 8, 19, 10, 19, 12, 19, 398, 9, 19, 1, 19, 5, 19, 401, 8,
		19, 10, 19, 12, 19, 404, 9, 19, 3, 19, 406, 8, 19, 1, 19, 1, 19, 1, 20,
		1, 20, 1, 20, 1, 20, 1, 21, 1, 21, 1, 21, 3, 21, 417, 8, 21, 1, 22, 5,
		22, 420, 8, 22, 10, 22, 12, 22, 423, 9, 22, 1, 22, 1, 22, 5, 22, 427, 8,
		22, 10, 22, 12, 22, 430, 9, 22, 1, 22, 1, 22, 3, 22, 434, 8, 22, 1, 22,
		5, 22, 437, 8, 22, 10, 22, 12, 22, 440, 9, 22, 5, 22, 442, 8, 22, 10, 22,
		12, 22, 445, 9, 22, 1, 22, 1, 22, 1, 23, 1, 23, 1, 23, 3, 23, 452, 8, 23,
		1, 24, 1, 24, 3, 24, 456, 8, 24, 1, 25, 5, 25, 459, 8, 25, 10, 25, 12,
		25, 462, 9, 25, 1, 25, 1, 25, 5, 25, 466, 8, 25, 10, 25, 12, 25, 469, 9,
		25, 1, 25, 1, 25, 4, 25, 473, 8, 25, 11, 25, 12, 25, 474, 1, 25, 1, 25,
		5, 25, 479, 8, 25, 10, 25, 12, 25, 482, 9, 25, 3, 25, 484, 8, 25, 1, 25,
		5, 25, 487, 8, 25, 10, 25, 12, 25, 490, 9, 25, 1, 25, 1, 25, 1, 26, 1,
		26, 3, 26, 496, 8, 26, 1, 26, 1, 26, 3, 26, 500, 8, 26, 1, 26, 3, 26, 503,
		8, 26, 1, 26, 3, 26, 506, 8, 26, 3, 26, 508, 8, 26, 1, 27, 1, 27, 1, 27,
		1, 27, 1, 27, 3, 27, 515, 8, 27, 1, 28, 1, 28, 1, 28, 1, 28, 1, 28, 1,
		28, 1, 28, 1, 29, 1, 29, 1, 29, 1, 29, 5, 29, 528, 8, 29, 10, 29, 12, 29,
		531, 9, 29, 1, 29, 1, 29, 1, 30, 1, 30, 1, 30, 3, 30, 538, 8, 30, 1, 31,
		1, 31, 1, 31, 3, 31, 543, 8, 31, 1, 32, 1, 32, 5, 32, 547, 8, 32, 10, 32,
		12, 32, 550, 9, 32, 1, 33, 1, 33, 1, 33, 1, 33, 1, 33, 1, 33, 1, 33, 3,
		33, 559, 8, 33, 1, 34, 1, 34, 5, 34, 563, 8, 34, 10, 34, 12, 34, 566, 9,
		34, 1, 34, 1, 34, 1, 34, 5, 34, 571, 8, 34, 10, 34, 12, 34, 574, 9, 34,
		1, 34, 5, 34, 577, 8, 34, 10, 34, 12, 34, 580, 9, 34, 3, 34, 582, 8, 34,
		1, 34, 3, 34, 585, 8, 34, 1, 34, 5, 34, 588, 8, 34, 10, 34, 12, 34, 591,
		9, 34, 1, 34, 1, 34, 1, 35, 1, 35, 1, 35, 1, 36, 1, 36, 1, 36, 1, 36, 3,
		36, 602, 8, 36, 1, 36, 5, 36, 605, 8, 36, 10, 36, 12, 36, 608, 9, 36, 1,
		36, 1, 36, 5, 36, 612, 8, 36, 10, 36, 12, 36, 615, 9, 36, 1, 36, 1, 36,
		1, 36, 1, 37, 1, 37, 3, 37, 622, 8, 37, 1, 37, 5, 37, 625, 8, 37, 10, 37,
		12, 37, 628, 9, 37, 5, 37, 630, 8, 37, 10, 37, 12, 37, 633, 9, 37, 1, 38,
		1, 38, 1, 38, 1, 38, 3, 38, 639, 8, 38, 1, 39, 5, 39, 642, 8, 39, 10, 39,
		12, 39, 645, 9, 39, 1, 39, 1, 39, 5, 39, 649, 8, 39, 10, 39, 12, 39, 652,
		9, 39, 1, 39, 1, 39, 5, 39, 656, 8, 39, 10, 39, 12, 39, 659, 9, 39, 5,
		39, 661, 8, 39, 10, 39, 12, 39, 664, 9, 39, 1, 39, 1, 39, 1, 40, 1, 40,
		3, 40, 670, 8, 40, 1, 40, 1, 40, 3, 40, 674, 8, 40, 1, 41, 5, 41, 677,
		8, 41, 10, 41, 12, 41, 680, 9, 41, 1, 41, 1, 41, 5, 41, 684, 8, 41, 10,
		41, 12, 41, 687, 9, 41, 1, 41, 1, 41, 5, 41, 691, 8, 41, 10, 41, 12, 41,
		694, 9, 41, 5, 41, 696, 8, 41, 10, 41, 12, 41, 699, 9, 41, 1, 41, 1, 41,
		1, 41, 0, 0, 42, 0, 2, 4, 6, 8, 10, 12, 14, 16, 18, 20, 22, 24, 26, 28,
		30, 32, 34, 36, 38, 40, 42, 44, 46, 48, 50, 52, 54, 56, 58, 60, 62, 64,
		66, 68, 70, 72, 74, 76, 78, 80, 82, 0, 1, 2, 0, 20, 20, 22, 22, 773, 0,
		87, 1, 0, 0, 0, 2, 128, 1, 0, 0, 0, 4, 130, 1, 0, 0, 0, 6, 133, 1, 0, 0,
		0, 8, 141, 1, 0, 0, 0, 10, 177, 1, 0, 0, 0, 12, 179, 1, 0, 0, 0, 14, 196,
		1, 0, 0, 0, 16, 220, 1, 0, 0, 0, 18, 249, 1, 0, 0, 0, 20, 275, 1, 0, 0,
		0, 22, 278, 1, 0, 0, 0, 24, 281, 1, 0, 0, 0, 26, 285, 1, 0, 0, 0, 28, 315,
		1, 0, 0, 0, 30, 318, 1, 0, 0, 0, 32, 340, 1, 0, 0, 0, 34, 345, 1, 0, 0,
		0, 36, 347, 1, 0, 0, 0, 38, 378, 1, 0, 0, 0, 40, 409, 1, 0, 0, 0, 42, 413,
		1, 0, 0, 0, 44, 421, 1, 0, 0, 0, 46, 448, 1, 0, 0, 0, 48, 455, 1, 0, 0,
		0, 50, 460, 1, 0, 0, 0, 52, 507, 1, 0, 0, 0, 54, 514, 1, 0, 0, 0, 56, 516,
		1, 0, 0, 0, 58, 523, 1, 0, 0, 0, 60, 537, 1, 0, 0, 0, 62, 539, 1, 0, 0,
		0, 64, 544, 1, 0, 0, 0, 66, 558, 1, 0, 0, 0, 68, 560, 1, 0, 0, 0, 70, 594,
		1, 0, 0, 0, 72, 597, 1, 0, 0, 0, 74, 631, 1, 0, 0, 0, 76, 634, 1, 0, 0,
		0, 78, 643, 1, 0, 0, 0, 80, 667, 1, 0, 0, 0, 82, 678, 1, 0, 0, 0, 84, 86,
		3, 2, 1, 0, 85, 84, 1, 0, 0, 0, 86, 89, 1, 0, 0, 0, 87, 85, 1, 0, 0, 0,
		87, 88, 1, 0, 0, 0, 88, 99, 1, 0, 0, 0, 89, 87, 1, 0, 0, 0, 90, 94, 3,
		4, 2, 0, 91, 93, 3, 2, 1, 0, 92, 91, 1, 0, 0, 0, 93, 96, 1, 0, 0, 0, 94,
		92, 1, 0, 0, 0, 94, 95, 1, 0, 0, 0, 95, 98, 1, 0, 0, 0, 96, 94, 1, 0, 0,
		0, 97, 90, 1, 0, 0, 0, 98, 101, 1, 0, 0, 0, 99, 97, 1, 0, 0, 0, 99, 100,
		1, 0, 0, 0, 100, 111, 1, 0, 0, 0, 101, 99, 1, 0, 0, 0, 102, 106, 3, 6,
		3, 0, 103, 105, 3, 2, 1, 0, 104, 103, 1, 0, 0, 0, 105, 108, 1, 0, 0, 0,
		106, 104, 1, 0, 0, 0, 106, 107, 1, 0, 0, 0, 107, 110, 1, 0, 0, 0, 108,
		106, 1, 0, 0, 0, 109, 102, 1, 0, 0, 0, 110, 113, 1, 0, 0, 0, 111, 109,
		1, 0, 0, 0, 111, 112, 1, 0, 0, 0, 112, 123, 1, 0, 0, 0, 113, 111, 1, 0,
		0, 0, 114, 118, 3, 8, 4, 0, 115, 117, 3, 2, 1, 0, 116, 115, 1, 0, 0, 0,
		117, 120, 1, 0, 0, 0, 118, 116, 1, 0, 0, 0, 118, 119, 1, 0, 0, 0, 119,
		122, 1, 0, 0, 0, 120, 118, 1, 0, 0, 0, 121, 114, 1, 0, 0, 0, 122, 125,
		1, 0, 0, 0, 123, 121, 1, 0, 0, 0, 123, 124, 1, 0, 0, 0, 124, 126, 1, 0,
		0, 0, 125, 123, 1, 0, 0, 0, 126, 127, 5, 0, 0, 1, 127, 1, 1, 0, 0, 0, 128,
		129, 5, 27, 0, 0, 129, 3, 1, 0, 0, 0, 130, 131, 5, 3, 0, 0, 131, 132, 5,
		20, 0, 0, 132, 5, 1, 0, 0, 0, 133, 134, 5, 17, 0, 0, 134, 136, 5, 24, 0,
		0, 135, 137, 3, 48, 24, 0, 136, 135, 1, 0, 0, 0, 136, 137, 1, 0, 0, 0,
		137, 7, 1, 0, 0, 0, 138, 142, 3, 10, 5, 0, 139, 142, 3, 72, 36, 0, 140,
		142, 3, 80, 40, 0, 141, 138, 1, 0, 0, 0, 141, 139, 1, 0, 0, 0, 141, 140,
		1, 0, 0, 0, 142, 9, 1, 0, 0, 0, 143, 144, 5, 24, 0, 0, 144, 146, 5, 1,
		0, 0, 145, 147, 3, 16, 8, 0, 146, 145, 1, 0, 0, 0, 146, 147, 1, 0, 0, 0,
		147, 150, 1, 0, 0, 0, 148, 149, 5, 4, 0, 0, 149, 151, 3, 12, 6, 0, 150,
		148, 1, 0, 0, 0, 150, 151, 1, 0, 0, 0, 151, 155, 1, 0, 0, 0, 152, 154,
		3, 2, 1, 0, 153, 152, 1, 0, 0, 0, 154, 157, 1, 0, 0, 0, 155, 153, 1, 0,
		0, 0, 155, 156, 1, 0, 0, 0, 156, 158, 1, 0, 0, 0, 157, 155, 1, 0, 0, 0,
		158, 162, 5, 7, 0, 0, 159, 161, 3, 2, 1, 0, 160, 159, 1, 0, 0, 0, 161,
		164, 1, 0, 0, 0, 162, 160, 1, 0, 0, 0, 162, 163, 1, 0, 0, 0, 163, 165,
		1, 0, 0, 0, 164, 162, 1, 0, 0, 0, 165, 166, 3, 20, 10, 0, 166, 167, 5,
		8, 0, 0, 167, 178, 1, 0, 0, 0, 168, 170, 5, 24, 0, 0, 169, 171, 3, 16,
		8, 0, 170, 169, 1, 0, 0, 0, 170, 171, 1, 0, 0, 0, 171, 172, 1, 0, 0, 0,
		172, 173, 5, 16, 0, 0, 173, 175, 3, 52, 26, 0, 174, 176, 3, 14, 7, 0, 175,
		174, 1, 0, 0, 0, 175, 176, 1, 0, 0, 0, 176, 178, 1, 0, 0, 0, 177, 143,
		1, 0, 0, 0, 177, 168, 1, 0, 0, 0, 178, 11, 1, 0, 0, 0, 179, 190, 3, 52,
		26, 0, 180, 184, 5, 13, 0, 0, 181, 183, 3, 2, 1, 0, 182, 181, 1, 0, 0,
		0, 183, 186, 1, 0, 0, 0, 184, 182, 1, 0, 0, 0, 184, 185, 1, 0, 0, 0, 185,
		187, 1, 0, 0, 0, 186, 184, 1, 0, 0, 0, 187, 189, 3, 52, 26, 0, 188, 180,
		1, 0, 0, 0, 189, 192, 1, 0, 0, 0, 190, 188, 1, 0, 0, 0, 190, 191, 1, 0,
		0, 0, 191, 13, 1, 0, 0, 0, 192, 190, 1, 0, 0, 0, 193, 195, 3, 2, 1, 0,
		194, 193, 1, 0, 0, 0, 195, 198, 1, 0, 0, 0, 196, 194, 1, 0, 0, 0, 196,
		197, 1, 0, 0, 0, 197, 199, 1, 0, 0, 0, 198, 196, 1, 0, 0, 0, 199, 203,
		5, 7, 0, 0, 200, 202, 3, 2, 1, 0, 201, 200, 1, 0, 0, 0, 202, 205, 1, 0,
		0, 0, 203, 201, 1, 0, 0, 0, 203, 204, 1, 0, 0, 0, 204, 215, 1, 0, 0, 0,
		205, 203, 1, 0, 0, 0, 206, 210, 3, 46, 23, 0, 207, 209, 3, 2, 1, 0, 208,
		207, 1, 0, 0, 0, 209, 212, 1, 0, 0, 0, 210, 208, 1, 0, 0, 0, 210, 211,
		1, 0, 0, 0, 211, 214, 1, 0, 0, 0, 212, 210, 1, 0, 0, 0, 213, 206, 1, 0,
		0, 0, 214, 217, 1, 0, 0, 0, 215, 213, 1, 0, 0, 0, 215, 216, 1, 0, 0, 0,
		216, 218, 1, 0, 0, 0, 217, 215, 1, 0, 0, 0, 218, 219, 5, 8, 0, 0, 219,
		15, 1, 0, 0, 0, 220, 224, 5, 11, 0, 0, 221, 223, 3, 2, 1, 0, 222, 221,
		1, 0, 0, 0, 223, 226, 1, 0, 0, 0, 224, 222, 1, 0, 0, 0, 224, 225, 1, 0,
		0, 0, 225, 227, 1, 0, 0, 0, 226, 224, 1, 0, 0, 0, 227, 238, 3, 18, 9, 0,
		228, 232, 5, 13, 0, 0, 229, 231, 3, 2, 1, 0, 230, 229, 1, 0, 0, 0, 231,
		234, 1, 0, 0, 0, 232, 230, 1, 0, 0, 0, 232, 233, 1, 0, 0, 0, 233, 235,
		1, 0, 0, 0, 234, 232, 1, 0, 0, 0, 235, 237, 3, 18, 9, 0, 236, 228, 1, 0,
		0, 0, 237, 240, 1, 0, 0, 0, 238, 236, 1, 0, 0, 0, 238, 239, 1, 0, 0, 0,
		239, 244, 1, 0, 0, 0, 240, 238, 1, 0, 0, 0, 241, 243, 3, 2, 1, 0, 242,
		241, 1, 0, 0, 0, 243, 246, 1, 0, 0, 0, 244, 242, 1, 0, 0, 0, 244, 245,
		1, 0, 0, 0, 245, 247, 1, 0, 0, 0, 246, 244, 1, 0, 0, 0, 247, 248, 5, 12,
		0, 0, 248, 17, 1, 0, 0, 0, 249, 251, 5, 24, 0, 0, 250, 252, 5, 14, 0, 0,
		251, 250, 1, 0, 0, 0, 251, 252, 1, 0, 0, 0, 252, 255, 1, 0, 0, 0, 253,
		254, 5, 4, 0, 0, 254, 256, 3, 52, 26, 0, 255, 253, 1, 0, 0, 0, 255, 256,
		1, 0, 0, 0, 256, 259, 1, 0, 0, 0, 257, 258, 5, 16, 0, 0, 258, 260, 3, 52,
		26, 0, 259, 257, 1, 0, 0, 0, 259, 260, 1, 0, 0, 0, 260, 19, 1, 0, 0, 0,
		261, 266, 3, 30, 15, 0, 262, 266, 3, 22, 11, 0, 263, 266, 3, 26, 13, 0,
		264, 266, 3, 46, 23, 0, 265, 261, 1, 0, 0, 0, 265, 262, 1, 0, 0, 0, 265,
		263, 1, 0, 0, 0, 265, 264, 1, 0, 0, 0, 266, 270, 1, 0, 0, 0, 267, 269,
		3, 2, 1, 0, 268, 267, 1, 0, 0, 0, 269, 272, 1, 0, 0, 0, 270, 268, 1, 0,
		0, 0, 270, 271, 1, 0, 0, 0, 271, 274, 1, 0, 0, 0, 272, 270, 1, 0, 0, 0,
		273, 265, 1, 0, 0, 0, 274, 277, 1, 0, 0, 0, 275, 273, 1, 0, 0, 0, 275,
		276, 1, 0, 0, 0, 276, 21, 1, 0, 0, 0, 277, 275, 1, 0, 0, 0, 278, 279, 5,
		18, 0, 0, 279, 280, 5, 24, 0, 0, 280, 23, 1, 0, 0, 0, 281, 282, 5, 18,
		0, 0, 282, 283, 5, 17, 0, 0, 283, 284, 5, 24, 0, 0, 284, 25, 1, 0, 0, 0,
		285, 286, 5, 6, 0, 0, 286, 290, 5, 24, 0, 0, 287, 289, 3, 2, 1, 0, 288,
		287, 1, 0, 0, 0, 289, 292, 1, 0, 0, 0, 290, 288, 1, 0, 0, 0, 290, 291,
		1, 0, 0, 0, 291, 293, 1, 0, 0, 0, 292, 290, 1, 0, 0, 0, 293, 297, 5, 7,
		0, 0, 294, 296, 3, 2, 1, 0, 295, 294, 1, 0, 0, 0, 296, 299, 1, 0, 0, 0,
		297, 295, 1, 0, 0, 0, 297, 298, 1, 0, 0, 0, 298, 300, 1, 0, 0, 0, 299,
		297, 1, 0, 0, 0, 300, 301, 3, 28, 14, 0, 301, 302, 5, 8, 0, 0, 302, 27,
		1, 0, 0, 0, 303, 306, 3, 30, 15, 0, 304, 306, 3, 46, 23, 0, 305, 303, 1,
		0, 0, 0, 305, 304, 1, 0, 0, 0, 306, 310, 1, 0, 0, 0, 307, 309, 3, 2, 1,
		0, 308, 307, 1, 0, 0, 0, 309, 312, 1, 0, 0, 0, 310, 308, 1, 0, 0, 0, 310,
		311, 1, 0, 0, 0, 311, 314, 1, 0, 0, 0, 312, 310, 1, 0, 0, 0, 313, 305,
		1, 0, 0, 0, 314, 317, 1, 0, 0, 0, 315, 313, 1, 0, 0, 0, 315, 316, 1, 0,
		0, 0, 316, 29, 1, 0, 0, 0, 317, 315, 1, 0, 0, 0, 318, 321, 5, 24, 0, 0,
		319, 322, 3, 52, 26, 0, 320, 322, 3, 60, 30, 0, 321, 319, 1, 0, 0, 0, 321,
		320, 1, 0, 0, 0, 321, 322, 1, 0, 0, 0, 322, 325, 1, 0, 0, 0, 323, 324,
		5, 16, 0, 0, 324, 326, 3, 32, 16, 0, 325, 323, 1, 0, 0, 0, 325, 326, 1,
		0, 0, 0, 326, 331, 1, 0, 0, 0, 327, 330, 3, 42, 21, 0, 328, 330, 3, 24,
		12, 0, 329, 327, 1, 0, 0, 0, 329, 328, 1, 0, 0, 0, 330, 333, 1, 0, 0, 0,
		331, 329, 1, 0, 0, 0, 331, 332, 1, 0, 0, 0, 332, 335, 1, 0, 0, 0, 333,
		331, 1, 0, 0, 0, 334, 336, 3, 44, 22, 0, 335, 334, 1, 0, 0, 0, 335, 336,
		1, 0, 0, 0, 336, 31, 1, 0, 0, 0, 337, 341, 3, 66, 33, 0, 338, 341, 3, 36,
		18, 0, 339, 341, 3, 38, 19, 0, 340, 337, 1, 0, 0, 0, 340, 338, 1, 0, 0,
		0, 340, 339, 1, 0, 0, 0, 341, 33, 1, 0, 0, 0, 342, 346, 3, 66, 33, 0, 343,
		346, 3, 36, 18, 0, 344, 346, 3, 38, 19, 0, 345, 342, 1, 0, 0, 0, 345, 343,
		1, 0, 0, 0, 345, 344, 1, 0, 0, 0, 346, 35, 1, 0, 0, 0, 347, 351, 5, 9,
		0, 0, 348, 350, 3, 2, 1, 0, 349, 348, 1, 0, 0, 0, 350, 353, 1, 0, 0, 0,
		351, 349, 1, 0, 0, 0, 351, 352, 1, 0, 0, 0, 352, 374, 1, 0, 0, 0, 353,
		351, 1, 0, 0, 0, 354, 365, 3, 34, 17, 0, 355, 359, 5, 13, 0, 0, 356, 358,
		3, 2, 1, 0, 357, 356, 1, 0, 0, 0, 358, 361, 1, 0, 0, 0, 359, 357, 1, 0,
		0, 0, 359, 360, 1, 0, 0, 0, 360, 362, 1, 0, 0, 0, 361, 359, 1, 0, 0, 0,
		362, 364, 3, 34, 17, 0, 363, 355, 1, 0, 0, 0, 364, 367, 1, 0, 0, 0, 365,
		363, 1, 0, 0, 0, 365, 366, 1, 0, 0, 0, 366, 371, 1, 0, 0, 0, 367, 365,
		1, 0, 0, 0, 368, 370, 3, 2, 1, 0, 369, 368, 1, 0, 0, 0, 370, 373, 1, 0,
		0, 0, 371, 369, 1, 0, 0, 0, 371, 372, 1, 0, 0, 0, 372, 375, 1, 0, 0, 0,
		373, 371, 1, 0, 0, 0, 374, 354, 1, 0, 0, 0, 374, 375, 1, 0, 0, 0, 375,
		376, 1, 0, 0, 0, 376, 377, 5, 10, 0, 0, 377, 37, 1, 0, 0, 0, 378, 382,
		5, 7, 0, 0, 379, 381, 3, 2, 1, 0, 380, 379, 1, 0, 0, 0, 381, 384, 1, 0,
		0, 0, 382, 380, 1, 0, 0, 0, 382, 383, 1, 0, 0, 0, 383, 405, 1, 0, 0, 0,
		384, 382, 1, 0, 0, 0, 385, 396, 3, 40, 20, 0, 386, 390, 5, 13, 0, 0, 387,
		389, 3, 2, 1, 0, 388, 387, 1, 0, 0, 0, 389, 392, 1, 0, 0, 0, 390, 388,
		1, 0, 0, 0, 390, 391, 1, 0, 0, 0, 391, 393, 1, 0, 0, 0, 392, 390, 1, 0,
		0, 0, 393, 395, 3, 40, 20, 0, 394, 386, 1, 0, 0, 0, 395, 398, 1, 0, 0,
		0, 396, 394, 1, 0, 0, 0, 396, 397, 1, 0, 0, 0, 397, 402, 1, 0, 0, 0, 398,
		396, 1, 0, 0, 0, 399, 401, 3, 2, 1, 0, 400, 399, 1, 0, 0, 0, 401, 404,
		1, 0, 0, 0, 402, 400, 1, 0, 0, 0, 402, 403, 1, 0, 0, 0, 403, 406, 1, 0,
		0, 0, 404, 402, 1, 0, 0, 0, 405, 385, 1, 0, 0, 0, 405, 406, 1, 0, 0, 0,
		406, 407, 1, 0, 0, 0, 407, 408, 5, 8, 0, 0, 408, 39, 1, 0, 0, 0, 409, 410,
		5, 24, 0, 0, 410, 411, 5, 16, 0, 0, 411, 412, 3, 34, 17, 0, 412, 41, 1,
		0, 0, 0, 413, 414, 5, 17, 0, 0, 414, 416, 5, 24, 0, 0, 415, 417, 3, 48,
		24, 0, 416, 415, 1, 0, 0, 0, 416, 417, 1, 0, 0, 0, 417, 43, 1, 0, 0, 0,
		418, 420, 3, 2, 1, 0, 419, 418, 1, 0, 0, 0, 420, 423, 1, 0, 0, 0, 421,
		419, 1, 0, 0, 0, 421, 422, 1, 0, 0, 0, 422, 424, 1, 0, 0, 0, 423, 421,
		1, 0, 0, 0, 424, 428, 5, 7, 0, 0, 425, 427, 3, 2, 1, 0, 426, 425, 1, 0,
		0, 0, 427, 430, 1, 0, 0, 0, 428, 426, 1, 0, 0, 0, 428, 429, 1, 0, 0, 0,
		429, 443, 1, 0, 0, 0, 430, 428, 1, 0, 0, 0, 431, 434, 3, 46, 23, 0, 432,
		434, 3, 24, 12, 0, 433, 431, 1, 0, 0, 0, 433, 432, 1, 0, 0, 0, 434, 438,
		1, 0, 0, 0, 435, 437, 3, 2, 1, 0, 436, 435, 1, 0, 0, 0, 437, 440, 1, 0,
		0, 0, 438, 436, 1, 0, 0, 0, 438, 439, 1, 0, 0, 0, 439, 442, 1, 0, 0, 0,
		440, 438, 1, 0, 0, 0, 441, 433, 1, 0, 0, 0, 442, 445, 1, 0, 0, 0, 443,
		441, 1, 0, 0, 0, 443, 444, 1, 0, 0, 0, 444, 446, 1, 0, 0, 0, 445, 443,
		1, 0, 0, 0, 446, 447, 5, 8, 0, 0, 447, 45, 1, 0, 0, 0, 448, 449, 5, 17,
		0, 0, 449, 451, 5, 24, 0, 0, 450, 452, 3, 48, 24, 0, 451, 450, 1, 0, 0,
		0, 451, 452, 1, 0, 0, 0, 452, 47, 1, 0, 0, 0, 453, 456, 3, 50, 25, 0, 454,
		456, 3, 64, 32, 0, 455, 453, 1, 0, 0, 0, 455, 454, 1, 0, 0, 0, 456, 49,
		1, 0, 0, 0, 457, 459, 3, 2, 1, 0, 458, 457, 1, 0, 0, 0, 459, 462, 1, 0,
		0, 0, 460, 458, 1, 0, 0, 0, 460, 461, 1, 0, 0, 0, 461, 463, 1, 0, 0, 0,
		462, 460, 1, 0, 0, 0, 463, 467, 5, 7, 0, 0, 464, 466, 3, 2, 1, 0, 465,
		464, 1, 0, 0, 0, 466, 469, 1, 0, 0, 0, 467, 465, 1, 0, 0, 0, 467, 468,
		1, 0, 0, 0, 468, 483, 1, 0, 0, 0, 469, 467, 1, 0, 0, 0, 470, 480, 3, 64,
		32, 0, 471, 473, 3, 2, 1, 0, 472, 471, 1, 0, 0, 0, 473, 474, 1, 0, 0, 0,
		474, 472, 1, 0, 0, 0, 474, 475, 1, 0, 0, 0, 475, 476, 1, 0, 0, 0, 476,
		477, 3, 64, 32, 0, 477, 479, 1, 0, 0, 0, 478, 472, 1, 0, 0, 0, 479, 482,
		1, 0, 0, 0, 480, 478, 1, 0, 0, 0, 480, 481, 1, 0, 0, 0, 481, 484, 1, 0,
		0, 0, 482, 480, 1, 0, 0, 0, 483, 470, 1, 0, 0, 0, 483, 484, 1, 0, 0, 0,
		484, 488, 1, 0, 0, 0, 485, 487, 3, 2, 1, 0, 486, 485, 1, 0, 0, 0, 487,
		490, 1, 0, 0, 0, 488, 486, 1, 0, 0, 0, 488, 489, 1, 0, 0, 0, 489, 491,
		1, 0, 0, 0, 490, 488, 1, 0, 0, 0, 491, 492, 5, 8, 0, 0, 492, 51, 1, 0,
		0, 0, 493, 495, 3, 56, 28, 0, 494, 496, 3, 60, 30, 0, 495, 494, 1, 0, 0,
		0, 495, 496, 1, 0, 0, 0, 496, 508, 1, 0, 0, 0, 497, 499, 3, 62, 31, 0,
		498, 500, 3, 58, 29, 0, 499, 498, 1, 0, 0, 0, 499, 500, 1, 0, 0, 0, 500,
		502, 1, 0, 0, 0, 501, 503, 3, 54, 27, 0, 502, 501, 1, 0, 0, 0, 502, 503,
		1, 0, 0, 0, 503, 505, 1, 0, 0, 0, 504, 506, 3, 60, 30, 0, 505, 504, 1,
		0, 0, 0, 505, 506, 1, 0, 0, 0, 506, 508, 1, 0, 0, 0, 507, 493, 1, 0, 0,
		0, 507, 497, 1, 0, 0, 0, 508, 53, 1, 0, 0, 0, 509, 510, 5, 9, 0, 0, 510,
		515, 5, 10, 0, 0, 511, 512, 5, 9, 0, 0, 512, 513, 5, 22, 0, 0, 513, 515,
		5, 10, 0, 0, 514, 509, 1, 0, 0, 0, 514, 511, 1, 0, 0, 0, 515, 55, 1, 0,
		0, 0, 516, 517, 5, 5, 0, 0, 517, 518, 5, 11, 0, 0, 518, 519, 3, 52, 26,
		0, 519, 520, 5, 13, 0, 0, 520, 521, 3, 52, 26, 0, 521, 522, 5, 12, 0, 0,
		522, 57, 1, 0, 0, 0, 523, 524, 5, 11, 0, 0, 524, 529, 3, 52, 26, 0, 525,
		526, 5, 13, 0, 0, 526, 528, 3, 52, 26, 0, 527, 525, 1, 0, 0, 0, 528, 531,
		1, 0, 0, 0, 529, 527, 1, 0, 0, 0, 529, 530, 1, 0, 0, 0, 530, 532, 1, 0,
		0, 0, 531, 529, 1, 0, 0, 0, 532, 533, 5, 12, 0, 0, 533, 59, 1, 0, 0, 0,
		534, 535, 5, 14, 0, 0, 535, 538, 5, 14, 0, 0, 536, 538, 5, 14, 0, 0, 537,
		534, 1, 0, 0, 0, 537, 536, 1, 0, 0, 0, 538, 61, 1, 0, 0, 0, 539, 542, 5,
		24, 0, 0, 540, 541, 5, 15, 0, 0, 541, 543, 5, 24, 0, 0, 542, 540, 1, 0,
		0, 0, 542, 543, 1, 0, 0, 0, 543, 63, 1, 0, 0, 0, 544, 548, 5, 24, 0, 0,
		545, 547, 3, 66, 33, 0, 546, 545, 1, 0, 0, 0, 547, 550, 1, 0, 0, 0, 548,
		546, 1, 0, 0, 0, 548, 549, 1, 0, 0, 0, 549, 65, 1, 0, 0, 0, 550, 548, 1,
		0, 0, 0, 551, 559, 5, 19, 0, 0, 552, 559, 5, 20, 0, 0, 553, 559, 5, 22,
		0, 0, 554, 559, 5, 21, 0, 0, 555, 559, 5, 23, 0, 0, 556, 559, 3, 68, 34,
		0, 557, 559, 3, 62, 31, 0, 558, 551, 1, 0, 0, 0, 558, 552, 1, 0, 0, 0,
		558, 553, 1, 0, 0, 0, 558, 554, 1, 0, 0, 0, 558, 555, 1, 0, 0, 0, 558,
		556, 1, 0, 0, 0, 558, 557, 1, 0, 0, 0, 559, 67, 1, 0, 0, 0, 560, 564, 5,
		7, 0, 0, 561, 563, 3, 2, 1, 0, 562, 561, 1, 0, 0, 0, 563, 566, 1, 0, 0,
		0, 564, 562, 1, 0, 0, 0, 564, 565, 1, 0, 0, 0, 565, 581, 1, 0, 0, 0, 566,
		564, 1, 0, 0, 0, 567, 578, 3, 70, 35, 0, 568, 572, 5, 13, 0, 0, 569, 571,
		3, 2, 1, 0, 570, 569, 1, 0, 0, 0, 571, 574, 1, 0, 0, 0, 572, 570, 1, 0,
		0, 0, 572, 573, 1, 0, 0, 0, 573, 575, 1, 0, 0, 0, 574, 572, 1, 0, 0, 0,
		575, 577, 3, 70, 35, 0, 576, 568, 1, 0, 0, 0, 577, 580, 1, 0, 0, 0, 578,
		576, 1, 0, 0, 0, 578, 579, 1, 0, 0, 0, 579, 582, 1, 0, 0, 0, 580, 578,
		1, 0, 0, 0, 581, 567, 1, 0, 0, 0, 581, 582, 1, 0, 0, 0, 582, 584, 1, 0,
		0, 0, 583, 585, 5, 13, 0, 0, 584, 583, 1, 0, 0, 0, 584, 585, 1, 0, 0, 0,
		585, 589, 1, 0, 0, 0, 586, 588, 3, 2, 1, 0, 587, 586, 1, 0, 0, 0, 588,
		591, 1, 0, 0, 0, 589, 587, 1, 0, 0, 0, 589, 590, 1, 0, 0, 0, 590, 592,
		1, 0, 0, 0, 591, 589, 1, 0, 0, 0, 592, 593, 5, 8, 0, 0, 593, 69, 1, 0,
		0, 0, 594, 595, 5, 24, 0, 0, 595, 596, 3, 66, 33, 0, 596, 71, 1, 0, 0,
		0, 597, 598, 5, 24, 0, 0, 598, 601, 5, 2, 0, 0, 599, 600, 5, 4, 0, 0, 600,
		602, 3, 12, 6, 0, 601, 599, 1, 0, 0, 0, 601, 602, 1, 0, 0, 0, 602, 606,
		1, 0, 0, 0, 603, 605, 3, 2, 1, 0, 604, 603, 1, 0, 0, 0, 605, 608, 1, 0,
		0, 0, 606, 604, 1, 0, 0, 0, 606, 607, 1, 0, 0, 0, 607, 609, 1, 0, 0, 0,
		608, 606, 1, 0, 0, 0, 609, 613, 5, 7, 0, 0, 610, 612, 3, 2, 1, 0, 611,
		610, 1, 0, 0, 0, 612, 615, 1, 0, 0, 0, 613, 611, 1, 0, 0, 0, 613, 614,
		1, 0, 0, 0, 614, 616, 1, 0, 0, 0, 615, 613, 1, 0, 0, 0, 616, 617, 3, 74,
		37, 0, 617, 618, 5, 8, 0, 0, 618, 73, 1, 0, 0, 0, 619, 622, 3, 76, 38,
		0, 620, 622, 3, 46, 23, 0, 621, 619, 1, 0, 0, 0, 621, 620, 1, 0, 0, 0,
		622, 626, 1, 0, 0, 0, 623, 625, 3, 2, 1, 0, 624, 623, 1, 0, 0, 0, 625,
		628, 1, 0, 0, 0, 626, 624, 1, 0, 0, 0, 626, 627, 1, 0, 0, 0, 627, 630,
		1, 0, 0, 0, 628, 626, 1, 0, 0, 0, 629, 621, 1, 0, 0, 0, 630, 633, 1, 0,
		0, 0, 631, 629, 1, 0, 0, 0, 631, 632, 1, 0, 0, 0, 632, 75, 1, 0, 0, 0,
		633, 631, 1, 0, 0, 0, 634, 635, 5, 24, 0, 0, 635, 636, 5, 16, 0, 0, 636,
		638, 7, 0, 0, 0, 637, 639, 3, 78, 39, 0, 638, 637, 1, 0, 0, 0, 638, 639,
		1, 0, 0, 0, 639, 77, 1, 0, 0, 0, 640, 642, 3, 2, 1, 0, 641, 640, 1, 0,
		0, 0, 642, 645, 1, 0, 0, 0, 643, 641, 1, 0, 0, 0, 643, 644, 1, 0, 0, 0,
		644, 646, 1, 0, 0, 0, 645, 643, 1, 0, 0, 0, 646, 650, 5, 7, 0, 0, 647,
		649, 3, 2, 1, 0, 648, 647, 1, 0, 0, 0, 649, 652, 1, 0, 0, 0, 650, 648,
		1, 0, 0, 0, 650, 651, 1, 0, 0, 0, 651, 662, 1, 0, 0, 0, 652, 650, 1, 0,
		0, 0, 653, 657, 3, 46, 23, 0, 654, 656, 3, 2, 1, 0, 655, 654, 1, 0, 0,
		0, 656, 659, 1, 0, 0, 0, 657, 655, 1, 0, 0, 0, 657, 658, 1, 0, 0, 0, 658,
		661, 1, 0, 0, 0, 659, 657, 1, 0, 0, 0, 660, 653, 1, 0, 0, 0, 661, 664,
		1, 0, 0, 0, 662, 660, 1, 0, 0, 0, 662, 663, 1, 0, 0, 0, 663, 665, 1, 0,
		0, 0, 664, 662, 1, 0, 0, 0, 665, 666, 5, 8, 0, 0, 666, 79, 1, 0, 0, 0,
		667, 669, 5, 24, 0, 0, 668, 670, 3, 16, 8, 0, 669, 668, 1, 0, 0, 0, 669,
		670, 1, 0, 0, 0, 670, 671, 1, 0, 0, 0, 671, 673, 3, 52, 26, 0, 672, 674,
		3, 82, 41, 0, 673, 672, 1, 0, 0, 0, 673, 674, 1, 0, 0, 0, 674, 81, 1, 0,
		0, 0, 675, 677, 3, 2, 1, 0, 676, 675, 1, 0, 0, 0, 677, 680, 1, 0, 0, 0,
		678, 676, 1, 0, 0, 0, 678, 679, 1, 0, 0, 0, 679, 681, 1, 0, 0, 0, 680,
		678, 1, 0, 0, 0, 681, 685, 5, 7, 0, 0, 682, 684, 3, 2, 1, 0, 683, 682,
		1, 0, 0, 0, 684, 687, 1, 0, 0, 0, 685, 683, 1, 0, 0, 0, 685, 686, 1, 0,
		0, 0, 686, 697, 1, 0, 0, 0, 687, 685, 1, 0, 0, 0, 688, 692, 3, 46, 23,
		0, 689, 691, 3, 2, 1, 0, 690, 689, 1, 0, 0, 0, 691, 694, 1, 0, 0, 0, 692,
		690, 1, 0, 0, 0, 692, 693, 1, 0, 0, 0, 693, 696, 1, 0, 0, 0, 694, 692,
		1, 0, 0, 0, 695, 688, 1, 0, 0, 0, 696, 699, 1, 0, 0, 0, 697, 695, 1, 0,
		0, 0, 697, 698, 1, 0, 0, 0, 698, 700, 1, 0, 0, 0, 699, 697, 1, 0, 0, 0,
		700, 701, 5, 8, 0, 0, 701, 83, 1, 0, 0, 0, 102, 87, 94, 99, 106, 111, 118,
		123, 136, 141, 146, 150, 155, 162, 170, 175, 177, 184, 190, 196, 203, 210,
		215, 224, 232, 238, 244, 251, 255, 259, 265, 270, 275, 290, 297, 305, 310,
		315, 321, 325, 329, 331, 335, 340, 345, 351, 359, 365, 371, 374, 382, 390,
		396, 402, 405, 416, 421, 428, 433, 438, 443, 451, 455, 460, 467, 474, 480,
		483, 488, 495, 499, 502, 505, 507, 514, 529, 537, 542, 548, 558, 564, 572,
		578, 581, 584, 589, 601, 606, 613, 621, 626, 631, 638, 643, 650, 657, 662,
		669, 673, 678, 685, 692, 697,
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
	OracleParserRULE_schema             = 0
	OracleParserRULE_nl                 = 1
	OracleParserRULE_importStmt         = 2
	OracleParserRULE_fileDomain         = 3
	OracleParserRULE_definition         = 4
	OracleParserRULE_structDef          = 5
	OracleParserRULE_typeRefList        = 6
	OracleParserRULE_aliasBody          = 7
	OracleParserRULE_typeParams         = 8
	OracleParserRULE_typeParam          = 9
	OracleParserRULE_structBody         = 10
	OracleParserRULE_fieldOmit          = 11
	OracleParserRULE_domainOmit         = 12
	OracleParserRULE_actionDef          = 13
	OracleParserRULE_actionBody         = 14
	OracleParserRULE_fieldDef           = 15
	OracleParserRULE_fieldDefault       = 16
	OracleParserRULE_defaultValue       = 17
	OracleParserRULE_arrayDefault       = 18
	OracleParserRULE_structDefault      = 19
	OracleParserRULE_structFieldDefault = 20
	OracleParserRULE_inlineDomain       = 21
	OracleParserRULE_fieldBody          = 22
	OracleParserRULE_domain             = 23
	OracleParserRULE_domainContent      = 24
	OracleParserRULE_domainBlock        = 25
	OracleParserRULE_typeRef            = 26
	OracleParserRULE_arrayModifier      = 27
	OracleParserRULE_mapType            = 28
	OracleParserRULE_typeArgs           = 29
	OracleParserRULE_typeModifiers      = 30
	OracleParserRULE_qualifiedIdent     = 31
	OracleParserRULE_expression         = 32
	OracleParserRULE_expressionValue    = 33
	OracleParserRULE_objectLiteral      = 34
	OracleParserRULE_objectField        = 35
	OracleParserRULE_enumDef            = 36
	OracleParserRULE_enumBody           = 37
	OracleParserRULE_enumValue          = 38
	OracleParserRULE_enumValueBody      = 39
	OracleParserRULE_typeDefDef         = 40
	OracleParserRULE_typeDefBody        = 41
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
	p.SetState(87)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(84)
			p.Nl()
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

	for _la == OracleParserIMPORT {
		{
			p.SetState(90)
			p.ImportStmt()
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

	for _la == OracleParserAT {
		{
			p.SetState(102)
			p.FileDomain()
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
	p.SetState(123)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserIDENT {
		{
			p.SetState(114)
			p.Definition()
		}
		p.SetState(118)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(115)
				p.Nl()
			}

			p.SetState(120)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}

		p.SetState(125)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(126)
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
		p.SetState(128)
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
		p.SetState(130)
		p.Match(OracleParserIMPORT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(131)
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
		p.SetState(133)
		p.Match(OracleParserAT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(134)
		p.Match(OracleParserIDENT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(136)
	p.GetErrorHandler().Sync(p)

	if p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 7, p.GetParserRuleContext()) == 1 {
		{
			p.SetState(135)
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
	p.SetState(141)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 8, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(138)
			p.StructDef()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(139)
			p.EnumDef()
		}

	case 3:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(140)
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

	p.SetState(177)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 15, p.GetParserRuleContext()) {
	case 1:
		localctx = NewStructFullContext(p, localctx)
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(143)
			p.Match(OracleParserIDENT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(144)
			p.Match(OracleParserSTRUCT)
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

		if _la == OracleParserLT {
			{
				p.SetState(145)
				p.TypeParams()
			}

		}
		p.SetState(150)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		if _la == OracleParserEXTENDS {
			{
				p.SetState(148)
				p.Match(OracleParserEXTENDS)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}
			{
				p.SetState(149)
				p.TypeRefList()
			}

		}
		p.SetState(155)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(152)
				p.Nl()
			}

			p.SetState(157)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}
		{
			p.SetState(158)
			p.Match(OracleParserLBRACE)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		p.SetState(162)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(159)
				p.Nl()
			}

			p.SetState(164)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}
		{
			p.SetState(165)
			p.StructBody()
		}
		{
			p.SetState(166)
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
			p.SetState(168)
			p.Match(OracleParserIDENT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		p.SetState(170)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		if _la == OracleParserLT {
			{
				p.SetState(169)
				p.TypeParams()
			}

		}
		{
			p.SetState(172)
			p.Match(OracleParserEQUALS)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(173)
			p.TypeRef()
		}
		p.SetState(175)
		p.GetErrorHandler().Sync(p)

		if p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 14, p.GetParserRuleContext()) == 1 {
			{
				p.SetState(174)
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
		p.SetState(179)
		p.TypeRef()
	}
	p.SetState(190)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserCOMMA {
		{
			p.SetState(180)
			p.Match(OracleParserCOMMA)
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
			p.TypeRef()
		}

		p.SetState(192)
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
	p.SetState(196)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(193)
			p.Nl()
		}

		p.SetState(198)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(199)
		p.Match(OracleParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(203)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(200)
			p.Nl()
		}

		p.SetState(205)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	p.SetState(215)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserAT {
		{
			p.SetState(206)
			p.Domain()
		}
		p.SetState(210)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(207)
				p.Nl()
			}

			p.SetState(212)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}

		p.SetState(217)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(218)
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
		p.SetState(220)
		p.Match(OracleParserLT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(224)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(221)
			p.Nl()
		}

		p.SetState(226)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(227)
		p.TypeParam()
	}
	p.SetState(238)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserCOMMA {
		{
			p.SetState(228)
			p.Match(OracleParserCOMMA)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
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
			p.TypeParam()
		}

		p.SetState(240)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	p.SetState(244)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(241)
			p.Nl()
		}

		p.SetState(246)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(247)
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
		p.SetState(249)
		p.Match(OracleParserIDENT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(251)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == OracleParserQUESTION {
		{
			p.SetState(250)
			p.Match(OracleParserQUESTION)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	}
	p.SetState(255)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == OracleParserEXTENDS {
		{
			p.SetState(253)
			p.Match(OracleParserEXTENDS)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(254)
			p.TypeRef()
		}

	}
	p.SetState(259)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == OracleParserEQUALS {
		{
			p.SetState(257)
			p.Match(OracleParserEQUALS)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(258)
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
	p.SetState(275)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&17170496) != 0 {
		p.SetState(265)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}

		switch p.GetTokenStream().LA(1) {
		case OracleParserIDENT:
			{
				p.SetState(261)
				p.FieldDef()
			}

		case OracleParserMINUS:
			{
				p.SetState(262)
				p.FieldOmit()
			}

		case OracleParserACTION:
			{
				p.SetState(263)
				p.ActionDef()
			}

		case OracleParserAT:
			{
				p.SetState(264)
				p.Domain()
			}

		default:
			p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
			goto errorExit
		}
		p.SetState(270)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(267)
				p.Nl()
			}

			p.SetState(272)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}

		p.SetState(277)
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
		p.SetState(278)
		p.Match(OracleParserMINUS)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(279)
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
		p.SetState(281)
		p.Match(OracleParserMINUS)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(282)
		p.Match(OracleParserAT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(283)
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
		p.SetState(285)
		p.Match(OracleParserACTION)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(286)
		p.Match(OracleParserIDENT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(290)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(287)
			p.Nl()
		}

		p.SetState(292)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(293)
		p.Match(OracleParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(297)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(294)
			p.Nl()
		}

		p.SetState(299)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(300)
		p.ActionBody()
	}
	{
		p.SetState(301)
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
	p.SetState(315)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserAT || _la == OracleParserIDENT {
		p.SetState(305)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}

		switch p.GetTokenStream().LA(1) {
		case OracleParserIDENT:
			{
				p.SetState(303)
				p.FieldDef()
			}

		case OracleParserAT:
			{
				p.SetState(304)
				p.Domain()
			}

		default:
			p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
			goto errorExit
		}
		p.SetState(310)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(307)
				p.Nl()
			}

			p.SetState(312)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}

		p.SetState(317)
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
		p.SetState(318)
		p.Match(OracleParserIDENT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(321)
	p.GetErrorHandler().Sync(p)

	if p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 37, p.GetParserRuleContext()) == 1 {
		{
			p.SetState(319)
			p.TypeRef()
		}

	} else if p.HasError() { // JIM
		goto errorExit
	} else if p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 37, p.GetParserRuleContext()) == 2 {
		{
			p.SetState(320)
			p.TypeModifiers()
		}

	} else if p.HasError() { // JIM
		goto errorExit
	}
	p.SetState(325)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == OracleParserEQUALS {
		{
			p.SetState(323)
			p.Match(OracleParserEQUALS)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(324)
			p.FieldDefault()
		}

	}
	p.SetState(331)
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
			p.SetState(329)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}

			switch p.GetTokenStream().LA(1) {
			case OracleParserAT:
				{
					p.SetState(327)
					p.InlineDomain()
				}

			case OracleParserMINUS:
				{
					p.SetState(328)
					p.DomainOmit()
				}

			default:
				p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
				goto errorExit
			}

		}
		p.SetState(333)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 40, p.GetParserRuleContext())
		if p.HasError() {
			goto errorExit
		}
	}
	p.SetState(335)
	p.GetErrorHandler().Sync(p)

	if p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 41, p.GetParserRuleContext()) == 1 {
		{
			p.SetState(334)
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
	StructDefault() IStructDefaultContext

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

func (s *FieldDefaultContext) StructDefault() IStructDefaultContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IStructDefaultContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IStructDefaultContext)
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
	p.SetState(340)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 42, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(337)
			p.ExpressionValue()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(338)
			p.ArrayDefault()
		}

	case 3:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(339)
			p.StructDefault()
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

// IDefaultValueContext is an interface to support dynamic dispatch.
type IDefaultValueContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	ExpressionValue() IExpressionValueContext
	ArrayDefault() IArrayDefaultContext
	StructDefault() IStructDefaultContext

	// IsDefaultValueContext differentiates from other interfaces.
	IsDefaultValueContext()
}

type DefaultValueContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyDefaultValueContext() *DefaultValueContext {
	var p = new(DefaultValueContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_defaultValue
	return p
}

func InitEmptyDefaultValueContext(p *DefaultValueContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_defaultValue
}

func (*DefaultValueContext) IsDefaultValueContext() {}

func NewDefaultValueContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *DefaultValueContext {
	var p = new(DefaultValueContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = OracleParserRULE_defaultValue

	return p
}

func (s *DefaultValueContext) GetParser() antlr.Parser { return s.parser }

func (s *DefaultValueContext) ExpressionValue() IExpressionValueContext {
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

func (s *DefaultValueContext) ArrayDefault() IArrayDefaultContext {
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

func (s *DefaultValueContext) StructDefault() IStructDefaultContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IStructDefaultContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IStructDefaultContext)
}

func (s *DefaultValueContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *DefaultValueContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *DefaultValueContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.EnterDefaultValue(s)
	}
}

func (s *DefaultValueContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.ExitDefaultValue(s)
	}
}

func (p *OracleParser) DefaultValue() (localctx IDefaultValueContext) {
	localctx = NewDefaultValueContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 34, OracleParserRULE_defaultValue)
	p.SetState(345)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 43, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(342)
			p.ExpressionValue()
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(343)
			p.ArrayDefault()
		}

	case 3:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(344)
			p.StructDefault()
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
	AllDefaultValue() []IDefaultValueContext
	DefaultValue(i int) IDefaultValueContext
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

func (s *ArrayDefaultContext) AllDefaultValue() []IDefaultValueContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IDefaultValueContext); ok {
			len++
		}
	}

	tst := make([]IDefaultValueContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IDefaultValueContext); ok {
			tst[i] = t.(IDefaultValueContext)
			i++
		}
	}

	return tst
}

func (s *ArrayDefaultContext) DefaultValue(i int) IDefaultValueContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IDefaultValueContext); ok {
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

	return t.(IDefaultValueContext)
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
	p.EnterRule(localctx, 36, OracleParserRULE_arrayDefault)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(347)
		p.Match(OracleParserLBRACKET)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(351)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(348)
			p.Nl()
		}

		p.SetState(353)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	p.SetState(374)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if (int64(_la) & ^0x3f) == 0 && ((int64(1)<<_la)&33030784) != 0 {
		{
			p.SetState(354)
			p.DefaultValue()
		}
		p.SetState(365)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserCOMMA {
			{
				p.SetState(355)
				p.Match(OracleParserCOMMA)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}
			p.SetState(359)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)

			for _la == OracleParserNEWLINE {
				{
					p.SetState(356)
					p.Nl()
				}

				p.SetState(361)
				p.GetErrorHandler().Sync(p)
				if p.HasError() {
					goto errorExit
				}
				_la = p.GetTokenStream().LA(1)
			}
			{
				p.SetState(362)
				p.DefaultValue()
			}

			p.SetState(367)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}
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

	}
	{
		p.SetState(376)
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

// IStructDefaultContext is an interface to support dynamic dispatch.
type IStructDefaultContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	LBRACE() antlr.TerminalNode
	RBRACE() antlr.TerminalNode
	AllNl() []INlContext
	Nl(i int) INlContext
	AllStructFieldDefault() []IStructFieldDefaultContext
	StructFieldDefault(i int) IStructFieldDefaultContext
	AllCOMMA() []antlr.TerminalNode
	COMMA(i int) antlr.TerminalNode

	// IsStructDefaultContext differentiates from other interfaces.
	IsStructDefaultContext()
}

type StructDefaultContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyStructDefaultContext() *StructDefaultContext {
	var p = new(StructDefaultContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_structDefault
	return p
}

func InitEmptyStructDefaultContext(p *StructDefaultContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_structDefault
}

func (*StructDefaultContext) IsStructDefaultContext() {}

func NewStructDefaultContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *StructDefaultContext {
	var p = new(StructDefaultContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = OracleParserRULE_structDefault

	return p
}

func (s *StructDefaultContext) GetParser() antlr.Parser { return s.parser }

func (s *StructDefaultContext) LBRACE() antlr.TerminalNode {
	return s.GetToken(OracleParserLBRACE, 0)
}

func (s *StructDefaultContext) RBRACE() antlr.TerminalNode {
	return s.GetToken(OracleParserRBRACE, 0)
}

func (s *StructDefaultContext) AllNl() []INlContext {
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

func (s *StructDefaultContext) Nl(i int) INlContext {
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

func (s *StructDefaultContext) AllStructFieldDefault() []IStructFieldDefaultContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IStructFieldDefaultContext); ok {
			len++
		}
	}

	tst := make([]IStructFieldDefaultContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IStructFieldDefaultContext); ok {
			tst[i] = t.(IStructFieldDefaultContext)
			i++
		}
	}

	return tst
}

func (s *StructDefaultContext) StructFieldDefault(i int) IStructFieldDefaultContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IStructFieldDefaultContext); ok {
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

	return t.(IStructFieldDefaultContext)
}

func (s *StructDefaultContext) AllCOMMA() []antlr.TerminalNode {
	return s.GetTokens(OracleParserCOMMA)
}

func (s *StructDefaultContext) COMMA(i int) antlr.TerminalNode {
	return s.GetToken(OracleParserCOMMA, i)
}

func (s *StructDefaultContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *StructDefaultContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *StructDefaultContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.EnterStructDefault(s)
	}
}

func (s *StructDefaultContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.ExitStructDefault(s)
	}
}

func (p *OracleParser) StructDefault() (localctx IStructDefaultContext) {
	localctx = NewStructDefaultContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 38, OracleParserRULE_structDefault)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(378)
		p.Match(OracleParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(382)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(379)
			p.Nl()
		}

		p.SetState(384)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	p.SetState(405)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == OracleParserIDENT {
		{
			p.SetState(385)
			p.StructFieldDefault()
		}
		p.SetState(396)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserCOMMA {
			{
				p.SetState(386)
				p.Match(OracleParserCOMMA)
				if p.HasError() {
					// Recognition error - abort rule
					goto errorExit
				}
			}
			p.SetState(390)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)

			for _la == OracleParserNEWLINE {
				{
					p.SetState(387)
					p.Nl()
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
				p.StructFieldDefault()
			}

			p.SetState(398)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}
		p.SetState(402)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(399)
				p.Nl()
			}

			p.SetState(404)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}

	}
	{
		p.SetState(407)
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

// IStructFieldDefaultContext is an interface to support dynamic dispatch.
type IStructFieldDefaultContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	IDENT() antlr.TerminalNode
	EQUALS() antlr.TerminalNode
	DefaultValue() IDefaultValueContext

	// IsStructFieldDefaultContext differentiates from other interfaces.
	IsStructFieldDefaultContext()
}

type StructFieldDefaultContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyStructFieldDefaultContext() *StructFieldDefaultContext {
	var p = new(StructFieldDefaultContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_structFieldDefault
	return p
}

func InitEmptyStructFieldDefaultContext(p *StructFieldDefaultContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_structFieldDefault
}

func (*StructFieldDefaultContext) IsStructFieldDefaultContext() {}

func NewStructFieldDefaultContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *StructFieldDefaultContext {
	var p = new(StructFieldDefaultContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = OracleParserRULE_structFieldDefault

	return p
}

func (s *StructFieldDefaultContext) GetParser() antlr.Parser { return s.parser }

func (s *StructFieldDefaultContext) IDENT() antlr.TerminalNode {
	return s.GetToken(OracleParserIDENT, 0)
}

func (s *StructFieldDefaultContext) EQUALS() antlr.TerminalNode {
	return s.GetToken(OracleParserEQUALS, 0)
}

func (s *StructFieldDefaultContext) DefaultValue() IDefaultValueContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IDefaultValueContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IDefaultValueContext)
}

func (s *StructFieldDefaultContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *StructFieldDefaultContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *StructFieldDefaultContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.EnterStructFieldDefault(s)
	}
}

func (s *StructFieldDefaultContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.ExitStructFieldDefault(s)
	}
}

func (p *OracleParser) StructFieldDefault() (localctx IStructFieldDefaultContext) {
	localctx = NewStructFieldDefaultContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 40, OracleParserRULE_structFieldDefault)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(409)
		p.Match(OracleParserIDENT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(410)
		p.Match(OracleParserEQUALS)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(411)
		p.DefaultValue()
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
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
	p.EnterRule(localctx, 42, OracleParserRULE_inlineDomain)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(413)
		p.Match(OracleParserAT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(414)
		p.Match(OracleParserIDENT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(416)
	p.GetErrorHandler().Sync(p)

	if p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 54, p.GetParserRuleContext()) == 1 {
		{
			p.SetState(415)
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
	p.EnterRule(localctx, 44, OracleParserRULE_fieldBody)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(421)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(418)
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
		p.SetState(424)
		p.Match(OracleParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(428)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(425)
			p.Nl()
		}

		p.SetState(430)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	p.SetState(443)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserAT || _la == OracleParserMINUS {
		p.SetState(433)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}

		switch p.GetTokenStream().LA(1) {
		case OracleParserAT:
			{
				p.SetState(431)
				p.Domain()
			}

		case OracleParserMINUS:
			{
				p.SetState(432)
				p.DomainOmit()
			}

		default:
			p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
			goto errorExit
		}
		p.SetState(438)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(435)
				p.Nl()
			}

			p.SetState(440)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
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
	p.EnterRule(localctx, 46, OracleParserRULE_domain)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(448)
		p.Match(OracleParserAT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(449)
		p.Match(OracleParserIDENT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(451)
	p.GetErrorHandler().Sync(p)

	if p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 60, p.GetParserRuleContext()) == 1 {
		{
			p.SetState(450)
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
	p.EnterRule(localctx, 48, OracleParserRULE_domainContent)
	p.SetState(455)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case OracleParserLBRACE, OracleParserNEWLINE:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(453)
			p.DomainBlock()
		}

	case OracleParserIDENT:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(454)
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
	p.EnterRule(localctx, 50, OracleParserRULE_domainBlock)
	var _la int

	var _alt int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(460)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(457)
			p.Nl()
		}

		p.SetState(462)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(463)
		p.Match(OracleParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(467)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 63, p.GetParserRuleContext())
	if p.HasError() {
		goto errorExit
	}
	for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
		if _alt == 1 {
			{
				p.SetState(464)
				p.Nl()
			}

		}
		p.SetState(469)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 63, p.GetParserRuleContext())
		if p.HasError() {
			goto errorExit
		}
	}
	p.SetState(483)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == OracleParserIDENT {
		{
			p.SetState(470)
			p.Expression()
		}
		p.SetState(480)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 65, p.GetParserRuleContext())
		if p.HasError() {
			goto errorExit
		}
		for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
			if _alt == 1 {
				p.SetState(472)
				p.GetErrorHandler().Sync(p)
				if p.HasError() {
					goto errorExit
				}
				_la = p.GetTokenStream().LA(1)

				for ok := true; ok; ok = _la == OracleParserNEWLINE {
					{
						p.SetState(471)
						p.Nl()
					}

					p.SetState(474)
					p.GetErrorHandler().Sync(p)
					if p.HasError() {
						goto errorExit
					}
					_la = p.GetTokenStream().LA(1)
				}
				{
					p.SetState(476)
					p.Expression()
				}

			}
			p.SetState(482)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 65, p.GetParserRuleContext())
			if p.HasError() {
				goto errorExit
			}
		}

	}
	p.SetState(488)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(485)
			p.Nl()
		}

		p.SetState(490)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(491)
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
	p.EnterRule(localctx, 52, OracleParserRULE_typeRef)
	var _la int

	p.SetState(507)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case OracleParserMAP:
		localctx = NewTypeRefMapContext(p, localctx)
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(493)
			p.MapType()
		}
		p.SetState(495)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		if _la == OracleParserQUESTION {
			{
				p.SetState(494)
				p.TypeModifiers()
			}

		}

	case OracleParserIDENT:
		localctx = NewTypeRefNormalContext(p, localctx)
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(497)
			p.QualifiedIdent()
		}
		p.SetState(499)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		if _la == OracleParserLT {
			{
				p.SetState(498)
				p.TypeArgs()
			}

		}
		p.SetState(502)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		if _la == OracleParserLBRACKET {
			{
				p.SetState(501)
				p.ArrayModifier()
			}

		}
		p.SetState(505)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		if _la == OracleParserQUESTION {
			{
				p.SetState(504)
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
	p.EnterRule(localctx, 54, OracleParserRULE_arrayModifier)
	p.SetState(514)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 73, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(509)
			p.Match(OracleParserLBRACKET)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(510)
			p.Match(OracleParserRBRACKET)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(511)
			p.Match(OracleParserLBRACKET)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(512)
			p.Match(OracleParserINT_LIT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(513)
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
	p.EnterRule(localctx, 56, OracleParserRULE_mapType)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(516)
		p.Match(OracleParserMAP)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(517)
		p.Match(OracleParserLT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(518)
		p.TypeRef()
	}
	{
		p.SetState(519)
		p.Match(OracleParserCOMMA)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(520)
		p.TypeRef()
	}
	{
		p.SetState(521)
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
	p.EnterRule(localctx, 58, OracleParserRULE_typeArgs)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(523)
		p.Match(OracleParserLT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(524)
		p.TypeRef()
	}
	p.SetState(529)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserCOMMA {
		{
			p.SetState(525)
			p.Match(OracleParserCOMMA)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(526)
			p.TypeRef()
		}

		p.SetState(531)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(532)
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
	p.EnterRule(localctx, 60, OracleParserRULE_typeModifiers)
	p.SetState(537)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 75, p.GetParserRuleContext()) {
	case 1:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(534)
			p.Match(OracleParserQUESTION)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(535)
			p.Match(OracleParserQUESTION)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case 2:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(536)
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
	p.EnterRule(localctx, 62, OracleParserRULE_qualifiedIdent)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(539)
		p.Match(OracleParserIDENT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(542)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == OracleParserDOT {
		{
			p.SetState(540)
			p.Match(OracleParserDOT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(541)
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
	p.EnterRule(localctx, 64, OracleParserRULE_expression)
	var _alt int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(544)
		p.Match(OracleParserIDENT)
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
	_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 77, p.GetParserRuleContext())
	if p.HasError() {
		goto errorExit
	}
	for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
		if _alt == 1 {
			{
				p.SetState(545)
				p.ExpressionValue()
			}

		}
		p.SetState(550)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 77, p.GetParserRuleContext())
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
	ObjectLiteral() IObjectLiteralContext
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

func (s *ExpressionValueContext) ObjectLiteral() IObjectLiteralContext {
	var t antlr.RuleContext
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IObjectLiteralContext); ok {
			t = ctx.(antlr.RuleContext)
			break
		}
	}

	if t == nil {
		return nil
	}

	return t.(IObjectLiteralContext)
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
	p.EnterRule(localctx, 66, OracleParserRULE_expressionValue)
	p.SetState(558)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}

	switch p.GetTokenStream().LA(1) {
	case OracleParserTRIPLE_STRING_LIT:
		p.EnterOuterAlt(localctx, 1)
		{
			p.SetState(551)
			p.Match(OracleParserTRIPLE_STRING_LIT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case OracleParserSTRING_LIT:
		p.EnterOuterAlt(localctx, 2)
		{
			p.SetState(552)
			p.Match(OracleParserSTRING_LIT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case OracleParserINT_LIT:
		p.EnterOuterAlt(localctx, 3)
		{
			p.SetState(553)
			p.Match(OracleParserINT_LIT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case OracleParserFLOAT_LIT:
		p.EnterOuterAlt(localctx, 4)
		{
			p.SetState(554)
			p.Match(OracleParserFLOAT_LIT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case OracleParserBOOL_LIT:
		p.EnterOuterAlt(localctx, 5)
		{
			p.SetState(555)
			p.Match(OracleParserBOOL_LIT)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	case OracleParserLBRACE:
		p.EnterOuterAlt(localctx, 6)
		{
			p.SetState(556)
			p.ObjectLiteral()
		}

	case OracleParserIDENT:
		p.EnterOuterAlt(localctx, 7)
		{
			p.SetState(557)
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

// IObjectLiteralContext is an interface to support dynamic dispatch.
type IObjectLiteralContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	LBRACE() antlr.TerminalNode
	RBRACE() antlr.TerminalNode
	AllNl() []INlContext
	Nl(i int) INlContext
	AllObjectField() []IObjectFieldContext
	ObjectField(i int) IObjectFieldContext
	AllCOMMA() []antlr.TerminalNode
	COMMA(i int) antlr.TerminalNode

	// IsObjectLiteralContext differentiates from other interfaces.
	IsObjectLiteralContext()
}

type ObjectLiteralContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyObjectLiteralContext() *ObjectLiteralContext {
	var p = new(ObjectLiteralContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_objectLiteral
	return p
}

func InitEmptyObjectLiteralContext(p *ObjectLiteralContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_objectLiteral
}

func (*ObjectLiteralContext) IsObjectLiteralContext() {}

func NewObjectLiteralContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ObjectLiteralContext {
	var p = new(ObjectLiteralContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = OracleParserRULE_objectLiteral

	return p
}

func (s *ObjectLiteralContext) GetParser() antlr.Parser { return s.parser }

func (s *ObjectLiteralContext) LBRACE() antlr.TerminalNode {
	return s.GetToken(OracleParserLBRACE, 0)
}

func (s *ObjectLiteralContext) RBRACE() antlr.TerminalNode {
	return s.GetToken(OracleParserRBRACE, 0)
}

func (s *ObjectLiteralContext) AllNl() []INlContext {
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

func (s *ObjectLiteralContext) Nl(i int) INlContext {
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

func (s *ObjectLiteralContext) AllObjectField() []IObjectFieldContext {
	children := s.GetChildren()
	len := 0
	for _, ctx := range children {
		if _, ok := ctx.(IObjectFieldContext); ok {
			len++
		}
	}

	tst := make([]IObjectFieldContext, len)
	i := 0
	for _, ctx := range children {
		if t, ok := ctx.(IObjectFieldContext); ok {
			tst[i] = t.(IObjectFieldContext)
			i++
		}
	}

	return tst
}

func (s *ObjectLiteralContext) ObjectField(i int) IObjectFieldContext {
	var t antlr.RuleContext
	j := 0
	for _, ctx := range s.GetChildren() {
		if _, ok := ctx.(IObjectFieldContext); ok {
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

	return t.(IObjectFieldContext)
}

func (s *ObjectLiteralContext) AllCOMMA() []antlr.TerminalNode {
	return s.GetTokens(OracleParserCOMMA)
}

func (s *ObjectLiteralContext) COMMA(i int) antlr.TerminalNode {
	return s.GetToken(OracleParserCOMMA, i)
}

func (s *ObjectLiteralContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ObjectLiteralContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ObjectLiteralContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.EnterObjectLiteral(s)
	}
}

func (s *ObjectLiteralContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.ExitObjectLiteral(s)
	}
}

func (p *OracleParser) ObjectLiteral() (localctx IObjectLiteralContext) {
	localctx = NewObjectLiteralContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 68, OracleParserRULE_objectLiteral)
	var _la int

	var _alt int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(560)
		p.Match(OracleParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(564)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 79, p.GetParserRuleContext())
	if p.HasError() {
		goto errorExit
	}
	for _alt != 2 && _alt != antlr.ATNInvalidAltNumber {
		if _alt == 1 {
			{
				p.SetState(561)
				p.Nl()
			}

		}
		p.SetState(566)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 79, p.GetParserRuleContext())
		if p.HasError() {
			goto errorExit
		}
	}
	p.SetState(581)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == OracleParserIDENT {
		{
			p.SetState(567)
			p.ObjectField()
		}
		p.SetState(578)
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
					p.SetState(568)
					p.Match(OracleParserCOMMA)
					if p.HasError() {
						// Recognition error - abort rule
						goto errorExit
					}
				}
				p.SetState(572)
				p.GetErrorHandler().Sync(p)
				if p.HasError() {
					goto errorExit
				}
				_la = p.GetTokenStream().LA(1)

				for _la == OracleParserNEWLINE {
					{
						p.SetState(569)
						p.Nl()
					}

					p.SetState(574)
					p.GetErrorHandler().Sync(p)
					if p.HasError() {
						goto errorExit
					}
					_la = p.GetTokenStream().LA(1)
				}
				{
					p.SetState(575)
					p.ObjectField()
				}

			}
			p.SetState(580)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_alt = p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 81, p.GetParserRuleContext())
			if p.HasError() {
				goto errorExit
			}
		}

	}
	p.SetState(584)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == OracleParserCOMMA {
		{
			p.SetState(583)
			p.Match(OracleParserCOMMA)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}

	}
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

// IObjectFieldContext is an interface to support dynamic dispatch.
type IObjectFieldContext interface {
	antlr.ParserRuleContext

	// GetParser returns the parser.
	GetParser() antlr.Parser

	// Getter signatures
	IDENT() antlr.TerminalNode
	ExpressionValue() IExpressionValueContext

	// IsObjectFieldContext differentiates from other interfaces.
	IsObjectFieldContext()
}

type ObjectFieldContext struct {
	antlr.BaseParserRuleContext
	parser antlr.Parser
}

func NewEmptyObjectFieldContext() *ObjectFieldContext {
	var p = new(ObjectFieldContext)
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_objectField
	return p
}

func InitEmptyObjectFieldContext(p *ObjectFieldContext) {
	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, nil, -1)
	p.RuleIndex = OracleParserRULE_objectField
}

func (*ObjectFieldContext) IsObjectFieldContext() {}

func NewObjectFieldContext(parser antlr.Parser, parent antlr.ParserRuleContext, invokingState int) *ObjectFieldContext {
	var p = new(ObjectFieldContext)

	antlr.InitBaseParserRuleContext(&p.BaseParserRuleContext, parent, invokingState)

	p.parser = parser
	p.RuleIndex = OracleParserRULE_objectField

	return p
}

func (s *ObjectFieldContext) GetParser() antlr.Parser { return s.parser }

func (s *ObjectFieldContext) IDENT() antlr.TerminalNode {
	return s.GetToken(OracleParserIDENT, 0)
}

func (s *ObjectFieldContext) ExpressionValue() IExpressionValueContext {
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

func (s *ObjectFieldContext) GetRuleContext() antlr.RuleContext {
	return s
}

func (s *ObjectFieldContext) ToStringTree(ruleNames []string, recog antlr.Recognizer) string {
	return antlr.TreesStringTree(s, ruleNames, recog)
}

func (s *ObjectFieldContext) EnterRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.EnterObjectField(s)
	}
}

func (s *ObjectFieldContext) ExitRule(listener antlr.ParseTreeListener) {
	if listenerT, ok := listener.(OracleParserListener); ok {
		listenerT.ExitObjectField(s)
	}
}

func (p *OracleParser) ObjectField() (localctx IObjectFieldContext) {
	localctx = NewObjectFieldContext(p, p.GetParserRuleContext(), p.GetState())
	p.EnterRule(localctx, 70, OracleParserRULE_objectField)
	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(594)
		p.Match(OracleParserIDENT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(595)
		p.ExpressionValue()
	}

errorExit:
	if p.HasError() {
		v := p.GetError()
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
	p.EnterRule(localctx, 72, OracleParserRULE_enumDef)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(597)
		p.Match(OracleParserIDENT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(598)
		p.Match(OracleParserENUM)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(601)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == OracleParserEXTENDS {
		{
			p.SetState(599)
			p.Match(OracleParserEXTENDS)
			if p.HasError() {
				// Recognition error - abort rule
				goto errorExit
			}
		}
		{
			p.SetState(600)
			p.TypeRefList()
		}

	}
	p.SetState(606)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(603)
			p.Nl()
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
		p.Match(OracleParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(613)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(610)
			p.Nl()
		}

		p.SetState(615)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(616)
		p.EnumBody()
	}
	{
		p.SetState(617)
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
	p.EnterRule(localctx, 74, OracleParserRULE_enumBody)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(631)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserAT || _la == OracleParserIDENT {
		p.SetState(621)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}

		switch p.GetTokenStream().LA(1) {
		case OracleParserIDENT:
			{
				p.SetState(619)
				p.EnumValue()
			}

		case OracleParserAT:
			{
				p.SetState(620)
				p.Domain()
			}

		default:
			p.SetError(antlr.NewNoViableAltException(p, nil, nil, nil, nil, nil))
			goto errorExit
		}
		p.SetState(626)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(623)
				p.Nl()
			}

			p.SetState(628)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}

		p.SetState(633)
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
	p.EnterRule(localctx, 76, OracleParserRULE_enumValue)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(634)
		p.Match(OracleParserIDENT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(635)
		p.Match(OracleParserEQUALS)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	{
		p.SetState(636)
		_la = p.GetTokenStream().LA(1)

		if !(_la == OracleParserSTRING_LIT || _la == OracleParserINT_LIT) {
			p.GetErrorHandler().RecoverInline(p)
		} else {
			p.GetErrorHandler().ReportMatch(p)
			p.Consume()
		}
	}
	p.SetState(638)
	p.GetErrorHandler().Sync(p)

	if p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 91, p.GetParserRuleContext()) == 1 {
		{
			p.SetState(637)
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
	p.EnterRule(localctx, 78, OracleParserRULE_enumValueBody)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(643)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(640)
			p.Nl()
		}

		p.SetState(645)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(646)
		p.Match(OracleParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(650)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(647)
			p.Nl()
		}

		p.SetState(652)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	p.SetState(662)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserAT {
		{
			p.SetState(653)
			p.Domain()
		}
		p.SetState(657)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(654)
				p.Nl()
			}

			p.SetState(659)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}

		p.SetState(664)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(665)
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
	p.EnterRule(localctx, 80, OracleParserRULE_typeDefDef)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	{
		p.SetState(667)
		p.Match(OracleParserIDENT)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(669)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	if _la == OracleParserLT {
		{
			p.SetState(668)
			p.TypeParams()
		}

	}
	{
		p.SetState(671)
		p.TypeRef()
	}
	p.SetState(673)
	p.GetErrorHandler().Sync(p)

	if p.GetInterpreter().AdaptivePredict(p.BaseParser, p.GetTokenStream(), 97, p.GetParserRuleContext()) == 1 {
		{
			p.SetState(672)
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
	p.EnterRule(localctx, 82, OracleParserRULE_typeDefBody)
	var _la int

	p.EnterOuterAlt(localctx, 1)
	p.SetState(678)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(675)
			p.Nl()
		}

		p.SetState(680)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(681)
		p.Match(OracleParserLBRACE)
		if p.HasError() {
			// Recognition error - abort rule
			goto errorExit
		}
	}
	p.SetState(685)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserNEWLINE {
		{
			p.SetState(682)
			p.Nl()
		}

		p.SetState(687)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	p.SetState(697)
	p.GetErrorHandler().Sync(p)
	if p.HasError() {
		goto errorExit
	}
	_la = p.GetTokenStream().LA(1)

	for _la == OracleParserAT {
		{
			p.SetState(688)
			p.Domain()
		}
		p.SetState(692)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)

		for _la == OracleParserNEWLINE {
			{
				p.SetState(689)
				p.Nl()
			}

			p.SetState(694)
			p.GetErrorHandler().Sync(p)
			if p.HasError() {
				goto errorExit
			}
			_la = p.GetTokenStream().LA(1)
		}

		p.SetState(699)
		p.GetErrorHandler().Sync(p)
		if p.HasError() {
			goto errorExit
		}
		_la = p.GetTokenStream().LA(1)
	}
	{
		p.SetState(700)
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
