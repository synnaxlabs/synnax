// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { URL } from "@synnaxlabs/x";
import axios from "axios";
import type { AxiosRequestConfig, AxiosResponse, RawAxiosRequestHeaders } from "axios";
import { z } from "zod";

import { EncoderDecoder } from "@/encoder";
import { errorZ, decodeError } from "@/errors";
import { Context, MiddlewareCollector } from "@/middleware";
import { UnaryClient } from "@/unary";

export const CONTENT_TYPE_HEADER_KEY = "Content-Type";

/**
 * HTTPClientFactory provides a POST and GET implementation of the Unary
 * protocol.
 *
 * @param url - The base URL of the API.
 * @param encoder - The encoder/decoder to use for the request/response.
 */
export class HTTPClient extends MiddlewareCollector implements UnaryClient {
  endpoint: URL;
  encoder: EncoderDecoder;

  constructor(endpoint: URL, encoder: EncoderDecoder, secure: boolean = false) {
    super();
    this.endpoint = endpoint.replace({ protocol: secure ? "https" : "http" });
    this.encoder = encoder;
  }

  get headers(): RawAxiosRequestHeaders {
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

  async execute<RS extends z.ZodTypeAny>(
    request: AxiosRequestConfig,
    resSchema: RS | null
  ): Promise<[z.output<RS> | null, Error | null]> {
    let rs: RS | null = null;

    if (request.url == null)
      throw new Error("[freighter.http] - expected valid request url");

    const [, err] = await this.executeMiddleware(
      { target: request.url, protocol: "http", params: {}, role: "client" },
      async (ctx: Context): Promise<[Context, Error | null]> => {
        const outCtx: Context = { ...ctx, params: {} };
        request.headers = {
          ...(request.headers as RawAxiosRequestHeaders),
          ...this.headers,
          ...ctx.params,
        };
        let httpRes: AxiosResponse;
        try {
          httpRes = await axios.request(request);
        } catch (err) {
          return [outCtx, err as Error];
        }
        outCtx.params = httpRes.headers as Record<string, string>;
        if (httpRes.status < 200 || httpRes.status >= 300) {
          try {
            const err = this.encoder.decode(httpRes.data, errorZ);
            return [outCtx, decodeError(err)];
          } catch {
            return [outCtx, new Error(httpRes.data)];
          }
        }
        if (resSchema != null) rs = this.encoder.decode(httpRes.data, resSchema);
        return [outCtx, null];
      }
    );

    return [rs, err];
  }

  async send<RQ extends z.ZodTypeAny, RS extends z.ZodTypeAny = RQ>(
    target: string,
    req: z.input<RQ> | null,
    resSchema: RS | null
  ): Promise<[z.output<RS> | null, Error | null]> {
    const url = this.endpoint.child(target).toString();
    const request = this.requestConfig();
    request.method = "POST";
    request.url = url;
    request.data = this.encoder.encode(req ?? {});
    return await this.execute(request, resSchema);
  }
}
