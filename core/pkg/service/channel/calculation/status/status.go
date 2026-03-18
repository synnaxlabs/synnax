// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package status provides shared types and helpers for calculated channel status
// entries, used by both the channel calculation graph and the framer calculation
// service.
package status

import (
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	svcstatus "github.com/synnaxlabs/synnax/pkg/service/status"
	xstatus "github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/telem"
)

// Details is the payload attached to status entries for calculated channels.
type Details struct {
	Channel channel.Key `json:"channel" msgpack:"channel"`
}

// Status is a calculated channel status entry.
type Status = svcstatus.Status[Details]

// Key returns the status key for the given channel key.
func Key(key channel.Key) string {
	return channel.OntologyID(key).String()
}

// Error builds an error status for a calculated channel.
func Error(key channel.Key, name string, msg string, err error) *Status {
	return &Status{
		Key:         Key(key),
		Name:        name,
		Variant:     xstatus.VariantError,
		Message:     msg,
		Description: err.Error(),
		Time:        telem.Now(),
		Details:     Details{Channel: key},
	}
}
