// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package bind_test

import (
	"context"
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
)

var _ = Describe("Write and large records", func() {
	It(
		"Should check a write that keeps every link of a record with many links in "+
			"about the time of the write",
		func(ctx SpecContext) {
			const n = 20_000
			views := openTagged[tagged](ctx, string(resource.TypeFramer))
			labels := make([]label.Label, n)
			reads := make([]string, n)
			for i := range labels {
				labels[i] = label.Label{Key: uuid.New(), Name: "read"}
				reads[i] = labels[i].Key.String()
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Label.NewWriter(tx).CreateMany(ctx, &labels)
			})).To(Succeed())
			// A task or schematic that reads many channels holds a link to each.
			holder := tagged{Key: "large", Reads: reads}
			Expect(views.NewCreate().Entry(&holder).Exec(ctx, db)).To(Succeed())
			id := resource.ID{Type: resource.TypeFramer, Key: holder.Key}
			u := newUser(ctx)
			grant(ctx, u, access.ActionUpdate, id)
			kept := change[tagged](holder.Key, func(t *tagged) { t.Name += "!" })
			unchecked := bind.WriteUnchecked(
				db,
				func(ctx context.Context, tx gorp.Tx, _ struct{}) (struct{}, error) {
					return struct{}{}, kept(ctx, tx)
				},
			)
			timed := func(f func() error) time.Duration {
				start := time.Now()
				Expect(f()).To(Succeed())
				return time.Since(start)
			}
			plain := timed(func() error {
				_, err := unchecked(ctx, struct{}{})
				return err
			})
			checked := timed(func() error { return write(ctx, u, kept) })
			GinkgoWriter.Printf(
				"%d links: unchecked write %s, checked write %s\n",
				n,
				plain,
				checked,
			)
			Expect(checked - plain).To(BeNumerically("<", 500*time.Millisecond))
		},
	)
})
