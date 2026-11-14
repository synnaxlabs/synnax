// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package icd_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/icd/versions"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	. "github.com/synnaxlabs/synnax/pkg/service/imex/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("ImEx", func() {
	It("Should never match a legacy file", func() {
		Expect(svc.Match(map[string]any{"messages": []any{}})).To(BeFalse())
	})

	It("Should round trip an ICD through export and import", func(ctx SpecContext) {
		ic := icd.ICD{
			Name: "Exported",
			Messages: []icd.Message{
				canMessage("Status", 0x100, binaryField("rpm", 0, 16)),
			},
		}
		// Export reads only committed ICDs, and the suite's open transaction holds the
		// write lock of its DB, so the spec commits to a DB of its own.
		exportDB := DeferClose(gorp.Wrap(memkv.New()))
		registry := imex.NewService()
		exportSvc := MustOpen(icd.OpenService(ctx, icd.ServiceConfig{
			DB:   exportDB,
			ImEx: registry,
		}))
		Expect(exportDB.WithTx(ctx, func(tx gorp.Tx) error {
			return exportSvc.NewWriter(tx).Create(ctx, &ic)
		})).To(Succeed())
		env := MustSucceed(exportSvc.Export(ctx, exportDB, icd.ResourceID(ic.Key)))
		Expect(env.Version).To(Equal(versions.Latest))
		Expect(env.Type).To(Equal("icd"))
		Expect(env.Name).To(Equal("Exported"))
		var id resource.ID
		Expect(exportDB.WithTx(ctx, func(tx gorp.Tx) (err error) {
			id, err = registry.Import(ctx, tx, WireRoundTrip(env), imex.ImportOptions{})
			return err
		})).To(Succeed())
		Expect(id.Type).To(Equal(resource.TypeICD))
		key := MustSucceed(uuid.Parse(id.Key))
		Expect(key).ToNot(Equal(ic.Key))
		var res icd.ICD
		Expect(exportSvc.NewRetrieve().
			Where(icd.MatchKeys(key)).
			Entry(&res).
			Exec(ctx, exportDB)).To(Succeed())
		Expect(res.Name).To(Equal("Exported"))
		Expect(res.Messages).To(HaveExactElements(SatisfyAll(
			HaveField("Key", ic.Messages[0].Key),
			HaveField("Name", "Status"),
		)))
	})

	It(
		"Should return not found when exporting a missing ICD",
		func(ctx SpecContext) {
			Expect(svc.Export(ctx, db, icd.ResourceID(uuid.New()))).
				Error().To(MatchError(query.ErrNotFound))
		},
	)
})
