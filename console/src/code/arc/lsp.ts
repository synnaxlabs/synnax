// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type arc, type Synnax } from "@synnaxlabs/client";
import { type Stream } from "@synnaxlabs/freighter";
import { type AsyncDestructor } from "@synnaxlabs/x";
import { MonacoLanguageClient } from "monaco-languageclient";
import { type Message, type MessageReader, type MessageWriter } from "vscode-jsonrpc";
import { CloseAction, ErrorAction } from "vscode-languageclient/browser";

import { type Extension } from "@/code/init/initialize";

const NOOP_DISPOSER = () => ({ dispose: () => {} });

export const LANGUAGE = "arc";

interface FreighterTransportProps {
  stream: Stream<typeof arc.lspMessageZ, typeof arc.lspMessageZ>;
}

const createFreighterTransport = ({
  stream,
}: FreighterTransportProps): {
  reader: MessageReader;
  writer: MessageWriter;
} => {
  let isClosed = false;
  let onCloseCallback: (() => void) | null = null;
  let onErrorCallback: ((error: Error) => void) | null = null;

  // Start receiving messages in the background
  const receiveLoop = async () => {
    try {
      while (!isClosed) {
        const [msg, err] = await stream.receive();
        if (err != null) {
          if (onErrorCallback != null) onErrorCallback(err);
          break;
        }
        if (msg == null) break;

        // Parse the raw JSON message
        try {
          const parsed = JSON.parse(msg.content);
          if (onMessageCallback != null) onMessageCallback(parsed);
        } catch (parseError) {
          if (onErrorCallback != null)
            onErrorCallback(
              parseError instanceof Error ? parseError : new Error(String(parseError)),
            );
        }
      }
    } finally {
      isClosed = true;
      if (onCloseCallback != null) onCloseCallback();
    }
  };

  let onMessageCallback: ((message: Message) => void) | null = null;

  const reader: MessageReader = {
    listen: (callback) => {
      onMessageCallback = callback as (message: Message) => void;
      // Start the receive loop
      receiveLoop().catch((err) => {
        if (onErrorCallback != null) onErrorCallback(err);
      });
      return {
        dispose: () => {
          onMessageCallback = null;
        },
      };
    },
    dispose: () => {
      isClosed = true;
    },
    onError: (callback) => {
      onErrorCallback = callback;
      return {
        dispose: () => {
          onErrorCallback = null;
        },
      };
    },
    onClose: (callback) => {
      onCloseCallback = callback;
      return {
        dispose: () => {
          onCloseCallback = null;
        },
      };
    },
    onPartialMessage: NOOP_DISPOSER,
  };

  const writer: MessageWriter = {
    write: async (message) => {
      if (isClosed) throw new Error("Stream is closed");
      // Send raw JSON without Content-Length headers
      stream.send({ content: JSON.stringify(message) });
    },
    dispose: () => {
      isClosed = true;
    },
    onError: (callback) => {
      const wrappedCallback = (err: Error) => callback([err, undefined, undefined]);
      onErrorCallback = wrappedCallback;
      return {
        dispose: () => {
          onErrorCallback = null;
        },
      };
    },
    onClose: (callback) => {
      onCloseCallback = callback;
      return {
        dispose: () => {
          onCloseCallback = null;
        },
      };
    },
    end: () => {
      isClosed = true;
      stream.closeSend();
    },
  };

  return { reader, writer };
};

let synnaxClient: Synnax | null = null;

export const setSynnaxClient = (client: Synnax) => {
  synnaxClient = client;
};

const startArcLSP = async (): Promise<AsyncDestructor> => {
  if (synnaxClient == null) {
    console.warn("Synnax client not set, Arc LSP will not start");
    return async () => {};
  }

  try {
    // Open LSP stream
    const stream = await synnaxClient.arcs.openLSP();

    // Create Freighter-based transport
    const { reader, writer } = createFreighterTransport({ stream });

    // Create Monaco language client
    const languageClient = new MonacoLanguageClient({
      name: "Arc Language Server",
      clientOptions: {
        documentSelector: [LANGUAGE],
        errorHandler: {
          error: () => ({ action: ErrorAction.Continue }),
          closed: () => ({ action: CloseAction.DoNotRestart }),
        },
      },
      messageTransports: { reader, writer },
    });

    await languageClient.start();

    return async () => {
      await languageClient.stop();
      stream.closeSend();
    };
  } catch (error) {
    console.error("Failed to start Arc LSP:", error);
    return async () => {};
  }
};

const registerArcLanguage = async (): Promise<AsyncDestructor> => {
  const monaco = await import("monaco-editor");

  // Register the language ID but skip Monarch tokenization
  // Monarch doesn't work with VSCode services - theme isn't initialized properly
  // The LSP will provide all language features (diagnostics, hover, autocomplete)
  monaco.languages.register({ id: LANGUAGE });

  return async () => {};
};

export const EXTENSIONS: Extension[] = [];

export const SERVICES = [registerArcLanguage, startArcLSP];
