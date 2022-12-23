import {
  HTTPClientFactory,
  JSONEncoderDecoder,
  URL,
  WebSocketClient,
} from "@synnaxlabs/freighter";
import type { Middleware, StreamClient, UnaryClient } from "@synnaxlabs/freighter";

const baseAPIEndpoint = "/api/v1/";

export default class Transport {
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
