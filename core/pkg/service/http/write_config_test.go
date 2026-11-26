// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package http_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/http"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	tasktestutil "github.com/synnaxlabs/synnax/pkg/service/task/testutil"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// writeEndpoint returns an endpoint that sends m of ic, with field 0 of m bound to ch,
// then each of bindings.
func writeEndpoint(
	ic icd.ICD,
	m icd.Message,
	ch channel.Channel,
	bindings ...map[string]any,
) map[string]any {
	return map[string]any{
		"key":    uuid.New().String(),
		"method": "PUT",
		"path":   "/setpoint",
		"icd":    ic.Key,
		"message": map[string]any{
			"message": m.Key,
			"bindings": append([]any{map[string]any{
				"type": "channel", "field": fieldKey(m, 0), "channel": ch.Key(),
			}}, lo.ToAnySlice(bindings)...),
		},
	}
}

// writeConfigOf returns a write config that holds endpoints.
func writeConfigOf(endpoints ...map[string]any) msgpack.EncodedJSON {
	return msgpack.EncodedJSON{"endpoints": lo.ToAnySlice(endpoints)}
}

var _ = Describe("Write config", func() {
	var (
		svc   *http.Service
		tasks *tasktestutil.Tasks
		// ic holds a Setpoint message of one float and an Other message that no
		// endpoint sends.
		ic          icd.ICD
		m, other    icd.Message
		cmd, valves channel.Channel
	)
	BeforeEach(func(ctx SpecContext) {
		svc = MustOpen(http.OpenService(ctx, http.ServiceConfig{
			DB:      db,
			Channel: channelSvc,
			ICD:     icdSvc,
		}))
		tasks = MustOpen(tasktestutil.OpenTasks(ctx, tasktestutil.TasksConfig{
			DB:     db,
			Status: statusSvc,
			Stores: svc.Stores(),
		}))
		m = jsonMessage("Setpoint", jsonField("value", "/value", icd.FloatJSONField{}))
		other = jsonMessage("Other", jsonField("v", "/v", icd.FloatJSONField{}))
		ic = icd.ICD{Name: "Plant", Messages: []icd.Message{m, other}}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return icdSvc.NewWriter(tx).Create(ctx, &ic)
		})).To(Succeed())
		idx := createChannel(ctx, telem.TimestampT, channel.Channel{})
		cmd = createChannel(ctx, telem.Float64T, idx)
		valves = createChannel(ctx, telem.Float64T, idx)
	})

	It("Should make the task a use that reads each command channel it binds", func(
		ctx SpecContext,
	) {
		key := MustSucceed(tasks.Create(
			ctx, svc.Write, writeConfigOf(writeEndpoint(ic, m, cmd)),
		))
		Expect(channelSvc.Uses(ctx, db, []channel.Key{cmd.Key()})).To(HaveExactElements(
			HaveExactElements(SatisfyAll(
				HaveField("Holder.Key", gorp.FormatKey(key)),
				HaveField("Direction", channel.DirectionRead),
			)),
		))
	})

	It("Should save an endpoint that sends a message of its ICD", func(
		ctx SpecContext,
	) {
		key := MustSucceed(tasks.Create(
			ctx, svc.Write, writeConfigOf(writeEndpoint(ic, m, cmd)),
		))
		data := MustSucceed(svc.Write.Read(ctx, db, key))
		Expect(data["endpoints"]).To(HaveExactElements(SatisfyAll(
			HaveKeyWithValue("icd", ic.Key.String()),
			HaveKeyWithValue("message", HaveKeyWithValue("message", m.Key.String())),
		)))
	})

	DescribeTable(
		"Should reject an endpoint that names only one of an ICD and a message",
		func(ctx SpecContext, disabled bool, absent, expected string) {
			ep := writeEndpoint(ic, m, cmd)
			ep["disabled"] = disabled
			delete(ep, absent)
			Expect(svc.Write.Prepare(
				ctx, db, uuid.New(), "Valves", writeConfigOf(ep),
			)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(expected),
			))
		},
		Entry("an ICD without a message", false, "message",
			"endpoints.0.message: required for an endpoint with an ICD",
		),
		Entry("a message without an ICD", false, "icd",
			"endpoints.0.icd: required for an endpoint with a message",
		),
		Entry("a disabled endpoint", true, "message",
			"endpoints.0.message: required for an endpoint with an ICD",
		),
	)

	It("Should reject an endpoint that sends a message its ICD lacks", func(
		ctx SpecContext,
	) {
		lacked := jsonMessage("Lacked", jsonField("v", "/v", icd.FloatJSONField{}))
		Expect(svc.Write.Prepare(
			ctx,
			db,
			uuid.New(),
			"Valves",
			writeConfigOf(writeEndpoint(ic, lacked, cmd)),
		)).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError("endpoints.0.message: message "+lacked.Key.String()+
				" does not exist in ICD Plant"),
		))
	})

	It("Should save a disabled endpoint without checking its message", func(
		ctx SpecContext,
	) {
		lacked := jsonMessage("Lacked", jsonField("v", "/v", icd.FloatJSONField{}))
		disabled := writeEndpoint(ic, lacked, valves)
		disabled["disabled"] = true
		MustSucceed(svc.Write.Prepare(ctx, db, uuid.New(), "Valves", writeConfigOf(
			writeEndpoint(ic, m, cmd),
			disabled,
		)))
	})

	lacked := uuid.New()
	DescribeTable(
		"Should reject an endpoint whose message does not build from its bindings",
		func(
			ctx SpecContext,
			extra func(m icd.Message, ch channel.Channel) map[string]any,
			expected string,
		) {
			pair := jsonMessage(
				"Pair",
				jsonField("value", "/value", icd.FloatJSONField{}),
				jsonField("unit", "/unit", icd.StringJSONField{}),
			)
			pairs := icd.ICD{Name: "Pairs", Messages: []icd.Message{pair}}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return icdSvc.NewWriter(tx).Create(ctx, &pairs)
			})).To(Succeed())
			ep := writeEndpoint(pairs, pair, cmd, extra(pair, valves))
			Expect(svc.Write.Prepare(
				ctx, db, uuid.New(), "Valves", writeConfigOf(ep),
			)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(expected),
			))
		},
		Entry(
			"a binding to a field the message lacks",
			func(_ icd.Message, ch channel.Channel) map[string]any {
				return map[string]any{
					"type": "channel", "field": lacked.String(), "channel": ch.Key(),
				}
			},
			"endpoints.0.message: message Pair: field "+lacked.String()+
				" does not exist: validation error",
		),
		Entry(
			"a value that does not suit its field",
			func(m icd.Message, _ channel.Channel) map[string]any {
				return map[string]any{
					"type":  "value",
					"field": fieldKey(m, 1),
					"value": map[string]any{"type": "float", "value": 1.5},
				}
			},
			"endpoints.0.message.bindings.1.value: field unit of kind string needs a "+
				"value of its kind",
		),
	)

	It("Should import a staged config whose lifted message does not build, and "+
		"refuse its next save", func(ctx SpecContext) {
		data := MustSucceed(svc.Write.Normalize(ctx, db, 4, msgpack.EncodedJSON{
			"endpoints": []any{map[string]any{
				"key":    "sp",
				"method": "PUT",
				"path":   "/setpoint",
				"channel": map[string]any{
					"pointer": "", "channel": cmd.Key(), "data_type": "float64",
				},
				"fields": []any{map[string]any{
					"type":      "static",
					"key":       "u",
					"pointer":   "/unit",
					"json_type": "string",
					"value":     "psi",
				}},
			}},
		}))
		key := MustSucceed(tasks.Create(ctx, svc.Write, data))
		stored := MustSucceed(svc.Write.Read(ctx, db, key))
		Expect(svc.Write.Prepare(ctx, db, key, "Valves", stored)).
			Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(`endpoints.0.message: message PUT /setpoint: field value "" `+
				`and field unit "/unit" overlap: validation error`),
		))
	})

	It("Should import an endpoint whose ICD the Core lacks, without its message", func(
		ctx SpecContext,
	) {
		missing := ic
		missing.Key = uuid.New()
		data := MustSucceed(svc.Write.Normalize(
			ctx, db, svc.Write.Version(), writeConfigOf(writeEndpoint(missing, m, cmd)),
		))
		key := MustSucceed(tasks.Create(ctx, svc.Write, data))
		var cfg http.WriteConfig
		Expect(MustSucceed(svc.Write.Read(ctx, db, key)).Unmarshal(&cfg)).
			To(Succeed())
		Expect(cfg.Endpoints).To(HaveExactElements(SatisfyAll(
			HaveField("Path", "/setpoint"),
			HaveField("ICD", BeNil()),
			HaveField("Message", BeNil()),
		)))
	})

	Describe("Config hash", func() {
		// set applies value at path of the ICD.
		set := func(ctx context.Context, path []string, value any) {
			GinkgoHelper()
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return icdSvc.NewWriter(tx).Apply(ctx, ic.Key, icd.NewSetAction(
					icd.SetPayload{Path: path, Value: value},
				))
			})).To(Succeed())
		}
		// pointerPath is the path of the pointer of field 0 of msg.
		pointerPath := func(msg icd.Message) []string {
			return []string{
				"messages", msg.Key.String(),
				"payload", "fields", fieldKey(msg, 0).String(),
				"pointer",
			}
		}

		It("Should change when the format of a message the task sends changes",
			func(ctx SpecContext) {
				key := MustSucceed(tasks.Create(
					ctx, svc.Write, writeConfigOf(writeEndpoint(ic, m, cmd)),
				))
				before := MustSucceed(svc.Write.Hash(ctx, db, key))
				set(ctx, pointerPath(m), "/setpoint")
				Expect(svc.Write.Hash(ctx, db, key)).ToNot(Equal(before))
			},
		)

		It("Should not change for a message that only a disabled endpoint sends",
			func(ctx SpecContext) {
				disabled := writeEndpoint(ic, other, valves)
				disabled["disabled"] = true
				key := MustSucceed(tasks.Create(ctx, svc.Write, writeConfigOf(
					writeEndpoint(ic, m, cmd),
					disabled,
				)))
				before := MustSucceed(svc.Write.Hash(ctx, db, key))
				set(ctx, pointerPath(other), "/valve")
				Expect(svc.Write.Hash(ctx, db, key)).To(Equal(before))
			},
		)
	})
})
