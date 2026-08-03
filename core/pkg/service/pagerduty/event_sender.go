// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pagerduty

import (
	"context"

	"github.com/PagerDuty/go-pagerduty"
)

// EventSender abstracts PagerDuty event sending for testability.
type EventSender interface {
	SendEvent(context.Context, pagerduty.V2Event) (*pagerduty.V2EventResponse, error)
}

type defaultEventSenderImpl struct{}

func (d defaultEventSenderImpl) SendEvent(
	ctx context.Context,
	event pagerduty.V2Event,
) (*pagerduty.V2EventResponse, error) {
	return pagerduty.ManageEventWithContext(ctx, event)
}

var defaultEventSender EventSender = defaultEventSenderImpl{}
