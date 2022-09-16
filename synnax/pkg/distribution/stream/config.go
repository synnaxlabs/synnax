package stream

import (
	distribcore "github.com/synnaxlabs/synnax/pkg/distribution/core"
	"go.uber.org/zap"
)

type Config struct {
	Resolver  distribcore.HostResolver
	Transport Transport
	Logger    *zap.Logger
}
