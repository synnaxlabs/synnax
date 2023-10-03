package workspace_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/workspace"
)

var _ = Describe("Retrieve", func() {
	Describe("By Author", func() {
		It("Should retrieve workspaces by author", func() {
			ws1 := workspace.Workspace{Name: "test", Author: author.Key}
			ws2 := workspace.Workspace{Name: "test2", Author: author.Key}
			Expect(svc.NewWriter(tx).Create(ctx, &ws1)).To(Succeed())
			Expect(svc.NewWriter(tx).Create(ctx, &ws2)).To(Succeed())
			var res []workspace.Workspace
			Expect(svc.NewRetrieve().WhereAuthor(author.Key).Entries(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res).To(ConsistOf(ws1, ws2))
		})
	})

})
