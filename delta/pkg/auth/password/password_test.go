package password_test

import (
	"github.com/arya-analytics/delta/pkg/auth/password"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Password", func() {
	Describe("Hash", func() {
		It("Should hash a password without error", func() {
			raw := password.Raw("password")
			_, err := raw.Hash()
			Expect(err).ToNot(HaveOccurred())
		})
	})
	Describe("Compare", func() {
		It("Should return a nil error for a valid password", func() {
			raw := password.Raw("password")
			hashed, err := raw.Hash()
			Expect(err).ToNot(HaveOccurred())
			err = hashed.Validate(raw)
			Expect(err).ToNot(HaveOccurred())
		})
		It("Should return a password.Invalid error for an invalid password", func() {
			raw := password.Raw("password")
			hashed, err := raw.Hash()
			Expect(err).ToNot(HaveOccurred())
			err = hashed.Validate(password.Raw("wrong"))
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(password.Invalid))
		})
	})
})
