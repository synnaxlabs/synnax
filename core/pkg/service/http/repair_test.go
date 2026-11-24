// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package http_test

import (
	"maps"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/http"
	tasktestutil "github.com/synnaxlabs/synnax/pkg/service/task/testutil"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Upgrade repair", func() {
	var (
		svc   *http.Service
		tasks *tasktestutil.Tasks
		dev   string
	)
	BeforeEach(func(ctx SpecContext) {
		svc = MustOpen(http.OpenService(ctx, http.ServiceConfig{
			DB:      db,
			Channel: channelSvc,
			ICD:     icdSvc,
		}))
		tasks = MustOpen(tasktestutil.OpenTasks(ctx, tasktestutil.TasksConfig{
			DB:     db,
			Status: statusSvc,
			Stores: svc.Stores(),
		}))
		dev = uuid.New().String()
		Expect(tasks.CreateDevice(ctx, dev)).To(Succeed())
	})

	// shipped returns a write config in the shape that a released Core stored.
	shipped := func(endpoint map[string]any) msgpack.EncodedJSON {
		ep := map[string]any{
			"key":          "ep",
			"method":       "POST",
			"path":         "/cmd",
			"headers":      []any{},
			"query_params": []any{},
			"channel":      map[string]any{"pointer": "/value", "json_type": "number"},
			"fields":       []any{},
		}
		maps.Copy(ep, endpoint)
		return msgpack.EncodedJSON{"device": dev, "endpoints": []any{ep}}
	}
	// upgrade runs a shipped config through the import repair and the save checks, as
	// the task upgrade does.
	upgrade := func(ctx SpecContext, data msgpack.EncodedJSON) error {
		normalized, err := svc.Write.Normalize(ctx, db, 2, data)
		if err != nil {
			return err
		}
		_, err = svc.Write.Prepare(ctx, db, uuid.New(), "Valve", normalized)
		return err
	}
	first := func(normalized msgpack.EncodedJSON) map[string]any {
		GinkgoHelper()
		eps, ok := normalized["endpoints"].([]any)
		Expect(ok).To(BeTrue())
		ep, ok := eps[0].(map[string]any)
		Expect(ok).To(BeTrue())
		return ep
	}
	endpoint := func(ctx SpecContext, data msgpack.EncodedJSON) map[string]any {
		GinkgoHelper()
		return first(MustSucceed(svc.Write.Normalize(ctx, db, 2, data)))
	}

	DescribeTable("Should repair a row that the Driver skipped or sent as is",
		func(
			ctx SpecContext,
			in map[string]any,
			path string,
			headers []any,
			params []any,
		) {
			data := shipped(in)
			ep := endpoint(ctx, data)
			Expect(ep["path"]).To(Equal(path))
			Expect(ep["headers"]).To(Equal(headers))
			Expect(ep["query_params"]).To(Equal(params))
			Expect(upgrade(ctx, shipped(in))).To(Succeed())
		},
		Entry(
			"a header with no name",
			map[string]any{"headers": []any{
				map[string]any{"name": "", "value": "x"},
				map[string]any{"name": "Accept", "value": "json"},
			}},
			"/cmd",
			[]any{map[string]any{"name": "Accept", "value": "json"}},
			[]any{},
		),
		Entry(
			"a query parameter with no name",
			map[string]any{"query_params": []any{
				map[string]any{"parameter": "", "value": "1"},
			}},
			"/cmd",
			[]any{},
			[]any{},
		),
		Entry(
			"a query in the path",
			map[string]any{"path": "/cmd?a=1&units=si"},
			"/cmd",
			[]any{},
			[]any{
				map[string]any{"parameter": "a", "value": "1"},
				map[string]any{"parameter": "units", "value": "si"},
			},
		),
		Entry(
			"an unsorted query in the path",
			map[string]any{"path": "/cmd?units=si&a=1"},
			"/cmd",
			[]any{},
			[]any{
				map[string]any{"parameter": "units", "value": "si"},
				map[string]any{"parameter": "a", "value": "1"},
			},
		),
		Entry(
			"an encoded query in the path",
			map[string]any{"path": "/cmd?city=New%20York&tag=a%2Bb"},
			"/cmd",
			[]any{},
			[]any{
				map[string]any{"parameter": "city", "value": "New York"},
				map[string]any{"parameter": "tag", "value": "a+b"},
			},
		),
	)

	DescribeTable("Should set aside a row that could never have worked",
		func(ctx SpecContext, in map[string]any, path string) {
			Expect(upgrade(ctx, shipped(in))).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(path)),
			))
		},
		Entry(
			"a header name that is not a token",
			map[string]any{"headers": []any{
				map[string]any{"name": "X Key", "value": "1"},
			}},
			"name",
		),
		Entry(
			"a header value with a line break",
			map[string]any{"headers": []any{
				map[string]any{"name": "Accept", "value": "a\r\nb"},
			}},
			"value",
		),
		Entry(
			"a malformed pointer",
			map[string]any{"channel": map[string]any{
				"pointer":   "value",
				"json_type": "number",
			}},
			"pointer",
		),
	)

	It("Should repair a shipped read config", func(ctx SpecContext) {
		data := msgpack.EncodedJSON{
			"device": dev,
			"rate":   1,
			"endpoints": []any{map[string]any{
				"key":          "ep",
				"method":       "GET",
				"path":         "/now?units=si",
				"headers":      []any{map[string]any{"name": "", "value": "x"}},
				"query_params": []any{},
				"fields":       []any{},
			}},
		}
		ep := first(MustSucceed(svc.Read.Normalize(ctx, db, 2, data)))
		Expect(ep["path"]).To(Equal("/now"))
		Expect(ep["headers"]).To(Equal([]any{}))
		Expect(ep["query_params"]).To(Equal([]any{
			map[string]any{"parameter": "units", "value": "si"},
		}))
	})

	It("Should set aside a query in the path that repeats a name",
		func(ctx SpecContext) {
			data := shipped(map[string]any{"path": "/cmd?a=1&a=2"})
			Expect(endpoint(ctx, data)["path"]).To(Equal("/cmd?a=1&a=2"))
			Expect(upgrade(ctx, data)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("path")),
			))
		},
	)
})
