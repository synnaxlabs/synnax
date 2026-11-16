// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kept

func Expect(any) Assertion { return Assertion{} }
func Succeed() any         { return nil }
func MatchError(any) any   { return nil }

type Assertion struct{}

func (Assertion) To(any) {}

type testErr struct{}

func (testErr) Error() string { return "" }

var errSentinel error = testErr{}

func returnsValErr() (int, error) { return 0, nil }
func returnsErr() error           { return nil }

// Deleting the binding leaves err with no reads.
func declaredBefore() {
	var err error
	_, err = returnsValErr() // want "assert the error inline"
	Expect(err).To(MatchError(errSentinel))
}

// Deleting each binding leaves the other without a declaration or a read.
func laterAssignment() {
	err := returnsErr() // want "can be replaced with Expect"
	Expect(err).To(Succeed())
	err = returnsErr() // want "can be replaced with Expect"
	Expect(err).To(Succeed())
}
