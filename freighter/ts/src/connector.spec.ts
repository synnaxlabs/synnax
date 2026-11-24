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
  type RequestListener,
  type Server as HTTPServer,
} from "node:http";
import { createServer, type Server } from "node:https";
import {
  type AddressInfo,
  connect,
  createServer as createTCPServer,
  type Server as TCPServer,
  type Socket,
} from "node:net";
import { tmpdir } from "node:os";
import { join } from "node:path";

import { binary, sleep, TimeSpan, url } from "@synnaxlabs/x";
import {
  afterAll,
  afterEach,
  beforeAll,
  beforeEach,
  describe,
  expect,
  it,
  vi,
} from "vitest";
import { WebSocketServer } from "ws";
import { z } from "zod";

import {
  HTTPClient,
  trustCA,
  Unreachable,
  UntrustedCertificate,
  WebSocketClient,
} from "@/index";

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

  it("should reject a request without the CA as an untrusted certificate", async () => {
    const client = new HTTPClient({ endpoint, codec: binary.JSON_CODEC, secure: true });
    const err = await client.send("/", { message: "" }, messageZ, messageZ).then(
      () => null,
      (e: unknown) => e,
    );
    expect(UntrustedCertificate.matches(err)).toBe(true);
    expect(err).toHaveProperty(
      "message",
      expect.stringMatching(/certificate of .* is not trusted.*caCert/),
    );
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
    expect(err).toHaveProperty("message", expect.stringContaining("certificate"));
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

  describe("with the CA in the system certificate store", () => {
    beforeAll(() => {
      const tls = process.getBuiltinModule("node:tls");
      const getCACertificates: typeof tls.getCACertificates = (type) =>
        type === "system" ? [ca] : tls.getCACertificates(type);
      const getBuiltinModule = process.getBuiltinModule.bind(process);
      vi.spyOn(process, "getBuiltinModule").mockImplementation((id: string) =>
        id === "node:tls" ? { ...tls, getCACertificates } : getBuiltinModule(id),
      );
    });

    afterAll(() => {
      vi.restoreAllMocks();
    });

    it("should send a request without the CA", async () => {
      const client = new HTTPClient({
        endpoint,
        codec: binary.JSON_CODEC,
        secure: true,
      });
      const res = await client.send("/", { message: "hi" }, messageZ, messageZ);
      expect(res).toEqual({ message: "hi" });
    });

    it("should trust only the caCert, not the system store", async () => {
      const client = new HTTPClient({
        endpoint,
        codec: binary.JSON_CODEC,
        secure: true,
        connector: trustCA(createCA(dir, "other")),
      });
      const err = await client.send("/", { message: "" }, messageZ, messageZ).then(
        () => null,
        (e: unknown) => e,
      );
      expect(UntrustedCertificate.matches(err)).toBe(true);
    });

    it("should open a stream without the CA", async () => {
      const client = new WebSocketClient({
        endpoint,
        codec: binary.JSON_CODEC,
        secure: true,
      });
      const stream = await client.stream("/", messageZ, messageZ);
      expect(await stream.receive()).toEqual({ message: "hi" });
      stream.closeSend();
    });
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

    it("should not trust NODE_EXTRA_CA_CERTS with a caCert", async () => {
      vi.stubEnv("NODE_EXTRA_CA_CERTS", join(dir, "ca.crt"));
      const client = new HTTPClient({
        endpoint,
        codec: binary.JSON_CODEC,
        secure: true,
        connector: trustCA(createCA(dir, "other")),
      });
      const err = await client.send("/", { message: "" }, messageZ, messageZ).then(
        () => null,
        (e: unknown) => e,
      );
      expect(UntrustedCertificate.matches(err)).toBe(true);
    });

    it("should trust only Node's defaults without the CA", async () => {
      const client = new HTTPClient({
        endpoint,
        codec: binary.JSON_CODEC,
        secure: true,
      });
      const err = await client.send("/", { message: "" }, messageZ, messageZ).then(
        () => null,
        (e: unknown) => e,
      );
      expect(UntrustedCertificate.matches(err)).toBe(true);
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

const echo: RequestListener = (req, res) => {
  res.setHeader("Content-Type", "application/json");
  req.pipe(res);
};

const serveStreams = (server: HTTPServer | Server): WebSocketServer => {
  const wss = new WebSocketServer({ server });
  wss.on("connection", (ws) => {
    ws.send(Buffer.from(JSON.stringify({ type: "open" })));
    ws.send(Buffer.from(JSON.stringify({ type: "data", payload: { message: "hi" } })));
  });
  return wss;
};

const listen = async (server: HTTPServer | Server | TCPServer): Promise<number> => {
  await new Promise<void>((resolve) => server.listen(0, "127.0.0.1", resolve));
  const address = server.address();
  if (address == null || typeof address === "string")
    throw new Error("server has no port");
  return address.port;
};

describe("proxies", () => {
  let dir: string;
  let ca: string;
  let plain: HTTPServer;
  let tls: Server;
  let proxy: HTTPServer;
  let plainEndpoint: url.URL;
  let tlsEndpoint: url.URL;
  let tunnels: string[];
  // Servers do not close upgraded or tunneled sockets.
  const sockets: Socket[] = [];
  let proxyURL: string;
  let refusal: string | null;

  beforeAll(async () => {
    dir = mkdtempSync(join(tmpdir(), "freighter-proxy-"));
    createCertificates(dir);
    ca = readFileSync(join(dir, "ca.crt"), "utf8");
    plain = createHTTPServer(echo);
    serveStreams(plain);
    tls = createServer(
      {
        key: readFileSync(join(dir, "server.key")),
        cert: readFileSync(join(dir, "server.crt")),
      },
      echo,
    );
    serveStreams(tls);
    proxy = createHTTPServer((_, res) => res.writeHead(502).end());
    proxy.on("connect", (req: IncomingMessage, socket, head) => {
      const target = new URL(`http://${req.url}`);
      tunnels.push(target.port);
      if (refusal != null) {
        socket.end(`HTTP/1.1 ${refusal}\r\n\r\n`);
        return;
      }
      const upstream = connect(Number(target.port), target.hostname, () => {
        socket.write("HTTP/1.1 200 Connection Established\r\n\r\n");
        upstream.write(head);
        upstream.pipe(socket);
        socket.pipe(upstream);
      });
      sockets.push(upstream);
      upstream.on("error", () => socket.end("HTTP/1.1 502 Bad Gateway\r\n\r\n"));
      socket.on("error", () => upstream.destroy());
    });
    for (const server of [plain, tls, proxy])
      server.on("connection", (socket: Socket) => sockets.push(socket));
    plainEndpoint = new url.URL({ host: "127.0.0.1", port: await listen(plain) });
    tlsEndpoint = new url.URL({ host: "127.0.0.1", port: await listen(tls) });
    proxyURL = `http://127.0.0.1:${await listen(proxy)}`;
    vi.stubEnv("HTTP_PROXY", proxyURL);
    vi.stubEnv("HTTPS_PROXY", proxyURL);
    // Undici reads the lowercase variables first.
    for (const name of ["http_proxy", "https_proxy", "no_proxy"])
      vi.stubEnv(name, undefined);
  });

  afterAll(async () => {
    vi.unstubAllEnvs();
    sockets.forEach((socket) => socket.destroy());
    await Promise.all(
      [plain, tls, proxy].map(
        async (server) => await new Promise((resolve) => server.close(resolve)),
      ),
    );
    rmSync(dir, { recursive: true });
  });

  beforeEach(() => {
    tunnels = [];
    refusal = null;
    vi.stubEnv("HTTP_PROXY", proxyURL);
    vi.stubEnv("http_proxy", undefined);
    vi.stubEnv("NO_PROXY", "");
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  const sendAndStream = async (secure: boolean, caCert?: string): Promise<void> => {
    const endpoint = secure ? tlsEndpoint : plainEndpoint;
    const connector = caCert == null ? undefined : trustCA(caCert);
    const config = { endpoint, codec: binary.JSON_CODEC, secure, connector };
    const res = await new HTTPClient(config).send(
      "/",
      { message: "hi" },
      messageZ,
      messageZ,
    );
    expect(res).toEqual({ message: "hi" });
    const stream = await new WebSocketClient(config).stream("/", messageZ, messageZ);
    expect(await stream.receive()).toEqual({ message: "hi" });
    stream.closeSend();
  };

  it("should send requests and streams through the proxy", async () => {
    await sendAndStream(false);
    const port = plainEndpoint.port.toString();
    expect(tunnels).toEqual([port, port]);
  });

  it("should send requests and streams through the proxy with a CA", async () => {
    await sendAndStream(true, ca);
    const port = tlsEndpoint.port.toString();
    expect(tunnels).toEqual([port, port]);
  });

  it("should throw Unreachable when the proxy cannot reach the target", async () => {
    const closed = createTCPServer();
    const port = await listen(closed);
    await new Promise((resolve) => closed.close(resolve));
    const client = new HTTPClient({
      endpoint: new url.URL({ host: "127.0.0.1", port }),
      codec: binary.JSON_CODEC,
    });
    const err = await client.send("/", { message: "hi" }, messageZ, messageZ).then(
      () => null,
      (e: unknown) => e,
    );
    expect(Unreachable.matches(err)).toBe(true);
  });

  it("should not throw Unreachable when the proxy needs authentication", async () => {
    refusal = "407 Proxy Authentication Required";
    const client = new HTTPClient({
      endpoint: plainEndpoint,
      codec: binary.JSON_CODEC,
    });
    const err = await client.send("/", { message: "hi" }, messageZ, messageZ).then(
      () => null,
      (e: unknown) => e,
    );
    expect(err).toBeInstanceOf(TypeError);
    expect(Unreachable.matches(err)).toBe(false);
  });

  it("should name a proxy variable that is not a valid URL", () => {
    vi.stubEnv("HTTP_PROXY", " ");
    const config = { endpoint: plainEndpoint, codec: binary.JSON_CODEC };
    expect(() => new HTTPClient(config)).toThrow(
      'HTTP_PROXY is not a valid proxy URL: " "',
    );
    expect(() => trustCA(ca)).toThrow("HTTP_PROXY is not a valid proxy URL");
  });

  it.each(["proxy.example.com:3128", "localhost:3128"])(
    "should name a proxy variable with no scheme: %s",
    (value) => {
      vi.stubEnv("HTTP_PROXY", value);
      const config = { endpoint: plainEndpoint, codec: binary.JSON_CODEC };
      expect(() => new HTTPClient(config)).toThrow(
        `HTTP_PROXY is not a valid proxy URL: "${value}"`,
      );
    },
  );

  it("should use a lowercase proxy variable over an uppercase one", async () => {
    vi.stubEnv("HTTP_PROXY", " ");
    vi.stubEnv("http_proxy", proxyURL);
    await sendAndStream(false);
    const port = plainEndpoint.port.toString();
    expect(tunnels).toEqual([port, port]);
  });

  it("should name an invalid lowercase proxy variable", () => {
    vi.stubEnv("http_proxy", " ");
    const config = { endpoint: plainEndpoint, codec: binary.JSON_CODEC };
    expect(() => new HTTPClient(config)).toThrow(
      'http_proxy is not a valid proxy URL: " "',
    );
  });

  it("should connect directly when the lowercase proxy variable is empty", async () => {
    vi.stubEnv("http_proxy", "");
    await sendAndStream(false);
    expect(tunnels).toEqual([]);
  });

  it.each(["Bun", "Deno"])(
    "should leave the proxy to the runtime in %s",
    async (name) => {
      vi.stubGlobal(name, {});
      await sendAndStream(false);
      expect(tunnels).toEqual([]);
    },
  );

  it("should connect directly to a host in NO_PROXY", async () => {
    vi.stubEnv("NO_PROXY", "127.0.0.1");
    await sendAndStream(true, ca);
    expect(tunnels).toEqual([]);
  });
});
