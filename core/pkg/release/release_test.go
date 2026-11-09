// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package release_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/release"
	"github.com/synnaxlabs/synnax/pkg/release/testutil"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

// tauriPublicKey is a test key. The Tauri signer wrote testdata/file.sig with it.
const tauriPublicKey = "RWTHN9nA4VXGAgEjq/Lfm/g1xdDH9Z14A87x+XPsx++zKtO/IxXxRDh7"

var _ = Describe("Download", func() {
	var (
		files, dst string
		signer     testutil.Signer
		cfg        release.Config
	)
	BeforeEach(func() {
		dir := GinkgoT().TempDir()
		files = filepath.Join(dir, "files")
		dst = filepath.Join(dir, "dst")
		signer = testutil.NewSigner()
		server := httptest.NewServer(http.FileServer(http.Dir(files)))
		DeferCleanup(server.Close)
		cfg = release.Config{URL: server.URL, PublicKey: signer.PublicKey}
	})

	write := func(name string, data []byte) {
		GinkgoHelper()
		path := filepath.Join(files, "core", "v0.61.0", name)
		Expect(os.MkdirAll(filepath.Dir(path), 0o755)).To(Succeed())
		Expect(os.WriteFile(path, data, 0o644)).To(Succeed())
	}

	download := func(ctx context.Context) error {
		return release.Download(ctx, cfg, "core", "0.61.0", "synnax", dst)
	}

	It("Should download a file that matches its signature", func(ctx SpecContext) {
		write("synnax", []byte("core"))
		write("synnax.sig", signer.Sign([]byte("core")))
		Expect(download(ctx)).To(Succeed())
		Expect(os.ReadFile(dst)).To(Equal([]byte("core")))
		Expect(MustSucceed(os.Stat(dst)).Mode().Perm()).To(Equal(os.FileMode(0o600)))
	})

	It("Should check a signature that the Tauri signer wrote", func(ctx SpecContext) {
		write("synnax", MustSucceed(os.ReadFile("testdata/file")))
		write("synnax.sig", MustSucceed(os.ReadFile("testdata/file.sig")))
		cfg.PublicKey = tauriPublicKey
		Expect(download(ctx)).To(Succeed())
	})

	It("Should remove a file that does not match its signature", func(ctx SpecContext) {
		write("synnax", []byte("changed core"))
		write("synnax.sig", signer.Sign([]byte("core")))
		Expect(download(ctx)).To(MatchError(release.ErrSignature))
		Expect(dst).ToNot(BeAnExistingFile())
	})

	It("Should reject a signature from another key", func(ctx SpecContext) {
		write("synnax", []byte("core"))
		write("synnax.sig", testutil.NewSigner().Sign([]byte("core")))
		Expect(download(ctx)).To(MatchError(release.ErrSignature))
		Expect(dst).ToNot(BeAnExistingFile())
	})

	It("Should reject a signature that is not base64", func(ctx SpecContext) {
		write("synnax", []byte("core"))
		write("synnax.sig", []byte("not a signature"))
		Expect(download(ctx)).To(MatchError(release.ErrSignature))
		Expect(dst).ToNot(BeAnExistingFile())
	})

	It(
		"Should return ErrNotFound when the release has no file of the name",
		func(ctx SpecContext) {
			write("synnax.sig", signer.Sign([]byte("core")))
			err := download(ctx)
			Expect(err).To(MatchError(release.ErrNotFound))
			Expect(err).To(MatchError(query.ErrNotFound))
			Expect(dst).ToNot(BeAnExistingFile())
		},
	)

	It(
		"Should return ErrNotFound when the file has no signature",
		func(ctx SpecContext) {
			write("synnax", []byte("core"))
			Expect(download(ctx)).To(MatchError(release.ErrNotFound))
			Expect(dst).ToNot(BeAnExistingFile())
		},
	)

	It("Should reject an invalid public key", func(ctx SpecContext) {
		cfg.PublicKey = "not a key"
		Expect(download(ctx)).To(MatchError(ContainSubstring(
			"invalid release public key",
		)))
	})
})
