// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package apikey

import (
	"context"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/api/auth/apikey"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	grpcauth "github.com/synnaxlabs/synnax/pkg/transport/grpc/auth"
	"github.com/synnaxlabs/x/errors"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	createServer = grpc.UnaryServer[
		apikey.CreateRequest,
		*grpcauth.APIKeyCreateRequest,
		apikey.CreateResponse,
		*grpcauth.APIKeyCreateResponse,
	]
	retrieveServer = grpc.UnaryServer[
		apikey.RetrieveRequest,
		*grpcauth.APIKeyRetrieveRequest,
		apikey.RetrieveResponse,
		*grpcauth.APIKeyRetrieveResponse,
	]
	deleteServer = grpc.UnaryServer[
		apikey.DeleteRequest,
		*grpcauth.APIKeyDeleteRequest,
		struct{},
		*emptypb.Empty,
	]
)

type (
	createRequestTranslator    struct{}
	createResponseTranslator   struct{}
	retrieveRequestTranslator  struct{}
	retrieveResponseTranslator struct{}
	deleteRequestTranslator    struct{}
)

var (
	_ grpc.BackwardTranslator[apikey.CreateRequest, *grpcauth.APIKeyCreateRequest] = (*createRequestTranslator)(
		nil,
	)
	_ grpc.ForwardTranslator[apikey.CreateResponse, *grpcauth.APIKeyCreateResponse] = (*createResponseTranslator)(
		nil,
	)
	_ grpc.BackwardTranslator[apikey.RetrieveRequest, *grpcauth.APIKeyRetrieveRequest] = (*retrieveRequestTranslator)(
		nil,
	)
	_ grpc.ForwardTranslator[apikey.RetrieveResponse, *grpcauth.APIKeyRetrieveResponse] = (*retrieveResponseTranslator)(
		nil,
	)
	_ grpc.BackwardTranslator[apikey.DeleteRequest, *grpcauth.APIKeyDeleteRequest] = (*deleteRequestTranslator)(
		nil,
	)
)

func apiKeyForward(k apikey.APIKey) *grpcauth.APIKey {
	return &grpcauth.APIKey{
		Key:     k.Key.String(),
		Subject: grpcauth.SubjectForward(k.Subject),
		Name:    k.Name,
		Created: int64(k.Created),
	}
}

func (createRequestTranslator) Backward(
	_ context.Context,
	r *grpcauth.APIKeyCreateRequest,
) (apikey.CreateRequest, error) {
	return apikey.CreateRequest{
		Subject: grpcauth.SubjectBackward(r.Subject),
		Name:    r.Name,
	}, nil
}

func (createResponseTranslator) Forward(
	_ context.Context,
	r apikey.CreateResponse,
) (*grpcauth.APIKeyCreateResponse, error) {
	return &grpcauth.APIKeyCreateResponse{
		Token: r.Token,
		Key:   apiKeyForward(r.Key),
	}, nil
}

func (retrieveRequestTranslator) Backward(
	_ context.Context,
	r *grpcauth.APIKeyRetrieveRequest,
) (apikey.RetrieveRequest, error) {
	return apikey.RetrieveRequest{
		Subjects: lo.Map(r.Subjects, func(s *grpcauth.Subject, _ int) resource.ID {
			return grpcauth.SubjectBackward(s)
		}),
	}, nil
}

func (retrieveResponseTranslator) Forward(
	_ context.Context,
	r apikey.RetrieveResponse,
) (*grpcauth.APIKeyRetrieveResponse, error) {
	return &grpcauth.APIKeyRetrieveResponse{
		Keys: lo.Map(r.Keys, func(k apikey.APIKey, _ int) *grpcauth.APIKey {
			return apiKeyForward(k)
		}),
	}, nil
}

func (deleteRequestTranslator) Backward(
	_ context.Context,
	r *grpcauth.APIKeyDeleteRequest,
) (apikey.DeleteRequest, error) {
	keys := make([]uuid.UUID, 0, len(r.Keys))
	for _, k := range r.Keys {
		key, err := uuid.Parse(k)
		if err != nil {
			return apikey.DeleteRequest{}, err
		}
		keys = append(keys, key)
	}
	return apikey.DeleteRequest{Keys: keys}, nil
}

// New binds the API key servers onto t. The servers encode errors with errs.
func New(t *api.Transport, errs *errors.Codec) grpc.BindableTransport {
	create := &createServer{
		RequestTranslator:  createRequestTranslator{},
		ResponseTranslator: createResponseTranslator{},
		ServiceDesc:        &grpcauth.AuthAPIKeyCreateService_ServiceDesc,
		Errors:             errs,
	}
	retrieve := &retrieveServer{
		RequestTranslator:  retrieveRequestTranslator{},
		ResponseTranslator: retrieveResponseTranslator{},
		ServiceDesc:        &grpcauth.AuthAPIKeyRetrieveService_ServiceDesc,
		Errors:             errs,
	}
	del := &deleteServer{
		RequestTranslator:  deleteRequestTranslator{},
		ResponseTranslator: grpc.EmptyTranslator{},
		ServiceDesc:        &grpcauth.AuthAPIKeyDeleteService_ServiceDesc,
		Errors:             errs,
	}
	t.AuthAPIKeyCreate = create
	t.AuthAPIKeyRetrieve = retrieve
	t.AuthAPIKeyDelete = del
	return grpc.CompoundBindableTransport{create, retrieve, del}
}
