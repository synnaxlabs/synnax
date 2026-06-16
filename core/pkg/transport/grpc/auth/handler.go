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

	"github.com/google/uuid"
	"github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/service/node"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/telem"
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
	_ grpc.Translator[auth.LoginRequest, *LoginRequest]   = (*loginRequestTranslator)(nil)
	_ grpc.Translator[auth.LoginResponse, *LoginResponse] = (*loginResponseTranslator)(nil)
)

func (loginRequestTranslator) Forward(
	_ context.Context,
	req auth.LoginRequest,
) (*LoginRequest, error) {
	return &LoginRequest{Username: req.Username, Password: string(req.Password)}, nil
}

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
		User: &User{
			Key:      r.User.Key.String(),
			Username: r.User.Username,
		},
		ClusterInfo: &ClusterInfo{
			ClusterKey:  r.ClusterInfo.ClusterKey,
			NodeVersion: r.ClusterInfo.NodeVersion,
			NodeKey:     uint32(r.ClusterInfo.NodeKey),
			NodeTime:    int64(r.ClusterInfo.NodeTime),
		},
	}, nil
}

func (loginResponseTranslator) Backward(
	_ context.Context,
	r *LoginResponse,
) (auth.LoginResponse, error) {
	key, err := uuid.Parse(r.User.Key)
	if err != nil {
		return auth.LoginResponse{}, err
	}
	return auth.LoginResponse{
		Token: r.Token,
		User: user.User{
			Key:      key,
			Username: r.User.Username,
		},
		ClusterInfo: auth.ClusterInfo{
			ClusterKey:  r.ClusterInfo.ClusterKey,
			NodeVersion: r.ClusterInfo.NodeVersion,
			NodeKey:     node.Key(r.ClusterInfo.NodeKey),
			NodeTime:    telem.TimeStamp(r.ClusterInfo.NodeTime),
		},
	}, nil
}

func New(t *api.Transport) grpc.BindableTransport {
	s := &loginServer{
		RequestTranslator:  loginRequestTranslator{},
		ResponseTranslator: loginResponseTranslator{},
		ServiceDesc:        &AuthLoginService_ServiceDesc,
	}
	t.AuthLogin = s
	return s
}
