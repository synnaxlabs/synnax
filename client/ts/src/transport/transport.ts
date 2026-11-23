// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type Connector,
  type FileTransport,
  HTTPClient,
  type Middleware,
  type UnaryClient,
  unaryWithBreaker,
  WebSocketClient,
} from "@synnaxlabs/freighter";
import { binary, type breaker, type url } from "@synnaxlabs/x";

/** Configures a {@link Transport}. */
export interface TransportConfig {
  /** The address of the Core. */
  url: url.URL;
  /** How the unary client retries a request that fails. Defaults to the breaker's. */
  retry?: breaker.Config;
  /** Whether to connect with TLS. Defaults to false. */
  secure?: boolean;
  /** Receives each failed attempt that the unary client retries. */
  onRetry?: (error: Error) => void;
  /** Opens the connections. Defaults to the global fetch and WebSocket. */
  connector?: Connector;
}

export class Transport {
  readonly url: url.URL;
  readonly unary: UnaryClient;
  /**
   * The unary client without the breaker's retry wrapper, for callers that own their
   * retry policy (the connection checker). Shares middleware with {@link unary}.
   */
  readonly unaryNoRetry: UnaryClient;
  readonly file: FileTransport;
  readonly stream: WebSocketClient;
  readonly secure: boolean;

  constructor({
    url,
    retry = {},
    secure = false,
    onRetry,
    connector,
  }: TransportConfig) {
    this.secure = secure;
    this.url = url.child("/api/v1/");
    const codec = new binary.JSONCodec();
    // The streaming file client and the unary client share one HTTPClient, so
    // middleware (e.g. auth) registered through unary applies to file too. The file
    // client deliberately skips the breaker's retry wrapper: an upload body may be a
    // one-shot ReadableStream that cannot be re-sent on a retry.
    const http = new HTTPClient({ endpoint: this.url, codec, secure, connector });
    this.unary = unaryWithBreaker(http, retry, onRetry);
    this.unaryNoRetry = http;
    this.file = http;
    this.stream = new WebSocketClient({ endpoint: this.url, codec, secure, connector });
  }

  use(...middleware: Middleware[]): void {
    this.unary.use(...middleware);
    this.stream.use(...middleware);
  }
}
