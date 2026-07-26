// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type FileTransport,
  HTTPClient,
  type Middleware,
  type UnaryClient,
  unaryWithBreaker,
  WebSocketClient,
} from "@synnaxlabs/freighter";
import { binary, type breaker, type url } from "@synnaxlabs/x";

export class Transport {
  readonly url: url.URL;
  readonly unary: UnaryClient;
  /**
   * The unary client without the breaker's retry wrapper, for callers that
   * own their retry policy (the connection prober). Shares middleware with
   * {@link unary}.
   */
  readonly unaryNoRetry: UnaryClient;
  readonly file: FileTransport;
  readonly stream: WebSocketClient;
  readonly secure: boolean;

  constructor(url: url.URL, breakerCfg: breaker.Config = {}, secure: boolean = false) {
    this.secure = secure;
    this.url = url.child("/api/v1/");
    const codec = new binary.JSONCodec();
    // The streaming file client and the unary client share one HTTPClient, so
    // middleware (e.g. auth) registered through unary applies to file too. The file
    // client deliberately skips the breaker's retry wrapper: an upload body may be a
    // one-shot ReadableStream that cannot be re-sent on a retry.
    const http = new HTTPClient(this.url, codec, this.secure);
    this.unary = unaryWithBreaker(http, breakerCfg);
    this.unaryNoRetry = http;
    this.file = http;
    this.stream = new WebSocketClient(this.url, codec, this.secure);
  }

  use(...middleware: Middleware[]): void {
    this.unary.use(...middleware);
    this.stream.use(...middleware);
  }
}
