// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package slack_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/device"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	slk "github.com/synnaxlabs/synnax/pkg/service/slack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

func TestSlack(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Slack Suite")
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	db        *gorp.DB
	statusSvc *status.Service
	deviceSvc *device.Service
	rackSvc   *rack.Service
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	db = DeferClose(gorp.Wrap(memkv.New()))
	otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
	searchIdx := MustOpen(search.OpenIndex())
	g := MustOpen(group.OpenService(ctx, group.ServiceConfig{
		DB: db, Ontology: otg, Search: searchIdx,
	}))
	labelSvc := MustOpen(label.OpenService(ctx, label.ServiceConfig{
		DB: db, Ontology: otg, Group: g, Search: searchIdx,
	}))
	statusSvc = MustOpen(status.OpenService(ctx, status.ServiceConfig{
		DB: db, Ontology: otg, Label: labelSvc, Group: g, Search: searchIdx,
	}))
	rackSvc = MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
		DB: db, Ontology: otg, Group: g, Status: statusSvc, Search: searchIdx,
		HostProvider: mock.NewStaticHostProvider(1),
	}))
	deviceSvc = MustOpen(device.OpenService(ctx, device.ServiceConfig{
		DB: db, Ontology: otg, Group: g, Status: statusSvc, Rack: rackSvc,
		Search: searchIdx,
	}))
	Expect(searchIdx.Initialize(ctx)).To(Succeed())
})

// createDevice persists a Slack device with the given key and bot token, returning the
// device key.
func createDevice(ctx context.Context, key, botToken string) string {
	tx := db.OpenTx()
	dev := device.Device{
		Key:      key,
		Rack:     rackSvc.EmbeddedKey,
		Location: "workspace",
		Make:     "slack",
		Model:    "Slack workspace",
		Name:     "Test Workspace",
		Properties: msgpack.EncodedJSON{
			"bot_token": botToken,
		},
	}
	Expect(deviceSvc.NewWriter(tx).Create(ctx, &dev)).To(Succeed())
	Expect(tx.Commit(ctx)).To(Succeed())
	return dev.Key
}

// setStatus writes a status, driving the observable the Slack task subscribes to.
func setStatus(ctx context.Context, key string, variant status.Variant, name, message string) {
	stat := status.Status[any]{
		Key:     key,
		Name:    name,
		Variant: variant,
		Message: message,
		Time:    telem.Now(),
	}
	Expect(status.NewWriter[any](statusSvc, nil).Set(ctx, &stat)).To(Succeed())
}

// postCall records a single Sender.Post invocation.
type postCall struct {
	token string
	msg   slk.Message
}

// mockSender records posts for assertions and can be made to fail.
type mockSender struct {
	mu    sync.Mutex
	posts []postCall
	err   error
	calls atomic.Int32
}

var _ slk.Sender = (*mockSender)(nil)

func newMockSender() *mockSender { return &mockSender{} }

func (m *mockSender) Post(_ context.Context, token string, msg slk.Message) error {
	m.calls.Add(1)
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	m.posts = append(m.posts, postCall{token: token, msg: msg})
	return nil
}

func (m *mockSender) callCount() int32 { return m.calls.Load() }

func (m *mockSender) getPosts() []postCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]postCall, len(m.posts))
	copy(cp, m.posts)
	return cp
}

func (m *mockSender) setError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = err
}
