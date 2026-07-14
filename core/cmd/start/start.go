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
	"strconv"
	"time"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	aspentransport "github.com/synnaxlabs/aspen/transport/grpc"
	"github.com/synnaxlabs/freighter/http"
	cmdcert "github.com/synnaxlabs/synnax/cmd/cert"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/console"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	disttransport "github.com/synnaxlabs/synnax/pkg/distribution/transport/grpc"
	"github.com/synnaxlabs/synnax/pkg/driver"
	"github.com/synnaxlabs/synnax/pkg/security"
	"github.com/synnaxlabs/synnax/pkg/security/cert"
	"github.com/synnaxlabs/synnax/pkg/security/cert/auto"
	"github.com/synnaxlabs/synnax/pkg/security/cert/file"
	"github.com/synnaxlabs/synnax/pkg/security/cert/tailscale"
	"github.com/synnaxlabs/synnax/pkg/server"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/synnax/pkg/transport"
	"github.com/synnaxlabs/synnax/pkg/version"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/override"
	xservice "github.com/synnaxlabs/x/service"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

type CoreConfig struct {
	insecure             *bool
	debug                *bool
	autoCert             *bool
	validateChannelNames *bool
	memBacked            *bool
	noDriver             *bool
	alamos.Instrumentation
	dataPath             string
	verifier             string
	rootCredentials      auth.Credentials
	listeners            []listenerSpec
	peers                []address.Address
	disabledIntegrations []string
	enabledIntegrations  []string
	certFactoryConfig    cert.FactoryConfig
	taskShutdownTimeout  time.Duration
	taskPollInterval     time.Duration
	taskOpTimeout        time.Duration
	slowConsumerTimeout  time.Duration
	taskWorkerCount      uint8
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
	validateListeners(v, c.listeners)
	validate.NotEmptyString(v, "data_path", c.dataPath)
	validate.NonZero(v, "slow_consumer_timeout", c.slowConsumerTimeout)
	validate.NotNil(v, "no_driver", c.noDriver)
	v.Exec(c.rootCredentials.Validate)
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
		listeners:            override.Slice(c.listeners, other.listeners),
		peers:                override.Slice(c.peers, other.peers),
		dataPath:             override.String(c.dataPath, other.dataPath),
		slowConsumerTimeout:  override.Numeric(c.slowConsumerTimeout, other.slowConsumerTimeout),
		rootCredentials:      override.Zero(c.rootCredentials, other.rootCredentials),
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

// advertiseAddress returns the address peers use to reach this node. It is the sole
// listener marked advertise, or the first listener when none is marked.
func (c CoreConfig) advertiseAddress() address.Address {
	for _, l := range c.listeners {
		if l.advertise {
			return l.address
		}
	}
	return c.listeners[0].address
}

// sourceFactories is the set of certificate sources a listener may select. cmd/start
// owns this list so implementation-specific sources (tailscale) never enter the base
// cert package.
var sourceFactories = []cert.SourceFactory{
	file.Factory{},
	auto.Factory{},
	tailscale.Factory{},
}

// buildServerListeners resolves each listener spec into a server.Listener, backing every
// secure listener with a TLS config from its certificate source.
func (c CoreConfig) buildServerListeners(
	p security.Provider,
	factories []cert.SourceFactory,
) ([]server.Listener, error) {
	out := make([]server.Listener, len(c.listeners))
	for i, spec := range c.listeners {
		l := server.Listener{Address: spec.address}
		if !*c.insecure {
			src, err := cert.Resolve(factories, spec.sourceConfig(c.certFactoryConfig))
			if err != nil {
				return nil, err
			}
			l.TLS = p.TLSConfigFor(src)
		}
		out[i] = l
	}
	return out, nil
}

// BootupCore contains the most important Core startup logic. It does and should not
// read any variables from viper, and instead should be called with  fully configured
// CoreConfigs.
func BootupCore(ctx context.Context, onServerStarted chan struct{}, cfgs ...CoreConfig) (err error) {
	cfg, err := config.New(DefaultCoreConfig, cfgs...)
	if err != nil {
		return err
	}

	if *cfg.autoCert {
		if err = cmdcert.GenerateAuto(cfg.certFactoryConfig); err != nil {
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
		transportLayer    transport.Layer
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

	workDir, closeWorkDir, err := openWorkDir()
	if !ok(err, closeWorkDir) {
		return errors.Wrapf(err, "failed to resolve working directory")
	}
	cfg.L.Info("using working directory", zap.String("dir", workDir))

	if storageLayer, err = storage.OpenLayer(ctx, storage.LayerConfig{
		Instrumentation: cfg.Child("storage"),
		InMemory:        cfg.memBacked,
		Dirname:         cfg.dataPath,
	}); !ok(err, storageLayer) {
		return err
	}

	grpcClientPool := configureClientGRPC(securityProvider, *cfg.insecure)
	// Register the pool first so it closes LAST: every transport that
	// references it must finish shutting down before its connections are
	// torn down.
	if !ok(nil, grpcClientPool) {
		return ctx.Err()
	}
	var (
		aspenTransport = aspentransport.New(grpcClientPool)
		distTransport  = disttransport.New(grpcClientPool)
	)

	if distributionLayer, err = distribution.OpenLayer(ctx, distribution.LayerConfig{
		Instrumentation:      cfg.Child("distribution"),
		AdvertiseAddress:     cfg.advertiseAddress(),
		PeerAddresses:        cfg.peers,
		AspenTransport:       aspenTransport,
		Transport:            distTransport,
		Verifier:             cfg.verifier,
		Storage:              storageLayer,
		ValidateChannelNames: cfg.validateChannelNames,
	}); !ok(err, distributionLayer) {
		return err
	}

	if serviceLayer, err = service.OpenLayer(ctx, service.LayerConfig{
		Instrumentation: cfg.Child("service"),
		Distribution:    distributionLayer,
		Security:        securityProvider,
		Storage:         storageLayer,
		RootCredentials: cfg.rootCredentials,
	}); !ok(err, serviceLayer) {
		return err
	}

	apiCfg := api.LayerConfig{
		Instrumentation: cfg.Child("api"),
		Service:         serviceLayer,
		Distribution:    distributionLayer,
	}
	if apiLayer, err = api.NewLayer(apiCfg); !ok(err, nil) {
		return err
	}

	// Configure the HTTP Layer AspenTransport.
	var r *http.Router
	if r, err = http.NewRouter(http.RouterConfig{
		Instrumentation:     cfg.Instrumentation,
		StreamWriteDeadline: cfg.slowConsumerTimeout,
	}); !ok(err, nil) {
		return err
	}
	if transportLayer, err = transport.NewLayer(transport.LayerConfig{
		Instrumentation: cfg.Child("transport"),
		API:             apiLayer,
		Router:          r,
		Channel:         distributionLayer.Channel,
	}); !ok(err, nil) {
		return err
	}

	var embeddedConsole *console.Console
	if embeddedConsole, err = console.New(); !ok(err, nil) {
		return err
	}

	serverListeners, err := cfg.buildServerListeners(securityProvider, sourceFactories)
	if !ok(err, nil) {
		return err
	}

	if rootServer, err = server.Serve(
		server.Config{
			Branches: []server.Branch{
				&server.SecureHTTPBranch{
					Transports: []http.BindableTransport{r, embeddedConsole},
				},
				&server.GRPCBranch{Transports: append(
					transportLayer.GRPC,
					aspenTransport,
					distTransport,
				)},
				server.NewHTTPRedirectBranch(),
			},
			Debug:           cfg.debug,
			Listeners:       serverListeners,
			Instrumentation: cfg.Child("server"),
			Security:        server.SecurityConfig{Insecure: cfg.insecure},
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

	if embeddedDriver, err = driver.Open(
		ctx,
		driver.Config{
			Enabled:             new(!*cfg.noDriver),
			Insecure:            cfg.insecure,
			Integrations:        parseIntegrations(cfg.enabledIntegrations, cfg.disabledIntegrations),
			Instrumentation:     cfg.Child("driver"),
			Address:             cfg.advertiseAddress(),
			RackKey:             serviceLayer.Rack.EmbeddedKey,
			ClusterKey:          distributionLayer.Cluster.Key(),
			Credentials:         cfg.rootCredentials,
			Debug:               cfg.debug,
			CACertPath:          cfg.certFactoryConfig.AbsoluteNodeCertPath(),
			ClientCertFile:      cfg.certFactoryConfig.AbsoluteCACertPath(),
			ClientKeyFile:       cfg.certFactoryConfig.AbsoluteCAKeyPath(),
			ParentDirname:       workDir,
			TaskWorkerCount:     cfg.taskWorkerCount,
			TaskShutdownTimeout: cfg.taskShutdownTimeout,
			TaskPollInterval:    cfg.taskPollInterval,
			TaskOpTimeout:       cfg.taskOpTimeout,
		},
	); !ok(err, embeddedDriver) {
		return err
	}

	cfg.L.Infof(
		"\033[32mSynnax is running and available at %v \033[0m",
		cfg.advertiseAddress(),
	)

	if onServerStarted != nil {
		onServerStarted <- struct{}{}
	}
	<-ctx.Done()
	return err
}

func openWorkDir() (string, io.Closer, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", nil, err
	}
	dir := filepath.Join(
		cacheDir,
		"synnax",
		"core",
		"workdir",
		strconv.Itoa(os.Getpid()),
	)
	if err = os.MkdirAll(dir, fs.UserRWX); err != nil {
		return "", nil, err
	}
	return dir, xio.CloserFunc(func() error { return os.RemoveAll(dir) }), nil
}

func runStartupSearchIndexing(
	ctx context.Context,
	dist *distribution.Layer,
) io.Closer {
	// Run indexing inside an isolated signal context, so that if we receive an early
	// cancellation signal, we can ensure that we exit indexing before we close any
	// resources that it depends on (notably storage KV).
	searchIndexCtx, cancelIndexing := signal.WithCancel(ctx)
	searchIndexCtx.Go(
		dist.Search.Initialize,
		signal.WithKey("startup_search_indexing"),
	)
	return signal.NewHardShutdown(searchIndexCtx, cancelIndexing)
}

func parseIntegrations(enabled, disabled []string) []string {
	if len(enabled) > 0 {
		return enabled
	}
	return lo.Filter(driver.AllIntegrations, func(integration string, _ int) bool {
		return !lo.Contains(disabled, integration)
	})
}
