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
