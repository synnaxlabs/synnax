package server

import (
	"net"

	"github.com/cockroachdb/cmux"
)

type BranchConfig struct {
	Lis      net.Listener
	Security SecurityConfig
	Debug    bool
}

type Branch interface {
	Match() []cmux.Matcher
	Key() string
	Serve(cfg BranchConfig) error
	Stop()
}
