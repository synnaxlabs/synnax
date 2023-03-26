// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cmd

import (
	"context"
	"os"
	"os/signal"
	"time"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/security"
	"google.golang.org/grpc/credentials"
	insecureGRPC "google.golang.org/grpc/credentials/insecure"

	"github.com/cockroachdb/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/freighter/fhttp"
	"github.com/synnaxlabs/synnax/pkg/access"
	"github.com/synnaxlabs/synnax/pkg/api"
	httpapi "github.com/synnaxlabs/synnax/pkg/api/http"
	"github.com/synnaxlabs/synnax/pkg/auth"
	"github.com/synnaxlabs/synnax/pkg/auth/password"
	"github.com/synnaxlabs/synnax/pkg/auth/token"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/server"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/synnax/pkg/user"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/alamos"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	xsignal "github.com/synnaxlabs/x/signal"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Starts a Synnax Node",
	Long: `
Starts a Synnax Node using the data directory specified by the --data flag,
and listening on the address specified by the --listen flag. If --peers
is specified and no existing data is found, the node will attempt to join the cluster
formed by its peers.
	`,
	Example: `synnax start --listen [host:port] --data /mnt/ssd1 --peers [host:port],[host:port] --insecure`,
	Args:    cobra.NoArgs,
	Run:     func(cmd *cobra.Command, _ []string) { start(cmd) },
}

// start a Synnax node using the configuration specified by the command line flags,
// environment variables, and configuration files.
func start(cmd *cobra.Command) {
	var (
		insecure = viper.GetBool("insecure")
		verbose  = viper.GetBool("verbose")
	)

	logger, err := configureLogging(verbose)
	if err != nil {
		zap.S().Fatal(err)
	}

	interruptC := make(chan os.Signal, 1)
	signal.Notify(interruptC, os.Interrupt)

	// Any data stored on the node is considered sensitive, so we need to set the
	// permission mask for all files appropriately.
	disablePermissionBits()

	sCtx, cancel := xsignal.WithCancel(cmd.Context(), xsignal.WithLogger(logger))
	defer cancel()

	// Perform the rest of the startup within a separate goroutine so we can properly
	// handle signal interrupts.
	sCtx.Go(func(ctx context.Context) error {
		// Set up the tracing backend.
		exp := configureObservability(verbose)

		secProvider, err := configureSecurity(logger, insecure)
		if err != nil {
			return err
		}

		// An array to hold the grpcTransports we use for cluster internal communication.
		grpcTransports := &[]fgrpc.BindableTransport{}

		grpcPool := configureClientGRPC(secProvider, insecure)

		// Open the distribution layer.
		storageCfg := buildStorageConfig(exp, logger)
		distConfig, err := buildDistributionConfig(
			grpcPool,
			exp,
			logger,
			storageCfg,
			grpcTransports,
		)
		dist, err := distribution.Open(ctx, distConfig)
		if err != nil {
			return err
		}
		defer func() { err = dist.Close() }()

		// Set up our high level services.
		gorpDB := dist.Storage.Gorpify()
		userSvc := &user.Service{DB: gorpDB, Ontology: dist.Ontology}
		tokenSvc := &token.Service{KeyProvider: secProvider, Expiration: 24 * time.Hour}
		authenticator := &auth.KV{DB: gorpDB}

		// Provision the root user.
		if err := maybeProvisionRootUser(gorpDB, authenticator, userSvc); err != nil {
			return err
		}

		// Configure the API core.
		_api := api.New(api.Config{
			Logger:        logger,
			Channel:       dist.Channel,
			Framer:        dist.Framer,
			Storage:       dist.Storage,
			User:          userSvc,
			Token:         tokenSvc,
			Authenticator: authenticator,
			Enforcer:      access.AllowAll{},
			Insecure:      insecure,
			Cluster:       dist.Cluster,
			Ontology:      dist.Ontology,
		})

		// Configure the HTTP API Transport.
		r := fhttp.NewRouter(fhttp.RouterConfig{Logger: logger.Sugar()})
		_api.BindTo(httpapi.New(r))

		srv, err := server.New(buildServerConfig(
			*grpcTransports,
			[]fhttp.BindableTransport{r},
			secProvider,
			logger,
		))
		if err != nil {
			return err
		}
		sCtx.Go(func(_ context.Context) error {
			return srv.Serve()
		}, xsignal.WithKey("server"))
		defer srv.Stop()
		<-ctx.Done()
		return nil
	}, xsignal.WithKey("start"))

	select {
	case <-interruptC:
		logger.Info("received interrupt signal, shutting down")
		cancel()
	case <-sCtx.Stopped():
	}

	if err := sCtx.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		logger.Fatal("shutdown failed", zap.Error(err))
	}
	logger.Info("shutdown successful")
}

func init() {
	rootCmd.AddCommand(startCmd)
	configureStartFlags()
	bindFlags(startCmd)
}

func buildStorageConfig(
	exp alamos.Experiment,
	logger *zap.Logger,
) storage.Config {
	return storage.Config{
		MemBacked:  config.BoolPointer(viper.GetBool("mem")),
		Dirname:    viper.GetString("data"),
		Logger:     logger.Named("storage"),
		Experiment: exp,
	}
}

func parsePeerAddresses() ([]address.Address, error) {
	peerStrings := viper.GetStringSlice("peer-addresses")
	peerAddresses := make([]address.Address, len(peerStrings))
	for i, listenString := range peerStrings {
		peerAddresses[i] = address.Address(listenString)
	}
	return peerAddresses, nil
}

func buildDistributionConfig(
	pool *fgrpc.Pool,
	exp alamos.Experiment,
	logger *zap.Logger,
	storage storage.Config,
	transports *[]fgrpc.BindableTransport,
) (distribution.Config, error) {
	peers, err := parsePeerAddresses()
	return distribution.Config{
		Logger:           logger.Named("distrib"),
		Experiment:       exp,
		AdvertiseAddress: address.Address(viper.GetString("listen")),
		PeerAddresses:    peers,
		Pool:             pool,
		Storage:          storage,
		Transports:       transports,
	}, err
}

func buildServerConfig(
	grpcTransports []fgrpc.BindableTransport,
	httpTransports []fhttp.BindableTransport,
	sec security.Provider,
	logger *zap.Logger,
) (cfg server.Config) {
	cfg.Branches = append(cfg.Branches,
		&server.SecureHTTPBranch{Transports: httpTransports},
		&server.GRPCBranch{Transports: grpcTransports},
		server.NewHTTPRedirectBranch(),
	)
	cfg.ListenAddress = address.Address(viper.GetString("listen"))
	cfg.Logger = logger.Named("server")
	cfg.Security.TLS = sec.TLS()
	cfg.Security.Insecure = config.BoolPointer(viper.GetBool("insecure"))
	return cfg
}

func configureLogging(verbose bool) (*zap.Logger, error) {
	var cfg zap.Config
	if verbose {
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		cfg.Encoding = "console"
	} else {
		cfg = zap.NewProductionConfig()
		cfg.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	}
	return cfg.Build()
}

var rootExperimentKey = "experiment"

func configureObservability(verbose bool) alamos.Experiment {
	var opt alamos.Option
	if verbose {
		opt = alamos.WithFilters(alamos.LevelFilterAll{})
	} else {
		opt = alamos.WithFilters(alamos.LevelFilterThreshold{Level: alamos.Production})
	}
	return alamos.New(rootExperimentKey, opt)
}

func configureSecurity(logger *zap.Logger, insecure bool) (security.Provider, error) {
	return security.NewProvider(security.ProviderConfig{
		LoaderConfig: buildCertLoaderConfig(logger),
		Insecure:     config.BoolPointer(insecure),
		KeySize:      viper.GetInt("key-size"),
	})
}

func maybeProvisionRootUser(
	db *gorp.DB,
	authSvc auth.Authenticator,
	userSvc *user.Service,
) error {
	uname := viper.GetString("username")
	pass := password.Raw(viper.GetString("password"))
	exists, err := userSvc.UsernameExists(uname)
	if err != nil || exists {
		return err
	}
	txn := db.BeginTxn()
	if err = authSvc.NewWriterUsingTxn(txn).Register(auth.InsecureCredentials{
		Username: uname,
		Password: pass,
	}); err != nil {
		return err
	}
	if err = userSvc.NewWriterUsingTxn(txn).Create(&user.User{Username: uname}); err != nil {
		return err
	}
	return txn.Commit()
}

func configureClientGRPC(
	sec security.Provider,
	insecure bool,
) *fgrpc.Pool {
	return fgrpc.NewPool(
		grpc.WithTransportCredentials(getClientGRPCTransportCredentials(sec, insecure)),
	)
}

func getClientGRPCTransportCredentials(sec security.Provider, insecure bool) credentials.TransportCredentials {
	return lo.Ternary(insecure, insecureGRPC.NewCredentials(), credentials.NewTLS(sec.TLS()))
}
