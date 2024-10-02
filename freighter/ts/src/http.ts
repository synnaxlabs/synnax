// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/* eslint-disable @typescript-eslint/no-var-requires */

import { type binary, runtime, type URL } from "@synnaxlabs/x";
import { type z } from "zod";

import { decodeError, errorZ, Unreachable } from "@/errors";
import { type Context, MiddlewareCollector } from "@/middleware";
import { type UnaryClient } from "@/unary";

export const CONTENT_TYPE_HEADER_KEY = "Content-Type";

const resolveFetchAPI = (protocol: "http" | "https"): typeof fetch => {
  if (runtime.RUNTIME !== "node") return fetch;
  const _fetch: typeof fetch = require("node-fetch");
  if (protocol === "http") return _fetch;
  const https = require("https");
  const agent = new https.Agent({ rejectUnauthorized: false });
  // @ts-expect-error - TS doesn't know about qhis option
  return async (info, init) => await _fetch(info, { ...init, agent });
};

/**
 * HTTPClientFactory provides a POST and GET implementation of the Unary
 * protocol.
 *
 * @param url - The base URL of the API.
 * @param encoder - The encoder/decoder to use for the request/response.
 */
export class HTTPClient extends MiddlewareCollector implements UnaryClient {
  endpoint: URL;
  encoder: binary.Codec;
  fetch: typeof fetch;

  constructor(endpoint: URL, encoder: binary.Codec, secure: boolean = false) {
    super();
    this.endpoint = endpoint.replace({ protocol: secure ? "https" : "http" });
    this.encoder = encoder;
    this.fetch = resolveFetchAPI(this.endpoint.protocol as "http" | "https");

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
    req: z.input<RQ> | z.output<RQ>,
    reqSchema: RQ,
    resSchema: RS,
  ): Promise<[z.output<RS> | null, Error | null]> {
    req = reqSchema?.parse(req);
    let res: RS | null = null;
    const url = this.endpoint.child(target);
    const request: RequestInit = {};
    request.method = "POST";
    request.body = this.encoder.encode(req ?? {});

    const [, err] = await this.executeMiddleware(
      {
        target: url.toString(),
        protocol: this.endpoint.protocol,
        params: {},
        role: "client",
      },
      async (ctx: Context): Promise<[Context, Error | null]> => {
        const outCtx: Context = { ...ctx, params: {} };
        request.headers = {
          ...this.headers,
          ...ctx.params,
        };
        let httpRes: Response;
        try {
          const f = resolveFetchAPI(ctx.protocol as "http" | "https");
          httpRes = await f(ctx.target, request);
        } catch (err_) {
          let err = err_ as Error;
          if (err.message === "Load failed") err = new Unreachable({ url });
          return [outCtx, err];
        }
        const data = await httpRes.arrayBuffer();
        if (httpRes?.ok) {
          if (resSchema != null) res = this.encoder.decode(data, resSchema);
          return [outCtx, null];
        }
        try {
          if (httpRes.status !== 400) return [outCtx, new Error(httpRes.statusText)];
          const err = this.encoder.decode(data, errorZ);
          const decoded = decodeError(err);
          return [outCtx, decoded];
        } catch (e) {
          return [
            outCtx,
            new Error(
              `[freighter] - failed to decode error: ${httpRes.statusText}: ${
                (e as Error).message
              }`,
            ),
          ];
        }
      },
    );

    return [res, err];
  }
}
