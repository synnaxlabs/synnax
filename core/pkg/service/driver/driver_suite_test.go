// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package driver_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/search"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/gorp"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	dist         mock.Node
	db           *gorp.DB
	rackService  *rack.Service
	taskService  *task.Service
	channelSvc   *channel.Service
	framerSvc    *framer.Service
	statusSvc    *status.Service
	hostProvider = mock.NewStaticHostProvider(1)
)

func TestDriver(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Driver Suite")
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	dist = mock.MustOpenNode(ctx)
	db = dist.DB
	searchIdx := MustOpen(search.Open())
	labelSvc := MustOpen(label.OpenService(ctx, label.ServiceConfig{
		DB:       dist.DB,
		Ontology: dist.Ontology,
		Group:    dist.Group,
		Search:   searchIdx,
	}))
	statusSvc = MustOpen(status.OpenService(ctx, status.ServiceConfig{
		Ontology: dist.Ontology,
		DB:       dist.DB,
		Group:    dist.Group,
		Label:    labelSvc,
		Search:   searchIdx,
	}))
	rackService = MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
		DB:           dist.DB,
		Ontology:     dist.Ontology,
		Group:        dist.Group,
		HostProvider: hostProvider,
		Status:       statusSvc,
		Search:       searchIdx,
	}))
	channelSvc = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
		Channel:      dist.Channel,
		DB:           dist.DB,
		HostResolver: dist.Cluster,
		Ontology:     dist.Ontology,
		Group:        dist.Group,
		Search:       searchIdx,
	}))
	framerSvc = MustOpen(framer.OpenService(ctx, framer.ServiceConfig{
		Framer:       dist.Framer,
		Channel:      channelSvc,
		DB:           dist.DB,
		Status:       statusSvc,
		HostResolver: dist.Cluster,
	}))
	taskService = MustOpen(task.OpenService(ctx, task.ServiceConfig{
		DB:       dist.DB,
		Ontology: dist.Ontology,
		Group:    dist.Group,
		Rack:     rackService,
		Status:   statusSvc,
		Channel:  channelSvc,
		Search:   searchIdx,
	}))
})

// mockFactory is a test implementation of driver.Factory.
type mockFactory struct {
	configureFunc func(context.Context, task.Task) (driver.Task, error)
	name          string
}

func (f *mockFactory) ConfigureTask(
	ctx context.Context,
	t task.Task,
) (driver.Task, error) {
	if f.configureFunc != nil {
		return f.configureFunc(ctx, t)
	}
	return nil, driver.ErrTaskNotHandled
}

func (f *mockFactory) Name() string { return f.name }

// mockTask is a test implementation of driver.Task.
type mockTask struct {
	execFunc func(context.Context, task.Command) error
	stopFunc func() error
	key      task.Key
}

func (t *mockTask) Exec(ctx context.Context, cmd task.Command) error {
	if t.execFunc != nil {
		return t.execFunc(ctx, cmd)
	}
	return nil
}

func (t *mockTask) Stop() error {
	if t.stopFunc != nil {
		return t.stopFunc()
	}
	return nil
}
