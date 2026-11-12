// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//go:build !windows

package secret_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("LoadEncryptionKey", func() {
	It("Should create the key file readable only by its owner", func() {
		path := filepath.Join(GinkgoT().TempDir(), "secrets.key")
		MustSucceed(secret.LoadEncryptionKey(path))
		Expect(MustSucceed(os.Stat(path)).Mode().Perm()).To(Equal(xfs.UserRW))
	})
})
