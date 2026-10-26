// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package device_test

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// The case table is shared with the TypeScript matches, so that the Core and the
// client cache agree on every filter.
const vectorsPath = "../../../../testdata/filter/vectors.json"

type whereCase struct {
	Name     string       `json:"name"`
	Where    device.Where `json:"where"`
	Expected []string     `json:"expected"`
	// ServerComputed marks a case that only the Core evaluates, which the TypeScript
	// matches checks.
	ServerComputed bool `json:"server_computed"`
}

// vectors is the device section of the case table.
type vectors struct {
	Records map[string]device.Device `json:"records"`
	Cases   []whereCase              `json:"cases"`
}

var _ = Describe("Where", Ordered, func() {
	var (
		v    vectors
		svc  *device.Service
		file map[string]jsontext.Value
	)
	Expect(json.Unmarshal(MustSucceed(os.ReadFile(vectorsPath)), &file)).To(Succeed())
	Expect(json.Unmarshal(
		file["devices"],
		&v,
		json.RejectUnknownMembers(true),
	)).To(Succeed())
	aliases := lo.MapEntries(v.Records, func(
		alias string,
		d device.Device,
	) (device.Key, string) {
		return d.Key, alias
	})

	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		db := DeferClose(gorp.Wrap(memkv.New()))
		otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
		g := MustOpen(group.OpenService(ctx, group.ServiceConfig{
			DB:       db,
			Ontology: otg,
		}))
		lbl := MustOpen(label.OpenService(ctx, label.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    g,
		}))
		stat := MustOpen(status.OpenService(ctx, status.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    g,
			Label:    lbl,
		}))
		racks := MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
			DB:           db,
			Ontology:     otg,
			Group:        g,
			HostProvider: mock.NewStaticHostProvider(1),
			Status:       stat,
		}))
		svc = MustOpen(device.OpenService(ctx, device.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    g,
			Status:   stat,
			Rack:     racks,
		}))
		for _, d := range v.Records {
			r := rack.Rack{Key: d.Rack, Name: "Rack"}
			Expect(racks.NewWriter(nil).Create(ctx, &r)).To(Succeed())
			Expect(svc.NewWriter(nil).Create(ctx, &d)).To(Succeed())
		}
	})

	DescribeTable("Should match the case", func(ctx SpecContext, c whereCase) {
		f := MustSucceed(device.MatchWhere(c.Where))
		var res []device.Device
		Expect(svc.NewRetrieve().
			SkipMissingKeys().
			Where(f).
			Entries(&res).
			Exec(ctx, nil)).To(Succeed())
		Expect(lo.Map(res, func(d device.Device, _ int) string {
			return aliases[d.Key]
		})).To(ConsistOf(c.Expected))
	}, lo.Map(v.Cases, func(c whereCase, _ int) TableEntry {
		return Entry(c.Name, c)
	}))

	It("Should return a path error for an invalid regex", func() {
		var w device.Where
		Expect(json.Unmarshal(
			[]byte(`{"or":[{"key":{"in":[]}},{"model":{"regex":"(a"}}]}`),
			&w,
		)).To(Succeed())
		Expect(device.MatchWhere(w)).Error().To(SatisfyAll(
			BeAssignableToTypeOf(validate.PathError{}),
			MatchError(
				"or.1.model.regex: error parsing regexp: missing closing ): `(a`: validation error",
			),
		))
	})
})
