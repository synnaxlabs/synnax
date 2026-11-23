// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc

import (
	"fmt"
	"math"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/synnaxlabs/x/zyn"
)

type (
	breaker3Int   int64
	breaker3Uint  uint32
	breaker3Float float64
)

var _ = ginkgo.Describe("channelKey breaker 3", func() {
	ginkgo.It(
		"Should read a numeric node input as the graph compiler's coerce does",
		func() {
			one := 1
			f := 42.0
			corpus := []any{
				0,
				1,
				-1,
				42,
				math.MaxUint32,
				math.MaxUint32 + 1,
				math.MinInt64,
				int8(-1),
				int8(7),
				int16(300),
				int32(-5),
				int32(1048577),
				int64(
					0,
				),
				int64(1048577),
				int64(math.MaxUint32),
				int64(math.MaxUint32 + 1),
				int64(math.MaxInt64),
				int64(math.MinInt64),
				uint(0),
				uint(9),
				uint8(255),
				uint16(65535),
				uint32(0),
				uint32(1),
				uint32(
					math.MaxUint32,
				),
				uint64(math.MaxUint32),
				uint64(math.MaxUint32 + 1),
				uint64(math.MaxUint64),
				uint64(1 << 63),
				float32(0),
				float32(math.Copysign(0, -1)),
				float32(1),
				float32(1.5),
				float32(4294967040),
				float32(4294967296),
				float32(16777217),
				float32(math.NaN()),
				float32(math.Inf(1)),
				float32(math.Inf(-1)),
				float32(math.SmallestNonzeroFloat32),
				float32(-1),
				0.0,
				math.Copysign(0, -1),
				1.0,
				1048577.0,
				1048577.5,
				1048577.0000001,
				float64(math.MaxUint32),
				math.MaxUint32 + 0.5,
				math.MaxUint32 + 1.0,
				math.Nextafter(math.MaxUint32, math.Inf(1)),
				1e300,
				-1e300,
				-1.0,
				math.NaN(),
				math.Inf(1),
				math.Inf(-1),
				math.SmallestNonzeroFloat64,
				math.MaxFloat64,
				float64(1 << 53),
				float64(1<<53 + 2),
				breaker3Int(42),
				breaker3Int(-42),
				breaker3Uint(42),
				breaker3Float(42),
				breaker3Float(42.5),
				true,
				false,
				nil,
				&one,
				&f,
				[]any{42},
				map[string]any{"key": 42},
				complex(42, 0),
				struct{}{},
			}
			var mismatches []string
			for _, v := range corpus {
				var k uint32
				zerr := zyn.Uint32().Coerce().Parse(v, &k)
				oldOK := zerr == nil && k != 0
				got, ok := channelKey(v)
				if ok != oldOK || (ok && uint32(got) != k) {
					mismatches = append(mismatches, fmt.Sprintf(
						"%T(%v): zyn (%d, %v), channelKey (%d, %v)",
						v, v, k, oldOK, got, ok,
					))
				}
			}
			gomega.Expect(mismatches).To(gomega.BeEmpty())
		},
	)
})
