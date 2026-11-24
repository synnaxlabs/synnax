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

// writeEndpoint returns an endpoint that sends m of ic, with field 0 of m bound to ch.
func writeEndpoint(ic icd.ICD, m icd.Message, ch channel.Channel) map[string]any {
	return map[string]any{
		"key":    uuid.New().String(),
		"method": "PUT",
		"path":   "/setpoint",
		"icd":    ic.Key,
		"message": map[string]any{
			"message": m.Key,
			"bindings": []any{map[string]any{
				"type": "channel", "field": fieldKey(m, 0), "channel": ch.Key(),
			}},
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
