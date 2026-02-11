// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:build driver

package driver_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/x/testutil"
)

var mockBinaryPath string

func TestDriver(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Driver Suite")
}

var _ = BeforeSuite(func() {
	tmpDir := GinkgoT().TempDir()
	mockBinaryPath = filepath.Join(tmpDir, "mockdriver")
	cmd := exec.Command(
		"go", "build", "-o", mockBinaryPath,
		"./testdata/mockdriver",
	)
	cmd.Dir = MustSucceed(os.Getwd())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	Expect(cmd.Run()).To(Succeed())
})
