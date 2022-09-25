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
    this.url = url.child('/api/v1/');
    this.httpFactory = new HTTPClientFactory(
      this.url,
      new JSONEncoderDecoder()
    );
    this.streamClient = new WebSocketClient(new JSONEncoderDecoder(), this.url);
  }

  getClient(): UnaryClient {
    return this.httpFactory.getClient();
  }

  postClient(): UnaryClient {
    return this.httpFactory.postClient();
  }
}
