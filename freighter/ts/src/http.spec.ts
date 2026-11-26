// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createServer, type RequestListener, type Server } from "node:http";
import type { AddressInfo } from "node:net";

import { binary, type CrudeTimeSpan, TimeSpan, url, zod } from "@synnaxlabs/x";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { z } from "zod";

import {
  type FetchResponse,
  HTTPClient,
  Timeout,
  unaryWithBreaker,
  Unreachable,
  UntrustedCertificate,
} from "@/index";

const ENDPOINT = new url.URL({
  host: "127.0.0.1",
  protocol: "http",
  port: 8080,
  pathPrefix: "unary",
});

const client = new HTTPClient({ endpoint: ENDPOINT, codec: new binary.JSONCodec() });

const messageZ = z.object({
  id: z.number().optional(),
  message: z.string().optional(),
});

const paramsZ = z.object({ fileName: z.string(), project: z.string() });

const encodeJSON = (value: unknown): Uint8Array<ArrayBuffer> =>
  new TextEncoder().encode(JSON.stringify(value));

describe("http", () => {
  test("echo", async () => {
    const response = await client.send(
      "/echo",
      { id: 1, message: "hello" },
      messageZ,
      messageZ,
    );
    expect(response).toEqual({ id: 2, message: "hello" });
  });

  test("not found", async () => {
    await expect(client.send("/not-found", {}, messageZ, messageZ)).rejects.toThrow(
      "Not Found",
    );
  });

  test("response parse failures name the target", async () => {
    const mismatchedZ = z.object({ id: z.string() });
    await expect(
      client.send("/echo", { id: 1, message: "hello" }, messageZ, mismatchedZ),
    ).rejects.toThrow("Failed to parse /echo response");
  });

  test("request params", async () => {
    const c = new HTTPClient({ endpoint: ENDPOINT, codec: new binary.JSONCodec() });
    const response = await c.send("/middlewareCheck", {}, messageZ, messageZ, {
      Test: "test",
    });
    expect(response.message).toEqual("");
  });

  test("request params missing", async () => {
    const c = new HTTPClient({ endpoint: ENDPOINT, codec: new binary.JSONCodec() });
    await expect(c.send("/middlewareCheck", {}, messageZ, messageZ)).rejects.toThrow();
  });

  test("middleware", async () => {
    client.use(async (md, next) => {
      md.params.Test = "test";
      return await next(md);
    });
    const response = await client.send("/middlewareCheck", {}, messageZ, messageZ);
    expect(response.message).toEqual("");
  });

  test("unreachable", async () => {
    const c = new HTTPClient({
      endpoint: new url.URL({
        host: "127.0.0.1",
        protocol: "http",
        port: 9999,
        pathPrefix: "unary",
      }),
      codec: new binary.JSONCodec(),
    });
    const send = c.send("/unreachable", {}, messageZ, messageZ);
    await expect(send).rejects.toThrow(Unreachable);
    await expect(send).rejects.toThrow("Unreachable");
  });

  describe("deno fetch errors", () => {
    const failWith = (cause: string): Promise<unknown> =>
      new HTTPClient({
        endpoint: ENDPOINT,
        codec: new binary.JSONCodec(),
        connector: {
          fetch: async () => {
            throw new TypeError("fetch failed", { cause: new Error(cause) });
          },
          dial: async () => {
            throw new Error("unused");
          },
        },
      }).send("/echo", {}, messageZ, messageZ);

    test("throws unreachable for a refused connection", async () => {
      await expect(
        failWith(
          "error sending request for url (http://127.0.0.1:9311/x): client error " +
            "(Connect): tcp connect error: Connection refused (os error 61)",
        ),
      ).rejects.toThrow(Unreachable);
    });

    test("throws an untrusted certificate for an unknown issuer", async () => {
      await expect(
        failWith(
          "error sending request for url (https://127.0.0.1:9312/x): client error " +
            "(Connect): invalid peer certificate: UnknownIssuer",
        ),
      ).rejects.toThrow(UntrustedCertificate);
    });

    test("does not throw unreachable for a TLS failure", async () => {
      const err = await failWith(
        "error sending request for url (https://127.0.0.1:9312/x): client error " +
          "(Connect): received corrupt message of type InvalidContentType",
      ).catch((e: unknown) => e);
      expect(Unreachable.matches(err)).toBe(false);
      expect(err).toBeInstanceOf(TypeError);
    });
  });

  describe("browser fetch failures", () => {
    const browserFailure = (): TypeError => new TypeError("Failed to fetch");

    const sendWith = (
      fetch: () => Promise<FetchResponse>,
      secure: boolean,
    ): Promise<unknown> =>
      new HTTPClient({
        endpoint: ENDPOINT,
        codec: new binary.JSONCodec(),
        secure,
        connector: {
          fetch,
          dial: async () => {
            throw new Error("unused");
          },
        },
      })
        .send("/echo", {}, messageZ, messageZ)
        .catch((e: unknown) => e);

    test("name a possible certificate problem with TLS", async () => {
      const err = await sendWith(async () => {
        throw browserFailure();
      }, true);
      expect(Unreachable.matches(err)).toBe(true);
      expect(err).toHaveProperty("message", expect.stringContaining("certificate"));
    });

    test("do not name a certificate without TLS", async () => {
      const err = await sendWith(async () => {
        throw browserFailure();
      }, false);
      expect(Unreachable.matches(err)).toBe(true);
      expect(err).toHaveProperty("message", "Unreachable");
    });

    test("do not name a certificate while the body is read", async () => {
      const err = await sendWith(
        async () => ({
          ok: true,
          status: 200,
          statusText: "OK",
          body: null,
          arrayBuffer: async () => {
            throw browserFailure();
          },
        }),
        true,
      );
      expect(Unreachable.matches(err)).toBe(true);
      expect(err).toHaveProperty("message", "Unreachable");
    });
  });

  test("keeps the runtime error for an expired certificate", async () => {
    const cause = Object.assign(new Error("certificate has expired"), {
      code: "CERT_HAS_EXPIRED",
    });
    const err = await new HTTPClient({
      endpoint: ENDPOINT,
      codec: new binary.JSONCodec(),
      connector: {
        fetch: async () => {
          throw new TypeError("fetch failed", { cause });
        },
        dial: async () => {
          throw new Error("unused");
        },
      },
    })
      .send("/echo", {}, messageZ, messageZ)
      .catch((e: unknown) => e);
    expect(UntrustedCertificate.matches(err)).toBe(false);
    expect(err).toBeInstanceOf(TypeError);
  });

  describe("GOAWAY refusals", () => {
    const refusal = (): TypeError => {
      const cause = Object.assign(
        new Error('HTTP/2: "GOAWAY" frame received with code 0'),
        { code: "UND_ERR_INFO" },
      );
      return new TypeError("fetch failed", { cause });
    };

    const refusingClient = (refusals: number) => {
      const bodies: unknown[] = [];
      const c = new HTTPClient({
        endpoint: ENDPOINT,
        codec: new binary.JSONCodec(),
        connector: {
          fetch: async (target, init) => {
            bodies.push(init.body);
            if (bodies.length <= refusals) throw refusal();
            return await fetch(target, init);
          },
          dial: async () => await Promise.reject(new Error("not used")),
        },
      });
      return { c, bodies };
    };

    test("should send a refused request again", async () => {
      const { c, bodies } = refusingClient(1);
      const res = await c.send("/echo", { id: 1, message: "hi" }, messageZ, messageZ);
      expect(res).toEqual({ id: 2, message: "hi" });
      expect(bodies).toHaveLength(2);
      expect(bodies[1]).toBe(bodies[0]);
    });

    test("should send a refused request again only once", async () => {
      const { c, bodies } = refusingClient(2);
      const err = await c
        .send("/echo", { id: 1 }, messageZ, messageZ)
        .catch((e: unknown) => e);
      expect(err).toBeInstanceOf(TypeError);
      expect(bodies).toHaveLength(2);
    });

    test("should not send a refused stream upload again", async () => {
      const { c, bodies } = refusingClient(1);
      const body = new ReadableStream<Uint8Array>({
        start(controller) {
          controller.enqueue(encodeJSON({ id: 1 }));
          controller.close();
        },
      });
      await expect(
        c.upload("/echo", body, { encoding: "JSON" }, messageZ),
      ).rejects.toThrow(TypeError);
      expect(bodies).toHaveLength(1);
    });

    test("should not send a request again for a GOAWAY with an error", async () => {
      const cause = Object.assign(
        new Error('HTTP/2: "GOAWAY" frame received with code 2'),
        { code: "UND_ERR_SOCKET" },
      );
      let calls = 0;
      const c = new HTTPClient({
        endpoint: ENDPOINT,
        codec: new binary.JSONCodec(),
        connector: {
          fetch: async () => {
            calls++;
            throw new TypeError("fetch failed", { cause });
          },
          dial: async () => await Promise.reject(new Error("not used")),
        },
      });
      await expect(c.send("/echo", {}, messageZ, messageZ)).rejects.toThrow();
      expect(calls).toBe(1);
    });
  });

  describe("timeout", () => {
    let server: Server;

    const serve = async (
      handler: RequestListener,
      timeout: CrudeTimeSpan = TimeSpan.milliseconds(50),
    ): Promise<HTTPClient> => {
      server = createServer(handler);
      await new Promise<void>((resolve) => server.listen(0, "127.0.0.1", resolve));
      const { port } = server.address() as AddressInfo;
      return new HTTPClient({
        endpoint: new url.URL({ host: "127.0.0.1", port }),
        codec: new binary.JSONCodec(),
        timeout,
      });
    };

    afterEach(async () => {
      server.closeAllConnections();
      await new Promise((resolve) => server.close(resolve));
    });

    test("rejects a send that the server never answers", async () => {
      const c = await serve(() => {});
      await expect(c.send("/stall", {}, messageZ, messageZ)).rejects.toThrow(
        /^\[freighter\] no response from http:\/\/127\.0\.0\.1:\d+\/stall within 50ms$/,
      );
    });

    test("throws a timeout that the breaker does not retry", async () => {
      let requests = 0;
      const c = await serve(() => {
        requests++;
      });
      const unary = unaryWithBreaker(c, {
        maxRetries: 2,
        baseInterval: TimeSpan.milliseconds(1),
      });
      const err = await unary
        .send("/stall", {}, messageZ, messageZ)
        .catch((e: unknown) => e);
      expect(Timeout.matches(err)).toBe(true);
      expect(Unreachable.matches(err)).toBe(false);
      expect((err as Error).cause).toMatchObject({ name: "TimeoutError" });
      expect(requests).toBe(1);
    });

    test("rejects a send whose response body never ends", async () => {
      const c = await serve((_, res) => {
        res.writeHead(200, { "Content-Type": "application/json" });
        res.write("{");
      });
      await expect(c.send("/stall", {}, messageZ, messageZ)).rejects.toThrow(
        "no response from",
      );
    });

    test.each([
      ["a zero", TimeSpan.ZERO],
      ["an infinite", Infinity],
      ["a 30 day", TimeSpan.days(30)],
      ["the largest", TimeSpan.MAX],
    ])("sends with no limit for %s timeout", async (_, timeout) => {
      const c = await serve((_, res) => {
        setTimeout(() => {
          res.writeHead(200, { "Content-Type": "application/json" });
          res.end(JSON.stringify({ id: 1 }));
        }, 20);
      }, timeout);
      expect(await c.send("/slow", {}, messageZ, messageZ)).toEqual({ id: 1 });
    });
  });

  describe("reset while reading the response body", () => {
    let server: Server;
    let c: HTTPClient;
    let requests: number;

    const serve = async (status: number): Promise<void> => {
      requests = 0;
      server = createServer((req, res) => {
        requests++;
        res.writeHead(status, {
          "Content-Type": "application/json",
          "Content-Length": "100",
        });
        res.write("{", () => req.socket.destroy());
      });
      await new Promise<void>((resolve) => server.listen(0, "127.0.0.1", resolve));
      const address = server.address();
      if (address == null || typeof address === "string")
        throw new Error("server is not listening on a port");
      c = new HTTPClient({
        endpoint: new url.URL({ host: "127.0.0.1", port: address.port }),
        codec: new binary.JSONCodec(),
      });
    };

    afterEach(async () => {
      server.closeAllConnections();
      await new Promise((resolve) => server.close(resolve));
    });

    test("send throws unreachable", async () => {
      await serve(200);
      await expect(c.send("/reset", {}, messageZ, messageZ)).rejects.toThrow(
        Unreachable,
      );
    });

    test("the breaker does not resend a delivered request", async () => {
      await serve(200);
      const unary = unaryWithBreaker(c, {
        maxRetries: 2,
        baseInterval: TimeSpan.milliseconds(1),
      });
      const err = await unary
        .send("/reset", {}, messageZ, messageZ)
        .catch((e: unknown) => e);
      expect(Unreachable.matches(err)).toBe(true);
      expect(requests).toBe(1);
    });

    test("upload throws unreachable", async () => {
      await serve(200);
      await expect(
        c.upload("/reset", encodeJSON({}), { encoding: "JSON" }, messageZ),
      ).rejects.toThrow(Unreachable);
    });

    test("download throws unreachable for an error response", async () => {
      await serve(400);
      await expect(
        c.download("/reset", {}, messageZ, { encoding: "JSON" }),
      ).rejects.toThrow(Unreachable);
    });
  });

  describe("upload", () => {
    test("blob body", async () => {
      const body = new Blob([JSON.stringify({ id: 1, message: "hello" })], {
        type: "application/json",
      });
      const response = await client.upload(
        "/echo",
        body,
        { encoding: "JSON" },
        messageZ,
      );
      expect(response).toEqual({ id: 2, message: "hello" });
    });

    test("bytes body", async () => {
      const body = encodeJSON({ id: 1, message: "hello" });
      const response = await client.upload(
        "/echo",
        body,
        { encoding: "JSON" },
        messageZ,
      );
      expect(response).toEqual({ id: 2, message: "hello" });
    });

    test("readable stream body", async () => {
      const bytes = encodeJSON({ id: 1, message: "hello" });
      const body = new ReadableStream<Uint8Array>({
        start(controller) {
          controller.enqueue(bytes);
          controller.close();
        },
      });
      const response = await client.upload(
        "/echo",
        body,
        { encoding: "JSON" },
        messageZ,
      );
      expect(response).toEqual({ id: 2, message: "hello" });
    });

    test("not found", async () => {
      await expect(
        client.upload("/not-found", encodeJSON({}), { encoding: "JSON" }, messageZ),
      ).rejects.toThrow("Not Found");
    });

    test("params reach the server as a single JSON request param", async () => {
      const response = await client.upload(
        "/paramEcho",
        encodeJSON({ message: "params" }),
        {
          encoding: "JSON",
          params: { fileName: "My Log.json", project: "project:abc" },
          paramsSchema: paramsZ,
        },
        messageZ,
      );
      expect(JSON.parse(response.message ?? "")).toEqual({
        file_name: "My Log.json",
        project: "project:abc",
      });
    });

    test("the server sees no request params when params are omitted", async () => {
      const response = await client.upload(
        "/paramEcho",
        encodeJSON({ message: "params" }),
        { encoding: "JSON" },
        messageZ,
      );
      expect(response.message).toEqual("");
    });

    test("params failing their schema reject before the request is sent", async () => {
      await expect(
        client.upload(
          "/paramEcho",
          encodeJSON({ message: "params" }),
          {
            encoding: "JSON",
            params: { fileName: "My Log.json", project: 42 as unknown as string },
            paramsSchema: paramsZ,
          },
          messageZ,
        ),
      ).rejects.toThrow(zod.ParseError);
    });

    test("unreachable", async () => {
      const c = new HTTPClient({
        endpoint: new url.URL({
          host: "127.0.0.1",
          protocol: "http",
          port: 9999,
          pathPrefix: "unary",
        }),
        codec: new binary.JSONCodec(),
      });
      await expect(
        c.upload("/echo", encodeJSON({}), { encoding: "JSON" }, messageZ),
      ).rejects.toThrow(Unreachable);
    });

    describe("in a Node runtime with a document", () => {
      beforeEach(() => {
        vi.stubGlobal("document", {});
        vi.resetModules();
      });
      afterEach(() => vi.unstubAllGlobals());

      test("should buffer a readable stream body to a blob", async () => {
        const { HTTPClient: RendererClient } = await import("@/index");
        let sent: unknown = null;
        const c = new RendererClient({
          endpoint: ENDPOINT,
          codec: new binary.JSONCodec(),
          connector: {
            fetch: async (_, init) => {
              sent = init.body;
              return new Response(encodeJSON({ id: 2 }));
            },
            dial: async () => await Promise.reject(new Error("not used")),
          },
        });
        const body = new ReadableStream<Uint8Array>({
          start(controller) {
            controller.enqueue(encodeJSON({ id: 1 }));
            controller.close();
          },
        });
        await c.upload("/echo", body, { encoding: "JSON" }, messageZ);
        expect(sent).toBeInstanceOf(Blob);
      });
    });
  });

  describe("download", () => {
    test("returns the response body as a byte stream", async () => {
      const stream = await client.download(
        "/echo",
        { id: 1, message: "hello" },
        messageZ,
        { encoding: "JSON" },
      );
      expect(stream).toBeInstanceOf(ReadableStream);
      const decoded = await new Response(stream).json();
      expect(decoded).toEqual({ id: 2, message: "hello" });
    });

    test("sends a contentType encoding verbatim as the Accept header", async () => {
      const stream = await client.download(
        "/echo",
        { id: 1, message: "hello" },
        messageZ,
        { encoding: { contentType: "application/json" } },
      );
      const decoded = await new Response(stream).json();
      expect(decoded).toEqual({ id: 2, message: "hello" });
    });

    test("rejects a contentType the server cannot produce", async () => {
      await expect(
        client.download("/echo", {}, messageZ, {
          encoding: { contentType: "application/vnd.unknown" },
        }),
      ).rejects.toThrow("Not Acceptable");
    });

    test("params reach the server as a single JSON request param", async () => {
      const stream = await client.download(
        "/paramEcho",
        { message: "params" },
        messageZ,
        {
          encoding: "JSON",
          params: { fileName: "My Log.json", project: "project:abc" },
          paramsSchema: paramsZ,
        },
      );
      const decoded = await new Response(stream).json();
      expect(JSON.parse(decoded.message)).toEqual({
        file_name: "My Log.json",
        project: "project:abc",
      });
    });

    test("the server sees no request params when params are omitted", async () => {
      const stream = await client.download(
        "/paramEcho",
        { message: "params" },
        messageZ,
        { encoding: "JSON" },
      );
      const decoded = await new Response(stream).json();
      expect(decoded.message).toEqual("");
    });

    test("not found", async () => {
      await expect(
        client.download("/not-found", {}, messageZ, { encoding: "JSON" }),
      ).rejects.toThrow("Not Found");
    });

    const TRUNCATED = { encoding: { contentType: "application/x-truncated" } };

    test("rejects with the error of an encoder that fails before it writes", async () => {
      await expect(
        client.download("/truncated", { id: 0, message: "ab" }, messageZ, TRUNCATED),
      ).rejects.toThrow("truncating encoder failed partway");
    });

    test("rejects a body the server cuts short", async () => {
      const read = async (): Promise<ArrayBuffer> => {
        const stream = await client.download(
          "/truncated",
          { id: 1 << 14, message: "ab" },
          messageZ,
          TRUNCATED,
        );
        return await new Response(stream).arrayBuffer();
      };
      // The cut can land before fetch resolves the response or while the body is
      // read, depending on how the bytes arrive.
      await expect(read()).rejects.toSatisfy(
        (e) =>
          e instanceof Unreachable ||
          (e instanceof TypeError && e.message === "terminated"),
      );
    });

    test("unreachable", async () => {
      const c = new HTTPClient({
        endpoint: new url.URL({
          host: "127.0.0.1",
          protocol: "http",
          port: 9999,
          pathPrefix: "unary",
        }),
        codec: new binary.JSONCodec(),
      });
      await expect(
        c.download("/echo", {}, messageZ, { encoding: "JSON" }),
      ).rejects.toThrow(Unreachable);
    });
  });
});
