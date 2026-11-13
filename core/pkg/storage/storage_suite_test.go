// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package storage_test

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/cockroachdb/pebble/v2/vfs"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/x/testutil"
)

// holdLockEnv names a data directory. A copy of the suite started with it holds the
// lock of that directory until its stdin closes, and runs no specs.
const holdLockEnv = "STORAGE_SPEC_HOLD_LOCK"

func TestMain(m *testing.M) {
	if dir := os.Getenv(holdLockEnv); dir != "" {
		os.Exit(holdLock(dir))
	}
	os.Exit(m.Run())
}

func holdLock(dir string) int {
	lock, err := vfs.Default.Lock(filepath.Join(dir, "LOCK"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Println("locked")
	if _, err = io.Copy(io.Discard, os.Stdin); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err = lock.Close(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func TestStorage(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Storage Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()
