// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package library_test

import (
	"context"
	"testing"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/library"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	taskconfig "github.com/synnaxlabs/synnax/pkg/service/task/config"
	"github.com/synnaxlabs/x/encoding/orc"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

func TestLibrary(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Library Suite")
}

var _ = ShouldNotLeakGoroutinesPerSpec()

// testTaskType is the type of the task config the suite stores. Its config embeds a
// library reference, as a bus integration's config does.
const testTaskType = "library_test"

type testTaskConfig struct {
	Key uuid.UUID `json:"key" msgpack:"key"`
	library.Reference
}

var _ gorp.Entry[uuid.UUID] = testTaskConfig{}

func (c testTaskConfig) GorpKey() uuid.UUID { return c.Key }

func (testTaskConfig) SetOptions() []any { return nil }

func (c testTaskConfig) EncodeOrc(w *orc.Writer) error {
	w.Write(c.Key[:])
	return c.Reference.EncodeOrc(w)
}

func (c *testTaskConfig) DecodeOrc(r *orc.Reader) error {
	if _, err := r.Read(c.Key[:]); err != nil {
		return err
	}
	return c.Reference.DecodeOrc(r)
}

var (
	db       *gorp.DB
	otg      *ontology.Ontology
	svc      *library.Service
	taskSvc  *task.Service
	imexSvc  *imex.Service
	testRack *rack.Rack
	tx       gorp.Tx
	// stamps counts the task configs the suite's config store has stamped.
	stamps int
)

var (
	_ = BeforeSuite(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		db = DeferClose(gorp.Wrap(memkv.New()))
		otg = MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
		searchIdx := MustOpen(search.OpenIndex())
		groupSvc := MustOpen(group.OpenService(ctx, group.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Search:   searchIdx,
		}))
		labelSvc := MustOpen(label.OpenService(ctx, label.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    groupSvc,
			Search:   searchIdx,
		}))
		statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    groupSvc,
			Label:    labelSvc,
			Search:   searchIdx,
		}))
		rackSvc := MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
			DB:                  db,
			Ontology:            otg,
			Group:               groupSvc,
			HostProvider:        mock.NewStaticHostProvider(1),
			Status:              statusSvc,
			HealthCheckInterval: 10 * telem.Millisecond,
			Search:              searchIdx,
		}))
		stamper := library.Stamper{DB: db, Ontology: otg}
		store := MustOpen(taskconfig.OpenService(
			ctx,
			taskconfig.ServiceConfig[testTaskConfig]{
				DB:   db,
				Type: testTaskType,
				SetEntryKey: func(c *testTaskConfig, key uuid.UUID) {
					c.Key = key
				},
				ResolveEntry: func(
					ctx context.Context,
					tx gorp.Tx,
					key uuid.UUID,
					c *testTaskConfig,
				) error {
					stamps++
					_, err := stamper.Stamp(ctx, tx, key, &c.Reference)
					return err
				},
			},
		))
		configs := MustSucceed(taskconfig.NewRegistry(store))
		imexSvc = imex.NewService()
		taskSvc = MustOpen(task.OpenService(ctx, task.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    groupSvc,
			Rack:     rackSvc,
			Status:   statusSvc,
			Search:   searchIdx,
			ImEx:     imexSvc,
			Configs:  configs,
		}))
		testRack = &rack.Rack{Name: "Test Rack"}
		Expect(rackSvc.NewWriter(nil).Create(ctx, testRack)).To(Succeed())
		svc = MustOpen(library.OpenService(ctx, library.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Task:     taskSvc,
			Search:   searchIdx,
			ImEx:     imexSvc,
		}))
		Expect(searchIdx.Initialize(ctx)).To(Succeed())
	})
	_ = BeforeEach(func() { tx = DeferClose(db.OpenTx()) })
)
