// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package session

import (
	"context"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/api/auth/session"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	grpcauth "github.com/synnaxlabs/synnax/pkg/transport/grpc/auth"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	loginServer = grpc.UnaryServer[
		session.LoginRequest,
		*grpcauth.LoginRequest,
		session.LoginResponse,
		*grpcauth.LoginResponse,
	]
)

type (
	loginRequestTranslator  struct{}
	loginResponseTranslator struct{}
)

var (
	_ grpc.BackwardTranslator[session.LoginRequest, *grpcauth.LoginRequest] = (*loginRequestTranslator)(
		nil,
	)
	_ grpc.ForwardTranslator[session.LoginResponse, *grpcauth.LoginResponse] = (*loginResponseTranslator)(
		nil,
	)
)

func (loginRequestTranslator) Backward(
	_ context.Context,
	req *grpcauth.LoginRequest,
) (session.LoginRequest, error) {
	creds := auth.Credentials{Username: req.Username, Password: req.Password}
	return session.LoginRequest{Credentials: creds}, nil
}

func (loginResponseTranslator) Forward(
	_ context.Context,
	r session.LoginResponse,
) (*grpcauth.LoginResponse, error) {
	return &grpcauth.LoginResponse{
		Token: r.Token,
		User:  &grpcauth.User{Key: r.User.Key.String()},
		ClusterInfo: &grpcauth.ClusterInfo{
			ClusterKey:  r.ClusterInfo.ClusterKey,
			NodeVersion: r.ClusterInfo.NodeVersion,
			NodeKey:     uint32(r.ClusterInfo.NodeKey),
			NodeTime:    int64(r.ClusterInfo.NodeTime),
		},
	}, nil
}

type (
	createServer = grpc.UnaryServer[
		session.CreateRequest,
		*grpcauth.SessionCreateRequest,
		session.CreateResponse,
		*grpcauth.SessionCreateResponse,
	]
	renewServer = grpc.UnaryServer[
		struct{},
		*emptypb.Empty,
		session.RenewResponse,
		*grpcauth.SessionRenewResponse,
	]
	retrieveServer = grpc.UnaryServer[
		session.RetrieveRequest,
		*grpcauth.SessionRetrieveRequest,
		session.RetrieveResponse,
		*grpcauth.SessionRetrieveResponse,
	]
	deleteServer = grpc.UnaryServer[
		session.DeleteRequest,
		*grpcauth.SessionDeleteRequest,
		struct{},
		*emptypb.Empty,
	]
)

type (
	createRequestTranslator    struct{}
	createResponseTranslator   struct{}
	renewResponseTranslator    struct{}
	retrieveRequestTranslator  struct{}
	retrieveResponseTranslator struct{}
	deleteRequestTranslator    struct{}
)

var (
	_ grpc.BackwardTranslator[session.CreateRequest, *grpcauth.SessionCreateRequest] = (*createRequestTranslator)(
		nil,
	)
	_ grpc.ForwardTranslator[session.RenewResponse, *grpcauth.SessionRenewResponse] = (*renewResponseTranslator)(
		nil,
	)
	_ grpc.ForwardTranslator[session.CreateResponse, *grpcauth.SessionCreateResponse] = (*createResponseTranslator)(
		nil,
	)
	_ grpc.BackwardTranslator[session.RetrieveRequest, *grpcauth.SessionRetrieveRequest] = (*retrieveRequestTranslator)(
		nil,
	)
	_ grpc.ForwardTranslator[session.RetrieveResponse, *grpcauth.SessionRetrieveResponse] = (*retrieveResponseTranslator)(
		nil,
	)
	_ grpc.BackwardTranslator[session.DeleteRequest, *grpcauth.SessionDeleteRequest] = (*deleteRequestTranslator)(
		nil,
	)
)

func proofBackward(p *grpcauth.Proof) (auth.Proof, error) {
	if p == nil {
		return auth.Proof{}, errors.Wrap(validate.ErrValidation, "proof: required")
	}
	switch auth.Method(p.Method) {
	case auth.MethodPassword:
		return auth.Proof{Variant: auth.PasswordProof{
			Credentials: auth.Credentials{
				Username: p.Username,
				Password: p.Password,
			},
		}}, nil
	case auth.MethodAPIKey:
		return auth.Proof{Variant: auth.APIKeyProof{
			APIKeyCredentials: auth.APIKeyCredentials{Key: p.Key},
		}}, nil
	case auth.MethodOIDC:
		return auth.Proof{Variant: auth.OIDCProof{
			OIDCCredentials: auth.OIDCCredentials{
				Provider:    p.Provider,
				Code:        p.Code,
				Verifier:    p.Verifier,
				RedirectURI: p.RedirectUri,
			},
		}}, nil
	default:
		return auth.Proof{}, errors.Wrapf(
			validate.ErrValidation,
			"unknown method %q",
			p.Method,
		)
	}
}

func sessionForward(s auth.Session) *grpcauth.Session {
	return &grpcauth.Session{
		Key:        s.Key.String(),
		Subject:    grpcauth.SubjectForward(s.Subject),
		Credential: s.Credential.String(),
		LastActive: int64(s.LastActive),
	}
}

func userForward(u *session.User) *grpcauth.User {
	if u == nil {
		return nil
	}
	return &grpcauth.User{Key: u.Key.String()}
}

func (createRequestTranslator) Backward(
	_ context.Context,
	r *grpcauth.SessionCreateRequest,
) (session.CreateRequest, error) {
	p, err := proofBackward(r.Proof)
	if err != nil {
		return session.CreateRequest{}, err
	}
	return session.CreateRequest{Proof: p}, nil
}

func (createResponseTranslator) Forward(
	_ context.Context,
	r session.CreateResponse,
) (*grpcauth.SessionCreateResponse, error) {
	return &grpcauth.SessionCreateResponse{
		Token:   r.Token,
		Session: sessionForward(r.Session),
		User:    userForward(r.User),
	}, nil
}

func (renewResponseTranslator) Forward(
	_ context.Context,
	r session.RenewResponse,
) (*grpcauth.SessionRenewResponse, error) {
	return &grpcauth.SessionRenewResponse{
		Session: sessionForward(r.Session),
		User:    userForward(r.User),
	}, nil
}

func (retrieveRequestTranslator) Backward(
	_ context.Context,
	r *grpcauth.SessionRetrieveRequest,
) (session.RetrieveRequest, error) {
	return session.RetrieveRequest{
		Subjects: lo.Map(r.Subjects, func(s *grpcauth.Subject, _ int) resource.ID {
			return grpcauth.SubjectBackward(s)
		}),
	}, nil
}

func (retrieveResponseTranslator) Forward(
	_ context.Context,
	r session.RetrieveResponse,
) (*grpcauth.SessionRetrieveResponse, error) {
	return &grpcauth.SessionRetrieveResponse{
		Sessions: lo.Map(r.Sessions, func(s auth.Session, _ int) *grpcauth.Session {
			return sessionForward(s)
		}),
	}, nil
}

func (deleteRequestTranslator) Backward(
	_ context.Context,
	r *grpcauth.SessionDeleteRequest,
) (session.DeleteRequest, error) {
	keys := make([]auth.SessionKey, 0, len(r.Keys))
	for _, k := range r.Keys {
		key, err := uuid.Parse(k)
		if err != nil {
			return session.DeleteRequest{}, err
		}
		keys = append(keys, key)
	}
	return session.DeleteRequest{Keys: keys}, nil
}

// New binds the login and session servers onto t. The servers encode errors with errs.
func New(t *api.Transport, errs *errors.Codec) grpc.BindableTransport {
	login := &loginServer{
		RequestTranslator:  loginRequestTranslator{},
		ResponseTranslator: loginResponseTranslator{},
		ServiceDesc:        &grpcauth.AuthLoginService_ServiceDesc,
		Errors:             errs,
	}
	create := &createServer{
		RequestTranslator:  createRequestTranslator{},
		ResponseTranslator: createResponseTranslator{},
		ServiceDesc:        &grpcauth.AuthSessionCreateService_ServiceDesc,
		Errors:             errs,
	}
	renew := &renewServer{
		RequestTranslator:  grpc.EmptyTranslator{},
		ResponseTranslator: renewResponseTranslator{},
		ServiceDesc:        &grpcauth.AuthSessionRenewService_ServiceDesc,
		Errors:             errs,
	}
	retrieve := &retrieveServer{
		RequestTranslator:  retrieveRequestTranslator{},
		ResponseTranslator: retrieveResponseTranslator{},
		ServiceDesc:        &grpcauth.AuthSessionRetrieveService_ServiceDesc,
		Errors:             errs,
	}
	del := &deleteServer{
		RequestTranslator:  deleteRequestTranslator{},
		ResponseTranslator: grpc.EmptyTranslator{},
		ServiceDesc:        &grpcauth.AuthSessionDeleteService_ServiceDesc,
		Errors:             errs,
	}
	t.AuthLogin = login
	t.AuthSessionCreate = create
	t.AuthSessionRenew = renew
	t.AuthSessionRetrieve = retrieve
	t.AuthSessionDelete = del
	return grpc.CompoundBindableTransport{login, create, renew, retrieve, del}
}
