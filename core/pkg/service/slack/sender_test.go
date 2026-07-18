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
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	slk "github.com/synnaxlabs/synnax/pkg/service/slack"
)

// recordedRequest captures what the mock Slack server received.
type recordedRequest struct {
	authorization string
	body          map[string]any
}

// mockSlack is an httptest server standing in for the Slack Web API.
type mockSlack struct {
	server   *httptest.Server
	requests map[string]recordedRequest
	// ok controls the "ok" field returned for every endpoint.
	ok bool
	// errMsg is the "error" field returned when ok is false.
	errMsg string
}

func newMockSlack() *mockSlack {
	m := &mockSlack{requests: map[string]recordedRequest{}, ok: true}
	mux := http.NewServeMux()
	handler := func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		var body map[string]any
		if err := json.Unmarshal(raw, &body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		m.requests[r.URL.Path] = recordedRequest{
			authorization: r.Header.Get("Authorization"),
			body:          body,
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"ok":    m.ok,
			"error": m.errMsg,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
	mux.HandleFunc("/auth.test", handler)
	mux.HandleFunc("/chat.postMessage", handler)
	m.server = httptest.NewServer(mux)
	return m
}

var _ = Describe("Sender against a mock Slack server", func() {
	var (
		mock   *mockSlack
		sender slk.Sender
	)

	BeforeEach(func() {
		mock = newMockSlack()
		DeferCleanup(mock.server.Close)
		sender = slk.NewSender(mock.server.URL)
	})

	Describe("AuthTest", func() {
		It("Should call auth.test with the bearer token", func(ctx context.Context) {
			Expect(sender.AuthTest(ctx, "xoxb-good")).To(Succeed())
			req := mock.requests["/auth.test"]
			Expect(req.authorization).To(Equal("Bearer xoxb-good"))
		})

		It("Should return the Slack error when the token is invalid",
			func(ctx context.Context) {
				mock.ok = false
				mock.errMsg = "invalid_auth"
				Expect(sender.AuthTest(ctx, "xoxb-bad")).
					To(MatchError(ContainSubstring("invalid_auth")))
			})
	})

	Describe("Post", func() {
		It("Should post the message with the channel, token, and attachment",
			func(ctx context.Context) {
				Expect(sender.Post(ctx, "xoxb-good", slk.Message{
					Channel:  "#alerts",
					Fallback: "Pump: overpressure",
					Color:    "#e01e5a",
					Emoji:    "🔴",
					Headline: "Pump",
					Body:     "overpressure",
					Context:  "detail • now",
				})).To(Succeed())
				req := mock.requests["/chat.postMessage"]
				Expect(req.authorization).To(Equal("Bearer xoxb-good"))
				Expect(req.body).To(HaveKeyWithValue("channel", "#alerts"))
				Expect(req.body).To(HaveKey("attachments"))
			})

		It("Should return the Slack error when posting fails",
			func(ctx context.Context) {
				mock.ok = false
				mock.errMsg = "channel_not_found"
				Expect(sender.Post(ctx, "xoxb-good", slk.Message{Channel: "#nope"})).
					To(MatchError(ContainSubstring("channel_not_found")))
			})
	})
})
