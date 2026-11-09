// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/plugin/ts/types"
)

var _ = DescribeTable("FieldName",
	func(name, expected string) { Expect(types.FieldName(name)).To(Equal(expected)) },
	Entry("an empty name", "", ""),
	Entry("a snake case name", "created_at", "createdAt"),
	Entry("a trailing acronym", "EntityID", "entityID"),
	Entry("a trailing two-letter run", "ClientXY", "clientXY"),
	Entry("a trailing single capital", "PointA", "pointA"),
	Entry("a name that is all acronym", "ID", "id"),
)
