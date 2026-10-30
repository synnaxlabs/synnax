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
	apiapikey "github.com/synnaxlabs/synnax/pkg/api/auth/apikey"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	grpcauth "github.com/synnaxlabs/synnax/pkg/transport/grpc/auth"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	createServer = grpc.UnaryServer[
		apiapikey.CreateRequest,
		*grpcauth.APIKeyCreateRequest,
		apiapikey.CreateResponse,
		*grpcauth.APIKeyCreateResponse,
	]
	retrieveServer = grpc.UnaryServer[
		apiapikey.RetrieveRequest,
		*grpcauth.APIKeyRetrieveRequest,
		apiapikey.RetrieveResponse,
		*grpcauth.APIKeyRetrieveResponse,
	]
	deleteServer = grpc.UnaryServer[
		apiapikey.DeleteRequest,
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
	_ grpc.BackwardTranslator[apiapikey.CreateRequest, *grpcauth.APIKeyCreateRequest] = (*createRequestTranslator)(
		nil,
	)
	_ grpc.ForwardTranslator[apiapikey.CreateResponse, *grpcauth.APIKeyCreateResponse] = (*createResponseTranslator)(
		nil,
	)
	_ grpc.BackwardTranslator[apiapikey.RetrieveRequest, *grpcauth.APIKeyRetrieveRequest] = (*retrieveRequestTranslator)(
		nil,
	)
	_ grpc.ForwardTranslator[apiapikey.RetrieveResponse, *grpcauth.APIKeyRetrieveResponse] = (*retrieveResponseTranslator)(
		nil,
	)
	_ grpc.BackwardTranslator[apiapikey.DeleteRequest, *grpcauth.APIKeyDeleteRequest] = (*deleteRequestTranslator)(
		nil,
	)
)

func apiKeyForward(k apiapikey.APIKey) *grpcauth.APIKey {
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
) (apiapikey.CreateRequest, error) {
	return apiapikey.CreateRequest{
		Subject: grpcauth.SubjectBackward(r.Subject),
		Name:    r.Name,
	}, nil
}

func (createResponseTranslator) Forward(
	_ context.Context,
	r apiapikey.CreateResponse,
) (*grpcauth.APIKeyCreateResponse, error) {
	return &grpcauth.APIKeyCreateResponse{
		Token: r.Token,
		Key:   apiKeyForward(r.Key),
	}, nil
}

func (retrieveRequestTranslator) Backward(
	_ context.Context,
	r *grpcauth.APIKeyRetrieveRequest,
) (apiapikey.RetrieveRequest, error) {
	return apiapikey.RetrieveRequest{
		Subjects: lo.Map(r.Subjects, func(s *grpcauth.Subject, _ int) ontology.ID {
			return grpcauth.SubjectBackward(s)
		}),
	}, nil
}

func (retrieveResponseTranslator) Forward(
	_ context.Context,
	r apiapikey.RetrieveResponse,
) (*grpcauth.APIKeyRetrieveResponse, error) {
	return &grpcauth.APIKeyRetrieveResponse{
		Keys: lo.Map(r.Keys, func(k apiapikey.APIKey, _ int) *grpcauth.APIKey {
			return apiKeyForward(k)
		}),
	}, nil
}

func (deleteRequestTranslator) Backward(
	_ context.Context,
	r *grpcauth.APIKeyDeleteRequest,
) (apiapikey.DeleteRequest, error) {
	keys := make([]uuid.UUID, 0, len(r.Keys))
	for _, k := range r.Keys {
		key, err := uuid.Parse(k)
		if err != nil {
			return apiapikey.DeleteRequest{}, err
		}
		keys = append(keys, key)
	}
	return apiapikey.DeleteRequest{Keys: keys}, nil
}

// New binds the API key servers onto t.
func New(t *api.Transport) grpc.BindableTransport {
	create := &createServer{
		RequestTranslator:  createRequestTranslator{},
		ResponseTranslator: createResponseTranslator{},
		ServiceDesc:        &grpcauth.AuthAPIKeyCreateService_ServiceDesc,
	}
	retrieve := &retrieveServer{
		RequestTranslator:  retrieveRequestTranslator{},
		ResponseTranslator: retrieveResponseTranslator{},
		ServiceDesc:        &grpcauth.AuthAPIKeyRetrieveService_ServiceDesc,
	}
	del := &deleteServer{
		RequestTranslator:  deleteRequestTranslator{},
		ResponseTranslator: grpc.EmptyTranslator{},
		ServiceDesc:        &grpcauth.AuthAPIKeyDeleteService_ServiceDesc,
	}
	t.AuthAPIKeyCreate = create
	t.AuthAPIKeyRetrieve = retrieve
	t.AuthAPIKeyDelete = del
	return grpc.CompoundBindableTransport{create, retrieve, del}
}
