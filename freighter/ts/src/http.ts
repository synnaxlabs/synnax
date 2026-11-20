// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  binary,
  type CrudeTimeSpan,
  errors,
  runtime,
  TimeSpan,
  type url,
} from "@synnaxlabs/x";
import type { z } from "zod";

import {
  type Connector,
  DEFAULT_CONNECTOR,
  type FetchInit,
  type FetchResponse,
} from "@/connector";
import { Timeout, Unreachable } from "@/errors";
import type { FileEncoding, FileOptions, FileTransport, UploadBody } from "@/file";
import { type Context, MiddlewareCollector } from "@/middleware";
import type { UnaryClient } from "@/unary";

export const CONTENT_TYPE_HEADER_KEY = "Content-Type";
const ACCEPT_HEADER_KEY = "Accept";

/**
 * The prefix freighter servers require on query-string parameters that should be
 * exposed to handlers as request params. Unprefixed query parameters are ignored so
 * arbitrary query strings never leak into the request context.
 */
export const FREIGHTER_METADATA_PREFIX = "freighterctx";

const ENCODING_CONTENT_TYPES: Record<FileEncoding, string> = {
  JSON: "application/json",
  ZIP: "application/zip",
  CSV: "text/csv",
  TDMS: "application/x-tdms",
};

const contentType = (encoding: FileOptions["encoding"]): string =>
  typeof encoding === "string"
    ? ENCODING_CONTENT_TYPES[encoding]
    : encoding.contentType;

const UNREACHABLE_CODES = new Set([
  "ECONNREFUSED",
  "ECONNRESET",
  "ETIMEDOUT",
  "EPIPE",
  "UND_ERR_CONNECT_TIMEOUT",
  "UND_ERR_SOCKET",
]);

const DENO_TCP_CONNECT_ERROR = "client error (Connect): tcp connect error";

const shouldCastToUnreachable = (
  err: Error & { code?: unknown; errno?: unknown },
): boolean => {
  // First try Node/Undici codes.
  const causeCode =
    typeof err.cause === "object" && err.cause !== null && "code" in err.cause
      ? err.cause.code
      : undefined;
  const code = causeCode ?? err.code ?? err.errno;
  if (typeof code === "string" && UNREACHABLE_CODES.has(code)) return true;

  // Deno's fetch error has no code, so its cause message is the only signal. The text
  // stops at "tcp connect error" so that TLS and DNS failures do not match.
  if (
    err.name === "TypeError" &&
    err.cause instanceof Error &&
    err.cause.message.includes(DENO_TCP_CONNECT_ERROR)
  )
    return true;

  // Browser/Safari fallback: detect canonical network-failure TypeError messages
  if (err.name === "TypeError") {
    const msg = String(err.message || "").toLowerCase();
    if (/load failed|failed to fetch|networkerror|network error/.test(msg)) {
      if (typeof navigator !== "undefined" && navigator.onLine === false) return true;
      // If you want to be conservative, return false here and treat generically. If you
      // want parity with Node for user messaging, you can return true.
      return true;
    }
  }

  // Abort should not be "unreachable"
  if (err.name === "AbortError" || err.code === "ABORT_ERR") return false;

  return false;
};

const HTTP_STATUS_BAD_REQUEST = 400;

const DEFAULT_TIMEOUT = TimeSpan.seconds(60);

// Browser and Node.js timers fire at once past a signed 32-bit count of milliseconds.
const MAX_TIMEOUT = TimeSpan.milliseconds(2 ** 31 - 1);

/** Returns the timeout as a limit, or null when it sets no limit. */
const limitOf = (timeout: CrudeTimeSpan): TimeSpan | null => {
  if (typeof timeout === "number" && !Number.isFinite(timeout)) return null;
  const span = new TimeSpan(timeout);
  return span.greaterThan(TimeSpan.ZERO) && span.lessThanOrEqual(MAX_TIMEOUT)
    ? span
    : null;
};

/**
 * The single query parameter file transfer params travel on. The server exposes its
 * value to the handler as the "params" request param.
 */
const PARAMS_QUERY_KEY = `${FREIGHTER_METADATA_PREFIX}params`;

/**
 * Appends options.params to target as a single percent-encoded query parameter,
 * validated against options.paramsSchema and JSON-encoded with snake_case keys. Params
 * are always JSON regardless of the transport's body codec — a query string cannot
 * carry a binary encoding. Returns target unchanged when params are absent.
 */
const appendQueryParams = (
  target: string,
  { params, paramsSchema }: FileOptions,
): string => {
  if (params == null) return target;
  const search = new URLSearchParams();
  search.set(PARAMS_QUERY_KEY, binary.JSON_CODEC.encodeString(params, paramsSchema));
  return `${target}?${search.toString()}`;
};

/* * Reports whether this engine implements streaming request bodies. */
const detectRequestStreams = (): boolean => {
  let duplexRead = false;
  try {
    const request = new Request("http://request.streams.probe", {
      method: "POST",
      body: new ReadableStream(),
      get duplex() {
        duplexRead = true;
        return "half";
      },
    } as RequestInit);
    return duplexRead && !request.headers.has(CONTENT_TYPE_HEADER_KEY);
  } catch {
    return false;
  }
};

// Browsers implement streaming request bodies but send them only over HTTP/2 or HTTP/3,
// and the negotiated protocol is unknowable before the request goes out. Everywhere
// else a stream rides HTTP/1.1 chunked, which is what the Core speaks.
const STREAMS_REQUEST_BODIES = runtime.RUNTIME === "node" && detectRequestStreams();

const toSendableBody = async (body: UploadBody): Promise<UploadBody> => {
  if (STREAMS_REQUEST_BODIES || !(body instanceof ReadableStream)) return body;
  return await new Response(body).blob();
};

/** Configures an {@link HTTPClient}. */
export interface HTTPClientConfig {
  /** The base URL of the API. */
  endpoint: url.URL;
  /** Encodes requests and decodes responses. */
  codec: binary.Codec;
  /** Whether to send requests over HTTPS. Defaults to false. */
  secure?: boolean;
  /**
   * The longest a send waits for its response, body included. Defaults to 60 seconds.
   * A timeout of zero, infinity, or more than 2^31 - 1 milliseconds (about 24.8 days)
   * sets no limit. File transfers have no limit, since their bodies can take longer.
   */
  timeout?: CrudeTimeSpan;
  /** Sends the requests. Defaults to the global fetch. */
  connector?: Connector;
}

/** HTTPClient provides a POST implementation of the Unary protocol. */
export class HTTPClient
  extends MiddlewareCollector
  implements UnaryClient, FileTransport
{
  endpoint: url.URL;
  encoder: binary.Codec;
  private readonly timeout: TimeSpan | null;
  private readonly connector: Connector;

  constructor({
    endpoint,
    codec,
    secure = false,
    timeout = DEFAULT_TIMEOUT,
    connector = DEFAULT_CONNECTOR,
  }: HTTPClientConfig) {
    super();
    this.endpoint = endpoint.replace({ protocol: secure ? "https" : "http" });
    this.encoder = codec;
    this.timeout = limitOf(timeout);
    this.connector = connector;

    return new Proxy(this, {
      get: (target, prop, receiver) => {
        if (prop === "endpoint") return this.endpoint;
        return Reflect.get(target, prop, receiver);
      },
    });
  }

  private get defaultHeaders(): Record<string, string> {
    return {
      [CONTENT_TYPE_HEADER_KEY]: this.encoder.contentType,
      [ACCEPT_HEADER_KEY]: this.encoder.contentType,
    };
  }

  /**
   * Sends a request to the target and waits for its response.
   * @throws {Timeout} if no response arrives within the timeout.
   * @throws {Unreachable} if the target cannot be reached.
   */
  async send<RQ extends z.ZodType, RS extends z.ZodType = RQ>(
    target: string,
    req: z.input<RQ> | z.infer<RQ>,
    reqSchema: RQ,
    resSchema: RS,
    params: Record<string, string> = {},
  ): Promise<z.infer<RS>> {
    let res: z.infer<RS> | null = null;
    const url = this.endpoint.child(target);
    await this.executeMiddleware(
      { ...this.context(url), params: { ...params } },
      async (ctx: Context): Promise<Context> => {
        const outCtx: Context = { ...ctx, params: {} };
        const { timeout } = this;
        const signal =
          timeout == null ? undefined : AbortSignal.timeout(timeout.milliseconds);
        let httpRes: FetchResponse;
        let data: ArrayBuffer;
        try {
          httpRes = await this.fetch(url, ctx.target, {
            method: "POST",
            body: this.encoder.encode(req, reqSchema),
            headers: { ...this.defaultHeaders, ...ctx.params },
            signal,
          });
          data = await httpRes.arrayBuffer();
        } catch (e) {
          if (timeout == null || signal?.aborted !== true) throw errors.fromUnknown(e);
          throw new Timeout(
            `[freighter] no response from ${ctx.target} within ${timeout.toString()}`,
            { cause: e },
          );
        }
        if (httpRes.ok) {
          if (resSchema != null)
            res = this.encoder.decode<RS>(data, resSchema, {
              label: `${target} response`,
            });
          return outCtx;
        }
        throw this.decodeError(data, httpRes, ctx.target);
      },
    );
    if (res == null) throw new Error("Response must be defined");
    return res;
  }

  async upload<RS extends z.ZodType, P extends z.ZodType = z.ZodType>(
    target: string,
    body: UploadBody,
    options: FileOptions<P>,
    resSchema: RS,
  ): Promise<z.infer<RS>> {
    let res: z.infer<RS> | null = null;
    const url = this.endpoint.child(target);
    await this.executeMiddleware(
      this.context(url),
      async (ctx: Context): Promise<Context> => {
        const outCtx: Context = { ...ctx, params: {} };
        // duplex is required by the Fetch standard whenever the body is a stream.
        const init: FetchInit = {
          method: "POST",
          body: await toSendableBody(body),
          headers: {
            ...this.defaultHeaders,
            [CONTENT_TYPE_HEADER_KEY]: contentType(options.encoding),
            ...ctx.params,
          },
          duplex: "half",
        };
        const httpRes = await this.fetch(
          url,
          appendQueryParams(ctx.target, options),
          init,
        );
        const data = await httpRes.arrayBuffer();
        if (httpRes.ok) {
          res = this.encoder.decode<RS>(data, resSchema, {
            label: `${target} response`,
          });
          return outCtx;
        }
        throw this.decodeError(data, httpRes, ctx.target);
      },
    );
    if (res == null) throw new Error("Response must be defined");
    return res;
  }

  async download<RQ extends z.ZodType, P extends z.ZodType = z.ZodType>(
    target: string,
    req: z.input<RQ> | z.infer<RQ>,
    reqSchema: RQ,
    options: FileOptions<P>,
  ): Promise<ReadableStream<Uint8Array>> {
    let stream: ReadableStream<Uint8Array> | null = null;
    const url = this.endpoint.child(target);
    await this.executeMiddleware(
      this.context(url),
      async (ctx: Context): Promise<Context> => {
        const outCtx: Context = { ...ctx, params: {} };
        const httpRes = await this.fetch(url, appendQueryParams(ctx.target, options), {
          method: "POST",
          body: this.encoder.encode(req, reqSchema),
          headers: {
            ...this.defaultHeaders,
            [ACCEPT_HEADER_KEY]: contentType(options.encoding),
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
    request: FetchInit,
  ): Promise<FetchResponse> {
    const cast = (e: unknown, delivered: boolean): Error => {
      const err = errors.fromUnknown(e);
      return shouldCastToUnreachable(err)
        ? new Unreachable({ url, cause: err, delivered })
        : err;
    };
    let res: FetchResponse;
    try {
      res = await this.connector.fetch(target, request);
    } catch (e) {
      throw cast(e, false);
    }
    // The connection can fail after the headers arrive, while the body is read.
    return {
      ok: res.ok,
      status: res.status,
      statusText: res.statusText,
      body: res.body,
      arrayBuffer: async () => {
        try {
          return await res.arrayBuffer();
        } catch (e) {
          throw cast(e, true);
        }
      },
    };
  }

  private decodeError(
    data: ArrayBuffer,
    httpRes: FetchResponse,
    target: string,
  ): Error {
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
