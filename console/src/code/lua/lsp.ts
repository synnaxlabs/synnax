// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type AsyncDestructor, jsonRPC } from "@synnaxlabs/x";
import { Command } from "@tauri-apps/plugin-shell";
import { MonacoLanguageClient } from "monaco-languageclient";
import { type MessageReader, type MessageWriter } from "vscode-jsonrpc";
import { CloseAction, ErrorAction } from "vscode-languageclient/browser";

import { type Extension } from "@/code/init/initialize";

const NOOP_DISPOSER = () => ({ dispose: () => {} });

const stringToError = (str: string): Error => new Error(str);

export const LANGUAGE = "lua";

export const startLuaLSP = async (): Promise<AsyncDestructor> => {
  const command = Command.create("lua-language-server");
  const child = await command.spawn();
  const reader: MessageReader = {
    listen: (callback) => {
      const decoder = jsonRPC.streamDecodeChunks(callback);
      command.stdout.on("data", decoder);
      return { dispose: () => command.stdout.off("data", decoder) };
    },
    dispose: () => {},
    onError: (callback) => {
      const bk = (str: string) => callback(stringToError(str));
      command.on("error", bk);
      return { dispose: () => command.off("error", bk) };
    },
    onClose: (callback) => {
      const bk = () => callback();
      command.on("close", bk);
      return { dispose: () => command.off("close", bk) };
    },
    onPartialMessage: NOOP_DISPOSER,
  };

  const writer: MessageWriter = {
    write: async (message) => {
      await child.write(jsonRPC.encodeMessage(message));
    },
    dispose: () => {},
    onError: (callback) => {
      const bk = (err: string) => callback([stringToError(err), undefined, undefined]);
      command.on("error", bk);
      return { dispose: () => command.off("error", bk) };
    },
    onClose: (callback) => {
      const bk = () => callback();
      command.on("close", bk);
      return { dispose: () => command.off("close", bk) };
    },
    end: () => {
      child.kill().catch(console.error);
    },
  };
  const languageClient = new MonacoLanguageClient({
    name: "Lua Language Server",
    clientOptions: {
      documentSelector: ["lua"],
      errorHandler: {
        error: () => ({ action: ErrorAction.Continue }),
        closed: () => ({ action: CloseAction.DoNotRestart }),
      },
    },
    messageTransports: { reader, writer },
  });
  await languageClient.start();
  return async () => await languageClient.stop();
};

export const EXTENSIONS: Extension[] = [
  async () => {
    await import("@codingame/monaco-vscode-lua-default-extension");
  },
];

export const SERVICES = [startLuaLSP];

export const stringifyVar = (name: string, value: string, docs?: string) => {
  if (docs == null) return `${name} = ${value}`;
  return `-- ${docs}\n${name} = ${value}`;
};
