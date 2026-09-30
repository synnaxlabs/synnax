// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package rack_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/gorp"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// openRackService opens a rack service and its dependencies on db, as a Core start
// does. Closing the returned closer shuts all of them down.
func openRackService(
	ctx context.Context,
	db *gorp.DB,
	healthCheckInterval telem.TimeSpan,
) (*rack.Service, xio.MultiCloser) {
	otg := MustSucceed(ontology.Open(ctx, ontology.Config{DB: db}))
	searchIdx := MustSucceed(search.OpenIndex())
	g := MustSucceed(group.OpenService(ctx, group.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Search:   searchIdx,
	}))
	lbl := MustSucceed(label.OpenService(ctx, label.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Group:    g,
		Search:   searchIdx,
	}))
	stat := MustSucceed(status.OpenService(ctx, status.ServiceConfig{
		Ontology: otg,
		DB:       db,
		Group:    g,
		Label:    lbl,
		Search:   searchIdx,
	}))
	svc := MustSucceed(rack.OpenService(ctx, rack.ServiceConfig{
		DB:                  db,
		Ontology:            otg,
		Group:               g,
		HostProvider:        mock.NewStaticHostProvider(1),
		Status:              stat,
		HealthCheckInterval: healthCheckInterval,
		Search:              searchIdx,
	}))
	return svc, xio.MultiCloser{otg, searchIdx, g, lbl, stat, svc}
}

var _ = Describe("Monitor", func() {
	It(
		"Should mark a stored healthy rack as dead when it is silent after a restart",
		func(ctx SpecContext) {
			db := DeferClose(gorp.Wrap(memkv.New()))
			svc, closer := openRackService(ctx, db, telem.Hour)
			r := rack.Rack{
				Name: "restarted rack",
				Status: &rack.Status{
					Variant: status.VariantSuccess,
					Message: "Driver is running",
					Time:    telem.Now(),
				},
			}
			Expect(svc.NewWriter(nil).Create(ctx, &r)).To(Succeed())
			Expect(closer.Close()).To(Succeed())

			svc, closer = openRackService(ctx, db, 10*telem.Millisecond)
			DeferClose(closer)
			Eventually(func(g Gomega) {
				s := MustSucceed(svc.RetrieveStatus(ctx, r.Key))
				g.Expect(s.Variant).To(Equal(status.VariantWarning))
				g.Expect(s.Message).
					To(Equal("Synnax Driver on restarted rack not running"))
			}).Should(Succeed())
		},
	)
})
