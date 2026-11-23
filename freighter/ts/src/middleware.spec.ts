// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { binary, url } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { type Context, HTTPClient, type Middleware } from "@/index";

const CTX: Context = { target: "test", role: "client", protocol: "test", params: {} };

describe("middleware", () => {
  it("should run the rest of the chain each time a middleware calls next", async () => {
    const client = new HTTPClient({
      endpoint: new url.URL({ host: "127.0.0.1", port: 8080 }),
      codec: binary.JSON_CODEC,
    });
    const calls: string[] = [];
    const retry: Middleware = async (ctx, next) => {
      await next(ctx);
      return await next(ctx);
    };
    const record: Middleware = async (ctx, next) => {
      calls.push("record");
      return await next(ctx);
    };
    client.use(retry, record);
    await client.executeMiddleware(CTX, async (ctx) => {
      calls.push("finalizer");
      return ctx;
    });
    expect(calls).toEqual(["record", "finalizer", "record", "finalizer"]);
  });
});
