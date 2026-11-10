// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createServer, type Server } from "node:http";
import { type AddressInfo } from "node:net";

import { afterAll, beforeAll, describe, expect, it } from "vitest";

import { sleep } from "@/sleep";
import { shortenKeepAlive } from "@/testutil/keepAlive";

describe("shortenKeepAlive", () => {
  let server: Server;
  let url: string;
  let sockets = 0;

  const request = async (): Promise<void> => {
    await (await fetch(url)).text();
  };

  beforeAll(async () => {
    shortenKeepAlive();
    server = createServer((_, res) => res.end("ok"));
    // A Keep-Alive header would override the client timeout. The Core sends none.
    server.keepAliveTimeout = 0;
    server.on("connection", () => sockets++);
    await new Promise<void>((resolve) => server.listen(0, "127.0.0.1", resolve));
    url = `http://127.0.0.1:${(server.address() as AddressInfo).port}`;
  });

  afterAll(() => {
    server.closeAllConnections();
    server.close();
  });

  it("should reuse a recent socket and drop one idle for over 1 s", async () => {
    await request();
    // A socket returns to the pool shortly after its response ends.
    await sleep.sleep(100);
    await request();
    expect(sockets).toBe(1);
    await sleep.sleep(1500);
    await request();
    expect(sockets).toBe(2);
  });
});
