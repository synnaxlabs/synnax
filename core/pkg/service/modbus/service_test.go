// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package modbus_test

import (
	"maps"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/modbus"
	v1 "github.com/synnaxlabs/synnax/pkg/service/modbus/versions/v1"
	"github.com/synnaxlabs/synnax/pkg/service/task/config"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
)

// register returns a holding register read channel carrying the given extra fields.
func register(fields map[string]any) map[string]any {
	ch := map[string]any{"type": "holding_register", "key": "chan-1", "address": 2}
	maps.Copy(ch, fields)
	return ch
}

// channel returns the only channel of a normalized or stored config.
func channel(cfg msgpack.EncodedJSON) map[string]any {
	GinkgoHelper()
	chs := cfg["channels"].([]any)
	Expect(chs).To(HaveLen(1))
	return chs[0].(map[string]any)
}

var _ = Describe("Service", func() {
	var svc *modbus.Service
	BeforeEach(func(ctx SpecContext) {
		svc = MustOpen(modbus.OpenService(ctx, modbus.ServiceConfig{
			DB: db,
		}))
	})

	Describe("OpenService", func() {
		It("Should reject a config missing the DB", func(ctx SpecContext) {
			Expect(modbus.OpenService(ctx, modbus.ServiceConfig{})).Error().
				To(MatchError(ContainSubstring("db: must be non-nil")))
		})
	})

	Describe("Stores", func() {
		It("Should expose one store per Modbus task type", func() {
			types := []string{}
			for _, s := range svc.Stores() {
				types = append(types, s.Type())
			}
			Expect(types).To(ConsistOf(
				"modbus_read",
				"modbus_write",
				"modbus_scan",
			))
		})
	})

	Describe("Write", func() {
		It("Should store a decoded read config under the given key", func(
			ctx SpecContext,
		) {
			key := uuid.New()
			Expect(svc.Read.Write(ctx, nil, key, msgpack.EncodedJSON{
				"device":      "dev-1",
				"sample_rate": 25,
				"channels": []any{map[string]any{
					"type":    "coil",
					"key":     "chan-1",
					"address": 3,
					"channel": 42,
				}},
			})).To(Succeed())
			data := MustSucceed(svc.Read.Read(ctx, nil, key))
			Expect(data["key"]).To(Equal(key.String()))
			Expect(data["device"]).To(Equal("dev-1"))
			Expect(data["sample_rate"]).To(BeNumerically("==", 25))
		})

		It("Should apply read config schema defaults to absent fields", func(
			ctx SpecContext,
		) {
			key := uuid.New()
			Expect(svc.Read.Write(ctx, nil, key, msgpack.EncodedJSON{})).To(Succeed())
			data := MustSucceed(svc.Read.Read(ctx, nil, key))
			Expect(data["sample_rate"]).To(BeNumerically("==", 10))
			Expect(data["stream_rate"]).To(BeNumerically("==", 5))
		})

		It("Should apply write channel schema defaults to absent fields", func(
			ctx SpecContext,
		) {
			key := uuid.New()
			Expect(svc.Write.Write(ctx, nil, key, msgpack.EncodedJSON{
				"channels": []any{map[string]any{
					"type": "holding_register",
					"key":  "chan-1",
				}},
			})).To(Succeed())
			data := MustSucceed(svc.Write.Read(ctx, nil, key))
			Expect(data["channels"]).To(HaveExactElements(
				HaveKeyWithValue("data_type", "uint8"),
			))
		})

		It("Should apply scan config schema defaults to absent fields", func(
			ctx SpecContext,
		) {
			key := uuid.New()
			Expect(svc.Scan.Write(ctx, nil, key, msgpack.EncodedJSON{})).To(Succeed())
			data := MustSucceed(svc.Scan.Read(ctx, nil, key))
			Expect(data["rate"]).To(BeNumerically("==", 0.2))
		})
	})

	Describe("Byte and word order", func() {
		It("Should report version 2 for read and write configs", func() {
			Expect(svc.Read.Version()).To(Equal(imex.Version(2)))
			Expect(svc.Write.Version()).To(Equal(imex.Version(2)))
		})

		It("Should store a channel with no override without swap fields", func(
			ctx SpecContext,
		) {
			key := uuid.New()
			Expect(svc.Read.Write(ctx, nil, key, msgpack.EncodedJSON{
				"channels": []any{register(nil)},
			})).To(Succeed())
			ch := channel(MustSucceed(svc.Read.Read(ctx, nil, key)))
			Expect(ch).ToNot(HaveKey("bytes_swapped"))
			Expect(ch).ToNot(HaveKey("words_swapped"))
		})

		It("Should keep an explicit override, false included", func(ctx SpecContext) {
			key := uuid.New()
			Expect(svc.Write.Write(ctx, nil, key, msgpack.EncodedJSON{
				"channels": []any{register(map[string]any{
					"bytes_swapped": false,
					"words_swapped": true,
				})},
			})).To(Succeed())
			ch := channel(MustSucceed(svc.Write.Read(ctx, nil, key)))
			Expect(ch).To(HaveKeyWithValue("bytes_swapped", false))
			Expect(ch).To(HaveKeyWithValue("words_swapped", true))
		})

		It("Should make a v1 false follow the device and keep a v1 true", func(
			ctx SpecContext,
		) {
			for _, store := range []config.Store{svc.Read, svc.Write} {
				ch := channel(MustSucceed(store.Normalize(ctx, 1, msgpack.EncodedJSON{
					"channels": []any{register(map[string]any{
						"bytes_swapped": false,
						"words_swapped": true,
					})},
				})))
				Expect(ch).ToNot(HaveKey("bytes_swapped"))
				Expect(ch).To(HaveKeyWithValue("words_swapped", true))
			}
		})

		It("Should make a released Console's swap_bytes false follow the device", func(
			ctx SpecContext,
		) {
			ch := channel(MustSucceed(svc.Read.Normalize(ctx, 0, msgpack.EncodedJSON{
				"channels": []any{map[string]any{
					"type":       "holding_register_input",
					"key":        "chan-1",
					"swap_bytes": false,
					"swap_words": true,
				}},
			})))
			Expect(ch).To(HaveKeyWithValue("type", "holding_register"))
			Expect(ch).ToNot(HaveKey("bytes_swapped"))
			Expect(ch).To(HaveKeyWithValue("words_swapped", true))
		})

		It("Should migrate a stored v1 record when the service opens", func(
			ctx SpecContext,
		) {
			stored := DeferClose(gorp.Wrap(memkv.New()))
			MustSucceed(gorp.OpenTable(
				ctx, gorp.TableConfig[uuid.UUID, v1.ReadConfig]{DB: stored},
			))
			key := uuid.New()
			old := v1.ReadConfig{Channels: []v1.ReadChannel{
				{Variant: v1.HoldingRegisterReadChannel{
					BaseReadChannel: v1.BaseReadChannel{Key: "a"},
					RegisterValue:   v1.RegisterValue{DataType: "float32"},
				}},
				{Variant: v1.InputRegisterReadChannel{
					BaseReadChannel: v1.BaseReadChannel{Key: "b"},
					RegisterValue: v1.RegisterValue{
						DataType:     "float32",
						BytesSwapped: true,
					},
				}},
			}}
			old.SetKey(key)
			Expect(gorp.NewCreate[uuid.UUID, v1.ReadConfig]().
				Entry(&old).Exec(ctx, stored)).To(Succeed())
			migrated := MustOpen(modbus.OpenService(
				ctx, modbus.ServiceConfig{DB: stored},
			))
			chs := MustSucceed(migrated.Read.Read(ctx, nil, key))["channels"].([]any)
			Expect(chs).To(HaveLen(2))
			Expect(chs[0]).ToNot(HaveKey("bytes_swapped"))
			Expect(chs[0]).ToNot(HaveKey("words_swapped"))
			Expect(chs[1]).To(HaveKeyWithValue("bytes_swapped", true))
			Expect(chs[1]).ToNot(HaveKey("words_swapped"))
		})
	})
})
