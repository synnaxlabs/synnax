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
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	racktestutil "github.com/synnaxlabs/synnax/pkg/service/rack/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	taskconfig "github.com/synnaxlabs/synnax/pkg/service/task/config"
	"github.com/synnaxlabs/x/content"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/encoding/orc"
	"github.com/synnaxlabs/x/patch"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
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

func (c *testTaskConfig) Patch(cur patch.Cursor) error {
	switch cur.Segment() {
	case "key":
		return patch.Value(&c.Key, cur.Next())
	case "icd":
		return patch.OptionalValue(&c.ICD, cur.Next())
	case "icd_hash":
		return patch.Value(&c.ICDHash, cur.Next())
	case "messages":
		return patch.Value(&c.Messages, cur.Next())
	default:
		return cur.UnknownField()
	}
}

func (c *testTaskConfig) HashContent(h *content.Hash) {
	h.Bool(c.ICD != nil)
	if c.ICD != nil {
		h.Bytes((*c.ICD)[:])
	}
	h.String(c.ICDHash)
	h.Len(len(c.Messages))
	for _, m := range c.Messages {
		h.Bytes(m.Message[:])
		h.Len(len(m.Fields))
		for _, f := range m.Fields {
			h.Bytes(f[:])
		}
	}
}

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

// icdReference is the reference from a test config to the ICD it uses.
func icdReference() gorp.Reference[uuid.UUID, testTaskConfig] {
	return gorp.NewKeyReference(
		gorp.KeyReferenceConfig[uuid.UUID, testTaskConfig, icd.Key]{
			Name:     "icd",
			Table:    "icd",
			OnDelete: gorp.Restrict,
			Kind:     gorp.Use,
			Targets: func(c *testTaskConfig) []icd.Key {
				if c.ICD == nil {
					return nil
				}
				return []icd.Key{*c.ICD}
			},
			Detach: func(c *testTaskConfig, _ icd.Key) { c.ICD = nil },
		},
	)
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
	table    *icd.Table
	svc      *icd.Service
	taskSvc  *task.Service
	imexSvc  *imex.Service
	testRack rack.Key
	tx       gorp.Tx
)

var (
	_ = BeforeSuite(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		db = DeferClose(gorp.Wrap(memkv.New()))
		statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{DB: db}))
		roleSvc := MustOpen(role.OpenService(ctx, role.ServiceConfig{DB: db}))
		rackSvc := MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
			DB:         db,
			HostKey:    1,
			Status:     statusSvc,
			DriverRole: MustSucceed(racktestutil.CreateDriverRole(ctx, db, roleSvc)),
			// No Driver reports for the rack, so a check would overwrite task statuses.
			HealthCheckInterval: telem.Hour,
		}))
		testRack = rackSvc.EmbeddedKey
		table = MustOpen(icd.OpenTable(ctx, icd.TableConfig{DB: db}))
		store := MustOpen(taskconfig.OpenService(
			ctx,
			taskconfig.ServiceConfig[testTaskConfig]{
				DB:   db,
				Type: testTaskType,
				SetEntryKey: func(c *testTaskConfig, key uuid.UUID) {
					c.Key = key
				},
				PatchEntry: (*testTaskConfig).Patch,
				HashEntry:  (*testTaskConfig).HashContent,
				References: []gorp.Reference[uuid.UUID, testTaskConfig]{
					icdReference(),
				},
				Ladder: imex.Ladder[testTaskConfig]{
					Decode: func(
						_ context.Context,
						version imex.Version,
						data msgpack.EncodedJSON,
					) (testTaskConfig, error) {
						return imex.DecodeBlob[testTaskConfig](
							data,
							testTaskType,
							version,
						)
					},
				},
				ResolveEntry: func(
					ctx context.Context,
					snap gorp.Snapshot,
					c *testTaskConfig,
				) error {
					messages := make([]icd.MessageKey, len(c.Messages))
					for i, m := range c.Messages {
						messages[i] = m.Message
					}
					_, err := icd.Stamp(ctx, snap, &c.Reference, messages)
					return err
				},
			},
		))
		imexSvc = imex.NewService()
		taskSvc = MustOpen(task.OpenService(ctx, task.ServiceConfig{
			DB:        db,
			Rack:      rackSvc,
			Status:    statusSvc,
			ImEx:      imexSvc,
			Configs:   MustSucceed(taskconfig.NewRegistry(store)),
			MatchICDs: table.MatchTasks,
		}))
		svc = MustOpen(icd.OpenService(ctx, icd.ServiceConfig{
			Table: table,
			Task:  taskSvc,
			Uses:  testUses,
			ImEx:  imexSvc,
		}))
	})
	_ = BeforeEach(func() { tx = DeferClose(db.OpenTx()) })
)
