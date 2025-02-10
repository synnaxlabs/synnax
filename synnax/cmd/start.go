// Copyright 2025 Synnax Labs, Inc.
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
	"os"
	"os/signal"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/freighter/fhttp"
	"github.com/synnaxlabs/synnax/pkg/api"
	grpcapi "github.com/synnaxlabs/synnax/pkg/api/grpc"
	httpapi "github.com/synnaxlabs/synnax/pkg/api/http"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/security"
	"github.com/synnaxlabs/synnax/pkg/server"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/auth/token"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/hardware"
	"github.com/synnaxlabs/synnax/pkg/service/hardware/embedded"
	"github.com/synnaxlabs/synnax/pkg/service/hardware/rack"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/synnax/pkg/service/workspace"
	"github.com/synnaxlabs/synnax/pkg/service/workspace/lineplot"
	"github.com/synnaxlabs/synnax/pkg/service/workspace/log"
	"github.com/synnaxlabs/synnax/pkg/service/workspace/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/workspace/table"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/synnax/pkg/version"
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

const stopKeyWord = "stop"

func scanForStopKeyword(interruptC chan os.Signal) {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		if scanner.Text() == stopKeyWord {
			interruptC <- os.Interrupt
		}
	}
}

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

// start a Synnax node using the configuration specified by the command line flags,
// environment variables, and configuration files.
func start(cmd *cobra.Command) {
	v := version.Get()
	decodedName, _ := base64.StdEncoding.DecodeString("bGljZW5zZS1rZXk=")
	var (
		ins, prettyLogger = configureInstrumentation(v)
		insecure          = viper.GetBool(insecureFlag)
		debug             = viper.GetBool(debugFlag)
		autoCert          = viper.GetBool(autoCertFlag)
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

	// Listen for a custom stop keyword that can be used in place of a Ctrl+C signal.
	// It's fine to let this get garbage collected.
	go scanForStopKeyword(interruptC)

	// Perform the rest of the startup within a separate goroutine, so we can properly
	// handle signal interrupts.
	sCtx.Go(func(ctx context.Context) (err error) {

		secProvider, err := configureSecurity(ins, insecure)
		if err != nil {
			return err
		}

		// An array to hold the grpcServerTransports we use for cluster internal communication.
		grpcServerTransports := &[]fgrpc.BindableTransport{}
		grpcClientPool := configureClientGRPC(secProvider, insecure)

		// Open the distribution layer.
		storageCfg := buildStorageConfig(ins)
		distConfig, err := buildDistributionConfig(
			grpcClientPool,
			ins,
			storageCfg,
			grpcServerTransports,
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
			err = errors.Combine(err, dist.Close())
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
		rbacSvc, err := rbac.NewService(rbac.Config{DB: gorpDB})
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
		labelSvc, err := label.OpenService(
			ctx,
			label.Config{
				DB:       gorpDB,
				Ontology: dist.Ontology,
				Group:    dist.Group,
				Signals:  dist.Signals,
			},
		)
		if err != nil {
			return err
		}
		logSvc, err := log.NewService(log.Config{DB: gorpDB, Ontology: dist.Ontology})
		if err != nil {
			return err
		}
		tableSvc, err := table.NewService(table.Config{
			DB:       gorpDB,
			Ontology: dist.Ontology,
		})
		hardwareSvc, err := hardware.OpenService(
			ctx,
			hardware.Config{
				DB:           gorpDB,
				Ontology:     dist.Ontology,
				Group:        dist.Group,
				HostProvider: dist.Cluster,
				Signals:      dist.Signals,
				Channel:      dist.Channel,
			})
		defer func() {
			err = errors.Combine(err, hardwareSvc.Close())
		}()
		frameSvc, err := framer.OpenService(
			ctx,
			framer.Config{
				Instrumentation: ins.Child("framer"),
				Framer:          dist.Framer,
				Channel:         dist.Channel,
			},
		)
		if err != nil {
			return err
		}
		defer func() {
			err = errors.Combine(err, frameSvc.Close())
		}()

		// Provision the root user.
		if err = maybeProvisionRootUser(ctx, gorpDB, authenticator, userSvc, rbacSvc); err != nil {
			return err
		}

		// Set the base permissions for all users.
		if err = maybeSetBasePermission(ctx, gorpDB, rbacSvc); err != nil {
			return err
		}

		// Configure the API core.
		_api, err := api.New(
			api.Config{
				Instrumentation: ins.Child("api"),
				Authenticator:   authenticator,
				Enforcer:        &access.AllowAll{},
				RBAC:            rbacSvc,
				Schematic:       schematicSvc,
				LinePlot:        linePlotSvc,
				Insecure:        config.Bool(insecure),
				Channel:         dist.Channel,
				Framer:          frameSvc,
				Storage:         dist.Storage,
				User:            userSvc,
				Token:           tokenSvc,
				Table:           tableSvc,
				Cluster:         dist.Cluster,
				Ontology:        dist.Ontology,
				Group:           dist.Group,
				Ranger:          rangeSvc,
				Log:             logSvc,
				Workspace:       workspaceSvc,
				Label:           labelSvc,
				Hardware:        hardwareSvc,
			},
		)
		if err != nil {
			return err
		}

		// Configure the HTTP API Transport.
		r := fhttp.NewRouter(fhttp.RouterConfig{
			Instrumentation:     ins,
			StreamWriteDeadline: viper.GetDuration(slowConsumerTimeoutFlag),
		})
		_api.BindTo(httpapi.New(r))

		// Configure the GRPC API Transport.
		grpcAPI, grpcAPITrans := grpcapi.New()
		*grpcServerTransports = append(*grpcServerTransports, grpcAPITrans...)
		_api.BindTo(grpcAPI)

		srv, err := server.New(buildServerConfig(
			*grpcServerTransports,
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
		},
			xsignal.WithKey("server"),
			xsignal.RecoverWithErrOnPanic(),
		)
		defer srv.Stop()

		d, err := embedded.OpenDriver(
			ctx,
			buildEmbeddedDriverConfig(
				ins.Child("driver"),
				hardwareSvc.Rack.EmbeddedKey,
				dist.Cluster.Key(),
				insecure,
			),
		)
		if err != nil {
			return err
		}
		defer func() {
			err = errors.Combine(err, d.Stop())
		}()

		prettyLogger.Info("\033[32mSynnax is running and available at " + viper.GetString(listenFlag) + "\033[0m")

		<-ctx.Done()
		return err
	},
		xsignal.WithKey("start"),
		xsignal.RecoverWithErrOnPanic(),
	)

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
		MemBacked:       config.Bool(viper.GetBool(memFlag)),
		Dirname:         viper.GetString(dataFlag),
	}
}

func parsePeerAddresses() ([]address.Address, error) {
	peerStrings := viper.GetStringSlice(peersFlag)
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
		AdvertiseAddress: address.Address(viper.GetString(listenFlag)),
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
	cfg.Branches = append(
		cfg.Branches,
		&server.SecureHTTPBranch{Transports: httpTransports},
		&server.GRPCBranch{Transports: grpcTransports},
		server.NewHTTPRedirectBranch(),
	)
	cfg.Debug = config.Bool(debug)
	cfg.ListenAddress = address.Address(viper.GetString(listenFlag))
	cfg.Instrumentation = ins.Child("server")
	cfg.Security.TLS = sec.TLS()
	cfg.Security.Insecure = config.Bool(viper.GetBool(insecureFlag))
	return cfg
}

func buildEmbeddedDriverConfig(
	ins alamos.Instrumentation,
	rackKey rack.Key,
	clusterKey uuid.UUID,
	insecure bool,
) embedded.Config {
	cfg := embedded.Config{
		Enabled: config.Bool(!viper.GetBool(noDriverFlag)),
		Integrations: getIntegrations(
			viper.GetStringSlice(enableIntegrationsFlag),
			viper.GetStringSlice(disableIntegrationsFlag),
		),
		Instrumentation: ins,
		Address:         address.Address(viper.GetString(listenFlag)),
		RackKey:         rackKey,
		ClusterKey:      clusterKey,
		Username:        viper.GetString(usernameFlag),
		Password:        viper.GetString(passwordFlag),
		Debug:           config.Bool(viper.GetBool(debugFlag)),
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
		KeySize:      viper.GetInt(keySizeFlag),
	})
}

func getIntegrations(enabled, disabled []string) []string {
	if len(enabled) > 0 {
		return enabled
	}
	return lo.Filter(embedded.AllIntegrations, func(integration string, _ int) bool {
		return !lo.Contains(disabled, integration)
	})
}

// sets the base permissions that need to exist in the server.
func maybeSetBasePermission(
	ctx context.Context,
	db *gorp.DB,
	rbacSvc *rbac.Service,
) error {
	return db.WithTx(ctx, func(tx gorp.Tx) error {
		// base policies that need to be created
		basePolicies := map[ontology.Type]access.Action{
			"label":       access.All,
			"cluster":     access.All,
			"channel":     access.All,
			"node":        access.All,
			"group":       access.All,
			"range":       access.All,
			"range-alias": access.All,
			"workspace":   access.All,
			"log":         access.All,
			"lineplot":    access.All,
			"rack":        access.All,
			"device":      access.All,
			"task":        access.All,
			"table":       access.All,
			"user":        access.Retrieve,
			"schematic":   access.Retrieve,
			"policy":      access.Retrieve,
			"builtin":     access.Retrieve,
			"framer":      access.All,
		}
		// for migration purposes, some old base policies that need to be deleted
		oldBasePolicies := map[ontology.Type]access.Action{}

		existingPolicies := make([]rbac.Policy, 0, len(basePolicies))
		policiesToDelete := make([]uuid.UUID, 0, len(oldBasePolicies))
		if err := rbacSvc.NewRetriever().WhereSubjects(user.OntologyTypeID).
			Entries(&existingPolicies).Exec(ctx, tx); err != nil {
			return err
		}
		for _, p := range existingPolicies {
			if len(p.Subjects) != 1 || len(p.Objects) != 1 || len(p.Actions) != 1 {
				// then this policy is not one of the policies created in maybeSetBasePermission
				continue
			}
			s := p.Subjects[0]
			o := p.Objects[0]
			a := p.Actions[0]
			if (s != user.OntologyTypeID) || (o.Key != "") {
				// the policy does not apply to the general user ontology type
				continue
			}
			if basePolicies[o.Type] == a {
				delete(basePolicies, o.Type)
			} else if oldBasePolicies[o.Type] == a {
				policiesToDelete = append(policiesToDelete, p.Key)
			}
		}
		for t := range basePolicies {
			if err := rbacSvc.NewWriter(tx).Create(ctx, &rbac.Policy{
				Subjects: []ontology.ID{user.OntologyTypeID},
				Objects:  []ontology.ID{{Type: t, Key: ""}},
				Actions:  []access.Action{basePolicies[t]},
			}); err != nil {
				return err
			}
		}
		return rbacSvc.NewWriter(tx).Delete(ctx, policiesToDelete...)
	})
}

func maybeProvisionRootUser(
	ctx context.Context,
	db *gorp.DB,
	authSvc auth.Authenticator,
	userSvc *user.Service,
	rbacSvc *rbac.Service,
) error {
	creds := auth.InsecureCredentials{
		Username: viper.GetString(usernameFlag),
		Password: password.Raw(viper.GetString(passwordFlag)),
	}
	exists, err := userSvc.UsernameExists(ctx, creds.Username)
	if err != nil {
		return err
	}
	if exists {
		// we potentially need to update the root user flag

		// we want to make sure the root user still has the allow_all policy
		return db.WithTx(ctx, func(tx gorp.Tx) error {
			// For cluster versions before v0.31.0, the root user flag was not set. We
			// need to set it here.
			if err = userSvc.NewWriter(tx).MaybeSetRootUser(ctx, creds.Username); err != nil {
				return err
			}

			var u user.User
			if err = userSvc.NewRetrieve().WhereUsernames(creds.Username).Entry(&u).Exec(ctx, tx); err != nil {
				return err
			}
			if !u.RootUser {
				return nil
			}
			policies := make([]rbac.Policy, 0, 1)
			rbacSvc.NewRetriever().WhereSubjects(user.OntologyID(u.Key)).Entries(&policies).Exec(ctx, tx)
			for _, p := range policies {
				if lo.Contains(p.Objects, rbac.AllowAllOntologyID) {
					return nil
				}
			}
			return rbacSvc.NewWriter(tx).Create(ctx, &rbac.Policy{
				Subjects: []ontology.ID{user.OntologyID(u.Key)},
				Objects:  []ontology.ID{rbac.AllowAllOntologyID},
				Actions:  []access.Action{},
			})
		})
	}

	// Register the user first, then give them all permissions
	return db.WithTx(ctx, func(tx gorp.Tx) error {
		if err = authSvc.NewWriter(tx).Register(ctx, creds); err != nil {
			return err
		}
		userObj := user.User{Username: creds.Username, RootUser: true}
		if err = userSvc.NewWriter(tx).Create(ctx, &userObj); err != nil {
			return err
		}
		return rbacSvc.NewWriter(tx).Create(
			ctx,
			&rbac.Policy{
				Subjects: []ontology.ID{user.OntologyID(userObj.Key)},
				Objects:  []ontology.ID{rbac.AllowAllOntologyID},
				Actions:  []access.Action{},
			},
		)
	})
}

func configureClientGRPC(
	sec security.Provider,
	insecure bool,
) *fgrpc.Pool {
	return fgrpc.NewPool(
		"",
		grpc.WithTransportCredentials(getClientGRPCTransportCredentials(sec, insecure)),
	)
}

func getClientGRPCTransportCredentials(sec security.Provider, insecure bool) credentials.TransportCredentials {
	return lo.Ternary(insecure, insecureGRPC.NewCredentials(), credentials.NewTLS(sec.TLS()))
}
