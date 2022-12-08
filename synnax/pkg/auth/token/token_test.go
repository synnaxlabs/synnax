package token_test

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/auth/token"
	"time"
)

type mockKeyService struct {
	key *rsa.PrivateKey
}

func (m *mockKeyService) NodeSecret() crypto.PrivateKey {
	return m.key
}

var _ = Describe("token", func() {
	var (
		svc *token.Service
	)
	BeforeEach(func() {
		k, err := rsa.GenerateKey(rand.Reader, 1024)
		Expect(err).ToNot(HaveOccurred())
		svc = &token.Service{
			KeyService: &mockKeyService{key: k},
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
