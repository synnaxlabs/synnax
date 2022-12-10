package cmd

import (
	"context"
	"github.com/synnaxlabs/synnax/pkg/security"
	"github.com/synnaxlabs/synnax/pkg/security/cert"
	"github.com/synnaxlabs/x/httputil"
	"github.com/synnaxlabs/x/query"
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
			exp := configureObservability()

			secProvider, err := configureSecurity()
			if err != nil {
				return err
			}

			// An array to hold the grpcTransports we use for cluster internal communication.
			grpcTransports := &[]fgrpc.BindableTransport{}

			// Set up a pool to load balance RPC connections.
			var opts []grpc.DialOption
			if viper.GetBool("insecure") {
				opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
			} else {
				opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(secProvider.TLS())))
			}
			pool := fgrpc.NewPool(opts...)

			// Open the distribution layer.
			storageCfg := newStorageConfig(exp, logger)
			distConfig, err := newDistributionConfig(
				pool,
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
			tokenSvc := &token.Service{KeyService: secProvider, Expiration: 15 * time.Minute}
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
			_api.BindTo(httpapi.New(r))

			srv, err := server.New(newServerConfig(
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

		if err := sCtx.Wait(); err != nil && err != context.Canceled {
			logger.Sugar().Fatalw("server exited with error", "error", err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
	configureStartFlags()
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

func newServerConfig(
	grpcTransports []fgrpc.BindableTransport,
	httpTransports []fhttp.BindableTransport,
	sec security.Provider,
	logger *zap.Logger,
) (cfg server.Config) {
	cfg.Branches = append(cfg.Branches,
		&server.GRPCBranch{Transports: grpcTransports},
		&server.SecureHTTPBranch{
			Transports:   httpTransports,
			ContentTypes: httputil.SupportedContentTypes(),
		},
		server.NewHTTPRedirectBranch(),
	)
	cfg.ListenAddress = address.Address(viper.GetString("listen-address"))
	cfg.Logger = logger.Named("server")
	cfg.Security.TLS = sec.TLS()
	cfg.Security.Insecure = config.BoolPointer(viper.GetBool("insecure"))
	return cfg
}

func configureLogging() (*zap.Logger, error) {
	var cfg zap.Config
	if viper.GetBool("debug") {
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		cfg.Encoding = "console"
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

func configureSecurity() (security.Provider, error) {
	return security.NewProvider(security.ProviderConfig{
		LoaderConfig: cert.LoaderConfig{
			CertsDir: rootCmd.PersistentFlags().Lookup("certs-dir").Value.String(),
		},
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
	if err != nil {
		return err
	}
	if exists {
		return errors.Wrapf(query.UniqueViolation, "user %q already exists", uname)
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
