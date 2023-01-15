// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package aspen_test

import (
	"context"
	"github.com/synnaxlabs/aspen"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/alamos"
	"go.uber.org/zap"
	"os"
)

var _ = Describe("StreamServer", func() {
	var (
		db1    aspen.DB
		db2    aspen.DB
		logger *zap.SugaredLogger
		exp    alamos.Experiment
	)
	BeforeEach(func() {
		log := zap.NewNop()
		logger = log.Sugar()
		exp = alamos.New("aspen_join_test")
		var err error
		db1, err = aspen.Open(
			context.TODO(),
			"",
			"localhost:22646",
			[]address.Address{},
			aspen.Bootstrap(),
			aspen.WithLogger(logger),
			aspen.WithExperiment(alamos.Sub(exp, "db1")),
			aspen.MemBacked(),
			aspen.WithPropagationConfig(aspen.FastPropagationConfig),
		)
		Expect(err).ToNot(HaveOccurred())
		db2, err = aspen.Open(
			context.TODO(),
			"",
			"localhost:22647",
			[]address.Address{"localhost:22646"},
			aspen.WithLogger(logger),
			aspen.WithExperiment(alamos.Sub(exp, "db2")),
			aspen.MemBacked(),
			aspen.WithPropagationConfig(aspen.FastPropagationConfig),
		)
		Expect(err).ToNot(HaveOccurred())
	})
	AfterEach(func() {
		Expect(db1.Close()).To(Succeed())
		Expect(db2.Close()).To(Succeed())
		f, err := os.Create("aspen_join_test_report.json")
		defer func() {
			Expect(f.Close()).To(Succeed())
		}()
		Expect(err).ToNot(HaveOccurred())
		Expect(exp.Report().WriteJSON(f)).To(Succeed())
	})
	It("Should be able to join two clusters", func() {
		Eventually(db1.Nodes).Should(HaveLen(2))
		Eventually(db2.Nodes).Should(HaveLen(2))
	})
})
