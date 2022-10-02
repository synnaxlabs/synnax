package server

import (
	"crypto/tls"
	"github.com/cockroachdb/cmux"
)

type BranchConfig struct {
	Mux cmux.CMux
	TLS *tls.Config
}

type Branch interface {
	Key() string
	Serve(cfg BranchConfig) error
	Stop()
}
