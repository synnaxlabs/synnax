package core

import (
	"github.com/synnaxlabs/x/errors"
)

// ErrChannelNotFound is returned when a channel or a range of data cannot be found in the DB.
var ErrChannelNotFound = errors.New("channel not found")

func NewErrChannelNotFound(ch ChannelKey) error {
	return errors.Wrapf(ErrChannelNotFound, "channel %d not found in the database", ch)
}

func EntityClosed(entityName string) error {
	return errors.Newf("operation on %s is invalid because it is already closed", entityName)
}
