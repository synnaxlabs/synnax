// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package secret_test

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Encryption key", func() {
	Describe("NewEncryptionKey", func() {
		It("Should return a different random key on each call", func() {
			a := secret.NewEncryptionKey()
			Expect(a).ToNot(Equal(secret.EncryptionKey{}))
			Expect(secret.NewEncryptionKey()).ToNot(Equal(a))
		})
	})

	Describe("ParseEncryptionKey", func() {
		It("Should decode a base64 key and ignore surrounding space", func() {
			k := secret.NewEncryptionKey()
			encoded := base64.StdEncoding.EncodeToString(k[:])
			Expect(secret.ParseEncryptionKey(" " + encoded + "\n")).To(Equal(k))
		})
		DescribeTable("Should reject an invalid key",
			func(s, msg string) {
				Expect(secret.ParseEncryptionKey(s)).Error().To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(msg)),
				))
			},
			Entry("not base64", "not base64!", "secrets key is not base64"),
			Entry(
				"too short",
				base64.StdEncoding.EncodeToString(make([]byte, 16)),
				"secrets key is 16 bytes, not 32",
			),
			Entry("empty", "", "secrets key is 0 bytes, not 32"),
			Entry(
				"all zeros",
				base64.StdEncoding.EncodeToString(make([]byte, 32)),
				"secrets key is all zeros",
			),
		)
	})

	Describe("LoadEncryptionKey", func() {
		var path string
		BeforeEach(func() {
			path = filepath.Join(GinkgoT().TempDir(), "secrets.key")
		})
		It("Should create a key file when none exists", func() {
			k := MustSucceed(secret.LoadEncryptionKey(path))
			Expect(k).ToNot(Equal(secret.EncryptionKey{}))
			Expect(path).To(BeAnExistingFile())
		})
		It("Should return the saved key on later calls", func() {
			k := MustSucceed(secret.LoadEncryptionKey(path))
			Expect(secret.LoadEncryptionKey(path)).To(Equal(k))
		})
		It("Should return the key of the caller that creates the file first", func() {
			var (
				wg   sync.WaitGroup
				keys = make([]secret.EncryptionKey, 16)
				errs = make([]error, len(keys))
			)
			for i := range keys {
				wg.Go(func() { keys[i], errs[i] = secret.LoadEncryptionKey(path) })
			}
			wg.Wait()
			Expect(errors.Join(errs...)).To(Succeed())
			saved := MustSucceed(secret.LoadEncryptionKey(path))
			Expect(keys).To(HaveEach(Equal(saved)))
			Expect(os.ReadDir(filepath.Dir(path))).To(HaveLen(1))
		})
		It("Should reject a file that holds an invalid key", func() {
			Expect(os.WriteFile(path, []byte("short"), xfs.UserRW)).To(Succeed())
			Expect(secret.LoadEncryptionKey(path)).Error().To(MatchError(
				ContainSubstring("invalid secrets key in " + path),
			))
		})
		It("Should return the error when the folder does not exist", func() {
			missing := filepath.Join(path, "nested", "secrets.key")
			Expect(secret.LoadEncryptionKey(missing)).Error().
				To(MatchError(os.ErrNotExist))
		})
	})
})
