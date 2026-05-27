// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package table

import "fmt"

const (
	// minCellDim is the floor enforced on row and column sizes.
	minCellDim = 32
	// baseRowDim and baseColDim are the defaults the AddRow / AddCol
	// reducers use when bootstrapping the missing axis on an empty table.
	baseRowDim = 36
	baseColDim = 72
)

// deriveCellKey derives a unique-per-index cell key from a template key by
// replacing the last four hex digits with the index encoded in hex. The
// template key is assumed to be a 36-character UUID; positions 14 (version)
// and 19 (variant) sit in the prefix so the derived key keeps the UUID v4
// layout. Both Go and TS reducers run the same scheme so the optimistic
// flux store agrees with the server.
func deriveCellKey(templateKey string, index int) string {
	if len(templateKey) < 36 {
		return fmt.Sprintf("%s-%04x", templateKey, index)
	}
	return templateKey[:32] + fmt.Sprintf("%04x", index)
}

// expandTemplate returns a slice of replica cells derived from a template,
// one per axis position. The first replica gets key deriveCellKey(template,
// 0), the second deriveCellKey(template, 1), and so on.
func expandTemplate(template Cell, count int) []Cell {
	cells := make([]Cell, count)
	for i := range cells {
		cells[i] = Cell{
			Key:     deriveCellKey(template.Key, i),
			Variant: template.Variant,
			Props:   template.Props,
		}
	}
	return cells
}

// Handle replaces the table's name.
func (p RenamePayload) Handle(state Table) (Table, error) {
	state.Name = p.Name
	return state, nil
}

// Handle inserts a row at the given index. When CellTemplate is set
// (Key != ""), the reducer ignores Cells and creates one replica per
// existing column (or one if the table has no columns yet), copying the
// template's variant and props and deriving each replica's key via
// deriveCellKey. When CellTemplate is unset, Cells carries one Cell per
// column in left-to-right order and the reducer uses them as-is (the
// inverse path of RemoveRow goes through this branch). Sizes below the
// minimum cell dimension are clamped up to the floor. Out-of-range indices
// clamp to the end of the rows slice.
func (p AddRowPayload) Handle(state Table) (Table, error) {
	hasTemplate := p.CellTemplate.Key != ""
	cells := p.Cells
	if hasTemplate {
		n := max(len(state.Columns), 1)
		cells = expandTemplate(p.CellTemplate, n)
	}
	// Bootstrap: a row arriving against an empty table implies the columns
	// the row needs. Create one default-sized column per cell.
	if len(state.Columns) == 0 && len(cells) > 0 {
		state.Columns = make([]Column, len(cells))
		for i := range state.Columns {
			state.Columns[i].Size = baseColDim
		}
	}
	keys := make([]string, len(cells))
	for i, c := range cells {
		keys[i] = c.Key
	}
	row := Row{Size: max(p.Size, minCellDim), Cells: keys}
	idx := int(p.Index)
	if idx > len(state.Rows) {
		idx = len(state.Rows)
	}
	state.Rows = append(state.Rows[:idx], append([]Row{row}, state.Rows[idx:]...)...)
	if state.Cells == nil {
		state.Cells = make(map[string]Cell, len(cells))
	}
	for _, c := range cells {
		state.Cells[c.Key] = c
	}
	return state, nil
}

// Handle removes the row at the given index and drops every cell it
// referenced. No-op if the index is out of range.
func (p RemoveRowPayload) Handle(state Table) (Table, error) {
	idx := int(p.Index)
	if idx >= len(state.Rows) {
		return state, nil
	}
	for _, k := range state.Rows[idx].Cells {
		delete(state.Cells, k)
	}
	state.Rows = append(state.Rows[:idx], state.Rows[idx+1:]...)
	return state, nil
}

// Handle inserts a column at the given index. When CellTemplate is set
// (Key != ""), the reducer ignores Cells and creates one replica per
// existing row (or one if the table has no rows yet), copying the
// template's variant and props and deriving each replica's key via
// deriveCellKey. When CellTemplate is unset, Cells carries one Cell per row
// in top-to-bottom order; cells whose row index exceeds the row count are
// added to the map but not referenced by any row. Sizes below the minimum
// cell dimension are clamped up to the floor. Out-of-range indices clamp to
// the end of every row's cells list.
func (p AddColPayload) Handle(state Table) (Table, error) {
	hasTemplate := p.CellTemplate.Key != ""
	cells := p.Cells
	if hasTemplate {
		n := max(len(state.Rows), 1)
		cells = expandTemplate(p.CellTemplate, n)
	}
	// Bootstrap: a column arriving against an empty table implies the rows
	// the column needs. Create one default-sized empty row per cell; the
	// loop below splices each cell into its row.
	if len(state.Rows) == 0 && len(cells) > 0 {
		state.Rows = make([]Row, len(cells))
		for i := range state.Rows {
			state.Rows[i].Size = baseRowDim
		}
	}
	idx := int(p.Index)
	if idx > len(state.Columns) {
		idx = len(state.Columns)
	}
	state.Columns = append(
		state.Columns[:idx],
		append([]Column{{Size: max(p.Size, minCellDim)}}, state.Columns[idx:]...)...,
	)
	if state.Cells == nil {
		state.Cells = make(map[string]Cell, len(cells))
	}
	for i := range state.Rows {
		if i >= len(cells) {
			break
		}
		rowIdx := idx
		if rowIdx > len(state.Rows[i].Cells) {
			rowIdx = len(state.Rows[i].Cells)
		}
		state.Rows[i].Cells = append(
			state.Rows[i].Cells[:rowIdx],
			append([]string{cells[i].Key}, state.Rows[i].Cells[rowIdx:]...)...,
		)
	}
	for _, c := range cells {
		state.Cells[c.Key] = c
	}
	return state, nil
}

// Handle removes the column at the given index and drops every cell that
// column referenced across every row. No-op if the index is out of range.
func (p RemoveColPayload) Handle(state Table) (Table, error) {
	idx := int(p.Index)
	if idx >= len(state.Columns) {
		return state, nil
	}
	state.Columns = append(state.Columns[:idx], state.Columns[idx+1:]...)
	for i := range state.Rows {
		if idx >= len(state.Rows[i].Cells) {
			continue
		}
		delete(state.Cells, state.Rows[i].Cells[idx])
		state.Rows[i].Cells = append(
			state.Rows[i].Cells[:idx],
			state.Rows[i].Cells[idx+1:]...,
		)
	}
	return state, nil
}

// Handle resizes the row at the given index. Sizes below the minimum cell
// dimension are clamped up to the floor. No-op if the index is out of range.
func (p ResizeRowPayload) Handle(state Table) (Table, error) {
	idx := int(p.Index)
	if idx >= len(state.Rows) {
		return state, nil
	}
	state.Rows[idx].Size = max(p.Size, minCellDim)
	return state, nil
}

// Handle resizes the column at the given index. Sizes below the minimum
// cell dimension are clamped up to the floor. No-op if the index is out of
// range.
func (p ResizeColPayload) Handle(state Table) (Table, error) {
	idx := int(p.Index)
	if idx >= len(state.Columns) {
		return state, nil
	}
	state.Columns[idx].Size = max(p.Size, minCellDim)
	return state, nil
}

// Handle replaces the cell stored under p.Cell.Key. No-op if no entry with
// that key exists.
func (p SetCellPayload) Handle(state Table) (Table, error) {
	if _, ok := state.Cells[p.Cell.Key]; !ok {
		return state, nil
	}
	state.Cells[p.Cell.Key] = p.Cell
	return state, nil
}
