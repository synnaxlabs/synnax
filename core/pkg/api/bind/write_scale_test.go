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

var _ = Describe("Write and records with many links", func() {
	It("Should check an update that doubles the links of a record in linear time", func(
		ctx SpecContext,
	) {
		const n = 20_000
		views := openTagged[tagged](ctx, string(resource.TypeFramer))
		createLabels := func() []string {
			GinkgoHelper()
			labels := make([]label.Label, n)
			for i := range labels {
				labels[i] = label.Label{Key: uuid.New(), Name: "read"}
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Label.NewWriter(tx).CreateMany(ctx, &labels)
			})).To(Succeed())
			keys := make([]string, n)
			for i, l := range labels {
				keys[i] = l.Key.String()
			}
			return keys
		}
		held, added := createLabels(), createLabels()
		// Each write adds n links to a record that holds n.
		doubled := func(key string) apply {
			holder := tagged{Key: key, Reads: held}
			Expect(views.NewCreate().Entry(&holder).Exec(ctx, db)).To(Succeed())
			return change[tagged](key, func(t *tagged) {
				t.Reads = append(t.Reads, added...)
			})
		}
		plainWrite, checkedWrite := doubled("plain"), doubled("checked")
		u := newUser(ctx)
		grant(
			ctx,
			u,
			access.ActionUpdate,
			resource.ID{Type: resource.TypeFramer, Key: "checked"},
		)
		grant(ctx, u, access.ActionRetrieve, resource.ID{Type: resource.TypeLabel})
		unchecked := bind.WriteUnchecked(
			db,
			func(ctx context.Context, tx gorp.Tx, _ struct{}) (struct{}, error) {
				return struct{}{}, plainWrite(ctx, tx)
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
		checked := timed(func() error { return write(ctx, u, checkedWrite) })
		GinkgoWriter.Printf("unchecked %s, checked %s\n", plain, checked)
		Expect(checked - plain).To(BeNumerically("<", time.Second))
	})
})
