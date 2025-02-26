// Copyright 2025 Synnax Labs, Inc.
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
	dcore "github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/x/telem"

	"github.com/google/uuid"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	gapi "github.com/synnaxlabs/synnax/pkg/api/grpc/v1"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"google.golang.org/grpc"
)

type (
	authLoginServer = fgrpc.UnaryServer[
		api.AuthLoginRequest,
		*gapi.LoginRequest,
		api.AuthLoginResponse,
		*gapi.LoginResponse,
	]
	authLoginClient = fgrpc.UnaryClient[
		api.AuthLoginRequest,
		*gapi.LoginRequest,
		api.AuthLoginResponse,
		*gapi.LoginResponse,
	]
)

type (
	loginRequestTranslator  struct{}
	loginResponseTranslator struct{}
)

var (
	_ fgrpc.Translator[api.AuthLoginRequest, *gapi.LoginRequest]   = (*loginRequestTranslator)(nil)
	_ fgrpc.Translator[api.AuthLoginResponse, *gapi.LoginResponse] = (*loginResponseTranslator)(nil)
)

func (l loginRequestTranslator) Forward(
	_ context.Context,
	req api.AuthLoginRequest,
) (*gapi.LoginRequest, error) {
	return &gapi.LoginRequest{Username: req.Username, Password: string(req.Password)}, nil
}

func (l loginRequestTranslator) Backward(
	_ context.Context,
	req *gapi.LoginRequest,
) (api.AuthLoginRequest, error) {
	creds := auth.InsecureCredentials{Username: req.Username, Password: password.Raw(req.Password)}
	return api.AuthLoginRequest{InsecureCredentials: creds}, nil
}

func (l loginResponseTranslator) Forward(
	_ context.Context,
	r api.AuthLoginResponse,
) (*gapi.LoginResponse, error) {
	return &gapi.LoginResponse{
		Token: r.Token,
		User: &gapi.User{
			Key:      r.User.Key.String(),
			Username: r.User.Username,
		},
		ClusterInfo: &gapi.ClusterInfo{
			ClusterKey:  r.ClusterInfo.ClusterKey,
			NodeVersion: r.ClusterInfo.NodeVersion,
			NodeKey:     uint32(r.ClusterInfo.NodeKey),
			NodeTime:    int64(r.ClusterInfo.NodeTime),
		},
	}, nil
}

func (l loginResponseTranslator) Backward(
	_ context.Context,
	r *gapi.LoginResponse,
) (api.AuthLoginResponse, error) {
	key, err := uuid.Parse(r.User.Key)
	return api.AuthLoginResponse{
		Token: r.Token,
		User: user.User{
			Key:      key,
			Username: r.User.Username,
		},
		ClusterInfo: api.ClusterInfo{
			ClusterKey:  r.ClusterInfo.ClusterKey,
			NodeVersion: r.ClusterInfo.NodeVersion,
			NodeKey:     dcore.NodeKey(r.ClusterInfo.NodeKey),
			NodeTime:    telem.TimeStamp(r.ClusterInfo.NodeTime),
		},
	}, err
}

func newAuth(a *api.Transport) fgrpc.BindableTransport {
	s := &authLoginServer{
		RequestTranslator:  loginRequestTranslator{},
		ResponseTranslator: loginResponseTranslator{},
		ServiceDesc:        &gapi.AuthLoginService_ServiceDesc,
	}
	a.AuthLogin = s
	return s
}

func NewAuthLoginClient(pool *fgrpc.Pool) freighter.UnaryClient[api.AuthLoginRequest, api.AuthLoginResponse] {
	return &authLoginClient{
		Pool:               pool,
		RequestTranslator:  loginRequestTranslator{},
		ResponseTranslator: loginResponseTranslator{},
		ServiceDesc:        &gapi.AuthLoginService_ServiceDesc,
		Exec: func(ctx context.Context, connInterface grpc.ClientConnInterface, request *gapi.LoginRequest) (*gapi.LoginResponse, error) {
			return gapi.NewAuthLoginServiceClient(connInterface).Exec(ctx, request)
		},
	}
}
