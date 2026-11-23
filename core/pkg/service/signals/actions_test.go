// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package signals_test

import (
	"context"
	"encoding/json/v2"
	"slices"
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

type action struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
}

type scoped = actions.Scoped[uuid.UUID, action]

// decodeBatches decodes the action batches of s.
func decodeBatches(s telem.Series) []scoped {
	GinkgoHelper()
	var decoded []scoped
	for sample := range s.Samples() {
		var sa scoped
		Expect(json.Unmarshal(sample, &sa)).To(Succeed())
		decoded = append(decoded, sa)
	}
	return decoded
}

var _ = Describe("Actions", func() {
	Describe("ActionConfig", func() {
		full := func() signals.ActionConfig[uuid.UUID, action] {
			return signals.ActionConfig[uuid.UUID, action]{
				Name:     "actions_validate",
				Actions:  actions.NewState[uuid.UUID, action](nil),
				Resource: testResource[uuid.UUID],
			}
		}

		It("Should succeed when all required fields are set", func() {
			Expect(full().Validate()).To(Succeed())
		})

		DescribeTable(
			"Should fail when a required field is missing",
			func(unset func(*signals.ActionConfig[uuid.UUID, action]), field string) {
				cfg := full()
				unset(&cfg)
				Expect(cfg.Validate()).To(MatchError(ContainSubstring(field)))
			},
			Entry(
				"actions",
				func(c *signals.ActionConfig[uuid.UUID, action]) { c.Actions = nil },
				"actions",
			),
			Entry(
				"name",
				func(c *signals.ActionConfig[uuid.UUID, action]) { c.Name = "" },
				"name",
			),
			Entry(
				"resource",
				func(c *signals.ActionConfig[uuid.UUID, action]) { c.Resource = nil },
				"resource",
			),
		)
	})

	Describe("BatchCheck", func() {
		var (
			sigs     *signals.Provider
			state    *actions.State[uuid.UUID, action]
			ch       channel.Channel
			received confluence.Outlet[framer.StreamerResponse]
			checks   int
			marker   = resource.ID{Type: "marker", Key: "batch"}
		)
		BeforeEach(func(ctx SpecContext) {
			checks = 0
			sigs = MustSucceed(signals.New(signals.Config{
				DB:      db,
				Channel: channelSvc,
				Framer:  nodeFramer,
			}))
			state = actions.NewState[uuid.UUID, action](db)
			name := nextName()
			Expect(sigs.PublishActions(ctx, signals.ActionConfig[uuid.UUID, action]{
				Name:     name,
				Actions:  state,
				Resource: testResource[uuid.UUID],
				BatchChecks: func(
					gorp.Commit,
				) func(uuid.UUID, uint64) (func(func(resource.ID) bool) ([]byte, bool), bool) {
					checks++
					return func(
						uuid.UUID,
						uint64,
					) (func(func(resource.ID) bool) ([]byte, bool), bool) {
						return func(allowed func(resource.ID) bool) ([]byte, bool) {
							return nil, allowed(marker)
						}, true
					}
				},
			})).To(Succeed())
			MustOpen(sigs.Start(ctx))
			ch = retrieveChannel(ctx, name)
			received = openStreamer(ctx, ch.Key())
		})

		It("Should check each batch by the lookup that its commit made once", func(
			ctx SpecContext,
		) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				state.Dispatcher().Attach(tx, uuid.New(), action{Type: "rename"})
				return nil
			})).To(Succeed())
			var res framer.StreamerResponse
			Eventually(received.Outlet(), 5*time.Second).Should(Receive(&res))
			sample := slices.Collect(res.Frame.SeriesAt(0).Samples())[0]
			record := sigs.Records()[ch.Key()]
			sends := func(allowed func(resource.ID) bool) bool {
				_, send := MustSucceed(record.Check(sample))(allowed)
				return send
			}
			Expect(
				sends(func(id resource.ID) bool { return id == marker }),
			).To(BeTrue())
			Expect(sends(func(resource.ID) bool { return false })).To(BeFalse())
			Expect(checks).To(Equal(1))
		})

		It("Should fail the check of a batch whose check it does not hold", func() {
			sample := MustSucceed(json.Marshal(scoped{Key: uuid.New(), Seq: 1 << 40}))
			Expect(sigs.Records()[ch.Key()].Check(sample)).Error().
				To(MatchError("[signals] - the check of a change is gone"))
		})
	})

	Describe("PublishActions", func() {
		var (
			state     *actions.State[uuid.UUID, action]
			ch        channel.Channel
			responses confluence.Outlet[framer.StreamerResponse]
			logs      *observer.ObservedLogs
		)
		BeforeEach(func(ctx SpecContext) {
			var core zapcore.Core
			core, logs = observer.New(zapcore.DPanicLevel)
			logger := MustSucceed(alamos.NewLogger(alamos.LoggerConfig{
				ZapLogger: zap.New(core),
			}))
			sigs := MustSucceed(signals.New(signals.Config{
				DB:              db,
				Channel:         channelSvc,
				Framer:          nodeFramer,
				Instrumentation: alamos.New("signals", alamos.WithLogger(logger)),
			}))
			state = actions.NewState[uuid.UUID, action](db)
			name := nextName()
			Expect(sigs.PublishActions(ctx, signals.ActionConfig[uuid.UUID, action]{
				Name:     name,
				Actions:  state,
				Resource: testResource[uuid.UUID],
				Table:    uuidTableName,
			})).To(Succeed())
			MustOpen(sigs.Start(ctx))
			ch = retrieveChannel(ctx, name)
			responses = openStreamer(ctx, ch.Key())
		})

		It("Should create the channel as a free, built-in JSON channel", func() {
			Expect(ch.Virtual).To(BeTrue())
			Expect(ch.Free()).To(BeTrue())
			Expect(channelSvc.IsBuiltIn(ch.Key())).To(BeTrue())
			Expect(ch.DataType).To(Equal(telem.JSONT))
		})

		It("Should write each attached batch as one sample", func(ctx SpecContext) {
			key := uuid.New()
			acts := []action{{Type: "rename", Payload: "next-name"}}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				state.Dispatcher().WithDispatchKey("dk-1").Attach(tx, key, acts...)
				return nil
			})).To(Succeed())
			var res framer.StreamerResponse
			Eventually(responses.Outlet(), 5*time.Second).Should(Receive(&res))
			Expect(res.Frame.KeysSlice()).To(ConsistOf(ch.Key()))
			Expect(decodeBatches(res.Frame.SeriesAt(0))).To(ConsistOf(
				And(
					HaveField("Key", key),
					HaveField("DispatchKey", "dk-1"),
					HaveField("Actions", acts),
				),
			))
		})

		It("Should write the Seq of each batch", func(ctx SpecContext) {
			key := uuid.New()
			var attached []uint64
			DeferCleanup(db.Observe().OnChange(func(_ context.Context, c gorp.Commit) {
				for _, b := range state.Of(c) {
					attached = append(attached, b.Seq)
				}
			}))
			d := state.Dispatcher()
			for _, t := range []string{"a", "b", "c"} {
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					d.Attach(tx, key, action{Type: t})
					return nil
				})).To(Succeed())
			}
			var seqs []uint64
			for len(seqs) < 3 {
				var res framer.StreamerResponse
				Eventually(responses.Outlet(), 5*time.Second).Should(Receive(&res))
				for _, sa := range decodeBatches(res.Frame.SeriesAt(0)) {
					seqs = append(seqs, sa.Seq)
				}
			}
			Expect(seqs).To(Equal(attached))
		})

		It(
			"Should report a set of a table record with no batch for its key",
			func(ctx SpecContext) {
				e := testUUIDEntry{Key: uuid.New(), Name: "uncovered"}
				Expect(uuidTable.NewCreate().Entry(&e).Exec(ctx, db)).To(Succeed())
				reports := logs.FilterMessage("a change has no action batch").All()
				Expect(reports).To(HaveLen(1))
				Expect(reports[0].Level).To(Equal(zapcore.DPanicLevel))
				Expect(reports[0].ContextMap()).To(Equal(map[string]any{
					"channel": ch.Name,
					"table":   uuidTableName,
					"key":     gorp.FormatKey(e.Key),
				}))
			},
		)

		It(
			"Should not report a covered set or a delete of a table record",
			func(ctx SpecContext) {
				e := testUUIDEntry{Key: uuid.New(), Name: "covered"}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					if err := uuidTable.NewCreate().
						Entry(&e).
						Exec(ctx, tx); err != nil {
						return err
					}
					state.Dispatcher().Attach(tx, e.Key, action{Type: "create"})
					return nil
				})).To(Succeed())
				Expect(uuidTable.NewDelete().
					Where(gorp.MatchKeys[uuid.UUID, testUUIDEntry](e.Key)).
					Exec(ctx, db)).To(Succeed())
				Expect(logs.All()).To(BeEmpty())
			},
		)
	})
})
