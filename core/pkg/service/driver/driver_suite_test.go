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
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"

	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
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
	hostProvider = mock.StaticHostKeyProvider(1)
	ctx          context.Context
)

func TestDriver(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Driver Suite")
}

var _ = BeforeSuite(func() {
	ctx = context.Background()
	distB := mock.NewCluster()
	dist = distB.Provision(ctx)
	db = dist.DB
	labelSvc := MustSucceed(label.OpenService(
		ctx,
		label.ServiceConfig{
			DB:       dist.DB,
			Ontology: dist.Ontology,
			Group:    dist.Group,
		}),
	)
	statusSvc = MustSucceed(status.OpenService(
		ctx,
		status.ServiceConfig{
			Ontology: dist.Ontology,
			DB:       dist.DB,
			Group:    dist.Group,
			Label:    labelSvc,
		}),
	)
	rackService = MustSucceed(rack.OpenService(ctx, rack.ServiceConfig{
		DB:           dist.DB,
		Ontology:     dist.Ontology,
		Group:        dist.Group,
		HostProvider: mock.StaticHostKeyProvider(1),
		Status:       statusSvc,
	}))
	channelSvc = channel.Wrap(dist.Channel)
	framerSvc = dist.Framer
	taskService = MustSucceed(task.OpenService(ctx, task.ServiceConfig{
		DB:       dist.DB,
		Ontology: dist.Ontology,
		Group:    dist.Group,
		Rack:     rackService,
		Status:   statusSvc,
		Channel:  dist.Channel,
	}))
})

var _ = AfterSuite(func() {
	Expect(dist.Close()).To(Succeed())
})

// mockFactory is a test implementation of driver.Factory.
type mockFactory struct {
	configureFunc func(task.Task) (driver.Task, error)
	name          string
}

func (f *mockFactory) ConfigureTask(
	_ driver.Context,
	t task.Task,
) (driver.Task, error) {
	if f.configureFunc != nil {
		return f.configureFunc(t)
	}
	return nil, driver.ErrTaskNotHandled
}

func (f *mockFactory) Name() string { return f.name }

// mockTask is a test implementation of driver.Task.
type mockTask struct {
	execFunc func(cmd task.Command) error
	stopFunc func() error
	key      task.Key
}

func (t *mockTask) Exec(_ context.Context, cmd task.Command) error {
	if t.execFunc != nil {
		return t.execFunc(cmd)
	}
	return nil
}

func (t *mockTask) Stop() error {
	if t.stopFunc != nil {
		return t.stopFunc()
	}
	return nil
}
