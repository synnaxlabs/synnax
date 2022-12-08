import axios, { AxiosRequestConfig, AxiosResponse } from 'axios';
import { ZodSchema } from 'zod';

import { EncoderDecoder } from './encoder';
import { ErrorPayloadSchema, decodeError } from './errors';
import { MetaData, MiddlewareCollector } from './middleware';
import { UnaryClient } from './unary';
import URL from './url';

/**
 * HTTPClientFactory provides a POST and GET implementation of the Unary
 * protocol.
 *
 * @param baseEndpointer - The base endpoint of the API.
 * @param encoder - The encoder/decoder to use for the request/response.
 * @param secure - Whether to use https or http.
 */
export class HTTPClientFactory extends MiddlewareCollector {
  baseEndpoint: URL;
  encoder: EncoderDecoder;
  secure: boolean;

  constructor(baseEndpoint: URL, encoder: EncoderDecoder, secure: boolean = false) {
    super();
    this.baseEndpoint = baseEndpoint;
    this.encoder = encoder;
    this.secure = secure;
  }

  /** @returns A UnaryClient that uses GET requests. */
  getClient(): GETClient {
    const gc = new GETClient(this.baseEndpoint, this.encoder, this.secure);
    gc.use(...this.middleware);
    return gc;
  }

  /** @returns a UnaryClient that uses POST requests. */
  postClient(): POSTClient {
    const pc = new POSTClient(this.baseEndpoint, this.encoder, this.secure);
    pc.use(...this.middleware);
    return pc;
  }
}

export const CONTENT_TYPE_HEADER_KEY = 'Content-Type';

class Core extends MiddlewareCollector {
  endpoint: URL;
  encoder: EncoderDecoder;

  constructor(endpoint: URL, encoder: EncoderDecoder, secure: boolean = false) {
    super();
    this.endpoint = endpoint.replace({ protocol: secure ? 'https' : 'http' });
    this.encoder = encoder;
  }

  get headers() {
    return {
      [CONTENT_TYPE_HEADER_KEY]: this.encoder.contentType,
    };
  }

  requestConfig(): AxiosRequestConfig {
    return {
      headers: this.headers,
      responseType: 'arraybuffer',
      withCredentials: false,
      validateStatus: () => true,
      data: null,
    };
  }

  async execute<RS>(
    request: AxiosRequestConfig,
    resSchema: ZodSchema<RS> | null
  ): Promise<[RS | undefined, Error | undefined]> {
    let res: RS | undefined = undefined;

    if (!request.url) throw new Error('[freighter.http] - expected valid request url');

    const [, err] = await this.executeMiddleware(
      { target: request.url, protocol: 'http', params: {} },
      async (md: MetaData): Promise<[MetaData, Error | undefined]> => {
        let outMD: MetaData = { ...md };
        request.headers = { ...request.headers, ...this.headers, ...md.params };
        let rawRes: AxiosResponse;
        try {
          rawRes = await axios.request(request);
        } catch (err) {
          return [outMD, err as Error];
        }
        outMD.params = rawRes.headers;
        if (rawRes.status < 200 || rawRes.status >= 300) {
          try {
            const err = this.encoder.decode(rawRes.data, ErrorPayloadSchema);
            return [outMD, decodeError(err)];
          } catch {
            return [outMD, new Error(rawRes.data)];
          }
        }
        if (resSchema) res = this.encoder.decode(rawRes.data, resSchema);
        return [outMD, undefined];
      }
    );

    return [res, err];
  }
}

/**
 * Implementation of the UnaryClient protocol backed by HTTP GET requests. It
 * should not be instantiated directly, but through the HTTPClientFactory.
 */
export class GETClient extends Core implements UnaryClient {
  async send<RQ, RS>(
    target: string,
    req: RQ | null,
    resSchema: ZodSchema<RS> | null
  ): Promise<[RS | undefined, Error | undefined]> {
    const cfg = this.requestConfig();
    cfg.method = 'GET';
    cfg.url =
      this.endpoint.child(target).stringify() +
      buildQueryString({ req: req as Record<string, unknown> });
    return await this.execute(cfg, resSchema);
  }
}

/**
 * Implementation of the UnaryClient protocol backed by HTTP POST requests. It
 * should not be instantiated directly, but through the HTTPClientFactory.
 */
export class POSTClient extends Core implements UnaryClient {
  async send<RQ, RS>(
    target: string,
    req: RQ | null,
    resSchema: ZodSchema<RS> | null
  ): Promise<[RS | undefined, Error | undefined]> {
    const cfg = this.requestConfig();
    cfg.method = 'POST';
    cfg.url = this.endpoint.child(target).stringify();
    if (req) cfg.data = this.encoder.encode(req);
    return await this.execute(cfg, resSchema);
  }
}

export const buildQueryString = ({
  req,
  prefix = '',
}: {
  req: Record<string, unknown> | null;
  prefix?: string;
}): string => {
  if (req === null) return '';
  return (
    '?' +
    Object.entries(req)
      .filter(([, value]) => {
        if (value === undefined || value === null) return false;
        if (Array.isArray(value)) return value.length > 0;
        return true;
      })
      .map(([key, value]) => `${prefix}${key}=${value}`)
      .join('&')
  );
};
