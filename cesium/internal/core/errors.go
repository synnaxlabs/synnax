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

func NewErrorWrapper(key ChannelKey, name string) func(error) error {
	return func(err error) error {
		if err == nil {
			return nil
		}
		return errors.Wrapf(err, "channel [%s]<%d>", name, key)
	}
}
