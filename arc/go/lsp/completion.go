// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package lsp

import (
	"context"
	"strings"

	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/types"
	"go.lsp.dev/protocol"
)

type completionCategory int

const (
	categoryType completionCategory = 1 << iota
	categoryTopLevelKeyword
	categoryFunctionKeyword
	categoryFunction
	categoryUnit
	categoryValue
	categorySequenceKeyword
	categoryStageKeyword
)

type completionInfo struct {
	Label        string
	Detail       string
	Doc          string
	Insert       string
	InsertFormat protocol.InsertTextFormat
	Kind         protocol.CompletionItemKind
	Category     completionCategory
}

var completions = []completionInfo{
	{
		Label:    parser.LiteralI8,
		Detail:   "Signed 8-bit integer",
		Doc:      "Range: -128 to 172",
		Kind:     protocol.CompletionItemKindClass,
		Category: categoryType,
	},
	{
		Label:    parser.LiteralU8,
		Detail:   "Unsigned 8-bit integer",
		Doc:      "Range: 0 to 255",
		Kind:     protocol.CompletionItemKindClass,
		Category: categoryType,
	},
	{
		Label:    parser.LiteralI16,
		Detail:   "Signed 16-bit integer",
		Doc:      "Range: -32768 to 32767",
		Kind:     protocol.CompletionItemKindClass,
		Category: categoryType,
	},
	{
		Label:    parser.LiteralU16,
		Detail:   "Unsigned 16-bit integer",
		Doc:      "Range: 0 to 65535",
		Kind:     protocol.CompletionItemKindClass,
		Category: categoryType,
	},
	{
		Label:    parser.LiteralI32,
		Detail:   "Signed 32-bit integer",
		Doc:      "Range: -2147483648 to 2147483647",
		Kind:     protocol.CompletionItemKindClass,
		Category: categoryType,
	},
	{
		Label:    parser.LiteralU32,
		Detail:   "Unsigned 32-bit integer",
		Doc:      "Range: 0 to 4294967295",
		Kind:     protocol.CompletionItemKindClass,
		Category: categoryType,
	},
	{
		Label:    parser.LiteralI64,
		Detail:   "Signed 64-bit integer",
		Doc:      "Range: -9223372036854775808 to 9223372036854775807",
		Kind:     protocol.CompletionItemKindClass,
		Category: categoryType,
	},
	{
		Label:    parser.LiteralU64,
		Detail:   "Unsigned 64-bit integer",
		Doc:      "Range: 0 to 18446744073709551615",
		Kind:     protocol.CompletionItemKindClass,
		Category: categoryType,
	},
	{
		Label:    parser.LiteralF32,
		Detail:   "32-bit float",
		Doc:      "Single precision floating point",
		Kind:     protocol.CompletionItemKindClass,
		Category: categoryType,
	},
	{
		Label:    parser.LiteralF64,
		Detail:   "64-bit float",
		Kind:     protocol.CompletionItemKindClass,
		Category: categoryType,
	},
	{
		Label:    "string",
		Detail:   "String type",
		Doc:      "Immutable UTF-8 string",
		Kind:     protocol.CompletionItemKindClass,
		Category: categoryType,
	},
	{
		Label:    "timestamp",
		Detail:   "Timestamp type",
		Kind:     protocol.CompletionItemKindClass,
		Category: categoryType,
	},
	{
		Label:    parser.LiteralSERIES,
		Detail:   "Series type",
		Doc:      "Homogeneous array of values",
		Kind:     protocol.CompletionItemKindClass,
		Category: categoryType,
	},
	{
		Label:    parser.LiteralCHAN,
		Detail:   "Channel type",
		Doc:      "Communication channel",
		Kind:     protocol.CompletionItemKindClass,
		Category: categoryType,
	},
	{
		Label:        "len",
		Detail:       "len(series) i64",
		Doc:          "Returns the length of a series",
		Insert:       "len($0)",
		Kind:         protocol.CompletionItemKindFunction,
		InsertFormat: protocol.InsertTextFormatSnippet,
		Category:     categoryFunction | categoryValue,
	},
	{
		Label:        "now",
		Detail:       "now() timestamp",
		Doc:          "Returns the current timestamp",
		Insert:       "now()",
		Kind:         protocol.CompletionItemKindFunction,
		InsertFormat: protocol.InsertTextFormatSnippet,
		Category:     categoryFunction | categoryValue,
	},
	{
		Label:    "ns",
		Detail:   "Nanoseconds",
		Doc:      "1/1000000000 seconds",
		Insert:   "ns",
		Kind:     protocol.CompletionItemKindUnit,
		Category: categoryUnit | categoryValue,
	},
	{
		Label:    "us",
		Detail:   "Microseconds",
		Doc:      "1/1000000 seconds",
		Insert:   "us",
		Kind:     protocol.CompletionItemKindUnit,
		Category: categoryUnit | categoryValue,
	},
	{
		Label:    "ms",
		Detail:   "Milliseconds",
		Doc:      "1/1000 seconds",
		Insert:   "ms",
		Kind:     protocol.CompletionItemKindUnit,
		Category: categoryUnit | categoryValue,
	},
	{
		Label:    "s",
		Detail:   "Seconds",
		Doc:      "1 second",
		Insert:   "s",
		Kind:     protocol.CompletionItemKindUnit,
		Category: categoryUnit | categoryValue,
	},
	{
		Label:    "m",
		Detail:   "Minutes",
		Kind:     protocol.CompletionItemKindUnit,
		Category: categoryUnit | categoryValue,
	},
	{
		Label:    "h",
		Detail:   "Hours",
		Doc:      "1 hour",
		Insert:   "h",
		Kind:     protocol.CompletionItemKindUnit,
		Category: categoryUnit | categoryValue,
	},
	{
		Label:    "hz",
		Detail:   "Hertz",
		Kind:     protocol.CompletionItemKindUnit,
		Category: categoryUnit | categoryValue,
	},
	{
		Label:    "khz",
		Detail:   "Kilohertz",
		Doc:      "1000 hertz",
		Insert:   "khz",
		Kind:     protocol.CompletionItemKindUnit,
		Category: categoryUnit | categoryValue,
	},
	{
		Label:    "mhz",
		Detail:   "Megahertz",
		Doc:      "1000000 hertz",
		Insert:   "mhz",
		Kind:     protocol.CompletionItemKindUnit,
		Category: categoryUnit | categoryValue,
	},
	{
		Label:    "ghz",
		Detail:   "Gigahertz",
		Doc:      "1000000 hertz",
		Insert:   "ghz",
		Kind:     protocol.CompletionItemKindUnit,
		Category: categoryUnit | categoryValue,
	},
	{
		Label:        parser.LiteralSEQUENCE,
		Detail:       "sequence declaration",
		Doc:          "Declares a sequence (state machine)",
		Insert:       "sequence ${1:name} {\n\tstage ${2:first} {\n\t\t$0\n\t}\n}",
		Kind:         protocol.CompletionItemKindKeyword,
		InsertFormat: protocol.InsertTextFormatSnippet,
		Category:     categoryTopLevelKeyword,
	},
	{
		Label:        parser.LiteralSTAGE,
		Detail:       "stage declaration",
		Doc:          "Declares a stage within a sequence",
		Insert:       "stage ${1:name} {\n\t$0\n}",
		Kind:         protocol.CompletionItemKindKeyword,
		InsertFormat: protocol.InsertTextFormatSnippet,
		Category:     categorySequenceKeyword,
	},
	{
		Label:        parser.LiteralNEXT,
		Detail:       "next statement",
		Doc:          "Transitions to a stage unconditionally",
		Insert:       "next ${1:stage}",
		Kind:         protocol.CompletionItemKindKeyword,
		InsertFormat: protocol.InsertTextFormatSnippet,
		Category:     categoryStageKeyword,
	},
	{
		Label:        parser.LiteralAUTHORITY,
		Detail:       "authority declaration",
		Doc:          "Sets control authority for output channels",
		Insert:       "authority ${1:255}",
		Kind:         protocol.CompletionItemKindKeyword,
		InsertFormat: protocol.InsertTextFormatSnippet,
		Category:     categoryKeyword,
	},
	{
		Label:        parser.LiteralFUNC,
		Detail:       "func declaration",
		Doc:          "Declares a function",
		Insert:       "func ${1:name}($2) $3 {\n\t$0\n}",
		Kind:         protocol.CompletionItemKindKeyword,
		InsertFormat: protocol.InsertTextFormatSnippet,
		Category:     categoryTopLevelKeyword,
	},
	{
		Label:        parser.LiteralIF,
		Detail:       "if statement",
		Doc:          "Conditional statement",
		Insert:       "if ${1:condition} {\n\t$0\n}",
		Kind:         protocol.CompletionItemKindKeyword,
		InsertFormat: protocol.InsertTextFormatSnippet,
		Category:     categoryFunctionKeyword,
	},
	{
		Label:        parser.LiteralELSE,
		Detail:       "else clause",
		Doc:          "Alternative branch",
		Insert:       "else {\n\t$0\n}",
		Kind:         protocol.CompletionItemKindKeyword,
		InsertFormat: protocol.InsertTextFormatSnippet,
		Category:     categoryFunctionKeyword,
	},
	{
		Label:        parser.LiteralELSE + " " + parser.LiteralIF,
		Detail:       "else-if clause",
		Doc:          "Alternative conditional branch",
		Insert:       "else if ${1:condition} {\n\t$0\n}",
		Kind:         protocol.CompletionItemKindKeyword,
		InsertFormat: protocol.InsertTextFormatSnippet,
		Category:     categoryFunctionKeyword,
	},
	{
		Label:        parser.LiteralRETURN,
		Detail:       "return statement",
		Doc:          "Returns a value",
		Insert:       "return $0",
		Kind:         protocol.CompletionItemKindKeyword,
		InsertFormat: protocol.InsertTextFormatSnippet,
		Category:     categoryFunctionKeyword,
	},
}

func (s *Server) Completion(
	ctx context.Context,
	params *protocol.CompletionParams,
) (*protocol.CompletionList, error) {
	doc, ok := s.getDocument(params.TextDocument.URI)
	if !ok {
		return nil, nil
	}

	displayContent := doc.displayContent()

	line, ok := getLine(displayContent, params.Position.Line)
	if !ok {
		return &protocol.CompletionList{}, nil
	}

	prefix := ""
	if int(params.Position.Character) <= len(line) {
		start := int(params.Position.Character)
		for start > 0 && isWordChar(line[start-1]) {
			start--
		}
		prefix = line[start:params.Position.Character]
	}

	items := s.getCompletionItems(ctx, doc, prefix, params.Position)

	return &protocol.CompletionList{
		IsIncomplete: false,
		Items:        items,
	}, nil
}

func (s *Server) getCompletionItems(
	ctx context.Context,
	doc *Document,
	prefix string,
	pos protocol.Position,
) []protocol.CompletionItem {
	completionCtx := DetectCompletionContext(doc.displayContent(), pos)

	if completionCtx == ContextComment {
		return []protocol.CompletionItem{}
	}

	if completionCtx == ContextAuthorityEntry {
		return s.getAuthorityEntryCompletions(ctx, doc, prefix, pos)
	}

	if completionCtx == ContextConfigParamName || completionCtx == ContextConfigParamValue {
		configInfo := extractConfigContext(doc.displayContent(), pos)
		if configInfo != nil {
			if completionCtx == ContextConfigParamName {
				return s.getConfigParamCompletions(ctx, doc, prefix, configInfo)
			}
			return s.getConfigValueCompletions(ctx, doc, prefix, configInfo)
		}
	}

	allowed := getAllowedCategories(completionCtx)
	tokens := tokenizeContent(doc.displayContent())
	tokensBeforeCursor := getTokensBeforeCursor(tokens, pos)
	nesting := detectNesting(tokensBeforeCursor)
	if doc.isBlock() && nesting == NestingTopLevel {
		nesting = NestingFunction
	}
	if nesting == NestingSequenceBody {
		allowed = categorySequenceKeyword
	} else if completionCtx == ContextStatementStart || completionCtx == ContextUnknown {
		switch nesting {
		case NestingTopLevel:
			allowed |= categoryTopLevelKeyword
		case NestingFunction:
			allowed |= categoryFunctionKeyword
		case NestingStageBody:
			allowed |= categoryStageKeyword
		}
	}
	items := make([]protocol.CompletionItem, 0, len(completions))

	for _, c := range completions {
		if (c.Category & allowed) == 0 {
			continue
		}
		if !strings.HasPrefix(c.Label, prefix) {
			continue
		}
		item := protocol.CompletionItem{
			Label:         c.Label,
			Kind:          c.Kind,
			Detail:        c.Detail,
			Documentation: c.Doc,
		}
		if c.Insert != "" {
			item.InsertText = c.Insert
			item.InsertTextFormat = c.InsertFormat
		}
		items = append(items, item)
	}

	if completionCtx != ContextTypeAnnotation && nesting != NestingSequenceBody && doc.IR.Symbols != nil {
		scopeAtCursor := doc.findScopeAtPosition(pos)
		if scopeAtCursor != nil {
			scopes, err := scopeAtCursor.Search(ctx, prefix)
			if err == nil {
				for _, scope := range scopes {
					items = append(items, symbolCompletionItem(scope.Name, scope.Type))
				}
			}
		} else if s.cfg.GlobalResolver != nil {
			symbols, err := s.cfg.GlobalResolver.Search(ctx, prefix)
			if err == nil {
				for _, sym := range symbols {
					items = append(items, symbolCompletionItem(sym.Name, sym.Type))
				}
			}
		}
	}
	return items
}

func getAllowedCategories(ctx CompletionContext) completionCategory {
	switch ctx {
	case ContextTypeAnnotation:
		return categoryType
	case ContextExpression:
		return categoryValue | categoryFunction | categoryUnit
	case ContextStatementStart:
		return categoryValue | categoryFunction
	default:
		return categoryFunction | categoryUnit | categoryValue
	}
}

func symbolCompletionItem(name string, t types.Type) protocol.CompletionItem {
	var (
		kind   protocol.CompletionItemKind
		detail string
	)
	if typeStr := t.String(); typeStr != "" {
		if strings.Contains(typeStr, "->") {
			kind = protocol.CompletionItemKindFunction
		} else {
			kind = protocol.CompletionItemKindVariable
		}
		detail = typeStr
	} else {
		kind = protocol.CompletionItemKindVariable
	}
	return protocol.CompletionItem{
		Label:  name,
		Kind:   kind,
		Detail: detail,
	}
}

func (s *Server) getAuthorityEntryCompletions(
	ctx context.Context,
	doc *Document,
	prefix string,
	pos protocol.Position,
) []protocol.CompletionItem {
	existing := extractAuthorityExistingChannels(doc.displayContent(), pos)
	existingSet := make(map[string]bool, len(existing))
	for _, name := range existing {
		existingSet[name] = true
	}
	var items []protocol.CompletionItem
	if s.cfg.GlobalResolver != nil {
		symbols, err := s.cfg.GlobalResolver.Search(ctx, prefix)
		if err == nil {
			for _, sym := range symbols {
				if sym.Type.Kind != types.KindChan || existingSet[sym.Name] {
					continue
				}
				items = append(items, protocol.CompletionItem{
					Label:  sym.Name,
					Kind:   protocol.CompletionItemKindVariable,
					Detail: sym.Type.String(),
				})
			}
		}
	}
	if doc.IR.Symbols != nil {
		scopes, err := doc.IR.Symbols.Search(ctx, prefix)
		if err == nil {
			for _, scope := range scopes {
				if scope.Type.Kind != types.KindChan || existingSet[scope.Name] {
					continue
				}
				items = append(items, protocol.CompletionItem{
					Label:  scope.Name,
					Kind:   protocol.CompletionItemKindVariable,
					Detail: scope.Type.String(),
				})
			}
		}
	}
	return items
}

func (s *Server) resolveFunctionType(
	ctx context.Context,
	doc *Document,
	name string,
) (types.Type, bool) {
	if doc.IR.Symbols != nil {
		sym, err := doc.IR.Symbols.Resolve(ctx, name)
		if err == nil && sym.Type.Kind == types.KindFunction {
			return sym.Type, true
		}
	}
	if s.cfg.GlobalResolver != nil {
		sym, err := s.cfg.GlobalResolver.Resolve(ctx, name)
		if err == nil && sym.Type.Kind == types.KindFunction {
			return sym.Type, true
		}
	}
	return types.Type{}, false
}

func (s *Server) collectSymbols(
	ctx context.Context,
	doc *Document,
	prefix string,
	filter func(types.Type) bool,
) []protocol.CompletionItem {
	seen := make(map[string]bool)
	var items []protocol.CompletionItem
	addItem := func(name string, t types.Type) {
		if seen[name] || !filter(t) {
			return
		}
		seen[name] = true
		items = append(items, protocol.CompletionItem{
			Label:  name,
			Kind:   protocol.CompletionItemKindVariable,
			Detail: t.String(),
		})
	}
	if s.cfg.GlobalResolver != nil {
		symbols, err := s.cfg.GlobalResolver.Search(ctx, prefix)
		if err == nil {
			for _, sym := range symbols {
				addItem(sym.Name, sym.Type)
			}
		}
	}
	if doc.IR.Symbols != nil {
		scopes, err := doc.IR.Symbols.Search(ctx, prefix)
		if err == nil {
			for _, scope := range scopes {
				addItem(scope.Name, scope.Type)
			}
		}
	}
	return items
}

func (s *Server) getConfigParamCompletions(
	ctx context.Context,
	doc *Document,
	prefix string,
	configInfo *configContextInfo,
) []protocol.CompletionItem {
	fnType, ok := s.resolveFunctionType(ctx, doc, configInfo.functionName)
	if !ok {
		return []protocol.CompletionItem{}
	}
	existingSet := make(map[string]bool)
	for _, param := range configInfo.existingParams {
		existingSet[param] = true
	}
	var items []protocol.CompletionItem
	for _, param := range fnType.Config {
		if existingSet[param.Name] || !strings.HasPrefix(param.Name, prefix) {
			continue
		}
		items = append(items, protocol.CompletionItem{
			Label:            param.Name,
			Kind:             protocol.CompletionItemKindProperty,
			Detail:           param.Type.String(),
			InsertText:       param.Name + "=",
			InsertTextFormat: protocol.InsertTextFormatPlainText,
		})
	}
	return items
}

func (s *Server) getConfigValueCompletions(
	ctx context.Context,
	doc *Document,
	prefix string,
	configInfo *configContextInfo,
) []protocol.CompletionItem {
	fnType, ok := s.resolveFunctionType(ctx, doc, configInfo.functionName)
	if !ok {
		return []protocol.CompletionItem{}
	}
	param, found := fnType.Config.Get(configInfo.currentParamName)
	if !found {
		return []protocol.CompletionItem{}
	}
	var items []protocol.CompletionItem
	if param.Type.Kind == types.KindChan {
		items = s.collectSymbols(ctx, doc, prefix, func(t types.Type) bool {
			return t.Kind == types.KindChan
		})
	} else if param.Type.IsNumeric() {
		for _, c := range completions {
			if (c.Category&categoryUnit) == 0 || !strings.HasPrefix(c.Label, prefix) {
				continue
			}
			items = append(items, protocol.CompletionItem{
				Label:  c.Label,
				Kind:   c.Kind,
				Detail: c.Detail,
			})
		}
	}
	nonChanSymbols := s.collectSymbols(ctx, doc, prefix, func(t types.Type) bool {
		return t.Kind != types.KindChan
	})
	items = append(items, nonChanSymbols...)
	return items
}
