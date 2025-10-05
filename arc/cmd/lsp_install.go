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
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

//go:embed arc-language.vsix
var vsixFile []byte

var lspInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install arc language support for editors",
	Long:  `Install the arc Language Server extension for various editors`,
}

var vscodeInstallCmd = &cobra.Command{
	Use:   "vscode",
	Short: "Install arc extension for VS Code",
	Long:  `Install the arc Language Server extension for Visual Studio Code`,
	RunE:  runVSCodeInstall,
}

func init() {
	lspCmd.AddCommand(lspInstallCmd)
	lspInstallCmd.AddCommand(vscodeInstallCmd)
}

func runVSCodeInstall(cmd *cobra.Command, args []string) error {
	// Check if VS Code is installed
	codePath, err := exec.LookPath("code")
	if err != nil {
		return fmt.Errorf("VS Code CLI not found. Please ensure VS Code is installed and 'code' command is available in PATH")
	}

	// Write VSIX to temp file
	tempDir, err := os.MkdirTemp("", "arc-lsp-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	vsixPath := filepath.Join(tempDir, "arc-language.vsix")
	if err := os.WriteFile(vsixPath, vsixFile, 0644); err != nil {
		return fmt.Errorf("failed to write VSIX file: %w", err)
	}

	// Install the extension
	fmt.Println("Installing arc Language extension for VS Code...")
	installCmd := exec.Command(codePath, "--install-extension", vsixPath)
	installCmd.Stdout = os.Stdout
	installCmd.Stderr = os.Stderr

	if err := installCmd.Run(); err != nil {
		return fmt.Errorf("failed to install extension: %w", err)
	}

	fmt.Println("\n✓ arc Language extension installed successfully!")
	fmt.Println("Restart VS Code to activate the extension.")
	return nil
}
