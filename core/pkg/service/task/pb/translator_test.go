// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pb_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/synnax/pkg/service/task/pb"
	"github.com/synnaxlabs/x/encoding/msgpack"
	xpb "github.com/synnaxlabs/x/pb"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Task", func() {
	details := task.StatusDetails{
		Task:       task.Key(uuid.New()),
		Running:    true,
		Cmd:        "start",
		ConfigHash: "abc123",
		Rack:       7,
		Data:       msgpack.EncodedJSON{"read_rate": 10.0},
	}
	t := task.Task{
		Key:    details.Task,
		Rack:   details.Rack,
		Name:   "read",
		Type:   "ni_analog_read",
		Config: msgpack.EncodedJSON{},
		Status: &task.Status{
			Key:     "status",
			Variant: status.VariantSuccess,
			Time:    telem.Now(),
			Details: details,
		},
	}

	It("Should keep every status detail through the protobuf form", func() {
		translated := MustSucceed(pb.TaskFromPB(MustSucceed(pb.TaskToPB(t))))
		Expect(translated.Status.Details).To(Equal(details))
	})

	It("Should name each status detail as its JSON tag does", func() {
		translated := MustSucceed(pb.TaskToPB(t))
		Expect(xpb.AnyFromPBAny(translated.Status.Details)).To(And(
			HaveKeyWithValue("config_hash", "abc123"),
			Not(HaveKey("configHash")),
		))
	})
})
