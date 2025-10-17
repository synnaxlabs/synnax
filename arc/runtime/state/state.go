// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package state

import (
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/x/telem"
)

type value struct {
	data telem.Series
	time telem.Series
}

type State struct {
	cfg     Config
	outputs map[ir.Handle]value
	indexes map[uint32]uint32
	channel struct {
		reads, writes map[uint32]telem.Series
	}
}

type ChannelDigest struct {
	Key      uint32
	DataType telem.DataType
	Index    uint32
}

type Config struct {
	ChannelDigests []ChannelDigest
	Edges          ir.Edges
	ReactiveDeps   map[uint32][]string
}

func New(cfg Config) *State {
	s := &State{
		cfg:     cfg,
		outputs: make(map[ir.Handle]value),
		indexes: make(map[uint32]uint32),
	}
	s.channel.reads = make(map[uint32]telem.Series)
	s.channel.writes = make(map[uint32]telem.Series)
	for _, d := range cfg.ChannelDigests {
		s.indexes[d.Key] = d.Index
	}
	return s
}

func (s *State) Ingest(fr telem.Frame[uint32], markDirty func(nodeKey string)) {
	for rawI, key := range fr.RawKeys() {
		if fr.ShouldExcludeRaw(rawI) {
			continue
		}
		s.channel.reads[key] = fr.RawSeriesAt(rawI)
		for _, nodeKey := range s.cfg.ReactiveDeps[key] {
			markDirty(nodeKey)
		}
		if indexKey, ok := s.indexes[key]; ok {
			for _, nodeKey := range s.cfg.ReactiveDeps[indexKey] {
				markDirty(nodeKey)
			}
		}
	}
}

func (s *State) FlushWrites(fr telem.Frame[uint32]) telem.Frame[uint32] {
	for key, data := range s.channel.writes {
		fr = fr.Append(key, data)
	}
	clear(s.channel.writes)
	return fr
}

func (s *State) ReadChannel(key uint32) (telem.MultiSeries, bool) {
	series, ok := s.channel.reads[key]
	if !ok {
		return telem.MultiSeries{}, false
	}
	return telem.MultiSeries{Series: []telem.Series{series}}, true
}

func (s *State) WriteChannel(key uint32, data, time telem.Series) {
	s.channel.writes[key] = data
}

func (s *State) GetChannelIndexKey(key uint32) uint32 {
	return s.indexes[key]
}

func (s *State) ReadEdge(handle ir.Handle) (value, bool) {
	v, ok := s.outputs[handle]
	return v, ok
}

func (s *State) Node(key string) *Node {
	return &Node{
		inputs:  s.cfg.Edges.GetInputs(key),
		outputs: s.cfg.Edges.GetOutputs(key),
		state:   s,
	}
}

type inputEntry struct {
	data      telem.MultiSeries
	time      telem.MultiSeries
	watermark telem.TimeStamp
}

type Node struct {
	inputs, outputs []ir.Edge
	state           *State
	accumulated     []inputEntry
	alignedData     []telem.Series
	alignedTime     []telem.Series
}

func (n *Node) RefreshInputs() (recalculate bool) {
	for i, edge := range n.inputs {
		sourceOutput, exists := n.state.outputs[edge.Source]
		if !exists || sourceOutput.data.Len() == 0 || sourceOutput.time.Len() == 0 {
			continue
		}
		lastTimestamp := telem.ValueAt[telem.TimeStamp](sourceOutput.time, -1)
		if lastTimestamp <= n.accumulated[i].watermark {
			continue
		}
		n.accumulated[i].data.Series = append(n.accumulated[i].data.Series, sourceOutput.data)
		n.accumulated[i].time.Series = append(n.accumulated[i].time.Series, sourceOutput.time)
	}
	for i := range n.inputs {
		if len(n.accumulated[i].data.Series) == 0 {
			return false
		}
	}
	var (
		triggerInputIdx  int = -1
		triggerTimestamp telem.TimeStamp
		triggerSeriesIdx int
	)
	for i := range n.inputs {
		for j, timeSeries := range n.accumulated[i].time.Series {
			if timeSeries.Len() == 0 {
				continue
			}
			ts := telem.ValueAt[telem.TimeStamp](timeSeries, -1)
			if ts > n.accumulated[i].watermark {
				if triggerInputIdx == -1 || ts < triggerTimestamp {
					triggerInputIdx = i
					triggerTimestamp = ts
					triggerSeriesIdx = j
				}
			}
		}
	}
	if triggerInputIdx == -1 {
		return false
	}
	for i := range n.inputs {
		if i == triggerInputIdx {
			n.alignedData[i] = n.accumulated[i].data.Series[triggerSeriesIdx]
			n.alignedTime[i] = n.accumulated[i].time.Series[triggerSeriesIdx]
		} else {
			latestIdx := len(n.accumulated[i].data.Series) - 1
			n.alignedData[i] = n.accumulated[i].data.Series[latestIdx]
			n.alignedTime[i] = n.accumulated[i].time.Series[latestIdx]
		}
	}
	n.accumulated[triggerInputIdx].watermark = triggerTimestamp
	for i := range n.inputs {
		var (
			newData []telem.Series
			newTime []telem.Series
		)
		for j, timeSeries := range n.accumulated[i].time.Series {
			if timeSeries.Len() == 0 {
				continue
			}
			ts := telem.ValueAt[telem.TimeStamp](timeSeries, -1)
			if ts > n.accumulated[i].watermark {
				newData = append(newData, n.accumulated[i].data.Series[j])
				newTime = append(newTime, timeSeries)
			}
		}
		if len(newData) == 0 && len(n.accumulated[i].data.Series) > 0 {
			lastIdx := len(n.accumulated[i].data.Series) - 1
			newData = []telem.Series{n.accumulated[i].data.Series[lastIdx]}
			newTime = []telem.Series{n.accumulated[i].time.Series[lastIdx]}
		}
		n.accumulated[i].data.Series = newData
		n.accumulated[i].time.Series = newTime
	}
	return true
}

func (n *Node) output(paramIndex int) value {
	return n.state.outputs[n.outputs[paramIndex].Source]
}

func (n *Node) InputTime(paramIndex int) telem.Series {
	return n.alignedTime[paramIndex]
}

func (n *Node) InputData(paramIndex int) telem.Series {
	return n.alignedData[paramIndex]
}

func (n *Node) OutputData(paramIndex int) *telem.Series {
	d := n.output(paramIndex)
	return &d.data
}

func (n *Node) OutputTime(paramIndex int) *telem.Series {
	d := n.output(paramIndex)
	return &d.time
}

func (n *Node) ReadChan(key uint32) (data telem.MultiSeries, time telem.MultiSeries, ok bool) {
	data, ok = n.state.ReadChannel(key)
	if !ok {
		return telem.MultiSeries{}, telem.MultiSeries{}, false
	}
	indexKey := n.state.indexes[key]
	if indexKey == 0 {
		return data, telem.MultiSeries{}, true
	}
	time, _ = n.state.ReadChannel(indexKey)
	return data, time, true
}

func (n *Node) WriteChan(key uint32, value, time telem.Series) {
	n.state.channel.writes[key] = value
	n.state.channel.writes[n.state.indexes[key]] = time
}
