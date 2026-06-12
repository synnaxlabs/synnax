// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package table

import (
	"fmt"
	"slices"

	"github.com/synnaxlabs/x/set"
)

// minCellDim is the floor enforced on row and column sizes.
const minCellDim = 32

// baseRowDim and baseColDim are the defaults used when an action bootstraps
// the opposing axis on an empty table.
const (
	baseRowDim = 36
	baseColDim = 72
)

// deriveCellKey returns the key for the index-th replica of template. Both
// reducers run the same scheme so optimistic client state agrees with the
// server.
func deriveCellKey(templateKey string, index int) string {
	if len(templateKey) < 36 {
		return fmt.Sprintf("%s-%04x", templateKey, index)
	}
	return templateKey[:32] + fmt.Sprintf("%04x", index)
}

// expandTemplate returns count replicas of template with keys derived via
// deriveCellKey.
func expandTemplate(template Cell, count int) []Cell {
	cells := make([]Cell, count)
	for i := range cells {
		cells[i] = Cell{
			Key:    deriveCellKey(template.Key, i),
			Config: template.Config,
		}
	}
	return cells
}

// Handle replaces the table's name.
func (p RenamePayload) Handle(state Table) (Table, error) {
	state.Name = p.Name
	return state, nil
}

// Handle inserts a row at the given index. When CellTemplate is non-nil it
// replaces Cells with one replica per existing column (or one when the table
// is empty, which also seeds a default column). When both are absent against
// a column-less table, the new row is inserted with no cells. Sizes clamp
// up to minCellDim; indices clamp down to the end of the rows slice.
func (p AddRowPayload) Handle(state Table) (Table, error) {
	cells := p.Cells
	if p.CellTemplate != nil {
		n := len(state.Columns)
		if n == 0 && len(state.Rows) == 0 {
			n = 1
		}
		cells = expandTemplate(*p.CellTemplate, n)
	}
	if len(state.Columns) == 0 && len(cells) > 0 {
		state.Columns = slices.Repeat([]Column{{Size: baseColDim}}, len(cells))
	}
	keys := make([]string, len(cells))
	for i, c := range cells {
		keys[i] = c.Key
	}
	row := Row{Size: max(p.Size, minCellDim), Cells: keys}
	idx := min(int(p.Index), len(state.Rows))
	state.Rows = slices.Insert(state.Rows, idx, row)
	if state.Cells == nil {
		state.Cells = make(map[string]CellConfig, len(cells))
	}
	for _, c := range cells {
		state.Cells[c.Key] = c.Config
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
	state.Rows = slices.Delete(state.Rows, idx, idx+1)
	return state, nil
}

// Handle inserts a column at the given index, mirroring AddRow with axes
// swapped. Cells whose row index exceeds the row count enter the cells map
// without being referenced by any row.
func (p AddColPayload) Handle(state Table) (Table, error) {
	cells := p.Cells
	if p.CellTemplate != nil {
		n := len(state.Rows)
		if n == 0 && len(state.Columns) == 0 {
			n = 1
		}
		cells = expandTemplate(*p.CellTemplate, n)
	}
	if len(state.Rows) == 0 && len(cells) > 0 {
		state.Rows = slices.Repeat([]Row{{Size: baseRowDim}}, len(cells))
	}
	idx := min(int(p.Index), len(state.Columns))
	state.Columns = slices.Insert(state.Columns, idx, Column{Size: max(p.Size, minCellDim)})
	if state.Cells == nil {
		state.Cells = make(map[string]CellConfig, len(cells))
	}
	for i := range state.Rows {
		if i >= len(cells) {
			break
		}
		rowIdx := min(idx, len(state.Rows[i].Cells))
		state.Rows[i].Cells = slices.Insert(state.Rows[i].Cells, rowIdx, cells[i].Key)
	}
	for _, c := range cells {
		state.Cells[c.Key] = c.Config
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
	state.Columns = slices.Delete(state.Columns, idx, idx+1)
	for i := range state.Rows {
		if idx >= len(state.Rows[i].Cells) {
			continue
		}
		delete(state.Cells, state.Rows[i].Cells[idx])
		state.Rows[i].Cells = slices.Delete(state.Rows[i].Cells, idx, idx+1)
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

// Handle replaces the cell config stored under p.Cell.Key. No-op if no entry
// with that key exists.
func (p SetCellPayload) Handle(state Table) (Table, error) {
	if _, ok := state.Cells[p.Cell.Key]; !ok {
		return state, nil
	}
	state.Cells[p.Cell.Key] = p.Cell.Config
	return state, nil
}

// Handle erases the selected cells. Fully-selected rows and columns are
// removed entirely; surviving cells in the selection have their variant and
// props replaced with the template's, keeping their original keys. Cells
// not in state.Cells are silently skipped.
func (p EraseCellsPayload) Handle(state Table) (Table, error) {
	if len(p.Cells) == 0 {
		return state, nil
	}
	selected := set.New(p.Cells...)
	type pos struct{ row, col int }
	posOf := make(map[string]pos)
	for r, row := range state.Rows {
		for c, k := range row.Cells {
			posOf[k] = pos{row: r, col: c}
		}
	}
	rowTally := make(map[int]int)
	colTally := make(map[int]int)
	for k := range selected {
		p, ok := posOf[k]
		if !ok {
			continue
		}
		rowTally[p.row]++
		colTally[p.col]++
	}
	fullRowIdx := []int{}
	for rowIdx, tally := range rowTally {
		if tally > 0 && tally == len(state.Rows[rowIdx].Cells) {
			fullRowIdx = append(fullRowIdx, rowIdx)
		}
	}
	slices.Sort(fullRowIdx)
	fullColIdx := []int{}
	for colIdx, tally := range colTally {
		if tally == len(state.Rows) {
			fullColIdx = append(fullColIdx, colIdx)
		}
	}
	slices.Sort(fullColIdx)
	for _, idx := range slices.Backward(fullRowIdx) {
		for _, k := range state.Rows[idx].Cells {
			delete(state.Cells, k)
		}
		state.Rows = slices.Delete(state.Rows, idx, idx+1)
	}
	for _, idx := range slices.Backward(fullColIdx) {
		state.Columns = slices.Delete(state.Columns, idx, idx+1)
		for r := range state.Rows {
			if idx >= len(state.Rows[r].Cells) {
				continue
			}
			delete(state.Cells, state.Rows[r].Cells[idx])
			state.Rows[r].Cells = slices.Delete(state.Rows[r].Cells, idx, idx+1)
		}
	}
	for _, k := range p.Cells {
		if _, ok := state.Cells[k]; !ok {
			continue
		}
		state.Cells[k] = p.Template
	}
	return state, nil
}
