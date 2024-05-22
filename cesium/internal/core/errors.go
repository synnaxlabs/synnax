package core

import (
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/cesium"
)

var ErrChannelNotFound = errors.New("channel not found")

// ChannelNotFound is returned when a channel or a range of data cannot be found in the DB.
func ChannelNotFound(ch cesium.ChannelKey) error {
	return errors.Wrapf(ErrChannelNotFound, "channel %d not found", ch)
}

func EntityClosed(entityName string) error {
	return errors.Newf("Operation on %s is invalid because it is already closed", entityName)
}
