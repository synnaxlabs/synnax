// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ranges

import (
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/lsp/doc"
	"github.com/synnaxlabs/x/telem"
)

const (
	createMemberName = "create"
	startMemberName  = "start"
	endMemberName    = "end"
	moduleName       = "ranges"
)

// nowSentinel is the default for start_time and time. The runtime substitutes
// the current time at fire when an input equals it, so the default tracks "now".
const nowSentinel = int64(-1)

// createDoc is the LSP hover body for ranges.create. The renderer prepends the
// title from the symbol name and kind, so it is omitted here.
var createDoc = doc.New(
	doc.Paragraph("Creates a range on Core spanning the given time bounds."),
	doc.Divider(),
	doc.Code("arc", "range_key := trigger -> ranges.create{name=\"Terminal Count\"}"),
	doc.Divider(),
	doc.Paragraph("start_time defaults to now, end_time to open. Outputs the range key."),
)

// startDoc is the LSP hover body for ranges.start.
var startDoc = doc.New(
	doc.Paragraph("Sets the start bound of an existing range on Core."),
	doc.Divider(),
	doc.Code("arc", "trigger -> ranges.start{key=range_key}"),
	doc.Divider(),
	doc.Paragraph("time defaults to now. Outputs the range key."),
)

// endDoc is the LSP hover body for ranges.end.
var endDoc = doc.New(
	doc.Paragraph("Sets the end bound of an existing range on Core."),
	doc.Divider(),
	doc.Code("arc", "trigger -> ranges.end{key=range_key}"),
	doc.Divider(),
	doc.Paragraph("time defaults to now. Outputs the range key."),
)

// moduleDoc is the LSP hover body for the ranges module itself.
var moduleDoc = doc.New(
	doc.Paragraph("Creates and manages ranges (named time regions) on Core."),
)

// newCreateSymbolType returns a fresh ranges.create function type per call so
// analysis never mutates a shared symbol. Inputs with a default are optional.
func newCreateSymbolType() types.Type {
	params := types.Params{
		{Name: "name", Type: types.String()},
		{Name: "start_time", Type: types.TimeStamp(), Value: nowSentinel},
		{Name: "end_time", Type: types.TimeStamp(), Value: int64(telem.TimeStampMax)},
		{Name: "color", Type: types.String(), Value: ""},
		{Name: "parent", Type: types.String(), Value: ""},
		{Name: "labels", Type: types.Series(types.String()), Value: []string{}},
	}
	return types.Function(types.FunctionProperties{
		Inputs:  params,
		Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.String()}},
	})
}

// newStartEndSymbolType returns a fresh ranges.start/ranges.end function type
// per call so analysis never mutates a shared symbol. time defaults to now.
func newStartEndSymbolType() types.Type {
	params := types.Params{
		{Name: "key", Type: types.String()},
		{Name: "time", Type: types.TimeStamp(), Value: nowSentinel},
	}
	return types.Function(types.FunctionProperties{
		Inputs:  params,
		Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.String()}},
	})
}

// NewSymbols returns a fresh slice of ambient prelude symbols this package
// contributes: the ranges module with its create, start, and end members.
func NewSymbols() []*symbol.Symbol {
	createMember := &symbol.Symbol{
		Name:    createMemberName,
		Kind:    symbol.KindFunction,
		Exec:    symbol.ExecBoth,
		Type:    newCreateSymbolType(),
		Trigger: symbol.TriggerOnly,
		Doc:     createDoc,
	}
	startMember := &symbol.Symbol{
		Name:    startMemberName,
		Kind:    symbol.KindFunction,
		Exec:    symbol.ExecBoth,
		Type:    newStartEndSymbolType(),
		Trigger: symbol.TriggerOnly,
		Doc:     startDoc,
	}
	endMember := &symbol.Symbol{
		Name:    endMemberName,
		Kind:    symbol.KindFunction,
		Exec:    symbol.ExecBoth,
		Type:    newStartEndSymbolType(),
		Trigger: symbol.TriggerOnly,
		Doc:     endDoc,
	}
	mod := &symbol.Symbol{Name: moduleName, Kind: symbol.KindModule, Doc: moduleDoc}
	mod.AddChild(createMember)
	mod.AddChild(startMember)
	mod.AddChild(endMember)
	return []*symbol.Symbol{mod}
}
