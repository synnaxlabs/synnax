// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package svc provides service management functionality for Synnax. This package
// encapsulates all service-related operations including installation, uninstallation,
// starting, and stopping. Currently, only Windows service is supported.
package service

// Config holds the configuration for installing the Synnax service. These settings are
// stored in the service configuration and used when the service starts.
type Config struct {
	// ListenAddress is the address to listen for client connections (e.g.,
	// "localhost:9090").
	ListenAddress string
	// DataDir is the directory where Synnax will store its data.
	DataDir string
	// Insecure disables encryption, authentication, and authorization.
	Insecure bool
	// Username is the username for the admin user.
	Username string
	// Password is the password for the admin user.
	Password string
	// AutoCert enables automatic generation of self-signed certificates.
	AutoCert bool
	// NoDriver disables the embedded Driver.
	NoDriver bool
	// Peers is a list of peer addresses in the cluster.
	Peers []string
	// EnableIntegrations is a list of device integrations to enable.
	EnableIntegrations []string
	// DisableIntegrations is a list of device integrations to disable.
	DisableIntegrations []string
	// AutoStart enables automatic service start on machine bootup.
	AutoStart bool
	// DelayedStart delays service start until after OS startup completes.
	DelayedStart bool
	// Verifier
	Verifier string
}
