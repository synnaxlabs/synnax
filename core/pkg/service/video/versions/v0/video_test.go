// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	v0 "github.com/synnaxlabs/synnax/pkg/service/video/versions/v0"
)

var _ = Describe("Video", func() {
	It("Should use its key as the gorp key", func() {
		key := uuid.New()
		Expect(v0.Video{Key: key}.GorpKey()).To(Equal(key))
	})

	It("Should return no options", func() {
		Expect(v0.Video{}.SetOptions()).To(BeNil())
	})

	It("Should return its ontology ID", func() {
		key := uuid.New()
		Expect(v0.Video{Key: key}.OntologyID()).To(Equal(ontology.ID{
			Type: ontology.ResourceTypeVideo,
			Key:  key.String(),
		}))
	})
})
