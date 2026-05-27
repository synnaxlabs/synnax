// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package table

const (
	// minCellDim is the floor enforced on row and column sizes.
	minCellDim = 32
	// baseRowDim and baseColDim are the defaults the AddRow / AddCol
	// reducers use when bootstrapping the missing axis on an empty table.
	baseRowDim = 36
	baseColDim = 72
)

// Handle replaces the table's name.
func (p RenamePayload) Handle(state Table) (Table, error) {
	state.Name = p.Name
	return state, nil
}

// Handle inserts a row at the given index with the provided cell values, and
// records each cell in the table's cells map. Sizes below the minimum cell
// dimension are clamped up to the floor. Out-of-range indices clamp to the
// end of the rows slice; cells whose keys collide with existing entries
// overwrite the prior value.
func (p AddRowPayload) Handle(state Table) (Table, error) {
	// Bootstrap: a row arriving against an empty table implies the columns
	// the row needs. Create one default-sized column per cell in the payload.
	if len(state.Columns) == 0 && len(p.Cells) > 0 {
		state.Columns = make([]Column, len(p.Cells))
		for i := range state.Columns {
			state.Columns[i].Size = baseColDim
		}
	}
	keys := make([]string, len(p.Cells))
	for i, c := range p.Cells {
		keys[i] = c.Key
	}
	row := Row{Size: max(p.Size, minCellDim), Cells: keys}
	idx := int(p.Index)
	if idx > len(state.Rows) {
		idx = len(state.Rows)
	}
	state.Rows = append(state.Rows[:idx], append([]Row{row}, state.Rows[idx:]...)...)
	if state.Cells == nil {
		state.Cells = make(map[string]Cell, len(p.Cells))
	}
	for _, c := range p.Cells {
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

// Handle inserts a column at the given index with the provided cell values.
// Each cell is inserted at the column index inside the corresponding row's
// cells list and recorded in the table's cells map. Sizes below the minimum
// cell dimension are clamped up to the floor. Out-of-range indices clamp to
// the end of every row's cells list. When the payload carries fewer cells
// than there are rows the trailing rows are left without a new cell entry;
// extra cells beyond the row count are still added to the map but not
// referenced by any row.
func (p AddColPayload) Handle(state Table) (Table, error) {
	// Bootstrap: a column arriving against an empty table implies the rows
	// the column needs. Create one default-sized empty row per cell in the
	// payload; the loop below splices each cell into its row.
	if len(state.Rows) == 0 && len(p.Cells) > 0 {
		state.Rows = make([]Row, len(p.Cells))
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
		state.Cells = make(map[string]Cell, len(p.Cells))
	}
	for i := range state.Rows {
		if i >= len(p.Cells) {
			break
		}
		rowIdx := idx
		if rowIdx > len(state.Rows[i].Cells) {
			rowIdx = len(state.Rows[i].Cells)
		}
		state.Rows[i].Cells = append(
			state.Rows[i].Cells[:rowIdx],
			append([]string{p.Cells[i].Key}, state.Rows[i].Cells[rowIdx:]...)...,
		)
	}
	for _, c := range p.Cells {
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
