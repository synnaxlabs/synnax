// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package icd_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/x/types"
)

// A stored record lives under the key prefix of its type name, so a new name hides
// every record that a released Core stored.
var _ = Describe("Table name", func() {
	It("Should keep the table name of stored ICDs", func() {
		Expect(types.Name[icd.ICD]()).To(Equal("ICD"))
	})
})
