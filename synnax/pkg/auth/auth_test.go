package auth_test

import (
	"github.com/cockroachdb/errors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/auth"
	"github.com/synnaxlabs/synnax/pkg/auth/password"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
)

var _ = Describe("KV", Ordered, func() {
	var (
		authenticator auth.Authenticator
		DB            *gorp.DB
		creds         auth.InsecureCredentials
	)
	BeforeAll(func() {
		DB = gorp.Wrap(memkv.New())
		authenticator = auth.MultiAuthenticator{&auth.KV{DB: DB}}
		creds = auth.InsecureCredentials{Username: "user", Password: "pass"}
	})
	AfterAll(func() {
		Expect(DB.Close()).To(Succeed())
	})
	Describe("New", func() {
		It("Should register the credentials in the key-value store", func() {
			w := authenticator.NewWriter()
			err := w.Register(creds)
			Expect(err).To(BeNil())
			var c auth.SecureCredentials
			Expect(gorp.NewRetrieve[string, auth.SecureCredentials]().
				WhereKeys(creds.Username).
				Entry(&c).
				Exec(DB)).To(Succeed())
		})
		It("Should return a UniqueViolation error when the username is already registered", func() {
			w := authenticator.NewWriter()
			err := w.Register(creds)
			Expect(errors.Is(err, errors.New("[auth] - username already registered"))).To(BeTrue())

		})
	})
	Describe("Authenticate", func() {
		It("Should return a nil error for valid credentials", func() {
			Expect(authenticator.Authenticate(creds)).To(Succeed())
		})
		It("Should return an InvalidCredentials error when the password is wrong", func() {
			invalidCreds := auth.InsecureCredentials{
				Username: creds.Username,
				Password: "invalid",
			}
			Expect(authenticator.Authenticate(invalidCreds)).
				To(MatchError(auth.InvalidCredentials))
		})
		It("Should return an InvalidCredentials error when the user can't be found", func() {
			invalidCreds := auth.InsecureCredentials{
				Username: "invalid",
				Password: "invalid",
			}
			Expect(authenticator.Authenticate(invalidCreds)).
				To(MatchError(auth.InvalidCredentials))

		})
	})
	Describe("UpdateUsername", func() {
		It("Should update the username", func() {
			w := authenticator.NewWriter()
			Expect(w.UpdateUsername(creds, "new-user")).To(Succeed())
			var c auth.SecureCredentials
			Expect(gorp.NewRetrieve[string, auth.SecureCredentials]().
				WhereKeys("new-user").
				Entry(&c).
				Exec(DB)).To(Succeed())
			creds.Username = c.Username
		})
		It("Should return an InvalidCredentials error when the password is wrong", func() {
			invalidCreds := auth.InsecureCredentials{
				Username: creds.Username,
				Password: "invalid",
			}
			Expect(authenticator.NewWriter().UpdateUsername(invalidCreds, "new-user")).
				To(MatchError(auth.InvalidCredentials))
		})
		It("Should return a UniqueViolation error when the username is already registered", func() {
			w := authenticator.NewWriter()
			Expect(w.Register(auth.InsecureCredentials{
				Username: "old-user",
				Password: "pass",
			})).To(Succeed())
			Expect(errors.Is(w.UpdateUsername(creds, "old-user"),
				errors.New("[auth] - username already registered"))).To(BeTrue())
		})
	})
	Describe("UpdatePassword", func() {
		It("Should update the users password", func() {
			w := authenticator.NewWriter()
			var newPass password.Raw = "new-pass"
			Expect(w.UpdatePassword(creds, newPass)).To(Succeed())
			Expect(authenticator.Authenticate(creds)).ToNot(Succeed())
			creds.Password = newPass
			Expect(authenticator.Authenticate(creds)).To(Succeed())
		})
		It("Should return an InvalidCredentials error when the password is wrong", func() {
			invalidCreds := auth.InsecureCredentials{
				Username: creds.Username,
				Password: "invalid",
			}
			w := authenticator.NewWriter()
			var newPass password.Raw = "new-pass"
			Expect(w.UpdatePassword(invalidCreds, newPass)).To(MatchError(auth.InvalidCredentials))
		})
		It("Should return an InvalidCredentials error when the user can't be found", func() {
			invalidCreds := auth.InsecureCredentials{
				Username: "invalid",
				Password: "invalid",
			}
			w := authenticator.NewWriter()
			var newPass password.Raw = "new-pass"
			Expect(w.UpdatePassword(invalidCreds, newPass)).To(MatchError(auth.InvalidCredentials))
		})
	})
})
