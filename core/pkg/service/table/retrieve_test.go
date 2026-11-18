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
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/table"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Retrieve", func() {
	var tx gorp.Tx
	BeforeEach(func() { tx = DeferClose(db.OpenTx()) })
	It("Should retrieve a Table", func(ctx SpecContext) {
		s := table.Table{
			Project: &proj.Key,
			Name:    "test",
			Rows:    []table.Row{row("r0", 30, textCfg("hello"), "a")},
			Columns: []table.Column{{Key: "c0", Size: 80}},
		}
		Expect(svc.NewWriter(tx).Create(ctx, &s)).To(Succeed())
		var res table.Table
		Expect(
			svc.NewRetrieve().Where(table.MatchKeys(s.Key)).Entry(&res).Exec(ctx, tx),
		).To(Succeed())
		Expect(res).To(Equal(s))
	})
})
