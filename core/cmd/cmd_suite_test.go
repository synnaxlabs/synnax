// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cmd_test

import (
	"os"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/x/testutil"
)

var version string

var _ = BeforeSuite(func() {
	data := MustSucceed(os.ReadFile("../pkg/version/VERSION"))
	version = "Synnax " + strings.ReplaceAll(string(data), "\r\n", "\n")
})

func TestCmd(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cmd Suite")
}
