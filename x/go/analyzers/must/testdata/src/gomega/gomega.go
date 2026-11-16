// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gomega

func Expect(any) Assertion        { return Assertion{} }
func Eventually(any) Assertion    { return Assertion{} }
func Succeed() any                { return nil }
func MatchError(any) any          { return nil }
func returnsErr() error           { return nil }
func returnsValErr() (int, error) { return 0, nil }

type Assertion struct{}

func (Assertion) To(any)     {}
func (Assertion) Should(any) {}

type Gomega struct{}

func (Gomega) Expect(any) Assertion { return Assertion{} }

// A Gomega closure in a variable retries under Eventually, so the global Expect would
// fail at once instead.
func storedClosure() {
	check := func(g Gomega) {
		err := returnsErr()
		g.Expect(err).To(Succeed())
	}
	Eventually(check).Should(Succeed())
}

func storedMatchClosure() {
	check := func(g Gomega) {
		_, err := returnsValErr()
		g.Expect(err).To(MatchError("x"))
	}
	Eventually(check).Should(Succeed())
}
