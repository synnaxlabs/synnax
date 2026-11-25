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
	channel "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v0"
	v4 "github.com/synnaxlabs/synnax/pkg/service/http/versions/v4"
	v5 "github.com/synnaxlabs/synnax/pkg/service/http/versions/v5"
	icdv0 "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v0"
	icdv1 "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v1"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// resolveWrite stages old, then resolves it with found as the channels that exist. It
// returns the ICD, the resolved config, and the change that a note gave each path.
func resolveWrite(
	ctx context.Context,
	old v4.WriteConfig,
	found map[channel.Key]icdv0.Channel,
) (icdv1.ICD, v5.WriteConfig, map[string]string) {
	GinkgoHelper()
	cfg := MustSucceed(v5.MigrateWriteConfig(ctx, old))
	notes := make(map[string]string)
	ic, ok := MustSucceed2(v5.ResolveWriteConfig(
		ctx, nil, channelsOf(found), "Plant", &cfg,
		func(path, change string) { notes[path] = change },
	))
	Expect(ok).To(BeTrue())
	Expect(cfg.Inline).To(BeNil())
	return ic, cfg, notes
}

// setpoint holds channel 42 of storedWriteConfig as a float64 channel.
var setpoint = map[channel.Key]icdv0.Channel{42: {DataType: telem.Float64T}}

var _ = Describe("ResolveWriteConfig", func() {
	It("Should return false for a config with nothing staged", func(ctx SpecContext) {
		cfg := v5.WriteConfig{}
		_, ok := MustSucceed2(v5.ResolveWriteConfig(ctx, nil, nil, "Plant", &cfg, nil))
		Expect(ok).To(BeFalse())
	})

	It("Should move each endpoint into a message of an ICD named after the task",
		func(ctx SpecContext) {
			ic, cfg, _ := resolveWrite(ctx, storedWriteConfig(), setpoint)
			Expect(ic.Name).To(Equal("Plant"))
			Expect(ic.Messages).To(HaveLen(2))
			Expect(ic.Messages[0].Name).To(Equal("PUT /setpoint"))
			Expect(ic.Messages[1].Name).To(Equal("POST /valve"))
			fields := jsonFields(ic)
			Expect(fields).To(HaveLen(3))
			value := as[icdv1.FloatJSONField](fields[0].Variant)
			Expect(value.Name).To(Equal("value"))
			Expect(value.Pointer).To(Equal("/value"))
			unit := as[icdv1.StringJSONField](fields[1].Variant)
			Expect(unit.Pointer).To(Equal("/unit"))
			Expect(unit.Default).To(Equal(new("psi")))
			id := as[icdv1.UuidJSONField](fields[2].Variant)
			Expect(id.Pointer).To(Equal("/request_id"))
			Expect(id.Default).To(Equal(&icdv1.UUIDDefault{
				Variant: icdv1.RandomUUIDDefault{},
			}))
			Expect(cfg.Endpoints[0].ICD).To(Equal(new(ic.Key)))
			Expect(cfg.Endpoints[0].Message.Message).To(Equal(ic.Messages[0].Key))
			Expect(cfg.Endpoints[1].ICD).To(Equal(new(ic.Key)))
			Expect(cfg.Endpoints[1].Message.Message).To(Equal(ic.Messages[1].Key))
			Expect(cfg.Endpoints[0].Message.Bindings).To(Equal([]icdv1.WriteBinding{{
				Variant: icdv1.ChannelWriteBinding{
					WriteBaseBinding: icdv1.WriteBaseBinding{Field: value.Key},
					Channel:          42,
				},
			}}))
		},
	)

	It("Should move a disabled endpoint into a message and keep it disabled",
		func(ctx SpecContext) {
			ic, cfg, _ := resolveWrite(ctx, storedWriteConfig(), setpoint)
			valve := cfg.Endpoints[1]
			Expect(valve.Disabled).To(BeTrue())
			Expect(valve.ICD).To(Equal(new(ic.Key)))
			Expect(valve.Message.Message).To(Equal(ic.Messages[1].Key))
		},
	)

	It("Should bind a new channel of the stored data type to a missing channel",
		func(ctx SpecContext) {
			old := storedWriteConfig()
			old.Endpoints[1].Channel.Name = "valve_cmd"
			ic, cfg, notes := resolveWrite(ctx, old, setpoint)
			binding := as[icdv1.ChannelWriteBinding](
				cfg.Endpoints[1].Message.Bindings[0].Variant,
			)
			Expect(binding.Channel).To(BeZero())
			Expect(binding.DataType).To(Equal(new(telem.Uint8T)))
			Expect(as[icdv1.JSONPayload](ic.Messages[1].Payload.Variant).Fields).
				To(HaveLen(1))
			Expect(notes).To(HaveKeyWithValue(
				"endpoints.1.channel",
				`channel name "valve_cmd" of field value on POST /valve was dropped, `+
					"so a client names the new channel",
			))
		},
	)

	It("Should name the messages of endpoints with the same request apart",
		func(ctx SpecContext) {
			old := storedWriteConfig()
			old.Endpoints[1].Method, old.Endpoints[1].Path = v4.MethodPut, "/setpoint"
			ic, _, _ := resolveWrite(ctx, old, setpoint)
			Expect(ic.Messages[0].Name).To(Equal("PUT /setpoint"))
			Expect(ic.Messages[1].Name).To(Equal("PUT /setpoint (2)"))
		},
	)

	It("Should drop a null field with a note at its path", func(ctx SpecContext) {
		old := storedWriteConfig()
		old.Endpoints[0].Fields = append(old.Endpoints[0].Fields, v4.WriteField{})
		ic, _, notes := resolveWrite(ctx, old, setpoint)
		Expect(jsonFields(ic)).To(HaveLen(3))
		Expect(notes).To(HaveKeyWithValue(
			"endpoints.0.fields.2",
			"a null field was dropped",
		))
	})

	It("Should place a channel with an empty pointer at the root of the body",
		func(ctx SpecContext) {
			old := storedWriteConfig()
			old.Endpoints[0].Channel.Pointer = ""
			old.Endpoints[0].Fields = nil
			ic, _, _ := resolveWrite(ctx, old, setpoint)
			fields := jsonFields(ic)
			Expect(fields).To(HaveLen(1))
			Expect(fields[0].Base().Pointer).To(BeEmpty())
		},
	)

	It("Should fail for a staged endpoint that the config does not hold",
		func(ctx SpecContext) {
			cfg := MustSucceed(v5.MigrateWriteConfig(ctx, storedWriteConfig()))
			cfg.Endpoints = cfg.Endpoints[:1]
			Expect(v5.ResolveWriteConfig(
				ctx, nil, channelsOf(setpoint), "Plant", &cfg, nil,
			)).Error().To(MatchError(
				"staged endpoint valve is not an endpoint of the config",
			))
		},
	)
})

var _ = Describe("NoteWriteLift", func() {
	It("Should log each change with the config and the field", func() {
		core, logs := observer.New(zapcore.WarnLevel)
		logger := MustSucceed(alamos.NewLogger(alamos.LoggerConfig{
			ZapLogger: zap.New(core),
		}))
		key := uuid.New()
		v5.NoteWriteLift(alamos.New("test", alamos.WithLogger(logger)), key)(
			"endpoints.1.channel",
			"channel name was dropped",
		)
		entries := logs.FilterMessage("changed an HTTP write field to v5").All()
		Expect(entries).To(HaveLen(1))
		Expect(entries[0].ContextMap()).To(Equal(map[string]any{
			"config": key.String(),
			"field":  "endpoints.1.channel",
			"change": "channel name was dropped",
		}))
	})
})
