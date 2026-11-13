// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package task_test

import (
	"context"
	"encoding/json/v2"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	apitask "github.com/synnaxlabs/synnax/pkg/api/task"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/pagerduty"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// denied matches an error that denies u the action on object.
func denied(u user.User, action access.Action, object resource.ID) types.GomegaMatcher {
	return SatisfyAll(
		MatchError(access.ErrDenied),
		MatchError(ContainSubstring(
			"%s cannot %s %s",
			u.ResourceID(),
			action,
			object,
		)),
	)
}

// need is an action on an object.
type need struct {
	action access.Action
	object resource.ID
}

// grantAll grants each need to u.
func grantAll(ctx SpecContext, u user.User, needs ...need) {
	GinkgoHelper()
	for _, n := range needs {
		grant(ctx, n.action, u.ResourceID(), n.object)
	}
}

// statusOf returns the status that t holds.
func statusOf(ctx SpecContext, t task.Task) resource.ID {
	GinkgoHelper()
	var st task.Status
	Expect(svc.Status.NewRetrieve[task.StatusDetails]().
		Where(status.MatchOwners[task.StatusDetails](t.ResourceID())).
		Entry(&st).
		Exec(ctx, node.DB)).To(Succeed())
	return st.ResourceID()
}

func createTask(ctx SpecContext) task.Task {
	GinkgoHelper()
	t := task.Task{
		Type: pagerduty.AlertTaskType,
		Rack: svc.Rack.EmbeddedKey,
		Name: "Before",
	}
	Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
		return svc.Task.NewWriter(tx).Create(ctx, &t)
	})).To(Succeed())
	return t
}

func retrieveTask(ctx SpecContext, key task.Key) task.Task {
	GinkgoHelper()
	var t task.Task
	Expect(svc.Task.NewRetrieve().
		Where(task.MatchKeys(key)).
		Entry(&t).
		Exec(ctx, node.DB)).To(Succeed())
	return t
}

var _ = Describe("Rename", func() {
	It("Should rename the task and its status with update on both", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		t := createTask(ctx)
		grant(
			ctx,
			access.ActionUpdate,
			u.ResourceID(),
			t.ResourceID(),
			statusOf(ctx, t),
		)
		MustSucceed(rename(
			AuthedCtx(ctx, u), apitask.RenameRequest{Key: t.Key, Name: "After"},
		))
		renamed := retrieveTask(ctx, t.Key)
		Expect(renamed.Name).To(Equal("After"))
		Expect(renamed.Config).To(Equal(t.Config))
	})
	It("Should reject an empty name", func(ctx SpecContext) {
		u := createUser(ctx)
		t := createTask(ctx)
		grant(
			ctx,
			access.ActionUpdate,
			u.ResourceID(),
			t.ResourceID(),
			statusOf(ctx, t),
		)
		Expect(rename(
			AuthedCtx(ctx, u), apitask.RenameRequest{Key: t.Key, Name: ""},
		)).Error().To(MatchError(ContainSubstring("name: required")))
		Expect(retrieveTask(ctx, t.Key).Name).To(Equal("Before"))
	})
	It("Should return a not found error for a missing task", func(ctx SpecContext) {
		u := createUser(ctx)
		key := uuid.New()
		grant(ctx, access.ActionUpdate, u.ResourceID(), task.ResourceID(key))
		Expect(rename(
			AuthedCtx(ctx, u), apitask.RenameRequest{Key: key, Name: "After"},
		)).Error().To(MatchError(query.ErrNotFound))
	})
	taskID := func(_ SpecContext, t task.Task) resource.ID { return t.ResourceID() }
	DescribeTable(
		"Should refuse the rename and keep the name",
		func(
			ctx SpecContext,
			granted func(SpecContext, task.Task) resource.ID,
			refused func(SpecContext, task.Task) resource.ID,
		) {
			u := createUser(ctx)
			t := createTask(ctx)
			grant(ctx, access.ActionUpdate, u.ResourceID(), granted(ctx, t))
			Expect(rename(
				AuthedCtx(ctx, u), apitask.RenameRequest{Key: t.Key, Name: "After"},
			)).Error().To(denied(u, access.ActionUpdate, refused(ctx, t)))
			Expect(retrieveTask(ctx, t.Key).Name).To(Equal("Before"))
		},
		Entry("without update on the task", statusOf, taskID),
		Entry("without update on its status", taskID, statusOf),
	)
})

var _ = Describe("Dispatch", func() {
	createDraft := func(ctx SpecContext) task.Task {
		GinkgoHelper()
		t := task.Task{
			Type: pagerduty.AlertTaskType,
			Rack: svc.Rack.EmbeddedKey,
			Name: "Alerts",
		}
		Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Task.NewWriter(tx).Create(ctx, &t)
		})).To(Succeed())
		return t
	}
	setRoutingKey := []task.Action{task.NewSetAction(task.SetPayload{
		Path:  []string{"config", "routing_key"},
		Value: "rk-2",
	})}
	It("Should apply the actions and return the ones the Core added", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		t := createDraft(ctx)
		grant(ctx, access.ActionUpdate, u.ResourceID(), t.ResourceID())
		grant(ctx, access.ActionRetrieve, u.ResourceID(), t.ResourceID())
		res := MustSucceed(dispatch(
			AuthedCtx(ctx, u),
			apitask.DispatchRequest{
				Key:         t.Key,
				DispatchKey: "dk",
				Actions:     setRoutingKey,
			},
		))
		stored := retrieveTask(ctx, t.Key)
		Expect(stored.Config["routing_key"]).To(Equal("rk-2"))
		Expect(stored.ConfigHash).ToNot(Equal(t.ConfigHash))
		Expect(res.Actions).To(Equal([]task.Action{task.NewSetAction(task.SetPayload{
			Path:  []string{"config_hash"},
			Value: stored.ConfigHash,
		})}))
	})
	It("Should return no actions to a subject that cannot read the task", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		t := createDraft(ctx)
		grant(ctx, access.ActionUpdate, u.ResourceID(), t.ResourceID())
		res := MustSucceed(dispatch(
			AuthedCtx(ctx, u),
			apitask.DispatchRequest{
				Key:         t.Key,
				DispatchKey: "dk",
				Actions:     setRoutingKey,
			},
		))
		Expect(res.Actions).To(BeEmpty())
		Expect(retrieveTask(ctx, t.Key).Config["routing_key"]).To(Equal("rk-2"))
	})
	It("Should reject the request when update is not granted", func(ctx SpecContext) {
		u := createUser(ctx)
		t := createDraft(ctx)
		Expect(dispatch(
			AuthedCtx(ctx, u),
			apitask.DispatchRequest{
				Key:         t.Key,
				DispatchKey: "dk",
				Actions:     setRoutingKey,
			},
		)).Error().To(denied(u, access.ActionUpdate, t.ResourceID()))
		Expect(retrieveTask(ctx, t.Key).ConfigHash).To(Equal(t.ConfigHash))
	})
	It("Should return the dispatch error", func(ctx SpecContext) {
		u := createUser(ctx)
		key := uuid.New()
		grant(ctx, access.ActionUpdate, u.ResourceID(), task.ResourceID(key))
		Expect(dispatch(
			AuthedCtx(ctx, u),
			apitask.DispatchRequest{
				Key:         key,
				DispatchKey: "dk",
				Actions:     setRoutingKey,
			},
		)).Error().To(MatchError(query.ErrNotFound))
	})
})

var _ = Describe("Create", func() {
	newTask := func() task.Task {
		return task.Task{
			Key:  uuid.New(),
			Type: pagerduty.AlertTaskType,
			Rack: svc.Rack.EmbeddedKey,
			Name: "Created",
		}
	}
	// createNeeds returns what a create of t needs: create on t and its new status, and
	// update on the rack that t sits in.
	createNeeds := func(t task.Task) []need {
		return []need{
			{access.ActionCreate, t.ResourceID()},
			{access.ActionCreate, resource.ID{Type: resource.TypeStatus}},
			{access.ActionUpdate, t.Rack.ResourceID()},
		}
	}
	It(
		"Should stamp the dispatch key onto the create broadcast",
		func(ctx SpecContext) {
			u := createUser(ctx)
			t := newTask()
			grantAll(ctx, u, createNeeds(t)...)
			var keys []string
			DeferCleanup(svc.Task.OnAction(func(
				_ context.Context,
				f actions.Scoped[task.Key, task.Action],
			) {
				keys = append(keys, f.DispatchKey)
			}))
			res := MustSucceed(create(AuthedCtx(ctx, u), apitask.CreateRequest{
				Tasks:       []task.Task{t},
				DispatchKey: "dk-create",
			}))
			Expect(res.Tasks).To(HaveLen(1))
			Expect(keys).To(Equal([]string{"dk-create"}))
		},
	)

	DescribeTable(
		"Should refuse the create and store nothing",
		func(ctx SpecContext, refused int) {
			u := createUser(ctx)
			t := newTask()
			needs := createNeeds(t)
			grantAll(ctx, u, slices.Delete(slices.Clone(needs), refused, refused+1)...)
			Expect(create(AuthedCtx(ctx, u), apitask.CreateRequest{
				Tasks: []task.Task{t},
			})).Error().To(denied(u, needs[refused].action, needs[refused].object))
			Expect(svc.Task.NewRetrieve().
				Where(task.MatchKeys(t.Key)).
				Exists(ctx, node.DB)).To(BeFalse())
		},
		Entry("without create on the task", 0),
		Entry("without update on its rack", 2),
	)
})

var _ = Describe("Retrieve", func() {
	It("Should leave out the tasks that the subject cannot retrieve", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		granted := createTask(ctx)
		createTask(ctx)
		grant(ctx, access.ActionRetrieve, u.ResourceID(), granted.ResourceID())
		res := MustSucceed(retrieve(AuthedCtx(ctx, u), apitask.RetrieveRequest{}))
		Expect(res.Tasks).To(HaveExactElements(HaveField("Key", granted.Key)))
	})

	It("Should attach only each task's own status", func(ctx SpecContext) {
		u := createUser(ctx)
		withStatus := createTask(ctx)
		withoutStatus := createTask(ctx)
		Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Status.NewWriter(tx).DeleteOwned(ctx, withoutStatus.ResourceID())
		})).To(Succeed())
		grant(
			ctx, access.ActionRetrieve, u.ResourceID(),
			withStatus.ResourceID(), withoutStatus.ResourceID(),
		)
		res := MustSucceed(retrieve(AuthedCtx(ctx, u), apitask.RetrieveRequest{
			Where: &task.Where{Key: &filter.Set[task.Key]{
				In: []task.Key{withoutStatus.Key, withStatus.Key},
			}},
			IncludeStatus: true,
		}))
		Expect(res.Tasks).To(HaveLen(2))
		Expect(res.Tasks[0].Key).To(Equal(withoutStatus.Key))
		Expect(res.Tasks[0].Status).To(BeNil())
		Expect(res.Tasks[1].Key).To(Equal(withStatus.Key))
		Expect(res.Tasks[1].Status.Owner).To(HaveValue(Equal(withStatus.ResourceID())))
	})

	It("Should keep only the tasks that run the given Arcs", func(ctx SpecContext) {
		u := createUser(ctx)
		grant(ctx, access.ActionRetrieve, u.ResourceID(), resource.ID{
			Type: resource.TypeTask,
		})
		deployed := arc.Arc{Variant: arc.AutomationArc{Base: arc.Base{
			Name:   "Deployed",
			Source: arc.Source{Variant: arc.GraphSource{}},
		}}}
		other := arc.Arc{Variant: arc.AutomationArc{Base: arc.Base{
			Name:   "Other",
			Source: arc.Source{Variant: arc.GraphSource{}},
		}}}
		Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Arc.NewWriter(tx).Create(ctx, &deployed)
		})).To(Succeed())
		Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Arc.NewWriter(tx).Create(ctx, &other)
		})).To(Succeed())
		update := arc.TaskUpdate{Rack: new(svc.Rack.EmbeddedKey)}
		var tsk *task.Task
		Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) (err error) {
			tsk, err = svc.Arc.NewWriter(tx).
				UpdateTask(ctx, deployed.Base().Key, update)
			return err
		})).To(Succeed())
		Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			_, err := svc.Arc.NewWriter(tx).UpdateTask(ctx, other.Base().Key, update)
			return err
		})).To(Succeed())
		res := MustSucceed(retrieve(AuthedCtx(ctx, u), apitask.RetrieveRequest{
			Where: &task.Where{Arcs: []uuid.UUID{deployed.Base().Key}},
		}))
		Expect(res.Tasks).To(HaveLen(1))
		Expect(res.Tasks[0].Key).To(Equal(tsk.Key))
	})

	It("Should treat a missing key as a non-match", func(ctx SpecContext) {
		u := createUser(ctx)
		t := createTask(ctx)
		missing := uuid.New()
		grant(
			ctx,
			access.ActionRetrieve,
			u.ResourceID(),
			t.ResourceID(),
			task.ResourceID(missing),
		)
		res := MustSucceed(retrieve(AuthedCtx(ctx, u), apitask.RetrieveRequest{
			Where: &task.Where{Key: &filter.Set[task.Key]{
				In: []task.Key{t.Key, missing},
			}},
		}))
		Expect(res.Tasks).To(HaveLen(1))
		Expect(res.Tasks[0].Key).To(Equal(t.Key))
	})

	It("Should return the tasks whose name matches the search term", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		match := task.Task{
			Type: pagerduty.AlertTaskType,
			Rack: svc.Rack.EmbeddedKey,
			Name: "Cryostat pressure",
		}
		Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Task.NewWriter(tx).Create(ctx, &match)
		})).To(Succeed())
		other := createTask(ctx)
		grant(
			ctx, access.ActionRetrieve, u.ResourceID(),
			match.ResourceID(), other.ResourceID(),
		)
		res := MustSucceed(retrieve(AuthedCtx(ctx, u), apitask.RetrieveRequest{
			SearchTerm: "cryostat",
		}))
		Expect(res.Tasks).To(HaveExactElements(HaveField("Key", match.Key)))
	})

	It("Should return a path error for an invalid regex", func(ctx SpecContext) {
		u := createUser(ctx)
		Expect(retrieve(AuthedCtx(ctx, u), apitask.RetrieveRequest{
			Where: &task.Where{Type: &filter.String{Regex: new("(a")}},
		})).Error().To(SatisfyAll(
			BeAssignableToTypeOf(validate.PathError{}),
			MatchError(
				"where.type.regex: error parsing regexp: missing closing ): `(a`: validation error",
			),
		))
	})
})

var _ = Describe("RetrieveRequest", func() {
	It("Should reject a request member that it does not declare", func() {
		var req apitask.RetrieveRequest
		Expect(json.Unmarshal([]byte(`{"names":["Task"]}`), &req)).To(SatisfyAll(
			BeAssignableToTypeOf(&json.SemanticError{}),
			MatchError(ContainSubstring(`unknown object member name "names"`)),
		))
	})
})

// createRunningTask creates a task on a new rack, because the rack monitor marks the
// tasks of a rack that is silent for a health check interval as not running.
func createRunningTask(ctx SpecContext, running bool) task.Task {
	GinkgoHelper()
	r := rack.Rack{Name: "Driver"}
	Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
		return svc.Rack.NewWriter(tx).Create(ctx, &r)
	})).To(Succeed())
	t := task.Task{
		Type: pagerduty.AlertTaskType,
		Rack: r.Key,
		Name: "Running",
		Status: &task.Status{
			Variant: status.VariantSuccess,
			Message: "Task is running",
			Time:    telem.Now(),
			Details: task.StatusDetails{Running: running},
		},
	}
	Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
		return svc.Task.NewWriter(tx).Create(ctx, &t)
	})).To(Succeed())
	return t
}

// reportStopped writes the status a task writes when it stops.
func reportStopped(ctx context.Context, t task.Task) {
	GinkgoHelper()
	stat := task.Status{
		Owner:   new(t.ResourceID()),
		Variant: status.VariantSuccess,
		Message: "Stopped task",
		Time:    telem.Now(),
	}
	Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
		return svc.Status.NewWriter(tx).Set(ctx, &stat)
	})).To(Succeed())
}

// streamCommands sends each command written to the task command channel on the returned
// channel. It returns once the stream is live.
func streamCommands(ctx SpecContext) <-chan task.Command {
	GinkgoHelper()
	streamer := MustSucceed(svc.Framer.NewStreamer(ctx, framer.StreamerConfig{
		Keys:        channel.Keys{svc.Task.CommandChannelKey()},
		SendOpenAck: true,
	}))
	requests, responses := confluence.Attach(streamer, 2)
	sCtx, cancel := signal.Isolated()
	shutdown := signal.NewHardShutdown(sCtx, cancel)
	streamer.Flow(sCtx, confluence.CloseOutputInletsOnExit())
	Eventually(responses.Outlet()).Should(Receive())
	commands := make(chan task.Command, 16)
	done := make(chan struct{})
	go func() {
		defer GinkgoRecover()
		defer close(done)
		for res := range responses.Outlet() {
			for series := range res.Frame.Series() {
				for sample := range series.Samples() {
					var cmd task.Command
					Expect(json.Unmarshal(sample, &cmd)).To(Succeed())
					commands <- cmd
				}
			}
		}
	}()
	DeferCleanup(func() {
		requests.Close()
		Eventually(done).Should(BeClosed())
		Expect(shutdown.Close()).To(Succeed())
	})
	return commands
}

// stallWrites leaves storage streamers that never read on the Core, then parks writers
// behind them. Storage holds each write until every one of its streamers takes the
// frame or a slow-consumer timeout passes, so a new write waits behind the parked ones.
func stallWrites(ctx SpecContext) {
	GinkgoHelper()
	ch := channel.Channel{
		Name:     "stall_" + strings.ReplaceAll(uuid.New().String(), "-", "_"),
		DataType: telem.Float64T,
		Virtual:  true,
	}
	Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
		return svc.Channel.NewWriter(tx).Create(ctx, &ch)
	})).To(Succeed())
	sCtx, cancel := signal.Isolated()
	shutdown := signal.NewHardShutdown(sCtx, cancel)
	const streamerCount = 16
	var (
		requests  = make([]confluence.Inlet[ts.StreamerRequest], streamerCount)
		responses = make([]confluence.Outlet[ts.StreamerResponse], streamerCount)
	)
	for i := range streamerCount {
		streamer := MustSucceed(node.Storage.TS.NewStreamer(ctx, ts.StreamerConfig{
			Channels: []ts.ChannelKey{ch.Key().StorageKey()},
		}))
		requests[i], responses[i] = confluence.Attach(streamer, 1)
		streamer.Flow(sCtx, confluence.CloseOutputInletsOnExit())
	}
	var (
		written atomic.Int64
		stopped atomic.Bool
		parked  sync.WaitGroup
	)
	for range 128 {
		// Ginkgo cancels ctx when the spec ends, which is before the cleanup stops the
		// writers.
		w := MustSucceed(svc.Framer.OpenWriter(sCtx, framer.WriterConfig{
			Keys:  channel.Keys{ch.Key()},
			Start: telem.Now(),
		}))
		parked.Go(func() {
			defer GinkgoRecover()
			for !stopped.Load() {
				fr := frame.NewUnary(ch.Key(), telem.NewSeriesV(1.0))
				Expect(w.Write(fr)).To(BeTrue())
				written.Add(1)
			}
			Expect(w.Close()).To(Succeed())
		})
	}
	Eventually(func(g Gomega) {
		before := written.Load()
		time.Sleep(100 * time.Millisecond)
		g.Expect(written.Load() - before).To(BeNumerically("<", 10))
	}).WithTimeout(30 * time.Second).Should(Succeed())
	DeferCleanup(func() {
		stopped.Store(true)
		for i := range streamerCount {
			requests[i].Close()
			confluence.Drain(responses[i])
		}
		parked.Wait()
		Expect(shutdown.Close()).To(Succeed())
	})
}

var _ = Describe("Delete", func() {
	var u user.User
	BeforeEach(func(ctx SpecContext) { u = createUser(ctx) })
	// deleteWith returns the delete route, bound as the API layer binds it, with the
	// stop timeout.
	deleteWith := func(
		timeout time.Duration,
	) freighter.UnaryHandler[apitask.DeleteRequest, struct{}] {
		GinkgoHelper()
		s := MustSucceed(apitask.NewService(apitask.ServiceConfig{
			DB:          node.DB,
			Task:        svc.Task,
			Status:      svc.Status,
			RBAC:        svc.RBAC,
			Framer:      svc.Framer,
			StopTimeout: timeout,
		}))
		return bind.WritePrepared(binder, s.PrepareDelete, s.Delete)
	}
	// deleteNeeds returns what a delete of t needs: delete on t and its status, and
	// update on the rack that t leaves.
	deleteNeeds := func(t task.Task) []need {
		return []need{
			{access.ActionDelete, t.ResourceID()},
			{access.ActionDelete, resource.ID{Type: resource.TypeStatus}},
			{access.ActionUpdate, t.Rack.ResourceID()},
		}
	}
	expectGone := func(ctx SpecContext, key task.Key) {
		GinkgoHelper()
		var t task.Task
		Expect(svc.Task.NewRetrieve().
			Where(task.MatchKeys(key)).
			Entry(&t).
			Exec(ctx, node.DB)).To(MatchError(query.ErrNotFound))
	}
	It("Should release the channels of a running task", func(ctx SpecContext) {
		t := createRunningTask(ctx, true)
		grantAll(ctx, u, deleteNeeds(t)...)
		ch := channel.Channel{
			Name:        "task_" + strings.ReplaceAll(uuid.New().String(), "-", "_"),
			DataType:    telem.TimestampT,
			IsIndex:     true,
			Leaseholder: 1,
		}
		Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Channel.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		w := MustSucceed(svc.Framer.OpenWriter(ctx, framer.WriterConfig{
			Keys:  channel.Keys{ch.Key()},
			Start: telem.Now(),
		}))
		commands := streamCommands(ctx)
		driver := make(chan struct{})
		go func() {
			defer GinkgoRecover()
			defer close(driver)
			var cmd task.Command
			Eventually(commands).Should(Receive(&cmd))
			Expect(cmd.Task).To(Equal(t.Key))
			Expect(cmd.Type).To(Equal("stop"))
			Expect(w.Close()).To(Succeed())
			reportStopped(ctx, t)
		}()
		Expect(deleteWith(10*time.Second)(
			AuthedCtx(ctx, u), apitask.DeleteRequest{Keys: []task.Key{t.Key}},
		)).Error().ToNot(HaveOccurred())
		Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Channel.NewWriter(tx).Delete(ctx, ch.Key(), false)
		})).To(Succeed())
		Eventually(driver).Should(BeClosed())
		expectGone(ctx, t.Key)
	})
	It(
		"Should return as soon as a running task reports its stop",
		func(ctx SpecContext) {
			t := createRunningTask(ctx, true)
			grantAll(ctx, u, deleteNeeds(t)...)
			commands := streamCommands(ctx)
			driver := make(chan struct{})
			go func() {
				defer GinkgoRecover()
				defer close(driver)
				Eventually(commands).Should(Receive())
				reportStopped(ctx, t)
			}()
			start := time.Now()
			Expect(deleteWith(5*time.Second)(
				AuthedCtx(ctx, u), apitask.DeleteRequest{Keys: []task.Key{t.Key}},
			)).Error().ToNot(HaveOccurred())
			Expect(time.Since(start)).To(BeNumerically("<", 2*time.Second))
			Eventually(driver).Should(BeClosed())
			expectGone(ctx, t.Key)
		},
	)
	It("Should stop a batch that never acks within one timeout", func(ctx SpecContext) {
		const timeout = 300 * time.Millisecond
		tasks := []task.Task{
			createRunningTask(ctx, true),
			createRunningTask(ctx, true),
			createRunningTask(ctx, true),
		}
		keys := make([]task.Key, len(tasks))
		for i, t := range tasks {
			keys[i] = t.Key
			grantAll(ctx, u, deleteNeeds(t)...)
		}
		commands := streamCommands(ctx)
		start := time.Now()
		Expect(deleteWith(timeout)(
			AuthedCtx(ctx, u), apitask.DeleteRequest{Keys: keys},
		)).Error().ToNot(HaveOccurred())
		elapsed := time.Since(start)
		Expect(elapsed).To(BeNumerically(">=", timeout))
		Expect(elapsed).To(BeNumerically("<", 2*timeout))
		stopped := make([]task.Key, 0, len(tasks))
		for range tasks {
			var cmd task.Command
			Eventually(commands).Should(Receive(&cmd))
			Expect(cmd.Type).To(Equal("stop"))
			stopped = append(stopped, cmd.Task)
		}
		Expect(stopped).To(ConsistOf(keys))
		Consistently(commands, 100*time.Millisecond).ShouldNot(Receive())
		for _, key := range keys {
			expectGone(ctx, key)
		}
	})
	It(
		"Should return within the timeout when a stop cannot be written",
		func(ctx SpecContext) {
			const timeout = 200 * time.Millisecond
			t := createRunningTask(ctx, true)
			grantAll(ctx, u, deleteNeeds(t)...)
			stallWrites(ctx)
			start := time.Now()
			Expect(deleteWith(timeout)(
				AuthedCtx(ctx, u), apitask.DeleteRequest{Keys: []task.Key{t.Key}},
			)).Error().ToNot(HaveOccurred())
			Expect(time.Since(start)).To(BeNumerically("<", 2*timeout))
			expectGone(ctx, t.Key)
		},
	)
	It("Should stop only the requested task", func(ctx SpecContext) {
		target := createRunningTask(ctx, true)
		createRunningTask(ctx, true)
		grantAll(ctx, u, deleteNeeds(target)...)
		commands := streamCommands(ctx)
		Expect(deleteWith(100*time.Millisecond)(
			AuthedCtx(ctx, u), apitask.DeleteRequest{Keys: []task.Key{target.Key}},
		)).Error().ToNot(HaveOccurred())
		var cmd task.Command
		Eventually(commands).Should(Receive(&cmd))
		Expect(cmd.Task).To(Equal(target.Key))
		Consistently(commands, 100*time.Millisecond).ShouldNot(Receive())
		expectGone(ctx, target.Key)
	})
	DescribeTable(
		"Should keep the task when the caller cancels",
		func(ctx SpecContext, after time.Duration) {
			t := createRunningTask(ctx, true)
			grantAll(ctx, u, deleteNeeds(t)...)
			reqCtx, cancel := context.WithCancel(ctx)
			defer time.AfterFunc(after, cancel).Stop()
			if after == 0 {
				cancel()
			}
			Expect(deleteWith(10*time.Second)(
				AuthedCtx(reqCtx, u), apitask.DeleteRequest{Keys: []task.Key{t.Key}},
			)).Error().To(MatchError(context.Canceled))
			Expect(retrieveTask(ctx, t.Key).Key).To(Equal(t.Key))
		},
		Entry("before the stop", time.Duration(0)),
		Entry("during the wait", 50*time.Millisecond),
	)
	DescribeTable(
		"Should not wait for a task that is not running",
		func(ctx SpecContext, create func(SpecContext) task.Task) {
			t := create(ctx)
			grantAll(ctx, u, deleteNeeds(t)...)
			commands := streamCommands(ctx)
			start := time.Now()
			Expect(deleteWith(10*time.Second)(
				AuthedCtx(ctx, u), apitask.DeleteRequest{Keys: []task.Key{t.Key}},
			)).Error().ToNot(HaveOccurred())
			Expect(time.Since(start)).To(BeNumerically("<", time.Second))
			Consistently(commands, 100*time.Millisecond).ShouldNot(Receive())
			expectGone(ctx, t.Key)
		},
		Entry("never started", func(ctx SpecContext) task.Task {
			t := createTask(ctx)
			Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Status.NewWriter(tx).Delete(
					ctx, t.ResourceID().String(),
				)
			})).To(Succeed())
			return t
		}),
		Entry("stopped", func(ctx SpecContext) task.Task {
			return createRunningTask(ctx, false)
		}),
		// The rack monitor clears the running flag of a task on a silent rack.
		Entry("on an offline Driver", func(ctx SpecContext) task.Task {
			t := createRunningTask(ctx, true)
			reportStopped(ctx, t)
			return t
		}),
	)
	It("Should not stop a task the caller cannot delete", func(ctx SpecContext) {
		t := createRunningTask(ctx, true)
		grantAll(ctx, u, deleteNeeds(t)[1:]...)
		commands := streamCommands(ctx)
		Expect(deleteWith(time.Second)(
			AuthedCtx(ctx, u), apitask.DeleteRequest{Keys: []task.Key{t.Key}},
		)).Error().To(denied(u, access.ActionDelete, t.ResourceID()))
		Consistently(commands, 100*time.Millisecond).ShouldNot(Receive())
		Expect(retrieveTask(ctx, t.Key).Key).To(Equal(t.Key))
	})
	DescribeTable(
		"Should refuse the delete and keep the task",
		func(
			ctx SpecContext,
			refused int,
			object func(SpecContext, task.Task) resource.ID,
		) {
			t := createTask(ctx)
			needs := deleteNeeds(t)
			grantAll(ctx, u, slices.Delete(slices.Clone(needs), refused, refused+1)...)
			Expect(deleteWith(time.Second)(
				AuthedCtx(ctx, u), apitask.DeleteRequest{Keys: []task.Key{t.Key}},
			)).Error().To(denied(u, needs[refused].action, object(ctx, t)))
			Expect(retrieveTask(ctx, t.Key).Key).To(Equal(t.Key))
		},
		Entry("without delete on its status", 1, statusOf),
		Entry(
			"without update on its rack",
			2,
			func(_ SpecContext, t task.Task) resource.ID { return t.Rack.ResourceID() },
		),
	)
})

var _ = Describe("Copy", func() {
	var (
		u    user.User
		t    task.Task
		name string
	)
	BeforeEach(func(ctx SpecContext) {
		u = createUser(ctx)
		t = createTask(ctx)
		name = "Copy " + uuid.New().String()
	})
	// copyNeeds returns what a copy of t needs: retrieve on t, create on the copy and
	// its status, and update on its rack.
	copyNeeds := func() []need {
		return []need{
			{access.ActionRetrieve, t.ResourceID()},
			{access.ActionCreate, resource.ID{Type: resource.TypeTask}},
			{access.ActionCreate, resource.ID{Type: resource.TypeStatus}},
			{access.ActionUpdate, t.Rack.ResourceID()},
		}
	}
	copyAs := func(ctx SpecContext) (apitask.CopyResponse, error) {
		return copyTask(AuthedCtx(ctx, u), apitask.CopyRequest{Key: t.Key, Name: name})
	}
	countCopies := func(ctx SpecContext) int {
		GinkgoHelper()
		return MustSucceed(svc.Task.NewRetrieve().
			Where(task.MatchNames(name)).
			Count(ctx, node.DB))
	}
	It("Should copy the task under a new key", func(ctx SpecContext) {
		grantAll(ctx, u, copyNeeds()...)
		res := MustSucceed(copyAs(ctx))
		Expect(res.Task.Key).ToNot(Equal(t.Key))
		Expect(retrieveTask(ctx, res.Task.Key).Name).To(Equal(name))
	})

	DescribeTable(
		"Should refuse the copy and store nothing",
		func(ctx SpecContext, refused int) {
			needs := copyNeeds()
			grantAll(ctx, u, slices.Delete(slices.Clone(needs), refused, refused+1)...)
			Expect(copyAs(ctx)).Error().
				To(denied(u, needs[refused].action, needs[refused].object))
			Expect(countCopies(ctx)).To(BeZero())
		},
		Entry("without retrieve on the task", 0),
		Entry("without update on its rack", 3),
	)
})
