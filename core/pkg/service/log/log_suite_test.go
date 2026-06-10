// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package log_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/search"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
)

func TestLog(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Log Suite")
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	db  *gorp.DB
	otg *ontology.Ontology
	ws  project.Project
	svc *log.Service
	tx  gorp.Tx
)

var (
	_ = BeforeSuite(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		otg = MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
		var (
			searchIdx = MustOpen(search.Open())
			g         = MustOpen(group.OpenService(ctx, group.ServiceConfig{
				DB:       db,
				Ontology: otg,
				Search:   searchIdx,
			}))
			projectSvc = MustOpen(project.OpenService(ctx, project.ServiceConfig{
				DB:       db,
				Ontology: otg,
				Group:    g,
				Search:   searchIdx,
			}))
			userSvc = MustOpen(user.OpenService(ctx, user.ServiceConfig{
				DB:       db,
				Ontology: otg,
				Group:    g,
				Search:   searchIdx,
			}))
		)
		svc = MustOpen(log.OpenService(ctx, log.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Search:   searchIdx,
		}))
		author := MustSucceed(userSvc.NewWriter(nil).Create(ctx, user.User{
			Username: "test",
		}))
		ws.Author = author.Key
		Expect(projectSvc.NewWriter(nil).Create(ctx, &ws)).To(Succeed())
	})
	_ = BeforeEach(func() { tx = DeferClose(db.OpenTx()) })
)
