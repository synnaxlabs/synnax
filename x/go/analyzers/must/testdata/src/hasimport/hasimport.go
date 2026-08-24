// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package hasimport

import (
	. "github.com/synnaxlabs/x/testutil"
)

func Expect(any) Assertion { return Assertion{} }
func HaveOccurred() any    { return nil }

type Assertion struct{}

func (a Assertion) ToNot(any) {}

func returnsValErr() (int, error) { return 0, nil }

// MustSucceed is already available via the dot import above, so the fix should NOT add
// a duplicate import.
func example() {
	result, err := returnsValErr() // want "can be replaced with MustSucceed"
	Expect(err).ToNot(HaveOccurred())
	_ = result
	_ = MustSucceed[int]
}
