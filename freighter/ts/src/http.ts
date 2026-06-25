// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type binary, errors, type url } from "@synnaxlabs/x";
import { type z } from "zod";

import { Unreachable } from "@/errors";
import {
  type FileClient,
  type FileEncoding,
  type FileOptions,
  type UploadBody,
} from "@/file";
import { type Context, MiddlewareCollector } from "@/middleware";
import { type UnaryClient } from "@/unary";

export const CONTENT_TYPE_HEADER_KEY = "Content-Type";

const ENCODING_CONTENT_TYPES: Record<FileEncoding, string> = {
  JSON: "application/json",
  MessagePack: "application/msgpack",
  YAML: "application/yaml",
  TOML: "application/toml",
};

/**
 * Builds the Content-Type / Accept header value for one or more encodings. A list is
 * joined into a comma-separated header so the server can negotiate which encoding to
 * use.
 */
const encodingHeader = (encoding: FileEncoding | FileEncoding[]): string =>
  (Array.isArray(encoding) ? encoding : [encoding])
    .map((e) => ENCODING_CONTENT_TYPES[e])
    .join(", ");

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
export class HTTPClient extends MiddlewareCollector implements UnaryClient, FileClient {
  endpoint: url.URL;
  encoder: binary.Codec;

  constructor(endpoint: url.URL, encoder: binary.Codec, secure: boolean = false) {
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
    return { [CONTENT_TYPE_HEADER_KEY]: this.encoder.contentType };
  }

  async send<RQ extends z.ZodType, RS extends z.ZodType = RQ>(
    target: string,
    req: z.input<RQ> | z.infer<RQ>,
    reqSchema: RQ,
    resSchema: RS,
  ): Promise<z.infer<RS>> {
    let res: z.infer<RS> | null = null;
    const url = this.endpoint.child(target);
    await this.executeMiddleware(
      this.context(url),
      async (ctx: Context): Promise<Context> => {
        const outCtx: Context = { ...ctx, params: {} };
        const httpRes = await this.fetch(url, ctx.target, {
          method: "POST",
          body: this.encoder.encode(req, reqSchema) as BodyInit,
          headers: { ...this.headers, ...ctx.params },
        });
        const data = await httpRes.arrayBuffer();
        if (httpRes.ok) {
          if (resSchema != null) res = this.encoder.decode<RS>(data, resSchema);
          return outCtx;
        }
        throw this.decodeError(data, httpRes, ctx.target);
      },
    );
    if (res == null) throw new Error("Response must be defined");
    return res;
  }

  async upload<RS extends z.ZodType>(
    target: string,
    body: UploadBody,
    { encoding }: FileOptions,
    resSchema: RS,
  ): Promise<z.infer<RS>> {
    let res: z.infer<RS> | null = null;
    const url = this.endpoint.child(target);
    await this.executeMiddleware(
      this.context(url),
      async (ctx: Context): Promise<Context> => {
        const outCtx: Context = { ...ctx, params: {} };
        const httpRes = await this.fetch(url, ctx.target, {
          method: "POST",
          body: body as BodyInit,
          headers: {
            [CONTENT_TYPE_HEADER_KEY]: encodingHeader(encoding),
            Accept: this.encoder.contentType,
            ...ctx.params,
          },
          // duplex is required by the Fetch standard whenever the body is a stream.
          duplex: "half",
        } as RequestInit);
        const data = await httpRes.arrayBuffer();
        if (httpRes.ok) {
          res = this.encoder.decode<RS>(data, resSchema);
          return outCtx;
        }
        throw this.decodeError(data, httpRes, ctx.target);
      },
    );
    if (res == null) throw new Error("Response must be defined");
    return res;
  }

  async download<RQ extends z.ZodType>(
    target: string,
    req: z.input<RQ> | z.infer<RQ>,
    reqSchema: RQ,
    { encoding }: FileOptions,
  ): Promise<ReadableStream<Uint8Array>> {
    let stream: ReadableStream<Uint8Array> | null = null;
    const url = this.endpoint.child(target);
    await this.executeMiddleware(
      this.context(url),
      async (ctx: Context): Promise<Context> => {
        const outCtx: Context = { ...ctx, params: {} };
        const httpRes = await this.fetch(url, ctx.target, {
          method: "POST",
          body: this.encoder.encode(req, reqSchema) as BodyInit,
          headers: {
            ...this.headers,
            Accept: encodingHeader(encoding),
            ...ctx.params,
          },
        });
        if (httpRes.ok) {
          if (httpRes.body == null)
            throw new Error("[freighter] response body is empty");
          stream = httpRes.body;
          return outCtx;
        }
        throw this.decodeError(await httpRes.arrayBuffer(), httpRes, ctx.target);
      },
    );
    if (stream == null) throw new Error("Response stream must be defined");
    return stream;
  }

  private context(url: url.URL): Context {
    return {
      target: url.toString(),
      protocol: this.endpoint.protocol,
      params: {},
      role: "client",
    };
  }

  private async fetch(
    url: url.URL,
    target: string,
    request: RequestInit,
  ): Promise<Response> {
    try {
      return await fetch(target, request);
    } catch (e) {
      const err = errors.fromUnknown(e);
      throw shouldCastToUnreachable(err) ? new Unreachable({ url, cause: err }) : err;
    }
  }

  private decodeError(data: ArrayBuffer, httpRes: Response, target: string): Error {
    if (httpRes.status !== HTTP_STATUS_BAD_REQUEST)
      return new Error(
        `[freighter] HTTP ${httpRes.status} from ${target}: ${httpRes.statusText}`,
      );
    let decoded: Error | null;
    try {
      decoded = errors.decode(this.encoder.decode(data, errors.payloadZ));
    } catch (e) {
      const err = errors.fromUnknown(e);
      return new Error(
        `[freighter] - failed to decode error: ${httpRes.statusText}: ${err.message}`,
        { cause: e },
      );
    }
    return (
      decoded ??
      new Error(
        `[freighter] HTTP ${httpRes.status} from ${target}: ${httpRes.statusText}`,
      )
    );
  }
}
