// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package service_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	groupv1 "github.com/synnaxlabs/synnax/pkg/service/group/versions/v1"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Name clash check", func() {
	It("Should keep a top folder and a channel in another folder that hold one name",
		func(ctx SpecContext) {
			n := testutil.NewNode(ctx)
			box := groupv1.Group{
				Key:   uuid.New(),
				Name:  "box",
				Scope: resource.TypeChannel,
			}
			pt := groupv1.Group{
				Key:   uuid.New(),
				Name:  "pt",
				Scope: resource.TypeChannel,
			}
			keys := createNamedV062Store(
				ctx,
				n,
				[]groupv1.Group{box, pt},
				storedChannel{name: "pt", group: &box.Key},
			)
			MustOpen(svctestutil.OpenLayer(ctx, n))
			Expect(retrieveFolder(ctx, n.DB, pt.Key).Name).To(Equal("pt"))
			Expect(retrieveChannel(ctx, n.DB, keys[0]).Name).To(Equal("pt"))
		})
})
