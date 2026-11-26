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
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/http"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	tasktestutil "github.com/synnaxlabs/synnax/pkg/service/task/testutil"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// jsonField returns a JSON field of an ICD message.
func jsonField(name, pointer string, v icd.JSONFieldVariant) icd.JSONField {
	base := icd.JSONBaseField{Key: uuid.New(), Name: name, Scale: 1, Pointer: pointer}
	f := icd.JSONField{Variant: v}
	f.SetBase(base)
	return f
}

// jsonMessage returns a JSON message of an ICD that holds fields.
func jsonMessage(name string, fields ...icd.JSONField) icd.Message {
	return icd.Message{
		Key:     uuid.New(),
		Name:    name,
		Payload: icd.Payload{Variant: icd.JSONPayload{Fields: fields}},
	}
}

// fieldKey returns the key of field i of m, which must have a JSON payload.
func fieldKey(m icd.Message, i int) icd.FieldKey {
	GinkgoHelper()
	p, ok := m.Payload.Variant.(icd.JSONPayload)
	Expect(ok).To(BeTrue())
	return p.Fields[i].Base().Key
}

// binding returns the binding that writes field i of m to ch.
func binding(m icd.Message, i int, ch channel.Channel) map[string]any {
	return map[string]any{"field": fieldKey(m, i), "channel": ch.Key()}
}

// readMessage returns the config that reads m through bindings.
func readMessage(m icd.Message, bindings ...map[string]any) map[string]any {
	list := make([]any, len(bindings))
	for i, b := range bindings {
		list[i] = b
	}
	return map[string]any{"message": m.Key, "bindings": list}
}

// endpointOf returns an endpoint that reads one message of ic through read.
func endpointOf(ic icd.ICD, read map[string]any) map[string]any {
	return map[string]any{
		"key":      uuid.New().String(),
		"path":     "/values",
		"icd":      ic.Key,
		"messages": []any{read},
	}
}

// readConfigOf returns a read config with one endpoint for each read, each reading one
// message of ic.
func readConfigOf(ic icd.ICD, reads ...map[string]any) msgpack.EncodedJSON {
	endpoints := make([]any, len(reads))
	for i, r := range reads {
		endpoints[i] = endpointOf(ic, r)
	}
	return msgpack.EncodedJSON{"endpoints": endpoints}
}

var _ = Describe("Read config", func() {
	var (
		svc   *http.Service
		tasks *tasktestutil.Tasks
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
	})

	// format is an ICD with a Status message of a timestamp and a float, an Other
	// message that no endpoint reads, and channels on one index to bind them to.
	type format struct {
		ic         icd.ICD
		m, other   icd.Message
		idx, value channel.Channel
	}

	createFormat := func(ctx context.Context) format {
		GinkgoHelper()
		m := jsonMessage("Status",
			jsonField("ts", "/ts", icd.TimestampJSONField{
				TimeFormat: payload.TimeFormatUnixSec,
			}),
			jsonField("temp", "/temp", icd.FloatJSONField{}),
		)
		other := jsonMessage("Other", jsonField("v", "/v", icd.FloatJSONField{}))
		ic := icd.ICD{Name: "Plant", Messages: []icd.Message{m, other}}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return icdSvc.NewWriter(tx).Create(ctx, &ic)
		})).To(Succeed())
		idx := createChannel(ctx, telem.TimestampT, channel.Channel{})
		return format{
			ic:    ic,
			m:     m,
			other: other,
			idx:   idx,
			value: createChannel(ctx, telem.Float64T, idx),
		}
	}

	It("Should save and load an endpoint that names no ICD", func(ctx SpecContext) {
		key := MustSucceed(tasks.Create(ctx, svc.Read, msgpack.EncodedJSON{
			"endpoints": []any{map[string]any{"key": "ep", "path": "/values"}},
		}))
		data := MustSucceed(svc.Read.Read(ctx, db, key))
		Expect(data["endpoints"]).To(HaveExactElements(SatisfyAll(
			HaveKeyWithValue("key", "ep"),
			Not(HaveKey("icd")),
			HaveKeyWithValue("messages", BeEmpty()),
		)))
	})

	It("Should refuse an endpoint that sets fields and reads no messages",
		func(ctx SpecContext) {
			Expect(tasks.Create(ctx, svc.Read, msgpack.EncodedJSON{
				"endpoints": []any{map[string]any{
					"key":    "ep",
					"path":   "/values",
					"fields": []any{map[string]any{"key": "f", "pointer": "/temp"}},
				}},
			})).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError("endpoints.0.fields: an endpoint no longer reads fields. "+
					"Set icd and messages to read the response as ICD messages"),
			))
		},
	)

	It("Should save an endpoint that reads its ICD", func(ctx SpecContext) {
		f := createFormat(ctx)
		key := MustSucceed(tasks.Create(ctx, svc.Read, readConfigOf(
			f.ic, readMessage(f.m, binding(f.m, 1, f.value)),
		)))
		data := MustSucceed(svc.Read.Read(ctx, db, key))
		Expect(data["endpoints"]).To(HaveExactElements(
			HaveKeyWithValue("icd", f.ic.Key.String()),
		))
	})

	DescribeTable("Should reject an endpoint with the path of the problem",
		func(
			ctx SpecContext,
			reads func(f format) []map[string]any,
			path string,
			msg string,
		) {
			f := createFormat(ctx)
			Expect(svc.Read.Prepare(
				ctx, db, uuid.New(), "Pump", readConfigOf(f.ic, reads(f)...),
			)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(HavePrefix(path+": ")),
				MatchError(ContainSubstring(msg)),
			))
		},
		Entry("a message that the ICD lacks", func(f format) []map[string]any {
			lacked := jsonMessage("Lacked", jsonField("v", "/v", icd.FloatJSONField{}))
			return []map[string]any{readMessage(lacked, binding(lacked, 0, f.value))}
		}, "endpoints.0.messages.0.message", "does not exist in ICD Plant"),
		Entry("a field that the message lacks", func(f format) []map[string]any {
			return []map[string]any{readMessage(f.m, map[string]any{
				"field": uuid.New(), "channel": f.value.Key(),
			})}
		},
			"endpoints.0.messages.0.bindings.0.field",
			"does not exist in message Status",
		),
		Entry("a channel that two endpoints write", func(f format) []map[string]any {
			return []map[string]any{
				readMessage(f.m, binding(f.m, 1, f.value)),
				readMessage(f.m, binding(f.m, 1, f.value)),
			}
		}, "endpoints.1.messages.0.bindings.0.channel",
			"is also written by endpoints.0.messages.0.bindings.0.channel"),
	)

	DescribeTable(
		"Should save a disabled endpoint without checking its reads",
		func(ctx SpecContext, read func(f format) map[string]any) {
			f := createFormat(ctx)
			disabled := endpointOf(f.ic, read(f))
			disabled["disabled"] = true
			MustSucceed(
				svc.Read.Prepare(ctx, db, uuid.New(), "Pump", msgpack.EncodedJSON{
					"endpoints": []any{
						endpointOf(f.ic, readMessage(f.m, binding(f.m, 1, f.value))),
						disabled,
					},
				}),
			)
		},
		Entry("a message that the ICD lacks", func(f format) map[string]any {
			lacked := jsonMessage("Lacked", jsonField("v", "/v", icd.FloatJSONField{}))
			return readMessage(lacked, binding(lacked, 0, f.value))
		}),
		Entry(
			"a channel that an enabled endpoint writes",
			func(f format) map[string]any {
				return readMessage(f.m, binding(f.m, 1, f.value))
			},
		),
	)

	Describe("Config hash", func() {
		var (
			f   format
			key task.Key
		)
		hash := func(ctx context.Context) string {
			GinkgoHelper()
			var t task.Task
			Expect(tasks.NewRetrieve().
				Where(task.MatchKeys(key)).
				Entry(&t).
				Exec(ctx, db)).To(Succeed())
			return t.ConfigHash
		}
		set := func(ctx context.Context, path []string, value any) {
			GinkgoHelper()
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return icdSvc.NewWriter(tx).Apply(ctx, f.ic.Key, icd.NewSetAction(
					icd.SetPayload{Path: path, Value: value},
				))
			})).To(Succeed())
		}
		// pointerPath is the path of the pointer of field i of m.
		pointerPath := func(m icd.Message, i int) []string {
			return []string{
				"messages", m.Key.String(),
				"payload", "fields", fieldKey(m, i).String(),
				"pointer",
			}
		}
		BeforeEach(func(ctx SpecContext) {
			f = createFormat(ctx)
			key = MustSucceed(tasks.Create(ctx, svc.Read, readConfigOf(
				f.ic, readMessage(f.m, binding(f.m, 1, f.value)),
			)))
		})

		It("Should change when the format of a message the task reads changes",
			func(ctx SpecContext) {
				before := hash(ctx)
				set(ctx, pointerPath(f.m, 1), "/temperature")
				Expect(hash(ctx)).ToNot(Equal(before))
			},
		)

		It("Should not change when the name of the ICD changes", func(ctx SpecContext) {
			before := hash(ctx)
			set(ctx, []string{"name"}, "Line")
			Expect(hash(ctx)).To(Equal(before))
		})

		It("Should not change when a message the task does not read changes",
			func(ctx SpecContext) {
				before := hash(ctx)
				set(ctx, pointerPath(f.other, 0), "/value")
				Expect(hash(ctx)).To(Equal(before))
			},
		)

		It("Should not change for a message that only a disabled endpoint reads",
			func(ctx SpecContext) {
				disabled := endpointOf(
					f.ic,
					readMessage(f.other, binding(f.other, 0, f.value)),
				)
				disabled["disabled"] = true
				key = MustSucceed(tasks.Create(ctx, svc.Read, msgpack.EncodedJSON{
					"endpoints": []any{
						endpointOf(f.ic, readMessage(f.m, binding(f.m, 1, f.value))),
						disabled,
					},
				}))
				before := hash(ctx)
				set(ctx, pointerPath(f.other, 0), "/value")
				Expect(hash(ctx)).To(Equal(before))
			},
		)
	})
})
