// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status, type Synnax } from "@synnaxlabs/client";
import { EOF } from "@synnaxlabs/freighter";
import { breaker, errors, TimeSpan } from "@synnaxlabs/x";
import { useEffect, useRef } from "react";
import { type Message, type MessageReader, type MessageWriter } from "vscode-jsonrpc";
import {
  type BaseLanguageClient,
  type LanguageClientOptions,
  type MessageTransports,
} from "vscode-languageclient/browser";

/** LSPMessage is a single framed LSP message carried over the transport. */
export interface LSPMessage {
  content: string;
}

/** LSPStream is the bidirectional message stream a language server is reached through. It
 * is structurally a freighter stream of LSPMessages, so a concrete
 * `Stream<lspMessageZ, lspMessageZ>` satisfies it without code depending on the schema. */
export interface LSPStream {
  receive: () => Promise<LSPMessage>;
  send: (req: LSPMessage) => void;
  closeSend: () => void;
}

const NOOP_DISPOSER = () => ({ dispose: () => {} });

interface FreighterTransportProps {
  stream: LSPStream;
}

const createFreighterTransport = ({
  stream,
}: FreighterTransportProps): {
  reader: MessageReader;
  writer: MessageWriter;
  closed: Promise<void>;
} => {
  let isClosed = false;
  let onCloseCallback: (() => void) | null = null;
  let onErrorCallback: ((error: Error) => void) | null = null;
  let onMessageCallback: ((message: Message) => void) | null = null;

  let resolveClosed: () => void;
  const closed = new Promise<void>((r) => (resolveClosed = r));

  const receiveLoop = async () => {
    try {
      while (!isClosed) {
        let msg: LSPMessage;
        try {
          msg = await stream.receive();
        } catch (err) {
          if (!EOF.matches(err)) onErrorCallback?.(errors.fromUnknown(err));
          break;
        }
        try {
          const parsed = JSON.parse(msg.content);
          onMessageCallback?.(parsed);
        } catch (parseError) {
          onErrorCallback?.(errors.fromUnknown(parseError));
        }
      }
    } finally {
      isClosed = true;
      onCloseCallback?.();
      resolveClosed();
    }
  };

  const reader: MessageReader = {
    listen: (callback) => {
      onMessageCallback = callback;
      receiveLoop().catch((err: unknown) => onErrorCallback?.(errors.fromUnknown(err)));
      return { dispose: () => (onMessageCallback = null) };
    },
    dispose: () => (isClosed = true),
    onError: (callback) => {
      onErrorCallback = callback;
      return { dispose: () => (onErrorCallback = null) };
    },
    onClose: (callback) => {
      onCloseCallback = callback;
      return { dispose: () => (onCloseCallback = null) };
    },
    onPartialMessage: NOOP_DISPOSER,
  };

  const writer: MessageWriter = {
    write: async (message) => {
      if (isClosed) throw new Error("Stream is closed");
      stream.send({ content: JSON.stringify(message) });
    },
    dispose: () => (isClosed = true),
    onError: (callback) => {
      onErrorCallback = (err: Error) => callback([err, undefined, undefined]);
      return { dispose: () => (onErrorCallback = null) };
    },
    onClose: (callback) => {
      onCloseCallback = callback;
      return { dispose: () => (onCloseCallback = null) };
    },
    end: () => {
      isClosed = true;
      stream.closeSend();
    },
  };

  return { reader, writer, closed };
};

type LanguageClientConstructor = new (
  name: string,
  clientOptions: LanguageClientOptions,
  transports: MessageTransports,
) => BaseLanguageClient;

let languageClientConstructor: LanguageClientConstructor | null = null;

// loadLanguageClientConstructor lazily imports vscode-languageclient and builds the
// subclass on first use. The import is deferred so that merely importing this module does
// not pull in the browser-only language client and its monaco dependencies.
const loadLanguageClientConstructor = async (): Promise<LanguageClientConstructor> => {
  if (languageClientConstructor != null) return languageClientConstructor;
  const { BaseLanguageClient } = await import("vscode-languageclient/browser");
  class LanguageClient extends BaseLanguageClient {
    private readonly transports: MessageTransports;

    constructor(
      name: string,
      clientOptions: LanguageClientOptions,
      transports: MessageTransports,
    ) {
      super(name.toLowerCase(), name, clientOptions);
      this.transports = transports;
    }

    protected createMessageTransports(): Promise<MessageTransports> {
      return Promise.resolve(this.transports);
    }
  }
  return (languageClientConstructor = LanguageClient);
};

export interface LSPClientHandle {
  client: BaseLanguageClient;
  closed: Promise<void>;
}

const startLSPClient = async (
  stream: LSPStream,
  languageID: string,
): Promise<LSPClientHandle> => {
  const { reader, writer, closed } = createFreighterTransport({ stream });
  const [LanguageClient, { CloseAction, ErrorAction }] = await Promise.all([
    loadLanguageClientConstructor(),
    import("vscode-languageclient/browser"),
  ]);
  const client = new LanguageClient(
    "Language Server",
    {
      documentSelector: [languageID],
      errorHandler: {
        error: () => ({ action: ErrorAction.Continue }),
        closed: () => ({ action: CloseAction.DoNotRestart }),
      },
    },
    { reader, writer },
  );
  await client.start();
  return { client, closed };
};

const stopLSPClient = async (client: BaseLanguageClient): Promise<void> => {
  try {
    await client.stop();
  } catch {
    // Client may already be stopped if the stream died.
  }
};

const LSP_BREAKER_CONFIG: breaker.Config = {
  baseInterval: TimeSpan.seconds(1),
  maxRetries: 50,
  scale: 1.5,
};

const CONNECTING_STATUS = (): status.Status =>
  status.create({ variant: "loading", message: "Connecting to language server" });
const CONNECTED_STATUS = (): status.Status =>
  status.create({ variant: "success", message: "Language server connected" });
const INACTIVE_STATUS = (): status.Status =>
  status.create({ variant: "disabled", message: "Language server inactive" });

export interface UseLanguageServerArgs {
  /** monaco gates connection: the server only starts once monaco has initialized. */
  monaco: unknown;
  /** client is the cluster connection the server stream is opened against. */
  client: Synnax | null;
  /** languageID is the document selector the client subscribes to. */
  languageID: string;
  /** open opens the LSP message stream against the cluster. */
  open: (client: Synnax) => Promise<LSPStream>;
  /** onStatus reports connection state transitions as status.Status values. */
  onStatus: (status: status.Status) => void;
}

/** useLanguageServer runs a reconnecting language client for the lifetime of the calling
 * component, opening the stream via open and reporting connection state through onStatus.
 * It is a no-op (and reports a disabled status) until both monaco and client are ready.
 * Connection failures never throw into render — they surface only through onStatus. */
export const useLanguageServer = ({
  monaco,
  client,
  languageID,
  open,
  onStatus,
}: UseLanguageServerArgs): void => {
  const onStatusRef = useRef(onStatus);
  onStatusRef.current = onStatus;
  useEffect(() => {
    if (monaco == null || client == null) {
      onStatusRef.current(INACTIVE_STATUS());
      return;
    }
    const abortController = new AbortController();
    const { signal } = abortController;
    const abortPromise = new Promise<void>((r) =>
      signal.addEventListener("abort", () => r()),
    );
    let currentHandle: LSPClientHandle | null = null;
    let currentStream: LSPStream | null = null;
    const run = async () => {
      const b = new breaker.Breaker(LSP_BREAKER_CONFIG);
      while (!signal.aborted) {
        onStatusRef.current(CONNECTING_STATUS());
        try {
          const stream = await open(client);
          if (signal.aborted) {
            stream.closeSend();
            return;
          }
          currentStream = stream;
          const handle = await startLSPClient(stream, languageID);
          if (signal.aborted) {
            await stopLSPClient(handle.client);
            stream.closeSend();
            return;
          }
          currentHandle = handle;
          b.reset();
          onStatusRef.current(CONNECTED_STATUS());
          await Promise.race([handle.closed, abortPromise]);
          currentHandle = null;
          currentStream = null;
          if (signal.aborted) return;
        } catch (e) {
          onStatusRef.current(
            status.fromException(e, "Language server connection failed"),
          );
          currentHandle = null;
          currentStream = null;
        }
        if (!(await b.wait())) {
          onStatusRef.current(
            status.fromException(
              new Error("Language server unreachable after repeated attempts"),
            ),
          );
          return;
        }
      }
    };
    run().catch((err: unknown) => {
      console.error("language server connection loop threw", err);
    });
    return () => {
      abortController.abort();
      if (currentHandle != null)
        stopLSPClient(currentHandle.client).catch((err: unknown) => {
          console.error("failed to stop language server client", err);
        });
      currentStream?.closeSend();
      onStatusRef.current(INACTIVE_STATUS());
    };
  }, [client, monaco, languageID, open]);
};
