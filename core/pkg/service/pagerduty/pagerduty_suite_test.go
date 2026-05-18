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
	"sync"
	"sync/atomic"
	"testing"

	"github.com/PagerDuty/go-pagerduty"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/search"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	pd "github.com/synnaxlabs/synnax/pkg/service/pagerduty"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
)

func TestPagerDuty(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "PagerDuty Suite")
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	db        *gorp.DB
	statusSvc *status.Service
)

var _ = BeforeSuite(func(ctx SpecContext) {
	db = DeferClose(gorp.Wrap(memkv.New()))
	otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
	searchIdx := MustOpen(search.Open())
	g := MustOpen(group.OpenService(ctx, group.ServiceConfig{
		DB: db, Ontology: otg, Search: searchIdx,
	}))
	labelSvc := MustOpen(label.OpenService(ctx, label.ServiceConfig{
		DB: db, Ontology: otg, Group: g, Search: searchIdx,
	}))
	statusSvc = MustOpen(status.OpenService(ctx, status.ServiceConfig{
		DB: db, Ontology: otg, Label: labelSvc, Group: g, Search: searchIdx,
	}))
	Expect(searchIdx.Initialize(ctx)).To(Succeed())
})

// mockEventSender records events sent through it for test assertions.
type mockEventSender struct {
	mu        sync.Mutex
	events    []pagerduty.V2Event
	err       error
	sendCalls atomic.Int32
}

var _ pd.EventSender = (*mockEventSender)(nil)

func newMockSender() *mockEventSender { return &mockEventSender{} }

func (m *mockEventSender) SendEvent(
	_ context.Context,
	event pagerduty.V2Event,
) (*pagerduty.V2EventResponse, error) {
	m.sendCalls.Add(1)
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

func (m *mockEventSender) sendCallCount() int32 { return m.sendCalls.Load() }

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
