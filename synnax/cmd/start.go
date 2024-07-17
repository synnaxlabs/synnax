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
	"bufio"
	"context"
	"encoding/base64"
	"github.com/synnaxlabs/synnax/pkg/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"os"
	"os/signal"
	"time"

	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/freighter/fhttp"
	"github.com/synnaxlabs/synnax/pkg/api"
	grpcapi "github.com/synnaxlabs/synnax/pkg/api/grpc"
	httpapi "github.com/synnaxlabs/synnax/pkg/api/http"
	"github.com/synnaxlabs/synnax/pkg/auth"
	"github.com/synnaxlabs/synnax/pkg/auth/password"
	"github.com/synnaxlabs/synnax/pkg/auth/token"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/hardware"
	"github.com/synnaxlabs/synnax/pkg/hardware/embedded"
	"github.com/synnaxlabs/synnax/pkg/label"
	"github.com/synnaxlabs/synnax/pkg/ranger"
	"github.com/synnaxlabs/synnax/pkg/security"
	"github.com/synnaxlabs/synnax/pkg/server"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/synnax/pkg/user"
	"github.com/synnaxlabs/synnax/pkg/version"
	"github.com/synnaxlabs/synnax/pkg/workspace"
	"github.com/synnaxlabs/synnax/pkg/workspace/lineplot"
	"github.com/synnaxlabs/synnax/pkg/workspace/schematic"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	xsignal "github.com/synnaxlabs/x/signal"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	insecureGRPC "google.golang.org/grpc/credentials/insecure"
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Starts a Synnax Node",
	Long: `
Starts a Synnax Node using the data directory specified by the --data flag,
and listening on the address specified by the --listen flag. If --peers
is specified and no existing data is found, the node will attempt to join the cluster
formed by its peers. If no peers are specified and no existing data is found, the node
will bootstrap a new cluster.
	`,
	Example: `synnax start --listen [host:port] --data /mnt/ssd1 --peers [host:port],[host:port] --insecure`,
	Args:    cobra.NoArgs,
	Run:     func(cmd *cobra.Command, _ []string) { start(cmd) },
}

var (
	stopKeyWord = "stop"
)

// start a Synnax node using the configuration specified by the command line flags,
// environment variables, and configuration files.
func start(cmd *cobra.Command) {
	v := version.Get()
	decodedName, _ := base64.StdEncoding.DecodeString("bGljZW5zZS1rZXk=")
	var (
		ins, prettyLogger = configureInstrumentation(v)
		insecure          = viper.GetBool("insecure")
		debug             = viper.GetBool("debug")
		autoCert          = viper.GetBool("auto-cert")
		verifier          = viper.GetString(string(decodedName))
	)
	defer cleanupInstrumentation(cmd.Context(), ins)

	if autoCert {
		if err := generateAutoCerts(ins); err != nil {
			ins.L.Fatal("failed to generate auto certs", zap.Error(err))
		}
	}

	prettyLogger.Sugar().Infof("\033[34mSynnax version %s starting\033[0m", v)
	ins.L.Info("starting synnax node", zap.String("version", v))

	interruptC := make(chan os.Signal, 1)
	signal.Notify(interruptC, os.Interrupt)

	// Any data stored on the node is considered sensitive, so we need to set the
	// permission mask for all files appropriately.
	disablePermissionBits()

	sCtx, cancel := xsignal.WithCancel(cmd.Context(), xsignal.WithInstrumentation(ins))
	defer cancel()

	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			if scanner.Text() == stopKeyWord {
				interruptC <- os.Interrupt
			}
		}
	}()

	// Perform the rest of the startup within a separate goroutine, so we can properly
	// handle signal interrupts.
	sCtx.Go(func(ctx context.Context) (err error) {

		secProvider, err := configureSecurity(ins, insecure)
		if err != nil {
			return err
		}

		// An array to hold the grpcTransports we use for cluster internal communication.
		grpcTransports := &[]fgrpc.BindableTransport{}

		grpcPool := configureClientGRPC(secProvider, insecure)

		// Open the distribution layer.
		storageCfg := buildStorageConfig(ins)
		distConfig, err := buildDistributionConfig(
			grpcPool,
			ins,
			storageCfg,
			grpcTransports,
			verifier,
		)
		if err != nil {
			return err
		}
		dist, err := distribution.Open(ctx, distConfig)
		if err != nil {
			return err
		}
		defer func() {
			err = errors.CombineErrors(err, dist.Close())
		}()

		// set up our high level services.
		gorpDB := dist.Storage.Gorpify()
		userSvc, err := user.NewService(ctx, user.Config{
			DB:       gorpDB,
			Ontology: dist.Ontology,
			Group:    dist.Group,
		})
		if err != nil {
			return err
		}
		accessSvc, err := rbac.NewService(rbac.Config{
			DB: gorpDB,
		})
		if err != nil {
			return err
		}
		tokenSvc := &token.Service{KeyProvider: secProvider, Expiration: 24 * time.Hour}
		authenticator := &auth.KV{DB: gorpDB}
		rangeSvc, err := ranger.OpenService(ctx, ranger.Config{
			DB:       gorpDB,
			Ontology: dist.Ontology,
			Group:    dist.Group,
			Signals:  dist.Signals,
		})
		if err != nil {
			return err
		}
		workspaceSvc, err := workspace.NewService(ctx, workspace.Config{DB: gorpDB, Ontology: dist.Ontology, Group: dist.Group})
		if err != nil {
			return err
		}
		schematicSvc, err := schematic.NewService(schematic.Config{DB: gorpDB, Ontology: dist.Ontology})
		if err != nil {
			return err
		}
		linePlotSvc, err := lineplot.NewService(lineplot.Config{DB: gorpDB, Ontology: dist.Ontology})
		if err != nil {
			return err
		}
		labelSvc, err := label.OpenService(ctx, label.Config{
			DB:       gorpDB,
			Ontology: dist.Ontology,
			Group:    dist.Group,
			Signals:  dist.Signals,
		})
		hardwareSvc, err := hardware.OpenService(ctx, hardware.Config{
			DB:           gorpDB,
			Ontology:     dist.Ontology,
			Group:        dist.Group,
			HostProvider: dist.Cluster,
			Signals:      dist.Signals,
			Channel:      dist.Channel,
		})
		if err != nil {
			return err
		}
		defer func() {
			err = errors.CombineErrors(err, hardwareSvc.Close())
		}()

		// Provision the root user.
		if err = maybeProvisionRootUser(ctx, gorpDB, authenticator, userSvc, accessSvc); err != nil {
			return err
		}

		// Configure the API core.
		_api, err := api.New(api.Config{
			Instrumentation: ins.Child("api"),
			Authenticator:   authenticator,
			Access:          accessSvc,
			Schematic:       schematicSvc,
			LinePlot:        linePlotSvc,
			Insecure:        config.Bool(insecure),
			Channel:         dist.Channel,
			Framer:          dist.Framer,
			Storage:         dist.Storage,
			User:            userSvc,
			Token:           tokenSvc,
			Cluster:         dist.Cluster,
			Ontology:        dist.Ontology,
			Group:           dist.Group,
			Ranger:          rangeSvc,
			Workspace:       workspaceSvc,
			Label:           labelSvc,
			Hardware:        hardwareSvc,
		})
		if err != nil {
			return err
		}

		// Configure the HTTP API Transport.
		r := fhttp.NewRouter(fhttp.RouterConfig{Instrumentation: ins})
		_api.BindTo(httpapi.New(r))

		// Configure the GRPC API Transport.
		grpcAPI, grpcAPITrans := grpcapi.New()
		*grpcTransports = append(*grpcTransports, grpcAPITrans...)
		_api.BindTo(grpcAPI)

		srv, err := server.New(buildServerConfig(
			*grpcTransports,
			[]fhttp.BindableTransport{r},
			secProvider,
			ins,
			debug,
		))
		if err != nil {
			return err
		}
		sCtx.Go(func(_ context.Context) error {
			defer cancel()
			return srv.Serve()
		}, xsignal.WithKey("server"))
		defer srv.Stop()

		d, err := embedded.OpenDriver(
			ctx,
			buildEmbeddedDriverConfig(
				ins.Child("driver"),
				hardwareSvc.Rack.EmbeddedRackName,
				insecure,
			),
		)
		if err != nil {
			return err
		}
		defer func() {
			err = errors.CombineErrors(err, d.Stop())
		}()

		prettyLogger.Info("\033[32mSynnax is running and available at " + viper.GetString("listen") + "\033[0m")

		<-ctx.Done()
		return err
	}, xsignal.WithKey("start"))

	select {
	case <-interruptC:
		ins.L.Info("received interrupt signal, shutting down")
		prettyLogger.Info("\033[33mSynnax is shutting down\033[0m")
		cancel()
	case <-sCtx.Stopped():
	}

	if err := sCtx.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		prettyLogger.Sugar().Errorf("\033[31mSynnax has encountered an error and is shutting down: %v\033[0m", err)
		ins.L.Fatal("synnax failed", zap.Error(err))
	}
	ins.L.Info("shutdown successful")
	prettyLogger.Info("\033[34mSynnax has shut down\033[0m")
}

func init() {
	rootCmd.AddCommand(startCmd)
	configureStartFlags()
	bindFlags(startCmd)
}

func buildStorageConfig(
	ins alamos.Instrumentation,
) storage.Config {
	return storage.Config{
		Instrumentation: ins.Child("storage"),
		MemBacked:       config.Bool(viper.GetBool("mem")),
		Dirname:         viper.GetString("data"),
	}
}

func parsePeerAddresses() ([]address.Address, error) {
	peerStrings := viper.GetStringSlice("peers")
	peerAddresses := make([]address.Address, len(peerStrings))
	for i, listenString := range peerStrings {
		peerAddresses[i] = address.Address(listenString)
	}
	return peerAddresses, nil
}

func buildDistributionConfig(
	pool *fgrpc.Pool,
	ins alamos.Instrumentation,
	storage storage.Config,
	transports *[]fgrpc.BindableTransport,
	verifier string,
) (distribution.Config, error) {
	peers, err := parsePeerAddresses()
	return distribution.Config{
		Instrumentation:  ins.Child("distribution"),
		AdvertiseAddress: address.Address(viper.GetString("listen")),
		PeerAddresses:    peers,
		Pool:             pool,
		Storage:          storage,
		Transports:       transports,
		Verifier:         verifier,
	}, err
}

func buildServerConfig(
	grpcTransports []fgrpc.BindableTransport,
	httpTransports []fhttp.BindableTransport,
	sec security.Provider,
	ins alamos.Instrumentation,
	debug bool,
) (cfg server.Config) {
	cfg.Branches = append(cfg.Branches,
		&server.SecureHTTPBranch{Transports: httpTransports},
		&server.GRPCBranch{Transports: grpcTransports},
		server.NewHTTPRedirectBranch(),
	)
	cfg.Debug = config.Bool(debug)
	cfg.ListenAddress = address.Address(viper.GetString("listen"))
	cfg.Instrumentation = ins.Child("server")
	cfg.Security.TLS = sec.TLS()
	cfg.Security.Insecure = config.Bool(viper.GetBool("insecure"))
	return cfg
}

func buildEmbeddedDriverConfig(
	ins alamos.Instrumentation,
	rackName string,
	insecure bool,
) embedded.Config {
	cfg := embedded.Config{
		Enabled:         config.Bool(!viper.GetBool("no-driver")),
		Instrumentation: ins,
		Address:         address.Address(viper.GetString("listen")),
		RackName:        rackName,
		Username:        viper.GetString("username"),
		Password:        viper.GetString("password"),
		Debug:           config.Bool(viper.GetBool("debug")),
	}
	if insecure {
		return cfg
	}
	loader := buildCertLoaderConfig(ins)
	cfg.CACertPath = loader.AbsoluteCACertPath()
	cfg.ClientCertFile = loader.AbsoluteNodeCertPath()
	cfg.ClientKeyFile = loader.AbsoluteNodeKeyPath()
	return cfg
}

func configureSecurity(ins alamos.Instrumentation, insecure bool) (security.Provider, error) {
	return security.NewProvider(security.ProviderConfig{
		LoaderConfig: buildCertLoaderConfig(ins),
		Insecure:     config.Bool(insecure),
		KeySize:      viper.GetInt("key-size"),
	})
}

func maybeProvisionRootUser(
	ctx context.Context,
	db *gorp.DB,
	authSvc auth.Authenticator,
	userSvc *user.Service,
	accessSvc *rbac.Service,
) error {
	creds := auth.InsecureCredentials{
		Username: viper.GetString("username"),
		Password: password.Raw(viper.GetString("password")),
	}
	exists, err := userSvc.UsernameExists(ctx, creds.Username)
	if err != nil || exists {
		return err
	}

	// Register the user first.
	return db.WithTx(ctx, func(tx gorp.Tx) error {
		if err = authSvc.NewWriter(tx).Register(ctx, creds); err != nil {
			return err
		}
		userObj := user.User{Username: creds.Username}
		if err = userSvc.NewWriter(tx).Create(ctx, &userObj); err != nil {
			return err
		}
		return accessSvc.NewWriter(tx).Create(
			ctx,
			&rbac.Policy{
				Subjects: []ontology.ID{user.OntologyID(userObj.Key)},
				Objects:  []ontology.ID{rbac.AllowAll},
			},
		)
	})
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
