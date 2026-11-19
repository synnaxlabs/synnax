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
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	. "github.com/synnaxlabs/x/testutil"
)

func TestFaultCheck(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "IO FS Fault Check Suite", SuiteConfig())
}

func read(fs xfs.FS, name string) string {
	GinkgoHelper()
	return string(MustSucceed(xfs.ReadFile(fs, name)))
}

func strict(fs *fault.FS) *fault.FS {
	GinkgoHelper()
	return MustSucceed(fs.Crash(fault.CrashConfig{}))
}

func create(fs xfs.FS, name string) xfs.File {
	GinkgoHelper()
	return MustSucceed(fs.Open(name, os.O_CREATE|os.O_RDWR))
}
