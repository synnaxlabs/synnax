package testutil

import (
	. "github.com/onsi/gomega"
)

func MustSucceed[T any](value T, err error) T {
	Expect(err).ToNot(HaveOccurred())
	return value
}

func MustSucceed2[A, B any](a A, b B, err error) (A, B) {
	Expect(err).ToNot(HaveOccurred())
	return a, b
}
