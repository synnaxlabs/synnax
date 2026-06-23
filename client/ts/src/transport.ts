// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type FileClient,
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
  readonly file: FileClient;
  readonly stream: WebSocketClient;
  readonly secure: boolean;

  constructor(url: url.URL, breakerCfg: breaker.Config = {}, secure: boolean = false) {
    this.secure = secure;
    this.url = url.child("/api/v1/");
    const codec = new binary.JSONCodec();
    // The streaming file client and the unary client share one HTTPClient, so
    // middleware (e.g. auth) registered through unary applies to file too. Streaming
    // bodies cannot be replayed, so the file client deliberately skips the breaker's
    // retry wrapper.
    const http = new HTTPClient(this.url, codec, this.secure);
    this.unary = unaryWithBreaker(http, breakerCfg);
    this.file = http;
    this.stream = new WebSocketClient(this.url, codec, this.secure);
  }

  use(...middleware: Middleware[]): void {
    this.unary.use(...middleware);
    this.stream.use(...middleware);
  }
}
