package server

import (
	"go.uber.org/zap"
	"net"

	"github.com/cockroachdb/cmux"
)

type BranchConfig struct {
	Lis      net.Listener
	Security SecurityConfig
	Debug    bool
	Logger   *zap.Logger
}

type Branch interface {
	Matchers() []cmux.Matcher
	Key() string
	Serve(cfg BranchConfig) error
	Stop()
}
