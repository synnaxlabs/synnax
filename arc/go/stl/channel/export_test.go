// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel

import "github.com/synnaxlabs/x/telem"

func (cs *ProgramState) WriteValue(key uint32, value telem.Series) {
	cs.writeValue(key, value)
}

func (cs *ProgramState) ReadSeries(
	key uint32,
) (data telem.MultiSeries, time telem.MultiSeries, ok bool) {
	return cs.readSeries(key)
}

func (cs *ProgramState) WriteChannel(key uint32, data, time telem.Series) {
	cs.writeChannel(key, data, time)
}
