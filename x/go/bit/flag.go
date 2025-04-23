// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package bit

import "github.com/synnaxlabs/x/types"

type FlagPos int

func (f FlagPos) Get(b byte) bool {
	return ((b >> f) & 1) == 1
}

func (f FlagPos) Set(b byte, value bool) byte {
	v := types.BoolToUint8(value) << f
	return b | v
}
