// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:build !windows

package timer

import "time"

// epoch is the zero of Now.
var epoch = time.Now()

// Now returns a reading of a monotonic clock. Only the span between two readings has
// meaning.
func Now() time.Duration { return time.Since(epoch) }
