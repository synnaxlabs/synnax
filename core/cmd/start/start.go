// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package start

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	aspentransport "github.com/synnaxlabs/aspen/transport/grpc"
	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/freighter/fhttp"
	cmdcert "github.com/synnaxlabs/synnax/cmd/cert"
	cmdauth "github.com/synnaxlabs/synnax/cmd/start/auth"
	"github.com/synnaxlabs/synnax/pkg/api"
	grpcapi "github.com/synnaxlabs/synnax/pkg/api/grpc"
	httpapi "github.com/synnaxlabs/synnax/pkg/api/http"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	channeltransport "github.com/synnaxlabs/synnax/pkg/distribution/transport/grpc/channel"
	framertransport "github.com/synnaxlabs/synnax/pkg/distribution/transport/grpc/framer"
	"github.com/synnaxlabs/synnax/pkg/security"
	"github.com/synnaxlabs/synnax/pkg/security/cert"
	"github.com/synnaxlabs/synnax/pkg/server"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/driver"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/synnax/pkg/version"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/override"
	xservice "github.com/synnaxlabs/x/service"
	xsignal "github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

type CoreConfig struct {
	alamos.Instrumentation
	insecure             *bool
	debug                *bool
	autoCert             *bool
	verifier             string
	memBacked            *bool
	listenAddress        address.Address
	peers                []address.Address
	dataPath             string
	slowConsumerTimeout  time.Duration
	rootUsername         string
	rootPassword         string
	noDriver             *bool
	taskOpTimeout        time.Duration
	taskPollInterval     time.Duration
	taskShutdownTimeout  time.Duration
	taskWorkerCount      uint8
	certFactoryConfig    cert.FactoryConfig
	enabledIntegrations  []string
	disabledIntegrations []string
	validateChannelNames *bool
}

var _ config.Config[CoreConfig] = CoreConfig{}

var DefaultCoreConfig = CoreConfig{
	certFactoryConfig: cert.DefaultFactoryConfig,
}

func (c CoreConfig) Validate() error {
	v := validate.New("core.config")
	validate.NotNil(v, "insecure", c.insecure)
	validate.NotNil(v, "debug", c.debug)
	validate.NotNil(v, "auto_cert", c.autoCert)
	validate.NotNil(v, "mem_backed", c.memBacked)
	validate.NotEmptyString(v, "listen_address", c.listenAddress)
	validate.NotEmptyString(v, "data_path", c.dataPath)
	validate.NonZero(v, "slow_consumer_timeout", c.slowConsumerTimeout)
	validate.NotEmptyString(v, "root_username", c.rootUsername)
	validate.NotEmptyString(v, "root_password", c.rootPassword)
	validate.NotNil(v, "no_driver", c.noDriver)
	validate.NonZero(v, "task_op_timeout", c.taskOpTimeout)
	validate.NonZero(v, "task_poll_interval", c.taskPollInterval)
	validate.NonZero(v, "task_shutdown_timeout", c.taskShutdownTimeout)
	validate.NonZero(v, "task_worker_count", c.taskWorkerCount)
	validate.NotNil(v, "validate_channel_names", c.validateChannelNames)
	v.Exec(c.certFactoryConfig.Validate)
	return v.Error()
}

func (c CoreConfig) Override(other CoreConfig) CoreConfig {
	return CoreConfig{
		Instrumentation:      override.Zero(c.Instrumentation, other.Instrumentation),
		insecure:             override.Nil(c.insecure, other.insecure),
		debug:                override.Nil(c.debug, other.debug),
		autoCert:             override.Nil(c.autoCert, other.autoCert),
		verifier:             override.String(c.verifier, other.verifier),
		memBacked:            override.Nil(c.memBacked, other.memBacked),
		listenAddress:        override.String(c.listenAddress, other.listenAddress),
		peers:                override.Slice(c.peers, other.peers),
		dataPath:             override.String(c.dataPath, other.dataPath),
		slowConsumerTimeout:  override.Numeric(c.slowConsumerTimeout, other.slowConsumerTimeout),
		rootUsername:         override.String(c.rootUsername, other.rootUsername),
		rootPassword:         override.String(c.rootPassword, other.rootPassword),
		noDriver:             override.Nil(c.noDriver, other.noDriver),
		taskOpTimeout:        override.Numeric(c.taskOpTimeout, other.taskOpTimeout),
		taskPollInterval:     override.Numeric(c.taskPollInterval, other.taskPollInterval),
		taskShutdownTimeout:  override.Numeric(c.taskShutdownTimeout, other.taskShutdownTimeout),
		taskWorkerCount:      override.Numeric(c.taskWorkerCount, other.taskWorkerCount),
		certFactoryConfig:    c.certFactoryConfig.Override(other.certFactoryConfig),
		enabledIntegrations:  override.Slice(c.enabledIntegrations, other.enabledIntegrations),
		disabledIntegrations: override.Slice(c.disabledIntegrations, other.disabledIntegrations),
		validateChannelNames: override.Nil(c.validateChannelNames, other.validateChannelNames),
	}
}

// BootupCore contains the most important Core startup logic. It does and should not
// read any variables from viper, and instead should be called with  fully configured
// CoreConfigs.
func BootupCore(ctx context.Context, onServerStarted chan struct{}, cfgs ...CoreConfig) error {
	cfg, err := config.New(DefaultCoreConfig, cfgs...)
	if err != nil {
		return err
	}

	if *cfg.autoCert {
		if err := cmdcert.GenerateAuto(cfg.certFactoryConfig); err != nil {
			return errors.Wrap(err, "failed to generate auto certs")
		}
	}

	vsn := version.Get()

	cfg.L.Zap().Sugar().Infof("\033[34mSynnax version %s starting\033[0m", vsn)
	cfg.L.Info(
		"starting synnax node",
		zap.String("version", vsn),
		zap.String("commit", version.Commit()),
		zap.Time("build", version.Time()),
	)

	// Any data stored on the node is considered sensitive, so we need to set the
	// permission mask for all files appropriately.
	disablePermissionBits()

	var (
		closer            xio.MultiCloser
		securityProvider  security.Provider
		storageLayer      *storage.Layer
		distributionLayer *distribution.Layer
		serviceLayer      *service.Layer
		apiLayer          *api.Layer
		rootServer        *server.Server
		embeddedDriver    *driver.Driver
	)
	cleanup, ok := xservice.NewOpener(ctx, &closer)
	defer func() {
		err = cleanup(err)
	}()

	if securityProvider, err = security.NewProvider(security.ProviderConfig{
		LoaderConfig: cfg.certFactoryConfig.LoaderConfig,
		Insecure:     cfg.insecure,
		KeySize:      cfg.certFactoryConfig.KeySize,
	}); !ok(err, nil) {
		return err
	}

	workDir, err := resolveWorkDir()
	if err != nil {
		return errors.Wrapf(err, "failed to resolve working directory")
	}
	cfg.L.Info("using working directory", zap.String("dir", workDir))

	if storageLayer, err = storage.Open(ctx, storage.Config{
		Instrumentation: cfg.Child("storage"),
		InMemory:        cfg.memBacked,
		Dirname:         cfg.dataPath,
	}); !ok(err, storageLayer) {
		return err
	}

	var (
		grpcClientPool         = configureClientGRPC(securityProvider, *cfg.insecure)
		aspenTransport         = aspentransport.New(grpcClientPool)
		frameTransport         = framertransport.New(grpcClientPool)
		channelTransport       = channeltransport.New(grpcClientPool)
		distributionTransports = []fgrpc.BindableTransport{
			aspenTransport,
			frameTransport,
			channelTransport,
		}
	)

	if distributionLayer, err = distribution.Open(ctx, distribution.Config{
		Instrumentation:      cfg.Child("distribution"),
		AdvertiseAddress:     cfg.listenAddress,
		PeerAddresses:        cfg.peers,
		AspenTransport:       aspenTransport,
		FrameTransport:       frameTransport,
		ChannelTransport:     channelTransport,
		Verifier:             cfg.verifier,
		Storage:              storageLayer,
		ValidateChannelNames: cfg.validateChannelNames,
	}); !ok(err, distributionLayer) {
		return err
	}

	if serviceLayer, err = service.Open(ctx, service.Config{
		Instrumentation: cfg.Child("service"),
		Distribution:    distributionLayer,
		Security:        securityProvider,
		Storage:         storageLayer,
	}); !ok(err, serviceLayer) {
		return err
	}

	if apiLayer, err = api.New(api.Config{
		Instrumentation: cfg.Child("api"),
		Service:         serviceLayer,
		Distribution:    distributionLayer,
	}); !ok(err, nil) {
		return err
	}
	creds := auth.InsecureCredentials{
		Username: cfg.rootUsername,
		Password: password.Raw(cfg.rootPassword),
	}
	if err = cmdauth.ProvisionRootUser(
		ctx,
		creds,
		distributionLayer,
		serviceLayer,
	); !ok(err, nil) {
		return err
	}

	// Configure the HTTP Layer AspenTransport.
	r := fhttp.NewRouter(fhttp.RouterConfig{
		Instrumentation:     cfg.Instrumentation,
		StreamWriteDeadline: cfg.slowConsumerTimeout,
	})
	apiLayer.BindTo(httpapi.New(r, api.NewHTTPCodecResolver(distributionLayer.Channel)))

	// Configure the GRPC Layer AspenTransport.
	grpcAPI, grpcAPITrans := grpcapi.New(distributionLayer.Channel)
	apiLayer.BindTo(grpcAPI)

	if rootServer, err = server.Serve(
		server.Config{
			Branches: []server.Branch{
				&server.SecureHTTPBranch{
					Transports: []fhttp.BindableTransport{r, serviceLayer.Console},
				},
				&server.GRPCBranch{Transports: slices.Concat(
					grpcAPITrans,
					distributionTransports,
				)},
				server.NewHTTPRedirectBranch(),
			},
			Debug:           cfg.debug,
			ListenAddress:   cfg.listenAddress,
			Instrumentation: cfg.Child("server"),
			Security: server.SecurityConfig{
				TLS:      securityProvider.TLS(),
				Insecure: cfg.insecure,
			},
		},
	); !ok(err, rootServer) {
		return err
	}

	// We run startup searching indexing after all services have been registered within
	// the ontology. We used to fork a new goroutine for every service at registration
	// time, but this caused a race condition where bleve would concurrently read and
	// write to a map. See
	// https://linear.app/synnax/issue/SY-1116/race-condition-on-server-startup for more
	// details on this issue.
	if stopSearchIndexing := runStartupSearchIndexing(
		ctx,
		distributionLayer,
	); !ok(nil, stopSearchIndexing) {
		return nil
	}

	if embeddedDriver, err = driver.OpenDriver(
		ctx,
		driver.Config{
			Enabled:             config.Bool(!*cfg.noDriver),
			Insecure:            cfg.insecure,
			Integrations:        parseIntegrations(cfg.enabledIntegrations, cfg.disabledIntegrations),
			Instrumentation:     cfg.Child("driver"),
			Address:             cfg.listenAddress,
			RackKey:             serviceLayer.Rack.EmbeddedKey,
			ClusterKey:          distributionLayer.Cluster.Key(),
			Username:            cfg.rootUsername,
			Password:            cfg.rootPassword,
			Debug:               cfg.debug,
			CACertPath:          cfg.certFactoryConfig.AbsoluteCACertPath(),
			ClientCertFile:      cfg.certFactoryConfig.AbsoluteNodeCertPath(),
			ClientKeyFile:       cfg.certFactoryConfig.AbsoluteNodeKeyPath(),
			ParentDirname:       workDir,
			TaskOpTimeout:       cfg.taskOpTimeout,
			TaskPollInterval:    cfg.taskPollInterval,
			TaskShutdownTimeout: cfg.taskShutdownTimeout,
			TaskWorkerCount:     cfg.taskWorkerCount,
		},
	); !ok(err, embeddedDriver) {
		return err
	}

	cfg.L.Infof(
		"\033[32mSynnax is running and available at %v \033[0m",
		cfg.listenAddress,
	)

	if onServerStarted != nil {
		onServerStarted <- struct{}{}
	}
	<-ctx.Done()
	return err
}

func resolveWorkDir() (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cacheDir, "synnax", "core", "workdir"), nil
}

func runStartupSearchIndexing(
	ctx context.Context,
	dist *distribution.Layer,
) io.Closer {
	// Run indexing inside an isolated signal context, so that if we receive an early
	// cancellation signal, we can ensure that we exit indexing before we close any
	// resources that it depends on (notably storage KV).
	searchIndexCtx, cancelIndexing := xsignal.WithCancel(ctx)
	searchIndexCtx.Go(
		dist.Ontology.InitializeSearchIndex,
		xsignal.WithKey("startup_search_indexing"),
	)
	return xsignal.NewHardShutdown(searchIndexCtx, cancelIndexing)
}

func parseIntegrations(enabled, disabled []string) []string {
	if len(enabled) > 0 {
		return enabled
	}
	return lo.Filter(driver.AllIntegrations, func(integration string, _ int) bool {
		return !lo.Contains(disabled, integration)
	})
}
