package testutil

import (
	. "github.com/onsi/gomega"
)

func MustSucceed[T any](value T, err error) T {
	Expect(err).ToNot(HaveOccurred())
	return value
}
