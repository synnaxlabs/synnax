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
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// The case table is shared with the TypeScript matches, so that the Core and the
// client cache agree on every filter.
const vectorsPath = "../../../../../x/testdata/filter/vectors.json"

type whereCase struct {
	Name     string       `json:"name"`
	Where    device.Where `json:"where"`
	Expected []string     `json:"expected"`
}

type vectors struct {
	Devices struct {
		Records map[string]device.Device `json:"records"`
		Cases   []whereCase              `json:"cases"`
	} `json:"devices"`
}

var _ = Describe("Where", Ordered, func() {
	var (
		v   vectors
		svc *device.Service
	)
	Expect(json.Unmarshal(MustSucceed(os.ReadFile(vectorsPath)), &v)).To(Succeed())
	aliases := lo.MapEntries(v.Devices.Records, func(
		alias string,
		d device.Device,
	) (device.Key, string) {
		return d.Key, alias
	})

	BeforeAll(func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		otg := DeferClose(MustOpen(ontology.Open(ctx, ontology.Config{DB: db})))
		g := DeferClose(MustOpen(group.OpenService(ctx, group.ServiceConfig{
			DB:       db,
			Ontology: otg,
		})))
		lbl := DeferClose(MustOpen(label.OpenService(ctx, label.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    g,
		})))
		stat := DeferClose(MustOpen(status.OpenService(ctx, status.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    g,
			Label:    lbl,
		})))
		racks := DeferClose(MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
			DB:           db,
			Ontology:     otg,
			Group:        g,
			HostProvider: mock.NewStaticHostProvider(1),
			Status:       stat,
		})))
		svc = DeferClose(MustOpen(device.OpenService(ctx, device.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    g,
			Status:   stat,
			Rack:     racks,
		})))
		for _, d := range v.Devices.Records {
			r := rack.Rack{Key: d.Rack, Name: "Rack"}
			Expect(racks.NewWriter(nil).Create(ctx, &r)).To(Succeed())
			Expect(svc.NewWriter(nil).Create(ctx, &d)).To(Succeed())
		}
	})

	for _, c := range v.Devices.Cases {
		It("Should match the case: "+c.Name, func(ctx SpecContext) {
			f := MustSucceed(device.MatchWhere(c.Where))
			var res []device.Device
			// The API skips not-found, since a missing key is a non-match.
			Expect(errors.Skip(
				svc.NewRetrieve().Where(f).Entries(&res).Exec(ctx, nil),
				query.ErrNotFound,
			)).To(Succeed())
			Expect(lo.Map(res, func(d device.Device, _ int) string {
				return aliases[d.Key]
			})).To(ConsistOf(c.Expected))
		})
	}

	It("Should return a path error for an invalid regex", func() {
		var w device.Where
		Expect(json.Unmarshal(
			[]byte(`{"or":[{"key":{"in":[]}},{"model":{"regex":"(a"}}]}`),
			&w,
		)).To(Succeed())
		_, err := device.MatchWhere(w)
		var pathErr validate.PathError
		Expect(err).To(BeAssignableToTypeOf(pathErr))
		Expect(err).To(MatchError(ContainSubstring(
			"or.1.model.regex: error parsing regexp: missing closing ): `(a`",
		)))
	})
})
