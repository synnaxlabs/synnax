// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cesium

// WithPieceSize returns cfg with the most samples one piece of a fixed-span Next or
// Prev holds, summed over the channels.
func (cfg IteratorConfig) WithPieceSize(n int64) IteratorConfig {
	cfg.pieceSize = n
	return cfg
}
