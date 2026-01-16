package role_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
)

var _ = Describe("Retrieve", func() {
	var (
		tx    gorp.Tx
		w     role.Writer
		roles []role.Role
	)
	BeforeEach(func() {
		tx = db.OpenTx()
		w = svc.NewWriter(tx)
		roles = []role.Role{
			{Name: "admin", Description: "Administrator"},
			{Name: "engineer", Description: "Engineer"},
			{Name: "viewer", Description: "Viewer"},
		}
		for i := range roles {
			Expect(w.Create(ctx, &roles[i])).To(Succeed())
		}
	})
	AfterEach(func() { Expect(tx.Close()).To(Succeed()) })

	Describe("WhereKeys", func() {
		It("Should retrieve a single role by key", func() {
			var r role.Role
			Expect(svc.NewRetrieve().
				WhereKeys(roles[0].Key).
				Entry(&r).
				Exec(ctx, tx)).To(Succeed())
			Expect(r.Key).To(Equal(roles[0].Key))
			Expect(r.Name).To(Equal(roles[0].Name))
		})

		It("Should retrieve multiple roles by keys", func() {
			var rs []role.Role
			Expect(svc.NewRetrieve().
				WhereKeys(roles[0].Key, roles[1].Key).
				Entries(&rs).
				Exec(ctx, tx)).To(Succeed())
			Expect(rs).To(HaveLen(2))
			Expect(rs).To(ConsistOf(roles[0], roles[1]))
		})

		It("Should return error when key not found", func() {
			var r role.Role
			Expect(svc.NewRetrieve().
				WhereKeys(uuid.New()).
				Entry(&r).
				Exec(ctx, tx)).To(MatchError(query.NotFound))
		})
	})

	Describe("WhereName", func() {
		It("Should retrieve a role by name", func() {
			var r role.Role
			Expect(svc.NewRetrieve().
				WhereName("admin").
				Entry(&r).
				Exec(ctx, tx)).To(Succeed())
			Expect(r.Name).To(Equal("admin"))
		})

		It("Should return error when name not found", func() {
			var r role.Role
			Expect(svc.NewRetrieve().
				WhereName("nonexistent").
				Entry(&r).
				Exec(ctx, tx)).To(MatchError(query.NotFound))
		})
	})

	Describe("Limit and Offset", func() {
		It("Should limit results", func() {
			var rs []role.Role
			Expect(svc.NewRetrieve().
				Limit(2).
				Entries(&rs).
				Exec(ctx, tx)).To(Succeed())
			Expect(rs).To(HaveLen(2))
		})

		It("Should apply offset", func() {
			var rs []role.Role
			Expect(svc.NewRetrieve().
				Offset(2).
				Entries(&rs).
				Exec(ctx, tx)).To(Succeed())
			Expect(rs).To(HaveLen(1))
		})

		It("Should handle offset beyond results", func() {
			var rs []role.Role
			Expect(svc.NewRetrieve().
				Offset(10).
				Entries(&rs).
				Exec(ctx, tx)).To(Succeed())
			Expect(rs).To(BeEmpty())
		})
	})
})
