// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package status_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/search"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
)

func TestStatus(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Status Suite")
}

var (
	db       *gorp.DB
	otg      *ontology.Ontology
	groupSvc *group.Service
	labelSvc *label.Service
	statSvc  *status.Service
)

var _ = BeforeSuite(func(ctx SpecContext) {
	db = gorp.Wrap(memkv.New())
	otg = MustSucceed(ontology.Open(ctx, ontology.Config{
		DB: db,
	}))
	searchIdx := MustSucceed(search.Open())
	groupSvc = MustSucceed(group.OpenService(ctx, group.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Search:   searchIdx,
	}))
	labelSvc = MustSucceed(label.OpenService(ctx, label.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Group:    groupSvc,
		Search:   searchIdx,
	}))
	statSvc = MustSucceed(status.OpenService(ctx, status.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Group:    groupSvc,
		Label:    labelSvc,
		Search:   searchIdx,
	}))
})

var _ = AfterSuite(func(ctx SpecContext) {
	Expect(statSvc.Close()).To(Succeed())
	Expect(labelSvc.Close()).To(Succeed())
	Expect(groupSvc.Close()).To(Succeed())
	Expect(otg.Close()).To(Succeed())
	Expect(db.Close()).To(Succeed())
})
