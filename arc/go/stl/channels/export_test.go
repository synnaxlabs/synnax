// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channels

import "github.com/synnaxlabs/x/telem"

func (ps *ProgramState) WriteValue(key uint32, value telem.Series) {
	ps.writeValue(key, value)
}

func (ps *ProgramState) ReadSeries(
	key uint32,
) (data, time telem.MultiSeries, ok bool) {
	return ps.readSeries(key)
}

func (ps *ProgramState) WriteChannel(key uint32, data, time telem.Series) {
	ps.writeChannel(key, data, time)
}

func (ps *ProgramState) ReadValue(key uint32) (telem.Series, bool) {
	return ps.readValue(key)
}

func WriteSample[T telem.FixedSample](ps *ProgramState, key uint32, v T) {
	writeSample(ps, key, v)
}
