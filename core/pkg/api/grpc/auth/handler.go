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
	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	svcauth "github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/telem"
)

type (
	loginServer = fgrpc.UnaryServer[
		apiauth.LoginRequest,
		*LoginRequest,
		apiauth.LoginResponse,
		*LoginResponse,
	]
)

type (
	loginRequestTranslator  struct{}
	loginResponseTranslator struct{}
)

var (
	_ fgrpc.Translator[apiauth.LoginRequest, *LoginRequest]   = (*loginRequestTranslator)(nil)
	_ fgrpc.Translator[apiauth.LoginResponse, *LoginResponse] = (*loginResponseTranslator)(nil)
)

func (l loginRequestTranslator) Forward(
	_ context.Context,
	req apiauth.LoginRequest,
) (*LoginRequest, error) {
	return &LoginRequest{Username: req.Username, Password: string(req.Password)}, nil
}

func (l loginRequestTranslator) Backward(
	_ context.Context,
	req *LoginRequest,
) (apiauth.LoginRequest, error) {
	creds := svcauth.InsecureCredentials{Username: req.Username, Password: password.Raw(req.Password)}
	return apiauth.LoginRequest{InsecureCredentials: creds}, nil
}

func (l loginResponseTranslator) Forward(
	_ context.Context,
	r apiauth.LoginResponse,
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

func (l loginResponseTranslator) Backward(
	_ context.Context,
	r *LoginResponse,
) (apiauth.LoginResponse, error) {
	key, err := uuid.Parse(r.User.Key)
	return apiauth.LoginResponse{
		Token: r.Token,
		User: user.User{
			Key:      key,
			Username: r.User.Username,
		},
		ClusterInfo: apiauth.ClusterInfo{
			ClusterKey:  r.ClusterInfo.ClusterKey,
			NodeVersion: r.ClusterInfo.NodeVersion,
			NodeKey:     cluster.NodeKey(r.ClusterInfo.NodeKey),
			NodeTime:    telem.TimeStamp(r.ClusterInfo.NodeTime),
		},
	}, err
}

func New(a *api.Transport) fgrpc.BindableTransport {
	s := &loginServer{
		RequestTranslator:  loginRequestTranslator{},
		ResponseTranslator: loginResponseTranslator{},
		ServiceDesc:        &AuthLoginService_ServiceDesc,
	}
	a.AuthLogin = s
	return s
}
