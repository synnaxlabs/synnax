// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import * as vscode from "vscode";
import * as path from "path";
import * as fs from "fs";
import {
  LanguageClient,
  LanguageClientOptions,
  ServerOptions,
  TransportKind,
} from "vscode-languageclient/node";

let client: LanguageClient;
let outputChannel: vscode.OutputChannel;

export function activate(context: vscode.ExtensionContext) {
  // Create output channel for logging
  outputChannel = vscode.window.createOutputChannel("Arc Language Server");
  outputChannel.show();
  outputChannel.appendLine("Arc Language extension is now active");
  outputChannel.appendLine(`Extension path: ${context.extensionPath}`);

  // Show activation message
  vscode.window.showInformationMessage("Arc Language extension activated!");

  // Get the LSP server path from configuration
  const config = vscode.workspace.getConfiguration("arc");
  let serverPath = config.get<string>("lsp.path", "arc");
  const verbose = config.get<boolean>("lsp.verbose", false);

  // Try to find the Arc binary in various locations
  const possiblePaths = [
    serverPath,
    path.join(context.extensionPath, "bin", "arc"),
    path.join(context.extensionPath, "..", "..", "..", "arc"),
    path.join(__dirname, "..", "..", "..", "..", "arc"),
    // Also check common installation locations
    "/usr/local/bin/arc",
    "/usr/bin/arc",
    path.join(process.env.HOME || "", "go", "bin", "arc"),
  ];

  outputChannel.appendLine("Searching for Arc binary in:");
  for (const p of possiblePaths) {
    const exists = fs.existsSync(p);
    outputChannel.appendLine(`  ${p}: ${exists ? "FOUND" : "not found"}`);
    if (exists) {
      serverPath = p;
      // Check if it's executable
      try {
        fs.accessSync(p, fs.constants.X_OK);
        outputChannel.appendLine(`  Using this binary (executable)`);
      } catch (err) {
        outputChannel.appendLine(`  WARNING: Binary exists but is not executable`);
      }
      break;
    }
  }

  outputChannel.appendLine(`Using Arc binary at: ${serverPath}`);

  // Check if the binary actually exists
  if (!fs.existsSync(serverPath)) {
    const errorMsg = `Arc binary not found at: ${serverPath}`;
    outputChannel.appendLine(`ERROR: ${errorMsg}`);
    vscode.window.showErrorMessage(errorMsg);
    return;
  }

  // Server options - call 'arc lsp' subcommand with stdio mode (default)
  const serverArgs: string[] = ["lsp"];
  // Note: The arc lsp command doesn't support -v flag currently

  outputChannel.appendLine(`Server command: ${serverPath} ${serverArgs.join(" ")}`);
  outputChannel.appendLine(`Verbose mode: ${verbose}`);

  const serverOptions: ServerOptions = {
    run: {
      command: serverPath,
      args: serverArgs,
      transport: TransportKind.stdio,
      options: {
        env: { ...process.env },
      },
    },
    debug: {
      command: serverPath,
      args: serverArgs,
      transport: TransportKind.stdio,
      options: {
        env: { ...process.env, SLATE_DEBUG: "1" },
      },
    },
  };

  // Client options
  const clientOptions: LanguageClientOptions = {
    documentSelector: [{ scheme: "file", language: "arc" }],
    synchronize: {
      fileEvents: vscode.workspace.createFileSystemWatcher("**/*.arc"),
    },
    outputChannel: outputChannel,
    traceOutputChannel: outputChannel,
  };

  // Create and start the language client
  client = new LanguageClient(
    "arc-lsp",
    "Arc Language Server",
    serverOptions,
    clientOptions,
  );

  // Start the client
  outputChannel.appendLine("Starting language client...");
  client.start().then(
    () => {
      outputChannel.appendLine("Arc LSP client started successfully!");
      vscode.window.showInformationMessage("Arc LSP client started successfully!");
    },
    (error) => {
      const errorMsg = `Failed to start Arc LSP: ${error}`;
      outputChannel.appendLine(`ERROR: ${errorMsg}`);
      outputChannel.appendLine(`Error details: ${JSON.stringify(error, null, 2)}`);
      vscode.window.showErrorMessage(errorMsg);
    },
  );

  // Register additional commands if needed
  const disposable = vscode.commands.registerCommand("arc.helloWorld", () => {
    vscode.window.showInformationMessage("Hello from Arc Language extension!");
  });

  context.subscriptions.push(disposable);
  context.subscriptions.push(outputChannel);
}

export function deactivate(): Thenable<void> | undefined {
  if (!client) {
    return undefined;
  }
  return client.stop();
}
