// Copyright 2024 Synnax Labs, Inc.
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
var ErrChannelNotFound = errors.Wrap(query.NotFound, "channel not found")

func NewErrChannelNotFound(ch ChannelKey) error {
	return errors.Wrapf(ErrChannelNotFound, "channel %d not found in the database", ch)
}

func EntityClosed(entityName string) error {
	return errors.Newf("operation on %s is invalid because it is already closed", entityName)
}

func NewErrorWrapper(ch Channel) func(error) error {
	return func(err error) error {
		if err == nil {
			return nil
		}
		return errors.Wrapf(err, "channel %v", ch)
	}
}
