// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package validate_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Problems", func() {
	Describe("Addf", func() {
		It("Should join the path segments with dots", func() {
			var p validate.Problems
			p.Addf([]string{"entries", "0", "name"}, "duplicate name %q", "a")
			Expect(p).To(Equal(validate.Problems{{
				Path:    "entries.0.name",
				Message: `duplicate name "a"`,
			}}))
		})
	})

	Describe("AddError", func() {
		It("Should add every error of a joined validation error", func() {
			v := validate.New("doc")
			v.NotEmptyString("name", "")
			v.Exec(func() error {
				return validate.PathedError(
					validate.PathedError(errors.New("required"), "tag"),
					"entries", "1",
				)
			})
			var p validate.Problems
			p.AddError(v.Error())
			Expect(p).To(Equal(validate.Problems{
				{Path: "name", Message: "required"},
				{Path: "entries.1.tag", Message: "required"},
			}))
		})

		It("Should add an error without a path as a problem with the document", func() {
			var p validate.Problems
			p.AddError(errors.New("broken"))
			Expect(p).To(Equal(validate.Problems{{Message: "broken"}}))
		})

		It("Should add nothing for a nil error", func() {
			var p validate.Problems
			p.AddError(nil)
			Expect(p).To(BeEmpty())
		})
	})
})
