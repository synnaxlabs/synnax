// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package telem

import latest "github.com/synnaxlabs/x/telem/types/v0"

// NumericSample represents any numeric value that can be stored in a Series and have
// mathematical operations performed on it.
type NumericSample = latest.NumericSample

// FixedSample represents any numeric value that can be stored in a Series and has a
// fixed density.
type FixedSample = latest.FixedSample

// VariableSample is a type that can be stored in a variable-density series.
type VariableSample = latest.VariableSample

// Sample represents any value that can be stored in a non-JSON Series.
type Sample = latest.Sample

// ByteOrder is the standard order for encoding/decoding numeric values across the
// Synnax telemetry ecosystem.
var ByteOrder = latest.ByteOrder
