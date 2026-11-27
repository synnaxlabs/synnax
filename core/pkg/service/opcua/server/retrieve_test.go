// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package server_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/opcua/server"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Retrieve", func() {
	var (
		e       env
		tx      gorp.Tx
		breaker server.Server
	)
	BeforeEach(func(ctx SpecContext) {
		e = openEnv(ctx, db, nil)
		tx = DeferClose(db.OpenTx())
		for _, name := range []string{"Valve", "amber", "Breaker"} {
			s := server.Server{
				Rack:       e.racks.EmbeddedKey,
				Name:       name,
				Connection: newConnection("opc.tcp://plc:4840"),
			}
			Expect(e.svc.NewWriter(tx).Create(ctx, &s)).To(Succeed())
			breaker = s
		}
	})
	names := func(ctx SpecContext, q server.Retrieve) []string {
		GinkgoHelper()
		var res []server.Server
		Expect(q.Entries(&res).Exec(ctx, tx)).To(Succeed())
		return lo.Map(res, func(s server.Server, _ int) string { return s.Name })
	}

	DescribeTable(
		"Should walk the servers by name, ignoring case",
		func(ctx SpecContext, dir gorp.Direction, expected []string) {
			q := e.svc.NewRetrieve().OrderBy(server.OrderByName(dir))
			Expect(names(ctx, q)).To(Equal(expected))
		},
		Entry("ascending", gorp.DirectionAsc, []string{"amber", "Breaker", "Valve"}),
		Entry("descending", gorp.DirectionDesc, []string{"Valve", "Breaker", "amber"}),
	)

	DescribeTable(
		"Should start strictly past the name and key",
		func(ctx SpecContext, dir gorp.Direction, expected []string) {
			o := server.OrderByNameAfter(dir, breaker.Name, breaker.Key)
			Expect(names(ctx, e.svc.NewRetrieve().OrderBy(o))).To(Equal(expected))
		},
		Entry("ascending", gorp.DirectionAsc, []string{"Valve"}),
		Entry("descending", gorp.DirectionDesc, []string{"amber"}),
	)
})
