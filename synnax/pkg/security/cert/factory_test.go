package cert_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/security/cert"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/testutil"
)

// smallKeySize to run tests faster.
const smallKeySize = 512

var _ = Describe("Factory", func() {
	var fs xfs.FS
	BeforeEach(func() {
		fs = xfs.NewMem()
	})
	Describe("CA Generation", func() {
		It("Should generate a CA and a key", func() {
			f := MustSucceed(cert.NewFactory(cert.FactoryConfig{
				LoaderConfig: cert.LoaderConfig{FS: fs},
				KeySize:      smallKeySize,
			}))
			Expect(f.CreateCAPair()).To(Succeed())
			c, k := MustSucceed2(f.Loader.LoadCAPair())
			Expect(c).ToNot(BeNil())
			Expect(k).ToNot(BeNil())
		})
		It("Should not allow key reuse by default", func() {
			f := MustSucceed(cert.NewFactory(cert.FactoryConfig{
				LoaderConfig: cert.LoaderConfig{FS: fs},
				KeySize:      smallKeySize,
			}))
			Expect(f.CreateCAPair()).To(Succeed())
			Expect(f.CreateCAPair()).ToNot(Succeed())
		})
		It("Should reuse the CA and key if they already exist", func() {
			f := MustSucceed(cert.NewFactory(cert.FactoryConfig{
				LoaderConfig:  cert.LoaderConfig{FS: fs},
				AllowKeyReuse: config.BoolPointer(true),
				KeySize:       smallKeySize,
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
				KeySize:      smallKeySize,
			}))
			Expect(f.CreateCAPair()).To(Succeed())
			Expect(f.CreateNodePair()).To(Succeed())
			c, k := MustSucceed2(f.Loader.LoadNodePair())
			Expect(c).ToNot(BeNil())
			Expect(k).ToNot(BeNil())
		})
		It("Should fail to generate a node cert and key if no CA is present", func() {
			f := MustSucceed(cert.NewFactory(cert.FactoryConfig{
				LoaderConfig: cert.LoaderConfig{FS: fs},
				KeySize:      smallKeySize,
			}))
			err := f.CreateNodePair()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("CA certificate not found"))
		})
		It("Should fail is no hosts are provided", func() {
			f := MustSucceed(cert.NewFactory(cert.FactoryConfig{
				LoaderConfig: cert.LoaderConfig{FS: fs},
				KeySize:      smallKeySize,
			}))
			Expect(f.CreateCAPair()).To(Succeed())
			err := f.CreateNodePair()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no hosts provided"))
		})
	})
})
