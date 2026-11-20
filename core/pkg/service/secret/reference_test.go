// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package secret_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("NewReference", func() {
	var (
		e       *env
		devices *gorp.Table[string, device]
	)
	BeforeEach(func(ctx SpecContext) {
		e = newEnv(ctx)
		// Each device runs on the Core's rack.
		devices = MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[string, device]{
			DB:   e.db,
			Name: "device",
			References: []gorp.Reference[string, device]{
				secret.NewReference(e.svc, secret.ReferenceConfig[string, device]{
					Name: "password",
					Targets: func(d *device) []secret.Key {
						return []secret.Key{d.Password}
					},
					Rack: func(*device) rack.Key { return coreRack },
				}),
			},
		}))
	})
	create := func(ctx SpecContext, s secret.Secret) error {
		return devices.NewCreate().
			Entry(&device{Key: "d1", Password: s.Base().Key}).
			Exec(ctx, e.db)
	}

	It("Should allow a stored secret", func(ctx SpecContext) {
		Expect(create(ctx, e.createStored(ctx, "Password", "a"))).To(Succeed())
	})
	It("Should allow a file secret of the entry's rack", func(ctx SpecContext) {
		Expect(create(ctx, e.createFile(ctx, "Key", "k.pem", coreRack))).To(Succeed())
	})
	It("Should refuse a file secret of another rack", func(ctx SpecContext) {
		err := create(ctx, e.createFile(ctx, "Key", "k.pem", otherRack))
		Expect(err).To(MatchError(validate.ErrValidation))
		Expect(err).To(MatchError(
			`password: secret "Key" is a file of rack 2, not of rack 1: validation error`,
		))
	})
	It("Should keep a secret from being deleted while an entry uses it", func(
		ctx SpecContext,
	) {
		s := e.createStored(ctx, "Password", "a")
		Expect(create(ctx, s)).To(Succeed())
		Expect(e.write(ctx, func(w secret.Writer) error {
			return w.Delete(ctx, s.Base().Key)
		})).To(MatchError(ContainSubstring("because device d1 references it")))
	})
})
