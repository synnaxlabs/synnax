// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createServer, type Server } from "node:http";

import { Unreachable } from "@synnaxlabs/freighter";
import { TimeSpan, url } from "@synnaxlabs/x";
import { afterEach, describe, expect, it } from "vitest";
import { z } from "zod";

import { errorsMiddleware } from "@/errors";
import { Transport } from "@/transport";

describe("Transport", () => {
  let server: Server;

  afterEach(async () => {
    server.closeAllConnections();
    await new Promise((resolve) => server.close(resolve));
  });

  it("should not resend a request the Core received", async () => {
    let requests = 0;
    server = createServer((req, res) => {
      requests++;
      res.writeHead(200, {
        "Content-Type": "application/json",
        "Content-Length": "100",
      });
      res.write("{", () => req.socket.destroy());
    });
    await new Promise<void>((resolve) => server.listen(0, "127.0.0.1", resolve));
    const address = server.address();
    if (address == null || typeof address === "string")
      throw new Error("server is not listening on a port");
    const transport = new Transport({
      url: new url.URL({ host: "127.0.0.1", port: address.port }),
      retry: { maxRetries: 2, baseInterval: TimeSpan.milliseconds(1) },
    });
    transport.use(errorsMiddleware);
    const err = await transport.unary
      .send("/reset", {}, z.object({}), z.object({}))
      .catch((e: unknown) => e);
    expect(Unreachable.matches(err)).toBe(true);
    expect(requests).toBe(1);
  });
});
