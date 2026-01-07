// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package start

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/synnaxlabs/alamos"
	aspentransport "github.com/synnaxlabs/aspen/transport/grpc"
	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/freighter/fhttp"
	cmdcert "github.com/synnaxlabs/synnax/cmd/cert"
	"github.com/synnaxlabs/synnax/cmd/instrumentation"
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
	"github.com/synnaxlabs/x/encoding/base64"
	"github.com/synnaxlabs/x/errors"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/override"
	xservice "github.com/synnaxlabs/x/service"
	xsignal "github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

func scanForStopKeyword(interruptC chan os.Signal) {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		if scanner.Text() == "stop" {
			interruptC <- os.Interrupt
		}
	}
}

// start is the entrypoint for starting a Synnax Core. It handles signal interrupts and
// delegates to startServer for the actual startup.
func start(cmd *cobra.Command) {
	ctx := cmd.Context()
	ins := instrumentation.Configure()
	defer instrumentation.Cleanup(ctx, ins)

	interruptC := make(chan os.Signal, 1)
	signal.Notify(interruptC, os.Interrupt)

	sCtx, cancel := xsignal.WithCancel(ctx, xsignal.WithInstrumentation(ins))
	defer cancel()

	// Listen for a custom stop keyword that can be used in place of a Ctrl+C signal.
	// It's fine to let this get garbage collected.
	go scanForStopKeyword(interruptC)

	sCtx.Go(func(ctx context.Context) error {
		return BootupCore(ctx, GetCoreConfigFromViper(ins))
	}, xsignal.WithKey("start"), xsignal.RecoverWithErrOnPanic())

	select {
	case <-interruptC:
		ins.L.Info(
			"\033[33mSynnax is shutting down. This can take up to 5 seconds. Please be patient\033[0m",
		)
		cancel()
	case <-sCtx.Stopped():
	}

	if err := sCtx.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		ins.L.Zap().Sugar().Errorf(
			"\033[31mSynnax has encountered an error and is shutting down: %v\033[0m",
			err,
		)
		ins.L.Fatal("synnax failed", zap.Error(err))
	}
	ins.L.Info("\033[34mSynnax has shut down\033[0m")
}

func GetCoreConfigFromViper(ins alamos.Instrumentation) CoreConfig {
	listenAddress := address.Address(viper.GetString(FlagListen))
	return CoreConfig{
		Instrumentation:     ins,
		insecure:            config.Bool(viper.GetBool(FlagInsecure)),
		debug:               config.Bool(viper.GetBool(instrumentation.FlagDebug)),
		autoCert:            config.Bool(viper.GetBool(FlagAutoCert)),
		verifier:            viper.GetString(string(base64.MustDecode("bGljZW5zZS1rZXk="))),
		memBacked:           config.Bool(viper.GetBool(FlagMem)),
		listenAddress:       listenAddress,
		peers:               parsePeerAddressFlag(),
		dataPath:            viper.GetString(FlagData),
		slowConsumerTimeout: viper.GetDuration(FlagSlowConsumerTimeout),
		rootUsername:        viper.GetString(FlagUsername),
		rootPassword:        viper.GetString(FlagPassword),
		noDriver:            config.Bool(viper.GetBool(FlagNoDriver)),
		keySize:             viper.GetInt(cmdcert.FlagKeySize),
		taskOpTimeout:       viper.GetDuration(FlagTaskOpTimeout),
		taskPollInterval:    viper.GetDuration(FlagTaskPollInterval),
		taskShutdownTimeout: viper.GetDuration(FlagTaskShutdownTimeout),
		taskWorkerCount:     viper.GetUint8(FlagTaskWorkerCount),
		certFactoryConfig:   cmdcert.BuildCertFactoryConfig(ins, listenAddress),
		certLoaderConfig:    cmdcert.BuildLoaderConfig(ins),
		integrations:        parseIntegrationsFlag(),
	}
}

type CoreConfig struct {
	alamos.Instrumentation
	insecure            *bool
	debug               *bool
	autoCert            *bool
	verifier            string
	memBacked           *bool
	listenAddress       address.Address
	peers               []address.Address
	dataPath            string
	slowConsumerTimeout time.Duration
	rootUsername        string
	rootPassword        string
	noDriver            *bool
	keySize             int
	taskOpTimeout       time.Duration
	taskPollInterval    time.Duration
	taskShutdownTimeout time.Duration
	taskWorkerCount     uint8
	certFactoryConfig   cert.FactoryConfig
	certLoaderConfig    cert.LoaderConfig
	integrations        []string
}

var _ config.Config[CoreConfig] = CoreConfig{}

func (c CoreConfig) Validate() error {
	v := validate.New("core.config")
	validate.NotNil(v, "insecure", c.insecure)
	validate.NotNil(v, "debug", c.debug)
	validate.NotNil(v, "auto_cert", c.autoCert)
	validate.NotNil(v, "mem_backed", c.memBacked)
	validate.NotEmptyString(v, "listen_address", c.listenAddress)
	validate.NotEmptyString(v, "data_path", c.dataPath)
	validate.NotEmptyString(v, "root_username", c.rootUsername)
	validate.NotEmptyString(v, "root_password", c.rootPassword)
	validate.NotNil(v, "no_driver", c.noDriver)
	return v.Error()
}

func (c CoreConfig) Override(other CoreConfig) CoreConfig {
	return CoreConfig{
		Instrumentation:     override.Zero(c.Instrumentation, other.Instrumentation),
		insecure:            override.Nil(c.insecure, other.insecure),
		debug:               override.Nil(c.debug, other.debug),
		autoCert:            override.Nil(c.autoCert, other.autoCert),
		verifier:            override.String(c.verifier, other.verifier),
		memBacked:           override.Nil(c.memBacked, other.memBacked),
		listenAddress:       override.String(c.listenAddress, other.listenAddress),
		peers:               override.Slice(c.peers, other.peers),
		dataPath:            override.String(c.dataPath, other.dataPath),
		slowConsumerTimeout: override.Numeric(c.slowConsumerTimeout, other.slowConsumerTimeout),
		rootUsername:        override.String(c.rootUsername, other.rootUsername),
		rootPassword:        override.String(c.rootPassword, other.rootPassword),
		noDriver:            override.Nil(c.noDriver, other.noDriver),
		keySize:             override.Numeric(c.keySize, other.keySize),
		taskOpTimeout:       override.Numeric(c.taskOpTimeout, other.taskOpTimeout),
		taskPollInterval:    override.Numeric(c.taskPollInterval, other.taskPollInterval),
		taskShutdownTimeout: override.Numeric(c.taskShutdownTimeout, other.taskShutdownTimeout),
		taskWorkerCount:     override.Numeric(c.taskWorkerCount, other.taskWorkerCount),
		certFactoryConfig:   c.certFactoryConfig.Override(other.certFactoryConfig),
		certLoaderConfig:    c.certLoaderConfig.Override(other.certLoaderConfig),
		integrations:        override.Slice(c.integrations, other.integrations),
	}

}

// BootupCore contains the most important Core startup logic. It does and should not
// read any variables from viper, and instead should be called with a fully configured
// CoreConfig.
func BootupCore(ctx context.Context, cfgs ...CoreConfig) error {
	cfg, err := config.New(CoreConfig{}, cfgs...)
	if err != nil {
		return err
	}
	defer instrumentation.Cleanup(ctx, cfg.Instrumentation)

	if *cfg.autoCert {
		if err := cmdcert.GenerateAuto(cfg.certFactoryConfig); err != nil {
			return errors.Wrap(err, "failed to generate auto certs")
		}
	}

	vsn := version.Get()
	cfg.L.Zap().Sugar().Infof("\033[34mSynnax version %s starting\033[0m", vsn)
	cfg.L.Info("starting synnax node", zap.String("version", vsn), zap.String("commit", version.Commit()), zap.Time("build", version.Time()))

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
		LoaderConfig: cfg.certLoaderConfig,
		Insecure:     cfg.insecure,
		KeySize:      cfg.keySize,
	}); !ok(err, nil) {
		return err
	}

	workDir, err := resolveWorkDir()
	if err != nil {
		return errors.Wrapf(err, "failed to resolve working directory")
	}
	cfg.L.Info("using working directory", zap.String("dir", workDir))

	if storageLayer, err = storage.Open(ctx, storage.Config{
		Instrumentation: cfg.Instrumentation.Child("storage"),
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
		Instrumentation:  cfg.Instrumentation.Child("distribution"),
		AdvertiseAddress: cfg.listenAddress,
		PeerAddresses:    cfg.peers,
		AspenTransport:   aspenTransport,
		FrameTransport:   frameTransport,
		ChannelTransport: channelTransport,
		Verifier:         cfg.verifier,
		Storage:          storageLayer,
	}); !ok(err, distributionLayer) {
		return err
	}

	if serviceLayer, err = service.Open(ctx, service.Config{
		Instrumentation: cfg.Instrumentation.Child("service"),
		Distribution:    distributionLayer,
		Security:        securityProvider,
	}); !ok(err, serviceLayer) {
		return err
	}

	if apiLayer, err = api.New(api.Config{
		Instrumentation: cfg.Instrumentation.Child("api"),
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
				&server.SecureHTTPBranch{Transports: []fhttp.BindableTransport{r, serviceLayer.Console}},
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
			Integrations:        cfg.integrations,
			Instrumentation:     cfg.Instrumentation.Child("driver"),
			Address:             cfg.listenAddress,
			RackKey:             serviceLayer.Rack.EmbeddedKey,
			ClusterKey:          distributionLayer.Cluster.Key(),
			Username:            cfg.rootUsername,
			Password:            cfg.rootPassword,
			Debug:               cfg.debug,
			CACertPath:          cfg.certLoaderConfig.AbsoluteCACertPath(),
			ClientCertFile:      cfg.certLoaderConfig.AbsoluteNodeCertPath(),
			ClientKeyFile:       cfg.certLoaderConfig.AbsoluteNodeKeyPath(),
			ParentDirname:       workDir,
			TaskOpTimeout:       cfg.taskOpTimeout,
			TaskPollInterval:    cfg.taskPollInterval,
			TaskShutdownTimeout: cfg.taskShutdownTimeout,
			TaskWorkerCount:     cfg.taskWorkerCount,
		},
	); !ok(err, embeddedDriver) {
		return err
	}

	cfg.L.Info(fmt.Sprintf("\033[32mSynnax is running and available at %v \033[0m", cfg.listenAddress))

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
