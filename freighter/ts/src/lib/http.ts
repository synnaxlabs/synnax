import axios, { AxiosRequestConfig } from 'axios';
import { ZodSchema } from 'zod';

import { EncoderDecoder } from './encoder';
import { decodeError, ErrorPayloadSchema } from './errors';
import { UnaryClient } from './unary';
import URL from './url';

export class HTTPClientFactory {
  endpoint: URL;
  encoder: EncoderDecoder;

  constructor(endpoint: URL, encoder: EncoderDecoder) {
    this.endpoint = endpoint;
    this.encoder = encoder;
  }

  getClient(): GETClient {
    return new GETClient(this.endpoint, this.encoder);
  }

  postClient(): POSTClient {
    return new POSTClient(this.endpoint, this.encoder);
  }
}

class Core {
  endpoint: URL;
  encoder: EncoderDecoder;

  private static ERROR_ENCODING_HEADER_KEY = 'Error-Encoding';
  private static ERROR_ENCODING_HEADER_VALUE = 'freighter';
  private static CONTENT_TYPE_HEADER_KEY = 'Content-Type';

  constructor(endpoint: URL, encoder: EncoderDecoder) {
    this.endpoint = endpoint.child({ protocol: 'http' });
    this.encoder = encoder;
  }

  get headers() {
    return {
      [Core.CONTENT_TYPE_HEADER_KEY]: this.encoder.contentType,
      [Core.ERROR_ENCODING_HEADER_KEY]: Core.ERROR_ENCODING_HEADER_VALUE,
    };
  }

  requestConfig(): AxiosRequestConfig {
    return {
      headers: this.headers,
      responseType: 'arraybuffer',
      withCredentials: false,
      validateStatus: () => true,
    };
  }

  async execute<RS>(
    request: AxiosRequestConfig,
    resSchema: ZodSchema<RS>
  ): Promise<[RS | undefined, Error | undefined]> {
    try {
      const response = await axios.request(request);
      if (response.status < 200 || response.status >= 300) {
        const err = ErrorPayloadSchema.parse(
          this.encoder.decode(response.data)
        );
        return [undefined, decodeError(err)];
      }
      const data = resSchema.parse(this.encoder.decode(response.data));
      return [data, undefined];
    } catch (err) {
      return [undefined, err as Error];
    }
  }
}

export class GETClient extends Core implements UnaryClient {
  async send<RQ, RS>(
    target: string,
    req: RQ,
    resSchema: ZodSchema<RS>
  ): Promise<[RS | undefined, Error | undefined]> {
    const queryString = buildQueryString(req as Record<string, unknown>);
    const request = this.requestConfig();
    request.method = 'GET';
    request.url = this.endpoint.path(target) + '?' + queryString;
    return await this.execute(request, resSchema);
  }
}

export class POSTClient extends Core implements UnaryClient {
  async send<RQ, RS>(
    target: string,
    req: RQ,
    resSchema: ZodSchema<RS>
  ): Promise<[RS | undefined, Error | undefined]> {
    const url = this.endpoint.path(target);
    const request = this.requestConfig();
    request.method = 'POST';
    request.url = url;
    request.data = this.encoder.encode(req);
    return await this.execute(request, resSchema);
  }
}

const buildQueryString = (request: Record<string, unknown>) => {
  return Object.keys(request)
    .map((key) => `${key}=${request[key]}`)
    .join('&');
};
