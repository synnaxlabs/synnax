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
	"strings"
	"time"

	"github.com/synnaxlabs/synnax/pkg/service/status"
)

// greyColor is the attachment color for variants without a severity color.
const greyColor = "#868686"

// buildMessage constructs a Slack Message from a watched status change.
func buildMessage(channel string, s status.Status[any]) Message {
	return Message{
		Channel:  channel,
		Fallback: fallback(s),
		Color:    variantColor(s.Variant),
		Emoji:    variantEmoji(s.Variant),
		Headline: s.Name,
		Body:     s.Message,
		Context:  contextLine(s),
	}
}

// fallback is the plain-text notification, combining name and message when both exist.
func fallback(s status.Status[any]) string {
	switch {
	case s.Name == "":
		return s.Message
	case s.Message == "":
		return s.Name
	default:
		return s.Name + ": " + s.Message
	}
}

// contextLine joins the description (when present) and the timestamp.
func contextLine(s status.Status[any]) string {
	ts := s.Time.Time().Format(time.RFC3339)
	if s.Description == "" {
		return ts
	}
	return strings.Join([]string{s.Description, ts}, " • ")
}

var variantColors = map[status.Variant]string{
	status.VariantSuccess:  "#2eb67d",
	status.VariantInfo:     "#4a9eff",
	status.VariantWarning:  "#ecb22e",
	status.VariantError:    "#e01e5a",
	status.VariantLoading:  greyColor,
	status.VariantDisabled: greyColor,
}

func variantColor(v status.Variant) string {
	if c, ok := variantColors[v]; ok {
		return c
	}
	return greyColor
}

var variantEmojis = map[status.Variant]string{
	status.VariantSuccess:  "🟢",
	status.VariantInfo:     "🔵",
	status.VariantWarning:  "🟠",
	status.VariantError:    "🔴",
	status.VariantLoading:  "⏳",
	status.VariantDisabled: "⚪",
}

func variantEmoji(v status.Variant) string { return variantEmojis[v] }
