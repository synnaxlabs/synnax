// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package log_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/log"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Log size check", func() {
	It("Should refuse to start a log of 3 MiB", func() {
		fs := xfs.NewMem()
		Expect(log.Prepare(fs, generation, 3<<20)).To(Succeed())
		Expect(log.Start(log.Config{
			FS:         fs,
			Executor:   goExec,
			OnDurable:  noDurable,
			OnRefused:  func([]log.Record, error) {},
			Prepare:    func(uint64) {},
			Switched:   func(uint64) {},
			DB:         db,
			Generation: generation,
			Sequence:   1,
		})).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(
				"log file of 3145728 bytes is not a whole number of blocks of at "+
					"least 4194304",
			)),
		))
	})
})
