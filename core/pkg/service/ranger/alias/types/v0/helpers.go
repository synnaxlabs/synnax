// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	channelv0 "github.com/synnaxlabs/synnax/pkg/service/channel/types/v0"
	rangerv1 "github.com/synnaxlabs/synnax/pkg/service/ranger/types/v1"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
)

const keySeparator = "---"

// GorpKey composes the storage key for the alias of ch on range r.
func GorpKey(r rangerv1.Key, ch channelv0.Key) string {
	return fmt.Sprintf("%s%s%s", r, keySeparator, ch)
}

// ParseGorpKey splits an alias storage key into its range and channel keys.
func ParseGorpKey(key string) (rangerv1.Key, channelv0.Key, error) {
	split := strings.Split(key, keySeparator)
	if len(split) != 2 {
		return uuid.Nil, 0, errors.Newf("[alias] - invalid key")
	}
	r, err := uuid.Parse(split[0])
	if err != nil {
		return uuid.Nil, 0, errors.Wrapf(err, "[alias] - invalid range")
	}
	c, err := channel.ParseKey(split[1])
	if err != nil {
		return uuid.Nil, 0, errors.Wrapf(err, "[alias] - invalid channel")
	}
	return r, c, nil
}

var _ gorp.Entry[string] = Alias{}

// GorpKey implements gorp.Entry.
func (a Alias) GorpKey() string { return GorpKey(a.Range, a.Channel) }

// SetOptions implements gorp.Entry.
func (a Alias) SetOptions() []any { return nil }
