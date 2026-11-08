// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package compare_test

import (
	"encoding/json/v2"
	"fmt"
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/compare"
	. "github.com/synnaxlabs/x/testutil"
)

const textVectorsPath = "../../testdata/compare/text.json"

var _ = Describe("TextKey", func() {
	var vectors struct {
		Ordered []string   `json:"ordered"`
		Equal   [][]string `json:"equal"`
	}
	Expect(json.Unmarshal(MustSucceed(os.ReadFile(textVectorsPath)), &vectors)).
		To(Succeed())
	for i := 1; i < len(vectors.Ordered); i++ {
		a, b := vectors.Ordered[i-1], vectors.Ordered[i]
		It(fmt.Sprintf("Should order %q before %q", a, b), func() {
			Expect(
				strings.Compare(compare.TextKey(a), compare.TextKey(b)),
			).To(Equal(-1))
		})
	}
	for _, group := range vectors.Equal {
		It(fmt.Sprintf("Should give %q one key", group), func() {
			for _, s := range group[1:] {
				Expect(compare.TextKey(s)).To(Equal(compare.TextKey(group[0])))
			}
		})
	}
})
