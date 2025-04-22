// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package core

import (
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
)

// ErrChannelNotFound is returned when a channel or a range of data cannot be found in the DB.
var (
	ErrChannelNotFound = errors.Wrap(query.NotFound, "channel not found")
	ErrOpenEntity      = errors.New("cannot close database because there are open entities on it")
	ErrClosedEntity    = errors.New("entity closed")
)

func NewErrChannelNotFound(ch ChannelKey) error {
	return errors.Wrapf(ErrChannelNotFound, "channel %d not found", ch)
}

func NewErrEntityClosed(entityName string) error {
	return errors.Wrapf(ErrClosedEntity, "cannot complete operation on closed %s", entityName)
}

func NewChannelErrWrapper(ch Channel) func(error) error {
	return func(err error) error {
		return errors.Wrapf(err, "channel %v", ch)
	}
}
