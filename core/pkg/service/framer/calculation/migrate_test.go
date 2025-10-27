package calculation_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	svcstatus "github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/telem"

	"github.com/synnaxlabs/synnax/pkg/service/framer/calculation"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Migrate", Ordered, func() {
	var (
		c      *calculation.Service
		arcSvc *arc.Service
		dist   mock.Node
	)
	BeforeAll(func() {
		distB := mock.NewCluster()
		dist = distB.Provision(ctx)
		labelSvc := MustSucceed(label.OpenService(ctx, label.Config{
			DB:       dist.DB,
			Ontology: dist.Ontology,
			Group:    dist.Group,
			Signals:  dist.Signals,
		}))
		DeferCleanup(func() {
			Expect(labelSvc.Close()).To(Succeed())
		})
		statusSvc := MustSucceed(svcstatus.OpenService(ctx, svcstatus.ServiceConfig{
			DB:       dist.DB,
			Label:    labelSvc,
			Ontology: dist.Ontology,
			Group:    dist.Group,
			Signals:  dist.Signals,
		}))
		DeferCleanup(func() {
			Expect(statusSvc.Close()).To(Succeed())
		})
		arcSvc = MustSucceed(arc.OpenService(ctx, arc.ServiceConfig{
			Channel:  dist.Channel,
			Ontology: dist.Ontology,
			DB:       dist.DB,
			Framer:   dist.Framer,
			Status:   statusSvc,
			Signals:  dist.Signals,
		}))
		DeferCleanup(func() {
			Expect(arcSvc.Close()).To(Succeed())
		})
	})

	createCalcSvc := func() {
		c = MustSucceed(calculation.OpenService(ctx, calculation.ServiceConfig{
			DB:                dist.DB,
			Framer:            dist.Framer,
			Channel:           dist.Channel,
			ChannelObservable: dist.Channel.NewObservable(),
			Arc:               arcSvc,
		}))
	}

	AfterAll(func() {
		Expect(c.Close()).To(Succeed())
		Expect(dist.Close()).To(Succeed())
	})

	Describe("Simple Expressions", func() {
		It("Should correctly migrate a simple logical expression", func() {
			baseTime := channel.Channel{
				Name:     "base_time",
				DataType: telem.TimeStampT,
				IsIndex:  true,
			}
			Expect(dist.Channel.Create(ctx, &baseTime)).To(Succeed())
			base := channel.Channel{
				Name:       "base",
				DataType:   telem.Float32T,
				LocalIndex: baseTime.LocalKey,
			}
			Expect(dist.Channel.Create(ctx, &base)).To(Succeed())
			calc := channel.Channel{
				Name:       "calc",
				DataType:   telem.Float32T,
				Expression: `return base * 2`,
				Requires:   []channel.Key{base.Key()},
			}
			Expect(dist.Channel.Create(ctx, &calc)).To(Succeed())
			createCalcSvc()

			var updatedCalc channel.Channel
			Expect(dist.Channel.NewRetrieve().
				WhereKeys(calc.Key()).
				Entry(&updatedCalc).
				Exec(ctx, nil),
			).To(Succeed())
			Expect(updatedCalc.Requires).To(BeEmpty())
			Expect(updatedCalc.LocalIndex).ToNot(BeZero())
			Expect(updatedCalc.Expression).To(Equal("return base * 2"))

			var newCalcIndex channel.Channel
			Expect(dist.Channel.NewRetrieve().
				WhereKeys(calc.Index()).
				Entry(&newCalcIndex).
				Exec(ctx, nil),
			).To(Succeed())
			Expect(newCalcIndex.Requires).To(BeEmpty())
			Expect(newCalcIndex.Virtual).To(BeTrue())
			Expect(newCalcIndex.Expression).To(BeEmpty())
		})

	})
})
