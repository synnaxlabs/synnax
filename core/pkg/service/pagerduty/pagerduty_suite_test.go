// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pagerduty_test

import (
	"context"
	"encoding/json/v2"
	"sync"
	"testing"
	"uuid"

	"github.com/PagerDuty/go-pagerduty"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	pd "github.com/synnaxlabs/synnax/pkg/service/pagerduty"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	secrettestutil "github.com/synnaxlabs/synnax/pkg/service/secret/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

func TestPagerDuty(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service PagerDuty Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	db        *gorp.DB
	statusSvc *status.Service
	tasks     *gorp.Table[task.Key, task.Task]
	secretSvc *secret.Service
	alertSvc  *pd.Service
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	db = DeferClose(gorp.Wrap(memkv.New()))
	MustOpen(label.OpenService(ctx, label.ServiceConfig{
		DB: db,
	}))
	statusSvc = MustOpen(status.OpenService(ctx, status.ServiceConfig{
		DB: db,
	}))
	// Task statuses and configs reference their task, so the task table must be open.
	tasks = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[task.Key, task.Task]{
		DB:   db,
		Name: string(resource.TypeTask),
	}))
	secretSvc = secrettestutil.OpenService(ctx, db)
	alertSvc = MustOpen(pd.OpenService(ctx, pd.ServiceConfig{
		DB:     db,
		Secret: secretSvc,
	}))
})

func createTask(ctx context.Context, t task.Task) task.Task {
	GinkgoHelper()
	Expect(tasks.NewCreate().Entry(&t).Exec(ctx, db)).To(Succeed())
	return t
}

// createRoutingKey creates a stored secret that holds value and returns its key.
func createRoutingKey(ctx context.Context, value string) secret.Key {
	GinkgoHelper()
	s := secret.Secret{Variant: secret.StoredSecret{
		Base: secret.Base{Name: "Routing key"},
	}}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return secretSvc.NewWriter(tx).Set(ctx, &s, []byte(value))
	})).To(Succeed())
	return s.Base().Key
}

// replaceRoutingKey replaces the value of the secret with the given key.
func replaceRoutingKey(ctx context.Context, key secret.Key, value string) {
	GinkgoHelper()
	s := secret.Secret{Variant: secret.StoredSecret{
		Base: secret.Base{Key: key, Name: "Routing key"},
	}}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return secretSvc.NewWriter(tx).Set(ctx, &s, []byte(value))
	})).To(Succeed())
}

// storeTask stores the row and the config of an alert task named name, as the task
// service does, and returns the task as the Driver receives it. It creates each status
// that an alert watches and that does not exist yet.
func storeTask(ctx context.Context, name string, cfg pd.TaskConfig) task.Task {
	GinkgoHelper()
	for _, a := range cfg.Alerts {
		err := statusSvc.NewRetrieve[any]().
			Where(status.MatchKeys[any](a.Status)).
			Exec(ctx, db)
		if errors.Is(err, query.ErrNotFound) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return statusSvc.NewWriter(tx).Set(ctx, &status.Status[any]{
					Key:     a.Status,
					Name:    a.Status,
					Variant: status.VariantLoading,
					Time:    telem.Now(),
				})
			})).To(Succeed())
		} else {
			Expect(err).ToNot(HaveOccurred())
		}
	}
	t := createTask(ctx, task.Task{Key: uuid.New(), Name: name, Type: pd.AlertTaskType})
	rec := MustSucceed(
		alertSvc.Alert.Prepare(ctx, db, t.Key, t.Name, encodeConfig(cfg)),
	)
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return rec.Write(ctx, tx)
	})).To(Succeed())
	t.Config = MustSucceed(rec.Data())
	return t
}

// encodeConfig converts a config into the msgpack.EncodedJSON shape the driver
// receives as task.Task.Config.
func encodeConfig(cfg pd.TaskConfig) msgpack.EncodedJSON {
	GinkgoHelper()
	b := MustSucceed(json.Marshal(cfg))
	var m msgpack.EncodedJSON
	Expect(json.Unmarshal(b, &m)).To(Succeed())
	return m
}

// mockEventSender records events sent through it for test assertions.
type mockEventSender struct {
	mu     sync.Mutex
	events []pagerduty.V2Event
	err    error
	// held makes each send wait until it closes.
	held chan struct{}
}

var _ pd.EventSender = (*mockEventSender)(nil)

func newMockSender() *mockEventSender { return &mockEventSender{} }

func (m *mockEventSender) SendEvent(
	ctx context.Context,
	event pagerduty.V2Event,
) (*pagerduty.V2EventResponse, error) {
	m.mu.Lock()
	held := m.held
	m.mu.Unlock()
	if held != nil {
		select {
		case <-held:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return nil, m.err
	}
	m.events = append(m.events, event)
	return &pagerduty.V2EventResponse{
		Status:   "success",
		DedupKey: event.DedupKey,
		Message:  "Event processed",
	}, nil
}

func (m *mockEventSender) getEvents() []pagerduty.V2Event {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]pagerduty.V2Event, len(m.events))
	copy(cp, m.events)
	return cp
}

// hold makes sends wait until release runs.
func (m *mockEventSender) hold() (release func()) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.held = make(chan struct{})
	return func() { close(m.held) }
}

func (m *mockEventSender) setError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = err
}
