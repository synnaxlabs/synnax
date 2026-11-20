// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package secret

import (
	"context"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/api/secret"
	"github.com/synnaxlabs/synnax/pkg/api/secret/pb"
	"github.com/synnaxlabs/synnax/pkg/service/certificate"
	taskpb "github.com/synnaxlabs/synnax/pkg/service/task/pb"
	"github.com/synnaxlabs/x/errors"
)

type (
	retrieveServer = grpc.UnaryServer[
		secret.RetrieveRequest,
		*pb.RetrieveRequest,
		secret.RetrieveResponse,
		*pb.RetrieveResponse,
	]
	resolveServer = grpc.UnaryServer[
		secret.ResolveRequest,
		*ResolveRequest,
		secret.ResolveResponse,
		*ResolveResponse,
	]
)

type (
	retrieveRequestTranslator  struct{}
	retrieveResponseTranslator struct{}
	resolveRequestTranslator   struct{}
	resolveResponseTranslator  struct{}
)

var (
	_ grpc.BackwardTranslator[secret.RetrieveRequest, *pb.RetrieveRequest]  = retrieveRequestTranslator{}
	_ grpc.ForwardTranslator[secret.RetrieveResponse, *pb.RetrieveResponse] = retrieveResponseTranslator{}
	_ grpc.BackwardTranslator[secret.ResolveRequest, *ResolveRequest]       = resolveRequestTranslator{}
	_ grpc.ForwardTranslator[secret.ResolveResponse, *ResolveResponse]      = resolveResponseTranslator{}
)

func (retrieveRequestTranslator) Backward(
	_ context.Context,
	req *pb.RetrieveRequest,
) (secret.RetrieveRequest, error) {
	return pb.RetrieveRequestFromPB(req)
}

func (retrieveResponseTranslator) Forward(
	_ context.Context,
	res secret.RetrieveResponse,
) (*pb.RetrieveResponse, error) {
	return pb.RetrieveResponseToPB(res)
}

// parseKey parses a UUID, where an empty string is the zero UUID.
func parseKey(s string) (uuid.UUID, error) {
	if s == "" {
		return uuid.Nil(), nil
	}
	return uuid.Parse(s)
}

func (resolveRequestTranslator) Backward(
	_ context.Context,
	req *ResolveRequest,
) (secret.ResolveRequest, error) {
	var (
		r   secret.ResolveRequest
		err error
	)
	if r.Grant, err = parseKey(req.Grant); err != nil {
		return secret.ResolveRequest{}, err
	}
	if r.Task, err = parseKey(req.Task); err != nil {
		return secret.ResolveRequest{}, err
	}
	if r.Secrets, err = lo.MapErr(
		req.Secrets,
		func(k string, _ int) (secret.Key, error) { return uuid.Parse(k) },
	); err != nil {
		return secret.ResolveRequest{}, err
	}
	return r, nil
}

func (resolveResponseTranslator) Forward(
	_ context.Context,
	res secret.ResolveResponse,
) (*ResolveResponse, error) {
	cmd, err := taskpb.CommandToPB(res.Command)
	if err != nil {
		return nil, err
	}
	return &ResolveResponse{
		Command: cmd,
		Secrets: lo.Map(res.Secrets, func(k secret.Key, _ int) string {
			return k.String()
		}),
		Certificates: lo.Map(res.Certificates, func(k certificate.Key, _ int) string {
			return k.String()
		}),
		Values: lo.Map(res.Values, func(v secret.Value, _ int) *Value {
			return &Value{Key: v.Key.String(), Value: v.Value}
		}),
	}, nil
}

// New binds the secret routes that the C++ client calls to t and returns their gRPC
// transport.
func New(t *api.Transport, errs *errors.Codec) grpc.BindableTransport {
	retrieve := &retrieveServer{
		RequestTranslator:  retrieveRequestTranslator{},
		ResponseTranslator: retrieveResponseTranslator{},
		ServiceDesc:        &SecretRetrieveService_ServiceDesc,
		Errors:             errs,
	}
	t.SecretRetrieve = retrieve
	resolve := &resolveServer{
		RequestTranslator:  resolveRequestTranslator{},
		ResponseTranslator: resolveResponseTranslator{},
		ServiceDesc:        &SecretResolveService_ServiceDesc,
		Errors:             errs,
	}
	t.SecretResolve = resolve
	return grpc.CompoundBindableTransport{retrieve, resolve}
}
