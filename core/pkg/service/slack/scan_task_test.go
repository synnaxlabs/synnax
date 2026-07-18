// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package slack_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	slk "github.com/synnaxlabs/synnax/pkg/service/slack"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("ScanTask", func() {
	var (
		sender  *mockSender
		factory driver.Factory
	)

	BeforeEach(func() {
		sender = newMockSender()
		factory = MustSucceed(slk.NewFactory(slk.FactoryConfig{
			Status: statusSvc, Device: deviceSvc, Sender: sender,
		}))
	})

	configureScan := func(ctx context.Context, key task.Key) driver.Task {
		t := task.Task{Key: key, Name: "Slack Scanner", Type: slk.ScanTaskType}
		return MustSucceed(factory.ConfigureTask(ctx, t))
	}

	testCmd := func(key task.Key, token string) task.Command {
		return task.Command{
			Task: key,
			Type: slk.TestConnectionCommandType,
			Key:  "cmd-" + key.String(),
			Args: msgpack.EncodedJSON{"token": token},
		}
	}

	It("Should validate a token and report success keyed by the command",
		func(ctx context.Context) {
			tsk := configureScan(ctx, 200)
			cmd := testCmd(200, "xoxb-good")
			Expect(tsk.Exec(ctx, cmd)).To(Succeed())
			Expect(sender.lastAuthToken()).To(Equal("xoxb-good"))
			stat := taskStatus(ctx, 200)
			Expect(stat.Variant).To(BeEquivalentTo("success"))
			Expect(stat.Details.Cmd).To(Equal(cmd.Key))
		})

	It("Should report an error keyed by the command when the token is invalid",
		func(ctx context.Context) {
			sender.setAuthError(errors.New("invalid_auth"))
			tsk := configureScan(ctx, 201)
			cmd := testCmd(201, "xoxb-bad")
			Expect(tsk.Exec(ctx, cmd)).To(Succeed())
			stat := taskStatus(ctx, 201)
			Expect(stat.Variant).To(BeEquivalentTo("error"))
			Expect(stat.Message).To(ContainSubstring("invalid_auth"))
			Expect(stat.Details.Cmd).To(Equal(cmd.Key))
		})

	It("Should reject an unsupported command", func(ctx context.Context) {
		tsk := configureScan(ctx, 202)
		Expect(tsk.Exec(ctx, task.Command{Task: 202, Type: "bogus"})).
			To(MatchError(driver.ErrUnsupportedCommand))
	})
})
