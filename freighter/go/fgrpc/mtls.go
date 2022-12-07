package fgrpc

import (
	"context"
	"crypto/tls"
	"github.com/cockroachdb/cmux"
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/freighter"
	"go.uber.org/zap"
	"google.golang.org/grpc/credentials"
	"net"
)

func NewMuxCredentials(logger *zap.Logger) credentials.TransportCredentials {
	return MuxTransportCredentials{logger: logger}
}

type MuxTransportCredentials struct {
	logger *zap.Logger
}

func (MuxTransportCredentials) ClientHandshake(
	context.Context, string, net.Conn,
) (net.Conn, credentials.AuthInfo, error) {
	panic("[synnax] MuxTransportCredentials should not be used for client connections")
}

const (
	muxCredentialsNonMuxMsg = "MuxTransportCredentials.ServerHandshake called with non-mux connection"
	muxCredentialsNonTLSMsg = "MuxTransportCredentials.ServerHandshake called with non-TLS connection"
)

func (c MuxTransportCredentials) ServerHandshake(conn net.Conn) (net.Conn, credentials.AuthInfo, error) {
	muxConn, ok := conn.(*cmux.MuxConn)
	if !ok {
		c.logger.DPanic(muxCredentialsNonMuxMsg, zap.String("type", conn.RemoteAddr().Network()))
		return nil, nil, errors.New(muxCredentialsNonMuxMsg)
	}
	tlsConn, ok := muxConn.Conn.(*tls.Conn)
	if !ok {
		c.logger.DPanic(muxCredentialsNonTLSMsg, zap.String("type", conn.RemoteAddr().Network()))
		return nil, nil, errors.New(muxCredentialsNonTLSMsg)
	}
	return conn, credentials.TLSInfo{
		State:          tlsConn.ConnectionState(),
		CommonAuthInfo: credentials.CommonAuthInfo{SecurityLevel: credentials.PrivacyAndIntegrity},
	}, nil
}

func (MuxTransportCredentials) Info() credentials.ProtocolInfo {
	return credentials.ProtocolInfo{
		SecurityProtocol: "tls",
		ServerName:       "mux",
	}
}

func (MuxTransportCredentials) Clone() credentials.TransportCredentials {
	return MuxTransportCredentials{}
}

func (MuxTransportCredentials) OverrideServerName(string) error {
	return nil
}

var MTLSAuthError = errors.Wrapf(
	freighter.SecurityError,
	"Unable to verify TLS credentials",
)

func MTLSMiddleware(expectedCN string) freighter.Middleware {
	return freighter.MiddlewareFunc(func(
		ctx context.Context,
		md freighter.MD,
		next freighter.Next,
	) (freighter.MD, error) {
		if !md.Sec.TLS.Used ||
			(len(md.Sec.TLS.VerifiedChains) == 0 || len(md.Sec.TLS.VerifiedChains[0]) == 0) ||
			md.Sec.TLS.VerifiedChains[0][0].Subject.CommonName != expectedCN {
			return md, MTLSAuthError
		}
		return next(ctx, md)
	})
}
