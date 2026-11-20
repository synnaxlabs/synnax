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
import type { IncomingMessage } from "node:http";
import { createServer, type Server } from "node:https";
import { type AddressInfo, createServer as createTCPServer } from "node:net";
import { tmpdir } from "node:os";
import { join } from "node:path";

import { binary, url } from "@synnaxlabs/x";
import { afterAll, afterEach, beforeAll, describe, expect, it, vi } from "vitest";
import { WebSocketServer } from "ws";
import { z } from "zod";

import { HTTPClient, trustCA, Unreachable, WebSocketClient } from "@/index";

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
  let upgrade: IncomingMessage | undefined;

  beforeAll(async () => {
    dir = mkdtempSync(join(tmpdir(), "freighter-ca-"));
    createCertificates(dir);
    ca = readFileSync(join(dir, "ca.crt"), "utf8");
    server = createServer(
      {
        key: readFileSync(join(dir, "server.key")),
        cert: readFileSync(join(dir, "server.crt")),
      },
      (req, res) => {
        res.setHeader("Content-Type", "application/json");
        req.pipe(res);
      },
    );
    wss = new WebSocketServer({ server });
    wss.on("connection", (ws, req) => {
      upgrade = req;
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

  const httpClient = () =>
    new HTTPClient({
      endpoint,
      codec: binary.JSON_CODEC,
      secure: true,
      connector: trustCA(ca),
    });

  it("should send a request to a server that the CA signs", async () => {
    const res = await httpClient().send("/", { message: "hi" }, messageZ, messageZ);
    expect(res).toEqual({ message: "hi" });
  });

  it("should upload a Blob to a server that the CA signs", async () => {
    const body = new Blob([JSON.stringify({ message: "hi" })]);
    const res = await httpClient().upload("/", body, { encoding: "JSON" }, messageZ);
    expect(res).toEqual({ message: "hi" });
  });

  it("should download from a server that the CA signs", async () => {
    const stream = await httpClient().download("/", { message: "hi" }, messageZ, {
      encoding: "JSON",
    });
    expect(await new Response(stream).json()).toEqual({ message: "hi" });
  });

  it("should open a stream to a server that the CA signs", async () => {
    const client = new WebSocketClient({
      endpoint,
      codec: binary.JSON_CODEC,
      secure: true,
      connector: trustCA(ca),
    });
    client.use(async (ctx, next) => {
      ctx.params.Authorization = "Bearer test-token";
      return await next(ctx);
    });
    const stream = await client.stream("/", messageZ, messageZ);
    expect(await stream.receive()).toEqual({ message: "hi" });
    stream.closeSend();
    expect(upgrade?.headers.authorization).toEqual("Bearer test-token");
    expect(upgrade?.url).not.toContain("test-token");
  });

  it("should reject a request without the CA", async () => {
    const client = new HTTPClient({ endpoint, codec: binary.JSON_CODEC, secure: true });
    await expect(
      client.send("/", { message: "" }, messageZ, messageZ),
    ).rejects.toMatchObject({ cause: { code: "UNABLE_TO_VERIFY_LEAF_SIGNATURE" } });
  });

  it("should reject a stream without the CA", async () => {
    const client = new WebSocketClient({
      endpoint,
      codec: binary.JSON_CODEC,
      secure: true,
    });
    // Node hides the TLS cause of a failed WebSocket dial.
    const err = await client.stream("/", messageZ, messageZ).then(
      () => null,
      (e: unknown) => e,
    );
    expect(Unreachable.matches(err)).toBe(true);
  });

  it("should throw Unreachable when a stream cannot connect", async () => {
    const tcp = createTCPServer();
    await new Promise<void>((resolve) => tcp.listen(0, "127.0.0.1", resolve));
    const { port } = tcp.address() as AddressInfo;
    await new Promise((resolve) => tcp.close(resolve));
    const client = new WebSocketClient({
      endpoint: new url.URL({ host: "127.0.0.1", port }),
      codec: binary.JSON_CODEC,
      connector: trustCA(ca),
    });
    const err = await client.stream("/", messageZ, messageZ).then(
      () => null,
      (e: unknown) => e,
    );
    expect(Unreachable.matches(err)).toBe(true);
  });

  describe("in runtimes without a CA option for WebSockets", () => {
    afterEach(() => {
      vi.unstubAllGlobals();
    });

    it("should tell a Deno user to run Deno with --cert", async () => {
      vi.stubGlobal("Deno", { createHttpClient: () => ({}) });
      await expect(trustCA(ca).dial("wss://127.0.0.1", {})).rejects.toThrow("--cert");
    });

    it("should throw in a browser", () => {
      vi.stubGlobal("process", undefined);
      expect(() => trustCA(ca)).toThrow("browser");
    });
  });
});
