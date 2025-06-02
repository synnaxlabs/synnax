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
	"fmt"
	"os"
	"os/signal"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/layer"
	"github.com/synnaxlabs/synnax/pkg/service"
	xio "github.com/synnaxlabs/x/io"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/freighter/fhttp"
	"github.com/synnaxlabs/synnax/pkg/api"
	grpcapi "github.com/synnaxlabs/synnax/pkg/api/grpc"
	httpapi "github.com/synnaxlabs/synnax/pkg/api/http"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/security"
	"github.com/synnaxlabs/synnax/pkg/server"
	"github.com/synnaxlabs/synnax/pkg/service/hardware/embedded"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/synnax/pkg/version"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
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
	var (
		v                   = version.Get()
		verifierFlag        = lo.Must(base64.StdEncoding.DecodeString("bGljZW5zZS1rZXk="))
		insecure            = viper.GetBool(insecureFlag)
		debug               = viper.GetBool(debugFlag)
		autoCert            = viper.GetBool(autoCertFlag)
		verifier            = viper.GetString(string(verifierFlag))
		memBacked           = viper.GetBool(memFlag)
		listenAddress       = address.Address(viper.GetString(listenFlag))
		dataPath            = viper.GetString(dataFlag)
		slowConsumerTimeout = viper.GetDuration(slowConsumerTimeoutFlag)
		rootUsername        = viper.GetString(usernameFlag)
		rootPassword        = viper.GetString(passwordFlag)
		noDriver            = viper.GetBool(noDriverFlag)
		keySizeFlag         = viper.GetInt(keySizeFlag)
		ins                 = configureInstrumentation()
	)
	defer cleanupInstrumentation(cmd.Context(), ins)

	if autoCert {
		if err := generateAutoCerts(ins); err != nil {
			ins.L.Fatal("failed to generate auto certs", zap.Error(err))
		}
	}

	ins.L.Zap().Sugar().Infof("\033[34mSynnax version %s starting\033[0m", v)
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
	// handle signal interrupts. We'll also repeatedly check for context cancellations
	// at each step in the process to ensure we can shut down early if necessary.
	sCtx.Go(func(ctx context.Context) error {
		var (
			err               error
			closer            xio.MultiCloser
			peers             []address.Address
			securityProvider  security.Provider
			distributionLayer *distribution.Layer
			serviceLayer      *service.Layer
			apiLayer          *api.Layer
			rootServer        *server.Server
			embeddedDriver    *embedded.Driver
			certLoaderConfig  = buildCertLoaderConfig(ins)
		)
		cleanup, ok := layer.NewOpener(ctx, &err, &closer)
		defer cleanup()
		if securityProvider, err = security.NewProvider(security.ProviderConfig{
			LoaderConfig: certLoaderConfig,
			Insecure:     config.Bool(insecure),
			KeySize:      keySizeFlag,
		}); !ok(nil) {
			return err
		}

		// An array to hold the grpcServerTransports we use for cluster internal communication.
		grpcServerTransports := &[]fgrpc.BindableTransport{}
		grpcClientPool := configureClientGRPC(securityProvider, insecure)

		if peers, err = parsePeerAddressFlag(); !ok(nil) {
			return err
		}

		if distributionLayer, err = distribution.Open(ctx, distribution.Config{
			Config: core.Config{
				Instrumentation:  ins.Child("distribution"),
				AdvertiseAddress: listenAddress,
				PeerAddresses:    peers,
				Pool:             grpcClientPool,
				Storage: storage.Config{
					Instrumentation: ins.Child("storage"),
					MemBacked:       config.Bool(memBacked),
					Dirname:         dataPath,
				},
				Transports: grpcServerTransports,
			},
			Verifier: verifier,
		}); !ok(distributionLayer) {
			return err
		}

		if serviceLayer, err = service.Open(ctx, service.Config{
			Instrumentation: ins.Child("service"),
			Distribution:    distributionLayer,
			Security:        securityProvider,
		}); !ok(serviceLayer) {
			return err
		}

		// Provision the root user.
		if err = maybeProvisionRootUser(ctx, serviceLayer); !ok(nil) {
			return err
		}

		// Set the base permissions for all users.
		if err = maybeSetBasePermission(ctx, serviceLayer); !ok(nil) {
			return err
		}

		// Configure the Layer core.
		if apiLayer, err = api.New(api.Config{
			Instrumentation: ins.Child("api"),
			Service:         serviceLayer,
			Distribution:    distributionLayer,
		}); !ok(nil) {
			return err
		}

		// We run startup searching indexing after all services have been
		// registered within the ontology. We used to fork a new goroutine for
		// every service at registration time, but this caused a race condition
		// where bleve would concurrently read and write to a map.
		// See https://linear.app/synnax/issue/SY-1116/race-condition-on-server-startup
		// for more details on this issue.
		sCtx.Go(distributionLayer.Ontology.RunStartupSearchIndexing, xsignal.WithKey("startup_search_indexing"))

		// Configure the HTTP Layer Transport.
		r := fhttp.NewRouter(fhttp.RouterConfig{
			Instrumentation:     ins,
			StreamWriteDeadline: slowConsumerTimeout,
		})
		apiLayer.BindTo(httpapi.New(r, api.NewHTTPCodecResolver(distributionLayer.Channel)))

		// Configure the GRPC Layer Transport.
		grpcAPI, grpcAPITrans := grpcapi.New(distributionLayer.Channel)
		*grpcServerTransports = append(*grpcServerTransports, grpcAPITrans...)
		apiLayer.BindTo(grpcAPI)

		if rootServer, err = server.Serve(
			server.Config{
				Branches: []server.Branch{
					&server.SecureHTTPBranch{Transports: []fhttp.BindableTransport{r}},
					&server.GRPCBranch{Transports: *grpcServerTransports},
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
		); !ok(rootServer) {
			return err
		}

		if embeddedDriver, err = embedded.OpenDriver(
			ctx,
			embedded.Config{
				Enabled:         config.Bool(!noDriver),
				Integrations:    parseIntegrationsFlag(),
				Instrumentation: ins,
				Address:         listenAddress,
				RackKey:         serviceLayer.Hardware.Rack.EmbeddedKey,
				ClusterKey:      distributionLayer.Cluster.Key(),
				Username:        rootUsername,
				Password:        rootPassword,
				Debug:           config.Bool(debug),
				CACertPath:      certLoaderConfig.AbsoluteCACertPath(),
				ClientCertFile:  certLoaderConfig.AbsoluteNodeCertPath(),
				ClientKeyFile:   certLoaderConfig.AbsoluteNodeKeyPath(),
			},
		); !ok(embeddedDriver) {
			return err
		}

		ins.L.Info(fmt.Sprintf("\033[32mSynnax is running and available at %v \033[0m", listenAddress))

		<-ctx.Done()
		return err
	},
		xsignal.WithKey("start"),
		xsignal.RecoverWithErrOnPanic(),
	)

	select {
	case <-interruptC:
		ins.L.Info("\033[33mSynnax is shutting down. This can take up to 5 seconds. Please be patient\033[0m")
		cancel()
	case <-sCtx.Stopped():
	}

	if err := sCtx.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		ins.L.Zap().Sugar().Errorf("\033[31mSynnax has encountered an error and is shutting down: %v\033[0m", err)
		ins.L.Fatal("synnax failed", zap.Error(err))
	}
	ins.L.Info("\033[34mSynnax has shut down\033[0m")
}

func init() {
	rootCmd.AddCommand(startCmd)
	configureStartFlags()
	bindFlags(startCmd)
}
