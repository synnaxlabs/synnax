package core

import (
	"context"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/arya-analytics/x/query"
	"github.com/cockroachdb/errors"
)

type Segment struct {
	ChannelKey      channel.Key `json:"channelKey"`
	storage.Segment `json:"segment"`
}

func ValidateChannelKeys(ctx context.Context, svc *channel.Service, keys []channel.Key) error {
	if len(keys) == 0 {
		return errors.New("[segment] - no channels provided")
	}
	exists, err := svc.NewRetrieve().WhereKeys(keys...).Exists(ctx)
	if !exists {
		return errors.Wrapf(query.NotFound, "[segment] - channel keys %s not found", keys)
	}
	if err != nil {
		return errors.Wrap(err, "[segment] - failed to validate channel keys")
	}
	return nil
}
