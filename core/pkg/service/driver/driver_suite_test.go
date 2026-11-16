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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	calcgraph "github.com/synnaxlabs/synnax/pkg/service/channel/calculation/graph"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/pagerduty"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	racktestutil "github.com/synnaxlabs/synnax/pkg/service/rack/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	secrettestutil "github.com/synnaxlabs/synnax/pkg/service/secret/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	taskconfig "github.com/synnaxlabs/synnax/pkg/service/task/config"
	"github.com/synnaxlabs/synnax/pkg/storage"
	storagetestutil "github.com/synnaxlabs/synnax/pkg/storage/testutil"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/hookkv"
)

var (
	node testutil.Node
	// hookedKV stores the node, so a spec can intercept the commits of db.
	hookedKV    *hookkv.DB
	db          *gorp.DB
	rackService *rack.Service
	deviceSvc   *device.Service
	taskService *task.Service
	channelSvc  *channel.Service
	framerSvc   *framer.Service
	statusSvc   *status.Service
)

func TestDriver(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Driver Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	st := DeferClose(storagetestutil.OpenLayer(ctx))
	hookedKV = hookkv.Wrap(st.KV)
	node = testutil.NewNode(ctx, distribution.LayerConfig{
		Storage: &storage.Layer{KV: hookedKV, TS: st.TS},
	})
	db = node.DB
	MustOpen(label.OpenService(ctx, label.ServiceConfig{
		DB: node.DB,
	}))
	statusSvc = MustOpen(status.OpenService(ctx, status.ServiceConfig{
		DB: node.DB,
	}))
	roleSvc := MustOpen(role.OpenService(ctx, role.ServiceConfig{
		DB: node.DB,
	}))
	rackService = MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
		DB:         node.DB,
		HostKey:    1,
		Status:     statusSvc,
		DriverRole: MustSucceed(racktestutil.CreateDriverRole(ctx, db, roleSvc)),
		// No Driver reports for the rack, so a check would overwrite task statuses.
		HealthCheckInterval: telem.Hour,
	}))
	deviceSvc = MustOpen(device.OpenService(ctx, device.ServiceConfig{
		DB:     node.DB,
		Status: statusSvc,
		Rack:   rackService,
	}))
	channelSvc = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
		Channel: node.Channel,
		DB:      node.DB,
		HostKey: node.Identity.HostKey,
		Status:  statusSvc,
	}))
	channelGraph := MustOpen(calcgraph.Open(ctx, calcgraph.Config{
		DB:      node.DB,
		Channel: channelSvc,
		Status:  statusSvc,
	}))
	framerSvc = MustOpen(framer.OpenService(ctx, framer.ServiceConfig{
		DB:           node.DB,
		Framer:       node.Framer,
		Channel:      channelSvc,
		ChannelGraph: channelGraph,
	}))
	secrets := secrettestutil.OpenService(ctx, node.DB)
	for _, key := range []string{routingKey1, routingKey2} {
		s := secret.Secret{Variant: secret.StoredSecret{
			Base: secret.Base{Key: uuid.MustParse(key), Name: key},
		}}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return secrets.NewWriter(tx).Set(ctx, &s, []byte(key))
		})).To(Succeed())
	}
	pd := MustOpen(pagerduty.OpenService(ctx, pagerduty.ServiceConfig{
		DB:     node.DB,
		Secret: secrets,
	}))
	configs := MustSucceed(taskconfig.NewRegistry(pd.Stores()...))
	taskService = MustOpen(task.OpenService(ctx, task.ServiceConfig{
		DB:      node.DB,
		Rack:    rackService,
		Status:  statusSvc,
		Channel: channelSvc,
		ImEx:    imex.NewService(),
		Configs: configs,
	}))
})

// createTask stores t so that it can own a status. It skips task.Writer, which writes
// a status of its own, so a spec sees only the statuses the code under test writes.
func createTask(ctx context.Context, t task.Task) {
	GinkgoHelper()
	Expect(gorp.NewCreate[task.Key, task.Task]().Entry(&t).Exec(ctx, db)).
		To(Succeed())
	DeferCleanup(func(ctx SpecContext) {
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return taskService.NewWriter(tx).Delete(ctx, t.Key, true)
		})).To(Succeed())
	})
}

// retrieveStatus returns the status that owner holds, or the zero status when it holds
// none.
func retrieveStatus[D any](ctx context.Context, owner resource.ID) status.Status[D] {
	GinkgoHelper()
	var stat status.Status[D]
	Expect(errors.Skip(
		statusSvc.NewRetrieve[D]().
			Where(status.MatchOwners[D](owner)).
			Entry(&stat).
			Exec(ctx, db),
		query.ErrNotFound,
	)).To(Succeed())
	return stat
}

// mockFactory is a test implementation of driver.Factory.
type mockFactory struct {
	configureFunc func(context.Context, task.Task) (driver.Task, error)
	name          string
	// initialTasks are the tasks returned by InitialTasks.
	initialTasks []task.Task
	// cmdKey records the cmdKey passed to ConfigureTask per task.
	cmdKey sync.Map
}

func (f *mockFactory) ConfigureTask(
	ctx context.Context,
	t task.Task,
	cmdKey string,
) (driver.Task, error) {
	GinkgoHelper()
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
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return statusSvc.NewWriter(tx).Set(ctx, &task.Status{
			Owner:       new(t.ResourceID()),
			Name:        t.Name,
			Time:        telem.Now(),
			Variant:     status.VariantError,
			Message:     "Failed to configure task",
			Description: err.Error(),
			Details:     details,
		})
	})).To(Succeed())
}

func (f *mockFactory) InitialTasks() []task.Task { return f.initialTasks }

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
