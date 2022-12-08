import {
  HTTPClientFactory,
  JSONEncoderDecoder,
  Middleware,
  StreamClient,
  URL,
  UnaryClient,
  WebSocketClient,
} from '@synnaxlabs/freighter';

const baseAPIEndpoint = '/api/v1/';

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
    return this.httpFactory.getClient();
  }

  postClient(): UnaryClient {
    return this.httpFactory.postClient();
  }

  use(...middleware: Middleware[]) {
    this.httpFactory.use(...middleware);
    this.streamClient.use(...middleware);
  }
}
