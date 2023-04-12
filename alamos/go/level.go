// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package alamos

type Level uint8

const (
	DebugLevel Level = iota + 1
	InfoLevel
)

type Filter func(level Level, key string) bool

func CompoundFilter(filters ...Filter) Filter {
	return func(level Level, key string) bool {
		for _, f := range filters {
			if f(level, key) {
				return true
			}
		}
		return false
	}
}

func ThresholdFilter(level Level) Filter {
	return func(l Level, _ string) bool {
		return l < level
	}
}
