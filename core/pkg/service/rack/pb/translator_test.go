// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pb_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/rack/pb"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Translator", func() {
	It("Should round trip a rack with a status", func() {
		r := rack.Rack{
			Key:  rack.NewKey(1, 2),
			Name: "Rack",
			Status: &rack.Status{
				Key:     "status",
				Owner:   new(rack.NewKey(1, 2).ResourceID()),
				Variant: status.VariantSuccess,
				Message: "Driver running",
				Labels:  []label.Key{label.Key(uuid.New())},
				Details: rack.StatusDetails{Running: true},
			},
		}
		translated := MustSucceed(pb.RackToPB(r))
		Expect(pb.RackFromPB(translated)).To(Equal(r))
	})
	It("Should leave the role out, as the Core assigns it", func() {
		r := rack.Rack{Key: rack.NewKey(1, 2), Name: "Rack", Role: uuid.New()}
		translated := MustSucceed(pb.RackToPB(r))
		Expect(pb.RackFromPB(translated)).
			To(Equal(rack.Rack{Key: r.Key, Name: r.Name}))
	})
})
