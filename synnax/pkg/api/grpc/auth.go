// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package grpc

import (
	"context"

	"github.com/google/uuid"
	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	gapi "github.com/synnaxlabs/synnax/pkg/api/grpc/v1"
	"github.com/synnaxlabs/synnax/pkg/auth"
	"github.com/synnaxlabs/synnax/pkg/auth/password"
	"github.com/synnaxlabs/synnax/pkg/user"
)

type (
	authServer = fgrpc.UnaryServer[
		auth.InsecureCredentials,
		*gapi.LoginRequest,
		api.TokenResponse,
		*gapi.LoginResponse,
	]
)

type (
	loginRequestTranslator  struct{}
	loginResponseTranslator struct{}
)

var (
	_ fgrpc.Translator[auth.InsecureCredentials, *gapi.LoginRequest] = (*loginRequestTranslator)(nil)
	_ fgrpc.Translator[api.TokenResponse, *gapi.LoginResponse]       = (*loginResponseTranslator)(nil)
)

func (l loginRequestTranslator) Forward(
	_ context.Context,
	creds auth.InsecureCredentials,
) (*gapi.LoginRequest, error) {
	return &gapi.LoginRequest{Username: creds.Username, Password: string(creds.Password)}, nil
}

func (l loginRequestTranslator) Backward(
	_ context.Context,
	req *gapi.LoginRequest,
) (auth.InsecureCredentials, error) {
	return auth.InsecureCredentials{Username: req.Username, Password: password.Raw(req.Password)}, nil
}

func (l loginResponseTranslator) Forward(
	_ context.Context,
	r api.TokenResponse,
) (*gapi.LoginResponse, error) {
	return &gapi.LoginResponse{
		Token: r.Token,
		User: &gapi.User{
			Key:      r.User.Key.String(),
			Username: r.User.Username,
		},
	}, nil
}

func (l loginResponseTranslator) Backward(
	_ context.Context,
	r *gapi.LoginResponse,
) (api.TokenResponse, error) {
	key, err := uuid.Parse(r.User.Key)
	return api.TokenResponse{
		Token: r.Token,
		User: user.User{
			Key:      key,
			Username: r.User.Username,
		},
	}, err
}

func newAuth(a *api.Transport) fgrpc.BindableTransport {
	s := &authServer{
		RequestTranslator:  loginRequestTranslator{},
		ResponseTranslator: loginResponseTranslator{},
		ServiceDesc:        &gapi.AuthLoginService_ServiceDesc,
	}
	a.AuthLogin = s
	return s
}
