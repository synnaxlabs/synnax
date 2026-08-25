// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package doctor implements the CLI command that inspects stored Core data.
package doctor

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/cmd/start"
	"github.com/synnaxlabs/synnax/pkg/doctor"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ErrProblems is returned when a run found at least one error-level problem. The CLI
// maps it to its own exit code, so scripts can tell a broken store from a failed run.
var ErrProblems = errors.New("errors found in stored data")

// Cmd inspects the data a Core stored on disk.
var Cmd = &cobra.Command{
	Use:   "doctor",
	Short: "Inspects the data a Core stored on disk",
	Long: "Inspects the data directory specified by the --data flag and reports " +
		"problems with the stored data: corrupt or unreadable telemetry, storage " +
		"the garbage collector never reclaimed, implausible timestamps, and " +
		"metadata that references entities which no longer exist. The command " +
		"never writes to the directory. A Core running against the directory " +
		"holds the key-value store open, so only the telemetry checks run.",
	Example:      "synnax doctor --data /mnt/ssd1",
	Args:         cobra.NoArgs,
	SilenceUsage: true,
	PreRunE: func(cmd *cobra.Command, _ []string) error {
		return viper.BindPFlags(cmd.Flags())
	},
	RunE: func(cmd *cobra.Command, _ []string) error { return run(cmd) },
}

func init() { AddFlags(Cmd) }

// run inspects the configured directory and writes the report to the command's output.
func run(cmd *cobra.Command) error {
	var (
		out     = cmd.OutOrStdout()
		asJSON  = viper.GetBool(FlagJSON)
		verbose = viper.GetBool(FlagVerbose)
	)
	ins, err := newInstrumentation(cmd, verbose)
	if err != nil {
		return err
	}
	cfg := doctor.Config{
		Instrumentation: ins,
		Dirname:         viper.GetString(start.FlagData),
		Deep:            new(!viper.GetBool(FlagSkipDeep)),
		KVDisabled:      viper.GetBool(FlagSkipKV),
		TSDisabled:      viper.GetBool(FlagSkipTS),
		Channels:        channelKeys(),
	}
	if !asJSON {
		cfg.Progress = func(phase doctor.Phase, done, total int) {
			fmt.Fprintf(cmd.ErrOrStderr(), "\rinspecting %s %d/%d", phase, done, total)
		}
	}
	report, err := doctor.Run(cmd.Context(), cfg)
	if err != nil {
		return err
	}
	if cfg.Progress != nil {
		fmt.Fprintln(cmd.ErrOrStderr())
	}
	if asJSON {
		err = doctor.RenderJSON(out, report)
	} else {
		err = doctor.Render(out, report, verbose)
	}
	if err != nil {
		return err
	}
	if report.Errors() > 0 {
		return ErrProblems
	}
	return nil
}

// newInstrumentation builds a logger writing to the command's error stream, so the
// report keeps standard output to itself. Only warnings and above are logged unless
// the run is verbose.
func newInstrumentation(
	cmd *cobra.Command,
	verbose bool,
) (alamos.Instrumentation, error) {
	encoder := zap.NewDevelopmentEncoderConfig()
	encoder.EncodeLevel = zapcore.CapitalColorLevelEncoder
	level := zap.WarnLevel
	if verbose {
		level = zap.InfoLevel
	}
	logger, err := alamos.NewLogger(alamos.LoggerConfig{
		ZapLogger: zap.New(zapcore.NewCore(
			zapcore.NewConsoleEncoder(encoder),
			zapcore.AddSync(cmd.ErrOrStderr()),
			level,
		)),
	})
	if err != nil {
		return alamos.Instrumentation{}, err
	}
	return alamos.New("doctor", alamos.WithLogger(logger)), nil
}

// channelKeys reads the channel filter from the configuration.
func channelKeys() []channel.Key {
	raw := viper.GetIntSlice(FlagChannels)
	keys := make([]channel.Key, len(raw))
	for i, k := range raw {
		keys[i] = channel.Key(k)
	}
	return keys
}
