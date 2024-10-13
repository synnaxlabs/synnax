// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package log_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/workspace/log"
	"github.com/synnaxlabs/x/gorp"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Writer", func() {
	Describe("Create", func() {
		It("Should create a Log", func() {
			log := log.Log{
				Name: "test",
				Data: "data",
			}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &log)).To(Succeed())
			Expect(log.Key).ToNot(Equal(uuid.Nil))
		})
	})
	Describe("Update", func() {
		It("Should rename a Log", func() {
			s := log.Log{Name: "test", Data: "data"}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &s)).To(Succeed())
			Expect(svc.NewWriter(tx).Rename(ctx, s.Key, "test2")).To(Succeed())
			var res log.Log
			Expect(gorp.NewRetrieve[uuid.UUID, log.Log]().WhereKeys(s.Key).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Name).To(Equal("test2"))
		})
	})
	Describe("SetData", func() {
		It("Should set the data of a Log", func() {
			s := log.Log{Name: "test", Data: "data"}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &s)).To(Succeed())
			Expect(svc.NewWriter(tx).SetData(ctx, s.Key, "data2")).To(Succeed())
			var res log.Log
			Expect(gorp.NewRetrieve[uuid.UUID, log.Log]().WhereKeys(s.Key).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Data).To(Equal("data2"))
		})
	})

	Describe("Copy", func() {
		It("Should copy a Log with a new name under the same workspace", func() {
			s := log.Log{Name: "test", Data: "data"}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &s)).To(Succeed())
			var cpy log.Log
			Expect(svc.NewWriter(tx).Copy(ctx, s.Key, "test2", false, &cpy)).To(Succeed())
			Expect(cpy.Key).ToNot(Equal(s.Key))
			Expect(cpy.Name).To(Equal("test2"))
			var res []ontology.Resource
			Expect(otg.NewRetrieve().WhereIDs(ws.OntologyID()).TraverseTo(ontology.Children).Entries(&res).Exec(ctx, tx)).To(Succeed())
			keys := lo.Map(res, func(r ontology.Resource, _ int) string { return r.ID.Key })
			Expect(keys).To(ContainElement(cpy.Key.String()))
		})
		It("Should copy a Log into a snapshot that cannot be modified", func() {
			s := log.Log{Name: "test", Data: "data"}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &s)).To(Succeed())
			var cpy log.Log
			Expect(svc.NewWriter(tx).Copy(ctx, s.Key, "test2", true, &cpy)).To(Succeed())
			Expect(svc.NewWriter(tx).SetData(ctx, cpy.Key, "data2")).To(HaveOccurredAs(validate.Error))
		})
	})
})
