// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil

import "github.com/onsi/gomega"

func MustSucceed[T any](value T, err error) T {
	return MustSucceedWithOffset[T](1)(value, err)
}

func MustSucceedWithOffset[T any](offset int) func(value T, err error) T {
	return func(value T, err error) T {
		gomega.ExpectWithOffset(offset+1, err).ToNot(gomega.HaveOccurred())
		return value
	}
}

func MustSucceed2[A, B any](a A, b B, err error) (A, B) {
	gomega.ExpectWithOffset(1, err).ToNot(gomega.HaveOccurred())
	return a, b
}

func MustSucceed3[A, B, C any](a A, b B, c C, err error) (A, B, C) {
	gomega.ExpectWithOffset(1, err).ToNot(gomega.HaveOccurred())
	return a, b, c
}

func MustSucceed4[A, B, C, D any](a A, b B, c C, d D, err error) (A, B, C, D) {
	gomega.ExpectWithOffset(1, err).ToNot(gomega.HaveOccurred())
	return a, b, c, d
}

func MustBeOk[T any](value T, ok bool) T {
	return MustBeOkWithOffset[T](1)(value, ok)
}

func MustBeOkWithOffset[T any](offset int) func(value T, ok bool) T {
	return func(value T, ok bool) T {
		gomega.ExpectWithOffset(offset+1, ok).To(gomega.BeTrue())
		return value
	}
}
