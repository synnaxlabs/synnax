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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/workspace/table"
)

var _ = Describe("Retrieve", func() {
	It("Should retrieve a Table", func() {
		s := table.Table{Name: "test", Data: "data"}
		Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &s)).To(Succeed())
		var res table.Table
		Expect(svc.NewRetrieve().WhereKeys(s.Key).Entry(&res).Exec(ctx, tx)).To(Succeed())
		Expect(res).To(Equal(s))
	})
})
