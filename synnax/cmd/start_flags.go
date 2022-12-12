package cmd

func configureStartFlags() {
	startCmd.Flags().StringP(
		"listen",
		"l",
		"127.0.0.1:9090",
		`
			`,
	)

	startCmd.Flags().StringSliceP(
		"peers",
		"p",
		nil,
		"Addresses of additional peers in the cluster.",
	)

	startCmd.Flags().StringP(
		"data",
		"d",
		"synnax-data",
		"Dirname where the synnax node will store its data.",
	)

	startCmd.Flags().BoolP(
		"mem",
		"m",
		false,
		"Use in-memory storage",
	)

	startCmd.Flags().BoolP(
		"insecure",
		"i",
		false,
		"Disable encryption, authentication, and authorization.",
	)

	startCmd.Flags().String(
		"username",
		"synnax",
		"Username for the admin user.",
	)

	startCmd.Flags().String(
		"password",
		"seldon",
		"Password for the admin user.",
	)
}
