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
	"slices"
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
	arrival := make([]bool, len(n.inputs))
	for i := range arrival {
		arrival[i] = slices.ContainsFunc(cfg.Node.Inputs, func(p types.Param) bool {
			return p.Name == ArrivalParam(i)
		})
	}
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
// again while runs or unjoined samples remain. An input whose samples the join refuses
// reports an error, and the other inputs still push. The step then emits nothing, so
// its output, which the error drops, holds no joined sample; the next step joins them.
func (n *Node) Next(ctx node.Context) {
	failed := false
	for i, idx := range n.inputs {
		data, t, ok := n.ConsumeTimed(idx)
		if !ok {
			continue
		}
		if err := n.join.Push(i, data, t); err != nil {
			ctx.ReportError(err)
			failed = true
			continue
		}
		n.unjoined = true
	}
	if failed {
		return
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
