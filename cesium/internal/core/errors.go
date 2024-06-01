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
	"fmt"
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/x/query"
)

var (
	// ChannelNotFound is returned when a channel or a range of data cannot be found in the DB.
	ChannelNotFound = errors.Wrap(query.NotFound, "[cesium] - channel not found")
)

func EntityClosed(entityName string) error {
	return fmt.Errorf("[cesium] - %s is already closed", entityName)
}
