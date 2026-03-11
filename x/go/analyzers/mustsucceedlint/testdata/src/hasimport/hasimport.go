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
