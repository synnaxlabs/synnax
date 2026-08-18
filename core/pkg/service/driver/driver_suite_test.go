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
	"sync"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/pagerduty"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	taskconfig "github.com/synnaxlabs/synnax/pkg/service/task/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	node         mock.Node
	db           *gorp.DB
	rackService  *rack.Service
	taskService  *task.Service
	taskWriter   task.Writer
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
	node = mock.NewNode(ctx)
	db = node.DB
	otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
	searchIdx := MustOpen(search.OpenIndex())
	groupSvc := MustOpen(group.OpenService(ctx, group.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Search:   searchIdx,
	}))
	labelSvc := MustOpen(label.OpenService(ctx, label.ServiceConfig{
		DB:       node.DB,
		Ontology: otg,
		Group:    groupSvc,
		Search:   searchIdx,
	}))
	statusSvc = MustOpen(status.OpenService(ctx, status.ServiceConfig{
		Ontology: otg,
		DB:       node.DB,
		Group:    groupSvc,
		Label:    labelSvc,
		Search:   searchIdx,
	}))
	rackService = MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
		DB:           node.DB,
		Ontology:     otg,
		Group:        groupSvc,
		HostProvider: hostProvider,
		Status:       statusSvc,
		Search:       searchIdx,
	}))
	channelSvc = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
		Channel:      node.Channel,
		DB:           node.DB,
		HostProvider: node.Cluster,
		Ontology:     otg,
		Group:        groupSvc,
		Search:       searchIdx,
		Status:       statusSvc,
	}))
	framerSvc = MustOpen(framer.OpenService(ctx, framer.ServiceConfig{
		Framer:  node.Framer,
		Channel: channelSvc,
		Status:  statusSvc,
	}))
	pd := MustOpen(pagerduty.OpenService(ctx, pagerduty.ServiceConfig{DB: node.DB}))
	configs := MustSucceed(taskconfig.NewRegistry(pd.Stores()...))
	taskService = MustOpen(task.OpenService(ctx, task.ServiceConfig{
		DB:       node.DB,
		Ontology: otg,
		Group:    groupSvc,
		Rack:     rackService,
		Status:   statusSvc,
		Channel:  channelSvc,
		Search:   searchIdx,
		ImEx:     imex.NewService(),
		Configs:  configs,
	}))
	taskWriter = taskService.NewWriter(nil)
})

// mockFactory is a test implementation of driver.Factory.
type mockFactory struct {
	configureFunc func(context.Context, task.Task) (driver.Task, error)
	name          string
	// cmdKey records the cmdKey passed to ConfigureTask per task.
	cmdKey sync.Map
}

func (f *mockFactory) ConfigureTask(
	ctx context.Context,
	t task.Task,
	cmdKey string,
) (driver.Task, error) {
	f.cmdKey.Store(t.Key, cmdKey)
	if f.configureFunc == nil {
		return nil, driver.ErrTaskNotHandled
	}
	tsk, err := f.configureFunc(ctx, t)
	// Real factories answer their own failures. Mirror that so specs exercise the
	// acknowledgment contract in driver.Factory.
	if err != nil && !errors.Is(err, driver.ErrTaskNotHandled) {
		writeConfigFailure(ctx, t, cmdKey, err)
	}
	return tsk, err
}

// writeConfigFailure writes the status a factory owes for a failed configure.
func writeConfigFailure(ctx context.Context, t task.Task, cmdKey string, err error) {
	GinkgoHelper()
	// A configure that ran out of time can't write anything, and nothing is left
	// waiting on it.
	if ctx.Err() != nil {
		return
	}
	details := task.NewStatusDetails(t, false)
	details.Cmd = cmdKey
	Expect(status.NewWriter[task.StatusDetails](statusSvc, nil).
		Set(ctx, &task.Status{
			Key:     t.OntologyID().String(),
			Name:    t.Name,
			Time:    telem.Now(),
			Variant: status.VariantError,
			Message: err.Error(),
			Details: details,
		})).To(Succeed())
}

func (f *mockFactory) Name() string { return f.name }

// mockTask is a test implementation of driver.Task.
type mockTask struct {
	execFunc func(context.Context, task.Command) error
	stopFunc func(sendStatus bool) error
	key      task.Key
}

func (t *mockTask) Exec(ctx context.Context, cmd task.Command) error {
	if t.execFunc != nil {
		return t.execFunc(ctx, cmd)
	}
	return nil
}

func (t *mockTask) Stop(sendStatus bool) error {
	if t.stopFunc != nil {
		return t.stopFunc(sendStatus)
	}
	return nil
}
