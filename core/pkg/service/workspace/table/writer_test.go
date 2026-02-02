// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package table_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/workspace/table"
	"github.com/synnaxlabs/x/gorp"
)

var _ = Describe("Writer", func() {
	Describe("Create", func() {
		It("Should create a Table", func() {
			t := table.Table{
				Name: "test",
				Data: map[string]any{"key": "data"},
			}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &t)).To(Succeed())
			Expect(t.Key).ToNot(Equal(uuid.Nil))
		})
	})
	Describe("Update", func() {
		It("Should rename a Table", func() {
			s := table.Table{Name: "test", Data: map[string]any{"key": "data"}}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &s)).To(Succeed())
			Expect(svc.NewWriter(tx).Rename(ctx, s.Key, "test2")).To(Succeed())
			var res table.Table
			Expect(gorp.NewRetrieve[uuid.UUID, table.Table]().WhereKeys(s.Key).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Name).To(Equal("test2"))
		})
	})
	Describe("SetData", func() {
		It("Should set the data of a Table", func() {
			s := table.Table{Name: "test", Data: map[string]any{"key": "data"}}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &s)).To(Succeed())
			Expect(svc.NewWriter(tx).SetData(ctx, s.Key, map[string]any{"key": "data2"})).To(Succeed())
			var res table.Table
			Expect(gorp.NewRetrieve[uuid.UUID, table.Table]().WhereKeys(s.Key).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Data["key"]).To(Equal("data2"))
		})
	})
})
