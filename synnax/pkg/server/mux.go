package server

import (
	"github.com/cockroachdb/cmux"
	"net"
)

type mux struct {
	base cmux.CMux
	http net.Listener
	grpc net.Listener
}

func (m mux) serve() error { return m.base.Serve() }

func newMux(lis net.Listener) mux {
	m := mux{base: cmux.New(lis)}
	m.http = m.base.Match(cmux.HTTP1Fast())
	m.grpc = m.base.Match(cmux.Any())
	return m
}
