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

var _ = Describe("Run", func() {
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
			Framer:  nodeFramer,
		}))
		state = actions.NewState[uuid.UUID, action](db)
		setName, actName = nextName(), nextName()
		Expect(sigs.PublishTable(ctx, signals.TableConfig[uuid.UUID, testUUIDEntry]{
			Changes:    uuidTable.Changes,
			Resource:   testResource[uuid.UUID],
			SetName:    setName,
			DeleteName: nextName(),
		})).To(Succeed())
		Expect(sigs.PublishActions(ctx, signals.ActionConfig[uuid.UUID, action]{
			Name:     actName,
			Actions:  state,
			Resource: testResource[uuid.UUID],
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
		StartRun(sigs.Run)
		set, act := retrieveChannel(ctx, setName), retrieveChannel(ctx, actName)
		responses := openStreamer(ctx, set.Key(), act.Key())
		e := createWithAction(ctx, "whole")
		var res framer.StreamerResponse
		Eventually(responses.Outlet(), 5*time.Second).Should(Receive(&res))
		Expect(res.Frame.KeysSlice()).To(ConsistOf(set.Key(), act.Key()))
		Expect(decodeBatches(res.Frame.Get(act.Key()).Series[0])).
			To(ConsistOf(HaveField("Key", e.Key)))
	})

	It("Should write nothing for a commit before Run", func(ctx SpecContext) {
		set := retrieveChannel(ctx, setName)
		responses := openStreamer(ctx, set.Key())
		createWithAction(ctx, "before")
		StartRun(sigs.Run)
		createWithAction(ctx, "after")
		Expect(receiveSamples(responses)).To(ConsistOf(
			WithTransform(func(b []byte) string { return string(b) },
				ContainSubstring(`"name":"after"`)),
		))
	})

	It("Should write nothing after Run returns", func(ctx SpecContext) {
		stop := StartRun(sigs.Run)
		responses := openStreamer(ctx, retrieveChannel(ctx, setName).Key())
		stop()
		createWithAction(ctx, "closed")
		Consistently(responses.Outlet(), 250*time.Millisecond).ShouldNot(Receive())
	})

	It("Should fail to publish a channel after Run starts", func(ctx SpecContext) {
		StartRun(sigs.Run)
		Expect(sigs.PublishTable(ctx, signals.TableConfig[uuid.UUID, testUUIDEntry]{
			Changes:    uuidTable.Changes,
			Resource:   testResource[uuid.UUID],
			DeleteName: nextName(),
		})).To(MatchError(ContainSubstring("after Run")))
	})

	It("Should return an error when called twice", func(ctx SpecContext) {
		StartRun(sigs.Run)
		Expect(sigs.Run(ctx)).To(MatchError(ContainSubstring("run called twice")))
	})

	It("Should not close while Run runs", func() {
		stop := StartRun(sigs.Run)
		Expect(sigs.Close()).
			To(MatchError(ContainSubstring("close called while Run runs")))
		stop()
		Expect(sigs.Close()).To(Succeed())
	})

	It("Should return an error when called after Close", func(ctx SpecContext) {
		Expect(sigs.Close()).To(Succeed())
		Expect(sigs.Run(ctx)).To(MatchError(ContainSubstring("run called after close")))
	})

	It("Should return nil without change channels", func(ctx SpecContext) {
		empty := MustSucceed(signals.New(signals.Config{
			DB:      db,
			Channel: channelSvc,
			Framer:  nodeFramer,
		}))
		runCtx, cancel := context.WithCancel(ctx)
		cancel()
		Expect(empty.Run(runCtx)).To(Succeed())
	})
})
