// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package freighter_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
)

var _ = DescribeTable("Role", func(r freighter.Role, s string) {
	Expect(r.String()).To(Equal(s))
},
	Entry("Client", freighter.Client, "Client"),
	Entry("Server", freighter.Server, "Server"),
	Entry("Invalid", freighter.Role(0), "Role(0)"),
)
