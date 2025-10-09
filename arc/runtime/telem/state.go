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

type State struct {
	Data map[uint32]telem.MultiSeries
	Deps map[uint32][]string
}

func (s *State) register(channel uint32, nodeKey string) {
	s.Deps[channel] = append(s.Deps[channel], nodeKey)
}

func (s *State) Ingest(
	fr telem.Frame[uint32],
	markDirty func(nodeKey string),
) {
	for rawI, key := range fr.RawKeys() {
		if fr.ShouldExcludeRaw(rawI) {
			continue
		}
		s.Data[key] = s.Data[key].Append(fr.RawSeriesAt(rawI))
		for _, dep := range s.Deps[key] {
			markDirty(dep)
		}
	}
}
