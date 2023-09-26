package workspace_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/workspace"
	"github.com/synnaxlabs/x/gorp"
)

var _ = Describe("Writer", func() {
	Describe("Create", func() {
		It("Should create a workspace", func() {
			ws := workspace.Workspace{
				Name:   "test",
				Author: author.Key,
				Layout: "data",
			}
			Expect(svc.NewWriter(tx).Create(ctx, &ws)).To(Succeed())
			Expect(ws.Key).ToNot(Equal(uuid.Nil))
		})
	})
	Describe("Rename", func() {
		It("Should rename a workspace", func() {
			ws := workspace.Workspace{Name: "test", Author: author.Key}
			Expect(svc.NewWriter(tx).Create(ctx, &ws)).To(Succeed())
			Expect(svc.NewWriter(tx).Rename(ctx, ws.Key, "test2")).To(Succeed())
			var res workspace.Workspace
			Expect(gorp.NewRetrieve[uuid.UUID, workspace.Workspace]().WhereKeys(ws.Key).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Name).To(Equal("test2"))
		})
	})
	Describe("SetLayout", func() {
		It("Should set the layout of a workspace", func() {
			ws := workspace.Workspace{Name: "test", Author: author.Key}
			Expect(svc.NewWriter(tx).Create(ctx, &ws)).To(Succeed())
			Expect(svc.NewWriter(tx).SetLayout(ctx, ws.Key, "data")).To(Succeed())
			var res workspace.Workspace
			Expect(gorp.NewRetrieve[uuid.UUID, workspace.Workspace]().WhereKeys(ws.Key).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Layout).To(Equal([]byte{1, 2, 3}))
		})
	})
	Describe("Delete", func() {
		It("Should delete a workspace", func() {
			ws := workspace.Workspace{Name: "test", Author: author.Key}
			Expect(svc.NewWriter(tx).Create(ctx, &ws)).To(Succeed())
			Expect(svc.NewWriter(tx).Delete(ctx, ws.Key)).To(Succeed())
			var res workspace.Workspace
			Expect(gorp.NewRetrieve[uuid.UUID, workspace.Workspace]().WhereKeys(ws.Key).Entry(&res).Exec(ctx, tx)).ToNot(Succeed())
		})
	})
})
