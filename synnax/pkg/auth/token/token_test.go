package token_test

import (
	"crypto/rand"
	"crypto/rsa"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/auth/token"
	"time"
)

var _ = Describe("token", func() {
	var (
		svc *token.Service
	)
	BeforeEach(func() {
		k, err := rsa.GenerateKey(rand.Reader, 2048)
		Expect(err).ToNot(HaveOccurred())
		svc = &token.Service{
			Secret:     k,
			Expiration: 5 * time.Second,
		}
	})
	It("Should generate a token for the given issuer", func() {
		issuer := uuid.New()
		token, err := svc.New(issuer)
		Expect(err).ToNot(HaveOccurred())
		Expect(token).ToNot(BeEmpty())
	})
	It("Should validate a token", func() {
		issuer := uuid.New()
		token, err := svc.New(issuer)
		Expect(err).ToNot(HaveOccurred())
		Expect(token).ToNot(BeEmpty())
		issuer2, err := svc.Validate(token)
		Expect(err).ToNot(HaveOccurred())
		Expect(issuer).To(Equal(issuer2))
	})
})
