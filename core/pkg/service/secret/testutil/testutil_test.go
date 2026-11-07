// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	secrettestutil "github.com/synnaxlabs/synnax/pkg/service/secret/testutil"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("OpenService", func() {
	It("Should open a service that stores and resolves secrets", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
		g := MustOpen(
			group.OpenService(ctx, group.ServiceConfig{DB: db, Ontology: otg}),
		)
		r := rack.Key(1).OntologyID()
		svc := secrettestutil.OpenService(ctx, db, otg, g, func(
			context.Context,
			gorp.Tx,
			[]secret.Key,
		) ([]secret.Use, error) {
			return nil, nil
		})
		s := secret.Secret{Variant: secret.StoredSecret{Base: secret.Base{Name: "Key"}}}
		Expect(svc.NewWriter(nil).Create(ctx, &s, []byte("hunter2"))).To(Succeed())
		grant := MustSucceed(svc.Grant(ctx, nil, r, s.Base().Key))
		Expect(svc.Resolve(ctx, nil, r, grant, s.Base().Key)).
			To(Equal(map[secret.Key][]byte{s.Base().Key: []byte("hunter2")}))
	})
})
