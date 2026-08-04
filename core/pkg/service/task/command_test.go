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
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/task"
)

var _ = Describe("Command", func() {
	Describe("String", func() {
		var key = task.Key(uuid.MustParse("0192a1b2-c3d4-7e5f-8a9b-0c1d2e3f4a5b"))

		It("Should include the type, key, and target task", func() {
			c := task.Command{Type: "start", Key: "cmd-1", Task: key}
			Expect(c.String()).To(Equal(
				"start (key=cmd-1, task=" + key.String() + ")",
			))
		})
	})
})
