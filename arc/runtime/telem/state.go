// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package telem

import "github.com/synnaxlabs/x/telem"

type Data struct {
	telem.MultiSeries
	IndexKey uint32
}

type State struct {
	Data    map[uint32]Data
	Writes  map[uint32]telem.Series
	Readers map[uint32][]string
	Writers map[uint32][]string
}

func NewState() *State {
	return &State{
		Data:    make(map[uint32]Data),
		Writes:  make(map[uint32]telem.Series),
		Readers: make(map[uint32][]string),
		Writers: make(map[uint32][]string),
	}
}

func (s *State) registerReader(channel uint32, nodeKey string) {
	s.Readers[channel] = append(s.Readers[channel], nodeKey)
}

func (s *State) registerWriter(channel uint32, nodeKey string) {
	s.Writers[channel] = append(s.Writers[channel], nodeKey)
}

func (s *State) Ingest(
	fr telem.Frame[uint32],
	markDirty func(nodeKey string),
) {
	for rawI, key := range fr.RawKeys() {
		if fr.ShouldExcludeRaw(rawI) {
			continue
		}
		e := s.Data[key]
		e.MultiSeries = e.Append(fr.RawSeriesAt(rawI))
		s.Data[key] = e
		for _, dep := range s.Readers[key] {
			markDirty(dep)
		}
		for k, n := range s.Readers {
			if s.Data[k].IndexKey == key {
				for _, v := range n {
					markDirty(v)
				}
			}
		}
	}
}
