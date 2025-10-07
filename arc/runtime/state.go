// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package runtime

import (
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
)

type channel struct {
	key      uint32
	dataType ir.Type
	buf      telem.MultiSeries
	hasData  bool
	deps     []string
}

type Stage interface{}

type node struct {
	key               string
	channelWaterMarks map[uint32]telem.Alignment
	firstActivated    bool
	requiredInputs    set.Set[uint32]
	state             map[string]any
	stage             Stage
	incoming          []ir.Edge
	outgoing          []ir.Edge
}
