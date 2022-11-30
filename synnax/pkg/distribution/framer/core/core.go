package core

import (
	"context"
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/x/query"
)

func ValidateChannelKeys(ctx context.Context, chReader channel.Reader, keys []channel.Key) error {
	if len(keys) == 0 {
		return errors.New("[segment] - no channels provided")
	}
	exists, err := chReader.NewRetrieve().WhereKeys(keys...).Exists(ctx)
	if !exists {
		return errors.Wrapf(query.NotFound, "[segment] - channel keys %s not found", keys)
	}
	if err != nil {
		return errors.Wrap(err, "[segment] - failed to validate channel keys")
	}
	return nil
}
