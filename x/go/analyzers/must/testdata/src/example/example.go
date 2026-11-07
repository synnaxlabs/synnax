// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package example // want "missing testutil import for MustSucceed"

func Expect(any) Assertion     { return Assertion{} }
func HaveOccurred() any        { return nil }
func Not(any) any              { return nil }
func Succeed() any             { return nil }
func Eventually(args ...any)   {}
func Consistently(args ...any) {}

type Assertion struct{}

func (Assertion) ToNot(any)  {}
func (Assertion) NotTo(any)  {}
func (Assertion) To(any)     {}
func (Assertion) Should(any) {}

func MatchError(any) any { return nil }
func Equal(any) any      { return nil }

type testErr struct{}

func (testErr) Error() string { return "" }

var errSentinel error = testErr{}

func returnsValErr() (int, error)            { return 0, nil }
func returnsErr() error                      { return nil }
func returnsTwoValErr() (int, string, error) { return 0, "", nil }
func returnsValBool() (int, bool)            { return 0, false }

func example() {
	var err error

	// Pattern A: result, err := f(); Expect(err).ToNot(HaveOccurred())
	result, err := returnsValErr() // want "can be replaced with MustSucceed"
	Expect(err).ToNot(HaveOccurred())
	_ = result

	// Pattern A with = instead of :=
	var x int
	x, err = returnsValErr() // want "can be replaced with MustSucceed"
	Expect(err).ToNot(HaveOccurred())
	_ = x

	// Pattern B: err only assignment
	err = returnsErr() // want "can be replaced with Expect"
	Expect(err).ToNot(HaveOccurred())

	// Pattern with NotTo variant
	r2, err := returnsValErr() // want "can be replaced with MustSucceed"
	Expect(err).NotTo(HaveOccurred())
	_ = r2

	// Pattern with To(Not(HaveOccurred()))
	r3, err := returnsValErr() // want "can be replaced with MustSucceed"
	Expect(err).To(Not(HaveOccurred()))
	_ = r3

	// Pattern C: three return values
	a, b, err := returnsTwoValErr() // want "can be replaced with MustSucceed2"
	Expect(err).ToNot(HaveOccurred())
	_, _ = a, b

	// Pattern D: _, err := f() → MustSucceed(f()) with no LHS
	_, err = returnsValErr() // want "can be replaced with MustSucceed"
	Expect(err).ToNot(HaveOccurred())

	// Pattern E: _, _, err := f() → MustSucceed2(f()) with no LHS
	_, _, err = returnsTwoValErr() // want "can be replaced with MustSucceed2"
	Expect(err).ToNot(HaveOccurred())

	// Pattern F: assignment and Expect(err) separated by an intervening statement
	// that does not reference err.
	t1, err := returnsValErr() // want "can be replaced with MustSucceed"
	_ = t1 + 1
	Expect(err).ToNot(HaveOccurred())

	// Pattern G: err-only assignment with an intervening statement.
	err = returnsErr() // want "can be replaced with Expect"
	_ = 42
	Expect(err).ToNot(HaveOccurred())

	// Pattern H: non-adjacent case where the Expect call is split across multiple
	// physical lines — the whole multi-line statement must be removed, not just
	// the first line.
	t2, err := returnsValErr() // want "can be replaced with MustSucceed"
	_ = t2 + 1
	Expect(err).
		ToNot(HaveOccurred())

	// Should NOT match: intervening statement reads err — removing the binding
	// would break that read.
	v2, err := returnsValErr()
	_ = err
	Expect(err).ToNot(HaveOccurred())
	_ = v2

	// Should NOT match: no preceding assignment
	Expect(err).ToNot(HaveOccurred())

	// Should NOT match: not an assignment to err
	val, err2 := returnsValErr()
	_ = val
	_ = err2

	// Should NOT match: inside Eventually callback
	Eventually(func() {
		v, err := returnsValErr()
		Expect(err).ToNot(HaveOccurred())
		_ = v
	})

	// Should NOT match: inside Consistently callback
	Consistently(func() {
		err = returnsErr()
		Expect(err).ToNot(HaveOccurred())
	})

	// Pattern I: err := f(); Expect(err).To(Succeed()) → Expect(f()).To(Succeed())
	err = returnsErr() // want "can be replaced with Expect"
	Expect(err).To(Succeed())

	// Pattern J: _, err := f(); Expect(err).To(MatchError(e)) →
	// Expect(f()).Error().To(MatchError(e))
	_, mErr := returnsValErr() // want "assert the error inline"
	Expect(mErr).To(MatchError(errSentinel))

	// Pattern K: err := f(); Expect(err).Should(Equal(e)) →
	// Expect(f()).Should(Equal(e))
	kErr := returnsErr() // want "assert the error inline"
	Expect(kErr).Should(Equal(errSentinel))

	// Should NOT match: a named result stays bound for a later assertion.
	v3, nErr := returnsValErr()
	Expect(nErr).To(MatchError(errSentinel))
	_ = v3

	// Should NOT match: the error is used after the assertion.
	_, uErr := returnsValErr()
	Expect(uErr).To(MatchError(errSentinel))
	_ = uErr

	// Should NOT match: the captured value is not an error.
	_, okv := returnsValBool()
	Expect(okv).To(Equal(true))

	// Should NOT match: Equal(nil) passes on a nil error.
	_, nilErr := returnsValErr()
	Expect(nilErr).To(Equal(nil))
}
