// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package icd_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	taskconfig "github.com/synnaxlabs/synnax/pkg/service/task/config"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/encoding/orc"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// testTaskType is the type of the task config the suite stores. Its config embeds an
// ICD reference and lists the messages it uses, as a bus integration's config does.
const testTaskType = "icd_test"

type testMessage struct {
	Message icd.MessageKey `json:"message" msgpack:"message"`
	Fields  []icd.FieldKey `json:"fields"  msgpack:"fields"`
}

type testTaskConfig struct {
	Key uuid.UUID `json:"key"      msgpack:"key"`
	icd.Reference
	Messages []testMessage `json:"messages" msgpack:"messages"`
}

var _ gorp.Entry[uuid.UUID] = testTaskConfig{}

func (c testTaskConfig) GorpKey() uuid.UUID { return c.Key }

func (testTaskConfig) SetOptions() []any { return nil }

func (c testTaskConfig) EncodeOrc(w *orc.Writer) error {
	w.Write(c.Key[:])
	if err := c.Reference.EncodeOrc(w); err != nil {
		return err
	}
	w.Uint32(uint32(len(c.Messages)))
	for _, m := range c.Messages {
		w.Write(m.Message[:])
		w.Uint32(uint32(len(m.Fields)))
		for _, f := range m.Fields {
			w.Write(f[:])
		}
	}
	return nil
}

func (c *testTaskConfig) DecodeOrc(r *orc.Reader) error {
	if _, err := r.Read(c.Key[:]); err != nil {
		return err
	}
	if err := c.Reference.DecodeOrc(r); err != nil {
		return err
	}
	n, err := r.Uint32()
	if err != nil {
		return err
	}
	c.Messages = make([]testMessage, n)
	for i := range c.Messages {
		m := &c.Messages[i]
		if _, err = r.Read(m.Message[:]); err != nil {
			return err
		}
		var fields uint32
		if fields, err = r.Uint32(); err != nil {
			return err
		}
		m.Fields = make([]icd.FieldKey, fields)
		for j := range m.Fields {
			if _, err = r.Read(m.Fields[j][:]); err != nil {
				return err
			}
		}
	}
	return nil
}

func testUses(config msgpack.EncodedJSON) ([]icd.Use, error) {
	var c testTaskConfig
	if err := config.Unmarshal(&c); err != nil {
		return nil, err
	}
	uses := make([]icd.Use, len(c.Messages))
	for i, m := range c.Messages {
		uses[i] = icd.Use{Message: m.Message, Fields: m.Fields}
	}
	return uses, nil
}

var (
	db       *gorp.DB
	otg      *ontology.Ontology
	svc      *icd.Service
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
		stamper := icd.Stamper{DB: db, Ontology: otg}
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
					messages := make([]icd.MessageKey, len(c.Messages))
					for i, m := range c.Messages {
						messages[i] = m.Message
					}
					_, err := stamper.Stamp(ctx, tx, key, &c.Reference, messages)
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
		svc = MustOpen(icd.OpenService(ctx, icd.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Task:     taskSvc,
			Uses:     testUses,
			Search:   searchIdx,
			ImEx:     imexSvc,
		}))
		Expect(searchIdx.Initialize(ctx)).To(Succeed())
	})
	_ = BeforeEach(func() { tx = DeferClose(db.OpenTx()) })
)
