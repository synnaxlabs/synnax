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

var _ = Describe("EditMap", func() {
	root := []string{"ports"}
	var (
		ports   map[string]port
		checked []string
	)
	check := func(key string, p port) error {
		checked = append(checked, key)
		if p.Port == 0 {
			return errors.Wrapf(validate.ErrValidation, "%s: the port is zero", key)
		}
		return nil
	}
	BeforeEach(func() {
		ports = map[string]port{"a": {Key: "a", Port: 1}, "b": {Key: "b"}}
		checked = nil
	})

	It("Should edit and check only the entry that the path names", func() {
		Expect(patch.EditMap(ports, root, patch.Set{
			Path:  []string{"ports", "a", "port"},
			Value: 2,
		}, check)).To(Equal(map[string]port{
			"a": {Key: "a", Port: 2},
			"b": {Key: "b"},
		}))
		Expect(checked).To(Equal([]string{"a"}))
	})

	It("Should return the error of check", func() {
		Expect(patch.EditMap(ports, root, patch.Set{
			Path:  []string{"ports", "a", "port"},
			Value: 0,
		}, check)).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("a: the port is zero")),
		))
	})

	It("Should delete the entry that the action removes", func() {
		Expect(patch.EditMap(ports, root, patch.Remove{
			Path: []string{"ports"},
			Key:  "b",
		}, check)).To(Equal(map[string]port{"a": {Key: "a", Port: 1}}))
		Expect(checked).To(BeEmpty())
	})

	It("Should check every entry when the action is at root", func() {
		Expect(patch.EditMap(ports, root, patch.Set{
			Path:  []string{"ports"},
			Value: map[string]port{"c": {Key: "c"}},
		}, check)).Error().To(MatchError(ContainSubstring("c: the port is zero")))
	})

	It("Should let a guard read another entry", func() {
		set := func(guard string) patch.Set {
			return patch.Set{
				Path:  []string{"ports", "a", "port"},
				Value: 3,
				Guards: []patch.Guard{
					{Path: []string{"ports", "b", "key"}, Value: guard},
				},
			}
		}
		Expect(patch.EditMap(ports, root, set("c"), check)).
			To(HaveKeyWithValue("a", port{Key: "a", Port: 1}))
		Expect(patch.EditMap(ports, root, set("b"), check)).
			To(HaveKeyWithValue("a", port{Key: "a", Port: 3}))
	})

	It("Should edit a map at a nested root", func() {
		Expect(patch.EditMap(ports, []string{"device", "ports"}, patch.Set{
			Path:  []string{"device", "ports", "b", "port"},
			Value: 4,
		}, check)).To(HaveKeyWithValue("b", port{Key: "b", Port: 4}))
	})

	It("Should add an entry to a nil map", func() {
		Expect(patch.EditMap[port](nil, root, patch.Set{
			Path:  []string{"ports", "a"},
			Value: port{Key: "a", Port: 1},
		}, nil)).To(Equal(map[string]port{"a": {Key: "a", Port: 1}}))
	})

	It("Should reject a path outside root", func() {
		Expect(patch.EditMap(ports, root, patch.Set{
			Path:  []string{"name"},
			Value: "dev",
		}, check)).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("the path is outside")),
		))
	})

	It("Should reject an entry that does not decode", func() {
		Expect(patch.EditMap(ports, root, patch.Set{
			Path:  []string{"ports", "a", "rack"},
			Value: 1,
		}, check)).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("decoding entries")),
		))
	})

	It("Should return the error of the action", func() {
		Expect(patch.EditMap(ports, root, patch.Set{
			Path:   []string{"ports", "a", "port"},
			Value:  2,
			Guards: []patch.Guard{{}},
		}, check)).Error().To(MatchError(ContainSubstring("a guard needs a path")))
	})
})
