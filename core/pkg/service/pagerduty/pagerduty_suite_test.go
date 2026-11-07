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
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	pd "github.com/synnaxlabs/synnax/pkg/service/pagerduty"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	secrettestutil "github.com/synnaxlabs/synnax/pkg/service/secret/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
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
	otg       *ontology.Ontology
	groupSvc  *group.Service
	statusSvc *status.Service
	secretSvc *secret.Service
	alertSvc  *pd.Service
)

// testRack is the rack that every stored task deploys to.
const testRack rack.Key = 1 << 16

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	db = DeferClose(gorp.Wrap(memkv.New()))
	otg = MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
	groupSvc = MustOpen(group.OpenService(ctx, group.ServiceConfig{
		DB: db, Ontology: otg,
	}))
	labelSvc := MustOpen(label.OpenService(ctx, label.ServiceConfig{
		DB: db, Ontology: otg, Group: groupSvc,
	}))
	statusSvc = MustOpen(status.OpenService(ctx, status.ServiceConfig{
		DB: db, Ontology: otg, Label: labelSvc, Group: groupSvc,
	}))
	secretSvc = secrettestutil.OpenService(ctx, db, otg, groupSvc, pd.FindSecretUses)
	alertSvc = MustOpen(pd.OpenService(ctx, pd.ServiceConfig{
		DB:     db,
		Secret: secretSvc,
	}))
})

// createRoutingKey creates a stored secret that holds value and returns its key.
func createRoutingKey(ctx context.Context, value string) secret.Key {
	GinkgoHelper()
	s := secret.Secret{Variant: secret.StoredSecret{
		Base: secret.Base{Name: "Routing key"},
	}}
	Expect(secretSvc.NewWriter(nil).Create(ctx, &s, []byte(value))).To(Succeed())
	return s.Base().Key
}

// replaceRoutingKey replaces the value of the secret with the given key.
func replaceRoutingKey(ctx context.Context, key secret.Key, value string) {
	GinkgoHelper()
	s := secret.Secret{Variant: secret.StoredSecret{
		Base: secret.Base{Key: key, Name: "Routing key"},
	}}
	Expect(secretSvc.NewWriter(nil).Create(ctx, &s, []byte(value))).To(Succeed())
}

// storeTask stores the config and the row of an alert task named name on testRack, as
// the task service does, and returns the task as the Driver receives it.
func storeTask(ctx context.Context, name string, cfg pd.TaskConfig) task.Task {
	GinkgoHelper()
	t := task.Task{Key: uuid.New(), Rack: testRack, Name: name, Type: pd.AlertTaskType}
	Expect(gorp.NewCreate[task.Key, task.Task]().Entry(&t).Exec(ctx, db)).
		To(Succeed())
	t.Config = MustSucceed(alertSvc.Alert.Write(
		ctx,
		nil,
		t.Key,
		t.Name,
		encodeConfig(cfg),
	))
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
}

var _ pd.EventSender = (*mockEventSender)(nil)

func newMockSender() *mockEventSender { return &mockEventSender{} }

func (m *mockEventSender) SendEvent(
	_ context.Context,
	event pagerduty.V2Event,
) (*pagerduty.V2EventResponse, error) {
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

func (m *mockEventSender) setError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = err
}
