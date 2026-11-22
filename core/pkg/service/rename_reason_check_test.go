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
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	groupv1 "github.com/synnaxlabs/synnax/pkg/service/group/versions/v1"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Rename reason check", func() {
	DescribeTable(
		"Should say why the upgrade renamed a channel",
		func(
			ctx SpecContext,
			folders []string,
			names []string,
			reason string,
		) {
			n := testutil.NewNode(ctx)
			stored := make([]groupv1.Group, len(folders))
			for i, name := range folders {
				stored[i] = groupv1.Group{
					Key:   uuid.New(),
					Name:  name,
					Scope: resource.TypeChannel,
				}
			}
			channels := make([]storedChannel, len(names))
			for i, name := range names {
				channels[i] = storedChannel{name: name}
			}
			createNamedV062Store(ctx, n, stored, channels...)
			MustOpen(svctestutil.OpenLayer(ctx, n))
			var renamed []status.Status[any]
			Expect(gorp.NewRetrieve[status.Key, status.Status[any]]().
				Where(gorp.Match(func(_ gorp.Context, s *status.Status[any]) (bool, error) {
					return s.Message == "Renamed channel", nil
				})).
				Entries(&renamed).
				Exec(ctx, n.DB)).To(Succeed())
			Expect(renamed).To(HaveLen(1))
			Expect(renamed[0].Description).To(ContainSubstring(reason))
		},
		Entry(
			"a '.'",
			[]string{},
			[]string{"stand.pt"},
			"a '.' now separates the folders",
		),
		Entry(
			"bytes that are not text",
			[]string{},
			[]string{"pt\xff01"},
			"not valid text",
		),
		Entry(
			"no visible character",
			[]string{},
			[]string{"\u200b"},
			"no visible character",
		),
		Entry(
			"an invisible character",
			[]string{},
			[]string{"pt\u200b01"},
			"shows as nothing",
		),
		Entry("a duplicate", []string{}, []string{"pressure", "pressure"},
			"another channel or a folder beside it held the same name"),
		Entry("the Metrics folder", []string{"Metrics"}, []string{"Metrics"},
			"another channel or a folder beside it held the same name"),
	)
})
