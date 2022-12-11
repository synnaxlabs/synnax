import { ZodSchema } from 'zod';

import { EncoderDecoder } from './encoder';
import { ErrorPayloadSchema, decodeError } from './errors';
import { MetaData, MiddlewareCollector } from './middleware';
import { UnaryClient } from './unary';
import URL from './url';
import { Runtime, RUNTIME } from './runtime';

/**
 * BaseHTTPClient is a base representation of the fetch API. Depending on the current
 * runtime, this can be swapped out for other implementations.
 */
type BaseHTTPClient = (
  url: RequestInfo,
  init?: RequestInit | undefined
) => Promise<Response>;

const resolveBaseHTTPClient = (): BaseHTTPClient => {
  if (RUNTIME == Runtime.Node) return require('node-fetch');
  return fetch;
};

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
  base: BaseHTTPClient;

  constructor(
    baseEndpoint: URL,
    encoder: EncoderDecoder,
    secure: boolean = false,
    base?: BaseHTTPClient
  ) {
    super();
    this.base = base ?? resolveBaseHTTPClient();
    this.baseEndpoint = baseEndpoint;
    this.encoder = encoder;
    this.secure = secure;
  }

  /** @returns A UnaryClient that uses GET requests. */
  getClient(): GETClient {
    const gc = new GETClient(this.baseEndpoint, this.encoder, this.secure, this.base);
    gc.use(...this.middleware);
    return gc;
  }

  /** @returns a UnaryClient that uses POST requests. */
  postClient(): POSTClient {
    const pc = new POSTClient(this.baseEndpoint, this.encoder, this.secure, this.base);
    pc.use(...this.middleware);
    return pc;
  }
}

export const CONTENT_TYPE_HEADER_KEY = 'Content-Type';

class Core extends MiddlewareCollector {
  endpoint: URL;
  encoder: EncoderDecoder;
  base: BaseHTTPClient;

  constructor(
    endpoint: URL,
    encoder: EncoderDecoder,
    secure: boolean = false,
    base: BaseHTTPClient
  ) {
    super();
    this.base = base;
    this.endpoint = endpoint.replace({ protocol: secure ? 'https' : 'http' });
    this.encoder = encoder;
  }

  get headers() {
    return {
      [CONTENT_TYPE_HEADER_KEY]: this.encoder.contentType,
    };
  }

  requestConfig(): RequestInit {
    return {
      headers: this.headers,
      credentials: 'omit',
    };
  }

  async execute<RS>(
    url: string,
    request: RequestInit,
    resSchema: ZodSchema<RS> | null
  ): Promise<[RS | undefined, Error | undefined]> {
    let res: RS | undefined = undefined;

    const [, err] = await this.executeMiddleware(
      { target: url, protocol: 'http', params: {} },
      async (md: MetaData): Promise<[MetaData, Error | undefined]> => {
        let outMD: MetaData = { ...md };
        request.headers = { ...request.headers, ...this.headers, ...md.params };
        let rawRes: Response;
        try {
          rawRes = await this.base(url, request);
        } catch (err) {
          return [outMD, err as Error];
        }
        rawRes.headers.forEach((value, key) => (outMD.params[key] = value));
        if (rawRes.status < 200 || rawRes.status >= 300) {
          try {
            const err = this.encoder.decode(
              await rawRes.arrayBuffer(),
              ErrorPayloadSchema
            );
            return [outMD, decodeError(err)];
          } catch (err) {
            return [outMD, err as Error];
          }
        }
        if (resSchema) res = this.encoder.decode(await rawRes.arrayBuffer(), resSchema);
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
    return await this.execute(
      this.endpoint.child(target).toString() +
        buildQueryString({ req: req as Record<string, unknown> }),
      cfg,
      resSchema
    );
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
    if (req) cfg.body = this.encoder.encode(req);
    return await this.execute(this.endpoint.child(target).toString(), cfg, resSchema);
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
