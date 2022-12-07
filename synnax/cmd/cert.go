package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/synnaxlabs/synnax/pkg/security/cert"
	"github.com/synnaxlabs/x/address"
)

var certFactoryConfig = cert.FactoryConfig{}

var certCmd = &cobra.Command{
	Use:   "cert",
	Short: "Generate self-signed certificates for securing a Synnax cluster.",
	Args:  cobra.NoArgs,
}

var certCA = &cobra.Command{
	Use:   "ca",
	Short: "Generate a self-signed CA certificate.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		l, err := configureLogging()
		if err != nil {
			return err
		}
		certFactoryConfig.Logger = l.Sugar()
		certFactoryConfig.CertsDir = viper.GetString("certs-dir")
		factory, err := cert.NewFactory(certFactoryConfig)
		if err != nil {
			return err
		}
		return factory.CreateCAPair()
	},
}

var certNode = &cobra.Command{
	Use:   "node",
	Short: "Generate a self-signed node certificate.",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, hosts []string) error {
		// convert hosts to addresses
		addresses := make([]address.Address, len(hosts))
		for i, host := range hosts {
			addresses[i] = address.Address(host)
		}
		certFactoryConfig.Hosts = addresses
		certFactoryConfig.CertsDir = viper.GetString("certs-dir")
		l, err := configureLogging()
		if err != nil {
			return err
		}
		certFactoryConfig.Logger = l.Sugar()
		factory, err := cert.NewFactory(certFactoryConfig)
		if err != nil {
			return err
		}
		return factory.CreateNodePair()
	},
}

func init() {
	rootCmd.AddCommand(certCmd)

	certCmd.PersistentFlags().StringVar(&certFactoryConfig.CAKeyPath, "ca-key", "", "The path to the CA key.")
	certCmd.PersistentFlags().StringVar(&certFactoryConfig.CACertPath, "ca-cert", "", "The path to the CA certificate.")
	certCmd.PersistentFlags().StringVar(&certFactoryConfig.NodeKeyPath, "node-key", "", "The path to the node key.")
	certCmd.PersistentFlags().StringVar(&certFactoryConfig.NodeCertPath, "node-cert", "", "The path to the node certificate.")

	certCmd.AddCommand(certCA)
	certCmd.AddCommand(certNode)
}
