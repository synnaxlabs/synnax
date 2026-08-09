// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package legacy_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/task/common/legacy"
	"github.com/synnaxlabs/x/encoding/msgpack"
)

var _ = Describe("Rewrite", func() {
	DescribeTable("era normalization",
		func(r legacy.Rewrite, in, want msgpack.EncodedJSON) {
			Expect(r.Apply(in)).To(Equal(want))
		},
		Entry("flips data_saving",
			legacy.Rewrite{},
			msgpack.EncodedJSON{"data_saving": false},
			msgpack.EncodedJSON{"data_saving_disabled": true},
		),
		Entry("converts camelCase base keys",
			legacy.Rewrite{},
			msgpack.EncodedJSON{
				"autoStart":  true,
				"dataSaving": true,
				"sampleRate": float64(50),
				"streamRate": float64(25),
			},
			msgpack.EncodedJSON{
				"auto_start":           true,
				"data_saving_disabled": false,
				"sample_rate":          float64(50),
				"stream_rate":          float64(25),
			},
		),
		Entry("flips channel enabled",
			legacy.Rewrite{},
			msgpack.EncodedJSON{
				"channels": []any{map[string]any{"enabled": false}},
			},
			msgpack.EncodedJSON{
				"channels": []any{map[string]any{"disabled": true}},
			},
		),
		Entry("keeps a canonical config unchanged",
			legacy.Rewrite{},
			msgpack.EncodedJSON{
				"data_saving_disabled": true,
				"channels":             []any{map[string]any{"disabled": false}},
			},
			msgpack.EncodedJSON{
				"data_saving_disabled": true,
				"channels":             []any{map[string]any{"disabled": false}},
			},
		),
		Entry("converts camelCase channel keys",
			legacy.Rewrite{},
			msgpack.EncodedJSON{
				"channels": []any{map[string]any{
					"terminalA":  "PFI0",
					"terminalB":  "PFI1",
					"terminalZ":  "PFI2",
					"activeEdge": "Rising",
				}},
			},
			msgpack.EncodedJSON{
				"channels": []any{map[string]any{
					"terminal_a":  "PFI0",
					"terminal_b":  "PFI1",
					"terminal_z":  "PFI2",
					"active_edge": "Rising",
				}},
			},
		),
		Entry("keeps a config-level device",
			legacy.Rewrite{},
			msgpack.EncodedJSON{
				"device":   "dev-1",
				"channels": []any{map[string]any{"port": float64(0)}},
			},
			msgpack.EncodedJSON{
				"device":   "dev-1",
				"channels": []any{map[string]any{"port": float64(0)}},
			},
		),
		Entry("keeps a channel key named channel",
			legacy.Rewrite{},
			msgpack.EncodedJSON{
				"channels": []any{map[string]any{"channel": float64(7)}},
			},
			msgpack.EncodedJSON{
				"channels": []any{map[string]any{"channel": float64(7)}},
			},
		),
	)

	DescribeTable("Scan",
		func(in, want msgpack.EncodedJSON) {
			Expect(legacy.Scan.Apply(in)).To(Equal(want))
		},
		Entry("renames scan_rate and flips enabled",
			msgpack.EncodedJSON{"scan_rate": float64(0.5), "enabled": false},
			msgpack.EncodedJSON{"rate": float64(0.5), "disabled": true},
		),
		Entry("keeps other scan fields",
			msgpack.EncodedJSON{"enabled": true, "tcp_scan_multiplier": float64(5)},
			msgpack.EncodedJSON{"disabled": false, "tcp_scan_multiplier": float64(5)},
		),
		Entry("renames scan_rate alone",
			msgpack.EncodedJSON{"scan_rate": float64(1)},
			msgpack.EncodedJSON{"rate": float64(1)},
		),
	)

	It("Should run Pre on the original keys and Post on the converted ones", func() {
		var preKeys, postKeys []string
		r := legacy.Rewrite{
			Pre: func(config msgpack.EncodedJSON) {
				for k := range config {
					preKeys = append(preKeys, k)
				}
			},
			Post: func(config msgpack.EncodedJSON) {
				for k := range config {
					postKeys = append(postKeys, k)
				}
			},
		}
		r.Apply(msgpack.EncodedJSON{"sampleRate": float64(1)})
		Expect(preKeys).To(ConsistOf("sampleRate"))
		Expect(postKeys).To(ConsistOf("sample_rate"))
	})

	It("Should not mutate the input config", func() {
		in := msgpack.EncodedJSON{"data_saving": true}
		legacy.Rewrite{}.Apply(in)
		Expect(in).To(HaveKeyWithValue("data_saving", true))
	})
})

var _ = Describe("Helpers", func() {
	Describe("RenameKey", func() {
		It("Should move the value under the new key", func() {
			m := msgpack.EncodedJSON{"old": 1}
			legacy.RenameKey(m, "old", "new")
			Expect(m).To(Equal(msgpack.EncodedJSON{"new": 1}))
		})
		It("Should keep an existing value under the new key", func() {
			m := msgpack.EncodedJSON{"old": 1, "new": 2}
			legacy.RenameKey(m, "old", "new")
			Expect(m).To(Equal(msgpack.EncodedJSON{"new": 2}))
		})
		It("Should do nothing when the old key is absent", func() {
			m := msgpack.EncodedJSON{"other": 1}
			legacy.RenameKey(m, "old", "new")
			Expect(m).To(Equal(msgpack.EncodedJSON{"other": 1}))
		})
	})

	Describe("FlipBool", func() {
		It("Should negate the value under the new key", func() {
			m := msgpack.EncodedJSON{"enabled": true}
			legacy.FlipBool(m, "enabled", "disabled")
			Expect(m).To(Equal(msgpack.EncodedJSON{"disabled": false}))
		})
		It("Should drop a non-bool value", func() {
			m := msgpack.EncodedJSON{"enabled": "yes"}
			legacy.FlipBool(m, "enabled", "disabled")
			Expect(m).To(BeEmpty())
		})
		It("Should keep an existing value under the new key", func() {
			m := msgpack.EncodedJSON{"enabled": true, "disabled": true}
			legacy.FlipBool(m, "enabled", "disabled")
			Expect(m).To(Equal(msgpack.EncodedJSON{"disabled": true}))
		})
	})

	Describe("EachChild", func() {
		It("Should apply the function to every object element", func() {
			m := msgpack.EncodedJSON{"channels": []any{
				map[string]any{"a": 1},
				"not-an-object",
				map[string]any{"a": 2},
			}}
			var seen []any
			legacy.EachChild(m, "channels", func(ch msgpack.EncodedJSON) {
				seen = append(seen, ch["a"])
			})
			Expect(seen).To(Equal([]any{1, 2}))
		})
		It("Should do nothing when the key does not hold a list", func() {
			m := msgpack.EncodedJSON{"channels": "nope"}
			legacy.EachChild(m, "channels", func(msgpack.EncodedJSON) {
				Fail("should not be called")
			})
		})
	})

	Describe("RecordToList", func() {
		It("Should convert entries in sorted key order", func() {
			m := msgpack.EncodedJSON{
				"headers": map[string]any{"b": 2, "a": 1},
			}
			legacy.RecordToList(m, "headers", "name", "value")
			Expect(m["headers"]).To(Equal([]any{
				map[string]any{"name": "a", "value": 1},
				map[string]any{"name": "b", "value": 2},
			}))
		})
		It("Should leave a non-record value alone", func() {
			m := msgpack.EncodedJSON{"headers": []any{"already-a-list"}}
			legacy.RecordToList(m, "headers", "name", "value")
			Expect(m["headers"]).To(Equal([]any{"already-a-list"}))
		})
	})
})
