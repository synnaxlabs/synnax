import {
  Endpoint,
  MsgPackEncoderDecoder,
  StreamClient,
  WebSocketClient,
  HTTPClient,
} from '@arya-analytics/freighter';

export default class Transport {
  endpoint: Endpoint;
  stream: StreamClient;
  http: HTTPClient;

  constructor(endpoint: Endpoint) {
    this.endpoint = endpoint;
    this.stream = new WebSocketClient(new MsgPackEncoderDecoder(), endpoint);
    this.http = new HTTPClient(endpoint, new MsgPackEncoderDecoder());
  }
}
