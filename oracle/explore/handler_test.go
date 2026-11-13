// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package explore_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/explore"
	"github.com/synnaxlabs/x/errors"
)

func serve(load explore.Loader, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	explore.Handler(load).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

var _ = Describe("Handler", func() {
	model := explore.Model{Types: []explore.Type{{ID: "widget.Rack", Kind: "struct"}}}
	load := func(context.Context) (explore.Model, error) { return model, nil }

	It("Should serve the page at the root", func() {
		rec := serve(load, "/")
		Expect(rec.Code).To(Equal(http.StatusOK))
		Expect(rec.Header().Get("Content-Type")).To(Equal("text/html; charset=utf-8"))
		Expect(
			rec.Body.String(),
		).To(ContainSubstring("<title>Oracle schema map</title>"))
	})

	It("Should serve the loaded model as JSON", func() {
		rec := serve(load, "/model.json")
		Expect(rec.Code).To(Equal(http.StatusOK))
		var got explore.Model
		Expect(json.Unmarshal(rec.Body.Bytes(), &got)).To(Succeed())
		Expect(got).To(Equal(model))
	})

	It("Should report a failed load", func() {
		rec := serve(func(context.Context) (explore.Model, error) {
			return explore.Model{}, errors.New("unknown type widget.Missing")
		}, "/model.json")
		Expect(rec.Code).To(Equal(http.StatusUnprocessableEntity))
		Expect(rec.Body.String()).To(ContainSubstring("unknown type widget.Missing"))
	})

	It("Should not serve other paths", func() {
		Expect(serve(load, "/other").Code).To(Equal(http.StatusNotFound))
	})
})
