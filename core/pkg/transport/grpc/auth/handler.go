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
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	svcauth "github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	loginServer = grpc.UnaryServer[
		auth.LoginRequest,
		*LoginRequest,
		auth.LoginResponse,
		*LoginResponse,
	]
)

type (
	loginRequestTranslator  struct{}
	loginResponseTranslator struct{}
)

var (
	_ grpc.BackwardTranslator[auth.LoginRequest, *LoginRequest] = (*loginRequestTranslator)(
		nil,
	)
	_ grpc.ForwardTranslator[auth.LoginResponse, *LoginResponse] = (*loginResponseTranslator)(
		nil,
	)
)

func (loginRequestTranslator) Backward(
	_ context.Context,
	req *LoginRequest,
) (auth.LoginRequest, error) {
	creds := auth.Credentials{Username: req.Username, Password: req.Password}
	return auth.LoginRequest{Credentials: creds}, nil
}

func (loginResponseTranslator) Forward(
	_ context.Context,
	r auth.LoginResponse,
) (*LoginResponse, error) {
	return &LoginResponse{
		Token: r.Token,
		User:  &User{Key: r.User.Key.String()},
		ClusterInfo: &ClusterInfo{
			ClusterKey:  r.ClusterInfo.ClusterKey,
			NodeVersion: r.ClusterInfo.NodeVersion,
			NodeKey:     uint32(r.ClusterInfo.NodeKey),
			NodeTime:    int64(r.ClusterInfo.NodeTime),
		},
	}, nil
}

type (
	sessionCreateServer = grpc.UnaryServer[
		auth.SessionCreateRequest,
		*SessionCreateRequest,
		auth.SessionCreateResponse,
		*SessionCreateResponse,
	]
	sessionRenewServer = grpc.UnaryServer[
		struct{},
		*emptypb.Empty,
		auth.SessionRenewResponse,
		*SessionRenewResponse,
	]
	sessionRetrieveServer = grpc.UnaryServer[
		auth.SessionRetrieveRequest,
		*SessionRetrieveRequest,
		auth.SessionRetrieveResponse,
		*SessionRetrieveResponse,
	]
	sessionDeleteServer = grpc.UnaryServer[
		auth.SessionDeleteRequest,
		*SessionDeleteRequest,
		struct{},
		*emptypb.Empty,
	]
)

type (
	sessionCreateRequestTranslator    struct{}
	sessionCreateResponseTranslator   struct{}
	sessionRenewResponseTranslator    struct{}
	sessionRetrieveRequestTranslator  struct{}
	sessionRetrieveResponseTranslator struct{}
	sessionDeleteRequestTranslator    struct{}
)

var (
	_ grpc.BackwardTranslator[auth.SessionCreateRequest, *SessionCreateRequest] = (*sessionCreateRequestTranslator)(
		nil,
	)
	_ grpc.ForwardTranslator[auth.SessionRenewResponse, *SessionRenewResponse] = (*sessionRenewResponseTranslator)(
		nil,
	)
	_ grpc.ForwardTranslator[auth.SessionCreateResponse, *SessionCreateResponse] = (*sessionCreateResponseTranslator)(
		nil,
	)
	_ grpc.BackwardTranslator[auth.SessionRetrieveRequest, *SessionRetrieveRequest] = (*sessionRetrieveRequestTranslator)(
		nil,
	)
	_ grpc.ForwardTranslator[auth.SessionRetrieveResponse, *SessionRetrieveResponse] = (*sessionRetrieveResponseTranslator)(
		nil,
	)
	_ grpc.BackwardTranslator[auth.SessionDeleteRequest, *SessionDeleteRequest] = (*sessionDeleteRequestTranslator)(
		nil,
	)
)

func proofBackward(p *Proof) (svcauth.Proof, error) {
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
	default:
		return svcauth.Proof{}, errors.Wrapf(
			validate.ErrValidation,
			"unknown method %q",
			p.Method,
		)
	}
}

func subjectForward(id ontology.ID) *Subject {
	return &Subject{Type: string(id.Type), Key: id.Key}
}

func subjectBackward(s *Subject) ontology.ID {
	if s == nil {
		return ontology.ID{}
	}
	return ontology.ID{Type: ontology.ResourceType(s.Type), Key: s.Key}
}

func sessionForward(s svcauth.Session) *Session {
	return &Session{
		Key:        s.Key.String(),
		Subject:    subjectForward(s.Subject),
		Credential: s.Credential.String(),
		LastActive: int64(s.LastActive),
	}
}

func userForward(u *user.User) *User {
	if u == nil {
		return nil
	}
	return &User{Key: u.Key.String()}
}

func (sessionCreateRequestTranslator) Backward(
	_ context.Context,
	r *SessionCreateRequest,
) (auth.SessionCreateRequest, error) {
	p, err := proofBackward(r.Proof)
	if err != nil {
		return auth.SessionCreateRequest{}, err
	}
	return auth.SessionCreateRequest{Proof: p}, nil
}

func (sessionCreateResponseTranslator) Forward(
	_ context.Context,
	r auth.SessionCreateResponse,
) (*SessionCreateResponse, error) {
	return &SessionCreateResponse{
		Token:   r.Token,
		Session: sessionForward(r.Session),
		User:    userForward(r.User),
	}, nil
}

func (sessionRenewResponseTranslator) Forward(
	_ context.Context,
	r auth.SessionRenewResponse,
) (*SessionRenewResponse, error) {
	return &SessionRenewResponse{
		Session: sessionForward(r.Session),
		User:    userForward(r.User),
	}, nil
}

func (sessionRetrieveRequestTranslator) Backward(
	_ context.Context,
	r *SessionRetrieveRequest,
) (auth.SessionRetrieveRequest, error) {
	return auth.SessionRetrieveRequest{
		Subjects: lo.Map(r.Subjects, func(s *Subject, _ int) ontology.ID {
			return subjectBackward(s)
		}),
	}, nil
}

func (sessionRetrieveResponseTranslator) Forward(
	_ context.Context,
	r auth.SessionRetrieveResponse,
) (*SessionRetrieveResponse, error) {
	return &SessionRetrieveResponse{
		Sessions: lo.Map(r.Sessions, func(s svcauth.Session, _ int) *Session {
			return sessionForward(s)
		}),
	}, nil
}

func (sessionDeleteRequestTranslator) Backward(
	_ context.Context,
	r *SessionDeleteRequest,
) (auth.SessionDeleteRequest, error) {
	keys := make([]svcauth.SessionKey, 0, len(r.Keys))
	for _, k := range r.Keys {
		key, err := uuid.Parse(k)
		if err != nil {
			return auth.SessionDeleteRequest{}, err
		}
		keys = append(keys, key)
	}
	return auth.SessionDeleteRequest{Keys: keys}, nil
}

type (
	keyCreateServer = grpc.UnaryServer[
		auth.KeyCreateRequest,
		*KeyCreateRequest,
		auth.KeyCreateResponse,
		*KeyCreateResponse,
	]
	keyRetrieveServer = grpc.UnaryServer[
		auth.KeyRetrieveRequest,
		*KeyRetrieveRequest,
		auth.KeyRetrieveResponse,
		*KeyRetrieveResponse,
	]
	keyDeleteServer = grpc.UnaryServer[
		auth.KeyDeleteRequest,
		*KeyDeleteRequest,
		struct{},
		*emptypb.Empty,
	]
)

type (
	keyCreateRequestTranslator    struct{}
	keyCreateResponseTranslator   struct{}
	keyRetrieveRequestTranslator  struct{}
	keyRetrieveResponseTranslator struct{}
	keyDeleteRequestTranslator    struct{}
)

var (
	_ grpc.BackwardTranslator[auth.KeyCreateRequest, *KeyCreateRequest] = (*keyCreateRequestTranslator)(
		nil,
	)
	_ grpc.ForwardTranslator[auth.KeyCreateResponse, *KeyCreateResponse] = (*keyCreateResponseTranslator)(
		nil,
	)
	_ grpc.BackwardTranslator[auth.KeyRetrieveRequest, *KeyRetrieveRequest] = (*keyRetrieveRequestTranslator)(
		nil,
	)
	_ grpc.ForwardTranslator[auth.KeyRetrieveResponse, *KeyRetrieveResponse] = (*keyRetrieveResponseTranslator)(
		nil,
	)
	_ grpc.BackwardTranslator[auth.KeyDeleteRequest, *KeyDeleteRequest] = (*keyDeleteRequestTranslator)(
		nil,
	)
)

func apiKeyForward(k auth.APIKey) *APIKey {
	return &APIKey{
		Key:     k.Key.String(),
		Subject: subjectForward(k.Subject),
		Name:    k.Name,
		Created: int64(k.Created),
	}
}

func (keyCreateRequestTranslator) Backward(
	_ context.Context,
	r *KeyCreateRequest,
) (auth.KeyCreateRequest, error) {
	return auth.KeyCreateRequest{Subject: subjectBackward(r.Subject), Name: r.Name}, nil
}

func (keyCreateResponseTranslator) Forward(
	_ context.Context,
	r auth.KeyCreateResponse,
) (*KeyCreateResponse, error) {
	return &KeyCreateResponse{Token: r.Token, Key: apiKeyForward(r.Key)}, nil
}

func (keyRetrieveRequestTranslator) Backward(
	_ context.Context,
	r *KeyRetrieveRequest,
) (auth.KeyRetrieveRequest, error) {
	return auth.KeyRetrieveRequest{
		Subjects: lo.Map(r.Subjects, func(s *Subject, _ int) ontology.ID {
			return subjectBackward(s)
		}),
	}, nil
}

func (keyRetrieveResponseTranslator) Forward(
	_ context.Context,
	r auth.KeyRetrieveResponse,
) (*KeyRetrieveResponse, error) {
	return &KeyRetrieveResponse{
		Keys: lo.Map(r.Keys, func(k auth.APIKey, _ int) *APIKey {
			return apiKeyForward(k)
		}),
	}, nil
}

func (keyDeleteRequestTranslator) Backward(
	_ context.Context,
	r *KeyDeleteRequest,
) (auth.KeyDeleteRequest, error) {
	keys := make([]uuid.UUID, 0, len(r.Keys))
	for _, k := range r.Keys {
		key, err := uuid.Parse(k)
		if err != nil {
			return auth.KeyDeleteRequest{}, err
		}
		keys = append(keys, key)
	}
	return auth.KeyDeleteRequest{Keys: keys}, nil
}

func New(t *api.Transport) grpc.BindableTransport {
	login := &loginServer{
		RequestTranslator:  loginRequestTranslator{},
		ResponseTranslator: loginResponseTranslator{},
		ServiceDesc:        &AuthLoginService_ServiceDesc,
	}
	create := &sessionCreateServer{
		RequestTranslator:  sessionCreateRequestTranslator{},
		ResponseTranslator: sessionCreateResponseTranslator{},
		ServiceDesc:        &AuthSessionCreateService_ServiceDesc,
	}
	renew := &sessionRenewServer{
		RequestTranslator:  grpc.EmptyTranslator{},
		ResponseTranslator: sessionRenewResponseTranslator{},
		ServiceDesc:        &AuthSessionRenewService_ServiceDesc,
	}
	retrieve := &sessionRetrieveServer{
		RequestTranslator:  sessionRetrieveRequestTranslator{},
		ResponseTranslator: sessionRetrieveResponseTranslator{},
		ServiceDesc:        &AuthSessionRetrieveService_ServiceDesc,
	}
	del := &sessionDeleteServer{
		RequestTranslator:  sessionDeleteRequestTranslator{},
		ResponseTranslator: grpc.EmptyTranslator{},
		ServiceDesc:        &AuthSessionDeleteService_ServiceDesc,
	}
	keyCreate := &keyCreateServer{
		RequestTranslator:  keyCreateRequestTranslator{},
		ResponseTranslator: keyCreateResponseTranslator{},
		ServiceDesc:        &AuthKeyCreateService_ServiceDesc,
	}
	keyRetrieve := &keyRetrieveServer{
		RequestTranslator:  keyRetrieveRequestTranslator{},
		ResponseTranslator: keyRetrieveResponseTranslator{},
		ServiceDesc:        &AuthKeyRetrieveService_ServiceDesc,
	}
	keyDelete := &keyDeleteServer{
		RequestTranslator:  keyDeleteRequestTranslator{},
		ResponseTranslator: grpc.EmptyTranslator{},
		ServiceDesc:        &AuthKeyDeleteService_ServiceDesc,
	}
	t.AuthLogin = login
	t.AuthSessionCreate = create
	t.AuthSessionRenew = renew
	t.AuthSessionRetrieve = retrieve
	t.AuthSessionDelete = del
	t.AuthKeyCreate = keyCreate
	t.AuthKeyRetrieve = keyRetrieve
	t.AuthKeyDelete = keyDelete
	return grpc.CompoundBindableTransport{
		login,
		create,
		renew,
		retrieve,
		del,
		keyCreate,
		keyRetrieve,
		keyDelete,
	}
}
