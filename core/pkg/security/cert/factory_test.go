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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/security/cert"
	"github.com/synnaxlabs/synnax/pkg/security/mock"
	"github.com/synnaxlabs/x/address"
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Factory", func() {
	var fs xfs.FS
	BeforeEach(func() {
		fs = xfs.NewMem()
	})
	Describe("CA Generation", func() {
		It("Should generate a CA and a key", func() {
			f := MustSucceed(cert.NewFactory(cert.FactoryConfig{
				LoaderConfig: cert.LoaderConfig{FS: fs},
				KeySize:      mock.SmallKeySize,
			}))
			Expect(f.CreateCAPair()).To(Succeed())
			c, k := MustSucceed2(f.Loader.LoadCAPair())
			Expect(c).ToNot(BeNil())
			Expect(k).ToNot(BeNil())
			cas, err := f.Loader.LoadCAs()
			Expect(err).ToNot(HaveOccurred())
			Expect(cas).To(HaveLen(1))
		})
		It("Should not allow key reuse by default", func() {
			f := MustSucceed(cert.NewFactory(cert.FactoryConfig{
				LoaderConfig: cert.LoaderConfig{FS: fs},
				KeySize:      mock.SmallKeySize,
			}))
			Expect(f.CreateCAPair()).To(Succeed())
			Expect(f.CreateCAPair()).ToNot(Succeed())
		})
		It("Should reuse the CA and key if they already exist", func() {
			f := MustSucceed(cert.NewFactory(cert.FactoryConfig{
				LoaderConfig:  cert.LoaderConfig{FS: fs},
				AllowKeyReuse: new(true),
				KeySize:       mock.SmallKeySize,
			}))
			Expect(f.CreateCAPair()).To(Succeed())
			Expect(f.CreateCAPair()).To(Succeed())
		})
	})
	Describe("Node Generation", func() {
		It("Should generate a node certificate and a key", func() {
			f := MustSucceed(cert.NewFactory(cert.FactoryConfig{
				LoaderConfig: cert.LoaderConfig{FS: fs},
				Hosts:        []address.Address{"synnaxlabs.com"},
				KeySize:      mock.SmallKeySize,
			}))
			Expect(f.CreateCAPair()).To(Succeed())
			Expect(f.CreateNodePair()).To(Succeed())
			c, k := MustSucceed2(f.Loader.LoadNodePair())
			Expect(c).ToNot(BeNil())
			Expect(k).ToNot(BeNil())
			tlsC, err := f.Loader.LoadNodeTLS()
			Expect(err).ToNot(HaveOccurred())
			Expect(tlsC).ToNot(BeNil())
		})
		It("Should fail to generate a node cert and key if no CA is present", func() {
			f := MustSucceed(cert.NewFactory(cert.FactoryConfig{
				LoaderConfig: cert.LoaderConfig{FS: fs},
				KeySize:      mock.SmallKeySize,
			}))
			err := f.CreateNodePair()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("CA certificate not found"))
		})
		It("Should fail is no hosts are provided", func() {
			f := MustSucceed(cert.NewFactory(cert.FactoryConfig{
				LoaderConfig: cert.LoaderConfig{FS: fs},
				KeySize:      mock.SmallKeySize,
			}))
			Expect(f.CreateCAPair()).To(Succeed())
			err := f.CreateNodePair()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no hosts provided"))
		})
	})
})
