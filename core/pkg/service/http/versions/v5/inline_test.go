// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v5_test

import (
	"context"
	"math"
	"strconv"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	channel "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v0"
	v4 "github.com/synnaxlabs/synnax/pkg/service/http/versions/v4"
	v5 "github.com/synnaxlabs/synnax/pkg/service/http/versions/v5"
	"github.com/synnaxlabs/synnax/pkg/service/icd/read"
	icdv0 "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v0"
	icdv1 "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v1"
	payload "github.com/synnaxlabs/x/payload/versions/v0"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// channelsOf returns the lookup of channels in which found holds every channel.
func channelsOf(found map[channel.Key]icdv0.Channel) icdv0.Channels {
	return func(
		context.Context,
		gorp.Snapshot,
		[]channel.Key,
	) (map[channel.Key]icdv0.Channel, error) {
		return found, nil
	}
}

// resolve resolves cfg with found as the channels that exist. It returns the ICD, the
// resolved config, and each path that a note named.
func resolve(
	ctx context.Context,
	cfg v5.ReadConfig,
	found map[channel.Key]icdv0.Channel,
) (icdv1.ICD, v5.ReadConfig, []string) {
	GinkgoHelper()
	var notes []string
	ic, ok := MustSucceed2(v5.ResolveReadConfig(
		ctx, nil, channelsOf(found), "Plant", &cfg,
		func(path, _ string) { notes = append(notes, path) },
	))
	Expect(ok).To(BeTrue())
	Expect(cfg.Inline).To(BeNil())
	return ic, cfg, notes
}

// staged returns a v5 config whose endpoints read fields, with the fields staged.
func staged(ctx context.Context, endpoints ...v4.ReadEndpoint) v5.ReadConfig {
	GinkgoHelper()
	return MustSucceed(v5.MigrateReadConfig(ctx, v4.ReadConfig{Endpoints: endpoints}))
}

// jsonFields returns the fields of the first message of ic, which must be JSON.
func jsonFields(ic icdv1.ICD) []icdv1.JSONField {
	GinkgoHelper()
	return as[icdv1.JSONPayload](ic.Messages[0].Payload.Variant).Fields
}

// as returns v as a T, and fails the spec when v is not a T.
func as[T any](v any) T {
	GinkgoHelper()
	t, ok := v.(T)
	Expect(ok).To(BeTrue())
	return t
}

// check runs each read of cfg through the read check, against ic and found.
func check(
	ctx context.Context,
	ic icdv1.ICD,
	cfg v5.ReadConfig,
	found map[channel.Key]icdv0.Channel,
) error {
	GinkgoHelper()
	// The Checker gets ic, so the Resolver retrieves no ICD.
	resolver := read.NewResolver(nil, channelsOf(found))
	var reads []read.Read
	for i, ep := range cfg.Endpoints {
		if ep.ICD != nil {
			reads = append(reads, read.Read{
				Path:     []string{"endpoints", strconv.Itoa(i)},
				ICD:      *ep.ICD,
				Messages: ep.Messages,
			})
		}
	}
	c := MustSucceed(resolver.NewChecker(
		ctx, nil, reads, map[icdv1.Key]icdv1.ICD{ic.Key: ic},
	))
	for _, rd := range reads {
		if err := c.Read(rd); err != nil {
			return err
		}
	}
	return nil
}

var _ = Describe("ResolveReadConfig", func() {
	It("Should return false for a config with no inline fields", func(ctx SpecContext) {
		cfg := v5.ReadConfig{}
		_, ok := MustSucceed2(v5.ResolveReadConfig(ctx, nil, nil, "Plant", &cfg, nil))
		Expect(ok).To(BeFalse())
	})

	It("Should name each message after the method and path of its endpoint",
		func(ctx SpecContext) {
			ic, cfg, _ := resolve(ctx, staged(
				ctx,
				v4.ReadEndpoint{Key: "a", Method: "GET", Path: "/a"},
				v4.ReadEndpoint{Key: "b", Method: "GET", Path: "/a"},
				v4.ReadEndpoint{Key: "c", Method: "POST", Path: "/c"},
			), nil)
			Expect(ic.Name).To(Equal("Plant"))
			names := make([]string, len(ic.Messages))
			for i, m := range ic.Messages {
				names[i] = m.Name
				Expect(*cfg.Endpoints[i].ICD).To(Equal(ic.Key))
				Expect(cfg.Endpoints[i].Messages).To(HaveLen(1))
				Expect(cfg.Endpoints[i].Messages[0].Message).To(Equal(m.Key))
			}
			Expect(names).To(Equal([]string{"GET /a", "GET /a (2)", "POST /c"}))
		},
	)

	It("Should bind each field to its channel and keep the fields of the endpoint",
		func(ctx SpecContext) {
			ic, cfg, _ := resolve(ctx, staged(ctx, v4.ReadEndpoint{
				Key:   "ep",
				Path:  "/values",
				Index: "t",
				Fields: []v4.ReadField{
					{Key: "t", Channel: 6, Pointer: "/t", DataType: telem.TimestampT},
					{Key: "p", Channel: 7, Pointer: "/p", DataType: telem.Float32T},
					{Key: "off", Channel: 8, Pointer: "/off", Disabled: true},
				},
			}), map[channel.Key]icdv0.Channel{
				6: {DataType: telem.TimestampT, Index: true},
				7: {DataType: telem.Float32T},
			})
			fields := jsonFields(ic)
			Expect(fields).To(HaveLen(3))
			read := cfg.Endpoints[0].Messages[0]
			Expect(*read.Index).To(Equal(fields[0].Base().Key))
			Expect(read.Bindings).To(Equal([]icdv0.ReadBinding{
				{Field: fields[0].Base().Key, Channel: 6},
				{Field: fields[1].Base().Key, Channel: 7},
			}))
			Expect(cfg.Endpoints[0].Fields).To(HaveLen(3))
		},
	)

	DescribeTable("Should turn a fill value into the default of its field",
		func(
			ctx SpecContext,
			f v4.ReadField,
			fallback func(icdv1.JSONFieldVariant) any,
			expected any,
		) {
			f.Key, f.Pointer, f.Optional = "f", "/f", true
			ic, _, notes := resolve(ctx, staged(ctx, v4.ReadEndpoint{
				Key:    "ep",
				Fields: []v4.ReadField{f},
			}), nil)
			Expect(notes).To(BeEmpty())
			Expect(fallback(jsonFields(ic)[0].Variant)).To(Equal(expected))
		},
		Entry(
			"a float",
			v4.ReadField{DataType: telem.Float64T, FillValue: new(any(2.5))},
			func(v icdv1.JSONFieldVariant) any {
				return *as[icdv1.FloatJSONField](v).Default
			},
			payload.Float(2.5),
		),
		Entry(
			"an integer",
			v4.ReadField{DataType: telem.Int32T, FillValue: new(any(-4))},
			func(v icdv1.JSONFieldVariant) any {
				return *as[icdv1.IntegerJSONField](v).Default
			},
			payload.Integer(-4),
		),
		Entry(
			"the label of an enum",
			v4.ReadField{
				DataType:   telem.Uint8T,
				EnumValues: []payload.EnumEntry{{Label: "running", Value: 2}},
				FillValue:  new(any("running")),
			},
			func(v icdv1.JSONFieldVariant) any {
				return *as[icdv1.UnsignedJSONField](v).Default
			},
			payload.Unsigned(2),
		),
		Entry(
			"a boolean",
			v4.ReadField{DataType: telem.BooleanT, FillValue: new(any(true))},
			func(v icdv1.JSONFieldVariant) any {
				return *as[icdv1.BooleanJSONField](v).Default
			},
			true,
		),
		Entry(
			"a JSON value",
			v4.ReadField{
				DataType:  telem.JSONT,
				FillValue: new(any(map[string]any{"a": 1})),
			},
			func(v icdv1.JSONFieldVariant) any {
				return *as[icdv1.JSONJSONField](v).Default
			},
			`{"a":1}`,
		),
		Entry(
			"a string",
			v4.ReadField{DataType: telem.StringT, FillValue: new(any("idle"))},
			func(v icdv1.JSONFieldVariant) any {
				return *as[icdv1.StringJSONField](v).Default
			},
			"idle",
		),
		Entry(
			"a timestamp in the time format of its field",
			v4.ReadField{
				DataType:   telem.TimestampT,
				TimeFormat: new(payload.TimeFormatUnixSec),
				FillValue:  new(any(3)),
			},
			func(v icdv1.JSONFieldVariant) any {
				return as[icdv1.TimestampJSONField](v).Default.Variant
			},
			icdv1.ConstantTimestampDefault{Value: telem.TimeStamp(3 * telem.Second)},
		),
		Entry(
			"a UUID",
			v4.ReadField{
				DataType:  telem.UUIDT,
				FillValue: new(any("f81d4fae-7dec-11d0-a765-00a0c91e6bf6")),
			},
			func(v icdv1.JSONFieldVariant) any {
				return as[icdv1.UuidJSONField](v).Default.Variant
			},
			icdv1.ConstantUUIDDefault{
				Value: uuid.MustParse("f81d4fae-7dec-11d0-a765-00a0c91e6bf6"),
			},
		),
		Entry(
			"bytes",
			v4.ReadField{DataType: telem.BytesT, FillValue: new(any("AQI="))},
			func(v icdv1.JSONFieldVariant) any {
				return *as[icdv1.BytesJSONField](v).Default
			},
			"AQI=",
		),
	)

	It("Should keep a fill value when the lift drops an enum entry", func(
		ctx SpecContext,
	) {
		ic, _, _ := resolve(ctx, staged(ctx, v4.ReadEndpoint{
			Key: "ep",
			Fields: []v4.ReadField{{
				Key:      "f",
				Pointer:  "/f",
				Optional: true,
				DataType: telem.Int32T,
				EnumValues: []payload.EnumEntry{
					{Label: "half", Value: 0.5},
					{Label: "on", Value: 1},
				},
				FillValue: new(any("on")),
			}},
		}), nil)
		f := as[icdv1.IntegerJSONField](jsonFields(ic)[0].Variant)
		Expect(f.EnumValues).To(HaveLen(1))
		Expect(f.Default).To(HaveValue(Equal(payload.Integer(1))))
	})

	It("Should drop a fill value that does not convert", func(ctx SpecContext) {
		ic, _, notes := resolve(ctx, staged(ctx, v4.ReadEndpoint{
			Key: "ep",
			Fields: []v4.ReadField{{
				Key:       "f",
				Pointer:   "/f",
				DataType:  telem.Float64T,
				FillValue: new(any("high")),
			}},
		}), nil)
		Expect(notes).To(Equal([]string{"endpoints.0.fields.0"}))
		Expect(as[icdv1.FloatJSONField](jsonFields(ic)[0].Variant).Default).
			To(BeNil())
	})

	It("Should give an optional float field with no fill value the default NaN",
		func(ctx SpecContext) {
			ic, _, notes := resolve(ctx, staged(ctx, v4.ReadEndpoint{
				Key: "ep",
				Fields: []v4.ReadField{
					{Key: "o", Pointer: "/o", Optional: true, DataType: telem.Float32T},
					{Key: "r", Pointer: "/r", DataType: telem.Float64T},
				},
			}), nil)
			Expect(notes).To(BeEmpty())
			fields := jsonFields(ic)
			d := as[icdv1.FloatJSONField](fields[0].Variant).Default
			Expect(d).ToNot(BeNil())
			Expect(math.IsNaN(float64(*d))).To(BeTrue())
			Expect(as[icdv1.FloatJSONField](fields[1].Variant).Default).To(BeNil())
		},
	)

	It("Should note an optional field with no fill value that is not a float",
		func(ctx SpecContext) {
			ic, _, notes := resolve(ctx, staged(ctx, v4.ReadEndpoint{
				Key: "ep",
				Fields: []v4.ReadField{
					{Key: "o", Pointer: "/o", Optional: true, DataType: telem.Int32T},
				},
			}), nil)
			Expect(notes).To(Equal([]string{"endpoints.0.fields.0"}))
			Expect(as[icdv1.IntegerJSONField](jsonFields(ic)[0].Variant).Default).
				To(BeNil())
		},
	)

	Describe("Read check", func() {
		found := map[channel.Key]icdv0.Channel{
			6:  {Name: "time", DataType: telem.TimestampT, Index: true},
			7:  {Name: "pressure", DataType: telem.Float32T},
			9:  {Name: "time_b", DataType: telem.TimestampT, Index: true},
			10: {Name: "stamp", DataType: telem.TimestampT},
		}
		timeField := func(key string, ch channel.Key) v4.ReadField {
			return v4.ReadField{
				Key:        key,
				Channel:    ch,
				Pointer:    "/" + key,
				DataType:   telem.TimestampT,
				TimeFormat: new(payload.TimeFormatUnixSec),
			}
		}
		pressure := func(key string) v4.ReadField {
			return v4.ReadField{
				Key:      key,
				Channel:  7,
				Pointer:  "/" + key,
				DataType: telem.Float32T,
			}
		}
		// lift resolves endpoints and expects each read to pass the read check.
		lift := func(
			ctx context.Context,
			endpoints ...v4.ReadEndpoint,
		) (icdv1.ICD, v5.ReadConfig, []string) {
			GinkgoHelper()
			ic, cfg, notes := resolve(ctx, staged(ctx, endpoints...), found)
			Expect(check(ctx, ic, cfg, found)).To(Succeed())
			return ic, cfg, notes
		}

		It("Should drop an index that names a disabled field", func(ctx SpecContext) {
			t := timeField("t", 6)
			t.Disabled = true
			_, cfg, notes := lift(ctx, v4.ReadEndpoint{
				Key:    "ep",
				Index:  "t",
				Fields: []v4.ReadField{t, pressure("p")},
			})
			Expect(cfg.Endpoints[0].Messages[0].Index).To(BeNil())
			Expect(notes).To(ContainElement("endpoints.0.index"))
		})

		It("Should drop an index whose field writes no index channel",
			func(ctx SpecContext) {
				_, cfg, notes := lift(ctx, v4.ReadEndpoint{
					Key:    "ep",
					Index:  "p",
					Fields: []v4.ReadField{pressure("p")},
				})
				Expect(cfg.Endpoints[0].Messages[0].Index).To(BeNil())
				Expect(notes).To(ContainElement("endpoints.0.index"))
			},
		)

		It("Should take the index from the field that writes an index channel",
			func(ctx SpecContext) {
				ic, cfg, notes := lift(ctx, v4.ReadEndpoint{
					Key:    "ep",
					Fields: []v4.ReadField{timeField("t", 6), pressure("p")},
				})
				Expect(cfg.Endpoints[0].Messages[0].Index).
					To(HaveValue(Equal(jsonFields(ic)[0].Base().Key)))
				Expect(notes).To(ContainElement("endpoints.0.index"))
			},
		)

		It("Should keep a timestamp field on a data channel as a data field",
			func(ctx SpecContext) {
				ic, cfg, notes := lift(ctx, v4.ReadEndpoint{
					Key:    "ep",
					Fields: []v4.ReadField{timeField("s", 10), timeField("t", 6)},
				})
				Expect(cfg.Endpoints[0].Messages[0].Index).
					To(HaveValue(Equal(jsonFields(ic)[1].Base().Key)))
				Expect(cfg.Endpoints[0].Messages[0].Bindings).To(ContainElement(
					SatisfyAll(
						HaveField("Field", jsonFields(ic)[0].Base().Key),
						HaveField("Channel", channel.Key(10)),
					),
				))
				Expect(notes).To(Equal([]string{"endpoints.0.index"}))
			},
		)

		Describe("Two timestamp fields on index channels", func() {
			// twoIndexes lifts an endpoint whose fields t and u write the index
			// channels time and time_b, with u as its index.
			twoIndexes := func(
				ctx context.Context,
			) (icdv1.ICD, v5.ReadConfig, []string) {
				GinkgoHelper()
				return resolve(ctx, staged(ctx, v4.ReadEndpoint{
					Key:    "ep",
					Index:  "u",
					Fields: []v4.ReadField{timeField("t", 6), timeField("u", 9)},
				}), found)
			}

			It("Should prefer the index field that the endpoint names",
				func(ctx SpecContext) {
					ic, cfg, _ := twoIndexes(ctx)
					Expect(cfg.Endpoints[0].Messages[0].Index).
						To(HaveValue(Equal(jsonFields(ic)[1].Base().Key)))
				},
			)

			It("Should keep the binding of the other timestamp field",
				func(ctx SpecContext) {
					ic, cfg, notes := twoIndexes(ctx)
					Expect(cfg.Endpoints[0].Messages[0].Bindings).To(ContainElement(
						SatisfyAll(
							HaveField("Field", jsonFields(ic)[0].Base().Key),
							HaveField("Channel", channel.Key(6)),
						),
					))
					Expect(notes).To(Equal([]string{"endpoints.0.fields.0"}))
				},
			)

			It("Should lift a read that the read check refuses", func(ctx SpecContext) {
				ic, cfg, _ := twoIndexes(ctx)
				Expect(check(ctx, ic, cfg, found)).To(MatchError(SatisfyAll(
					ContainSubstring("endpoints.0.messages.0.bindings.0.channel"),
					ContainSubstring(
						"field t cannot write to index channel time, because only the "+
							"index field can. Bind field t to a data channel",
					),
				)))
			})
		})

		It("Should drop the fill value of the index field", func(ctx SpecContext) {
			t := timeField("t", 6)
			t.Optional, t.FillValue = true, new(any(3))
			ic, _, notes := lift(ctx, v4.ReadEndpoint{
				Key:    "ep",
				Index:  "t",
				Fields: []v4.ReadField{t, pressure("p")},
			})
			Expect(as[icdv1.TimestampJSONField](jsonFields(ic)[0].Variant).Default).
				To(BeNil())
			Expect(notes).To(ContainElement("endpoints.0.fields.0"))
		})

		It("Should disable an endpoint whose fields are all disabled",
			func(ctx SpecContext) {
				p := pressure("p")
				p.Disabled = true
				_, cfg, notes := lift(ctx, v4.ReadEndpoint{
					Key:    "ep",
					Fields: []v4.ReadField{p},
				})
				Expect(cfg.Endpoints[0].Disabled).To(BeTrue())
				Expect(cfg.Endpoints[0].Messages).To(HaveLen(1))
				Expect(cfg.Endpoints[0].Messages[0].Bindings).To(BeEmpty())
				Expect(notes).To(Equal([]string{"endpoints.0"}))
			},
		)

		It("Should keep an endpoint enabled when its only field has no channel",
			func(ctx SpecContext) {
				p := pressure("p")
				p.Channel = 99
				_, cfg, _ := resolve(ctx, staged(ctx, v4.ReadEndpoint{
					Key:    "ep",
					Fields: []v4.ReadField{p},
				}), found)
				Expect(cfg.Endpoints[0].Disabled).To(BeFalse())
				Expect(cfg.Endpoints[0].Messages[0].Bindings).
					To(HaveExactElements(HaveField("Channel", channel.Key(0))))
			},
		)

		It(
			"Should keep an endpoint enabled when an earlier field writes its only channel",
			func(ctx SpecContext) {
				_, cfg, _ := lift(
					ctx,
					v4.ReadEndpoint{Key: "a", Fields: []v4.ReadField{pressure("p")}},
					v4.ReadEndpoint{Key: "b", Fields: []v4.ReadField{pressure("p")}},
				)
				Expect(cfg.Endpoints[1].Disabled).To(BeFalse())
				Expect(cfg.Endpoints[1].Messages[0].Bindings).To(BeEmpty())
			},
		)

		It("Should not note an endpoint that is already disabled",
			func(ctx SpecContext) {
				p := pressure("p")
				p.Disabled = true
				cfg := staged(
					ctx,
					v4.ReadEndpoint{Key: "ep", Fields: []v4.ReadField{p}},
				)
				cfg.Endpoints[0].Disabled = true
				_, cfg, notes := resolve(ctx, cfg, found)
				Expect(cfg.Endpoints[0].Disabled).To(BeTrue())
				Expect(notes).To(BeEmpty())
			},
		)

		It("Should bind a channel only from the first field that writes it",
			func(ctx SpecContext) {
				_, cfg, notes := lift(
					ctx,
					v4.ReadEndpoint{
						Key:    "a",
						Fields: []v4.ReadField{pressure("p"), pressure("q")},
					},
					v4.ReadEndpoint{Key: "b", Fields: []v4.ReadField{pressure("p")}},
				)
				Expect(cfg.Endpoints[0].Messages[0].Bindings).To(HaveLen(1))
				Expect(cfg.Endpoints[1].Messages[0].Bindings).To(BeEmpty())
				Expect(notes).To(ContainElements(
					"endpoints.0.fields.1",
					"endpoints.1.fields.0",
				))
			},
		)
	})

	It("Should fail when a staged endpoint is not an endpoint of the config",
		func(ctx SpecContext) {
			cfg := staged(ctx, v4.ReadEndpoint{Key: "ep"})
			cfg.Endpoints = nil
			Expect(v5.ResolveReadConfig(ctx, nil, nil, "Plant", &cfg, nil)).Error().
				To(MatchError("staged endpoint ep is not an endpoint of the config"))
		},
	)
})

var _ = Describe("NoteLift", func() {
	It("Should log each change with the config and the field", func() {
		core, logs := observer.New(zapcore.WarnLevel)
		logger := MustSucceed(alamos.NewLogger(alamos.LoggerConfig{
			ZapLogger: zap.New(core),
		}))
		key := uuid.New()
		v5.NoteLift(alamos.New("test", alamos.WithLogger(logger)), key)(
			"endpoints.0.fields.1",
			"fill value was dropped",
		)
		entries := logs.FilterMessage("changed an HTTP read field to v5").All()
		Expect(entries).To(HaveLen(1))
		Expect(entries[0].ContextMap()).To(Equal(map[string]any{
			"config": key.String(),
			"field":  "endpoints.0.fields.1",
			"change": "fill value was dropped",
		}))
	})
})
