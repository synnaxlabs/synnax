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
	"net"
	"time"
	"uuid"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	"github.com/synnaxlabs/synnax/pkg/service/secret/grant"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
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
		return Properties{}, errors.Wrapf(
			validate.ErrValidation,
			"device %s is not a Sift device",
			dev.Name,
		)
	}
	var props Properties
	if err := dev.Properties.Unmarshal(&props); err != nil {
		return Properties{}, errors.Wrapf(
			err,
			"parsing properties of device %s",
			dev.Name,
		)
	}
	return props, props.validate()
}

func (p Properties) validate() error {
	v := validate.New("sift.properties")
	v.NotEmptyString("uri", p.URI)
	v.Ternary("api_key", p.APIKey == uuid.Nil(), "required")
	return v.Error()
}

// connect reads the API key of props as rack and opens a connection to Sift. With a
// zero grant key, rack must hold a device that uses the API key.
func connect(
	ctx context.Context,
	secrets *secret.Service,
	props Properties,
	rack ontology.ID,
	grantKey grant.Key,
) (*grpc.ClientConn, error) {
	values, err := secrets.Resolve(ctx, nil, rack, grantKey, props.APIKey)
	if err != nil {
		return nil, err
	}
	return dial(props.URI, string(values[props.APIKey]))
}

// dial opens a connection to the Sift API at uri that sends apiKey with every call.
func dial(uri, apiKey string) (*grpc.ClientConn, error) {
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

// isLocal reports whether uri names this computer, which Sift's own clients reach
// without TLS.
func isLocal(uri string) bool {
	host, _, err := net.SplitHostPort(uri)
	if err != nil {
		host = uri
	}
	return host == "localhost" || net.ParseIP(host).IsLoopback()
}

// SecretUses returns a secret.Finder that reports each Sift device that uses one of
// the secrets. A device uses its API key on its own rack.
func SecretUses(devices *device.Service) secret.Finder {
	return func(ctx context.Context, tx gorp.Tx, keys []secret.Key) ([]secret.Use, error) {
		var devs []device.Device
		if err := devices.NewRetrieve().
			Where(device.MatchMakes(Make)).
			Entries(&devs).
			Exec(ctx, tx); err != nil {
			return nil, err
		}
		wanted := set.New(keys...)
		var uses []secret.Use
		for _, dev := range devs {
			var props Properties
			if err := dev.Properties.Unmarshal(&props); err != nil {
				return nil, errors.Wrapf(
					err,
					"parsing properties of device %s",
					dev.Name,
				)
			}
			if !wanted.Contains(props.APIKey) {
				continue
			}
			uses = append(uses, secret.Use{
				Resource: device.OntologyID(dev.Key),
				Name:     dev.Name,
				Racks:    []ontology.ID{dev.Rack.OntologyID()},
				Secret:   props.APIKey,
			})
		}
		return uses, nil
	}
}
