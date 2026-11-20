// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	v0 "github.com/synnaxlabs/synnax/pkg/service/certificate/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/certificate/versions/v1"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("NewMigration", func() {
	It("Should move each file certificate to the given rack", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		StartRun(db.Run)
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v0.Key, v0.Certificate]{DB: db}))
		file := v0.Base{Key: uuid.New(), Name: "Plant file"}
		stored := v0.Base{Key: uuid.New(), Name: "Plant"}
		details := v0.Details{Subject: "CN=plant", NotAfter: 10}
		Expect(gorp.NewCreate[v0.Key, v0.Certificate]().Entries(&[]v0.Certificate{
			{Variant: v0.FileCertificate{Base: file, CertificatePath: "/c.pem"}},
			{Variant: v0.StoredCertificate{Base: stored, Pem: "pem", Details: details}},
		}).Exec(ctx, db)).To(Succeed())
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v1.Key, v1.Certificate]{
			DB: db, Migrations: []gorp.Migration{v1.NewMigration(7)},
		}))
		var got []v1.Certificate
		Expect(gorp.NewRetrieve[v1.Key, v1.Certificate]().
			Entries(&got).
			Exec(ctx, db)).To(Succeed())
		Expect(got).To(ConsistOf(
			v1.Certificate{Variant: v1.FileCertificate{
				Base:            file,
				CertificatePath: "/c.pem",
				Rack:            7,
			}},
			v1.Certificate{Variant: v1.StoredCertificate{
				Base:    stored,
				Pem:     "pem",
				Details: details,
			}},
		))
	})
})
