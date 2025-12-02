// Copyright 2025 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/synnax/pkg/service/workspace/log"
	"github.com/synnaxlabs/x/gorp"
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
			l := log.Log{Name: "test", Data: "data"}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &l)).To(Succeed())
			Expect(svc.NewWriter(tx).Rename(ctx, l.Key, "test2")).To(Succeed())
			var res log.Log
			Expect(gorp.NewRetrieve[uuid.UUID, log.Log]().WhereKeys(l.Key).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Name).To(Equal("test2"))
		})
	})
	Describe("SetData", func() {
		It("Should set the data of a Log", func() {
			l := log.Log{Name: "test", Data: "data"}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &l)).To(Succeed())
			Expect(svc.NewWriter(tx).SetData(ctx, l.Key, "data2")).To(Succeed())
			var res log.Log
			Expect(gorp.NewRetrieve[uuid.UUID, log.Log]().WhereKeys(l.Key).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Data).To(Equal("data2"))
		})
	})
})
