// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package telem

import (
	"bytes"
	"fmt"

	"github.com/onsi/gomega/types"
)

// SeriesMatcherOption configures the behavior of the series matcher
type SeriesMatcherOption func(*seriesMatcher)

// ExcludeSeriesFields returns an option that configures the series matcher to exclude
// specific fields from comparison. Valid fields are "DataType", "TimeRange",
// "Alignment", and "Data".
func ExcludeSeriesFields(fields ...string) SeriesMatcherOption {
	return func(m *seriesMatcher) {
		m.excludedFields = make(map[string]bool)
		for _, field := range fields {
			m.excludedFields[field] = true
		}
	}
}

type seriesMatcher struct {
	expected       Series
	excludedFields map[string]bool
}

// MatchSeries returns a Gomega matcher that compares two Series for equality.
// Two series are considered equal if they have the same:
// - DataType (unless excluded)
// - TimeRange (unless excluded)
// - Data contents (unless excluded)
// - Alignment (unless excluded)
func MatchSeries(expected Series, opts ...SeriesMatcherOption) types.GomegaMatcher {
	m := &seriesMatcher{
		expected:       expected,
		excludedFields: make(map[string]bool),
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

func MatchWrittenSeries(expected Series, opts ...SeriesMatcherOption) types.GomegaMatcher {
	m := &seriesMatcher{
		expected:       expected,
		excludedFields: map[string]bool{"TimeRange": true, "Alignment": true},
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// MatchSeriesData returns a matcher that only compares the data contents of two Series,
// ignoring DataType, TimeRange, and Alignment.
func MatchSeriesData(expected Series) types.GomegaMatcher {
	return MatchSeries(expected, ExcludeSeriesFields("DataType", "TimeRange", "Alignment"))
}

func (m *seriesMatcher) Match(actual interface{}) (success bool, err error) {
	actualSeries, ok := actual.(Series)
	if !ok {
		return false, fmt.Errorf("MatchSeries matcher expects a Series but got %K", actual)
	}

	// Check data type
	if !m.excludedFields["DataType"] && actualSeries.DataType != m.expected.DataType {
		return false, nil
	}

	// Check time range
	if !m.excludedFields["TimeRange"] && actualSeries.TimeRange != m.expected.TimeRange {
		return false, nil
	}

	// Check alignment
	if !m.excludedFields["Alignment"] && actualSeries.Alignment != m.expected.Alignment {
		return false, nil
	}

	// Check data contents
	if !m.excludedFields["Data"] && !bytes.Equal(actualSeries.Data, m.expected.Data) {
		return false, nil
	}

	return true, nil
}

func (m *seriesMatcher) FailureMessage(actual interface{}) string {
	actualSeries, ok := actual.(Series)
	if !ok {
		return fmt.Sprintf("Expected Series but got %K", actual)
	}

	var differences []string

	if !m.excludedFields["DataType"] && actualSeries.DataType != m.expected.DataType {
		differences = append(differences, fmt.Sprintf(
			"DataType:\n\tExpected: %v\n\tActual: %v",
			m.expected.DataType,
			actualSeries.DataType,
		))
	}

	if !m.excludedFields["TimeRange"] && actualSeries.TimeRange != m.expected.TimeRange {
		differences = append(differences, fmt.Sprintf(
			"TimeRange:\n\tExpected: %v\n\tActual: %v",
			m.expected.TimeRange,
			actualSeries.TimeRange,
		))
	}

	if !m.excludedFields["Alignment"] && actualSeries.Alignment != m.expected.Alignment {
		differences = append(differences, fmt.Sprintf(
			"Alignment:\n\tExpected: %v\n\tActual: %v",
			m.expected.Alignment,
			actualSeries.Alignment,
		))
	}

	if !m.excludedFields["Data"] && !bytes.Equal(actualSeries.Data, m.expected.Data) {
		differences = append(differences, fmt.Sprintf(
			"Data:\n\tExpected: %v\n\tActual: %v",
			m.expected.DataString(),
			actualSeries.DataString(),
		))
	}

	return fmt.Sprintf("Series did not match in the following ways:\n%s",
		formatDifferences(differences))
}

func (m *seriesMatcher) NegatedFailureMessage(actual interface{}) string {
	return fmt.Sprintf("Expected series not to match:\n\tActual: %v\n\tExpected: %v",
		actual, m.expected)
}

func formatDifferences(differences []string) string {
	var result string
	for i, diff := range differences {
		if i > 0 {
			result += "\n"
		}
		result += diff
	}
	return result
}
