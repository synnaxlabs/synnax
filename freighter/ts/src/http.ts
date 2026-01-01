// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type binary, errors, type URL } from "@synnaxlabs/x";
import { type z } from "zod";

import { Unreachable } from "@/errors";
import { type Context, MiddlewareCollector } from "@/middleware";
import { type UnaryClient } from "@/unary";

export const CONTENT_TYPE_HEADER_KEY = "Content-Type";

const UNREACHABLE_CODES = new Set([
  "ECONNREFUSED",
  "ECONNRESET",
  "ETIMEDOUT",
  "EPIPE",
  "UND_ERR_CONNECT_TIMEOUT",
  "UND_ERR_SOCKET",
]);

export const shouldCastToUnreachable = (err: Error): boolean => {
  // First try Node/Undici codes
  const code = (err as any)?.cause?.code ?? (err as any)?.code ?? (err as any)?.errno;
  if (typeof code === "string" && UNREACHABLE_CODES.has(code)) return true;

  // Browser/Safari fallback: detect canonical network-failure TypeError messages
  if (err.name === "TypeError") {
    const msg = String(err.message || "").toLowerCase();
    if (/load failed|failed to fetch|networkerror|network error/.test(msg)) {
      // Optionally gate on being online:
      if (typeof navigator !== "undefined" && navigator.onLine === false) return true;
      // If you want to be conservative, return false here and treat generically.
      // If you want parity with Node for user messaging, you can return true.
      return true;
    }
  }

  // Abort should not be "unreachable"
  if ((err as any)?.name === "AbortError" || (err as any)?.code === "ABORT_ERR")
    return false;

  return false;
};

const HTTP_STATUS_BAD_REQUEST = 400;

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

  constructor(endpoint: URL, encoder: binary.Codec, secure: boolean = false) {
    super();
    this.endpoint = endpoint.replace({ protocol: secure ? "https" : "http" });
    this.encoder = encoder;

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

  async send<RQ extends z.ZodType, RS extends z.ZodType = RQ>(
    target: string,
    req: z.input<RQ> | z.infer<RQ>,
    reqSchema: RQ,
    resSchema: RS,
  ): Promise<[z.infer<RS>, null] | [null, Error]> {
    req = reqSchema?.parse(req);
    let res: z.infer<RS> | null = null;
    const url = this.endpoint.child(target);
    const request: RequestInit = {};
    request.method = "POST";
    request.body = this.encoder.encode(req ?? {}) as BodyInit;
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
          httpRes = await fetch(ctx.target, request);
        } catch (e) {
          if (!(e instanceof Error)) throw e;
          return [outCtx, shouldCastToUnreachable(e) ? new Unreachable({ url }) : e];
        }
        const data = await httpRes.arrayBuffer();
        if (httpRes?.ok) {
          if (resSchema != null) res = this.encoder.decode<RS>(data, resSchema);
          return [outCtx, null];
        }
        try {
          if (httpRes.status !== HTTP_STATUS_BAD_REQUEST)
            return [outCtx, new Error(httpRes.statusText)];
          const err = this.encoder.decode(data, errors.payloadZ);
          const decoded = errors.decode(err);
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

    if (err != null) return [null, err];
    if (res == null) throw new Error("Response must be defined");
    return [res, null];
  }
}
