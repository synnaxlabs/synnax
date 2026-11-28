// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gorp_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
)

var _ = Describe("RetargetEach", func() {
	type labeled struct {
		Label string
		Note  string
	}
	retarget := func(s []labeled, to map[string]string) []labeled {
		return gorp.RetargetEach(
			s,
			to,
			func(l labeled) string { return l.Label },
			func(l *labeled, t string) { l.Label = t },
		)
	}

	It("Should move each element whose target moves", func() {
		Expect(retarget(
			[]labeled{{Label: "a", Note: "1"}, {Label: "b", Note: "2"}},
			map[string]string{"a": "c"},
		)).To(Equal([]labeled{{Label: "c", Note: "1"}, {Label: "b", Note: "2"}}))
	})

	It("Should swap two targets at once", func() {
		Expect(retarget(
			[]labeled{{Label: "a"}, {Label: "b"}},
			map[string]string{"a": "b", "b": "a"},
		)).To(Equal([]labeled{{Label: "b"}, {Label: "a"}}))
	})

	It("Should drop a moved element whose new target an element holds", func() {
		Expect(retarget(
			[]labeled{{Label: "a", Note: "1"}, {Label: "b", Note: "2"}},
			map[string]string{"a": "b"},
		)).To(Equal([]labeled{{Label: "b", Note: "2"}}))
	})

	It("Should keep one element when two move to the same target", func() {
		Expect(retarget(
			[]labeled{{Label: "a", Note: "1"}, {Label: "b", Note: "2"}},
			map[string]string{"a": "c", "b": "c"},
		)).To(Equal([]labeled{{Label: "c", Note: "1"}}))
	})

	It("Should keep repeats that no move made", func() {
		Expect(retarget(
			[]labeled{{Label: "a"}, {Label: "a"}, {Label: "b"}},
			map[string]string{"b": "c"},
		)).To(Equal([]labeled{{Label: "a"}, {Label: "a"}, {Label: "c"}}))
	})

	It("Should return the slice itself when no element moves", func() {
		s := []labeled{{Label: "a"}}
		out := retarget(s, map[string]string{"b": "c"})
		Expect(&out[0]).To(BeIdenticalTo(&s[0]))
	})
})
