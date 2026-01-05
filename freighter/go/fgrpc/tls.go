// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package fgrpc

import (
	"context"
	"crypto/tls"
	"github.com/cockroachdb/cmux"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/errors"
	"go.uber.org/zap"
	"google.golang.org/grpc/credentials"
	"net"
)

// MuxCredentials implements the grpc.TransportCredentials interface that allows
// for TLS handshakes and verification to occur on a multiplexer that lies in front of
// the gRPC server. MuxCredentials extracts the underlying TLS connection from
// the cmux.MuxConn and passes it the gRPC server through its ServerHandshake method.
//
// It's important to note that MuxTransport credentials does not perform any TLS handshaking
// or certificate verification. It simply allows for a gRPC server to be run securely behind
// a cmux multiplexer.
//
// It's also important to note that MuxCredentials should only be used for server
// connections. Exec connections should use the standard TLS credentials.
type MuxCredentials struct {
	alamos.Instrumentation
	ServerName string
}

var _ credentials.TransportCredentials = (*MuxCredentials)(nil)

// ClientHandshake will panic if called. It is here purely for the purpose of implementing
// the grpc.TransportCredentials interface.
func (*MuxCredentials) ClientHandshake(
	context.Context, string, net.Conn,
) (net.Conn, credentials.AuthInfo, error) {
	panic("[synnax] MuxCredentials should not be used for client connections")
}

const (
	muxCredentialsNonMuxMsg = "MuxCredentials.ServerHandshake called with non-mux connection"
	muxCredentialsNonTLSMsg = "MuxCredentials.ServerHandshake called with non-TLS connection"
)

// ServerHandshake implements the grpc.TransportCredentials interface by extracting the
// underlying TLS connection from the cmux.MuxConn and passing it the gRPC server through
// its ServerHandshake method. If the given connection is not a cmux.MuxConn, or if the
// underlying connection is not a TLS connection, MuxCredentials will panic
// in development mode and return an error in production mode.
func (c *MuxCredentials) ServerHandshake(conn net.Conn) (net.Conn, credentials.AuthInfo, error) {
	muxConn, ok := conn.(*cmux.MuxConn)
	if !ok {
		c.L.DPanic(muxCredentialsNonMuxMsg, zap.String("type", conn.RemoteAddr().Network()))
		return nil, nil, errors.New(muxCredentialsNonMuxMsg)
	}
	tlsConn, ok := muxConn.Conn.(*tls.Conn)
	if !ok {
		c.L.DPanic(muxCredentialsNonTLSMsg, zap.String("type", conn.RemoteAddr().Network()))
		return nil, nil, errors.New(muxCredentialsNonTLSMsg)
	}
	return conn, credentials.TLSInfo{
		State:          tlsConn.ConnectionState(),
		CommonAuthInfo: credentials.CommonAuthInfo{SecurityLevel: credentials.PrivacyAndIntegrity},
	}, nil
}

// Info implements grpc.TransportCredentials.
func (c *MuxCredentials) Info() credentials.ProtocolInfo {
	return credentials.ProtocolInfo{SecurityProtocol: "tls", ServerName: c.ServerName}
}

// Clone implements grpc.TransportCredentials.
func (c *MuxCredentials) Clone() credentials.TransportCredentials {
	return &MuxCredentials{
		Instrumentation: c.Instrumentation,
		ServerName:      c.ServerName,
	}
}

// OverrideServerName implements grpc.TransportCredentials.
func (c *MuxCredentials) OverrideServerName(override string) error {
	c.ServerName = override
	return nil
}
