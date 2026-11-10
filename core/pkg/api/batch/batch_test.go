// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package batch_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/batch"
	"github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

type setRequest struct {
	Key   string `json:"key"   msgpack:"key"`
	Value string `json:"value" msgpack:"value"`
}

type setResponse struct {
	Previous string `json:"previous" msgpack:"previous"`
}

var errRefused = errors.New("refused")

var _ = Describe("Batch", func() {
	var (
		db   *gorp.DB
		svc  *batch.Service
		sets int
	)
	BeforeEach(func() {
		db = DeferClose(gorp.Wrap(memkv.New()))
		sets = 0
		set := func(ctx context.Context, tx gorp.Tx, req setRequest) (setResponse, error) {
			prev, closer, err := tx.Get(ctx, []byte(req.Key))
			if err == nil {
				defer func() { Expect(closer.Close()).To(Succeed()) }()
			}
			sets++
			res := setResponse{Previous: string(prev)}
			return res, tx.Set(ctx, []byte(req.Key), []byte(req.Value))
		}
		refuse := func(context.Context, gorp.Tx, setRequest) (struct{}, error) {
			return struct{}{}, errRefused
		}
		svc = batch.NewService(batch.Routes{
			"/set":    batch.Route(set),
			"/refuse": batch.Route(refuse),
		})
	})
	decode := func(ctx context.Context, body string) batch.ExecRequest {
		GinkgoHelper()
		var req batch.ExecRequest
		Expect(json.Codec.Decode([]byte(body), &req)).To(Succeed())
		return req
	}
	exec := func(ctx context.Context, req batch.ExecRequest) (batch.ExecResponse, error) {
		var res batch.ExecResponse
		err := db.WithTx(ctx, func(tx gorp.Tx) error {
			var err error
			res, err = svc.Exec(ctx, tx, req)
			return err
		})
		return res, err
	}
	stored := func(ctx context.Context, key string) string {
		GinkgoHelper()
		v, closer, err := db.Get(ctx, []byte(key))
		if err != nil {
			return ""
		}
		defer func() { Expect(closer.Close()).To(Succeed()) }()
		return string(v)
	}

	It(
		"Should run the requests in order and return each response",
		func(ctx SpecContext) {
			res := MustSucceed(exec(ctx, decode(ctx, `{"requests": [
			{"endpoint": "/set", "payload": {"key": "a", "value": "1"}},
			{"endpoint": "/set", "payload": {"key": "a", "value": "2"}}
		]}`)))
			Expect(res.Responses).To(Equal([]any{
				setResponse{Previous: ""},
				setResponse{Previous: "1"},
			}))
			Expect(stored(ctx, "a")).To(Equal("2"))
		},
	)

	It("Should decode MessagePack payloads", func(ctx SpecContext) {
		body := MustSucceed(msgpack.Codec.Encode(map[string]any{
			"requests": []map[string]any{{
				"endpoint": "/set",
				"payload":  map[string]any{"key": "b", "value": "3"},
			}},
		}))
		var req batch.ExecRequest
		Expect(msgpack.Codec.Decode(body, &req)).To(Succeed())
		MustSucceed(exec(ctx, req))
		Expect(stored(ctx, "b")).To(Equal("3"))
	})

	It("Should stop at the first error and commit nothing", func(ctx SpecContext) {
		Expect(exec(ctx, decode(ctx, `{"requests": [
			{"endpoint": "/set", "payload": {"key": "c", "value": "1"}},
			{"endpoint": "/refuse", "payload": {}},
			{"endpoint": "/set", "payload": {"key": "d", "value": "1"}}
		]}`))).Error().To(SatisfyAll(
			MatchError(errRefused),
			MatchError("requests.1: refused"),
		))
		Expect(sets).To(Equal(1))
		Expect(stored(ctx, "c")).To(BeEmpty())
		Expect(stored(ctx, "d")).To(BeEmpty())
	})

	It(
		"Should refuse an unknown endpoint before running anything",
		func(ctx SpecContext) {
			Expect(exec(ctx, decode(ctx, `{"requests": [
			{"endpoint": "/set", "payload": {"key": "e", "value": "1"}},
			{"endpoint": "/channel/create", "payload": {}}
		]}`))).Error().To(MatchError(ContainSubstring(
				`requests.1.endpoint: "/channel/create" cannot run in a batch`,
			)))
			Expect(sets).To(BeZero())
		},
	)

	It("Should reject a request with no payload", func(ctx SpecContext) {
		Expect(exec(ctx, decode(ctx, `{"requests": [{"endpoint": "/set"}]}`))).
			Error().To(MatchError("requests.0.payload: required: validation error"))
	})

	It("Should reject a payload that does not decode", func(ctx SpecContext) {
		Expect(exec(ctx, decode(ctx, `{"requests": [
			{"endpoint": "/set", "payload": {"key": 12}}
		]}`))).Error().To(SatisfyAll(
			HaveField("Path", Equal([]string{"requests", "0", "payload"})),
			HaveField("Err", MatchError(validate.ErrValidation)),
		))
	})
})
