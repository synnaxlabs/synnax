package rbac_test

import (
	"github.com/arya-analytics/delta/pkg/access"
	"github.com/arya-analytics/delta/pkg/access/rbac"
	"github.com/arya-analytics/delta/pkg/user"
	"github.com/arya-analytics/x/gorp"
	"github.com/arya-analytics/x/kv/memkv"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Enforcer", func() {
	var (
		db         *gorp.DB
		legislator *rbac.Legislator
		enforcer   access.Enforcer
	)
	BeforeEach(func() {
		db = gorp.Wrap(memkv.New())
		legislator = &rbac.Legislator{DB: db}
		enforcer = &rbac.Enforcer{
			DefaultEffect: access.Deny,
			Legislator:    legislator,
		}
		txn := db.BeginTxn()
		Expect(legislator.Create(txn, changePasswordPolicy)).To(Succeed())
		Expect(txn.Commit()).To(Succeed())
	})
	AfterEach(func() {
		Expect(db.Close()).To(Succeed())
	})
	It("Should allow access when a valid policy exists", func() {
		Expect(enforcer.Enforce(access.Request{
			Subject: userID,
			Object:  userID,
			Action:  "changePassword",
		})).To(BeNil())
	})
	It("Should return the default effect when a policy can't be found", func() {
		Expect(enforcer.Enforce(access.Request{
			Subject: user.OntologyID(uuid.New()),
			Object:  userID,
			Action:  "changePassword",
		})).To(Equal(access.Denied))
	})
	It("Should return the default effect when no policy applies to the request", func() {
		Expect(enforcer.Enforce(access.Request{
			Subject: userID,
			Object:  userID,
			Action:  "retrieve",
		})).To(Equal(access.Denied))
	})
})
