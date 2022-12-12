package server_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/server"
	"github.com/synnaxlabs/x/config"
	. "github.com/synnaxlabs/x/testutil"
	"go.uber.org/zap"
	"sync"
	"time"
)

var _ = Describe("Grpc", func() {
	It("Should start a grpc server", func() {
		b := MustSucceed(server.New(server.Config{
			ListenAddress: "localhost:26260",
			Security: server.SecurityConfig{
				Insecure: config.BoolPointer(true),
			},
			Debug:  config.BoolPointer(true),
			Logger: zap.NewNop(),
			Branches: []server.Branch{
				&server.GRPCBranch{},
			},
		}))
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer GinkgoRecover()
			Expect(b.Serve()).To(Succeed())
			wg.Done()
		}()
		time.Sleep(10 * time.Millisecond)
		b.Stop()
		wg.Wait()
	})
})
