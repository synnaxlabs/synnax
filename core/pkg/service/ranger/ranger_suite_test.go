// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ranger_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

func TestRanger(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Ranger Suite", SuiteConfig())
}

var (
	db       *gorp.DB
	svc      *ranger.Service
	w        ranger.Writer
	otg      *ontology.Ontology
	tx       gorp.Tx
	labelSvc *label.Service
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	db = DeferClose(gorp.Wrap(memkv.New()))
	otg = MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
	g := MustOpen(group.OpenService(ctx, group.ServiceConfig{
		DB: db, Ontology: otg,
	}))
	labelSvc = MustOpen(label.OpenService(ctx, label.ServiceConfig{
		DB: db, Ontology: otg, Group: g,
	}))
	svc = MustOpen(ranger.OpenService(ctx, ranger.ServiceConfig{
		DB: db, Ontology: otg, Group: g, Label: labelSvc,
	}))
})

var _ = BeforeEach(func() {
	tx = DeferClose(db.OpenTx())
	w = svc.NewWriter(tx)
})

var _ = ShouldNotLeakGoroutinesPerSpec()
