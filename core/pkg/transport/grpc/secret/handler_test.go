// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package secret_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/api/secret"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	grpcsecret "github.com/synnaxlabs/synnax/pkg/transport/grpc/secret"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Resolve", func() {
	var (
		client   grpcsecret.SecretResolveServiceClient
		received secret.ResolveRequest
		res      secret.ResolveResponse
	)
	BeforeEach(func() {
		client = grpcsecret.NewSecretResolveServiceClient(conn)
		transport.SecretResolve.BindHandler(func(
			_ context.Context,
			req secret.ResolveRequest,
		) (secret.ResolveResponse, error) {
			received = req
			return res, nil
		})
	})
	It("Should decode a grant and a task, and encode the command, secrets, and values",
		func(ctx SpecContext) {
			g, t, k, file := uuid.New(), uuid.New(), uuid.New(), uuid.New()
			res = secret.ResolveResponse{
				Command: task.Command{Task: t, Type: "test", Key: "cmd"},
				Secrets: []secret.Key{k, file},
				Values:  []secret.Value{{Key: k, Value: "v"}},
			}
			out := MustSucceed(client.Exec(ctx, &grpcsecret.ResolveRequest{
				Grant: g.String(),
				Task:  t.String(),
			}))
			Expect(received).To(Equal(secret.ResolveRequest{
				Grant:   g,
				Task:    t,
				Secrets: []secret.Key{},
			}))
			Expect(out.Command.Task).To(Equal(t.String()))
			Expect(out.Command.Type).To(Equal("test"))
			Expect(out.Command.Key).To(Equal("cmd"))
			Expect(out.Secrets).To(Equal([]string{k.String(), file.String()}))
			Expect(out.Values).To(HaveLen(1))
			Expect(out.Values[0].Key).To(Equal(k.String()))
			Expect(out.Values[0].Value).To(Equal("v"))
		},
	)
	It("Should decode an empty grant and task as zero keys", func(ctx SpecContext) {
		k := uuid.New()
		res = secret.ResolveResponse{}
		MustSucceed(client.Exec(ctx, &grpcsecret.ResolveRequest{
			Secrets: []string{k.String()},
		}))
		Expect(received).To(Equal(secret.ResolveRequest{Secrets: []secret.Key{k}}))
	})
	It("Should reject a secret key that is not a UUID", func(ctx SpecContext) {
		Expect(client.Exec(ctx, &grpcsecret.ResolveRequest{
			Secrets: []string{"nope"},
		})).Error().To(MatchError(ContainSubstring("invalid uuid")))
	})
})
