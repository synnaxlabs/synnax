// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package join

import (
	"strconv"
	"strings"

	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
)

// PrimaryParam names the param of a join node that holds the index of its primary
// input.
const PrimaryParam = "primary"

// InputParam returns the name of the param of a join node that takes input i.
func InputParam(i int) string { return "in_" + strconv.Itoa(i) }

// OutputParam returns the name of the param of a join node that gives the joined
// samples of input i.
func OutputParam(i int) string { return "out_" + strconv.Itoa(i) }

const arrivalPrefix = "arrival_"

// ArrivalParam returns the name of the constant param of a join node whose presence
// makes input i pair by arrival.
func ArrivalParam(i int) string { return arrivalPrefix + strconv.Itoa(i) }

// Symbol returns the symbol of the join nodes named name: input i and output i of type
// inputs[i], input i pairing by arrival when arrival[i] is set, and the input at
// primary timing the outputs. The Host of name creates its nodes.
func Symbol(
	name string,
	inputs []types.Type,
	arrival []bool,
	primary int,
) symbol.Symbol {
	var props types.FunctionProperties
	for i, t := range inputs {
		props.Inputs = append(props.Inputs, types.Param{Name: InputParam(i), Type: t})
		props.Outputs = append(
			props.Outputs,
			types.Param{Name: OutputParam(i), Type: t},
		)
	}
	for i, a := range arrival {
		if a {
			props.Inputs = append(props.Inputs, types.Param{
				Name:  ArrivalParam(i),
				Type:  types.U8(),
				Value: uint8(1),
			})
		}
	}
	props.Inputs = append(props.Inputs, types.Param{
		Name:  PrimaryParam,
		Type:  types.I64(),
		Value: int64(primary),
	})
	return symbol.Symbol{
		Name: name,
		Kind: symbol.KindFunction,
		Exec: symbol.ExecFlow,
		Type: types.Function(props),
	}
}

// Host creates the live join nodes of one node type.
type Host struct {
	nodeType string
	now      func() telem.TimeStamp
}

// NewHost returns a Host of the join nodes of nodeType, which judge the arrival of
// their inputs by now.
func NewHost(nodeType string, now func() telem.TimeStamp) *Host {
	return &Host{nodeType: nodeType, now: now}
}

// Create implements node.Factory. It returns an error for a node with no primary param.
func (h *Host) Create(cfg node.Config) (node.Node, error) {
	if cfg.Node.Type != h.nodeType {
		return nil, query.ErrNotFound
	}
	n := &Node{State: cfg.State}
	var (
		dataTypes []telem.DataType
		arrivals  []int
		primary   = -1
	)
	for _, p := range cfg.Node.Inputs {
		switch {
		case p.Name == PrimaryParam:
			v, ok := types.DecodeValue(p.Value, p.Type).(int64)
			if !ok {
				return nil, errors.Newf("join param %s is not an i64", PrimaryParam)
			}
			primary = int(v)
		case strings.HasPrefix(p.Name, arrivalPrefix):
			i, err := strconv.Atoi(strings.TrimPrefix(p.Name, arrivalPrefix))
			if err != nil || i < 0 {
				return nil, errors.Newf("join param %s names no input", p.Name)
			}
			arrivals = append(arrivals, i)
		default:
			idx, err := cfg.State.ResolveInput(p.Name)
			if err != nil {
				return nil, err
			}
			n.inputs = append(n.inputs, idx)
			dataTypes = append(dataTypes, p.Type.ToTelem())
		}
	}
	if primary < 0 {
		return nil, errors.Newf(
			"join node %s has no %s param",
			cfg.Node.Key,
			PrimaryParam,
		)
	}
	arrival, err := arrivalMarks(arrivals, len(n.inputs))
	if err != nil {
		return nil, err
	}
	n.arrival = arrival
	n.last = make([]telem.TimeStamp, len(n.inputs))
	n.pushed = make([]bool, len(n.inputs))
	j, err := New(Config{
		DataTypes: dataTypes,
		Primary:   primary,
		Arrival:   arrival,
		Live:      true,
		Now:       h.now,
	})
	if err != nil {
		return nil, err
	}
	n.join = j
	return n, nil
}

// arrivalMarks returns, for each of count inputs, whether arrivals marks it to pair by
// arrival. It returns an error for a mark past the inputs.
func arrivalMarks(arrivals []int, count int) ([]bool, error) {
	arrival := make([]bool, count)
	for _, i := range arrivals {
		if i >= count {
			return nil, errors.Newf(
				"join param %s marks no input to pair by arrival",
				ArrivalParam(i),
			)
		}
		arrival[i] = true
	}
	return arrival, nil
}

// Node is a live join in an Arc program. Each output gives the joined samples of the
// input at its index, with the stamps and alignment of the primary.
type Node struct {
	*node.State
	join *Join
	// inputs holds the state index of each input, in input order.
	inputs []int
	out    Output
	// next is the index of the next run in out to emit.
	next int
	// unjoined is set when the join holds samples that no Next has joined.
	unjoined bool
	dropped  int
	// arrival marks the inputs that pair by arrival.
	arrival []bool
	// last holds the newest stamp pushed to each input that pushed is set for.
	last   []telem.TimeStamp
	pushed []bool
}

var _ node.Node = (*Node)(nil)

// Dropped returns the number of primary stamps that got no output since the last call,
// counting from the step the join settled on.
func (n *Node) Dropped() int {
	dropped := n.dropped
	n.dropped = 0
	return dropped
}

// Next pushes the new samples of each input, and emits the next joined run. It joins
// the pushed samples once it has emitted every run of the last join. The node runs
// again while runs or unjoined samples remain. A late sample, whose stamp does not
// rise past the newest that its input pushed, is left to history.
func (n *Node) Next(ctx node.Context) {
	for i, idx := range n.inputs {
		data, t, ok := n.ConsumeTimed(idx)
		if !ok {
			continue
		}
		if data, t = n.rising(i, data, t); t.Len() == 0 {
			continue
		}
		if err := n.join.Push(i, data, t); err != nil {
			ctx.ReportError(err)
			continue
		}
		n.unjoined = true
		if !n.arrival[i] {
			n.last[i], n.pushed[i] = t.ValueAt[telem.TimeStamp](-1), true
		}
	}
	if n.next >= len(n.out.Runs) && n.unjoined {
		n.out.Reset()
		n.next = 0
		n.unjoined = false
		n.join.Next(telem.TimeStampMax, &n.out)
		if n.out.Settled {
			n.dropped += n.out.Dropped
		}
	}
	if n.next < len(n.out.Runs) {
		n.emit(ctx)
	}
	if n.next < len(n.out.Runs) || n.unjoined {
		ctx.MarkSelfChanged()
	}
}

// rising returns the samples of data and t, the series of input i and its stamps,
// that come after the newest stamp the input pushed. A series whose stamps do not rise
// from there comes back empty. The samples of an arrival input all come back, since
// its stamps time nothing.
func (n *Node) rising(i int, data, t telem.Series) (telem.Series, telem.Series) {
	if n.arrival[i] {
		return data, t
	}
	count := int(t.Len())
	start := 0
	for n.pushed[i] && start < count && t.ValueAt[telem.TimeStamp](start) <= n.last[i] {
		start++
	}
	for k := start + 1; k < count; k++ {
		if t.ValueAt[telem.TimeStamp](k) <= t.ValueAt[telem.TimeStamp](k-1) {
			return telem.Series{}, telem.Series{}
		}
	}
	if start == 0 {
		return data, t
	}
	return from(data, start), from(t, start)
}

// from returns the samples of s, a series of a fixed-size type, from index start on.
func from(s telem.Series, start int) telem.Series {
	density := int(s.DataType.Density())
	out := s
	out.Data = s.Data[start*density:]
	out.Alignment = s.Alignment + telem.Alignment(start)
	return out
}

// emit copies the next run of out into the outputs, which the next Next of the join
// overwrites.
func (n *Node) emit(ctx node.Context) {
	run := n.out.Runs[n.next]
	n.next++
	bounds := telem.TimeRange{
		Start: run.Time.ValueAt[telem.TimeStamp](0),
		End:   run.Time.ValueAt[telem.TimeStamp](-1),
	}
	// The range ends past its last stamp, unless that stamp is the largest.
	if bounds.End < telem.TimeStampMax {
		bounds.End++
	}
	for i := range n.inputs {
		copyRun(n.Output(i), run.Values[i], run.Time.Alignment, bounds)
		copyRun(n.OutputTime(i), run.Time, run.Time.Alignment, bounds)
		n.Emit(ctx, i)
	}
}

// copyRun copies src into dst, keeping the buffer of dst, with the alignment and time
// range of the run.
func copyRun(
	dst *telem.Series,
	src telem.Series,
	alignment telem.Alignment,
	bounds telem.TimeRange,
) {
	dst.DataType = src.DataType
	dst.Data = append(dst.Data[:0], src.Data...)
	dst.Alignment = alignment
	dst.TimeRange = bounds
}
