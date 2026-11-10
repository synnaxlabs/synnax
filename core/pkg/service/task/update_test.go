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
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Update", func() {
	It("Should replace the config whole", func() {
		t := task.Task{
			Name:   "Read",
			Config: msgpack.EncodedJSON{"sample_rate": 10, "stream_rate": 5},
		}
		next := MustSucceed(task.Reduce(t, task.NewUpdateAction(task.UpdatePayload{
			Value: task.Update{Config: msgpack.EncodedJSON{"sample_rate": 20}},
		})))
		Expect(next.Config).To(Equal(msgpack.EncodedJSON{"sample_rate": 20}))
		Expect(next.Name).To(Equal("Read"))
	})

	It("Should refuse a change to a field that only the Core sets", func() {
		p := task.UpdatePayload{Value: task.Update{ConfigHash: new("abc")}}
		err := p.CheckCoreOnly()
		Expect(errors.Is(err, actions.ErrCoreOnly)).To(BeTrue())
		Expect(err).To(And(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("config_hash is set by the Core")),
		))
	})

	It("Should allow a change to fields that clients set", func() {
		p := task.UpdatePayload{Value: task.Update{Name: new("Write")}}
		Expect(p.CheckCoreOnly()).To(Succeed())
	})
})
