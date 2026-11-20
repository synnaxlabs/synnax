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

	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/patch"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

// minCellDim is the floor enforced on row and column sizes.
const minCellDim = 32

// baseRow and baseCol are the entries an action gives the opposing axis when it
// bootstraps that axis on an empty table. Both take their size from the schema.
func baseRow(key string) Row {
	var r Row
	r.SetDefaults()
	r.Key = key
	return r
}

func baseCol(key string) Column {
	var c Column
	c.SetDefaults()
	c.Key = key
	return c
}

// deriveKey returns the index-th key derived from base. Both reducers run the same
// scheme so optimistic client state agrees with the server.
func deriveKey(base string, index int) string {
	if len(base) < 36 {
		return fmt.Sprintf("%s-%04x", base, index)
	}
	return base[:32] + fmt.Sprintf("%04x", index)
}

// expandTemplate returns count replicas of template with keys derived via deriveKey.
func expandTemplate(template Cell, count int) []Cell {
	cells := make([]Cell, count)
	for i := range cells {
		cells[i] = Cell{
			Key:    deriveKey(template.Key, i),
			Config: template.Config,
		}
	}
	return cells
}

// rowIndex returns the index of the row with the given key, or -1 if no row has it.
func rowIndex(rows []Row, key string) int {
	return slices.IndexFunc(rows, func(r Row) bool { return r.Key == key })
}

// colIndex returns the index of the column with the given key, or -1 if no column has
// it.
func colIndex(cols []Column, key string) int {
	return slices.IndexFunc(cols, func(c Column) bool { return c.Key == key })
}

// Handle returns an error wrapping validate.ErrValidation, as only the Core creates a
// table.
func (CreatePayload) Handle(Table) (Table, error) {
	return Table{}, actions.CoreOnlyError("a table create")
}

// Handle inserts a row with the given key at the given index. When CellTemplate is
// non-nil it replaces Cells with one replica per existing column (or one when the table
// is empty, which also creates a default column keyed from the row's key). When both
// are absent against a column-less table, the new row is inserted with no cells. Sizes
// clamp up to minCellDim; indices clamp down to the end of the rows slice. Returns a
// validation error if the key is empty.
func (p AddRowPayload) Handle(state Table) (Table, error) {
	if p.Key == "" {
		return Table{}, errors.Wrap(validate.ErrValidation, "row key is required")
	}
	cells := p.Cells
	if p.CellTemplate != nil {
		n := len(state.Columns)
		if n == 0 && len(state.Rows) == 0 {
			n = 1
		}
		cells = expandTemplate(*p.CellTemplate, n)
	}
	if len(state.Columns) == 0 && len(cells) > 0 {
		state.Columns = make([]Column, len(cells))
		for i := range state.Columns {
			state.Columns[i] = baseCol(deriveKey(p.Key, i))
		}
	}
	row := Row{Key: p.Key, Size: max(p.Size, minCellDim), Cells: slices.Clone(cells)}
	idx := min(int(p.Index), len(state.Rows))
	state.Rows = slices.Insert(state.Rows, idx, row)
	return state, nil
}

// Handle removes the row with the given key and its cells. No-op if no row has the key.
func (p RemoveRowPayload) Handle(state Table) (Table, error) {
	idx := rowIndex(state.Rows, p.Key)
	if idx == -1 {
		return state, nil
	}
	state.Rows = slices.Delete(state.Rows, idx, idx+1)
	return state, nil
}

// Handle inserts a column with the given key at the given index, mirroring AddRow with
// axes swapped. Cells whose index exceeds the row count are dropped. Returns a
// validation error if the key is empty.
func (p AddColPayload) Handle(state Table) (Table, error) {
	if p.Key == "" {
		return Table{}, errors.Wrap(validate.ErrValidation, "column key is required")
	}
	cells := p.Cells
	if p.CellTemplate != nil {
		n := len(state.Rows)
		if n == 0 && len(state.Columns) == 0 {
			n = 1
		}
		cells = expandTemplate(*p.CellTemplate, n)
	}
	if len(state.Rows) == 0 && len(cells) > 0 {
		state.Rows = make([]Row, len(cells))
		for i := range state.Rows {
			state.Rows[i] = baseRow(deriveKey(p.Key, i))
		}
	}
	idx := min(int(p.Index), len(state.Columns))
	state.Columns = slices.Insert(
		state.Columns,
		idx,
		Column{Key: p.Key, Size: max(p.Size, minCellDim)},
	)
	for i := range min(len(state.Rows), len(cells)) {
		row := &state.Rows[i]
		row.Cells = slices.Insert(row.Cells, min(idx, len(row.Cells)), cells[i])
	}
	return state, nil
}

// Handle removes the column with the given key and its cell in every row. No-op if no
// column has the key.
func (p RemoveColPayload) Handle(state Table) (Table, error) {
	idx := colIndex(state.Columns, p.Key)
	if idx == -1 {
		return state, nil
	}
	state.Columns = slices.Delete(state.Columns, idx, idx+1)
	for i := range state.Rows {
		deleteCell(&state.Rows[i], idx)
	}
	return state, nil
}

// deleteCell removes the cell at idx from the row. No-op if the row is shorter.
func deleteCell(row *Row, idx int) {
	if idx < len(row.Cells) {
		row.Cells = slices.Delete(row.Cells, idx, idx+1)
	}
}

// Handle resizes the row with the given key. Sizes below the minimum cell dimension are
// clamped up to the floor. No-op if no row has the key.
func (p ResizeRowPayload) Handle(state Table) (Table, error) {
	idx := rowIndex(state.Rows, p.Key)
	if idx == -1 {
		return state, nil
	}
	state.Rows[idx].Size = max(p.Size, minCellDim)
	return state, nil
}

// Handle resizes the column with the given key. Sizes below the minimum cell dimension
// are clamped up to the floor. No-op if no column has the key.
func (p ResizeColPayload) Handle(state Table) (Table, error) {
	idx := colIndex(state.Columns, p.Key)
	if idx == -1 {
		return state, nil
	}
	state.Columns[idx].Size = max(p.Size, minCellDim)
	return state, nil
}

// move relocates the element at from to index to, shifting the elements between them by
// one.
func move[T any](s []T, from, to int) []T {
	v := s[from]
	return slices.Insert(slices.Delete(s, from, from+1), to, v)
}

// Handle moves the row with key p.Key to index p.To, clamping p.To to the last row.
// No-op if no row has the key.
func (p MoveRowPayload) Handle(state Table) (Table, error) {
	from := rowIndex(state.Rows, p.Key)
	if from == -1 {
		return state, nil
	}
	state.Rows = move(state.Rows, from, min(int(p.To), len(state.Rows)-1))
	return state, nil
}

// Handle moves the column with key p.Key to index p.To, clamping p.To to the last
// column, and moves the matching cell in every row that holds both indices. No-op if no
// column has the key.
func (p MoveColPayload) Handle(state Table) (Table, error) {
	from := colIndex(state.Columns, p.Key)
	if from == -1 {
		return state, nil
	}
	to := min(int(p.To), len(state.Columns)-1)
	state.Columns = move(state.Columns, from, to)
	for i := range state.Rows {
		if max(from, to) < len(state.Rows[i].Cells) {
			state.Rows[i].Cells = move(state.Rows[i].Cells, from, to)
		}
	}
	return state, nil
}

// cellPos is the position of a cell in a table.
type cellPos struct{ row, col int }

// cellPositions returns the position of every cell in the table, by cell key.
func cellPositions(rows []Row) map[string]cellPos {
	pos := make(map[string]cellPos)
	for r, row := range rows {
		for c, cell := range row.Cells {
			pos[cell.Key] = cellPos{row: r, col: c}
		}
	}
	return pos
}

// Handle replaces the config of the cell with the key p.Cell.Key. No-op if no row holds
// a cell with that key.
func (p SetCellPayload) Handle(state Table) (Table, error) {
	pos, ok := cellPositions(state.Rows)[p.Cell.Key]
	if !ok {
		return state, nil
	}
	state.Rows[pos.row].Cells[pos.col].Config = p.Cell.Config
	return state, nil
}

// Handle erases the selected cells. Fully-selected rows and columns are removed
// entirely; surviving cells in the selection take the template's config, keeping their
// original keys. Keys that no row holds are skipped.
func (p EraseCellsPayload) Handle(state Table) (Table, error) {
	if len(p.Cells) == 0 {
		return state, nil
	}
	selected := set.New(p.Cells...)
	posOf := cellPositions(state.Rows)
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
		state.Rows = slices.Delete(state.Rows, idx, idx+1)
	}
	for _, idx := range slices.Backward(fullColIdx) {
		state.Columns = slices.Delete(state.Columns, idx, idx+1)
		for r := range state.Rows {
			deleteCell(&state.Rows[r], idx)
		}
	}
	posOf = cellPositions(state.Rows)
	for k := range selected {
		if pos, ok := posOf[k]; ok {
			state.Rows[pos.row].Cells[pos.col].Config = p.Template
		}
	}
	return state, nil
}

// Handle applies the set to the table's name, group, or project, or to the config of a
// cell.
func (p SetPayload) Handle(state Table) (Table, error) {
	return apply(state, patch.Set(p))
}

// Handle applies the remove to the table's group or project, or to the config of a
// cell.
func (p RemovePayload) Handle(state Table) (Table, error) {
	return apply(state, patch.Remove(p))
}

// Handle applies the insert to the config of a cell.
func (p InsertPayload) Handle(state Table) (Table, error) {
	return apply(state, patch.Insert(p))
}

// Handle applies the move to the config of a cell.
func (p MovePayload) Handle(state Table) (Table, error) {
	return apply(state, patch.Move(p))
}

// apply applies a path action to the table's name, group, or project, or to the config
// of a cell. Any other path fails, except that a path through a row or cell that does
// not exist does nothing.
func apply[A patch.Action](state Table, a A) (Table, error) {
	c, err := patch.Begin(a)
	if err != nil {
		return Table{}, err
	}
	if err = state.Patch(c); err != nil {
		return Table{}, err
	}
	return state, nil
}

func newSetAction(s patch.Set) Action { return NewSetAction(SetPayload(s)) }

func newRemoveAction(r patch.Remove) Action { return NewRemoveAction(RemovePayload(r)) }
