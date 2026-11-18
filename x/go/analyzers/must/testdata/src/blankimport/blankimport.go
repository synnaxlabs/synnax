// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package blankimport

import (
	_ "github.com/synnaxlabs/x/testutil"
) // want "missing testutil import for MustSucceed"

func Expect(any) Assertion { return Assertion{} }
func HaveOccurred() any    { return nil }

type Assertion struct{}

func (a Assertion) ToNot(any) {}

func returnsValErr() (int, error) { return 0, nil }

// A blank import does not bring MustSucceed into scope, so the fix adds a dot import.
func example() {
	result, err := returnsValErr() // want "can be replaced with MustSucceed"
	Expect(err).ToNot(HaveOccurred())
	_ = result
}
