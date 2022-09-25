import {
  HTTPClientFactory,
  JSONEncoderDecoder,
  StreamClient,
  UnaryClient,
  URL,
  WebSocketClient,
} from '@synnaxlabs/freighter';

export default class Transport {
  url: URL;
  httpFactory: HTTPClientFactory;
  streamClient: StreamClient;

  constructor(url: URL) {
    this.url = url.child({ path: '/api/v1/' });
    this.httpFactory = new HTTPClientFactory(
      this.url,
      new JSONEncoderDecoder()
    );
    this.streamClient = new WebSocketClient(this.url, new JSONEncoderDecoder());
  }

  getClient(): UnaryClient {
    return this.httpFactory.getClient();
  }

  postClient(): UnaryClient {
    return this.httpFactory.postClient();
  }
}
