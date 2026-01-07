// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import * as path from "path";
import * as fs from "fs";
import * as os from "os";
import { ExtensionContext, workspace, window, OutputChannel } from "vscode";
import {
  LanguageClient,
  LanguageClientOptions,
  ServerOptions,
  TransportKind,
} from "vscode-languageclient/node";

let client: LanguageClient | undefined;
let outputChannel: OutputChannel;

export function activate(context: ExtensionContext): void {
  outputChannel = window.createOutputChannel("Oracle Language Server");
  outputChannel.appendLine("Activating Oracle Language extension...");

  const serverPath = findOracleBinary(context);
  if (!serverPath) {
    window.showErrorMessage(
      "Oracle CLI not found. Please install it or set oracle.lsp.path in settings.",
    );
    return;
  }

  outputChannel.appendLine(`Using Oracle binary: ${serverPath}`);

  const serverOptions: ServerOptions = {
    run: {
      command: serverPath,
      args: ["lsp"],
      transport: TransportKind.stdio,
    },
    debug: {
      command: serverPath,
      args: ["lsp"],
      transport: TransportKind.stdio,
    },
  };

  const clientOptions: LanguageClientOptions = {
    documentSelector: [{ scheme: "file", language: "oracle" }],
    synchronize: {
      fileEvents: workspace.createFileSystemWatcher("**/*.oracle"),
    },
    outputChannel,
  };

  client = new LanguageClient(
    "oracle-lsp",
    "Oracle Language Server",
    serverOptions,
    clientOptions,
  );

  client.start();
  outputChannel.appendLine("Oracle Language Server started.");
}

export function deactivate(): Thenable<void> | undefined {
  if (!client) {
    return undefined;
  }
  return client.stop();
}

function findOracleBinary(context: ExtensionContext): string | undefined {
  const config = workspace.getConfiguration("oracle.lsp");
  const configPath = config.get<string>("path");

  // Check configured path first
  if (configPath && fs.existsSync(configPath)) {
    return configPath;
  }

  // Search locations
  const searchPaths = [
    // Extension bin directory
    path.join(context.extensionPath, "bin", "oracle"),
    // Go bin directory
    path.join(os.homedir(), "go", "bin", "oracle"),
    // Common system locations
    "/usr/local/bin/oracle",
    "/usr/bin/oracle",
  ];

  // Add Windows-specific paths
  if (process.platform === "win32") {
    searchPaths.push(
      path.join(context.extensionPath, "bin", "oracle.exe"),
      path.join(os.homedir(), "go", "bin", "oracle.exe"),
    );
  }

  for (const searchPath of searchPaths) {
    if (fs.existsSync(searchPath)) {
      return searchPath;
    }
  }

  // Try finding in PATH
  const pathEnv = process.env.PATH || "";
  const pathSeparator = process.platform === "win32" ? ";" : ":";
  const extensions = process.platform === "win32" ? [".exe", ""] : [""];

  for (const dir of pathEnv.split(pathSeparator)) {
    for (const ext of extensions) {
      const fullPath = path.join(dir, `oracle${ext}`);
      if (fs.existsSync(fullPath)) {
        return fullPath;
      }
    }
  }

  return undefined;
}
