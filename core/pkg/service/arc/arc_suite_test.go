// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc_test

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/arc/directory"
	arctask "github.com/synnaxlabs/synnax/pkg/service/arc/task"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	racktestutil "github.com/synnaxlabs/synnax/pkg/service/rack/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	taskconfig "github.com/synnaxlabs/synnax/pkg/service/task/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	timetestutil "github.com/synnaxlabs/x/time/testutil"
)

func TestArc(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Arc Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	db         *gorp.DB
	svc        *arc.Service
	groupSvc   *group.Service
	channelSvc *channel.Service
	taskSvc    *task.Service
	arcTaskSvc *arctask.Service
	statusSvc  *status.Service
	rackSvc    *rack.Service
	testRack   *rack.Rack
	imexSvc    *imex.Service
	arcTable   *arc.Table
	clock      *timetestutil.Clock
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node := testutil.NewNode(ctx)
	clock = timetestutil.NewClock(time.Now())
	db = node.DB
	groupSvc = MustOpen(group.OpenService(ctx, group.ServiceConfig{DB: db}))
	MustOpen(label.OpenService(ctx, label.ServiceConfig{
		DB: db,
	}))
	statusSvc = MustOpen(status.OpenService(ctx, status.ServiceConfig{
		DB: db,
	}))
	channelSvc = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
		Channel: node.Channel,
		DB:      node.DB,
		HostKey: node.Identity.HostKey,
		Status:  statusSvc,
	}))
	roleSvc := MustOpen(role.OpenService(ctx, role.ServiceConfig{
		DB: db,
	}))
	rackSvc = MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
		DB:                  db,
		HostKey:             1,
		Status:              statusSvc,
		HealthCheckInterval: 10 * telem.Millisecond,
		DriverRole: MustSucceed(
			racktestutil.CreateDriverRole(ctx, db, roleSvc),
		),
	}))
	imexSvc = imex.NewService()
	arcTable = MustOpen(arc.OpenTable(ctx, arc.TableConfig{DB: db}))
	arcTaskSvc = MustOpen(arctask.OpenService(ctx, arctask.ServiceConfig{
		DB: db,
	}))
	configs := MustSucceed(taskconfig.NewRegistry(arcTaskSvc.Stores()...))
	taskSvc = MustOpen(task.OpenService(ctx, task.ServiceConfig{
		DB:           db,
		Rack:         rackSvc,
		Status:       statusSvc,
		ImEx:         imexSvc,
		Configs:      configs,
		ImExExcluded: []string{arctask.Type},
	}))
	testRack = &rack.Rack{Name: "Test Rack"}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return rackSvc.NewWriter(tx).Create(ctx, testRack)
	})).To(Succeed())
	svc = MustOpen(arc.OpenService(ctx, arc.ServiceConfig{
		DB:                  db,
		Table:               arcTable,
		Channel:             channelSvc,
		Task:                taskSvc,
		MatchTasks:          arcTaskSvc.MatchArcs,
		Status:              statusSvc,
		ImEx:                imexSvc,
		TextSweepQuiescence: 5 * telem.Second,
		TextSweepThreshold:  1,
		TaskSyncDebounce:    5 * telem.Millisecond,
		Now: func() telem.TimeStamp {
			return telem.NewTimeStamp(clock.Now())
		},
	}))
})

// rootDirectory returns the directory that holds every Arc.
func rootDirectory(ctx SpecContext, tx gorp.Tx) directory.Directory {
	GinkgoHelper()
	var root directory.Directory
	Expect(svc.NewDirectoryRetrieve().
		Where(directory.MatchNested(false)).
		Entry(&root).
		Exec(ctx, tx)).To(Succeed())
	return root
}

// newAutomation returns an automation that holds b.
func newAutomation(b arc.Base) arc.Arc {
	return arc.Arc{Variant: arc.AutomationArc{Base: b}}
}

// newModule returns a module that holds b.
func newModule(b arc.Base) arc.Arc { return arc.Arc{Variant: arc.ModuleArc{Base: b}} }

// readAll is an LSP access read that lets the subject retrieve every record.
func readAll(context.Context) (func(resource.ID) bool, error) {
	return func(resource.ID) bool { return true }, nil
}

// noRename is an LSP rename for specs that never rename. It returns an error.
func noRename(context.Context, channel.Key, string) error {
	return errors.New("unexpected rename")
}

// lspConfig lets the LSP subject retrieve every record and rejects every rename.
var lspConfig = arc.LSPConfig{Allowed: readAll, Rename: noRename}

// newArc returns an Arc of type t that holds b.
func newArc(t arc.Type, b arc.Base) arc.Arc {
	if t == arc.TypeModule {
		return newModule(b)
	}
	return newAutomation(b)
}

// textOf returns the text of the text-mode Arc b, failing the spec otherwise.
func textOf(b arc.Base) text.Text {
	GinkgoHelper()
	src, ok := b.Source.Variant.(arc.TextSource)
	Expect(ok).To(BeTrue(), "expected a text source, got %T", b.Source.Variant)
	return src.Text
}

// graphOf returns the graph of the graph-mode Arc b, failing the spec otherwise.
func graphOf(b arc.Base) graph.Graph {
	GinkgoHelper()
	src, ok := b.Source.Variant.(arc.GraphSource)
	Expect(ok).To(BeTrue(), "expected a graph source, got %T", b.Source.Variant)
	return src.Graph
}

// dispatch runs Writer.Apply in a transaction of its own.
func dispatch(
	ctx context.Context,
	key arc.Key,
	dispatchKey string,
	acts []arc.Action,
) error {
	return db.WithTx(ctx, func(tx gorp.Tx) error {
		return svc.NewWriter(tx).WithDispatchKey(dispatchKey).Apply(ctx, key, acts...)
	})
}
