package channel

import "C"
import (
	"context"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// Service is central entity for managing channels within delta's distribution layer. It provides facilities for creating
// and retrieving channels.
type Service struct {
	clusterDB *gorp.DB
	proxy     *leaseProxy
}

type Config struct {
	HostResolver core.HostResolver
	ClusterDB    *gorp.DB
	TS           storage.TS
	Transport    Transport
	Ontology     *ontology.Ontology
}

var _ config.Config[Config] = Config{}

func (c Config) Validate() error {
	v := validate.New("distribution.channel")
	validate.NotNil(v, "HostResolver", c.HostResolver)
	validate.NotNil(v, "ClusterDB", c.ClusterDB)
	validate.NotNil(v, "TS", c.TS)
	validate.NotNil(v, "Transport", c.Transport)
	return v.Error()
}

func (c Config) Override(other Config) Config {
	c.HostResolver = override.Nil(c.HostResolver, other.HostResolver)
	c.ClusterDB = override.Nil(c.ClusterDB, other.ClusterDB)
	c.TS = override.Nil(c.TS, other.TS)
	c.Transport = override.Nil(c.Transport, other.Transport)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	return c
}

var DefaultConfig = Config{}

func New(configs ...Config) (*Service, error) {
	cfg, err := config.OverrideAndValidate(DefaultConfig, configs...)
	if err != nil {
		return nil, err
	}
	proxy, err := newLeaseProxy(cfg)
	if err != nil {
		return nil, err
	}
	return &Service{clusterDB: cfg.ClusterDB, proxy: proxy}, nil
}

func (s *Service) Create(channel *Channel) error {
	return s.CreateWithTxn(s.clusterDB, channel)
}

func (s *Service) CreateMany(channels *[]Channel) error {
	return s.CreateManyWithTxn(s.clusterDB, channels)
}

func (s *Service) CreateWithTxn(txn gorp.Txn, ch *Channel) error {
	channels := []Channel{*ch}
	err := s.proxy.create(context.TODO(), txn, &channels)
	if err != nil {
		return err
	}
	*ch = channels[0]
	return nil
}

func (s *Service) CreateManyWithTxn(txn gorp.Txn, channels *[]Channel) error {
	return s.proxy.create(context.TODO(), txn, channels)
}

func (s *Service) NewRetrieve() Retrieve { return newRetrieve(s.clusterDB) }
