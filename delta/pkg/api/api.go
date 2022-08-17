package api

import (
	"github.com/arya-analytics/delta/pkg/access"
	"github.com/arya-analytics/delta/pkg/auth"
	"github.com/arya-analytics/delta/pkg/auth/token"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/delta/pkg/distribution/ontology"
	"github.com/arya-analytics/delta/pkg/distribution/segment"
	"github.com/arya-analytics/delta/pkg/storage"
	"github.com/arya-analytics/delta/pkg/user"
	"go.uber.org/zap"
)

type Config struct {
	Logger        *zap.Logger
	Channel       *channel.Service
	Segment       *segment.Service
	Ontology      *ontology.Ontology
	Storage       *storage.Store
	User          *user.Service
	Token         *token.Service
	Authenticator auth.Authenticator
	Enforcer      access.Enforcer
}

type API struct {
	Provider Provider
	Config   Config
	Auth     *AuthService
	Segment  *SegmentService
	Channel  *ChannelService
}

func New(cfg Config) API {
	api := API{
		Config:   cfg,
		Provider: NewProvider(cfg),
	}
	api.Auth = NewAuthService(api.Provider)
	api.Segment = NewSegmentService(api.Provider)
	api.Channel = NewChannelService(api.Provider)
	return api
}
