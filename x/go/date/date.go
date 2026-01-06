// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// The date package contains a few functions and variables that are helpful for
// working with dates.

package date

// DaysInMonth is a map that represents the number of days in each month for a
// non-leap year
var DaysInMonth = map[int]int{
	1: 31, 2: 28, 3: 31, 4: 30, 5: 31, 6: 30,
	7: 31, 8: 31, 9: 30, 10: 31, 11: 30, 12: 31,
}

// IsLeapYear returns true if year is a leap year and false if year is not.
func IsLeapYear(year int) bool {
	return (year%4 == 0 && year%100 != 0) || (year%400 == 0)
}

// DateExists return true if the given year, month, and day exists and
// false if the given year, month, and day do not exist.
func DateExists(year, month, day int) bool {
	// Check if the year is valid
	if year < 1 {
		return false
	}

	// Check if the month is valid
	if month < 1 || month > 12 {
		return false
	}

	// Check if the day is valid
	daysInMonth := DaysInMonth[month]
	if month == 2 && IsLeapYear(year) {
		daysInMonth = 29
	}
	return day >= 1 && day <= daysInMonth
}
