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
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/errors"
	xsignal "github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// Cmd represents the start command
var Cmd = &cobra.Command{
	Use:     "start",
	Short:   "Starts a Synnax Core",
	Long:    "Starts a Synnax Core using the data directory specified by the --data flag, and listening on the address specified by the --listen flag.",
	Example: "synnax start --listen localhost:9091 --data /mnt/ssd1 --insecure",
	Args:    cobra.NoArgs,
	PreRunE: func(cmd *cobra.Command, _ []string) error {
		return viper.BindPFlags(cmd.Flags())
	},
	RunE: func(cmd *cobra.Command, _ []string) error { return start(cmd) },
}

// start is the entrypoint for starting a Synnax Core. It handles stop requests and
// delegates to BootupCore for the actual startup. It logs the error that ends the Core
// and returns it.
func start(cmd *cobra.Command) error {
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	ctx := cmd.Context()
	ins := instrumentation.Configure()
	defer instrumentation.Cleanup(ctx, ins)

	// sigC holds two signals, so a second signal sent right after the first is kept.
	sigC := make(chan os.Signal, 2)
	signal.Notify(sigC, os.Interrupt, syscall.SIGTERM)

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
		ins.L.Error("invalid configuration", zap.Error(err))
		return err
	}

	sCtx.Go(func(ctx context.Context) error {
		return BootupCore(ctx, nil, cfg)
	}, xsignal.WithKey("start"), xsignal.RecoverWithErrOnPanic())

	forceCtx, forceCancel := xsignal.Isolated(xsignal.WithInstrumentation(ins))
	forceExit := xsignal.NewHardShutdown(forceCtx, forceCancel)
	// shutDown cancels the Core, then exits at once if signalsToForce more stop signals
	// arrive before the shutdown completes. Stdin requests never count: the Desktop
	// supervisor sends the stop keyword, then closes stdin.
	shutDown := func(signalsToForce int) {
		ins.L.Info(
			"\033[33mSynnax is shutting down. Press Ctrl+C again to exit now.\033[0m",
		)
		cancel()
		forceCtx.Go(func(ctx context.Context) error {
			for range signalsToForce {
				select {
				case <-sigC:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			ins.L.Fatal(
				"received a second stop signal, exiting before shutdown completes",
			)
			return nil
		}, xsignal.WithKey("force_exit"))
	}

	select {
	case <-sigC:
		shutDown(1)
	case <-stdinC:
		shutDown(2)
	case <-sCtx.Stopped():
	}

	err = sCtx.Wait()
	if fErr := forceExit.Close(); fErr != nil {
		ins.L.Fatal("failed to stop the force exit routine", zap.Error(fErr))
	}
	if err != nil && !errors.Is(err, context.Canceled) {
		ins.L.Zap().Sugar().Errorf(
			"\033[31mSynnax has encountered an error and is shutting down: %v\033[0m",
			err,
		)
		return err
	}
	ins.L.Info("\033[34mSynnax has shut down\033[0m")
	return nil
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
	factoryCfg := cert.BuildCertFactoryConfig(ins)
	listeners, err := listener.Parse()
	if err != nil {
		return CoreConfig{}, errors.Mark(err, validate.ErrValidation)
	}
	factoryCfg.Hosts = lo.Map(
		listeners,
		func(l listener.Config, _ int) address.Address {
			return l.Address
		},
	)
	licenseKey, err := readLicenseKey()
	if err != nil {
		return CoreConfig{}, errors.Mark(err, validate.ErrValidation)
	}
	var secretsKey secret.EncryptionKey
	if v := viper.GetString(FlagSecretsKey); v != "" {
		if secretsKey, err = secret.ParseEncryptionKey(v); err != nil {
			return CoreConfig{}, errors.Wrapf(err, "invalid --%s", FlagSecretsKey)
		}
	}
	return CoreConfig{
		Instrumentation:     ins,
		insecure:            new(viper.GetBool(FlagInsecure)),
		fips:                new(viper.GetBool(FlagFIPS)),
		debug:               new(viper.GetBool(instrumentation.FlagDebug)),
		autoCert:            new(viper.GetBool(cert.FlagAutoCert)),
		licenseKey:          licenseKey,
		memBacked:           new(viper.GetBool(FlagMem)),
		listeners:           listeners,
		dataPath:            viper.GetString(FlagData),
		secretsKey:          secretsKey,
		secretsKeyFile:      viper.GetString(FlagSecretsKeyFile),
		slowConsumerTimeout: viper.GetDuration(FlagSlowConsumerTimeout),
		sessionIdleTimeout:  viper.GetDuration(FlagSessionIdleTimeout),
		rootCredentials: auth.Credentials{
			Username: viper.GetString(FlagUsername),
			Password: viper.GetString(FlagPassword),
		},
		noDriver:             new(viper.GetBool(FlagNoDriver)),
		updateBackupDisabled: new(viper.GetBool(FlagNoUpdateBackup)),
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
