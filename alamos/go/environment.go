// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package alamos

type Environment uint8

const (
	Bench Environment = iota + 1
	Debug
	Prod
)

type EnvironmentFilter func(env Environment, key string) bool

func ThresholdEnvFilter(level Environment) EnvironmentFilter {
	return func(l Environment, _ string) bool {
		return l >= level
	}
}
