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
	"github.com/synnaxlabs/x/errors"
	xtypes "github.com/synnaxlabs/x/types"
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

// MatchWrittenSeries returns a Gomega matcher that compares two Series for equality,
// but excludes TimeRange and Alignment fields by default. This is useful when comparing
// Series that have been written to and read from storage, where these fields might differ
// but the actual data remains the same.
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
	return MatchSeries(expected, ExcludeSeriesFields("TimeRange", "Alignment"))
}

// MatchSeriesDataV is a generic variant of MatchSeriesData that creates a Series from the
// provided sample data and returns a matcher. This is a convenience function for testing
// when you want to directly provide data values instead of constructing a Series first.
func MatchSeriesDataV[T Sample](data ...T) types.GomegaMatcher {
	return MatchSeriesData(NewSeriesV(data...))
}

func (m *seriesMatcher) Match(actual any) (success bool, err error) {
	actualSeries, ok := actual.(Series)
	if !ok {
		return false, errors.Newf("MatchSeries matcher expects a Series but got %K", actual)
	}
	if !m.excludedFields["DataType"] && actualSeries.DataType != m.expected.DataType {
		return false, nil
	}
	if !m.excludedFields["TimeRange"] && actualSeries.TimeRange != m.expected.TimeRange {
		return false, nil
	}
	if !m.excludedFields["Alignment"] && actualSeries.Alignment != m.expected.Alignment {
		return false, nil
	}
	if !m.excludedFields["Data"] && !bytes.Equal(actualSeries.Data, m.expected.Data) {
		return false, nil
	}
	return true, nil
}

func (m *seriesMatcher) FailureMessage(actual any) string {
	actualSeries, ok := actual.(Series)
	if !ok {
		return fmt.Sprintf("Expected Series but got %K", actual)
	}
	var (
		differences    []string
		dataTypesEqual = actualSeries.DataType == m.expected.DataType
	)
	if !m.excludedFields["DataType"] && !dataTypesEqual {
		differences = append(differences, fmt.Sprintf(
			"DataType:\n\tExpected: %v\n\tActual: %v",
			m.expected.DataType,
			actualSeries.DataType,
		))
	}
	if !m.excludedFields["TimeRange"] && actualSeries.TimeRange != m.expected.TimeRange {
		differences = append(differences, fmt.Sprintf(
			"TimeRange:\n\tExpected: %s\n\tActual: %s",
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
	if dataTypesEqual && !m.excludedFields["Data"] && !bytes.Equal(actualSeries.Data, m.expected.Data) {
		differences = append(differences, fmt.Sprintf(
			"Data:\n\tExpected: %v\n\tActual: %v",
			m.expected.DataString(),
			actualSeries.DataString(),
		))
	}
	return fmt.Sprintf("Series did not match:\n%s",
		formatDifferences(differences))
}

func (m *seriesMatcher) NegatedFailureMessage(actual any) string {
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

type frameMatcher[K xtypes.SizedNumeric] struct {
	expected        Frame[K]
	matchSeriesOpts []SeriesMatcherOption
}

// MatchFrame returns a Gomega matcher that compares two Frame objects for equality.
// The matcher verifies that both frames have the same number of series and that each
// series matches its corresponding one in the expected frame. K must be a numeric type.
func MatchFrame[K xtypes.SizedNumeric](expected Frame[K], matchSeriesOpts ...SeriesMatcherOption) types.GomegaMatcher {
	return &frameMatcher[K]{expected: expected, matchSeriesOpts: matchSeriesOpts}
}

func MatchWrittenFrame[K xtypes.SizedNumeric](expected Frame[K], opts ...SeriesMatcherOption) types.GomegaMatcher {
	return &frameMatcher[K]{expected: expected, matchSeriesOpts: append([]SeriesMatcherOption{ExcludeSeriesFields("TimeRange", "Alignment")}, opts...)}
}

func (m *frameMatcher[K]) Match(actual any) (success bool, err error) {
	actualFrame, ok := actual.(Frame[K])
	if !ok {
		return false, errors.Newf("MatchFrame matcher expects a Frame but got %K", actual)
	}
	if actualFrame.Count() != m.expected.Count() {
		return false, nil
	}
	for k := range actualFrame.Keys() {
		decodedSeries := actualFrame.Get(k)
		originalSeries := m.expected.Get(k)
		if len(decodedSeries.Series) != len(originalSeries.Series) {
			return false, nil
		}
		for i, s := range decodedSeries.Series {
			matched, err := MatchSeries(originalSeries.Series[i], m.matchSeriesOpts...).Match(s)
			if !matched || err != nil {
				return matched, err
			}
		}
	}
	return true, nil
}

func (m *frameMatcher[K]) FailureMessage(actual any) string {
	actualFrame, ok := actual.(Frame[K])
	if !ok {
		return fmt.Sprintf("Expected Frame but got %K", actual)
	}
	if actualFrame.Count() != m.expected.Count() {
		return fmt.Sprintf("Frames have different counts: expected %d, got %d",
			m.expected.Count(), actualFrame.Count())
	}
	for k := range actualFrame.Keys() {
		decodedSeries := actualFrame.Get(k)
		originalSeries := m.expected.Get(k)
		for i, s := range decodedSeries.Series {
			m := MatchSeries(originalSeries.Series[i], m.matchSeriesOpts...)
			success, _ := m.Match(s)
			if !success {
				return m.FailureMessage(s)
			}
		}
	}
	return "Frames match"
}

func (m *frameMatcher[K]) NegatedFailureMessage(actual any) string {
	actualFrame, ok := actual.(Frame[K])
	if !ok {
		return fmt.Sprintf("Expected Frame but got %K", actual)
	}
	if actualFrame.Count() != m.expected.Count() {
		return fmt.Sprintf("Frames have different number of series: expected %d, got %d",
			m.expected.Count(), actualFrame.Count())
	}
	for k := range actualFrame.Keys() {
		decodedSeries := actualFrame.Get(k)
		originalSeries := m.expected.Get(k)
		for i, s := range decodedSeries.Series {
			m := MatchSeries(originalSeries.Series[i], m.matchSeriesOpts...)
			success, _ := m.Match(s)
			if success {
				return fmt.Sprintf("Frames match for key %v", k)
			}
		}
	}
	return "Frames do not match"
}
