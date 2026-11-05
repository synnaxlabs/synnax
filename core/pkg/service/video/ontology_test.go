// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package video_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/video"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Ontology", func() {
	It("Should return the video type", func() {
		Expect(svc.Type()).To(Equal(ontology.ResourceTypeVideo))
	})

	It("Should return the matching views, closest match first", func(ctx SpecContext) {
		create := func(name string) ontology.ID {
			v := video.Video{Name: name}
			Expect(svc.NewWriter(tx).Create(ctx, proj.Key, &v)).To(Succeed())
			return video.OntologyID(v.Key)
		}
		a := create("chamber camera")
		b := create("chamber")
		Expect(MustSucceed(svc.Search(ctx, tx, "chamber", 0)).Keys).
			To(Equal([]ontology.ID{b, a}))
	})

	It("Should return the ontology IDs of views", func() {
		v := video.Video{Name: "camera"}
		Expect(video.OntologyIDsFromVideos([]video.Video{v})).
			To(Equal([]ontology.ID{video.OntologyID(v.Key)}))
	})
})
