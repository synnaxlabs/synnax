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
	"github.com/synnaxlabs/x/telem"
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
	_ grpc.Translator[apiapikey.CreateRequest, *grpcauth.APIKeyCreateRequest] = (*createRequestTranslator)(
		nil,
	)
	_ grpc.Translator[apiapikey.CreateResponse, *grpcauth.APIKeyCreateResponse] = (*createResponseTranslator)(
		nil,
	)
	_ grpc.Translator[apiapikey.RetrieveRequest, *grpcauth.APIKeyRetrieveRequest] = (*retrieveRequestTranslator)(
		nil,
	)
	_ grpc.Translator[apiapikey.RetrieveResponse, *grpcauth.APIKeyRetrieveResponse] = (*retrieveResponseTranslator)(
		nil,
	)
	_ grpc.Translator[apiapikey.DeleteRequest, *grpcauth.APIKeyDeleteRequest] = (*deleteRequestTranslator)(
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

func apiKeyBackward(k *grpcauth.APIKey) (apiapikey.APIKey, error) {
	if k == nil {
		return apiapikey.APIKey{}, nil
	}
	key, err := uuid.Parse(k.Key)
	if err != nil {
		return apiapikey.APIKey{}, err
	}
	return apiapikey.APIKey{
		Key:     key,
		Subject: grpcauth.SubjectBackward(k.Subject),
		Name:    k.Name,
		Created: telem.TimeStamp(k.Created),
	}, nil
}

func (createRequestTranslator) Forward(
	_ context.Context,
	r apiapikey.CreateRequest,
) (*grpcauth.APIKeyCreateRequest, error) {
	return &grpcauth.APIKeyCreateRequest{
		Subject: grpcauth.SubjectForward(r.Subject),
		Name:    r.Name,
	}, nil
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

func (createResponseTranslator) Backward(
	_ context.Context,
	r *grpcauth.APIKeyCreateResponse,
) (apiapikey.CreateResponse, error) {
	k, err := apiKeyBackward(r.Key)
	if err != nil {
		return apiapikey.CreateResponse{}, err
	}
	return apiapikey.CreateResponse{Token: r.Token, Key: k}, nil
}

func (retrieveRequestTranslator) Forward(
	_ context.Context,
	r apiapikey.RetrieveRequest,
) (*grpcauth.APIKeyRetrieveRequest, error) {
	return &grpcauth.APIKeyRetrieveRequest{
		Subjects: lo.Map(r.Subjects, func(id ontology.ID, _ int) *grpcauth.Subject {
			return grpcauth.SubjectForward(id)
		}),
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

func (retrieveResponseTranslator) Backward(
	_ context.Context,
	r *grpcauth.APIKeyRetrieveResponse,
) (apiapikey.RetrieveResponse, error) {
	keys := make([]apiapikey.APIKey, 0, len(r.Keys))
	for _, pk := range r.Keys {
		k, err := apiKeyBackward(pk)
		if err != nil {
			return apiapikey.RetrieveResponse{}, err
		}
		keys = append(keys, k)
	}
	return apiapikey.RetrieveResponse{Keys: keys}, nil
}

func (deleteRequestTranslator) Forward(
	_ context.Context,
	r apiapikey.DeleteRequest,
) (*grpcauth.APIKeyDeleteRequest, error) {
	return &grpcauth.APIKeyDeleteRequest{
		Keys: lo.Map(r.Keys, func(k uuid.UUID, _ int) string { return k.String() }),
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
