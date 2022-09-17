import axios, { AxiosRequestConfig } from 'axios';

import { EncoderDecoder } from './encoder';
import Endpoint from './endpoint';
import { decodeError, ErrorPayload } from './errors';
import { Payload } from './transport';

export default class HTTPClient {
  endpoint: Endpoint;
  encoder: EncoderDecoder;

  constructor(endpoint: Endpoint, encoder: EncoderDecoder) {
    this.endpoint = endpoint;
    this.encoder = encoder;
  }

  get(): GETClient {
    return new GETClient(this.endpoint, this.encoder);
  }

  post(): POSTClient {
    return new POSTClient(this.endpoint, this.encoder);
  }
}

class Core {
  endpoint: Endpoint;
  encoder: EncoderDecoder;

  constructor(endpoint: Endpoint, encoder: EncoderDecoder) {
    this.endpoint = endpoint.child({ protocol: 'http' });
    this.encoder = encoder;
  }

  get headers() {
    return {
      'Content-Type': this.encoder.contentType,
    };
  }

  requestConfig(): AxiosRequestConfig {
    return {
      headers: this.headers,
      responseType: 'arraybuffer',
    };
  }

  async execute<RS>(
    request: AxiosRequestConfig
  ): Promise<[RS | undefined, Error | undefined]> {
    try {
      const response = await axios.request(request);
      if (response.status !== 200) {
        const err = this.encoder.decode<ErrorPayload>(response.data);
        return [undefined, decodeError(err)];
      }

      const data = this.encoder.decode<RS>(response.data);
      return [data, undefined];
    } catch (err) {
      return [undefined, err as Error];
    }
  }
}

export class GETClient extends Core {
  async send<RQ extends Payload, RS extends Payload>(
    target: string,
    req: RQ
  ): Promise<[RS | undefined, Error | undefined]> {
    const queryString = buildQueryString(req as Record<string, unknown>);
    const request = this.requestConfig();
    request.method = 'GET';
    request.url = this.endpoint.path(target) + '?' + queryString;
    return await this.execute(request);
  }
}

export class POSTClient extends Core {
  async send<RQ extends Payload, RS extends Payload>(
    target: string,
    req: RQ
  ): Promise<[RS | undefined, Error | undefined]> {
    const url = this.endpoint.path(target);
    const request = this.requestConfig();
    request.method = 'POST';
    request.url = url;
    request.data = this.encoder.encode(req);
    return await this.execute(request);
  }
}

const buildQueryString = (request: Record<string, unknown>) => {
  const query = Object.keys(request)
    .map((key) => `${key}=${request[key]}`)
    .join('&');
  return query;
};
