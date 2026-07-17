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
	"sort"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	lsp "github.com/synnaxlabs/x/lsp"
	"github.com/synnaxlabs/x/set"
	"go.lsp.dev/protocol"
	"go.uber.org/zap"
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
		Doc:      "Range: -128 to 127",
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
		Label:        "time.now",
		Detail:       "time.now() timestamp",
		Doc:          "Returns the current timestamp",
		Insert:       "time.now()",
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
		Label:    "min",
		Detail:   "Minutes",
		Doc:      "60 seconds",
		Insert:   "min",
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
		Label:    "d",
		Detail:   "Days",
		Doc:      "24 hours",
		Insert:   "d",
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
		Insert:       "sequence ${1:name} {\n\t$0\n}",
		Kind:         protocol.CompletionItemKindKeyword,
		InsertFormat: protocol.InsertTextFormatSnippet,
		Category:     categoryTopLevelKeyword | categorySequenceKeyword | categoryStageKeyword,
	},
	{
		Label:        parser.LiteralSTAGE,
		Detail:       "stage declaration",
		Doc:          "Declares a stage within a sequence",
		Insert:       "stage ${1:name} {\n\t$0\n}",
		Kind:         protocol.CompletionItemKindKeyword,
		InsertFormat: protocol.InsertTextFormatSnippet,
		Category:     categorySequenceKeyword | categoryTopLevelKeyword,
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
		Category:     categoryTopLevelKeyword,
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
		Label:        parser.LiteralIMPORT,
		Detail:       "import declaration",
		Doc:          "Imports modules so their qualified members can be used",
		Insert:       "import ( ${1:module} )",
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
	{
		Label:        parser.LiteralFOR,
		Detail:       "for loop",
		Doc:          "Loop statement",
		Insert:       "for ${1:i} := range(${2:n}) {\n\t$0\n}",
		Kind:         protocol.CompletionItemKindKeyword,
		InsertFormat: protocol.InsertTextFormatSnippet,
		Category:     categoryFunctionKeyword,
	},
	{
		Label:        parser.LiteralBREAK,
		Detail:       "break statement",
		Doc:          "Exits the enclosing for loop",
		Insert:       "break",
		Kind:         protocol.CompletionItemKindKeyword,
		InsertFormat: protocol.InsertTextFormatSnippet,
		Category:     categoryFunctionKeyword,
	},
	{
		Label:        parser.LiteralCONTINUE,
		Detail:       "continue statement",
		Doc:          "Skips to the next iteration of the enclosing for loop",
		Insert:       "continue",
		Kind:         protocol.CompletionItemKindKeyword,
		InsertFormat: protocol.InsertTextFormatSnippet,
		Category:     categoryFunctionKeyword,
	},
	{
		Label:        "range",
		Detail:       "range(end) or range(start, end) or range(start, end, step)",
		Doc:          "Generates a numeric range for iteration",
		Insert:       "range($0)",
		Kind:         protocol.CompletionItemKindFunction,
		InsertFormat: protocol.InsertTextFormatSnippet,
		Category:     categoryFunction | categoryValue,
	},
}

// optStr wraps a string in an Optional, leaving it absent when empty so
// empty fields stay off the wire like the old omitempty behavior.
func optStr(s string) protocol.Optional[string] {
	if s == "" {
		return protocol.Optional[string]{}
	}
	return protocol.NewOptional(s)
}

func (s *Server) Completion(
	ctx context.Context,
	params *protocol.CompletionParams,
) (protocol.CompletionResult, error) {
	doc, ok := s.getDocument(params.TextDocument.URI)
	if !ok {
		return nil, nil
	}

	displayContent := doc.displayContent()

	line, ok := lsp.GetLine(displayContent, params.Position.Line)
	if !ok {
		return &protocol.CompletionList{}, nil
	}

	prefix := ""
	if int(params.Position.Character) <= len(line) {
		start := int(params.Position.Character)
		for start > 0 && (lsp.IsWordChar(line[start-1]) || line[start-1] == '.') {
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

	if completionCtx == ContextComment || completionCtx == ContextNone {
		return []protocol.CompletionItem{}
	}

	var root *symbol.Symbol
	if s.cfg.NewRoot != nil {
		root = s.cfg.NewRoot()
	}

	if completionCtx == ContextAuthorityEntry {
		return s.getAuthorityEntryCompletions(ctx, doc, prefix, pos, root)
	}

	if completionCtx == ContextImportPath {
		return getImportPathCompletions(doc, prefix, pos, root)
	}

	if completionCtx == ContextInputParamName || completionCtx == ContextInputParamValue {
		inputInfo := extractInputContext(doc.displayContent(), pos)
		if inputInfo != nil {
			if completionCtx == ContextInputParamName {
				return s.getInputParamCompletions(ctx, doc, prefix, inputInfo, root)
			}
			return s.getInputValueCompletions(ctx, doc, prefix, inputInfo, root)
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
		// A sequence body accepts `stage` / nested `sequence` declarations
		// plus flow statements and single invocations (per the grammar's
		// `sequenceItem`). That allows channel references, flow / exec-both
		// functions, constants, and module-qualified members alongside the
		// stage keyword. ExecWASM-only functions are filtered out by the
		// execFilter below.
		allowed = categorySequenceKeyword | categoryValue | categoryFunction
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
			Label:  c.Label,
			Kind:   c.Kind,
			Detail: optStr(c.Detail),
		}
		if c.Doc != "" {
			item.Documentation = protocol.String(c.Doc)
		}
		if c.Insert != "" {
			item.InsertText = protocol.NewOptional(c.Insert)
			item.InsertTextFormat = c.InsertFormat
		}
		items = append(items, item)
	}

	if completionCtx != ContextTypeAnnotation {
		var execFilter symbol.ExecContext
		switch nesting {
		case NestingFunction:
			execFilter = symbol.ExecWASM
		case NestingTopLevel, NestingStageBody, NestingSequenceBody:
			execFilter = symbol.ExecFlow
		}
		modulePrefix := ""
		if dotIdx := strings.LastIndex(prefix, "."); dotIdx >= 0 {
			modulePrefix = prefix[:dotIdx+1]
		}
		startChar := pos.Character - uint32(len(prefix))
		if modulePrefix != "" {
			// Module-qualified prefix (e.g. "math." or "time.n"): find the
			// module by name, prefer the document's scope tree (so user
			// imports are respected); fall back to the STL ambient prelude
			// so completion still works when the document hasn't yet been
			// analyzed (e.g. mid-typing "math.").
			moduleName := strings.TrimSuffix(modulePrefix, ".")
			memberPrefix := prefix[len(modulePrefix):]
			var mod *symbol.Symbol
			if scopeAtCursor := doc.findScopeAtPosition(pos); scopeAtCursor != nil {
				mod, _ = scopeAtCursor.Resolve(ctx, moduleName, symbol.IncludeInternal, symbol.WithoutUsageTracking)
			}
			if mod == nil && root != nil {
				mod, _ = root.Resolve(ctx, moduleName, symbol.IncludeInternal, symbol.WithoutUsageTracking)
			}
			// Resolve returns the alias for `import time as t`; follow Target
			// to reach the module body for member enumeration. Aliased
			// resolution also tells us the module is already imported, so
			// the member completions don't need an auto-import edit.
			moduleImported := false
			if mod != nil && mod.Kind == symbol.KindModuleAlias && mod.Target != nil {
				moduleImported = true
				mod = mod.Target
			}
			if mod != nil && mod.Kind == symbol.KindModule {
				var importEdit []protocol.TextEdit
				if !moduleImported {
					importEdit = buildAutoImportEdit(doc, mod.Name)
				}
				for _, sym := range mod.Children() {
					if sym.Kind != symbol.KindFunction || sym.Internal {
						continue
					}
					if !sym.Exec.Compatible(execFilter) {
						continue
					}
					if memberPrefix != "" && !strings.HasPrefix(sym.Name, memberPrefix) {
						continue
					}
					qualifiedName := modulePrefix + sym.Name
					item := symbolCompletionItem(sym)
					item.FilterText = protocol.NewOptional(qualifiedName)
					item.TextEdit = &protocol.TextEdit{
						Range: protocol.Range{
							Start: protocol.Position{Line: pos.Line, Character: startChar},
							End:   pos,
						},
						NewText: qualifiedName,
					}
					item.AdditionalTextEdits = importEdit
					applyInvocationSuffix(&item, sym.Kind, execFilter)
					items = append(items, item)
				}
			}
		} else {
			searchScope := doc.findScopeAtPosition(pos)
			if searchScope == nil {
				searchScope = root
			}
			if searchScope != nil {
				symbols, err := searchScope.Search(ctx, prefix)
				if err == nil {
					for _, sym := range symbols {
						if sym.Kind == symbol.KindFunction && !sym.Exec.Compatible(execFilter) {
							continue
						}
						item := symbolCompletionItem(sym)
						if sym.Kind == symbol.KindModule {
							item.AdditionalTextEdits = buildAutoImportEdit(doc, sym.Name)
						}
						applyInvocationSuffix(&item, sym.Kind, execFilter)
						items = append(items, item)
					}
				}
				items = appendModuleMemberCompletions(
					items, doc, searchScope, prefix, pos, startChar, execFilter,
				)
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

func symbolCompletionItem(sym *symbol.Symbol) protocol.CompletionItem {
	// Modules are namespace containers, not values, so they have no
	// types.Type. Rendering them via Type.String() would surface the
	// "invalid" fallback as the completion's detail, which is wrong for
	// the user. The Module completion kind also gives the editor an
	// accurate icon. The auto-import edit for a bare KindModule is
	// attached by the caller via buildAutoImportEdit, which needs the
	// document context this function does not have.
	if sym.Kind == symbol.KindModule || sym.Kind == symbol.KindModuleAlias {
		return protocol.CompletionItem{
			Label:  sym.Name,
			Kind:   protocol.CompletionItemKindModule,
			Detail: protocol.NewOptional("module"),
		}
	}
	var (
		kind   protocol.CompletionItemKind
		detail string
	)
	if typeStr := sym.Type.String(); typeStr != "" {
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
		Label:  sym.Name,
		Kind:   kind,
		Detail: optStr(detail),
	}
}

// getImportPathCompletions returns the modules reachable from doc's
// scope chain, surfaced as Module completion items. Used when the cursor
// sits in the module-name slot of an `import` statement, where only
// module names are valid suggestions.
func getImportPathCompletions(
	doc *Document,
	prefix string,
	pos protocol.Position,
	root *symbol.Symbol,
) []protocol.CompletionItem {
	scope := doc.findScopeAtPosition(pos)
	if scope == nil {
		scope = root
	}
	if scope == nil {
		return []protocol.CompletionItem{}
	}
	var items []protocol.CompletionItem
	seen := make(set.Set[string])
	for s := scope; s != nil; s = s.Parent {
		for _, child := range s.Children() {
			if child.Kind != symbol.KindModule || child.Internal {
				continue
			}
			if !strings.HasPrefix(child.Name, prefix) {
				continue
			}
			if seen.Contains(child.Name) {
				continue
			}
			seen.Add(child.Name)
			items = append(items, protocol.CompletionItem{
				Label:  child.Name,
				Kind:   protocol.CompletionItemKindModule,
				Detail: protocol.NewOptional("module"),
			})
		}
	}
	return items
}

// isModuleImportedInSource reports whether the document text contains a
// top-level import of moduleName (loose or block form). Matching is on the
// importPath, not the AS alias — `import (math as t)` matches "math" but
// not "t". This is the textual fallback used when the analyzer's symbol
// tree is unavailable because the document has a parse error. The lexer
// succeeds even when the parser fails, so this scan stays accurate while
// the user is mid-typing a dot completion.
func isModuleImportedInSource(content, moduleName string) bool {
	// tokenizeContent keeps whitespace and comment tokens, so filter them
	// down to the parser's view before scanning for import statements.
	raw := tokenizeContent(content)
	tokens := make([]antlr.Token, 0, len(raw))
	for _, t := range raw {
		tt := t.GetTokenType()
		if tt == antlr.TokenEOF ||
			tt == parser.ArcLexerWS ||
			tt == parser.ArcLexerSINGLE_LINE_COMMENT ||
			tt == parser.ArcLexerMULTI_LINE_COMMENT {
			continue
		}
		tokens = append(tokens, t)
	}
	for i := 0; i < len(tokens); i++ {
		if tokens[i].GetTokenType() != parser.ArcLexerIMPORT {
			continue
		}
		i++
		if i >= len(tokens) {
			break
		}
		if tokens[i].GetTokenType() != parser.ArcLexerLPAREN {
			if path, _ := readImportPath(tokens, i); path == moduleName {
				return true
			}
			continue
		}
		i++
		for i < len(tokens) && tokens[i].GetTokenType() != parser.ArcLexerRPAREN {
			path, next := readImportPath(tokens, i)
			if next == i {
				i++
				continue
			}
			if path == moduleName {
				return true
			}
			i = next
			if i < len(tokens) && tokens[i].GetTokenType() == parser.ArcLexerAS {
				i++
				if i < len(tokens) && tokens[i].GetTokenType() == parser.ArcLexerIDENTIFIER {
					i++
				}
			}
		}
	}
	return false
}

// readImportPath reads tokens at start as an importPath
// (importPathHead (DOT IDENTIFIER)*) and returns its dotted text along with
// the index of the first token past the path. Returns ("", start) when
// tokens[start] is not a valid importPath head.
func readImportPath(tokens []antlr.Token, start int) (string, int) {
	if start >= len(tokens) {
		return "", start
	}
	head := tokens[start].GetTokenType()
	if head != parser.ArcLexerIDENTIFIER && head != parser.ArcLexerAUTHORITY {
		return "", start
	}
	var b strings.Builder
	b.WriteString(tokens[start].GetText())
	i := start + 1
	for i+1 < len(tokens) &&
		tokens[i].GetTokenType() == parser.ArcLexerDOT &&
		tokens[i+1].GetTokenType() == parser.ArcLexerIDENTIFIER {
		b.WriteByte('.')
		b.WriteString(tokens[i+1].GetText())
		i += 2
	}
	return b.String(), i
}

// applyInvocationSuffix appends an invocation snippet — `($0)` in an
// imperative/WASM context, `{$0}` in a flow context — to a function
// completion item so the cursor lands inside ready to receive arguments
// or inputs. No-op for non-function kinds, which insert their bare name.
// The suffix is appended to TextEdit.NewText when set, otherwise to
// InsertText (falling back to Label when InsertText is empty).
func applyInvocationSuffix(item *protocol.CompletionItem, kind symbol.Kind, execFilter symbol.ExecContext) {
	if kind != symbol.KindFunction {
		return
	}
	suffix := "{$0}"
	if execFilter == symbol.ExecWASM {
		suffix = "($0)"
	}
	if edit, ok := item.TextEdit.(*protocol.TextEdit); ok {
		edit.NewText += suffix
	} else {
		base, _ := item.InsertText.Get()
		if base == "" {
			base = item.Label
		}
		item.InsertText = protocol.NewOptional(base + suffix)
	}
	item.InsertTextFormat = protocol.InsertTextFormatSnippet
}

// formatImportBlock renders module names as a multi-line `import (...)`
// block, one name per line, indented four spaces. The trailing newline
// closes the `)` line; callers append an additional `\n` when the edit
// is inserted at the top of a file and needs a blank-line separator from
// the following content.
func formatImportBlock(names []string) string {
	var b strings.Builder
	b.WriteString("import (\n")
	for _, n := range names {
		b.WriteString("    ")
		b.WriteString(n)
		b.WriteByte('\n')
	}
	b.WriteString(")\n")
	return b.String()
}

// buildAutoImportEdit returns the AdditionalTextEdits that import
// moduleName into doc. See buildAutoImportEditFromSnapshot for the edit
// semantics; this wrapper exists for callers that still hold a *Document
// rather than snapshotted fields.
func buildAutoImportEdit(doc *Document, moduleName string) []protocol.TextEdit {
	return buildAutoImportEditFromSnapshot(doc.displayContent(), doc.IR.Symbols, moduleName)
}

// buildAutoImportEditFromSnapshot is the snapshot-friendly form of
// buildAutoImportEdit. content is the document text and symbols is the
// document's root symbol scope, both captured by the caller under the
// server lock so the edit can be computed without touching live document
// state. When no imports exist a single loose `import <name>` statement
// is inserted at the top of the file followed by a blank line; when
// imports do exist they are consolidated with moduleName into a single
// canonical `import (...)` block. The block form is reserved for two or
// more imports so files with a single import keep the lighter loose
// form. Returns nil when moduleName is already imported.
func buildAutoImportEditFromSnapshot(
	content string,
	symbols *symbol.Symbol,
	moduleName string,
) []protocol.TextEdit {
	// The analyzer's symbol tree is dropped whenever the document has a
	// parse error — which happens routinely while the user is mid-typing
	// a `.` member access right after auto-importing. Without this
	// token-level fallback the dot completion would re-add the import
	// the user just inserted. A textual scan answers "is moduleName
	// already imported?" without needing a valid AST.
	if isModuleImportedInSource(content, moduleName) {
		return nil
	}
	var (
		existingNames []string
		stmts         []antlr.ParserRuleContext
		seenStmts     = set.New[antlr.ParserRuleContext]()
	)
	if symbols != nil {
		for _, child := range symbols.Children() {
			if child.Kind != symbol.KindModuleAlias || child.AST == nil {
				continue
			}
			if child.Name == moduleName {
				return nil
			}
			existingNames = append(existingNames, child.Name)
			parent, ok := child.AST.GetParent().(antlr.ParserRuleContext)
			if !ok || seenStmts.Contains(parent) {
				continue
			}
			seenStmts.Add(parent)
			stmts = append(stmts, parent)
		}
	}
	if len(stmts) == 0 {
		return []protocol.TextEdit{{
			Range: protocol.Range{
				Start: protocol.Position{Line: 0, Character: 0},
				End:   protocol.Position{Line: 0, Character: 0},
			},
			NewText: "import " + moduleName + "\n\n",
		}}
	}
	allNames := append(existingNames, moduleName)
	sort.Slice(stmts, func(i, j int) bool {
		return tokenBefore(stmts[i].GetStart(), stmts[j].GetStart())
	})
	// Replace the first statement in document order with the consolidated
	// block and delete each subsequent statement. Using one big
	// minStart-to-maxStop replacement instead would clobber any comments or
	// in-progress edits sitting between the import statements; the
	// statement-scoped edits leave that interstitial content untouched.
	first := stmts[0]
	edits := []protocol.TextEdit{{
		Range: protocol.Range{
			Start: protocol.Position{
				Line:      uint32(first.GetStart().GetLine() - 1),
				Character: uint32(first.GetStart().GetColumn()),
			},
			End: protocol.Position{
				Line:      uint32(first.GetStop().GetLine() - 1),
				Character: uint32(first.GetStop().GetColumn() + len(first.GetStop().GetText())),
			},
		},
		NewText: formatImportBlock(allNames),
	}}
	for _, st := range stmts[1:] {
		edits = append(edits, deleteStatementEdit(content, st))
	}
	return edits
}

// appendModuleMemberCompletions adds completion items for module
// members reachable from scope (walking via Parent), surfaced under
// their qualified name when the user has typed a bare prefix. Each item
// replaces the typed bare prefix with `module.member` and attaches an
// auto-import edit when the module is not already in scope. Items are
// deduped by qualified name across the scope chain so a module that
// appears both as a file-root alias and as an ambient KindModule is
// only emitted once (the imported alias is encountered first and wins).
func appendModuleMemberCompletions(
	items []protocol.CompletionItem,
	doc *Document,
	scope *symbol.Symbol,
	prefix string,
	pos protocol.Position,
	startChar uint32,
	execFilter symbol.ExecContext,
) []protocol.CompletionItem {
	seen := make(set.Set[string])
	for s := scope; s != nil; s = s.Parent {
		for _, child := range s.Children() {
			var (
				mod      *symbol.Symbol
				imported bool
			)
			switch {
			case child.Kind == symbol.KindModule && !child.Internal:
				mod = child
			case child.Kind == symbol.KindModuleAlias && child.Target != nil && !child.Target.Internal:
				mod = child.Target
				imported = true
			default:
				continue
			}
			for _, member := range mod.Children() {
				if member.Internal || member.Name == "" {
					continue
				}
				if member.Kind == symbol.KindFunction && !member.Exec.Compatible(execFilter) {
					continue
				}
				if !strings.HasPrefix(member.Name, prefix) {
					continue
				}
				qualifiedName := mod.Name + "." + member.Name
				if seen.Contains(qualifiedName) {
					continue
				}
				seen.Add(qualifiedName)
				item := symbolCompletionItem(member)
				item.Label = qualifiedName
				item.FilterText = protocol.NewOptional(qualifiedName)
				item.TextEdit = &protocol.TextEdit{
					Range: protocol.Range{
						Start: protocol.Position{Line: pos.Line, Character: startChar},
						End:   pos,
					},
					NewText: qualifiedName,
				}
				if !imported {
					item.AdditionalTextEdits = buildAutoImportEdit(doc, mod.Name)
				}
				applyInvocationSuffix(&item, member.Kind, execFilter)
				items = append(items, item)
			}
		}
	}
	return items
}

func tokenBefore(a, b antlr.Token) bool {
	if a.GetLine() != b.GetLine() {
		return a.GetLine() < b.GetLine()
	}
	return a.GetColumn() < b.GetColumn()
}

func (s *Server) getAuthorityEntryCompletions(
	ctx context.Context,
	doc *Document,
	prefix string,
	pos protocol.Position,
	root *symbol.Symbol,
) []protocol.CompletionItem {
	existing := extractAuthorityExistingChannels(doc.displayContent(), pos)
	existingSet := set.New(existing...)
	var items []protocol.CompletionItem
	appendChanCompletions := func(name string, t types.Type) {
		if t.Kind != types.KindChan || existingSet.Contains(name) {
			return
		}
		items = append(items, protocol.CompletionItem{
			Label:  name,
			Kind:   protocol.CompletionItemKindVariable,
			Detail: protocol.NewOptional(t.String()),
		})
	}
	if root != nil {
		symbols, err := root.Search(ctx, prefix)
		if err != nil {
			s.cfg.L.Error("failed to search global resolver for authority completions", zap.Error(err))
		}
		for _, sym := range symbols {
			appendChanCompletions(sym.Name, sym.Type)
		}
	}
	if doc.IR.Symbols != nil {
		scopes, err := doc.IR.Symbols.Search(ctx, prefix)
		if err != nil {
			s.cfg.L.Error("failed to search scope for authority completions", zap.Error(err))
		}
		for _, scope := range scopes {
			appendChanCompletions(scope.Name, scope.Type)
		}
	}
	return items
}

func (s *Server) resolveFunctionType(
	ctx context.Context,
	doc *Document,
	name string,
	root *symbol.Symbol,
) (types.Type, bool) {
	if doc.IR.Symbols != nil {
		sym, err := doc.IR.Symbols.Resolve(ctx, name, symbol.WithoutUsageTracking)
		if err == nil && sym.Type.Kind == types.KindFunction {
			return sym.Type, true
		}
	}
	if root != nil {
		sym, err := root.Resolve(ctx, name, symbol.WithoutUsageTracking)
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
	root *symbol.Symbol,
) []protocol.CompletionItem {
	seen := make(set.Set[string])
	var items []protocol.CompletionItem
	addItem := func(name string, t types.Type) {
		if seen.Contains(name) || !filter(t) {
			return
		}
		seen.Add(name)
		items = append(items, protocol.CompletionItem{
			Label:  name,
			Kind:   protocol.CompletionItemKindVariable,
			Detail: protocol.NewOptional(t.String()),
		})
	}
	if root != nil {
		symbols, err := root.Search(ctx, prefix)
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

func (s *Server) getInputParamCompletions(
	ctx context.Context,
	doc *Document,
	prefix string,
	inputInfo *inputContextInfo,
	root *symbol.Symbol,
) []protocol.CompletionItem {
	fnType, ok := s.resolveFunctionType(ctx, doc, inputInfo.functionName, root)
	if !ok {
		return []protocol.CompletionItem{}
	}
	existingSet := set.New(inputInfo.existingParams...)
	var items []protocol.CompletionItem
	for _, param := range fnType.Inputs {
		if existingSet.Contains(param.Name) || !strings.HasPrefix(param.Name, prefix) {
			continue
		}
		items = append(items, protocol.CompletionItem{
			Label:            param.Name,
			Kind:             protocol.CompletionItemKindProperty,
			Detail:           protocol.NewOptional(param.Type.String()),
			InsertText:       protocol.NewOptional(param.Name + "="),
			InsertTextFormat: protocol.InsertTextFormatPlainText,
		})
	}
	return items
}

func (s *Server) getInputValueCompletions(
	ctx context.Context,
	doc *Document,
	prefix string,
	inputInfo *inputContextInfo,
	root *symbol.Symbol,
) []protocol.CompletionItem {
	fnType, ok := s.resolveFunctionType(ctx, doc, inputInfo.functionName, root)
	if !ok {
		return []protocol.CompletionItem{}
	}
	param, found := fnType.Inputs.Get(inputInfo.currentParamName)
	if !found {
		return []protocol.CompletionItem{}
	}
	var items []protocol.CompletionItem
	if param.Type.Kind == types.KindChan {
		items = s.collectSymbols(ctx, doc, prefix, func(t types.Type) bool {
			return t.Kind == types.KindChan
		}, root)
	} else if param.Type.IsNumeric() {
		for _, c := range completions {
			if (c.Category&categoryUnit) == 0 || !strings.HasPrefix(c.Label, prefix) {
				continue
			}
			items = append(items, protocol.CompletionItem{
				Label:  c.Label,
				Kind:   c.Kind,
				Detail: optStr(c.Detail),
			})
		}
	}
	nonChanSymbols := s.collectSymbols(ctx, doc, prefix, func(t types.Type) bool {
		return t.Kind != types.KindChan
	}, root)
	items = append(items, nonChanSymbols...)
	return items
}
