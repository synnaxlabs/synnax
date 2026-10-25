// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package patch_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/patch"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

type device struct {
	Name  string `json:"name"`
	Ports []port `json:"ports"`
}

var _ = Describe("Edit", func() {
	doc := device{Name: "dev", Ports: []port{{Key: "a", Port: 1}}}

	It("Should apply the action to the JSON form of the document", func() {
		next := MustSucceed(patch.Edit(doc, patch.Set{
			Path:  []string{"ports", "a"},
			Value: port{Key: "a", Port: 2},
		}.Apply))
		Expect(next).To(Equal(device{Name: "dev", Ports: []port{{Key: "a", Port: 2}}}))
		Expect(doc.Ports[0].Port).To(Equal(uint8(1)))
	})

	It("Should reject a member that the document does not have", func() {
		Expect(patch.Edit(doc, patch.Set{Path: []string{"rack"}, Value: 1}.Apply)).
			Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(`unknown object member name "rack"`)),
		))
	})

	It("Should reject a value that the document cannot hold", func() {
		Expect(patch.Edit(doc, patch.Set{Path: []string{"name"}, Value: 1}.Apply)).
			Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("decoding document")),
		))
	})

	It("Should return the error of the function", func() {
		err := errors.New("failed")
		Expect(patch.Edit(doc, func(map[string]any) error { return err })).
			Error().To(MatchError(err))
	})

	It("Should reject a document that JSON cannot encode", func() {
		Expect(patch.Edit(make(chan int), func(map[string]any) error { return nil })).
			Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("encoding document")),
		))
	})
})
