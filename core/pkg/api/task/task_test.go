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
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	apitask "github.com/synnaxlabs/synnax/pkg/api/task"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/pagerduty"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

func createTask(ctx SpecContext) task.Task {
	GinkgoHelper()
	t := task.Task{
		Type:     pagerduty.AlertTaskType,
		Rack:     svc.Rack.EmbeddedKey,
		Name:     "Before",
		Snapshot: true,
	}
	Expect(svc.Task.NewWriter(nil).Create(ctx, &t)).To(Succeed())
	return t
}

func retrieveTask(ctx SpecContext, key task.Key) task.Task {
	GinkgoHelper()
	var t task.Task
	Expect(svc.Task.NewRetrieve().
		Where(task.MatchKeys(key)).
		Entry(&t).
		Exec(ctx, nil)).To(Succeed())
	return t
}

var _ = Describe("Rename", func() {
	It("Should rename the task when update is granted", func(ctx SpecContext) {
		u := createUser(ctx)
		t := createTask(ctx)
		grant(ctx, access.ActionUpdate, u.OntologyID(), t.OntologyID())
		Expect(apiSvc.Rename(
			AuthedCtx(ctx, u), nil,
			apitask.RenameRequest{Key: t.Key, Name: "After"},
		)).Error().ToNot(HaveOccurred())
		renamed := retrieveTask(ctx, t.Key)
		Expect(renamed.Name).To(Equal("After"))
		Expect(renamed.Snapshot).To(BeTrue())
		Expect(renamed.Config).To(Equal(t.Config))
	})
	It("Should reject an empty name", func(ctx SpecContext) {
		u := createUser(ctx)
		t := createTask(ctx)
		grant(ctx, access.ActionUpdate, u.OntologyID(), t.OntologyID())
		Expect(apiSvc.Rename(
			AuthedCtx(ctx, u), nil,
			apitask.RenameRequest{Key: t.Key, Name: ""},
		)).Error().To(MatchError(ContainSubstring("name: required")))
		Expect(retrieveTask(ctx, t.Key).Name).To(Equal("Before"))
	})
	It("Should return a not found error for a missing task", func(ctx SpecContext) {
		u := createUser(ctx)
		key := uuid.New()
		grant(ctx, access.ActionUpdate, u.OntologyID(), task.OntologyID(key))
		Expect(apiSvc.Rename(
			AuthedCtx(ctx, u), nil,
			apitask.RenameRequest{Key: key, Name: "After"},
		)).Error().To(MatchError(query.ErrNotFound))
	})
	It("Should reject the request when update is not granted", func(ctx SpecContext) {
		u := createUser(ctx)
		t := createTask(ctx)
		Expect(apiSvc.Rename(
			AuthedCtx(ctx, u), nil,
			apitask.RenameRequest{Key: t.Key, Name: "After"},
		)).Error().To(MatchError(access.ErrDenied))
		Expect(retrieveTask(ctx, t.Key).Name).To(Equal("Before"))
	})
})

var _ = Describe("Dispatch", func() {
	createDraft := func(ctx SpecContext) task.Task {
		GinkgoHelper()
		t := task.Task{
			Type: pagerduty.AlertTaskType,
			Rack: svc.Rack.EmbeddedKey,
			Name: "Alerts",
		}
		Expect(svc.Task.NewWriter(nil).Create(ctx, &t)).To(Succeed())
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
		grant(ctx, access.ActionUpdate, u.OntologyID(), t.OntologyID())
		grant(ctx, access.ActionRetrieve, u.OntologyID(), t.OntologyID())
		res := MustSucceed(apiSvc.Dispatch(
			AuthedCtx(ctx, u),
			nil,
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
		grant(ctx, access.ActionUpdate, u.OntologyID(), t.OntologyID())
		res := MustSucceed(apiSvc.Dispatch(
			AuthedCtx(ctx, u),
			nil,
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
		Expect(apiSvc.Dispatch(
			AuthedCtx(ctx, u),
			nil,
			apitask.DispatchRequest{
				Key:         t.Key,
				DispatchKey: "dk",
				Actions:     setRoutingKey,
			},
		)).Error().To(MatchError(access.ErrDenied))
		Expect(retrieveTask(ctx, t.Key).ConfigHash).To(Equal(t.ConfigHash))
	})
	It("Should return the dispatch error", func(ctx SpecContext) {
		u := createUser(ctx)
		key := uuid.New()
		grant(ctx, access.ActionUpdate, u.OntologyID(), task.OntologyID(key))
		Expect(apiSvc.Dispatch(
			AuthedCtx(ctx, u),
			nil,
			apitask.DispatchRequest{
				Key:         key,
				DispatchKey: "dk",
				Actions:     setRoutingKey,
			},
		)).Error().To(MatchError(query.ErrNotFound))
	})
})

var _ = Describe("Create", func() {
	It(
		"Should stamp the dispatch key onto the create broadcast",
		func(ctx SpecContext) {
			u := createUser(ctx)
			grant(ctx, access.ActionCreate, u.OntologyID(), ontology.ID{
				Type: ontology.ResourceTypeTask,
			})
			var keys []string
			DeferCleanup(svc.Task.OnAction(func(
				_ context.Context,
				f actions.Scoped[task.Key, task.Action],
			) {
				keys = append(keys, f.DispatchKey)
			}))
			res := MustSucceed(
				apiSvc.Create(AuthedCtx(ctx, u), nil, apitask.CreateRequest{
					Tasks: []task.Task{{
						Type: pagerduty.AlertTaskType,
						Rack: svc.Rack.EmbeddedKey,
						Name: "Created",
					}},
					DispatchKey: "dk-create",
				}),
			)
			Expect(res.Tasks).To(HaveLen(1))
			Expect(keys).To(Equal([]string{"dk-create"}))
		},
	)

	Describe("Concurrent writes", func() {
		var (
			t   task.Task
			u   user.User
			req apitask.CreateRequest
		)
		BeforeEach(func(ctx SpecContext) {
			t = task.Task{
				Type: pagerduty.AlertTaskType,
				Rack: svc.Rack.EmbeddedKey,
				Name: "Alerts",
			}
			Expect(svc.Task.NewWriter(nil).Create(ctx, &t)).To(Succeed())
			u = createUser(ctx)
			grant(ctx, access.ActionCreate, u.OntologyID(), ontology.ID{
				Type: ontology.ResourceTypeTask,
			})
			renamed := t
			renamed.Name = "Renamed"
			req = apitask.CreateRequest{Tasks: []task.Task{renamed}}
		})

		It("Should keep a dispatch that lands before the request commits", func(
			ctx SpecContext,
		) {
			tx := db.OpenTx()
			DeferCleanup(func() { Expect(tx.Close()).To(Succeed()) })
			MustSucceed(apiSvc.Create(AuthedCtx(ctx, u), tx, req))
			MustSucceed(svc.Task.Dispatch(ctx, t.Key, "", []task.Action{
				task.NewSetAction(task.SetPayload{
					Path:  []string{"config", "routing_key"},
					Value: "rk-2",
				}),
			}))
			Expect(tx.Commit(ctx)).To(Succeed())
			var stored task.Task
			Expect(svc.Task.NewRetrieve().
				Where(task.MatchKeys(t.Key)).
				Entry(&stored).
				Exec(ctx, nil)).To(Succeed())
			Expect(stored.Name).To(Equal("Renamed"))
			Expect(stored.Config).To(HaveKeyWithValue("routing_key", "rk-2"))
		})

		It("Should wait for a write that holds the task", func(ctx SpecContext) {
			created := make(chan error, 1)
			Expect(svc.Task.Transact(ctx, []task.Key{t.Key}, func(gorp.Tx) error {
				go func() {
					_, err := apiSvc.Create(AuthedCtx(ctx, u), nil, req)
					created <- err
				}()
				Consistently(created, 100*time.Millisecond).ShouldNot(Receive())
				return nil
			})).To(Succeed())
			Eventually(created).Should(Receive(Not(HaveOccurred())))
		})
	})
})

var _ = Describe("Retrieve", func() {
	It("Should attach only each task's own status", func(ctx SpecContext) {
		u := createUser(ctx)
		withStatus := createTask(ctx)
		withoutStatus := createTask(ctx)
		Expect(svc.Status.NewWriter(nil).Delete(
			ctx, withoutStatus.OntologyID().String(),
		)).To(Succeed())
		grant(
			ctx, access.ActionRetrieve, u.OntologyID(),
			withStatus.OntologyID(), withoutStatus.OntologyID(),
		)
		res := MustSucceed(apiSvc.Retrieve(AuthedCtx(ctx, u), apitask.RetrieveRequest{
			Where: &task.Where{Key: &filter.Set[task.Key]{
				In: []task.Key{withoutStatus.Key, withStatus.Key},
			}},
			IncludeStatus: true,
		}))
		Expect(res.Tasks).To(HaveLen(2))
		Expect(res.Tasks[0].Key).To(Equal(withoutStatus.Key))
		Expect(res.Tasks[0].Status).To(BeNil())
		Expect(res.Tasks[1].Key).To(Equal(withStatus.Key))
		Expect(res.Tasks[1].Status.Key).To(Equal(withStatus.OntologyID().String()))
	})

	It("Should treat a missing key as a non-match", func(ctx SpecContext) {
		u := createUser(ctx)
		t := createTask(ctx)
		grant(ctx, access.ActionRetrieve, u.OntologyID(), t.OntologyID())
		res := MustSucceed(apiSvc.Retrieve(AuthedCtx(ctx, u), apitask.RetrieveRequest{
			Where: &task.Where{Key: &filter.Set[task.Key]{
				In: []task.Key{t.Key, uuid.New()},
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
			Type:     pagerduty.AlertTaskType,
			Rack:     svc.Rack.EmbeddedKey,
			Name:     "Cryostat pressure",
			Snapshot: true,
		}
		Expect(svc.Task.NewWriter(nil).Create(ctx, &match)).To(Succeed())
		other := createTask(ctx)
		grant(
			ctx, access.ActionRetrieve, u.OntologyID(),
			match.OntologyID(), other.OntologyID(),
		)
		res := MustSucceed(apiSvc.Retrieve(AuthedCtx(ctx, u), apitask.RetrieveRequest{
			SearchTerm: "cryostat",
		}))
		Expect(res.Tasks).To(HaveExactElements(HaveField("Key", match.Key)))
	})

	It("Should return a path error for an invalid regex", func(ctx SpecContext) {
		u := createUser(ctx)
		Expect(apiSvc.Retrieve(AuthedCtx(ctx, u), apitask.RetrieveRequest{
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
	Expect(svc.Rack.NewWriter(nil).Create(ctx, &r)).To(Succeed())
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
	Expect(svc.Task.NewWriter(nil).Create(ctx, &t)).To(Succeed())
	return t
}

// reportStopped writes the status a task writes when it stops.
func reportStopped(ctx context.Context, t task.Task) {
	GinkgoHelper()
	stat := task.Status{
		Key:     t.OntologyID().String(),
		Variant: status.VariantSuccess,
		Message: "Stopped task",
		Time:    telem.Now(),
		Details: task.StatusDetails{Task: t.Key},
	}
	Expect(svc.Status.NewWriter(nil).Set(ctx, &stat)).To(Succeed())
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

// stallFreeWrites fills the free write queue of the Core behind a streamer that never
// reads, and leaves writers parked on it. A new write then waits behind every one of
// them.
func stallFreeWrites(ctx SpecContext) {
	GinkgoHelper()
	ch := channel.Channel{
		Name:     "stall_" + strings.ReplaceAll(uuid.New().String(), "-", "_"),
		DataType: telem.Float64T,
		Virtual:  true,
	}
	Expect(svc.Channel.NewWriter(nil).Create(ctx, &ch)).To(Succeed())
	streamer := MustSucceed(svc.Framer.NewStreamer(ctx, framer.StreamerConfig{
		Keys:        channel.Keys{ch.Key()},
		SendOpenAck: true,
	}))
	requests, responses := confluence.Attach(streamer, 1)
	sCtx, cancel := signal.Isolated()
	shutdown := signal.NewHardShutdown(sCtx, cancel)
	streamer.Flow(sCtx, confluence.CloseOutputInletsOnExit())
	Eventually(responses.Outlet()).Should(Receive())
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
		requests.Close()
		confluence.Drain(responses)
		parked.Wait()
		Expect(shutdown.Close()).To(Succeed())
	})
}

var _ = Describe("Delete", func() {
	var u user.User
	BeforeEach(func(ctx SpecContext) { u = createUser(ctx) })
	withTimeout := func(timeout time.Duration) *apitask.Service {
		GinkgoHelper()
		return MustSucceed(apitask.NewService(apitask.ServiceConfig{
			Task:   svc.Task,
			Status: svc.Status,
			RBAC:   svc.RBAC,
			Stopper: MustSucceed(task.NewStopper(task.StopperConfig{
				Task:    svc.Task,
				Status:  svc.Status,
				Framer:  svc.Framer,
				Timeout: timeout,
			})),
		}))
	}
	expectGone := func(ctx SpecContext, key task.Key) {
		GinkgoHelper()
		var t task.Task
		Expect(svc.Task.NewRetrieve().
			Where(task.MatchKeys(key)).
			Entry(&t).
			Exec(ctx, nil)).To(MatchError(query.ErrNotFound))
	}
	It("Should release the channels of a running task", func(ctx SpecContext) {
		t := createRunningTask(ctx, true)
		grant(ctx, access.ActionDelete, u.OntologyID(), t.OntologyID())
		ch := channel.Channel{
			Name:        "task_" + strings.ReplaceAll(uuid.New().String(), "-", "_"),
			DataType:    telem.TimestampT,
			IsIndex:     true,
			Leaseholder: 1,
		}
		Expect(svc.Channel.NewWriter(nil).Create(ctx, &ch)).To(Succeed())
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
		Expect(withTimeout(10*time.Second).Delete(
			AuthedCtx(ctx, u), nil, apitask.DeleteRequest{Keys: []task.Key{t.Key}},
		)).Error().ToNot(HaveOccurred())
		Expect(svc.Channel.NewWriter(nil).Delete(ctx, ch.Key(), false)).To(Succeed())
		Eventually(driver).Should(BeClosed())
		expectGone(ctx, t.Key)
	})
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
			grant(ctx, access.ActionDelete, u.OntologyID(), t.OntologyID())
		}
		commands := streamCommands(ctx)
		start := time.Now()
		Expect(withTimeout(timeout).Delete(
			AuthedCtx(ctx, u), nil, apitask.DeleteRequest{Keys: keys},
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
			grant(ctx, access.ActionDelete, u.OntologyID(), t.OntologyID())
			stallFreeWrites(ctx)
			start := time.Now()
			Expect(withTimeout(timeout).Delete(
				AuthedCtx(ctx, u), nil, apitask.DeleteRequest{Keys: []task.Key{t.Key}},
			)).Error().ToNot(HaveOccurred())
			Expect(time.Since(start)).To(BeNumerically("<", 2*timeout))
			expectGone(ctx, t.Key)
		},
	)
	It("Should stop only the requested task", func(ctx SpecContext) {
		target := createRunningTask(ctx, true)
		other := createRunningTask(ctx, true)
		grant(ctx, access.ActionDelete, u.OntologyID(), target.OntologyID())
		forged := task.Status{
			Key:     target.OntologyID().String(),
			Variant: status.VariantSuccess,
			Message: "Task is running",
			Time:    telem.Now(),
			Details: task.StatusDetails{Task: other.Key, Running: true},
		}
		Expect(svc.Status.NewWriter(nil).Set(ctx, &forged)).To(Succeed())
		commands := streamCommands(ctx)
		Expect(withTimeout(100*time.Millisecond).Delete(
			AuthedCtx(ctx, u), nil, apitask.DeleteRequest{Keys: []task.Key{target.Key}},
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
			grant(ctx, access.ActionDelete, u.OntologyID(), t.OntologyID())
			reqCtx, cancel := context.WithCancel(ctx)
			defer time.AfterFunc(after, cancel).Stop()
			if after == 0 {
				cancel()
			}
			Expect(withTimeout(10*time.Second).Delete(
				AuthedCtx(
					reqCtx,
					u,
				), nil, apitask.DeleteRequest{Keys: []task.Key{t.Key}},
			),
			).Error().To(MatchError(context.Canceled))
			Expect(retrieveTask(ctx, t.Key).Key).To(Equal(t.Key))
		},
		Entry("before the stop", time.Duration(0)),
		Entry("during the wait", 50*time.Millisecond),
	)
	DescribeTable(
		"Should not wait for a task that is not running",
		func(ctx SpecContext, create func(SpecContext) task.Task) {
			t := create(ctx)
			grant(ctx, access.ActionDelete, u.OntologyID(), t.OntologyID())
			commands := streamCommands(ctx)
			start := time.Now()
			Expect(withTimeout(10*time.Second).Delete(
				AuthedCtx(ctx, u), nil, apitask.DeleteRequest{Keys: []task.Key{t.Key}},
			)).Error().ToNot(HaveOccurred())
			Expect(time.Since(start)).To(BeNumerically("<", time.Second))
			Consistently(commands, 100*time.Millisecond).ShouldNot(Receive())
			expectGone(ctx, t.Key)
		},
		Entry("never started", func(ctx SpecContext) task.Task {
			t := createTask(ctx)
			Expect(svc.Status.NewWriter(nil).Delete(
				ctx, t.OntologyID().String(),
			)).To(Succeed())
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
		commands := streamCommands(ctx)
		Expect(apiSvc.Delete(
			AuthedCtx(ctx, u), nil, apitask.DeleteRequest{Keys: []task.Key{t.Key}},
		)).Error().To(MatchError(access.ErrDenied))
		Consistently(commands, 100*time.Millisecond).ShouldNot(Receive())
		Expect(retrieveTask(ctx, t.Key).Key).To(Equal(t.Key))
	})
})
