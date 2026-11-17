// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { execFileSync } from "node:child_process";
import { mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { createServer, type Server } from "node:https";
import { type AddressInfo } from "node:net";
import { tmpdir } from "node:os";
import { join } from "node:path";

import { binary, url } from "@synnaxlabs/x";
import { afterAll, beforeAll, describe, expect, it } from "vitest";
import { WebSocketServer } from "ws";
import { z } from "zod";

import { HTTPClient, trustCA, WebSocketClient } from "@/index";

const messageZ = z.object({ message: z.string() });

// Writes a test CA and a certificate for 127.0.0.1 that the CA signs.
const createCertificates = (dir: string): void => {
  const openssl = (args: string): void => {
    execFileSync("openssl", args.split(" "), { cwd: dir, stdio: "ignore" });
  };
  const key = "-newkey rsa:2048 -nodes -days 1";
  openssl(`req -x509 ${key} -keyout ca.key -out ca.crt -subj /CN=ca`);
  openssl(`req ${key} -keyout server.key -out server.csr -subj /CN=127.0.0.1`);
  writeFileSync(join(dir, "ext.cnf"), "subjectAltName=IP:127.0.0.1\n");
  openssl(
    "x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial " +
      "-out server.crt -days 1 -extfile ext.cnf",
  );
};

describe("trustCA", () => {
  let dir: string;
  let ca: string;
  let server: Server;
  let wss: WebSocketServer;
  let endpoint: url.URL;

  beforeAll(async () => {
    dir = mkdtempSync(join(tmpdir(), "freighter-ca-"));
    createCertificates(dir);
    ca = readFileSync(join(dir, "ca.crt"), "utf8");
    server = createServer(
      {
        key: readFileSync(join(dir, "server.key")),
        cert: readFileSync(join(dir, "server.crt")),
      },
      (_, res) => {
        res.setHeader("Content-Type", "application/json");
        res.end(JSON.stringify({ message: "hello" }));
      },
    );
    wss = new WebSocketServer({ server });
    wss.on("connection", (ws) => {
      ws.send(Buffer.from(JSON.stringify({ type: "open" })));
      ws.send(
        Buffer.from(JSON.stringify({ type: "data", payload: { message: "hi" } })),
      );
    });
    await new Promise<void>((resolve) => server.listen(0, "127.0.0.1", resolve));
    const { port } = server.address() as AddressInfo;
    endpoint = new url.URL({ host: "127.0.0.1", port });
  });

  afterAll(async () => {
    wss.clients.forEach((ws) => ws.terminate());
    server.closeAllConnections();
    await new Promise((resolve) => server.close(resolve));
    rmSync(dir, { recursive: true });
  });

  it("should send a request to a server that the CA signs", async () => {
    const client = new HTTPClient(
      endpoint,
      binary.JSON_CODEC,
      true,
      undefined,
      trustCA(ca),
    );
    const res = await client.send("/", { message: "" }, messageZ, messageZ);
    expect(res).toEqual({ message: "hello" });
  });

  it("should open a stream to a server that the CA signs", async () => {
    const client = new WebSocketClient(
      endpoint,
      binary.JSON_CODEC,
      true,
      undefined,
      trustCA(ca),
    );
    const stream = await client.stream("/", messageZ, messageZ);
    expect(await stream.receive()).toEqual({ message: "hi" });
    stream.closeSend();
  });

  it("should reject a server that the CA signs without the CA", async () => {
    const client = new HTTPClient(endpoint, binary.JSON_CODEC, true);
    await expect(
      client.send("/", { message: "" }, messageZ, messageZ),
    ).rejects.toMatchObject({ cause: { code: "UNABLE_TO_VERIFY_LEAF_SIGNATURE" } });
  });
});
