// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package telem_test

import (
	"testing"
	"time"

	"github.com/synnaxlabs/x/telem"
)

// BenchmarkNow compares telem.Now with time.Now, its source on every platform except
// Windows.
func BenchmarkNow(b *testing.B) {
	b.Run("telem", func(b *testing.B) {
		for b.Loop() {
			telem.Now()
		}
	})
	b.Run("time", func(b *testing.B) {
		for b.Loop() {
			time.Now()
		}
	})
}
