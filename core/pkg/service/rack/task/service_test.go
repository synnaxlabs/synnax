// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package task_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/rack/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Service", func() {
	var svc *task.Service
	BeforeEach(func(ctx SpecContext) {
		svc = MustOpen(task.OpenService(ctx, task.ServiceConfig{
			DB: db,
		}))
	})

	Describe("Stores", func() {
		It("Should expose the rack status store", func() {
			types := []string{}
			for _, s := range svc.Stores() {
				types = append(types, s.Type())
			}
			Expect(types).To(ConsistOf("rack_status"))
		})
	})

	Describe("Write", func() {
		It("Should store an empty config under the given key", func(ctx SpecContext) {
			key := uuid.New()
			Expect(svc.Status.Write(ctx, nil, key, msgpack.EncodedJSON{})).To(Succeed())
			data := MustSucceed(svc.Status.Read(ctx, nil, key))
			Expect(data["key"]).To(Equal(key.String()))
		})
	})
})
