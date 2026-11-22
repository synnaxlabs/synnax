// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package http_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/http"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	ontology "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task/config"
	v2 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v2"
	v3 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v3"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Task migration", func() {
	It("Should move the fields of a Console-era task into an ICD named after the task",
		func(ctx SpecContext) {
			db := DeferClose(gorp.Wrap(memkv.New()))
			StartRun(db.Run)
			MustOpen(label.OpenService(ctx, label.ServiceConfig{DB: db}))
			statuses := MustOpen(status.OpenService(ctx, status.ServiceConfig{DB: db}))
			icds := MustOpen(icd.OpenService(ctx, icd.ServiceConfig{
				DB:   db,
				ImEx: imex.NewService(),
			}))
			svc := MustOpen(http.OpenService(ctx, http.ServiceConfig{
				DB:      db,
				Channel: channelSvc,
				ICD:     icds,
			}))
			t := v2.Task{
				Key:  uuid.New(),
				Name: "Telemetry",
				Type: "http_read",
				Config: msgpack.EncodedJSON{
					"rate": 2,
					"endpoints": []any{map[string]any{
						"key":    "ep",
						"method": "GET",
						"path":   "/telemetry",
						"index":  "ts",
						"fields": []any{map[string]any{
							"enabled":         true,
							"key":             "ts",
							"pointer":         "/ts",
							"dataType":        "timestamp",
							"timestampFormat": "unix_ms",
						}},
					}},
				},
			}
			Expect(gorp.NewCreate[v2.Key, v2.Task]().Entry(&t).Exec(ctx, db)).
				To(Succeed())
			Expect(ontology.DefineResources(ctx, db, resource.ID{
				Type: resource.TypeTask,
				Key:  t.Key.String(),
			})).To(Succeed())
			MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v2.Key, v2.Task]{
				DB:   db,
				Name: string(resource.TypeTask),
				Migrations: []gorp.Migration{v3.NewMigration(v3.MigrationConfig{
					Status:  statuses,
					Configs: MustSucceed(config.NewRegistry(svc.Stores()...)),
				})},
			}))
			stored := MustSucceed(svc.Read.Read(ctx, db, t.Key))
			Expect(stored).ToNot(HaveKey("inline"))
			var cfg http.ReadConfig
			Expect(stored.Unmarshal(&cfg)).To(Succeed())
			Expect(cfg.Endpoints).To(HaveLen(1))
			Expect(cfg.Endpoints[0].Fields).To(HaveLen(1))
			var ic icd.ICD
			Expect(icds.NewRetrieve().
				Where(icd.MatchKeys(*cfg.Endpoints[0].ICD)).
				Entry(&ic).
				Exec(ctx, db)).To(Succeed())
			Expect(ic.Name).To(Equal("Telemetry"))
			Expect(ic.Messages).To(HaveLen(1))
			Expect(ic.Messages[0].Name).To(Equal("GET /telemetry"))
		},
	)
})
