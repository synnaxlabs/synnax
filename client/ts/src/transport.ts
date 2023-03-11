// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  HTTPClientFactory,
  JSONEncoderDecoder,
  WebSocketClient,
} from "@synnaxlabs/freighter";
import type { Middleware, StreamClient, UnaryClient } from "@synnaxlabs/freighter";
import { URL } from "@synnaxlabs/x";

const baseAPIEndpoint = "/api/v1/";

export class Transport {
  url: URL;
  httpFactory: HTTPClientFactory;
  streamClient: StreamClient;
  secure: boolean;

  constructor(url: URL, secure: boolean = false) {
    this.secure = secure;
    this.url = url.child(baseAPIEndpoint);
    const ecd = new JSONEncoderDecoder();
    this.httpFactory = new HTTPClientFactory(this.url, ecd, this.secure);
    this.streamClient = new WebSocketClient(this.url, ecd, this.secure);
  }

  getClient(): UnaryClient {
    return this.httpFactory.newGET();
  }

  postClient(): UnaryClient {
    return this.httpFactory.newPOST();
  }

  use(...middleware: Middleware[]): void {
    this.httpFactory.use(...middleware);
    this.streamClient.use(...middleware);
  }
}
