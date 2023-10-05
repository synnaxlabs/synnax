// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ranger

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/x/gorp"
)

type alias struct {
	Range   uuid.UUID   `json:"range" msgpack:"range"`
	Channel channel.Key `json:"channel" msgpack:"channel"`
	Alias   string      `json:"alias" msgpack:"alias"`
}

var _ gorp.Entry[string] = alias{}

// GorpKey implements gorp.Entry.
func (a alias) GorpKey() string { return fmt.Sprintf("%s:%s", a.Range, a.Channel) }

// SetOptions implements gorp.Entry.
func (a alias) SetOptions() []interface{} {
	// TODO: Figure out if we should return leaseholder here.
	return nil
}
