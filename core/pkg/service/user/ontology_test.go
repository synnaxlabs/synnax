// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package user_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/zyn"
)

var _ = Describe("Ontology", func() {
	Describe("Schema", func() {
		It("Should return the ontology schema", func(_ SpecContext) {
			schema := svc.Schema().Shape()
			Expect(schema.DataType()).To(Equal(zyn.ObjectT))
			fields := schema.Fields()
			Expect(fields).To(HaveKey("key"))
			Expect(fields).To(HaveKey("username"))
		})
	})
	Describe("RetrieveResource", func() {
		It("Should retrieve a user's schema entity by its key", func(ctx SpecContext) {
			key := uuid.New()
			created := MustSucceed(svc.NewWriter(nil).Create(ctx, user.User{
				Username: uuid.NewString(),
				Key:      key,
			}))
			resource := MustSucceed(svc.RetrieveResource(ctx, key.String(), nil))
			var resU user.User
			Expect(resource.Parse(&resU)).To(Succeed())
			Expect(resU).To(Equal(created))
		})
	})
})
