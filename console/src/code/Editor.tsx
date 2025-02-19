// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/code/Editor.css";
import "@codingame/monaco-vscode-lua-default-extension";
import "@codingame/monaco-vscode-python-default-extension";
import "@codingame/monaco-vscode-theme-defaults-default-extension";
import "vscode/localExtensionHost";

import { initialize } from "@codingame/monaco-vscode-api";
import getLanguagesServiceOverride from "@codingame/monaco-vscode-languages-service-override";
import getTextMateServiceOverride from "@codingame/monaco-vscode-textmate-service-override";
import getThemeServiceOverride from "@codingame/monaco-vscode-theme-service-override";
import { Align, type Input, Theming, useAsyncEffect } from "@synnaxlabs/pluto";
import * as monaco from "monaco-editor";
import { MonacoLanguageClient } from "monaco-languageclient";
import { useRef } from "react";
import {
  CloseAction,
  ErrorAction,
  type MessageTransports,
} from "vscode-languageclient/browser.js";
import {
  toSocket,
  WebSocketMessageReader,
  WebSocketMessageWriter,
} from "vscode-ws-jsonrpc";

import { CSS } from "@/css";

const loggingWebsocketWrapper = (inSocket: IWebSocket): IWebSocket => ({
  ...inSocket,
  send: (message) => {
    console.log("Sending message", message);
    inSocket.send(message);
  },
  onMessage: (listener) => {
    inSocket.onMessage((message) => {
      console.log("Message received", message);
      listener(message);
    });
  },
});
export const initWebSocketAndStartClient = async (url: string): Promise<WebSocket> => {
  const webSocket = new WebSocket(url);
  webSocket.onopen = () => {
    // creating messageTransport
    const socket = loggingWebsocketWrapper(toSocket(webSocket));
    const reader = new WebSocketMessageReader(socket);
    const writer = new WebSocketMessageWriter(socket);
    // creating language client
    const languageClient = createLanguageClient({
      reader,
      writer,
    });
    languageClient
      .start()
      .then(() => {
        console.log("Language client started");
      })
      .catch((err) => {
        console.error(err);
      });
    reader.onClose(() => languageClient.stop());
  };
  return webSocket;
};
const createLanguageClient = (
  messageTransports: MessageTransports,
): MonacoLanguageClient =>
  new MonacoLanguageClient({
    name: "Sample Language Client",
    clientOptions: {
      // use a language id as a document selector
      documentSelector: ["lua"],
      // disable the default error handler
      errorHandler: {
        error: () => ({ action: ErrorAction.Continue }),
        closed: () => ({ action: CloseAction.DoNotRestart }),
      },
    },
    // create a language client connection from the JSON RPC connection on demand
    messageTransports,
  });

export interface EditorProps
  extends Input.Control<string>,
    Omit<Align.SpaceProps, "value" | "onChange"> {}

export const Editor = ({ value, onChange, className, ...rest }: EditorProps) => {
  const editorRef = useRef<HTMLDivElement | null>(null);
  const monacoRef = useRef<monaco.editor.IStandaloneCodeEditor | null>(null);
  const theme = Theming.use();

  useAsyncEffect(async () => {
    if (!editorRef.current) return;

    // Configure Monaco web workers.
    const workerLoaders: Partial<Record<string, () => Worker>> = {
      TextEditorWorker: () =>
        new Worker(
          new URL("monaco-editor/esm/vs/editor/editor.worker.js", import.meta.url),
          { type: "module" },
        ),
      TextMateWorker: () =>
        new Worker(
          new URL(
            "@codingame/monaco-vscode-textmate-service-override/worker",
            import.meta.url,
          ),
          { type: "module" },
        ),
    };
    self.MonacoEnvironment = {
      getWorker: (_moduleId, label) => {
        const workerFactory = workerLoaders[label];
        if (workerFactory != null) return workerFactory();
        throw new Error(`Worker ${label} not found`);
      },
    };

    // Initialize Monaco services.
    await initialize({
      ...getTextMateServiceOverride(),
      ...getThemeServiceOverride(),
      ...getLanguagesServiceOverride(),
    });

    // Create the Monaco editor instance.
    monacoRef.current = monaco.editor.create(editorRef.current, {
      value,
      language: "lua",
      theme: "vs-dark",
      automaticLayout: true,
    });

    initWebSocketAndStartClient("ws://localhost:8080");

    // Update external state when the model changes.
    const dispose = monacoRef.current.onDidChangeModelContent(() => {
      if (monacoRef.current) onChange(monacoRef.current.getValue());
    });

    return () => {
      dispose.dispose();
      if (monacoRef.current) monacoRef.current.dispose();
    };
  }, [theme.key]);

  return (
    <Align.Space
      direction="y"
      grow
      {...rest}
      className={CSS(className, CSS.B("editor"))}
    >
      <div ref={editorRef} style={{ height: "100%", position: "relative" }} />
    </Align.Space>
  );
};
