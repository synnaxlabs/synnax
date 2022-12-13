package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func bindFlags(cmd *cobra.Command) {
	if err := viper.BindPFlags(cmd.PersistentFlags()); err != nil {
		zap.S().Error("failed to bind flags", zap.Error(err))
	}
	if err := viper.BindPFlags(cmd.Flags()); err != nil {
		zap.S().Error("failed to bind flags", zap.Error(err))
	}
}
