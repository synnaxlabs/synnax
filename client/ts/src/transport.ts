// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  HTTPClient,
  type Middleware,
  type StreamClient,
  type UnaryClient,
  unaryWithBreaker,
  WebSocketClient,
} from "@synnaxlabs/freighter";
import { breaker } from "@synnaxlabs/x";
import { binary } from "@synnaxlabs/x/binary";
import { type URL } from "@synnaxlabs/x/url";

const baseAPIEndpoint = "/api/v1/";

export class Transport {
  readonly url: URL;
  readonly unary: UnaryClient;
  readonly stream: StreamClient;
  readonly secure: boolean;

  constructor(url: URL, breakerCfg: breaker.Config = {}, secure: boolean = false) {
    this.secure = secure;
    this.url = url.child(baseAPIEndpoint);
    const codec = new binary.JSONCodec();
    (this.unary = unaryWithBreaker(
      new HTTPClient(this.url, codec, this.secure),
      breakerCfg,
    )),
      (this.stream = new WebSocketClient(this.url, codec, this.secure));
  }

  use(...middleware: Middleware[]): void {
    this.unary.use(...middleware);
    this.stream.use(...middleware);
  }
}
