// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { binary, url } from "@synnaxlabs/x";
import { describe, expect, test, vi } from "vitest";
import { z } from "zod";

import { Unreachable } from "@/errors";
import { HTTPClient } from "@/http";

const ENDPOINT = new url.URL({
  host: "127.0.0.1",
  protocol: "http",
  port: 8080,
  pathPrefix: "unary",
});

const client = new HTTPClient(ENDPOINT, new binary.JSONCodec());

const messageZ = z.object({
  id: z.number().optional(),
  message: z.string().optional(),
});

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

  test("middleware", async () => {
    client.use(async (md, next) => {
      md.params.Test = "test";
      return await next(md);
    });
    const response = await client.send("/middlewareCheck", {}, messageZ, messageZ);
    expect(response.message).toEqual("");
  });

  test("unreachable", async () => {
    const c = new HTTPClient(
      new url.URL({
        host: "127.0.0.1",
        protocol: "http",
        port: 9999,
        pathPrefix: "unary",
      }),
      new binary.JSONCodec(),
    );
    const send = c.send("/unreachable", {}, messageZ, messageZ);
    await expect(send).rejects.toThrow(Unreachable);
    await expect(send).rejects.toThrow("Unreachable");
  });

  describe("upload", () => {
    test("string body", async () => {
      const response = await client.upload(
        "/echo",
        JSON.stringify({ id: 1, message: "hello" }),
        { encoding: "JSON" },
        messageZ,
      );
      expect(response).toEqual({ id: 2, message: "hello" });
    });

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

    test("array buffer view body", async () => {
      const body = new TextEncoder().encode(
        JSON.stringify({ id: 1, message: "hello" }),
      );
      const response = await client.upload(
        "/echo",
        body,
        { encoding: "JSON" },
        messageZ,
      );
      expect(response).toEqual({ id: 2, message: "hello" });
    });

    test("readable stream body", async () => {
      const bytes = new TextEncoder().encode(
        JSON.stringify({ id: 1, message: "hello" }),
      );
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
        client.upload("/not-found", JSON.stringify({}), { encoding: "JSON" }, messageZ),
      ).rejects.toThrow("Not Found");
    });

    test("query params are percent-encoded and appended to the URL", async () => {
      const fetchSpy = vi
        .spyOn(globalThis, "fetch")
        .mockResolvedValue(
          new Response(JSON.stringify({ id: 1, message: "ok" }), { status: 200 }),
        );
      try {
        await client.upload(
          "/echo",
          JSON.stringify({}),
          {
            encoding: "JSON",
            params: {
              file_name: "My Log.json",
              parent: "project:abc",
              skip: undefined,
            },
          },
          messageZ,
        );
        const target = fetchSpy.mock.calls[0][0] as string;
        expect(target).toContain("/unary/echo?");
        expect(target).toContain("file_name=My+Log.json");
        expect(target).toContain("parent=project%3Aabc");
        expect(target).not.toContain("skip");
      } finally {
        fetchSpy.mockRestore();
      }
    });

    test("the URL carries no query string when params are omitted", async () => {
      const fetchSpy = vi
        .spyOn(globalThis, "fetch")
        .mockResolvedValue(
          new Response(JSON.stringify({ id: 1, message: "ok" }), { status: 200 }),
        );
      try {
        await client.upload(
          "/echo",
          JSON.stringify({}),
          { encoding: "JSON" },
          messageZ,
        );
        expect(fetchSpy.mock.calls[0][0] as string).not.toContain("?");
      } finally {
        fetchSpy.mockRestore();
      }
    });

    test("unreachable", async () => {
      const c = new HTTPClient(
        new url.URL({
          host: "127.0.0.1",
          protocol: "http",
          port: 9999,
          pathPrefix: "unary",
        }),
        new binary.JSONCodec(),
      );
      await expect(
        c.upload("/echo", JSON.stringify({}), { encoding: "JSON" }, messageZ),
      ).rejects.toThrow(Unreachable);
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

    test("not found", async () => {
      await expect(
        client.download("/not-found", {}, messageZ, { encoding: "JSON" }),
      ).rejects.toThrow("Not Found");
    });

    test("unreachable", async () => {
      const c = new HTTPClient(
        new url.URL({
          host: "127.0.0.1",
          protocol: "http",
          port: 9999,
          pathPrefix: "unary",
        }),
        new binary.JSONCodec(),
      );
      await expect(
        c.download("/echo", {}, messageZ, { encoding: "JSON" }),
      ).rejects.toThrow(Unreachable);
    });
  });
});
