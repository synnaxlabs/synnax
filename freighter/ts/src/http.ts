// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { runtime, URL, binary } from "@synnaxlabs/x";
import { z } from "zod";

import { errorZ, decodeError, Unreachable } from "@/errors";
import { Context, MiddlewareCollector } from "@/middleware";
import { UnaryClient } from "@/unary";

export const CONTENT_TYPE_HEADER_KEY = "Content-Type";

const resolveFetchAPI = (): typeof fetch =>
  runtime.RUNTIME === "node" ? require("node-fetch") : fetch;

/**
 * HTTPClientFactory provides a POST and GET implementation of the Unary
 * protocol.
 *
 * @param url - The base URL of the API.
 * @param encoder - The encoder/decoder to use for the request/response.
 */
export class HTTPClient extends MiddlewareCollector implements UnaryClient {
  endpoint: URL;
  encoder: binary.EncoderDecoder;
  fetch: typeof fetch;

  constructor(endpoint: URL, encoder: binary.EncoderDecoder, secure: boolean = false) {
    super();
    this.endpoint = endpoint.replace({ protocol: secure ? "https" : "http" });
    this.encoder = encoder;
    this.fetch = resolveFetchAPI();

    return new Proxy(this, {
      get: (target, prop, receiver) => {
        if (prop === "endpoint") return this.endpoint;
        return Reflect.get(target, prop, receiver);
      },
    });
  }

  get headers(): Record<string, string> {
    return {
      [CONTENT_TYPE_HEADER_KEY]: this.encoder.contentType,
    };
  }

  async send<RQ extends z.ZodTypeAny, RS extends z.ZodTypeAny = RQ>(
    target: string,
    req: z.input<RQ> | null,
    resSchema: RS | null
  ): Promise<[z.output<RS> | null, Error | null]> {
    let rs: RS | null = null;
    const url = this.endpoint.child(target);
    const request: RequestInit = {};
    request.method = "POST";
    request.body = this.encoder.encode(req ?? {});

    const [, err] = await this.executeMiddleware(
      { target: url.toString(), protocol: "http", params: {}, role: "client" },
      async (ctx: Context): Promise<[Context, Error | null]> => {
        const outCtx: Context = { ...ctx, params: {} };
        request.headers = {
          ...this.headers,
          ...ctx.params,
        };
        let httpRes: Response;
        try {
          httpRes = await fetch(ctx.target, request);
        } catch (err_) {
          let err = err_ as Error;
          if (err.message === "Load failed") err = new Unreachable({ url });
          return [outCtx, err];
        }
        const data = await httpRes.arrayBuffer();
        if (httpRes?.ok) {
          if (resSchema != null) rs = this.encoder.decode(data, resSchema);
          return [outCtx, null];
        }
        try {
          const err = this.encoder.decode(data, errorZ);
          return [outCtx, decodeError(err)];
        } catch {
          return [outCtx, new Error(httpRes.statusText)];
        }
      }
    );

    return [rs, err];
  }
}
