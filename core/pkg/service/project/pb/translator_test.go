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
	"testing"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/project/pb"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
)

func TestProjectPB(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Project PB Suite")
}

// sample builds a fully-populated project whose layout holds string and bool
// values that survive the structpb round-trip unchanged.
func sample() project.Project {
	return project.Project{
		Key:    uuid.New(),
		Name:   "ops",
		Layout: msgpack.EncodedJSON{"active": "plot-1", "pinned": true},
	}
}

var _ = Describe("Translator", func() {
	It("Should round-trip a project through protobuf", func() {
		p := sample()
		back := MustSucceed(pb.ProjectFromPB(MustSucceed(pb.ProjectToPB(p))))
		Expect(back).To(Equal(p))
	})

	It("Should round-trip a slice of projects", func() {
		ps := []project.Project{sample(), sample()}
		back := MustSucceed(pb.ProjectsFromPB(MustSucceed(pb.ProjectsToPB(ps))))
		Expect(back).To(Equal(ps))
	})

	It("Should return a zero project when decoding nil", func() {
		Expect(pb.ProjectFromPB(nil)).To(Equal(project.Project{}))
	})

	It("Should error when a layout is not structpb-encodable", func() {
		p := sample()
		p.Layout = msgpack.EncodedJSON{"bad": make(chan int)}
		Expect(pb.ProjectToPB(p)).Error().To(MatchError(ContainSubstring("invalid type")))
	})

	It("Should error when decoding a project with a malformed key", func() {
		Expect(pb.ProjectFromPB(&pb.Project{Key: "not-a-uuid"})).Error().
			To(MatchError(ContainSubstring("invalid UUID")))
	})
})
