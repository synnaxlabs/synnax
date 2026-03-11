package example // want "missing testutil import for MustSucceed"

func Expect(_ any) Assertion      { return Assertion{} }
func HaveOccurred() any           { return nil }
func Not(a any) any               { return a }
func Succeed() any                { return nil }

type Assertion struct{}

func (a Assertion) ToNot(_ any) {}
func (a Assertion) NotTo(_ any) {}
func (a Assertion) To(_ any)    {}

func returnsValErr() (int, error)            { return 0, nil }
func returnsErr() error                      { return nil }
func returnsTwoValErr() (int, string, error) { return 0, "", nil }

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

	// Should NOT match: no preceding assignment
	Expect(err).ToNot(HaveOccurred())

	// Should NOT match: not an assignment to err
	val, err2 := returnsValErr()
	_ = val
	_ = err2
}
