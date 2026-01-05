// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { binary, URL } from "@synnaxlabs/x";
import { describe, expect, test } from "vitest";
import { z } from "zod";

import { Unreachable } from "@/errors";
import { HTTPClient } from "@/http";

const ENDPOINT = new URL({
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
    const [response, error] = await client.send<typeof messageZ>(
      "/echo",
      {
        id: 1,
        message: "hello",
      },
      messageZ,
      messageZ,
    );
    expect(error).toBeNull();
    expect(response).toEqual({ id: 2, message: "hello" });
  });

  test("not found", async () => {
    const [response, error] = await client.send<typeof messageZ>(
      "/not-found",
      {},
      messageZ,
      messageZ,
    );
    expect(error?.message).toEqual("Not Found");
    expect(response).toBeNull();
  });

  test("middleware", async () => {
    client.use(async (md, next) => {
      md.params.Test = "test";
      return await next(md);
    });
    const [response, error] = await client.send<typeof messageZ>(
      "/middlewareCheck",
      {},
      messageZ,
      messageZ,
    );
    expect(error).toBeNull();
    expect(response?.message).toEqual("");
  });

  test("unreachable", async () => {
    const c = new HTTPClient(
      new URL({
        host: "127.0.0.1",
        protocol: "http",
        port: 9999,
        pathPrefix: "unary",
      }),
      new binary.JSONCodec(),
    );
    const [response, error] = await c.send<typeof messageZ>(
      "/unreachable",
      {},
      messageZ,
      messageZ,
    );
    expect(error).toBeInstanceOf(Unreachable);
    expect(error?.message).toEqual("Unreachable");
    expect(response).toBeNull();
  });
});
