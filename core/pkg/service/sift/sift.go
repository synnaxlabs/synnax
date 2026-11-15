// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package sift connects the Core to Sift. A Sift device holds the address of a Sift
// API and the secret that holds its API key.
package sift

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"time"
	"uuid"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	grpcstatus "google.golang.org/grpc/status"
)

// Make is the make of a Sift device.
const Make = "sift"

// keepaliveTime is how long a connection idles before it pings Sift, which closes
// streams that idle for 60 s.
const keepaliveTime = 20 * time.Second

// decodeProperties returns the connection properties of dev. It returns a validation
// error when dev is not a Sift device or its properties are incomplete.
func decodeProperties(dev device.Device) (Properties, error) {
	if dev.Make != Make {
		return Properties{}, validate.Errorf("device %s is not a Sift device", dev.Name)
	}
	var props Properties
	if err := dev.Properties.Unmarshal(&props); err != nil {
		return Properties{}, errors.Wrapf(
			err,
			"parsing properties of device %s",
			dev.Name,
		)
	}
	if err := props.validate(); err != nil {
		return Properties{}, err
	}
	return props, nil
}

func (p Properties) validate() error {
	v := validate.New("sift.properties")
	v.NotEmptyString("uri", p.URI)
	v.Ternary("api_key", p.APIKey == uuid.Nil(), "required")
	return v.Error()
}

// connect opens a connection to Sift with the properties of a saved device.
func connect(
	ctx context.Context,
	secrets *secret.Service,
	snap gorp.Snapshot,
	props Properties,
) (*grpc.ClientConn, error) {
	values, err := secrets.Reveal(ctx, snap, props.APIKey)
	if err != nil {
		return nil, err
	}
	return dial(props.URI, string(values[props.APIKey]))
}

// dial opens a connection to the Sift API at uri that sends apiKey with every call.
func dial(uri, apiKey string) (*grpc.ClientConn, error) {
	uri = target(uri)
	creds := credentials.NewTLS(&tls.Config{MinVersion: tls.VersionTLS12})
	if isLocal(uri) {
		creds = insecure.NewCredentials()
	}
	auth := func(ctx context.Context) context.Context {
		return metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+apiKey)
	}
	return grpc.NewClient(
		uri,
		grpc.WithTransportCredentials(creds),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{Time: keepaliveTime}),
		grpc.WithChainUnaryInterceptor(func(
			ctx context.Context,
			method string,
			req, reply any,
			cc *grpc.ClientConn,
			invoker grpc.UnaryInvoker,
			opts ...grpc.CallOption,
		) error {
			return invoker(auth(ctx), method, req, reply, cc, opts...)
		}),
		grpc.WithChainStreamInterceptor(func(
			ctx context.Context,
			desc *grpc.StreamDesc,
			cc *grpc.ClientConn,
			method string,
			streamer grpc.Streamer,
			opts ...grpc.CallOption,
		) (grpc.ClientStream, error) {
			return streamer(auth(ctx), desc, cc, method, opts...)
		}),
	)
}

// target returns the gRPC target of a Sift URI. It drops an http or https scheme and a
// trailing slash, so that a URI copied from a browser connects.
func target(uri string) string {
	if scheme, rest, ok := strings.Cut(uri, "://"); ok &&
		(strings.EqualFold(scheme, "https") || strings.EqualFold(scheme, "http")) {
		uri = rest
	}
	return strings.TrimSuffix(uri, "/")
}

// failure returns a driver.Failure that says why a call to the Sift API at uri failed,
// or err itself when the call did not fail with a gRPC status.
func failure(uri string, err error) error {
	st, ok := grpcstatus.FromError(err)
	if err == nil || !ok {
		return err
	}
	switch st.Code() {
	case codes.Unauthenticated:
		return driver.Failure{
			Message: "Authentication failed",
			Description: "Sift rejected the API key. Check that the key is current and " +
				"that it belongs to the Sift environment of the URI.",
		}
	case codes.PermissionDenied:
		return driver.Failure{
			Message:     "Permission denied",
			Description: "The API key has no permission for this: " + st.Message() + ".",
		}
	case codes.Unavailable, codes.DeadlineExceeded:
		reason := st.Message()
		switch {
		case st.Code() == codes.DeadlineExceeded:
			reason = "Sift did not answer in time"
		case strings.Contains(reason, "produced zero addresses"):
			reason = "the host was not found"
		// Windows says that the target machine actively refused the connection.
		case strings.Contains(reason, "connection refused"),
			strings.Contains(reason, "actively refused"):
			reason = "the connection was refused"
		}
		return driver.Failure{
			Message: "Failed to reach Sift",
			Description: fmt.Sprintf(
				"Could not reach %s: %s. Check the URI, and that the network of the "+
					"Core can reach Sift.",
				target(uri),
				reason,
			),
		}
	default:
		return driver.Failure{
			Message:     "Sift rejected the request",
			Description: st.Message() + ".",
		}
	}
}

// isLocal reports whether uri names this computer, which Sift's own clients reach
// without TLS.
func isLocal(uri string) bool {
	host, _, err := net.SplitHostPort(uri)
	if err != nil {
		host = uri
	}
	return host == "localhost" || net.ParseIP(host).IsLoopback()
}

// APIKeyReference returns the reference from each Sift device to the secret that holds
// its API key. Registered on the device table, it lists the device as a use of the
// secret and stops the secret from being deleted while the device exists.
func APIKeyReference() gorp.Reference[device.Key, device.Device] {
	return gorp.NewKeyReference(
		gorp.KeyReferenceConfig[device.Key, device.Device, secret.Key]{
			Name:     "properties.api_key",
			Table:    "secret",
			OnDelete: gorp.Restrict,
			Kind:     gorp.Use,
			Targets: func(dev *device.Device) []secret.Key {
				if dev.Make != Make {
					return nil
				}
				var props Properties
				if err := dev.Properties.Unmarshal(&props); err != nil ||
					props.APIKey == uuid.Nil() {
					return nil
				}
				return []secret.Key{props.APIKey}
			},
		},
	)
}
