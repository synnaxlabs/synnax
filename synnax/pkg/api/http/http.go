package http

import (
	"github.com/synnaxlabs/freighter/fhttp"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/auth"
	"go/types"
)

func New(router *fhttp.Router) (a api.Server) {
	a.AuthLogin = fhttp.UnaryPostServer[auth.InsecureCredentials, api.TokenResponse](router, "/api/v1/auth/login")
	a.AuthRegistration = fhttp.UnaryPostServer[api.RegistrationRequest, api.TokenResponse](router, "/api/v1/auth/register")
	a.AuthChangePassword = fhttp.UnaryPostServer[api.ChangePasswordRequest, types.Nil](router, "/api/v1/auth/protected/change-password")
	a.AuthChangeUsername = fhttp.UnaryPostServer[api.ChangeUsernameRequest, types.Nil](router, "/api/v1/auth/protected/change-username")
	a.ChannelCreate = fhttp.UnaryPostServer[api.ChannelCreateRequest, api.ChannelCreateResponse](router, "/api/v1/channel/create")
	a.ChannelRetrieve = fhttp.UnaryGetServer[api.ChannelRetrieveRequest, api.ChannelRetrieveResponse](router, "/api/v1/channel/retrieve")
	a.SegmentWriter = fhttp.StreamServer[api.SegmentWriterRequest, api.SegmentWriterResponse](router, "/api/v1/segment/write")
	a.SegmentIterator = fhttp.StreamServer[api.SegmentIteratorRequest, api.SegmentIteratorResponse](router, "/api/v1/segment/read")
	return a
}
