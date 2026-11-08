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
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Start", func() {
	var (
		sigs    *signals.Provider
		state   *actions.State[uuid.UUID, action]
		setName string
		actName string
	)
	BeforeEach(func(ctx SpecContext) {
		sigs = MustSucceed(signals.New(signals.Config{
			DB:      db,
			Channel: channelSvc,
			Framer:  framerSvc,
		}))
		state = actions.NewState[uuid.UUID, action](db)
		setName, actName = nextName(), nextName()
		Expect(sigs.PublishTable(ctx, signals.TableConfig[uuid.UUID, testUUIDEntry]{
			Changes:    uuidTable.Changes,
			SetName:    setName,
			DeleteName: nextName(),
		})).To(Succeed())
		Expect(sigs.PublishActions(ctx, signals.ActionConfig[uuid.UUID, action]{
			Name:    actName,
			Actions: state,
		})).To(Succeed())
	})

	createWithAction := func(ctx SpecContext, name string) testUUIDEntry {
		GinkgoHelper()
		e := testUUIDEntry{Key: uuid.New(), Name: name}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			if err := uuidTable.NewCreate().Entry(&e).Exec(ctx, tx); err != nil {
				return err
			}
			state.Dispatcher().Attach(tx, e.Key, action{Type: "create"})
			return nil
		})).To(Succeed())
		return e
	}

	It("Should write the changes of one commit as one frame", func(ctx SpecContext) {
		MustOpen(sigs.Start(ctx))
		set, act := retrieveChannel(ctx, setName), retrieveChannel(ctx, actName)
		responses := openStreamer(ctx, set.Key(), act.Key())
		e := createWithAction(ctx, "whole")
		var res framer.StreamerResponse
		Eventually(responses.Outlet(), 5*time.Second).Should(Receive(&res))
		Expect(res.Frame.KeysSlice()).To(ConsistOf(set.Key(), act.Key()))
		Expect(decodeBatches(res.Frame.Get(act.Key()).Series[0])).
			To(ConsistOf(HaveField("Key", e.Key)))
	})

	It("Should write nothing for a commit before Start", func(ctx SpecContext) {
		set := retrieveChannel(ctx, setName)
		responses := openStreamer(ctx, set.Key())
		createWithAction(ctx, "before")
		MustOpen(sigs.Start(ctx))
		createWithAction(ctx, "after")
		Expect(receiveSamples(responses)).To(ConsistOf(
			WithTransform(func(b []byte) string { return string(b) },
				ContainSubstring(`"name":"after"`)),
		))
	})

	It("Should write nothing after Close", func(ctx SpecContext) {
		closer := MustSucceed(sigs.Start(ctx))
		responses := openStreamer(ctx, retrieveChannel(ctx, setName).Key())
		Expect(closer.Close()).To(Succeed())
		createWithAction(ctx, "closed")
		Consistently(responses.Outlet(), 250*time.Millisecond).ShouldNot(Receive())
	})

	It("Should fail to publish a channel after Start", func(ctx SpecContext) {
		MustOpen(sigs.Start(ctx))
		Expect(sigs.PublishTable(ctx, signals.TableConfig[uuid.UUID, testUUIDEntry]{
			Changes:    uuidTable.Changes,
			DeleteName: nextName(),
		})).To(MatchError(ContainSubstring("after Start")))
	})

	It("Should fail when Start runs twice", func(ctx SpecContext) {
		MustOpen(sigs.Start(ctx))
		Expect(sigs.Start(ctx)).Error().To(MatchError(ContainSubstring("already ran")))
	})
})
