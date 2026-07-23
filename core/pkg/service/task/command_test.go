// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package task_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/task"
)

var _ = Describe("Command", func() {
	Describe("String", func() {
		// key composes rack key 65538 (node 1, local 2) in the high 32 bits and
		// local task key 7 in the low 32.
		const key = task.Key(65538<<32 | 7)

		It("Should include the type, key, and target task", func() {
			c := task.Command{Type: "start", Key: "cmd-1", Task: key}
			Expect(c.String()).To(Equal(
				"start (key=cmd-1, task=" + key.String() + ")",
			))
		})
	})
})
