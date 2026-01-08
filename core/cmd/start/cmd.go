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
	"os"
	"os/signal"

	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/cmd/cert"
	"github.com/synnaxlabs/synnax/cmd/instrumentation"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	xsignal "github.com/synnaxlabs/x/signal"
	"go.uber.org/zap"
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:     "start",
	Short:   "Starts a Synnax Core",
	Long:    "Starts a Synnax Core using the data directory specified by the --data flag, and listening on the address specified by the --listen flag. If --peers is specified and no existing data is found, the Core will attempt to join the cluster formed by its peers. If no peers are specified and no existing data is found, the Core will bootstrap a new cluster.",
	Example: "synnax start --listen localhost:9091 --data /mnt/ssd1 --peers localhost:9092,localhost:9093 --insecure",
	Args:    cobra.NoArgs,
	Run:     func(cmd *cobra.Command, _ []string) { start(cmd) },
}

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

	cfg := GetCoreConfigFromViper(ins)

	sCtx.Go(func(ctx context.Context) error {
		return BootupCore(ctx, func() {}, cfg)
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

// AddCommand adds the start command to the given parent command.
func AddCommand(cmd *cobra.Command) error {
	BindFlags(startCmd)
	cmd.AddCommand(startCmd)
	return viper.BindPFlags(startCmd.Flags())
}

// GetCoreConfigFromViper builds a CoreConfig from the current viper configuration.
// This is used by the Windows service to start the Core with the config loaded from
// a YAML file.
func GetCoreConfigFromViper(ins alamos.Instrumentation) CoreConfig {
	listenAddress := address.Address(viper.GetString(FlagListen))
	peers := lo.Map(viper.GetStringSlice(FlagPeers), func(peer string, _ int) address.Address {
		return address.Address(peer)
	})
	return CoreConfig{
		Instrumentation:              ins,
		insecure:                     config.Bool(viper.GetBool(FlagInsecure)),
		debug:                        config.Bool(viper.GetBool(instrumentation.FlagDebug)),
		autoCert:                     config.Bool(viper.GetBool(FlagAutoCert)),
		verifier:                     viper.GetString(FlagDecoded),
		memBacked:                    config.Bool(viper.GetBool(FlagMem)),
		listenAddress:                listenAddress,
		peers:                        peers,
		dataPath:                     viper.GetString(FlagData),
		slowConsumerTimeout:          viper.GetDuration(FlagSlowConsumerTimeout),
		rootUsername:                 viper.GetString(FlagUsername),
		rootPassword:                 viper.GetString(FlagPassword),
		noDriver:                     config.Bool(viper.GetBool(FlagNoDriver)),
		taskOpTimeout:                viper.GetDuration(FlagTaskOpTimeout),
		taskPollInterval:             viper.GetDuration(FlagTaskPollInterval),
		taskShutdownTimeout:          viper.GetDuration(FlagTaskShutdownTimeout),
		taskWorkerCount:              viper.GetUint8(FlagTaskWorkerCount),
		certFactoryConfig:            cert.BuildCertFactoryConfig(ins, listenAddress),
		enabledIntegrations:          viper.GetStringSlice(FlagEnableIntegrations),
		disabledIntegrations:         viper.GetStringSlice(FlagDisableIntegrations),
		disableChannelNameValidation: config.Bool(viper.GetBool(FlagDisableChannelNameValidation)),
	}
}
