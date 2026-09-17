// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package legacy_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/panel"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/x/gorp"
	. "github.com/synnaxlabs/x/testutil"
)

func TestLegacy(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Project Legacy Suite")
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	db       *gorp.DB
	otg      *ontology.Ontology
	groupSvc *group.Service
	panelSvc *panel.Service
	logSvc   *log.Service
	imexSvc  *imex.Service
	tx       gorp.Tx
)

var (
	_ = BeforeSuite(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		node := mock.NewNode(ctx)
		db = node.DB
		otg = MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
		searchIdx := MustOpen(search.OpenIndex())
		groupSvc = MustOpen(group.OpenService(ctx, group.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Search:   searchIdx,
		}))
		panelSvc = MustOpen(panel.OpenService(ctx, panel.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Search:   searchIdx,
		}))
		imexSvc = imex.NewService()
		logSvc = MustOpen(log.OpenService(ctx, log.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Search:   searchIdx,
			ImEx:     imexSvc,
		}))
	})
	_ = BeforeEach(func() { tx = DeferClose(db.OpenTx()) })
)
