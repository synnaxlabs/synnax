// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/synnaxlabs/x/errors"
)

// DefaultBaseURL is the base URL of the Slack Web API.
const DefaultBaseURL = "https://slack.com/api"

// Message is a Slack message posted to a channel. buildMessage constructs it from a
// status change; the Sender renders it to the chat.postMessage wire format.
type Message struct {
	// Channel is the target channel name or ID.
	Channel string
	// Fallback is the plain-text notification shown in previews and by clients that
	// cannot render attachments.
	Fallback string
	// Color is the hex color of the attachment's side bar.
	Color string
	// Emoji prefixes the headline to convey severity at a glance.
	Emoji string
	// Headline is the status name, rendered as the attachment header.
	Headline string
	// Body is the status message.
	Body string
	// Context is a secondary line holding the description and timestamp.
	Context string
}

// Sender performs outbound Slack calls. It is an injected seam so tests substitute a
// fake for the real Slack Web API.
type Sender interface {
	// Post sends msg to Slack, authenticating with the workspace bot token.
	Post(ctx context.Context, token string, msg Message) error
	// AuthTest verifies that the bot token is valid, returning an error otherwise.
	AuthTest(ctx context.Context, token string) error
}

type defaultSenderImpl struct {
	client  *http.Client
	baseURL string
}

// NewSender returns the default Slack Sender targeting baseURL. Production passes
// DefaultBaseURL; tests point it at a mock server.
func NewSender(baseURL string) Sender {
	return defaultSenderImpl{client: http.DefaultClient, baseURL: baseURL}
}

// Post renders msg to Block Kit JSON and sends it to chat.postMessage.
func (d defaultSenderImpl) Post(ctx context.Context, token string, msg Message) error {
	return d.call(ctx, token, d.baseURL+"/chat.postMessage", renderPayload(msg))
}

// AuthTest calls auth.test to verify the token authenticates a workspace.
func (d defaultSenderImpl) AuthTest(ctx context.Context, token string) error {
	return d.call(ctx, token, d.baseURL+"/auth.test", map[string]any{})
}

// call POSTs payload to a Slack Web API method. Slack returns HTTP 200 with an "ok"
// field even on logical failures, so the response body is checked rather than the
// status code.
func (d defaultSenderImpl) call(
	ctx context.Context,
	token, url string,
	payload map[string]any,
) (err error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := res.Body.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	var parsed struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	if err = json.Unmarshal(raw, &parsed); err != nil {
		return err
	}
	if !parsed.OK {
		return errors.Newf("slack request failed: %s", parsed.Error)
	}
	return nil
}

// renderPayload builds the chat.postMessage request body: a colored attachment holding
// a header, a body section, and a context line.
func renderPayload(msg Message) map[string]any {
	header := strings.TrimSpace(msg.Emoji + " " + msg.Headline)
	blocks := []map[string]any{{
		"type": "header",
		"text": map[string]any{"type": "plain_text", "text": header, "emoji": true},
	}}
	if msg.Body != "" {
		blocks = append(blocks, map[string]any{
			"type": "section",
			"text": map[string]any{"type": "mrkdwn", "text": msg.Body},
		})
	}
	if msg.Context != "" {
		blocks = append(blocks, map[string]any{
			"type":     "context",
			"elements": []map[string]any{{"type": "mrkdwn", "text": msg.Context}},
		})
	}
	return map[string]any{
		"channel":     msg.Channel,
		"text":        msg.Fallback,
		"attachments": []map[string]any{{"color": msg.Color, "blocks": blocks}},
	}
}

var defaultSender = NewSender(DefaultBaseURL)
