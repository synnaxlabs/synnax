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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	channel "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v0"
	v4 "github.com/synnaxlabs/synnax/pkg/service/http/versions/v4"
	v5 "github.com/synnaxlabs/synnax/pkg/service/http/versions/v5"
	icdv0 "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v0"
	icdv1 "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v1"
	payload "github.com/synnaxlabs/x/payload/versions/v0"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// resolve resolves cfg with found as the channels that exist. It returns the ICD, the
// resolved config, and each path that a note named.
func resolve(
	ctx context.Context,
	cfg v5.ReadConfig,
	found map[channel.Key]icdv0.Channel,
) (icdv1.ICD, v5.ReadConfig, []string) {
	GinkgoHelper()
	var notes []string
	channels := func(
		context.Context,
		gorp.Snapshot,
		[]channel.Key,
	) (map[channel.Key]icdv0.Channel, error) {
		return found, nil
	}
	ic, ok := MustSucceed2(v5.ResolveReadConfig(
		ctx, nil, channels, "Plant", &cfg,
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

// jsonFields returns the fields of the JSON message at i of ic.
func jsonFields(ic icdv1.ICD, i int) []icdv1.JSONField {
	GinkgoHelper()
	p, ok := ic.Messages[i].Payload.Variant.(icdv1.JSONPayload)
	Expect(ok).To(BeTrue())
	return p.Fields
}

// fill returns a pointer to v, as a fill value.
func fill(v any) *any { return &v }

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
			fields := jsonFields(ic, 0)
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
			Expect(fallback(jsonFields(ic, 0)[0].Variant)).To(Equal(expected))
		},
		Entry(
			"a float",
			v4.ReadField{DataType: telem.Float64T, FillValue: fill(2.5)},
			func(v icdv1.JSONFieldVariant) any {
				return *v.(icdv1.FloatJSONField).Default
			},
			payload.Float(2.5),
		),
		Entry(
			"an integer",
			v4.ReadField{DataType: telem.Int32T, FillValue: fill(-4)},
			func(v icdv1.JSONFieldVariant) any {
				return *v.(icdv1.IntegerJSONField).Default
			},
			payload.Integer(-4),
		),
		Entry(
			"the label of an enum",
			v4.ReadField{
				DataType:   telem.Uint8T,
				EnumValues: []payload.EnumEntry{{Label: "running", Value: 2}},
				FillValue:  fill("running"),
			},
			func(v icdv1.JSONFieldVariant) any {
				return *v.(icdv1.UnsignedJSONField).Default
			},
			payload.Unsigned(2),
		),
		Entry(
			"a boolean",
			v4.ReadField{DataType: telem.BooleanT, FillValue: fill(true)},
			func(v icdv1.JSONFieldVariant) any {
				return *v.(icdv1.BooleanJSONField).Default
			},
			true,
		),
		Entry(
			"a JSON value",
			v4.ReadField{
				DataType:  telem.JSONT,
				FillValue: fill(map[string]any{"a": 1}),
			},
			func(v icdv1.JSONFieldVariant) any {
				return *v.(icdv1.JSONJSONField).Default
			},
			`{"a":1}`,
		),
		Entry(
			"a string",
			v4.ReadField{DataType: telem.StringT, FillValue: fill("idle")},
			func(v icdv1.JSONFieldVariant) any {
				return *v.(icdv1.StringJSONField).Default
			},
			"idle",
		),
		Entry(
			"a timestamp in the time format of its field",
			v4.ReadField{
				DataType:   telem.TimestampT,
				TimeFormat: new(payload.TimeFormatUnixSec),
				FillValue:  fill(3),
			},
			func(v icdv1.JSONFieldVariant) any {
				return v.(icdv1.TimestampJSONField).Default.Variant
			},
			icdv1.ConstantTimestampDefault{Value: telem.TimeStamp(3 * telem.Second)},
		),
		Entry(
			"a UUID",
			v4.ReadField{
				DataType:  telem.UUIDT,
				FillValue: fill("f81d4fae-7dec-11d0-a765-00a0c91e6bf6"),
			},
			func(v icdv1.JSONFieldVariant) any {
				return v.(icdv1.UuidJSONField).Default.Variant
			},
			icdv1.ConstantUUIDDefault{
				Value: uuid.MustParse("f81d4fae-7dec-11d0-a765-00a0c91e6bf6"),
			},
		),
		Entry(
			"bytes",
			v4.ReadField{DataType: telem.BytesT, FillValue: fill("AQI=")},
			func(v icdv1.JSONFieldVariant) any {
				return *v.(icdv1.BytesJSONField).Default
			},
			"AQI=",
		),
	)

	It("Should drop a fill value that does not convert", func(ctx SpecContext) {
		ic, _, notes := resolve(ctx, staged(ctx, v4.ReadEndpoint{
			Key: "ep",
			Fields: []v4.ReadField{{
				Key:       "f",
				Pointer:   "/f",
				DataType:  telem.Float64T,
				FillValue: fill("high"),
			}},
		}), nil)
		Expect(notes).To(Equal([]string{"endpoints.0.fields.0"}))
		Expect(jsonFields(ic, 0)[0].Variant.(icdv1.FloatJSONField).Default).To(BeNil())
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
