// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package exit_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/cmd/exit"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Code", func() {
	DescribeTable(
		"Should map an error to its exit code",
		func(err error, code int) { Expect(exit.Code(err)).To(Equal(code)) },
		Entry("no error", nil, 0),
		Entry(
			"refused data",
			errors.Mark(
				errors.New("Data was last opened by Core 0.61.0"),
				storage.ErrRefused,
			),
			exit.DataRefused,
		),
		Entry(
			"locked data",
			errors.Mark(errors.New("Failed to acquire lock"), storage.ErrLocked),
			exit.DataLocked,
		),
		Entry(
			"invalid config",
			errors.Mark(errors.New("unknown flag: --bogus"), validate.ErrValidation),
			exit.Config,
		),
		Entry("any other error", errors.New("disk full"), exit.Failure),
	)
})
