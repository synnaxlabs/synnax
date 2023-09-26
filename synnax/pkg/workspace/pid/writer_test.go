package pid_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/workspace/pid"
	"github.com/synnaxlabs/x/gorp"
)

var _ = Describe("Writer", func() {
	Describe("Create", func() {
		It("Should create a PID", func() {
			pid := pid.PID{
				Name: "test",
				Data: "data",
			}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &pid)).To(Succeed())
			Expect(pid.Key).ToNot(Equal(uuid.Nil))
		})
	})
	Describe("Rename", func() {
		It("Should rename a PID", func() {
			p := pid.PID{Name: "test", Data: "data"}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &p)).To(Succeed())
			Expect(svc.NewWriter(tx).Rename(ctx, p.Key, "test2")).To(Succeed())
			var res pid.PID
			Expect(gorp.NewRetrieve[uuid.UUID, pid.PID]().WhereKeys(p.Key).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Name).To(Equal("test2"))
		})
	})
	Describe("SetData", func() {
		It("Should set the data of a PID", func() {
			p := pid.PID{Name: "test", Data: "data"}
			Expect(svc.NewWriter(tx).Create(ctx, ws.Key, &p)).To(Succeed())
			Expect(svc.NewWriter(tx).SetData(ctx, p.Key, []byte{4, 5, 6})).To(Succeed())
			var res pid.PID
			Expect(gorp.NewRetrieve[uuid.UUID, pid.PID]().WhereKeys(p.Key).Entry(&res).Exec(ctx, tx)).To(Succeed())
			Expect(res.Data).To(Equal([]byte{4, 5, 6}))
		})
	})
})
