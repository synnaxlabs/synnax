// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package version

import (
	"fmt"
	"strconv"
	"strings"
)

// Semantic represents a semantic version string in the format "X.Y.Z"
type Semantic string

// compareSemanticOptions configures which parts of the semantic version to compare
type compareSemanticOptions struct {
	// checkMajor determines whether to validate major versions are equal
	checkMajor bool
	// checkMinor determines whether to validate minor versions are equal
	checkMinor bool
	// checkPatch determines whether to validate patch versions are equal
	checkPatch bool
}

type CompareSemanticOption func(*compareSemanticOptions)

func WithSkipMajor() CompareSemanticOption {
	return func(o *compareSemanticOptions) { o.checkMajor = false }
}

func WithSkipMinor() CompareSemanticOption {
	return func(o *compareSemanticOptions) { o.checkMinor = false }
}

func WithSkipPatch() CompareSemanticOption {
	return func(o *compareSemanticOptions) { o.checkPatch = false }
}

func ParseSemVer(version Semantic) (major, minor, patch int, err error) {
	parts := strings.Split(string(version), ".")
	if len(parts) != 3 {
		return 0, 0, 0, fmt.Errorf("invalid semver format: %s", version)
	}

	major, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid major version: %s", parts[0])
	}

	minor, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid minor version: %s", parts[1])
	}

	patch, err = strconv.Atoi(parts[2])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid patch version: %s", parts[2])
	}

	return major, minor, patch, nil
}

// CompareSemantic compares two semantic versions
// Returns:
//
//	-1 if a is older than b
//	 0 if versions are equal
//	 1 if a is newer than b
func CompareSemantic(a, b Semantic, opts ...CompareSemanticOption) (int, error) {
	options := &compareSemanticOptions{
		checkMajor: true,
		checkMinor: true,
		checkPatch: true,
	}
	for _, opt := range opts {
		opt(options)
	}
	aMajor, aMinor, aPatch, err := ParseSemVer(a)
	if err != nil {
		return 0, fmt.Errorf("parsing version a: %w", err)
	}

	bMajor, bMinor, bPatch, err := ParseSemVer(b)
	if err != nil {
		return 0, fmt.Errorf("parsing version b: %w", err)
	}

	if options.checkMajor {
		if aMajor < bMajor {
			return -1, nil
		}
		if aMajor > bMajor {
			return 1, nil
		}
	}

	if options.checkMinor {
		if aMinor < bMinor {
			return -1, nil
		}
		if aMinor > bMinor {
			return 1, nil
		}
	}

	if options.checkPatch {
		if aPatch < bPatch {
			return -1, nil
		}
		if aPatch > bPatch {
			return 1, nil
		}
	}

	return 0, nil
}
