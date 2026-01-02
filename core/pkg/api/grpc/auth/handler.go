// Copyright 2025 Synnax Labs, Inc.
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
	gapi "github.com/synnaxlabs/synnax/pkg/api/grpc/v1"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	svcauth "github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/telem"
)

type (
	loginServer = fgrpc.UnaryServer[
		apiauth.LoginRequest,
		*gapi.LoginRequest,
		apiauth.LoginResponse,
		*gapi.LoginResponse,
	]
)

type (
	loginRequestTranslator  struct{}
	loginResponseTranslator struct{}
)

var (
	_ fgrpc.Translator[apiauth.LoginRequest, *gapi.LoginRequest]   = (*loginRequestTranslator)(nil)
	_ fgrpc.Translator[apiauth.LoginResponse, *gapi.LoginResponse] = (*loginResponseTranslator)(nil)
)

func (l loginRequestTranslator) Forward(
	_ context.Context,
	req apiauth.LoginRequest,
) (*gapi.LoginRequest, error) {
	return &gapi.LoginRequest{Username: req.Username, Password: string(req.Password)}, nil
}

func (l loginRequestTranslator) Backward(
	_ context.Context,
	req *gapi.LoginRequest,
) (apiauth.LoginRequest, error) {
	creds := svcauth.InsecureCredentials{Username: req.Username, Password: password.Raw(req.Password)}
	return apiauth.LoginRequest{InsecureCredentials: creds}, nil
}

func (l loginResponseTranslator) Forward(
	_ context.Context,
	r apiauth.LoginResponse,
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
		ServiceDesc:        &gapi.AuthLoginService_ServiceDesc,
	}
	a.AuthLogin = s
	return s
}
