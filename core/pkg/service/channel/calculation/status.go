// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package calculation implements analysis of calculated channel expressions and the
// statuses published for them. It depends on the channel storage shapes in
// channel/versions, never on the channel service itself, so the service can compose
// it without an import cycle.
package calculation

import (
	"go/types"

	"github.com/synnaxlabs/synnax/pkg/service/channel/versions"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/telem"
)

// Status is a calculated channel status entry.
type Status = status.Status[types.Nil]

// StatusKey returns the status key for the given channel key.
func StatusKey(key versions.Key) string {
	return ontology.ID{Type: ontology.ResourceTypeChannel, Key: key.String()}.String()
}

// StatusFromError builds an error status for a calculated channel.
func StatusFromError(key versions.Key, name string, msg string, err error) *Status {
	return &Status{
		Key:         StatusKey(key),
		Name:        name,
		Variant:     status.VariantError,
		Message:     msg,
		Description: err.Error(),
		Time:        telem.Now(),
	}
}
