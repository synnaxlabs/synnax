package mock

import (
	"crypto/rand"
	"crypto/rsa"
	"github.com/arya-analytics/delta/pkg/access"
	"github.com/arya-analytics/delta/pkg/api"
	"github.com/arya-analytics/delta/pkg/auth"
	"github.com/arya-analytics/delta/pkg/auth/token"
	"github.com/arya-analytics/delta/pkg/distribution"
	"github.com/arya-analytics/delta/pkg/distribution/mock"
	"github.com/arya-analytics/delta/pkg/user"
	"go.uber.org/zap"
	"time"
)

type ProviderBuilder struct {
	mock.Builder
}

func (b *ProviderBuilder) New() api.Provider {
	dist := b.Builder.New()
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		panic(err)
	}
	return api.NewProvider(api.Config{
		Logger:        zap.NewNop(),
		Channel:       dist.Channel,
		Segment:       dist.Segment,
		Ontology:      dist.Ontology,
		Storage:       dist.Storage,
		User:          &user.Service{DB: dist.Storage.Gorpify(), Ontology: dist.Ontology},
		Token:         &token.Service{Secret: key, Expiration: 10000 * time.Hour},
		Authenticator: &auth.KV{DB: dist.Storage.Gorpify()},
		Enforcer:      access.AllowAll{},
	})
}

func NewProviderBuilder(cfg ...distribution.Config) *ProviderBuilder {
	builder := &ProviderBuilder{}
	builder.Builder = *mock.NewBuilder(cfg...)
	return builder
}
