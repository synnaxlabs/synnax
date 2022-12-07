package cmd

import (
	"context"
	"github.com/synnaxlabs/synnax/pkg/security"
	"github.com/synnaxlabs/synnax/pkg/security/cert"
	"github.com/synnaxlabs/x/httputil"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
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
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Starts a Synnax Node",
	Long: `
Starts a Synnax Node using the data directory specified by the --data flag,
and listening on the address specified by the --listen-address flag. If --peer-addresses
is specified and no existing data is found, the node will attempt to join the cluster
formed by its peers.
	`,
	Example: `synnax start --listen-address [host:port] --data /mnt/ssd1 --peer-addresses [host:port],[host:port] --insecure`,
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
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
		sCtx.Go(func(ctx context.Context) error {
			// Set up the tracing backend.
			exp := configureObservability()

			secSvc, err := configureSecurity()
			if err != nil {
				return err
			}

			// An array to hold the transports we use for cluster internal communication.
			transports := &[]fgrpc.BindableTransport{}

			// SetState up a pool to load balance RPC connections.
			var opts []grpc.DialOption
			if viper.GetBool("insecure") {
				opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
			} else {
				opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(secSvc.TLS())))
			}
			pool := fgrpc.NewPool(opts...)

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

			// Set up our high level services.
			gorpDB := dist.Storage.Gorpify()
			userSvc := &user.Service{DB: gorpDB, Ontology: dist.Ontology}
			tokenSvc := &token.Service{KeyService: secSvc, Expiration: 15 * time.Minute}
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
				Insecure:      viper.GetBool("insecure"),
				Cluster:       dist.Cluster,
				Ontology:      dist.Ontology,
			})

			// Configure the HTTP API Transport.
			r := fhttp.NewRouter(fhttp.RouterConfig{Logger: logger.Sugar()})
			httpAPI := httpapi.New(r)
			_api.BindTo(httpAPI)

			// Configure our servers.
			grpcBranch := &server.GRPCBranch{Transports: *transports}
			httpBranch := &server.HTTPBranch{
				Transports:   []fhttp.BindableTransport{r},
				ContentTypes: httputil.SupportedContentTypes(),
			}
			serverCfg := server.Config{
				ListenAddress: address.Address(viper.GetString("listen-address")),
				Logger:        logger.Named("server"),
				Branches:      []server.Branch{httpBranch, grpcBranch},
			}
			serverCfg.Security.CAName = "Synnax CA"
			serverCfg.Security.TLS = secSvc.TLS()
			serverCfg.Security.Insecure = config.BoolPointer(viper.GetBool("insecure"))
			srv, err := server.New(serverCfg)
			if err != nil {
				return err
			}

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

func configureSecurity() (security.Service, error) {
	loader, err := cert.NewLoader(cert.LoaderConfig{
		CertsDir: rootCmd.PersistentFlags().Lookup("certs-dir").Value.String(),
	})
	if err != nil {
		return nil, err
	}
	return security.NewService(loader)
}

func maybeProvisionRootUser(
	db *gorp.DB,
	authSvc auth.Authenticator,
	userSvc *user.Service,
) error {
	rootUser := viper.GetString("username")
	rootPass := password.Raw(viper.GetString("password"))
	exists, err := userSvc.UsernameExists(rootUser)
	if exists || err != nil {
		return err
	}
	txn := db.BeginTxn()
	if err := authSvc.NewWriterUsingTxn(txn).Register(auth.InsecureCredentials{
		Username: rootUser,
		Password: rootPass,
	}); err != nil {
		return err
	}
	if err := userSvc.NewWriterUsingTxn(txn).Create(&user.User{
		Username: rootUser,
	}); err != nil {
		return err
	}
	if err := txn.Commit(); err != nil {
		return err
	}
	return nil
}
