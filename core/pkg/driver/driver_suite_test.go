// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package driver_test

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	mockFS     fs.FS
	mockBinDir string
)

func TestDriver(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Driver Suite")
}

var _ = ShouldNotLeakGoroutinesPerSpec()

// Process #1 compiles the mock Driver once and hands its directory to the others.
// Linking it is the suite's largest memory and disk cost, so it runs once rather than
// once per parallel process.
var _ = SynchronizedBeforeSuite(func() []byte {
	ShouldNotLeakGoroutines()
	dir := MustSucceed(os.MkdirTemp("", "mockdriver"))
	driverName := "driver"
	if runtime.GOOS == "windows" {
		driverName = "driver.exe"
	}
	cmd := exec.Command(
		"go", "build", "-o", filepath.Join(dir, driverName),
		"./testdata/mockdriver",
	)
	cmd.Dir = MustSucceed(os.Getwd())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	Expect(cmd.Run()).To(Succeed())
	return []byte(dir)
}, func(dir []byte) {
	mockBinDir = string(dir)
	mockFS = os.DirFS(mockBinDir)
	ShouldNotLeakGoroutines()
})

var _ = SynchronizedAfterSuite(func() {}, func() {
	Expect(os.RemoveAll(mockBinDir)).To(Succeed())
})
