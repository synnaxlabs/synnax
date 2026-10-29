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
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	apisession "github.com/synnaxlabs/synnax/pkg/api/auth/session"
	svcauth "github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/node"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	grpcauth "github.com/synnaxlabs/synnax/pkg/transport/grpc/auth"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	loginServer = grpc.UnaryServer[
		apisession.LoginRequest,
		*grpcauth.LoginRequest,
		apisession.LoginResponse,
		*grpcauth.LoginResponse,
	]
)

type (
	loginRequestTranslator  struct{}
	loginResponseTranslator struct{}
)

var (
	_ grpc.Translator[apisession.LoginRequest, *grpcauth.LoginRequest] = (*loginRequestTranslator)(
		nil,
	)
	_ grpc.Translator[apisession.LoginResponse, *grpcauth.LoginResponse] = (*loginResponseTranslator)(
		nil,
	)
)

func (loginRequestTranslator) Forward(
	_ context.Context,
	req apisession.LoginRequest,
) (*grpcauth.LoginRequest, error) {
	return &grpcauth.LoginRequest{
		Username: req.Username,
		Password: string(req.Password),
	}, nil
}

func (loginRequestTranslator) Backward(
	_ context.Context,
	req *grpcauth.LoginRequest,
) (apisession.LoginRequest, error) {
	creds := svcauth.Credentials{Username: req.Username, Password: req.Password}
	return apisession.LoginRequest{Credentials: creds}, nil
}

func (loginResponseTranslator) Forward(
	_ context.Context,
	r apisession.LoginResponse,
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

func (loginResponseTranslator) Backward(
	_ context.Context,
	r *grpcauth.LoginResponse,
) (apisession.LoginResponse, error) {
	key, err := uuid.Parse(r.User.Key)
	if err != nil {
		return apisession.LoginResponse{}, err
	}
	return apisession.LoginResponse{
		Token: r.Token,
		User:  user.User{Key: key},
		ClusterInfo: apiauth.ClusterInfo{
			ClusterKey:  r.ClusterInfo.ClusterKey,
			NodeVersion: r.ClusterInfo.NodeVersion,
			NodeKey:     node.Key(r.ClusterInfo.NodeKey),
			NodeTime:    telem.TimeStamp(r.ClusterInfo.NodeTime),
		},
	}, nil
}

type (
	createServer = grpc.UnaryServer[
		apisession.CreateRequest,
		*grpcauth.SessionCreateRequest,
		apisession.CreateResponse,
		*grpcauth.SessionCreateResponse,
	]
	renewServer = grpc.UnaryServer[
		struct{},
		*emptypb.Empty,
		apisession.RenewResponse,
		*grpcauth.SessionRenewResponse,
	]
	retrieveServer = grpc.UnaryServer[
		apisession.RetrieveRequest,
		*grpcauth.SessionRetrieveRequest,
		apisession.RetrieveResponse,
		*grpcauth.SessionRetrieveResponse,
	]
	deleteServer = grpc.UnaryServer[
		apisession.DeleteRequest,
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
	_ grpc.Translator[apisession.CreateRequest, *grpcauth.SessionCreateRequest] = (*createRequestTranslator)(
		nil,
	)
	_ grpc.Translator[apisession.RenewResponse, *grpcauth.SessionRenewResponse] = (*renewResponseTranslator)(
		nil,
	)
	_ grpc.Translator[apisession.CreateResponse, *grpcauth.SessionCreateResponse] = (*createResponseTranslator)(
		nil,
	)
	_ grpc.Translator[apisession.RetrieveRequest, *grpcauth.SessionRetrieveRequest] = (*retrieveRequestTranslator)(
		nil,
	)
	_ grpc.Translator[apisession.RetrieveResponse, *grpcauth.SessionRetrieveResponse] = (*retrieveResponseTranslator)(
		nil,
	)
	_ grpc.Translator[apisession.DeleteRequest, *grpcauth.SessionDeleteRequest] = (*deleteRequestTranslator)(
		nil,
	)
)

func proofForward(p svcauth.Proof) (*grpcauth.Proof, error) {
	switch v := p.Variant.(type) {
	case svcauth.PasswordProof:
		return &grpcauth.Proof{
			Method:   string(svcauth.MethodPassword),
			Username: v.Username,
			Password: v.Password,
		}, nil
	case svcauth.APIKeyProof:
		return &grpcauth.Proof{Method: string(svcauth.MethodAPIKey), Key: v.Key}, nil
	case svcauth.OIDCProof:
		return &grpcauth.Proof{
			Method:      string(svcauth.MethodOIDC),
			Provider:    v.Provider,
			Code:        v.Code,
			Verifier:    v.Verifier,
			RedirectUri: v.RedirectURI,
		}, nil
	default:
		return nil, errors.Wrapf(validate.ErrValidation, "unknown proof %T", v)
	}
}

func proofBackward(p *grpcauth.Proof) (svcauth.Proof, error) {
	if p == nil {
		return svcauth.Proof{}, errors.Wrap(validate.ErrValidation, "proof: required")
	}
	switch svcauth.Method(p.Method) {
	case svcauth.MethodPassword:
		return svcauth.Proof{Variant: svcauth.PasswordProof{
			Credentials: svcauth.Credentials{
				Username: p.Username,
				Password: p.Password,
			},
		}}, nil
	case svcauth.MethodAPIKey:
		return svcauth.Proof{Variant: svcauth.APIKeyProof{
			APIKeyCredentials: svcauth.APIKeyCredentials{Key: p.Key},
		}}, nil
	case svcauth.MethodOIDC:
		return svcauth.Proof{Variant: svcauth.OIDCProof{
			OIDCCredentials: svcauth.OIDCCredentials{
				Provider:    p.Provider,
				Code:        p.Code,
				Verifier:    p.Verifier,
				RedirectURI: p.RedirectUri,
			},
		}}, nil
	default:
		return svcauth.Proof{}, errors.Wrapf(
			validate.ErrValidation,
			"unknown method %q",
			p.Method,
		)
	}
}

func sessionForward(s svcauth.Session) *grpcauth.Session {
	return &grpcauth.Session{
		Key:        s.Key.String(),
		Subject:    grpcauth.SubjectForward(s.Subject),
		Credential: s.Credential.String(),
		LastActive: int64(s.LastActive),
	}
}

func sessionBackward(s *grpcauth.Session) (svcauth.Session, error) {
	key, err := uuid.Parse(s.Key)
	if err != nil {
		return svcauth.Session{}, err
	}
	credential, err := uuid.Parse(s.Credential)
	if err != nil {
		return svcauth.Session{}, err
	}
	return svcauth.Session{
		Key:        key,
		Subject:    grpcauth.SubjectBackward(s.Subject),
		Credential: credential,
		LastActive: telem.TimeStamp(s.LastActive),
	}, nil
}

func userForward(u *user.User) *grpcauth.User {
	if u == nil {
		return nil
	}
	return &grpcauth.User{Key: u.Key.String()}
}

func userBackward(u *grpcauth.User) (*user.User, error) {
	if u == nil {
		return nil, nil
	}
	key, err := uuid.Parse(u.Key)
	if err != nil {
		return nil, err
	}
	return &user.User{Key: key}, nil
}

func (createRequestTranslator) Forward(
	_ context.Context,
	r apisession.CreateRequest,
) (*grpcauth.SessionCreateRequest, error) {
	p, err := proofForward(r.Proof)
	if err != nil {
		return nil, err
	}
	return &grpcauth.SessionCreateRequest{Proof: p}, nil
}

func (createRequestTranslator) Backward(
	_ context.Context,
	r *grpcauth.SessionCreateRequest,
) (apisession.CreateRequest, error) {
	p, err := proofBackward(r.Proof)
	if err != nil {
		return apisession.CreateRequest{}, err
	}
	return apisession.CreateRequest{Proof: p}, nil
}

func (createResponseTranslator) Forward(
	_ context.Context,
	r apisession.CreateResponse,
) (*grpcauth.SessionCreateResponse, error) {
	return &grpcauth.SessionCreateResponse{
		Token:   r.Token,
		Session: sessionForward(r.Session),
		User:    userForward(r.User),
	}, nil
}

func (createResponseTranslator) Backward(
	_ context.Context,
	r *grpcauth.SessionCreateResponse,
) (apisession.CreateResponse, error) {
	session, err := sessionBackward(r.Session)
	if err != nil {
		return apisession.CreateResponse{}, err
	}
	u, err := userBackward(r.User)
	if err != nil {
		return apisession.CreateResponse{}, err
	}
	return apisession.CreateResponse{
		Token:   r.Token,
		Session: session,
		User:    u,
	}, nil
}

func (renewResponseTranslator) Forward(
	_ context.Context,
	r apisession.RenewResponse,
) (*grpcauth.SessionRenewResponse, error) {
	return &grpcauth.SessionRenewResponse{
		Session: sessionForward(r.Session),
		User:    userForward(r.User),
	}, nil
}

func (renewResponseTranslator) Backward(
	_ context.Context,
	r *grpcauth.SessionRenewResponse,
) (apisession.RenewResponse, error) {
	session, err := sessionBackward(r.Session)
	if err != nil {
		return apisession.RenewResponse{}, err
	}
	u, err := userBackward(r.User)
	if err != nil {
		return apisession.RenewResponse{}, err
	}
	return apisession.RenewResponse{Session: session, User: u}, nil
}

func (retrieveRequestTranslator) Forward(
	_ context.Context,
	r apisession.RetrieveRequest,
) (*grpcauth.SessionRetrieveRequest, error) {
	return &grpcauth.SessionRetrieveRequest{
		Subjects: lo.Map(r.Subjects, func(id ontology.ID, _ int) *grpcauth.Subject {
			return grpcauth.SubjectForward(id)
		}),
	}, nil
}

func (retrieveRequestTranslator) Backward(
	_ context.Context,
	r *grpcauth.SessionRetrieveRequest,
) (apisession.RetrieveRequest, error) {
	return apisession.RetrieveRequest{
		Subjects: lo.Map(r.Subjects, func(s *grpcauth.Subject, _ int) ontology.ID {
			return grpcauth.SubjectBackward(s)
		}),
	}, nil
}

func (retrieveResponseTranslator) Forward(
	_ context.Context,
	r apisession.RetrieveResponse,
) (*grpcauth.SessionRetrieveResponse, error) {
	return &grpcauth.SessionRetrieveResponse{
		Sessions: lo.Map(r.Sessions, func(s svcauth.Session, _ int) *grpcauth.Session {
			return sessionForward(s)
		}),
	}, nil
}

func (retrieveResponseTranslator) Backward(
	_ context.Context,
	r *grpcauth.SessionRetrieveResponse,
) (apisession.RetrieveResponse, error) {
	sessions := make([]svcauth.Session, 0, len(r.Sessions))
	for _, s := range r.Sessions {
		session, err := sessionBackward(s)
		if err != nil {
			return apisession.RetrieveResponse{}, err
		}
		sessions = append(sessions, session)
	}
	return apisession.RetrieveResponse{Sessions: sessions}, nil
}

func (deleteRequestTranslator) Forward(
	_ context.Context,
	r apisession.DeleteRequest,
) (*grpcauth.SessionDeleteRequest, error) {
	return &grpcauth.SessionDeleteRequest{
		Keys: lo.Map(r.Keys, func(k svcauth.SessionKey, _ int) string {
			return k.String()
		}),
	}, nil
}

func (deleteRequestTranslator) Backward(
	_ context.Context,
	r *grpcauth.SessionDeleteRequest,
) (apisession.DeleteRequest, error) {
	keys := make([]svcauth.SessionKey, 0, len(r.Keys))
	for _, k := range r.Keys {
		key, err := uuid.Parse(k)
		if err != nil {
			return apisession.DeleteRequest{}, err
		}
		keys = append(keys, key)
	}
	return apisession.DeleteRequest{Keys: keys}, nil
}

// New binds the login and session servers onto t.
func New(t *api.Transport) grpc.BindableTransport {
	login := &loginServer{
		RequestTranslator:  loginRequestTranslator{},
		ResponseTranslator: loginResponseTranslator{},
		ServiceDesc:        &grpcauth.AuthLoginService_ServiceDesc,
	}
	create := &createServer{
		RequestTranslator:  createRequestTranslator{},
		ResponseTranslator: createResponseTranslator{},
		ServiceDesc:        &grpcauth.AuthSessionCreateService_ServiceDesc,
	}
	renew := &renewServer{
		RequestTranslator:  grpc.EmptyTranslator{},
		ResponseTranslator: renewResponseTranslator{},
		ServiceDesc:        &grpcauth.AuthSessionRenewService_ServiceDesc,
	}
	retrieve := &retrieveServer{
		RequestTranslator:  retrieveRequestTranslator{},
		ResponseTranslator: retrieveResponseTranslator{},
		ServiceDesc:        &grpcauth.AuthSessionRetrieveService_ServiceDesc,
	}
	del := &deleteServer{
		RequestTranslator:  deleteRequestTranslator{},
		ResponseTranslator: grpc.EmptyTranslator{},
		ServiceDesc:        &grpcauth.AuthSessionDeleteService_ServiceDesc,
	}
	t.AuthLogin = login
	t.AuthSessionCreate = create
	t.AuthSessionRenew = renew
	t.AuthSessionRetrieve = retrieve
	t.AuthSessionDelete = del
	return grpc.CompoundBindableTransport{login, create, renew, retrieve, del}
}
