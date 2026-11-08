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
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
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
				Name:    "actions_validate",
				Actions: actions.NewState[uuid.UUID, action](nil),
			}
		}

		It("Should succeed when all required fields are set", func() {
			Expect(full().Validate()).To(Succeed())
		})

		DescribeTable(
			"Should fail when a required field is missing",
			func(clear func(*signals.ActionConfig[uuid.UUID, action]), field string) {
				cfg := full()
				clear(&cfg)
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
		)
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
				Framer:          framerSvc,
				Instrumentation: alamos.New("signals", alamos.WithLogger(logger)),
			}))
			state = actions.NewState[uuid.UUID, action](db)
			name := nextName()
			Expect(sigs.PublishActions(ctx, signals.ActionConfig[uuid.UUID, action]{
				Name:    name,
				Actions: state,
				Table:   uuidTableName,
			})).To(Succeed())
			MustOpen(sigs.Start(ctx))
			ch = retrieveChannel(ctx, name)
			responses = openStreamer(ctx, ch.Key())
		})

		It("Should create the channel as a free, internal JSON channel", func() {
			Expect(ch.Virtual).To(BeTrue())
			Expect(ch.Free()).To(BeTrue())
			Expect(ch.Internal).To(BeTrue())
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

		It("Should set the Seq of each batch to the number of its commit", func(
			ctx SpecContext,
		) {
			key := uuid.New()
			var commits []uint64
			DeferCleanup(db.Observe().OnChange(func(_ context.Context, c gorp.Commit) {
				if len(state.Of(c)) > 0 {
					commits = append(commits, c.Seq)
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
			Expect(seqs).To(Equal(commits))
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
