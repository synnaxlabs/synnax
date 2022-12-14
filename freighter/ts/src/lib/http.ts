// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import axios from "axios";
import type { AxiosRequestConfig, AxiosResponse } from "axios";
import { ZodSchema } from "zod";

import { EncoderDecoder } from "./encoder";
import { ErrorPayloadSchema, decodeError } from "./errors";
import { MetaData, MiddlewareCollector } from "./middleware";
import { UnaryClient } from "./unary";
import URL from "./url";

/**
 * HTTPClientFactory provides a POST and GET implementation of the Unary
 * protocol.
 *
 * @param url - The base URL of the API.
 * @param encoder - The encoder/decoder to use for the request/response.
 */
export class HTTPClientFactory extends MiddlewareCollector {
  endpoint: URL;
  encoder: EncoderDecoder;
  secure: boolean;

  constructor(endpoint: URL, encoder: EncoderDecoder, secure: boolean = false) {
    super();
    this.endpoint = endpoint;
    this.encoder = encoder;
    this.secure = secure;
  }

  newGET(): GETClient {
    const gc = new GETClient(this.endpoint, this.encoder, this.secure);
    gc.use(...this.middleware);
    return gc;
  }

  newPOST(): POSTClient {
    const pc = new POSTClient(this.endpoint, this.encoder, this.secure);
    pc.use(...this.middleware);
    return pc;
  }
}

export const CONTENT_TYPE_HEADER_KEY = "Content-Type";

class Core extends MiddlewareCollector {
  endpoint: URL;
  encoder: EncoderDecoder;

  constructor(endpoint: URL, encoder: EncoderDecoder, secure: boolean = false) {
    super();
    this.endpoint = endpoint.replace({ protocol: secure ? "https" : "http" });
    this.encoder = encoder;
  }

  get headers(): Record<string, string> {
    return {
      [CONTENT_TYPE_HEADER_KEY]: this.encoder.contentType,
    };
  }

  requestConfig(): AxiosRequestConfig {
    return {
      headers: this.headers,
      responseType: "arraybuffer",
      withCredentials: false,
      validateStatus: () => true,
    };
  }

  async execute<RS>(
    request: AxiosRequestConfig,
    resSchema: ZodSchema<RS> | null
  ): Promise<[RS | undefined, Error | undefined]> {
    let rs: RS | undefined;

    if (request.url == null)
      throw new Error("[freighter.http] - expected valid request url");

    const [, err] = await this.executeMiddleware(
      { target: request.url, protocol: "http", params: {} },
      async (md: MetaData): Promise<[MetaData, Error | undefined]> => {
        const outMD: MetaData = { ...md, params: {} };
        request.headers = { ...request.headers, ...this.headers, ...md.params };
        let httpRes: AxiosResponse;
        try {
          httpRes = await axios.request(request);
        } catch (err) {
          return [outMD, err as Error];
        }
        outMD.params = httpRes.headers as Record<string, string>;
        if (httpRes.status < 200 || httpRes.status >= 300) {
          try {
            const err = this.encoder.decode(httpRes.data, ErrorPayloadSchema);
            return [outMD, decodeError(err)];
          } catch {
            return [outMD, new Error(httpRes.data)];
          }
        }
        if (resSchema != null) rs = this.encoder.decode(httpRes.data, resSchema);
        return [outMD, undefined];
      }
    );

    return [rs, err];
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
    const request = this.requestConfig();
    request.method = "GET";
    request.url =
      this.endpoint.child(target).toString() +
      buildQueryString({ request: req as Record<string, unknown> });
    request.data = null;
    return await this.execute(request, resSchema);
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
    const url = this.endpoint.child(target).toString();
    const request = this.requestConfig();
    request.method = "POST";
    request.url = url;
    if (req != null) request.data = this.encoder.encode(req);
    return await this.execute(request, resSchema);
  }
}

export const buildQueryString = ({
  request,
  prefix = "",
}: {
  request: Record<string, unknown> | null;
  prefix?: string;
}): string => {
  if (request === null) return "";
  return (
    "?" +
    Object.entries(request)
      .filter(([, value]) => {
        if (value === undefined || value === null) return false;
        if (Array.isArray(value)) return value.length > 0;
        return true;
      })
      // eslint-disable-next-line @typescript-eslint/restrict-template-expressions
      .map(([key, value]) => `${prefix}${key}=${value}`)
      .join("&")
  );
};
