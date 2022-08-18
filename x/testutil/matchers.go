package testutil

import (
	"github.com/cockroachdb/errors"
	"github.com/onsi/gomega/matchers"
	"github.com/onsi/gomega/types"
)

func HaveOccurredAs(err error) types.GomegaMatcher {
	return &haveOccurredAsMatcher{
		MatchErrorMatcher: matchers.MatchErrorMatcher{Expected: err},
	}
}

type haveOccurredAsMatcher struct {
	matchers.MatchErrorMatcher
}

func (m *haveOccurredAsMatcher) Match(actual interface{}) (bool, error) {
	if actual == nil {
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
