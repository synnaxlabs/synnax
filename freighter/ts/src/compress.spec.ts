// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { binary, url } from "@synnaxlabs/x";
import { afterEach, describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { DEFAULT_COMPRESSION, HTTPClient } from "@/http";

const ENDPOINT = new url.URL({ host: "127.0.0.1", protocol: "http", port: 8080 });

const messageZ = z.object({ id: z.number(), message: z.string() });

/**
 * A body large enough to clear the compression floor and repetitive enough that gzip
 * shrinks it dramatically.
 */
const LARGE_MESSAGE = "synnax telemetry payload ".repeat(200);

/**
 * Replaces global fetch with a stub that records the request it was handed and always
 * answers with an uncompressed echo. fetch is the transport's boundary to the runtime,
 * so it is the right seam to substitute: everything above it is real client code.
 */
const stubFetch = () => {
  const calls: {
    body: Uint8Array<ArrayBuffer>;
    headers: Record<string, string>;
  }[] = [];
  const spy = vi
    .spyOn(globalThis, "fetch")
    .mockImplementation(async (_target, init) => {
      const body = new Uint8Array(
        init?.body instanceof Uint8Array ? init.body : new Uint8Array(),
      );
      calls.push({ body, headers: (init?.headers ?? {}) as Record<string, string> });
      return new Response(
        new TextEncoder().encode(JSON.stringify({ id: 1, message: "ok" })),
        { status: 200, headers: { "Content-Type": "application/json" } },
      );
    });
  return { calls, spy };
};

const gunzip = async (body: Uint8Array<ArrayBuffer>): Promise<string> => {
  const stream = new Blob([body]).stream().pipeThrough(new DecompressionStream("gzip"));
  return await new Response(stream).text();
};

describe("compression", () => {
  afterEach(() => vi.restoreAllMocks());

  it("should compress a request body above the size floor", async () => {
    const { calls } = stubFetch();
    const client = new HTTPClient(ENDPOINT, new binary.JSONCodec());
    await client.send("/echo", { id: 1, message: LARGE_MESSAGE }, messageZ, messageZ);
    const [call] = calls;
    expect(call.headers["Content-Encoding"]).toEqual("gzip");
    expect(call.body.byteLength).toBeLessThan(LARGE_MESSAGE.length);
    const decoded = JSON.parse(await gunzip(call.body));
    expect(decoded.message).toEqual(LARGE_MESSAGE);
  });

  it("should send a body below the size floor uncompressed", async () => {
    const { calls } = stubFetch();
    const client = new HTTPClient(ENDPOINT, new binary.JSONCodec());
    await client.send("/echo", { id: 1, message: "hello" }, messageZ, messageZ);
    const [call] = calls;
    expect(call.headers["Content-Encoding"]).toBeUndefined();
    expect(JSON.parse(new TextDecoder().decode(call.body)).message).toEqual("hello");
  });

  it("should send uncompressed when the encoding is null", async () => {
    const { calls } = stubFetch();
    const client = new HTTPClient(ENDPOINT, new binary.JSONCodec(), false, {
      ...DEFAULT_COMPRESSION,
      encoding: null,
    });
    await client.send("/echo", { id: 1, message: LARGE_MESSAGE }, messageZ, messageZ);
    const [call] = calls;
    expect(call.headers["Content-Encoding"]).toBeUndefined();
    expect(call.body.byteLength).toBeGreaterThan(LARGE_MESSAGE.length);
  });

  it("should honor a custom size floor", async () => {
    const { calls } = stubFetch();
    const client = new HTTPClient(ENDPOINT, new binary.JSONCodec(), false, {
      encoding: "gzip",
      minSize: 100,
    });
    // Well under the default floor of 1024, so only the lowered floor lets this
    // through.
    const message = "compress me ".repeat(20);
    await client.send("/echo", { id: 1, message }, messageZ, messageZ);
    expect(calls[0].headers["Content-Encoding"]).toEqual("gzip");
  });

  it("should send uncompressed when compressing would grow the body", async () => {
    const { calls } = stubFetch();
    const client = new HTTPClient(ENDPOINT, new binary.JSONCodec(), false, {
      encoding: "gzip",
      minSize: 1,
    });
    // A short message compresses to more bytes than it started with, so the client
    // must fall back to the original.
    await client.send("/echo", { id: 1, message: "a" }, messageZ, messageZ);
    expect(calls[0].headers["Content-Encoding"]).toBeUndefined();
  });

  it("should support deflate", async () => {
    const { calls } = stubFetch();
    const client = new HTTPClient(ENDPOINT, new binary.JSONCodec(), false, {
      encoding: "deflate",
      minSize: 1,
    });
    await client.send("/echo", { id: 1, message: LARGE_MESSAGE }, messageZ, messageZ);
    expect(calls[0].headers["Content-Encoding"]).toEqual("deflate");
  });
});
