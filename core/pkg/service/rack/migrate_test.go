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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/search"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	xstatus "github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Migration v0", func() {
	It("Should read a status whose rack key was stored as float64", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New(), gorp.WithCodec(msgpack.Codec)))
		otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
		searchIdx := MustOpen(search.Open())
		g := MustOpen(group.OpenService(ctx, group.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Search:   searchIdx,
		}))
		labelSvc := MustOpen(label.OpenService(ctx, label.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    g,
			Search:   searchIdx,
		}))
		stat := MustOpen(status.OpenService(ctx, status.ServiceConfig{
			Ontology: otg,
			DB:       db,
			Group:    g,
			Label:    labelSvc,
			Search:   searchIdx,
		}))

		rackKey := rack.NewKey(1, 1)
		testRack := &rack.Rack{Key: rackKey, Name: "Migration Test Rack"}
		Expect(gorp.NewCreate[rack.Key, rack.Rack]().Entry(testRack).Exec(ctx, db)).
			To(Succeed())

		// Write a status using Status[any] with the rack key as float64,
		// simulating legacy data where the key was encoded as a msgpack
		// float64 instead of uint32.
		legacyStatus := status.Status[any]{
			Key:     rack.OntologyID(rackKey).String(),
			Name:    "Legacy Rack Status",
			Variant: xstatus.VariantSuccess,
			Message: "Started",
			Time:    telem.Now(),
			Details: map[string]any{
				"rack": float64(rackKey),
			},
		}
		Expect(status.NewWriter[any](stat, nil).Set(ctx, &legacyStatus)).To(Succeed())

		// Opening the rack service triggers the v0.status_backfill migration,
		// which reads existing statuses as Status[StatusDetails]. This would
		// fail without the flex DecodeMsgpack on the Key type because the
		// rack key is stored as a msgpack float64.
		MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
			DB:                  db,
			Ontology:            otg,
			Group:               g,
			HostProvider:        mock.StaticHostKeyProvider(1),
			Status:              stat,
			HealthCheckInterval: 10 * telem.Millisecond,
			Search:              searchIdx,
		}))

		// Verify the status is readable with the correct typed key.
		var restoredStatus rack.Status
		Expect(status.NewRetrieve[rack.StatusDetails](stat).
			WhereKeys(rack.OntologyID(rackKey).String()).
			Entry(&restoredStatus).
			Exec(ctx, nil)).To(Succeed())
		Expect(restoredStatus.Details.Rack).To(Equal(rackKey))
	})
})
