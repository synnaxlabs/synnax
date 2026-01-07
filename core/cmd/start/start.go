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

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	aspentransport "github.com/synnaxlabs/aspen/transport/grpc"
	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/freighter/fhttp"
	"github.com/synnaxlabs/synnax/cmd/cert"
	"github.com/synnaxlabs/synnax/cmd/instrumentation"
	cmdauth "github.com/synnaxlabs/synnax/cmd/start/auth"
	"github.com/synnaxlabs/synnax/pkg/api"
	grpcapi "github.com/synnaxlabs/synnax/pkg/api/grpc"
	httpapi "github.com/synnaxlabs/synnax/pkg/api/http"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	channeltransport "github.com/synnaxlabs/synnax/pkg/distribution/transport/grpc/channel"
	framertransport "github.com/synnaxlabs/synnax/pkg/distribution/transport/grpc/framer"
	"github.com/synnaxlabs/synnax/pkg/security"
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
	xservice "github.com/synnaxlabs/x/service"
	xsignal "github.com/synnaxlabs/x/signal"
	"go.uber.org/zap"
)

const stopKeyWord = "stop"

func scanForStopKeyword(interruptC chan os.Signal) {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		if scanner.Text() == stopKeyWord {
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

	sCtx.Go(StartServer, xsignal.WithKey("start"), xsignal.RecoverWithErrOnPanic())

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

// startServer contains the most important Core startup logic. It reads configuration
// from viper and starts all server components.
func StartServer(ctx context.Context) error {
	var (
		vers                = version.Get()
		verifierFlag        = base64.MustDecode("bGljZW5zZS1rZXk=")
		insecure            = viper.GetBool(FlagInsecure)
		debug               = viper.GetBool(instrumentation.FlagDebug)
		autoCert            = viper.GetBool(FlagAutoCert)
		verifier            = viper.GetString(string(verifierFlag))
		memBacked           = viper.GetBool(FlagMem)
		listenAddress       = address.Address(viper.GetString(FlagListen))
		dataPath            = viper.GetString(FlagData)
		slowConsumerTimeout = viper.GetDuration(FlagSlowConsumerTimeout)
		rootUsername        = viper.GetString(FlagUsername)
		rootPassword        = viper.GetString(FlagPassword)
		noDriver            = viper.GetBool(FlagNoDriver)
		keySize             = viper.GetInt(cert.FlagKeySize)
		taskOpTimeout       = viper.GetDuration(FlagTaskOpTimeout)
		taskPollInterval    = viper.GetDuration(FlagTaskPollInterval)
		taskShutdownTimeout = viper.GetDuration(FlagTaskShutdownTimeout)
		taskWorkerCount     = viper.GetUint8(FlagTaskWorkerCount)
		ins                 = instrumentation.Configure()
	)
	defer instrumentation.Cleanup(ctx, ins)

	if autoCert {
		if err := cert.GenerateAuto(ins, listenAddress); err != nil {
			return errors.Wrap(err, "failed to generate auto certs")
		}
	}

	ins.L.Zap().Sugar().Infof("\033[34mSynnax version %s starting\033[0m", vers)
	ins.L.Info("starting synnax node", zap.String("version", vers), zap.String("commit", version.Commit()), zap.Time("build", version.Time()))

	// Any data stored on the node is considered sensitive, so we need to set the
	// permission mask for all files appropriately.
	disablePermissionBits()

	var (
		err               error
		closer            xio.MultiCloser
		peers             = parsePeerAddressFlag()
		securityProvider  security.Provider
		storageLayer      *storage.Layer
		distributionLayer *distribution.Layer
		serviceLayer      *service.Layer
		apiLayer          *api.Layer
		rootServer        *server.Server
		embeddedDriver    *driver.Driver
		certLoaderConfig  = cert.BuildLoaderConfig(ins)
	)
	cleanup, ok := xservice.NewOpener(ctx, &closer)
	defer func() {
		err = cleanup(err)
	}()

	if securityProvider, err = security.NewProvider(security.ProviderConfig{
		LoaderConfig: certLoaderConfig,
		Insecure:     config.Bool(insecure),
		KeySize:      keySize,
	}); !ok(err, nil) {
		return err
	}

	workDir, err := resolveWorkDir()
	if err != nil {
		return errors.Wrapf(err, "failed to resolve working directory")
	}
	ins.L.Info("using working directory", zap.String("dir", workDir))

	if storageLayer, err = storage.Open(ctx, storage.Config{
		Instrumentation: ins.Child("storage"),
		InMemory:        config.Bool(memBacked),
		Dirname:         dataPath,
	}); !ok(err, storageLayer) {
		return err
	}

	var (
		grpcClientPool         = configureClientGRPC(securityProvider, insecure)
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
		Instrumentation:  ins.Child("distribution"),
		AdvertiseAddress: listenAddress,
		PeerAddresses:    peers,
		AspenTransport:   aspenTransport,
		FrameTransport:   frameTransport,
		ChannelTransport: channelTransport,
		Verifier:         verifier,
		Storage:          storageLayer,
	}); !ok(err, distributionLayer) {
		return err
	}

	if serviceLayer, err = service.Open(ctx, service.Config{
		Instrumentation: ins.Child("service"),
		Distribution:    distributionLayer,
		Security:        securityProvider,
	}); !ok(err, serviceLayer) {
		return err
	}

	if apiLayer, err = api.New(api.Config{
		Instrumentation: ins.Child("api"),
		Service:         serviceLayer,
		Distribution:    distributionLayer,
	}); !ok(err, nil) {
		return err
	}
	creds := auth.InsecureCredentials{
		Username: viper.GetString(FlagUsername),
		Password: password.Raw(viper.GetString(FlagPassword)),
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
		Instrumentation:     ins,
		StreamWriteDeadline: slowConsumerTimeout,
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
			Debug:           config.Bool(debug),
			ListenAddress:   listenAddress,
			Instrumentation: ins.Child("server"),
			Security: server.SecurityConfig{
				TLS:      securityProvider.TLS(),
				Insecure: config.Bool(insecure),
			},
		},
	); !ok(err, rootServer) {
		return err
	}

	// We run startup searching indexing after all services have been
	// registered within the ontology. We used to fork a new goroutine for
	// every service at registration time, but this caused a race condition
	// where bleve would concurrently read and write to a map.
	// See https://linear.app/synnax/issue/SY-1116/race-condition-on-server-startup
	// for more details on this issue.
	if stopSearchIndexing := runStartupSearchIndexing(
		ctx,
		distributionLayer,
	); !ok(nil, stopSearchIndexing) {
		return nil
	}

	if embeddedDriver, err = driver.OpenDriver(
		ctx,
		driver.Config{
			Enabled:             config.Bool(!noDriver),
			Insecure:            config.Bool(insecure),
			Integrations:        parseIntegrationsFlag(),
			Instrumentation:     ins.Child("driver"),
			Address:             listenAddress,
			RackKey:             serviceLayer.Rack.EmbeddedKey,
			ClusterKey:          distributionLayer.Cluster.Key(),
			Username:            rootUsername,
			Password:            rootPassword,
			Debug:               config.Bool(debug),
			CACertPath:          certLoaderConfig.AbsoluteCACertPath(),
			ClientCertFile:      certLoaderConfig.AbsoluteNodeCertPath(),
			ClientKeyFile:       certLoaderConfig.AbsoluteNodeKeyPath(),
			ParentDirname:       workDir,
			TaskOpTimeout:       taskOpTimeout,
			TaskPollInterval:    taskPollInterval,
			TaskShutdownTimeout: taskShutdownTimeout,
			TaskWorkerCount:     taskWorkerCount,
		},
	); !ok(err, embeddedDriver) {
		return err
	}

	ins.L.Info(fmt.Sprintf("\033[32mSynnax is running and available at %v \033[0m", listenAddress))

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
	// Run indexing inside an isolated signal context, so that if
	// we receive an early cancellation signal, we can ensure that
	// we exit indexing before we close any resources that it depends
	// on (notably storage KV).
	searchIndexCtx, cancelIndexing := xsignal.WithCancel(ctx)
	searchIndexCtx.Go(
		dist.Ontology.InitializeSearchIndex,
		xsignal.WithKey("startup_search_indexing"),
	)
	return xsignal.NewHardShutdown(searchIndexCtx, cancelIndexing)
}
