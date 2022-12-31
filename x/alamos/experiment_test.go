// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package alamos_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/alamos"
)

var _ = Describe("experiment", func() {
	Describe("Creating a new experiment", func() {
		It("Should create the experiment without panicking", func() {
			Expect(func() { alamos.New("test") }).ToNot(Panic())
		})
	})
})
