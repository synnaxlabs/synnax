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
import {
  createServer as createHTTPServer,
  type IncomingMessage,
  type Server as HTTPServer,
} from "node:http";
import { createServer, type Server } from "node:https";
import { type AddressInfo, createServer as createTCPServer } from "node:net";
import { tmpdir } from "node:os";
import { join } from "node:path";

import { binary, sleep, TimeSpan, url } from "@synnaxlabs/x";
import { afterAll, afterEach, beforeAll, describe, expect, it, vi } from "vitest";
import { WebSocketServer } from "ws";
import { z } from "zod";

import { HTTPClient, trustCA, Unreachable, WebSocketClient } from "@/index";

const messageZ = z.object({ message: z.string() });

const KEY_ARGS = "-newkey rsa:2048 -nodes -days 1";

const openssl = (dir: string, args: string): void => {
  execFileSync("openssl", args.split(" "), { cwd: dir, stdio: "ignore" });
};

// Writes a test CA and returns its PEM text.
const createCA = (dir: string, name: string): string => {
  openssl(
    dir,
    `req -x509 ${KEY_ARGS} -keyout ${name}.key -out ${name}.crt -subj /CN=${name}`,
  );
  return readFileSync(join(dir, `${name}.crt`), "utf8");
};

// Writes a test CA and a certificate for 127.0.0.1 that the CA signs.
const createCertificates = (dir: string): void => {
  createCA(dir, "ca");
  openssl(
    dir,
    `req ${KEY_ARGS} -keyout server.key -out server.csr -subj /CN=127.0.0.1`,
  );
  writeFileSync(join(dir, "ext.cnf"), "subjectAltName=IP:127.0.0.1\n");
  openssl(
    dir,
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

  describe("in Node versions without tls.getCACertificates", () => {
    beforeAll(() => {
      const tls = {
        ...process.getBuiltinModule("node:tls"),
        getCACertificates: undefined,
      };
      const getBuiltinModule = process.getBuiltinModule.bind(process);
      vi.spyOn(process, "getBuiltinModule").mockImplementation((id: string) =>
        id === "node:tls" ? tls : getBuiltinModule(id),
      );
    });

    afterAll(() => {
      vi.restoreAllMocks();
    });

    afterEach(() => {
      vi.unstubAllEnvs();
    });

    it("should keep the certificates in NODE_EXTRA_CA_CERTS", async () => {
      vi.stubEnv("NODE_EXTRA_CA_CERTS", join(dir, "ca.crt"));
      const client = new HTTPClient({
        endpoint,
        codec: binary.JSON_CODEC,
        secure: true,
        connector: trustCA(createCA(dir, "other")),
      });
      const res = await client.send("/", { message: "hi" }, messageZ, messageZ);
      expect(res).toEqual({ message: "hi" });
    });

    it("should skip a NODE_EXTRA_CA_CERTS file that does not exist", async () => {
      vi.stubEnv("NODE_EXTRA_CA_CERTS", join(dir, "missing.crt"));
      const res = await httpClient().send("/", { message: "hi" }, messageZ, messageZ);
      expect(res).toEqual({ message: "hi" });
    });
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

describe("default connector", () => {
  // The Core closes a connection after two minutes idle and sends no Keep-Alive header,
  // so the client keeps its own idle timeout.
  const CORE_IDLE_TIMEOUT = TimeSpan.minutes(2);
  // Undici's default idle timeout is 4 s, on a native timer.
  const PAST_CLIENT_IDLE_TIMEOUT = TimeSpan.milliseconds(4500);
  let server: HTTPServer;
  let client: HTTPClient;
  let connections = 0;

  const send = async (): Promise<void> => {
    const res = await client.send("/", { message: "hi" }, messageZ, messageZ);
    expect(res).toEqual({ message: "hi" });
  };

  beforeAll(async () => {
    server = createHTTPServer((req, res) => {
      res.setHeader("Content-Type", "application/json");
      req.pipe(res);
    });
    server.keepAliveTimeout = 0;
    server.on("connection", (socket) => {
      connections++;
      socket.setTimeout(CORE_IDLE_TIMEOUT.milliseconds, () => socket.destroy());
    });
    await new Promise<void>((resolve) => server.listen(0, "127.0.0.1", resolve));
    const { port } = server.address() as AddressInfo;
    client = new HTTPClient({
      endpoint: new url.URL({ host: "127.0.0.1", port }),
      codec: binary.JSON_CODEC,
    });
  });

  afterAll(async () => {
    server.closeAllConnections();
    await new Promise((resolve) => server.close(resolve));
  });

  it("should drop an idle socket long before the Core closes it", async () => {
    await send();
    // A socket returns to the pool shortly after its response ends.
    await sleep.sleep(TimeSpan.milliseconds(100));
    await send();
    expect(connections).toBe(1);
    await sleep.sleep(PAST_CLIENT_IDLE_TIMEOUT);
    await send();
    expect(connections).toBe(2);
  }, 10_000);
});
