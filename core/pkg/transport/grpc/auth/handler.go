// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package auth

import (
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	svcauth "github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"google.golang.org/protobuf/types/known/emptypb"
)

// SubjectForward translates an ontology ID to its protobuf subject.
func SubjectForward(id ontology.ID) *Subject {
	return &Subject{Type: string(id.Type), Key: id.Key}
}

// SubjectBackward translates a protobuf subject to an ontology ID. A nil subject is the
// zero ID.
func SubjectBackward(s *Subject) ontology.ID {
	if s == nil {
		return ontology.ID{}
	}
	return ontology.ID{Type: ontology.ResourceType(s.Type), Key: s.Key}
}

type (
	methodsServer = grpc.UnaryServer[
		struct{},
		*emptypb.Empty,
		auth.MethodsResponse,
		*MethodsResponse,
	]
	methodsResponseTranslator struct{}
)

var _ grpc.ForwardTranslator[auth.MethodsResponse, *MethodsResponse] = (*methodsResponseTranslator)(
	nil,
)

func (methodsResponseTranslator) Forward(
	_ context.Context,
	r auth.MethodsResponse,
) (*MethodsResponse, error) {
	return &MethodsResponse{
		Methods: lo.Map(r.Methods, func(m svcauth.Method, _ int) string {
			return string(m)
		}),
	}, nil
}

// New binds the login methods server onto t.
func New(t *api.Transport) grpc.BindableTransport {
	methods := &methodsServer{
		RequestTranslator:  grpc.EmptyTranslator{},
		ResponseTranslator: methodsResponseTranslator{},
		ServiceDesc:        &AuthMethodsService_ServiceDesc,
	}
	t.AuthMethods = methods
	return methods
}
