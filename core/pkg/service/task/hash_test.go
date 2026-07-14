// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package task_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
)

// The expected hashes are shared golden vectors: the C++ (x/cpp/hash) and
// TypeScript (client/ts task) tests assert the same inputs produce the same
// values.
var _ = Describe("HashConfig", func() {
	DescribeTable("golden vectors",
		func(config msgpack.EncodedJSON, expected string) {
			Expect(task.HashConfig(config)).To(Equal(expected))
		},
		Entry("nil config", nil, "2e1472b57af294d1"),
		Entry("empty config", msgpack.EncodedJSON{}, "2e1472b57af294d1"),
		Entry(
			"flat config",
			msgpack.EncodedJSON{"rate": 50.0, "port": 8080, "host": "localhost"},
			"2de66015b3bdded8",
		),
		Entry(
			"nested config",
			msgpack.EncodedJSON{
				"enabled": true,
				"channels": []any{
					map[string]any{"key": 12, "name": `ch"1"`, "scale": 0.001},
				},
				"notes": "héllo⚡",
			},
			"811ef1fc462a59f2",
		),
	)

	It("should produce the same hash for integer and integral float values", func() {
		Expect(task.HashConfig(msgpack.EncodedJSON{"rate": 50})).
			To(Equal(task.HashConfig(msgpack.EncodedJSON{"rate": 50.0})))
	})

	It("should produce different hashes for different configs", func() {
		Expect(task.HashConfig(msgpack.EncodedJSON{"rate": 50})).
			ToNot(Equal(task.HashConfig(msgpack.EncodedJSON{"rate": 51})))
	})
})
