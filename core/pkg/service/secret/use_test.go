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
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	"github.com/synnaxlabs/x/errors"
)

var _ = Describe("MultiFinder", func() {
	// finding returns a Finder that returns uses, and records the keys it was asked
	// for in asked.
	finding := func(asked *[][]secret.Key, uses ...secret.Use) secret.Finder {
		return func(
			_ context.Context,
			_ gorp.Tx,
			keys []secret.Key,
		) ([]secret.Use, error) {
			*asked = append(*asked, keys)
			return uses, nil
		}
	}
	use := func(t ontology.ResourceType) secret.Use {
		return secret.Use{Secret: uuid.New(), Resource: ontology.ID{Type: t}}
	}

	It("Should return the uses of every finder in order", func(ctx SpecContext) {
		var asked [][]secret.Key
		cert := use(ontology.ResourceTypeCertificate)
		task := use(ontology.ResourceTypeTask)
		find := secret.MultiFinder(finding(&asked, cert), finding(&asked, task))
		key := uuid.New()
		Expect(find(ctx, nil, []secret.Key{key})).
			To(Equal([]secret.Use{cert, task}))
		Expect(asked).To(Equal([][]secret.Key{{key}, {key}}))
	})

	It("Should return no uses without finders", func(ctx SpecContext) {
		Expect(secret.MultiFinder()(ctx, nil, []secret.Key{uuid.New()})).To(BeEmpty())
	})

	It("Should return the error of a finder", func(ctx SpecContext) {
		var asked [][]secret.Key
		find := secret.MultiFinder(
			func(context.Context, gorp.Tx, []secret.Key) ([]secret.Use, error) {
				return nil, errors.New("scan failed")
			},
			finding(&asked, use(ontology.ResourceTypeTask)),
		)
		Expect(find(ctx, nil, []secret.Key{uuid.New()})).Error().
			To(MatchError("scan failed"))
		Expect(asked).To(BeEmpty())
	})
})
