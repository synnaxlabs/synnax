// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { URL } from "@synnaxlabs/x";
import { describe, expect, test } from "vitest";
import { z } from "zod";

import { JSONEncoderDecoder } from "@/encoder";
import { HTTPClientFactory } from "@/http";

const ENDPOINT = new URL({
  host: "127.0.0.1",
  port: 8080,
  pathPrefix: "unary",
});

const factory = new HTTPClientFactory(ENDPOINT, new JSONEncoderDecoder());

const messageZ = z.object({
  id: z.number().optional(),
  message: z.string().optional(),
});

const getClient = factory.newGET();
const postClient = factory.newPOST();

describe("http", () => {
  test("post echo", async () => {
    const [response, error] = await postClient.send<typeof messageZ>(
      "/echo",
      {
        id: 1,
        message: "hello",
      },
      messageZ
    );
    expect(error).toBeNull();
    expect(response).toEqual({ id: 2, message: "hello" });
  });

  test("get echo", async () => {
    const [response, error] = await getClient.send<typeof messageZ>(
      "/echo",
      {
        id: 1,
        message: "hello",
      },
      messageZ
    );
    expect(error).toBeNull();
    expect(response).toEqual({ id: 2, message: "hello" });
  });

  test("get not found", async () => {
    const [response, error] = await getClient.send<typeof messageZ>(
      "/not-found",
      {},
      messageZ
    );
    expect(error?.message).toEqual("Cannot GET /unary/not-found");
    expect(response).toBeNull();
  });

  test("post not found", async () => {
    const [response, error] = await postClient.send<typeof messageZ>(
      "/not-found",
      {},
      messageZ
    );
    expect(error?.message).toEqual("Cannot POST /unary/not-found");
    expect(response).toBeNull();
  });

  test("middleware", async () => {
    const client = factory.newGET();
    client.use(async (md, next) => {
      md.params.Test = "test";
      return await next(md);
    });
    const [response, error] = await client.send<typeof messageZ>(
      "/middlewareCheck",
      {},
      messageZ
    );
    expect(error).toBeNull();
    expect(response?.message).toEqual("");
  });
});
