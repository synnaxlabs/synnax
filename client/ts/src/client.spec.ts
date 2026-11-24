// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { execFileSync } from "node:child_process";
import {
  chmodSync,
  existsSync,
  mkdirSync,
  mkdtempSync,
  readFileSync,
  rmSync,
  writeFileSync,
} from "node:fs";
import { createServer as createHTTPServer } from "node:http";
import { createServer, type Server } from "node:https";
import type { AddressInfo } from "node:net";
import { tmpdir } from "node:os";
import { join } from "node:path";

import { type Middleware, trustCA } from "@synnaxlabs/freighter";
import { TimeSpan, url } from "@synnaxlabs/x";
import {
  afterAll,
  afterEach,
  assert,
  beforeAll,
  beforeEach,
  describe,
  expect,
  it,
  vi,
} from "vitest";
import { WebSocketServer } from "ws";
import { z } from "zod";

import type { connection } from "@/connection";
import {
  APIKeyUnavailableError,
  DisconnectedError,
  InvalidCredentialsError,
  Synnax,
  ValidationError,
} from "@/index";
import { label } from "@/label";
import {
  createTestClient,
  createTestClientWithPolicy,
  TEST_CREDENTIALS,
  testCoreAddress,
} from "@/testutil";
import { Transport } from "@/transport";

const reasonOf = (status: connection.Status): connection.Reason | undefined =>
  status.variant === "error" ? status.details.reason : undefined;

const spyOnCloseOf = (client: Synnax, field: "conn" | "cache") =>
  vi.spyOn(client[field], "close");

const FAST_RETRY = {
  baseInterval: TimeSpan.milliseconds(5),
  scale: 1,
  maxRetries: 2,
};

describe("constructor", () => {
  beforeEach(() => {
    vi.stubEnv("SYNNAX_HOST", "core.env");
    vi.stubEnv("SYNNAX_PORT", "9091");
    vi.stubEnv("SYNNAX_API_KEY", "syk_env");
    vi.stubEnv("SYNNAX_SECURE", "true");
  });

  afterEach(() => {
    vi.unstubAllEnvs();
  });

  it("should fill the missing options from the variables", async () => {
    const client = new Synnax();
    expect(client.params).toMatchObject({
      host: "core.env",
      port: 9091,
      secure: true,
    });
    await client.close();
  });

  it("should use an argument over its variable", async () => {
    const client = new Synnax({ host: "rig", port: 1 });
    expect(client.params).toMatchObject({
      host: "rig",
      port: 1,
      secure: true,
    });
    await client.close();
  });

  it("should throw when no argument, variable, or Synnax Desktop gives a host", () => {
    vi.stubEnv("SYNNAX_HOST", "");
    const home = mkdtempSync(join(tmpdir(), "synnax-home-"));
    vi.stubEnv("HOME", home);
    vi.stubEnv("USERPROFILE", home);
    try {
      expect(() => new Synnax()).toThrow(ValidationError);
      expect(() => new Synnax()).toThrow(/^give a host, or set SYNNAX_HOST$/);
    } finally {
      rmSync(home, { recursive: true });
    }
  });
});

describe.skipIf(process.platform === "win32")("constructor with Synnax Desktop", () => {
  let home: string;

  beforeEach(() => {
    home = mkdtempSync(join(tmpdir(), "synnax-home-"));
    vi.stubEnv("SYNNAX_HOST", "");
    vi.stubEnv("SYNNAX_API_KEY", "");
    vi.stubEnv("HOME", home);
    mkdirSync(join(home, ".synnax"));
  });

  afterEach(() => {
    vi.unstubAllEnvs();
    rmSync(home, { recursive: true });
  });

  // Saves a desktop.json for the Core at the address, with a helper that runs script.
  const writeDesktop = (
    script: string | null,
    { host, port }: { host: string; port: number } = { host: "127.0.0.1", port: 1 },
  ): void => {
    const helper = join(home, "synnax-desktop-key");
    if (script != null) {
      writeFileSync(helper, `#!/bin/sh\n${script}\n`);
      chmodSync(helper, 0o755);
    }
    writeFileSync(
      join(home, ".synnax", "desktop.json"),
      JSON.stringify({ host, port, secure: false, api_key_helper: helper }),
    );
  };

  const createClient = (): Synnax =>
    new Synnax({ retry: FAST_RETRY, onRetry: () => {}, onInternalError: () => {} });

  it("should run the key helper without blocking other tasks", async () => {
    const started = join(home, "started");
    writeDesktop(`touch ${started}\nsleep 1\necho syk_slow`);
    const start = performance.now();
    const client = createClient();
    expect(performance.now() - start).toBeLessThan(500);
    let ticks = 0;
    const timer = setInterval(() => (ticks += 1), 10);
    const request = client.channels.retrieve("ch").catch(() => {});
    await expect.poll(() => existsSync(started)).toBe(true);
    const before = ticks;
    await new Promise((resolve) => setTimeout(resolve, 200));
    clearInterval(timer);
    expect(ticks).toBeGreaterThan(before);
    await request;
    await client.close();
  });

  it("should reject the first request with the error of the helper", async () => {
    writeDesktop("echo 'no keychain' >&2\nexit 1", testCoreAddress());
    const client = createClient();
    await expect(client.channels.retrieve("ch")).rejects.toThrow(
      new APIKeyUnavailableError("no keychain"),
    );
    await expect.poll(() => reasonOf(client.connection.status)).toEqual("auth");
    await client.close();
  });

  it("should connect once the helper gives a key after it failed", async () => {
    const owner = createTestClient();
    const { token, key } = await owner.auth.createAPIKey({ name: "desktop" });
    const ready = join(home, "ready");
    writeDesktop(`[ -f ${ready} ] || exit 1\necho ${token}`, testCoreAddress());
    const client = createClient();
    await expect(client.connect()).rejects.toThrow(APIKeyUnavailableError);
    writeFileSync(ready, "");
    await expect(client.connect()).resolves.toBeDefined();
    await client.close();
    await owner.auth.deleteAPIKeys(key.key);
  });

  it("should tell the user to open Synnax Desktop when its helper is gone", async () => {
    writeDesktop(null);
    const client = createClient();
    await expect(client.channels.retrieve("ch")).rejects.toThrow(
      "Synnax Desktop is not running. Open it and try again.",
    );
    await client.close();
  });

  it("should stop the helper when the client closes", async () => {
    const started = join(home, "started");
    writeDesktop(`touch ${started}\nsleep 10\necho syk_slow`);
    const client = createClient();
    const request = client.channels.retrieve("ch").catch(() => {});
    await expect.poll(() => existsSync(started)).toBe(true);
    const start = performance.now();
    await client.close();
    await request;
    expect(performance.now() - start).toBeLessThan(1000);
    expect(client.auth.session).toBeUndefined();
  });
});

const CODE = {
  provider: "idp",
  code: "code",
  verifier: "verifier",
  redirectUri: "http://127.0.0.1:4242/callback",
};

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

const PEM = "-----BEGIN CERTIFICATE-----\n-----END CERTIFICATE-----\n";

describe("constructor with caCert", () => {
  beforeEach(() => {
    vi.stubEnv("SYNNAX_SECURE", "");
  });

  afterEach(() => {
    vi.unstubAllEnvs();
    vi.unstubAllGlobals();
  });

  it("should reject a CA certificate with secure false", () => {
    expect(
      () => new Synnax({ host: "localhost", port: 9090, secure: false, caCert: PEM }),
    ).toThrow("caCert requires secure; leave secure out or set it to true");
  });

  it("should reject an OpenID Connect code given with an API key", () => {
    expect(
      () => new Synnax({ host: "localhost", port: 9090, apiKey: "syk", oidc: CODE }),
    ).toThrow(
      "Give one of a username and password, an API key, or an OpenID Connect code",
    );
  });

  it("should not fill SYNNAX_API_KEY in next to an OpenID Connect code", async () => {
    vi.stubEnv("SYNNAX_API_KEY", "syk_env");
    const logins: string[] = [];
    const server = createHTTPServer((req, res) => {
      let body = "";
      req.on("data", (chunk: Buffer) => (body += chunk.toString()));
      req.on("end", () => {
        if (req.url === "/api/v1/auth/session/create") logins.push(body);
        res.statusCode = 500;
        res.end();
      });
    });
    await new Promise<void>((resolve) => server.listen(0, "127.0.0.1", resolve));
    const address = server.address();
    if (address == null || typeof address === "string") throw new Error("no port");
    const client = new Synnax({
      host: "127.0.0.1",
      port: address.port,
      oidc: CODE,
      retry: FAST_RETRY,
      onRetry: () => {},
      onInternalError: () => {},
    });
    await client.channels.retrieve(["ch"]).catch(() => {});
    await client.close();
    server.closeAllConnections();
    await new Promise((resolve) => server.close(resolve));
    expect(logins.length).toBeGreaterThan(0);
    logins.forEach((login) => {
      expect(login).toContain('"oidc"');
      expect(login).not.toContain("syk_env");
    });
  });

  it("should use TLS for a CA certificate with SYNNAX_SECURE false", async () => {
    vi.stubEnv("SYNNAX_SECURE", "false");
    const client = new Synnax({ host: "localhost", port: 9090, caCert: PEM });
    expect(client.params.secure).toBe(true);
    await client.close();
  });

  it("should reject a CA certificate that is not PEM text", () => {
    expect(
      () =>
        new Synnax({ host: "localhost", port: 9090, secure: true, caCert: "ca.crt" }),
    ).toThrow("The CA certificate must be PEM text");
  });

  it("should throw for a CA certificate in a browser", () => {
    vi.stubGlobal("process", undefined);
    expect(
      () => new Synnax({ host: "localhost", port: 9090, secure: true, caCert: PEM }),
    ).toThrow("browser");
  });

  it("should tell a Deno user to leave out caCert", () => {
    vi.stubGlobal("Deno", {});
    expect(
      () => new Synnax({ host: "localhost", port: 9090, secure: true, caCert: PEM }),
    ).toThrow("Deno cannot use caCert. Leave it out and start Deno with --cert");
  });

  it("should use TLS with no CA for SYNNAX_CA_CERT in Deno", async () => {
    vi.stubEnv("SYNNAX_CA_CERT", join(tmpdir(), "synnax-missing-ca.pem"));
    vi.stubGlobal("Deno", {});
    const client = new Synnax({ host: "localhost", port: 9090 });
    expect(client.params.secure).toBe(true);
    expect(client.params.caCert).toBeUndefined();
    await client.close();
  });

  it("should reject SYNNAX_CA_CERT with SYNNAX_SECURE false in Deno", () => {
    vi.stubEnv("SYNNAX_SECURE", "false");
    vi.stubEnv("SYNNAX_CA_CERT", join(tmpdir(), "synnax-missing-ca.pem"));
    vi.stubGlobal("Deno", {});
    expect(() => new Synnax({ host: "localhost", port: 9090 })).toThrow(
      "SYNNAX_CA_CERT requires SYNNAX_SECURE",
    );
  });
});

describe("caCert", () => {
  beforeEach(() => {
    vi.stubEnv("SYNNAX_SECURE", "");
  });

  afterEach(() => {
    vi.unstubAllEnvs();
  });

  let dir: string;
  let caCert: string;
  let server: Server;
  let wss: WebSocketServer;
  let port: number;
  const requests: string[] = [];

  beforeAll(async () => {
    dir = mkdtempSync(join(tmpdir(), "client-ca-"));
    createCertificates(dir);
    caCert = readFileSync(join(dir, "ca.crt"), "utf8");
    server = createServer(
      {
        key: readFileSync(join(dir, "server.key")),
        cert: readFileSync(join(dir, "server.crt")),
      },
      (req, res) => {
        requests.push(req.url ?? "");
        res.statusCode = 500;
        res.end();
      },
    );
    wss = new WebSocketServer({ server });
    wss.on("connection", (ws) =>
      ws.send(Buffer.from(JSON.stringify({ type: "open" }))),
    );
    await new Promise<void>((resolve) => server.listen(0, "127.0.0.1", resolve));
    ({ port } = server.address() as AddressInfo);
  });

  afterAll(async () => {
    wss.clients.forEach((ws) => ws.terminate());
    server.closeAllConnections();
    await new Promise((resolve) => server.close(resolve));
    rmSync(dir, { recursive: true });
  });

  it.each([
    ["no secure", {}],
    ["secure true", { secure: true }],
  ])("should connect with TLS and the CA certificate with %s", async (_, secure) => {
    requests.length = 0;
    const client = new Synnax({
      host: "127.0.0.1",
      port,
      ...secure,
      caCert,
      ...TEST_CREDENTIALS,
      retry: FAST_RETRY,
      onRetry: () => {},
      onInternalError: () => {},
    });
    expect(client.params.secure).toBe(true);
    await expect(client.connect()).rejects.toThrow("HTTP 500");
    expect(requests).toContain("/api/v1/auth/session/create");
    await client.close();
  });

  it("should open a transport stream through the CA certificate", async () => {
    const transport = new Transport({
      url: new url.URL({ host: "127.0.0.1", port }),
      secure: true,
      connector: trustCA(caCert),
    });
    const stream = await transport.stream.stream("/stream", z.unknown(), z.unknown());
    stream.closeSend();
  });
});

describe("connect", () => {
  it("should await the client's first success", async () => {
    const client = createTestClient();
    const state = await client.connect();
    expect(state.variant).toEqual("success");
    expect(state.details.streamLive).toBe(true);
    await client.close();
  });

  it("should be idempotent on an already-connected client", async () => {
    const client = createTestClient();
    await client.connect();
    const state = await client.connect();
    expect(state.variant).toEqual("success");
    await client.close();
  });

  it("should reject with InvalidCredentialsError on bad credentials", async () => {
    const client = createTestClient({ password: "definitely-wrong" });
    await expect(client.connect()).rejects.toThrow(InvalidCredentialsError);
    await client.close();
  });

  it("should reject against an unreachable cluster after the retry budget", async () => {
    const client = createTestClient({ port: 9999, retry: FAST_RETRY });
    await expect(client.connect()).rejects.toThrow();
    await client.close();
  });

  it("should recover from an auth failure via reauthenticate", async () => {
    const client = createTestClient({ password: "definitely-wrong" });
    await expect(client.connect()).rejects.toThrow(InvalidCredentialsError);
    expect(reasonOf(client.connection.status)).toEqual("auth");
    client.reauthenticate(TEST_CREDENTIALS);
    const state = await client.connect();
    expect(state.variant).toEqual("success");
    await client.close();
  });

  it("should reject bad credentials without a change stream", async () => {
    const client = createTestClient({ password: "definitely-wrong", cache: false });
    await expect(client.connect()).rejects.toThrow(InvalidCredentialsError);
    expect(reasonOf(client.connection.status)).toEqual("auth");
    await client.close();
  });

  it("should reject on timeout", async () => {
    // the backoff outlasts the timeout, so the timeout is what rejects rather
    // than an escalation to error(unreachable)
    const client = createTestClient({
      port: 9999,
      retry: { baseInterval: TimeSpan.seconds(1), scale: 1, maxRetries: 100 },
    });
    await expect(
      client.connect({ timeout: TimeSpan.milliseconds(10) }),
    ).rejects.toThrow(/timed out after/);
    await client.close();
  });

  it("should become disabled on close", async () => {
    const client = createTestClient();
    await client.connect();
    await client.close();
    expect(client.connection.status.variant).toEqual("disabled");
  });
});

describe("middleware", () => {
  it("should run the middleware on each request", async () => {
    const middleware = vi.fn<Middleware>(async (ctx, next) => await next(ctx));
    const client = createTestClient({ middleware: [middleware] });
    await client.channels.retrieve(["no-such-channel"]);
    const targets = middleware.mock.calls.map(([ctx]) => ctx.target);
    expect(targets.some((t) => t.endsWith("/channel/retrieve"))).toBe(true);
  });
});

describe("denied change stream", () => {
  it("should connect with a warning when the user cannot stream changes", async () => {
    const root = createTestClient();
    const restricted = await createTestClientWithPolicy(root, {
      name: "test",
      objects: [label.resourceID("")],
      actions: ["retrieve"],
    });
    const status = await restricted.connect();
    expect(status.variant).toEqual("warning");
    expect(status.details.streamDenied).toBe(true);
    expect(status.details.authenticated).toBe(true);
    expect(status.description).not.toEqual("");
    // the reads the user is allowed to make still work
    const created = await root.labels.create({
      name: "denied-stream",
      color: { type: "fixed", value: "#000000" },
    });
    await expect(restricted.labels.retrieve(created.key)).resolves.toMatchObject({
      key: created.key,
    });
    await restricted.close();
    await root.close();
  });
});

describe("short circuit", () => {
  it("should reject requests instantly while unreachable and heal on recovery", async () => {
    const client = createTestClient({
      port: 9999,
      retry: FAST_RETRY,
    });
    await expect(client.connect()).rejects.toThrow();
    expect(reasonOf(client.connection.status)).toEqual("unreachable");
    const start = performance.now();
    await expect(client.channels.retrieve(["missing"])).rejects.toThrow(
      DisconnectedError,
    );
    expect(performance.now() - start).toBeLessThan(100);
    await client.close();
  });

  it("should not short-circuit while connected", async () => {
    const client = createTestClient();
    await client.connect();
    const ch = await client.channels.create({
      name: `short_circuit_test_${Math.floor(Math.random() * 1e9)}`,
      dataType: "float32",
      virtual: true,
    });
    expect(ch.key).not.toBe(0);
    await client.close();
  });
});

describe("logout", () => {
  it.each([
    ["a password", { username: "synnax", password: "secret" }],
    ["an API key", { apiKey: "secret" }],
    ["a token", { token: "secret" }],
    ["an OpenID Connect code", { oidc: { ...CODE, code: "secret" } }],
  ])("should keep no copy of %s on the client", async (_, given) => {
    const client = new Synnax({ host: "localhost", port: 9090, ...given });
    await client.auth.logout();
    const text = JSON.stringify(client.params, (_, v: unknown) =>
      typeof v === "bigint" ? v.toString() : v,
    );
    expect(text).not.toContain("secret");
    await client.close();
  });
});

describe("close", () => {
  // The test harness closes every client again at file end, so a spy left
  // rejecting would fail the whole suite from afterAll.
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("should not wait past the deadline for a logout the Core never answers", async () => {
    const client = createTestClient();
    await client.connect();
    vi.spyOn(client.auth, "logout").mockReturnValue(new Promise(() => {}));
    const start = performance.now();
    await client.close();
    expect(performance.now() - start).toBeLessThan(5000);
  }, 10_000);

  it("should close the client at the end of an await using block", async () => {
    const client = createTestClient();
    {
      await using _ = client;
    }
    expect(client.connection.status.variant).toEqual("disabled");
  });

  it("should settle the connection status on disabled", async () => {
    const client = createTestClient();
    await client.connect();
    await client.close();
    expect(client.connection.status.variant).toEqual("disabled");
  });

  it("should close the connection when the cache close fails, rejecting instead of reporting internally", async () => {
    const onInternalError = vi.fn();
    const client = createTestClient({ onInternalError });
    await client.connect();
    spyOnCloseOf(client, "cache").mockRejectedValue(new Error("cache boom"));
    await expect(client.close()).rejects.toThrow(AggregateError);
    expect(client.connection.status.variant).toEqual("disabled");
    expect(onInternalError).not.toHaveBeenCalled();
  });

  it("should carry every underlying failure in the rejection", async () => {
    const client = createTestClient();
    spyOnCloseOf(client, "conn").mockRejectedValue(new Error("conn boom"));
    spyOnCloseOf(client, "cache").mockRejectedValue(new Error("cache boom"));
    const error = await client.close().then(
      () => null,
      (e: unknown) => e,
    );
    assert(error instanceof AggregateError);
    expect(
      error.errors.map((e) => {
        assert(e instanceof Error);
        return e.message;
      }),
    ).toEqual(["conn boom", "cache boom"]);
  });
});
