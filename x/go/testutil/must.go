// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

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

func MustSucceed3[A, B, C any](a A, b B, c C, err error) (A, B, C) {
	Expect(err).ToNot(HaveOccurred())
	return a, b, c
}

func MustSucceed4[A, B, C, D any](a A, b B, c C, d D, err error) (A, B, C, D) {
	Expect(err).ToNot(HaveOccurred())
	return a, b, c, d
}
