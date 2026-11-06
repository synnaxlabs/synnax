// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v3_test

import (
	"context"
	"encoding/json/v2"
	"math"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	v2 "github.com/synnaxlabs/synnax/pkg/service/ni/versions/v2"
	v3 "github.com/synnaxlabs/synnax/pkg/service/ni/versions/v3"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	taskv2 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v2"
	taskv3 "github.com/synnaxlabs/synnax/pkg/service/task/versions/v3"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("ScanMigration", func() {
	It("Should keep every scan config field and drop the ignored models",
		func(ctx SpecContext) {
			stored := v2.ScanConfig{
				Key:           uuid.New(),
				Rate:          5,
				Disabled:      true,
				IgnoredModels: []string{"^cRIO.*", "^nown.*"},
			}
			db := DeferClose(gorp.Wrap(memkv.New()))
			MustSucceed(gorp.OpenTable(
				ctx,
				gorp.TableConfig[uuid.UUID, v2.ScanConfig]{DB: db},
			))
			Expect(
				gorp.NewCreate[uuid.UUID, v2.ScanConfig]().Entry(&stored).Exec(ctx, db),
			).To(Succeed())
			Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
				DB:         db,
				Namespace:  "ni_scanner_config",
				Migrations: []gorp.Migration{v3.ScanMigration},
			})).To(Succeed())
			var got v3.ScanConfig
			Expect(gorp.NewRetrieve[uuid.UUID, v3.ScanConfig]().
				Where(gorp.MatchKeys[uuid.UUID, v3.ScanConfig](stored.Key)).
				Entry(&got).Exec(ctx, db)).To(Succeed())
			Expect(got.ScanConfig).To(Equal(stored.ScanConfig))
		},
	)
})

var _ = Describe("Port and line migrations", func() {
	var (
		db  *gorp.DB
		otg *ontology.Ontology
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		otg = MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
	})

	createTask := func(ctx SpecContext, key uuid.UUID) {
		GinkgoHelper()
		t := taskv2.Task{Key: key, Name: "Stored Task", Type: "ni_digital_read"}
		Expect(gorp.NewCreate[taskv2.Key, taskv2.Task]().Entry(&t).Exec(ctx, db)).
			To(Succeed())
		Expect(otg.NewWriter(nil).DefineResources(ctx, ontology.ID{
			Type: ontology.ResourceTypeTask,
			Key:  key.String(),
		})).To(Succeed())
	}

	run := func(ctx SpecContext, namespace string, m gorp.Migration) {
		GinkgoHelper()
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  namespace,
			Migrations: []gorp.Migration{m},
		})).To(Succeed())
	}

	It("Should keep each port of a stored analog read config", func(ctx SpecContext) {
		stored := v2.AnalogReadConfig{Channels: []v2.AIChannel{
			{Variant: v2.AIVoltageChannel{
				BaseAIChannel: v2.BaseAIChannel{Key: "a", Port: 300},
				CustomScale: v2.CustomScale{
					CustomScale: v2.Scale{Variant: v2.NoneScale{}},
				},
			}},
			{Variant: v2.AIThermocoupleChannel{
				BaseAIChannel: v2.BaseAIChannel{Key: "b", Port: 65535},
				Cjc:           v2.CJC{Variant: v2.ChanCJC{Port: 7}},
			}},
		}}
		stored.Key = uuid.New()
		Expect(gorp.NewCreate[uuid.UUID, v2.AnalogReadConfig]().Entry(&stored).
			Exec(ctx, db)).To(Succeed())
		run(ctx, "ni_analog_read_config", v3.NewAnalogReadMigration(otg))
		var got v3.AnalogReadConfig
		Expect(gorp.NewRetrieve[uuid.UUID, v3.AnalogReadConfig]().
			Where(gorp.MatchKeys[uuid.UUID, v3.AnalogReadConfig](stored.Key)).
			Entry(&got).Exec(ctx, db)).To(Succeed())
		Expect(got.Channels).To(HaveLen(2))
		voltage := got.Channels[0].Variant.(v3.AIVoltageChannel)
		Expect(voltage.Key).To(Equal("a"))
		Expect(voltage.Port).To(Equal(uint16(300)))
		thermocouple := got.Channels[1].Variant.(v3.AIThermocoupleChannel)
		Expect(thermocouple.Port).To(Equal(uint16(65535)))
		Expect(thermocouple.Cjc.Variant).To(Equal(v3.ChanCJC{Port: 7}))
	})

	It(
		"Should keep the port and line of a stored digital channel",
		func(ctx SpecContext) {
			stored := v2.DigitalWriteConfig{Channels: []v2.DOChannel{
				{Key: "a", Port: 2, Line: 255, StateChannel: 12, CmdChannel: 13},
			}}
			stored.Key = uuid.New()
			Expect(gorp.NewCreate[uuid.UUID, v2.DigitalWriteConfig]().Entry(&stored).
				Exec(ctx, db)).To(Succeed())
			run(ctx, "ni_digital_write_config", v3.NewDigitalWriteMigration(otg))
			var got v3.DigitalWriteConfig
			Expect(gorp.NewRetrieve[uuid.UUID, v3.DigitalWriteConfig]().
				Where(gorp.MatchKeys[uuid.UUID, v3.DigitalWriteConfig](stored.Key)).
				Entry(&got).Exec(ctx, db)).To(Succeed())
			Expect(got.Channels).To(HaveExactElements(v3.DOChannel{
				Key: "a", Port: 2, Line: 255, StateChannel: 12, CmdChannel: 13,
			}))
		},
	)

	DescribeTable("Should reject a value outside the new range",
		func(ctx SpecContext, migrate func(context.Context) error) {
			Expect(migrate(ctx)).To(MatchError(ContainSubstring("outside the range")))
		},
		Entry("negative port", func(ctx context.Context) error {
			_, err := v3.MigrateBaseAIChannel(ctx, v2.BaseAIChannel{Port: -1})
			return err
		}),
		Entry("port above 65535", func(ctx context.Context) error {
			_, err := v3.MigrateBaseCIChannel(ctx, v2.BaseCIChannel{Port: 65536})
			return err
		}),
		Entry("maximum port", func(ctx context.Context) error {
			_, err := v3.MigrateBaseAOChannel(
				ctx,
				v2.BaseAOChannel{Port: math.MaxInt32},
			)
			return err
		}),
		Entry("negative CJC port", func(ctx context.Context) error {
			_, err := v3.MigrateCJC(ctx, v2.CJC{Variant: v2.ChanCJC{Port: -1}})
			return err
		}),
		Entry("line above 255", func(ctx context.Context) error {
			_, err := v3.MigrateDIChannel(ctx, v2.DIChannel{Line: 256})
			return err
		}),
	)

	It("Should quarantine a config outside the range and keep the others",
		func(ctx SpecContext) {
			bad := v2.DigitalReadConfig{Channels: []v2.DIChannel{{Key: "a", Line: 300}}}
			bad.Key = uuid.New()
			good := v2.DigitalReadConfig{Channels: []v2.DIChannel{{Key: "b", Line: 3}}}
			good.Key = uuid.New()
			Expect(gorp.NewCreate[uuid.UUID, v2.DigitalReadConfig]().
				Entries(&[]v2.DigitalReadConfig{bad, good}).Exec(ctx, db)).To(Succeed())
			createTask(ctx, bad.Key)
			run(ctx, "ni_digital_read_config", v3.NewDigitalReadMigration(otg))

			var kept v3.DigitalReadConfig
			Expect(gorp.NewRetrieve[uuid.UUID, v3.DigitalReadConfig]().
				Where(gorp.MatchKeys[uuid.UUID, v3.DigitalReadConfig](good.Key)).
				Entry(&kept).Exec(ctx, db)).To(Succeed())
			Expect(kept.Channels[0].Line).To(Equal(uint8(3)))

			Expect(gorp.NewRetrieve[uuid.UUID, v3.DigitalReadConfig]().
				Where(gorp.MatchKeys[uuid.UUID, v3.DigitalReadConfig](bad.Key)).
				Entry(&v3.DigitalReadConfig{}).Exec(ctx, db)).
				To(MatchError(query.ErrNotFound))
			Expect(gorp.NewRetrieve[taskv2.Key, taskv2.Task]().
				Where(gorp.MatchKeys[taskv2.Key, taskv2.Task](bad.Key)).
				Entry(&taskv2.Task{}).Exec(ctx, db)).
				To(MatchError(query.ErrNotFound))

			staged, closer := MustSucceed2(db.Get(ctx, taskv3.QuarantineKVKey(bad.Key)))
			defer func() { Expect(closer.Close()).To(Succeed()) }()
			var row map[string]any
			Expect(json.Unmarshal(staged, &row)).To(Succeed())
			Expect(row).To(HaveKeyWithValue("name", "Stored Task"))
			config := row["config"].(map[string]any)
			channels := config["channels"].([]any)
			Expect(channels[0]).To(HaveKeyWithValue("line", float64(300)))
		},
	)
})
