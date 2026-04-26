// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package testutil provides shared test helpers for validating behavior across every
// available FS backend. Tests typically iterate over [FileSystems] in a Ginkgo
// table-style loop so that each spec runs once per backend. Every Factory in
// [FileSystems] schedules its own cleanup via [ginkgo.DeferCleanup], so callers
// receive an FS and never have to track a teardown function manually.
package testutil

import (
	"os"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/testutil"
)

// Factory constructs a fresh, isolated FS for a single test case. Any backing
// storage created by the factory is registered for cleanup via
// [ginkgo.DeferCleanup] at the call site's enclosing setup node, so the caller
// never needs to invoke a teardown function directly.
type Factory func() xfs.FS

// FileSystems maps FS implementation names to Factory constructors covering
// every backend test code needs to exercise (in-memory and on-disk). Each entry
// produces a fresh FS rooted at an isolated location and arranges for that
// location to be torn down when the surrounding Ginkgo node exits.
var FileSystems = map[string]Factory{
	"memFS": OpenMem,
	"osFS":  OpenOS,
}

// OpenMem returns a fresh in-memory FS rooted at "testdata". The rooting
// matches [OpenOS] so test expectations that include path segments behave
// identically across backends. No cleanup is needed; the FS is reclaimed by
// the garbage collector once references go out of scope.
func OpenMem() xfs.FS {
	return testutil.MustSucceed(xfs.NewMem().Sub("testdata"))
}

// OpenOS returns an on-disk FS rooted at a unique temporary directory and
// registers a [ginkgo.DeferCleanup] that removes the directory and all of its
// contents when the enclosing Ginkgo setup node exits.
func OpenOS() xfs.FS {
	dir := testutil.MustSucceed(os.MkdirTemp("", "testdata-*"))
	ginkgo.DeferCleanup(func() {
		gomega.Expect(os.RemoveAll(dir)).To(gomega.Succeed())
	})
	return testutil.MustSucceed(xfs.Default.Sub(dir))
}
