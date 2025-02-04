// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil

import (
	"fmt"
	"github.com/onsi/gomega/types"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	xt "github.com/synnaxlabs/x/types"
	"slices"
)

func EqualUnmarshal[T xt.Numeric](expected []T) types.GomegaMatcher {
	return &equalAfterUnmarshalMatcher[T]{
		expected: expected,
	}
}

type equalAfterUnmarshalMatcher[T xt.Numeric] struct {
	expected           []T
	actualUnmarshalled []T
}

func (m *equalAfterUnmarshalMatcher[T]) Match(actual interface{}) (bool, error) {
	v, ok := actual.([]byte)
	if !ok {
		return false, errors.Newf("EqualUnmarshal matcher expects a byte slice (actual: %T)", actual)
	}
	expectedT := telem.NewDataType[T](m.expected[0])

	m.actualUnmarshalled = telem.UnmarshalSlice[T](v, expectedT)
	return slices.Equal(m.actualUnmarshalled, m.expected), nil
}

func (m *equalAfterUnmarshalMatcher[T]) FailureMessage(actual interface{}) string {
	return fmt.Sprintf(
		"Expected\n\t%#v\nto unmarshal to\n\t%#v\nbut it actually unmarshalled to \n\t%#v\n",
		actual,
		m.expected,
		m.actualUnmarshalled,
	)
}

func (m *equalAfterUnmarshalMatcher[T]) NegatedFailureMessage(actual interface{}) string {
	return fmt.Sprintf("Expected\n\t%#v\nnot to unmarshal to\n\t%#v\nbut it did", actual, m.expected)
}
