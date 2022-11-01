package cmd

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"os"
	"os/signal"
	"time"

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
	"google.golang.org/grpc/credentials/insecure"
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "",
	Long:  ``,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger, err := configureLogging()
		if err != nil {
			return err
		}

		sigC := make(chan os.Signal, 1)
		signal.Notify(sigC, os.Interrupt)

		// any data store on the node is considered sensitive, so we need to set the
		// permission mask for all files appropriately.
		disablePermissionBits()

		sCtx, cancel := xsignal.WithCancel(cmd.Context(), xsignal.WithLogger(logger))
		defer cancel()

		// Perform the rest of the startup within a separate goroutine
		// we can properly handle signal interrupts.
		sCtx.Go(func(ctx context.Context) (err error) {
			// SetState up the tracing backend.
			exp := configureObservability()

			// An array to hold the transports we use for cluster internal communication.
			transports := &[]fgrpc.BindableTransport{}

			// SetState up a pool so we can load balance RPC connections.
			pool := fgrpc.NewPool(grpc.WithTransportCredentials(insecure.NewCredentials()))

			// AcquireSearcher the distribution layer.
			storageCfg := newStorageConfig(exp, logger)
			distConfig, err := newDistributionConfig(
				pool,
				exp,
				logger,
				storageCfg,
				transports,
			)
			dist, err := distribution.Open(ctx, distConfig)
			if err != nil {
				return err
			}
			defer func() { err = dist.Close() }()

			// SetState up our high level services.
			gorpDB := dist.Storage.Gorpify()
			userSvc := &user.Service{DB: gorpDB, Ontology: dist.Ontology}
			rsaKey, err := rsa.GenerateKey(rand.Reader, 1024)
			tokenSvc := &token.Service{Secret: rsaKey, Expiration: 15 * time.Minute}
			authenticator := &auth.KV{DB: gorpDB}

			// Provision the root user.
			if err := maybeProvisionRootUser(gorpDB, authenticator, userSvc); err != nil {
				return err
			}

			// Configure the core API.
			_api := api.New(api.Config{
				Logger:        logger,
				Channel:       dist.Channel,
				Segment:       dist.Segment,
				Storage:       dist.Storage,
				User:          userSvc,
				Token:         tokenSvc,
				Authenticator: authenticator,
				Enforcer:      access.AllowAll{},
				Insecure:      viper.GetBool("insecure"),
				Cluster:       dist.Cluster,
				Ontology:      dist.Ontology,
			})

			// Configure the HTTP API Transport.
			r := fhttp.NewRouter(fhttp.RouterConfig{Logger: logger.Sugar()})
			httpAPITransport := httpapi.New(r)
			_api.BindTo(httpAPITransport)

			// Configure our servers.
			grpcBranch := &server.GRPCBranch{Transports: *transports}
			httpBranch := &server.HTTPBranch{Transports: []fhttp.BindableTransport{r}}
			serverCfg := server.Config{
				ListenAddress: address.Address(viper.GetString("listen-address")),
				Logger:        logger,
				Branches: []server.Branch{
					httpBranch,
					grpcBranch,
				},
			}
			srv := server.New(serverCfg)
			sCtx.Go(srv.Start, xsignal.WithKey("server"))
			defer func() { err = errors.CombineErrors(err, srv.Stop()) }()
			<-ctx.Done()
			return nil
		}, xsignal.WithKey("start"))

		select {
		case <-sigC:
			logger.Info("received interrupt signal, shutting down")
			cancel()
		case <-sCtx.Stopped():
		}

		if err := sCtx.Wait(); err != nil && err != context.Canceled {
			logger.Sugar().Fatalw("server exited with error", "error", err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(startCmd)

	startCmd.Flags().StringP(
		"listen-address",
		"l",
		"127.0.0.1:9090",
		`
			`,
	)

	startCmd.Flags().StringSliceP(
		"peer-addresses",
		"p",
		nil,
		`
			Addresses of additional peers in the cluster.
		`,
	)

	startCmd.Flags().StringP(
		"data",
		"d",
		"synnax-data",
		`
			Dirname where synnax will store its data.
		`,
	)

	startCmd.Flags().BoolP(
		"mem",
		"m",
		false,
		`
			Use in-memory storage.
			`,
	)

	startCmd.Flags().BoolP(
		"debug",
		"v",
		false,
		"Enable debug mode.",
	)

	startCmd.Flags().BoolP(
		"insecure",
		"i",
		false,
		"Disable TLS and authentication.",
	)

	startCmd.Flags().String(
		"username",
		"synnax",
		"Username for the admin user.",
	)

	startCmd.Flags().String(
		"password",
		"seldon",
		"Password for the admin user.",
	)

	if err := viper.BindPFlags(startCmd.Flags()); err != nil {
		panic(err)
	}
}

func newStorageConfig(
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

func newDistributionConfig(
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
		AdvertiseAddress: address.Address(viper.GetString("listen-address")),
		PeerAddresses:    peers,
		Pool:             pool,
		Storage:          storage,
		Transports:       transports,
	}, err
}

func configureLogging() (*zap.Logger, error) {
	var cfg zap.Config
	if viper.GetBool("debug") {
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		cfg = zap.NewProductionConfig()
		cfg.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	}
	return cfg.Build()
}

var (
	rootExperimentKey = "experiment"
)

func configureObservability() alamos.Experiment {
	debug := viper.GetBool("debug")
	var opt alamos.Option
	if debug {
		opt = alamos.WithFilters(alamos.LevelFilterAll{})
	} else {
		opt = alamos.WithFilters(alamos.LevelFilterThreshold{Level: alamos.Production})
	}
	return alamos.New(rootExperimentKey, opt)
}

func maybeProvisionRootUser(
	db *gorp.DB,
	authSvc auth.Authenticator,
	userSvc *user.Service,
) error {
	rootUsername := viper.GetString("username")
	rootPassword := password.Raw(viper.GetString("password"))
	exists, err := userSvc.UsernameExists(rootUsername)
	if exists || err != nil {
		return err
	}
	txn := db.BeginTxn()
	if err := authSvc.NewWriterUsingTxn(txn).Register(auth.InsecureCredentials{
		Username: rootUsername,
		Password: rootPassword,
	}); err != nil {
		return err
	}
	if err := userSvc.NewWriterUsingTxn(txn).Create(&user.User{
		Username: rootUsername,
	}); err != nil {
		return err
	}
	if err := txn.Commit(); err != nil {
		return err
	}
	return nil
}
