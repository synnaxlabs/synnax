// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil

import (
	"github.com/onsi/gomega/matchers"
	"github.com/onsi/gomega/types"
	"github.com/synnaxlabs/x/errors"
)

func HaveOccurredAs(err error) types.GomegaMatcher {
	return &haveOccurredAsMatcher{
		MatchErrorMatcher: matchers.MatchErrorMatcher{Expected: err},
	}
}

type haveOccurredAsMatcher struct {
	matchers.MatchErrorMatcher
}

func (m *haveOccurredAsMatcher) Match(actual any) (bool, error) {
	if actual == nil {
		if m.Expected == nil {
			return true, nil
		}
		return false, errors.Newf(`Expected error

%s

to have occurred, but received nil instead.
		`, m.Expected)
	}
	success, err := m.MatchErrorMatcher.Match(actual)
	if err != nil {
		return false, err
	}
	if !success {
		return errors.Is(actual.(error), m.Expected.(error)), nil
	}
	return true, nil
}
