// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package calculation

import (
	"go/types"

	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/telem"
)

// Status is a calculated channel status entry.
type Status = status.Status[types.Nil]

// StatusKey returns the status key for the given channel key.
func StatusKey(key channel.Key) string {
	return channel.OntologyID(key).String()
}

// StatusFromError builds an error status for a calculated channel.
func StatusFromError(key channel.Key, name string, msg string, err error) *Status {
	s := &Status{Key: StatusKey(key), Name: name}
	s.Variant = status.VariantError
	s.Message = msg
	s.Description = err.Error()
	s.Time = telem.Now()
	return s
}
