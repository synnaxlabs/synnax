// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package faultcheck_test

import (
	"strings"
	"syscall"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Full disk", func() {
	It("Should count the bytes of a removed file that is still open", func() {
		fs := MustSucceed(fault.New(fault.Config{Capacity: 8192}))
		orphan := create(fs, "a.bin")
		Expect(orphan.Write([]byte(strings.Repeat("A", 4096)))).To(Equal(4096))
		Expect(fs.Remove("a.bin")).To(Succeed())
		other := create(fs, "b.bin")
		n, err := other.Write([]byte(strings.Repeat("B", 8192)))
		Expect(err).To(MatchError(syscall.ENOSPC))
		Expect(n).To(Equal(4096))
	})
})
