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
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/cmd/cert"
	"github.com/synnaxlabs/synnax/cmd/instrumentation"
	"github.com/synnaxlabs/synnax/cmd/listener"
	"github.com/synnaxlabs/synnax/cmd/start/internal/stdin"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/errors"
	xsignal "github.com/synnaxlabs/x/signal"
	"go.uber.org/zap"
)

// Cmd represents the start command
var Cmd = &cobra.Command{
	Use:     "start",
	Short:   "Starts a Synnax Core",
	Long:    "Starts a Synnax Core using the data directory specified by the --data flag, and listening on the address specified by the --listen flag. If --peers is specified and no existing data is found, the Core will attempt to join the cluster formed by its peers. If no peers are specified and no existing data is found, the Core will bootstrap a new cluster.",
	Example: "synnax start --listen localhost:9091 --data /mnt/ssd1 --peers localhost:9092,localhost:9093 --insecure",
	Args:    cobra.NoArgs,
	PreRunE: func(cmd *cobra.Command, _ []string) error {
		return viper.BindPFlags(cmd.Flags())
	},
	Run: func(cmd *cobra.Command, _ []string) { start(cmd) },
}

// start is the entrypoint for starting a Synnax Core. It handles stop requests and
// delegates to BootupCore for the actual startup.
func start(cmd *cobra.Command) {
	ctx := cmd.Context()
	ins := instrumentation.Configure()
	defer instrumentation.Cleanup(ctx, ins)

	// sigC holds two signals, so a second signal sent right after the first is kept.
	sigC := make(chan os.Signal, 2)
	signal.Notify(sigC, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigC)

	sCtx, cancel := xsignal.WithCancel(ctx, xsignal.WithInstrumentation(ins))
	defer cancel()

	// Listen for a custom stop keyword that can be used in place of a Ctrl+C signal.
	// A read from stdin cannot be interrupted, so this goroutine runs until exit.
	stdinC := make(chan struct{}, 1)
	go stdin.Watch(os.Stdin, viper.GetBool(FlagStopOnStdinClose), func() {
		select {
		case stdinC <- struct{}{}:
		default:
		}
	})

	cfg, err := GetCoreConfigFromViper(ins)
	if err != nil {
		ins.L.Fatal("invalid configuration", zap.Error(err))
		return
	}

	sCtx.Go(func(ctx context.Context) error {
		return BootupCore(ctx, nil, cfg)
	}, xsignal.WithKey("start"), xsignal.RecoverWithErrOnPanic())

	shutdownDone := make(chan struct{})
	var wg sync.WaitGroup
	// shutDown cancels the Core, then exits at once if signalsToForce more stop signals
	// arrive before the shutdown completes. Stdin requests never count: the Desktop
	// supervisor sends the stop keyword, then closes stdin.
	shutDown := func(signalsToForce int) {
		ins.L.Info(
			"\033[33mSynnax is shutting down. Press Ctrl+C again to exit now.\033[0m",
		)
		cancel()
		wg.Go(func() {
			for range signalsToForce {
				select {
				case <-sigC:
				case <-shutdownDone:
					return
				}
			}
			ins.L.Fatal(
				"received a second stop signal, exiting before shutdown completes",
			)
		})
	}

	select {
	case <-sigC:
		shutDown(1)
	case <-stdinC:
		shutDown(2)
	case <-sCtx.Stopped():
	}

	err = sCtx.Wait()
	close(shutdownDone)
	wg.Wait()
	if err != nil && !errors.Is(err, context.Canceled) {
		ins.L.Zap().Sugar().Errorf(
			"\033[31mSynnax has encountered an error and is shutting down: %v\033[0m",
			err,
		)
		ins.L.Fatal("synnax failed", zap.Error(err))
	}
	ins.L.Info("\033[34mSynnax has shut down\033[0m")
}

func init() { AddFlags(Cmd) }

// readLicenseKey returns the trimmed license key from the key flag, or the trimmed
// contents of the file the path flag names when the key flag is empty.
func readLicenseKey() (string, error) {
	if v := strings.TrimSpace(viper.GetString(FlagLicenseKey)); v != "" {
		return v, nil
	}
	path := viper.GetString(FlagLicenseFile)
	if path == "" {
		return "", nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", errors.Wrapf(err, "failed to read the --%s file", FlagLicenseFile)
	}
	return strings.TrimSpace(string(b)), nil
}

// GetCoreConfigFromViper builds a CoreConfig from the current viper configuration.
// This is used by the Windows service to start the Core with the config loaded from
// a YAML file.
func GetCoreConfigFromViper(ins alamos.Instrumentation) (CoreConfig, error) {
	peers := lo.Map(
		viper.GetStringSlice(FlagPeers),
		func(peer string, _ int) address.Address {
			return address.Address(peer)
		},
	)
	factoryCfg := cert.BuildCertFactoryConfig(ins)
	listeners, err := listener.Parse()
	if err != nil {
		return CoreConfig{}, err
	}
	factoryCfg.Hosts = lo.Map(
		listeners,
		func(l listener.Config, _ int) address.Address {
			return l.Address
		},
	)
	licenseKey, err := readLicenseKey()
	if err != nil {
		return CoreConfig{}, err
	}
	return CoreConfig{
		Instrumentation:     ins,
		insecure:            new(viper.GetBool(FlagInsecure)),
		debug:               new(viper.GetBool(instrumentation.FlagDebug)),
		autoCert:            new(viper.GetBool(cert.FlagAutoCert)),
		licenseKey:          licenseKey,
		memBacked:           new(viper.GetBool(FlagMem)),
		listeners:           listeners,
		peers:               peers,
		dataPath:            viper.GetString(FlagData),
		slowConsumerTimeout: viper.GetDuration(FlagSlowConsumerTimeout),
		rootCredentials: auth.Credentials{
			Username: viper.GetString(FlagUsername),
			Password: viper.GetString(FlagPassword),
		},
		noDriver:             new(viper.GetBool(FlagNoDriver)),
		taskOpTimeout:        viper.GetDuration(FlagTaskOpTimeout),
		taskPollInterval:     viper.GetDuration(FlagTaskPollInterval),
		taskShutdownTimeout:  viper.GetDuration(FlagTaskShutdownTimeout),
		taskWorkerCount:      viper.GetUint8(FlagTaskWorkerCount),
		certFactoryConfig:    factoryCfg,
		enabledIntegrations:  viper.GetStringSlice(FlagEnableIntegrations),
		disabledIntegrations: viper.GetStringSlice(FlagDisableIntegrations),
		validateChannelNames: new(!viper.GetBool(FlagDisableChannelNameValidation)),
	}, nil
}
