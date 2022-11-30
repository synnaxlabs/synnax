package core

import (
	"context"
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

func ValidateChannelKeys(ctx context.Context, reader channel.Reader, keys []channel.Key) error {
	if len(keys) == 0 {
		return errors.Wrap(validate.Error, "[distribution.framer.core] - no channels provided")
	}
	exists, err := reader.NewRetrieve().WhereKeys(keys...).Exists(ctx)
	if err != nil {
		return errors.Wrap(err, "[distribution.framer.core] - failed to validate channel keys")
	}
	if !exists {
		return errors.Wrapf(query.NotFound, "[distribution.framer.core] - channel keys %s not found", keys)
	}
	return nil
}
