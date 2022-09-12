package api_test

import (
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/storage"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
)

var _ = Describe("Api", func() {
	Describe("New", func() {
		It("Should open a new API without panicking", func() {
			Expect(func() {
				api.New(api.Config{
					Logger:  zap.NewNop(),
					Storage: &storage.Store{},
				})
			}).ToNot(Panic())
		})
	})
})
