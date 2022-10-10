package server

import (
	"crypto/tls"
	"net"

	"github.com/cockroachdb/cmux"
)

type BranchConfig struct {
	Lis net.Listener
	TLS *tls.Config
}

type Branch interface {
	Match() []cmux.Matcher
	Key() string
	Serve(cfg BranchConfig) error
	Stop()
}
