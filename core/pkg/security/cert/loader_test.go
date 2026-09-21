// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cert_test

import (
	"encoding/pem"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/security/cert"
	"github.com/synnaxlabs/synnax/pkg/security/mock"
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

func countPEMBlocks(b []byte) int {
	count := 0
	for {
		block, rest := pem.Decode(b)
		if block == nil {
			return count
		}
		count++
		b = rest
	}
}

var _ = Describe("Loader", func() {
	var fs xfs.FS
	BeforeEach(func() {
		fs = xfs.NewMem()
	})
	Describe("TrustAnchorsPEM", func() {
		It("Should return both the CA and the node certificate", func() {
			mock.GenerateCerts(fs)
			l := MustSucceed(cert.NewLoader(cert.LoaderConfig{FS: fs}))
			Expect(countPEMBlocks(MustSucceed(l.TrustAnchorsPEM()))).To(Equal(2))
		})

		It("Should return the node certificate when there is no CA", func() {
			mock.GenerateCerts(fs)
			l := MustSucceed(cert.NewLoader(cert.LoaderConfig{FS: fs}))
			Expect(l.FS.Remove(l.CACertPath)).To(Succeed())
			Expect(countPEMBlocks(MustSucceed(l.TrustAnchorsPEM()))).To(Equal(1))
		})

		It("Should return an error when no certificates exist", func() {
			l := MustSucceed(cert.NewLoader(cert.LoaderConfig{FS: fs}))
			Expect(l.TrustAnchorsPEM()).Error().To(MatchError(validate.ErrValidation))
		})
	})
})
