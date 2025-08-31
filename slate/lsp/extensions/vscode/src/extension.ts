// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import * as vscode from 'vscode';
import * as path from 'path';
import * as fs from 'fs';
import {
  LanguageClient,
  LanguageClientOptions,
  ServerOptions,
  TransportKind
} from 'vscode-languageclient/node';

let client: LanguageClient;

export function activate(context: vscode.ExtensionContext) {
  console.log('Slate Language extension is now active');
  
  // Show activation message
  vscode.window.showInformationMessage('Slate Language extension activated!');

  // Get the LSP server path from configuration
  const config = vscode.workspace.getConfiguration('slate');
  let serverPath = config.get<string>('lsp.path', 'slate');
  const verbose = config.get<boolean>('lsp.verbose', false);

  // Try to find the Slate binary in various locations
  const possiblePaths = [
    serverPath,
    path.join(context.extensionPath, 'bin', 'slate'),
    path.join(context.extensionPath, '..', '..', '..', 'slate'),
    path.join(__dirname, '..', '..', '..', '..', 'slate'),
    // Also check common installation locations
    '/usr/local/bin/slate',
    '/usr/bin/slate',
    path.join(process.env.HOME || '', 'go', 'bin', 'slate'),
  ];

  for (const p of possiblePaths) {
    if (fs.existsSync(p)) {
      serverPath = p;
      break;
    }
  }

  console.log(`Using Slate binary at: ${serverPath}`);
  vscode.window.showInformationMessage(`Slate binary path: ${serverPath}`);

  // Server options - call 'slate lsp' subcommand with stdio mode (default)
  const serverArgs: string[] = ['lsp'];
  if (verbose) {
    serverArgs.push('-v');
  }

  const serverOptions: ServerOptions = {
    run: {
      command: serverPath,
      args: serverArgs,
      options: {
        env: { ...process.env }
      }
    },
    debug: {
      command: serverPath,
      args: [...serverArgs, '-v'],
      options: {
        env: { ...process.env, SLATE_DEBUG: '1' }
      }
    }
  };

  // Client options
  const clientOptions: LanguageClientOptions = {
    documentSelector: [{ scheme: 'file', language: 'slate' }],
    synchronize: {
      fileEvents: vscode.workspace.createFileSystemWatcher('**/*.slate')
    }
  };

  // Create and start the language client
  client = new LanguageClient(
    'slate-lsp',
    'Slate Language Server',
    serverOptions,
    clientOptions
  );

  // Start the client
  client.start().then(() => {
    vscode.window.showInformationMessage('Slate LSP client started successfully!');
  }, (error) => {
    vscode.window.showErrorMessage(`Failed to start Slate LSP: ${error}`);
    console.error('Failed to start LSP client:', error);
  });

  // Register additional commands if needed
  const disposable = vscode.commands.registerCommand('slate.helloWorld', () => {
    vscode.window.showInformationMessage('Hello from Slate Language extension!');
  });

  context.subscriptions.push(disposable);
}

export function deactivate(): Thenable<void> | undefined {
  if (!client) {
    return undefined;
  }
  return client.stop();
}