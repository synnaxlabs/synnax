import axios, { AxiosRequestConfig } from 'axios';
import { z } from 'zod';

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

  get<RQ, RS>(
    reqSchema: z.ZodSchema<RQ>,
    resSchema: z.ZodSchema<RS>
  ): GETClient<RQ, RS> {
    return new GETClient<RQ, RS>(
      this.endpoint,
      this.encoder,
      reqSchema,
      resSchema
    );
  }

  post<RQ, RS>(
    reqSchema: z.ZodSchema<RQ>,
    resSchema: z.ZodSchema<RS>
  ): POSTClient<RQ, RS> {
    return new POSTClient<RQ, RS>(
      this.endpoint,
      this.encoder,
      reqSchema,
      resSchema
    );
  }
}

class Core<RQ, RS> {
  endpoint: URL;
  encoder: EncoderDecoder;
  reqSchema: z.ZodSchema<RQ>;
  resSchema: z.ZodSchema<RS>;

  constructor(
    endpoint: URL,
    encoder: EncoderDecoder,
    reqSchema: z.ZodSchema<RQ>,
    resSchema: z.ZodSchema<RS>
  ) {
    this.endpoint = endpoint.child({ protocol: 'http' });
    this.encoder = encoder;
    this.reqSchema = reqSchema;
    this.resSchema = resSchema;
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

  async execute(
    request: AxiosRequestConfig
  ): Promise<[RS | undefined, Error | undefined]> {
    try {
      const response = await axios.request(request);
      if (response.status !== 200) {
        const err = ErrorPayloadSchema.parse(
          this.encoder.decode(response.data)
        );
        return [undefined, decodeError(err)];
      }
      const data = this.resSchema.parse(this.encoder.decode(response.data));
      return [data, undefined];
    } catch (err) {
      return [undefined, err as Error];
    }
  }
}

export class GETClient<RQ, RS>
  extends Core<RQ, RS>
  implements UnaryClient<RQ, RS>
{
  async send(
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

export class POSTClient<RQ, RS>
  extends Core<RQ, RS>
  implements UnaryClient<RQ, RS>
{
  async send(
    target: string,
    req: RQ
  ): Promise<[RS | undefined, Error | undefined]> {
    const url = this.endpoint.path(target);
    const request = this.requestConfig();
    request.method = 'POST';
    request.url = url;
    request.data = this.encoder.encode(this.reqSchema.parse(req));
    return await this.execute(request);
  }
}

const buildQueryString = (request: Record<string, unknown>) => {
  return Object.keys(request)
    .map((key) => `${key}=${request[key]}`)
    .join('&');
};
