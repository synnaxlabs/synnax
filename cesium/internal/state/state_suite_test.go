// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package state_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

func TestState(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "State Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

// refused matches the error of a log record or a head that breaks a rule of the state.
func refused() types.GomegaMatcher {
	return MatchError(ContainSubstring("refused by the state"))
}

// conflict matches the error of a writer open or a commit that reaches stored data or
// another writer's claim.
func conflict() types.GomegaMatcher {
	return SatisfyAll(
		MatchError(validate.ErrValidation),
		MatchError(ContainSubstring("write conflict")),
	)
}
