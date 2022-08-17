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

var (
	userID               = user.OntologyID(uuid.New())
	changePasswordPolicy = rbac.Policy{
		Subject: userID,
		Object:  userID,
		Actions: []access.Action{"changePassword"},
	}
)

var _ = Describe("Legislator", func() {
	var (
		db         *gorp.DB
		legislator *rbac.Legislator
	)
	BeforeEach(func() {
		db = gorp.Wrap(memkv.New())
		legislator = &rbac.Legislator{DB: db}
	})
	AfterEach(func() {
		Expect(db.Close()).To(Succeed())
	})
	It("Should save a new policy", func() {
		txn := db.BeginTxn()
		// Giving a user the rights to change their own password
		Expect(legislator.Create(txn, changePasswordPolicy)).To(Succeed())
	})
	It("Should retrieve a policy", func() {
		txn := db.BeginTxn()
		// Giving a user the rights to change their own password
		Expect(legislator.Create(txn, changePasswordPolicy)).To(Succeed())
		Expect(txn.Commit()).To(Succeed())
		p, err := legislator.Retrieve(userID, userID)
		Expect(err).ToNot(HaveOccurred())
		Expect(p[0]).To(Equal(changePasswordPolicy))
	})
})
