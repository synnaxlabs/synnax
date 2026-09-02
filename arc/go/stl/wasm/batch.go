// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package wasm

import (
	"github.com/synnaxlabs/arc/compiler"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"github.com/tetratelabs/wazero/api"
)

const pageSize = 65536

// arena carves scratch regions out of guest linear memory by growing it past
// everything the compiler reserved. A region is never returned; a node that outgrows
// one takes a larger one.
type arena struct{ mem api.Memory }

func (a *arena) alloc(size uint32) (base, capacity uint32, ok bool) {
	pages := (size + pageSize - 1) / pageSize
	prev, ok := a.mem.Grow(pages)
	if !ok {
		return 0, 0, false
	}
	return prev * pageSize, pages * pageSize, true
}

// batchCall drives a node's batch wrapper and owns the scratch region of guest linear
// memory the wrapper reads its inputs from and writes its output to.
type batchCall struct {
	fn    api.Function
	arena *arena
	stack []uint64
	// offsets and samples are recomputed each pass: where in the scratch region each
	// input lands, and how many samples are staged there.
	offsets    []uint32
	samples    []uint32
	densities  []int
	outDensity int
	ptr        uint32
	capacity   uint32
}

// newBatchCall returns nil when the guest exports no wrapper for the node's function,
// or when the node's shape does not match one: a single returned value, fixed-width
// samples throughout, and one host param per guest param.
func (w *Module) newBatchCall(
	cfg node.Config,
	irFn ir.Function,
	outputMemoryBase uint32,
) *batchCall {
	fn := w.Module.ExportedFunction(cfg.Node.Type + compiler.BatchSuffix)
	if fn == nil || outputMemoryBase != 0 {
		return nil
	}
	if len(irFn.Outputs) != 1 || len(cfg.Node.Outputs) != 1 ||
		len(cfg.Node.Inputs) != len(irFn.Inputs) {
		return nil
	}
	outDensity, ok := fixedDensity(irFn.Outputs[0].Type)
	if !ok {
		return nil
	}
	densities := make([]int, len(irFn.Inputs))
	for i, inp := range irFn.Inputs {
		if densities[i], ok = fixedDensity(inp.Type); !ok {
			return nil
		}
	}
	if w.arena == nil {
		w.arena = &arena{mem: w.Memory}
	}
	return &batchCall{
		fn:         fn,
		arena:      w.arena,
		stack:      make([]uint64, compiler.BatchInputParam+2*len(irFn.Inputs)),
		offsets:    make([]uint32, len(irFn.Inputs)),
		samples:    make([]uint32, len(irFn.Inputs)),
		densities:  densities,
		outDensity: outDensity,
	}
}

func fixedDensity(t types.Type) (int, bool) {
	switch t.Kind {
	case types.KindU8, types.KindU16, types.KindU32, types.KindU64,
		types.KindI8, types.KindI16, types.KindI32, types.KindI64,
		types.KindF32, types.KindF64, types.KindBool:
		return t.Density(), true
	default:
		return 0, false
	}
}

func align8(v uint32) uint32 { return (v + 7) &^ 7 }

func putValue(dst []byte, density int, v uint64) {
	switch density {
	case 1:
		dst[0] = byte(v)
	case 2:
		telem.ByteOrder.PutUint16(dst, uint16(v))
	case 4:
		telem.ByteOrder.PutUint32(dst, uint32(v))
	case 8:
		telem.ByteOrder.PutUint64(dst, v)
	}
}

// runBatch computes the whole output series in one guest call, reporting false when
// the node's series do not fit the wrapper's fixed-width layout. A trap aborts every
// remaining sample, where the per-sample path would carry on.
func (n *nodeImpl) runBatch(
	ctx node.Context,
	count int64,
	inputTime telem.Series,
	clockStamp bool,
) bool {
	b := n.batch
	out := n.Output(0)
	if int(out.DataType.Density()) != b.outDensity {
		return false
	}
	size := align8(uint32(count) * uint32(b.outDensity))
	for i := range n.ir.Inputs {
		samples := uint32(1)
		if n.batchEdgeFed(i) {
			in := n.Input(i)
			if int(in.DataType.Density()) != b.densities[i] {
				return false
			}
			if in.Len() != 1 {
				samples = uint32(count)
			}
		}
		b.offsets[i], b.samples[i] = size, samples
		size += align8(samples * uint32(b.densities[i]))
	}
	if size > b.capacity {
		// Doubling bounds what a node abandons as its series grow: ever-longer
		// batches cost a logarithmic number of grows, not one each.
		base, capacity, ok := b.arena.alloc(max(size, 2*b.capacity))
		if !ok {
			return false
		}
		b.ptr, b.capacity = base, capacity
	}
	buf, ok := n.mem.Read(b.ptr, size)
	if !ok {
		return false
	}
	for i := range n.ir.Inputs {
		var (
			density = b.densities[i]
			off     = b.offsets[i]
			dst     = buf[off : off+b.samples[i]*uint32(density)]
			stride  = uint64(density)
		)
		switch in := n.Input(i); {
		case b.samples[i] == 1 && n.batchEdgeFed(i):
			stride = 0
			copy(dst, in.Data)
		case b.samples[i] == 1:
			stride = 0
			putValue(dst, density, n.params[i])
		case in.Len() == count:
			copy(dst, in.Data)
		default:
			for s := range int(count) {
				src := (s % int(in.Len())) * density
				copy(dst[s*density:], in.Data[src:src+density])
			}
		}
		b.stack[compiler.BatchInputParam+2*i] = uint64(b.ptr + off)
		b.stack[compiler.BatchInputParam+2*i+1] = stride
	}
	b.stack[compiler.BatchCountParam] = uint64(count)
	b.stack[compiler.BatchOutParam] = uint64(b.ptr)
	if err := b.fn.CallWithStack(ctx.Context, b.stack); err != nil {
		ctx.ReportError(errors.Wrapf(
			err,
			"WASM execution failed in node %s over %d samples",
			n.ir.Key,
			count,
		))
		return true
	}
	copy(out.Data, buf[:uint32(count)*uint32(b.outDensity)])
	outTime := n.OutputTime(0)
	if clockStamp {
		for i := range int(count) {
			setValueAt(*outTime, i, uint64(n.clock.Now()))
		}
	} else {
		copy(outTime.Data, inputTime.Data)
	}
	n.offsets[0] = int(count)
	return true
}

func (n *nodeImpl) batchEdgeFed(i int) bool {
	return n.ir.Inputs[i].Value == nil && !n.chanInputs[i] && !n.varInputs[i]
}
