// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package unary_test

import (
	"context"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/testutil"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	ctx         = context.Background()
	rootPath    = "unary-testdata"
	fileSystems map[string]func() xfs.FS
	cleanUp     func() error
)

var _ = BeforeSuite(func() { fileSystems, cleanUp = testutil.FileSystems() })
var _ = AfterSuite(func() { Expect(cleanUp()).To(Succeed()) })

func TestUnary(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Unary Suite")
}
