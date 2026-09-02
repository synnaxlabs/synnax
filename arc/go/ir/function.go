// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ir

// DispatcherSyntheticPrefix marks synthesized dispatchers; Body.Raw lists branch keys.
const DispatcherSyntheticPrefix = "disp$"

// BatchSuffix names the vectorized companion the compiler exports beside every
// element-wise function. The wrapper loops over a whole series inside the guest, so a
// runtime crosses the host-guest boundary once per series instead of once per sample.
const BatchSuffix = "$batch"

// Parameter slots of a BatchSuffix wrapper: the sample count, the output block's base
// pointer, then one (base pointer, byte stride) pair per input. A stride of zero
// repeats a single sample across the whole series.
const (
	BatchCountParam = 0
	BatchOutParam   = 1
	BatchInputParam = 2
)
