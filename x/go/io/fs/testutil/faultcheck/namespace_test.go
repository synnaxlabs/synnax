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
	"syscall"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Namespace", func() {
	It("Should not move a directory into itself on a crash image", func() {
		fs := MustSucceed(fault.New(fault.Config{}))
		MustSucceed(fs.Sub("d/x"))
		MustSucceed(fs.Sub("e"))
		Expect(fs.SyncDir("")).To(Succeed())
		Expect(fs.SyncDir("d")).To(Succeed())
		Expect(fs.Rename("d/x", "e/x")).To(Succeed())
		Expect(fs.SyncDir("e")).To(Succeed())
		image := MustSucceed(fs.Crash(fault.CrashConfig{}))
		Expect(MustSucceed(image.Exists("d/x"))).To(BeTrue())
		Expect(MustSucceed(image.Exists("e/x"))).To(BeTrue())
		Expect(image.Rename("e/x", "d/x/y")).To(MatchError(syscall.EINVAL))
		Expect(MustSucceed(image.Exists("d/x/y/y/y/y"))).To(BeFalse())
	})
})
