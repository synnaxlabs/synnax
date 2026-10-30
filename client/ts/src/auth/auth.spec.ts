// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Context, type UnaryClient } from "@synnaxlabs/freighter";
import { errors, id, url, uuid } from "@synnaxlabs/x";
import { describe, expect, it, test } from "vitest";

import { auth } from "@/auth";
import {
  AuthError,
  ExpiredTokenError,
  InvalidTokenError,
  SessionEndedError,
  ValidationError,
} from "@/errors";
import {
  createTestClient,
  createTestUser,
  TEST_CLIENT_PARAMS,
  TEST_CREDENTIALS,
} from "@/testutil";
import { Transport } from "@/transport";
import { user } from "@/user";

const DUMMY_CTX: Context = {
  target: "test",
  role: "client",
  protocol: "http",
  params: {},
};

// a login-only client: it opens no streams, so a user with no role can use it
const createLoginClient = (credentials: auth.Credentials): auth.Client => {
  const transport = new Transport(
    new url.URL({
      host: TEST_CLIENT_PARAMS.host,
      port: Number(TEST_CLIENT_PARAMS.port),
    }),
  );
  const client = new auth.Client(transport.unary, { credentials });
  transport.use(client.middleware());
  return client;
};

describe("auth", () => {
  test("valid credentials", async () => {
    const transport = new Transport(
      new url.URL({
        host: TEST_CLIENT_PARAMS.host,
        port: Number(TEST_CLIENT_PARAMS.port),
      }),
    );
    const client = new auth.Client(transport.unary, { credentials: TEST_CREDENTIALS });
    const mw = client.middleware();
    const res = await mw(DUMMY_CTX, async () => DUMMY_CTX);
    expect(res).toEqual(DUMMY_CTX);
  });

  test("invalid credentials", async () => {
    const transport = new Transport(
      new url.URL({
        host: TEST_CLIENT_PARAMS.host,
        port: Number(TEST_CLIENT_PARAMS.port),
      }),
    );
    const client = new auth.Client(transport.unary, {
      credentials: { username: TEST_CREDENTIALS.username, password: "wrong" },
    });
    const mw = client.middleware();
    await expect(mw(DUMMY_CTX, async () => DUMMY_CTX)).rejects.toThrow(AuthError);
  });

  describe("token retry", () => {
    const ERROR_TYPES = [InvalidTokenError, ExpiredTokenError];
    ERROR_TYPES.forEach((ErrorType) => {
      it(`should re-authenticate and retry the request for ${ErrorType.name}`, async () => {
        const transport = new Transport(
          new url.URL({
            host: TEST_CLIENT_PARAMS.host,
            port: Number(TEST_CLIENT_PARAMS.port),
          }),
        );
        const client = new auth.Client(transport.unary, {
          credentials: TEST_CREDENTIALS,
        });
        const mw = client.middleware();
        let isFirst = true;
        let tkOne: string | undefined;
        let tkTwo: string | undefined;
        const res = await mw(DUMMY_CTX, async () => {
          if (isFirst) {
            isFirst = false;
            tkOne = client.token;
            throw new ErrorType();
          }
          tkTwo = client.token;
          return DUMMY_CTX;
        });
        expect(res).toEqual(DUMMY_CTX);
        expect(tkOne).toBeDefined();
        expect(tkTwo).toBeDefined();
      });
    });

    it("should fail after MAX_RETRIES", async () => {
      const transport = new Transport(
        new url.URL({
          host: TEST_CLIENT_PARAMS.host,
          port: Number(TEST_CLIENT_PARAMS.port),
        }),
      );
      const client = new auth.Client(transport.unary, {
        credentials: TEST_CREDENTIALS,
      });
      const mw = client.middleware();
      await expect(
        mw(DUMMY_CTX, async () => {
          throw new InvalidTokenError();
        }),
      ).rejects.toThrow(InvalidTokenError);
    });
  });

  describe("retrieveMethods", () => {
    const client = auth.createTokenlessClient({
      host: TEST_CLIENT_PARAMS.host,
      port: TEST_CLIENT_PARAMS.port,
    });
    it("should list the methods the Core accepts without logging in", async () => {
      const methods = await auth.retrieveMethods(client);
      expect(methods.toSorted()).toEqual(["api_key", "password"]);
    });
    it("should serve repeated requests through one client", async () => {
      const [first, second] = await Promise.all([
        auth.retrieveMethods(client),
        auth.retrieveMethods(client),
      ]);
      expect(second).toEqual(first);
      expect(await auth.retrieveMethods(client)).toEqual(first);
    });
  });

  describe("createSession", () => {
    const tokenless = auth.createTokenlessClient({
      host: TEST_CLIENT_PARAMS.host,
      port: TEST_CLIENT_PARAMS.port,
    });
    it("should open a session without a logged-in client", async () => {
      const { token, session } = await auth.createSession(tokenless, {
        method: "password",
        ...TEST_CREDENTIALS,
      });
      expect(token.startsWith("sys_")).toBe(true);
      expect(session.subject.type).toBe("user");
    });

    it("should reject a wrong password", async () => {
      await expect(
        auth.createSession(tokenless, {
          method: "password",
          username: TEST_CREDENTIALS.username,
          password: "x",
        }),
      ).rejects.toThrow(AuthError);
    });
  });

  describe("retrieveProviders", () => {
    it("should list no providers when the Core sets none", async () => {
      const providers = await auth.retrieveProviders(
        auth.createTokenlessClient({
          host: TEST_CLIENT_PARAMS.host,
          port: TEST_CLIENT_PARAMS.port,
        }),
      );
      expect(providers).toEqual([]);
    });
  });

  describe("sessions", () => {
    const makeClient = (
      credentials: auth.Credentials = TEST_CREDENTIALS,
      observer: auth.Observer = {},
    ) => {
      const transport = new Transport(
        new url.URL({
          host: TEST_CLIENT_PARAMS.host,
          port: Number(TEST_CLIENT_PARAMS.port),
        }),
      );
      const client = new auth.Client(transport.unary, { credentials }, observer);
      transport.use(client.middleware());
      return client;
    };
    const makeResumed = (token: string, observer: auth.Observer = {}) => {
      const transport = new Transport(
        new url.URL({
          host: TEST_CLIENT_PARAMS.host,
          port: Number(TEST_CLIENT_PARAMS.port),
        }),
      );
      const client = new auth.Client(transport.unary, { token }, observer);
      transport.use(client.middleware());
      return client;
    };
    const login = async (client: auth.Client) => {
      await client.middleware()(DUMMY_CTX, async () => DUMMY_CTX);
    };

    it("should hold the session and subject after login", async () => {
      const client = makeClient();
      await client.middleware()(DUMMY_CTX, async () => DUMMY_CTX);
      expect(client.session).toBeDefined();
      expect(client.subject?.type).toEqual("user");
    });

    it("should renew, list, and end its session", async () => {
      const client = makeClient();
      await client.middleware()(DUMMY_CTX, async () => DUMMY_CTX);
      const key = client.session?.key;
      expect(key).toBeDefined();
      await client.renew();
      const before = await client.retrieveSessions();
      expect(before.map((s) => s.key)).toContain(key);
      await client.logout();
      expect(client.authenticated).toBe(false);
      const observer = makeClient();
      const after = await observer.retrieveSessions();
      expect(after.map((s) => s.key)).not.toContain(key);
    });

    it("should stay logged out until new credentials are set", async () => {
      const client = makeClient();
      await login(client);
      await client.logout();
      await expect(client.retrieveSessions()).rejects.toThrow(
        "no credentials, API key, token, or cookie",
      );
      expect(client.authenticated).toBe(false);
      client.setCredentials(TEST_CREDENTIALS);
      await login(client);
      expect(client.authenticated).toBe(true);
      await client.logout();
    });

    it("should forget the credentials when logging out before a login", async () => {
      const client = makeClient();
      await client.logout();
      await expect(client.retrieveSessions()).rejects.toThrow(
        "no credentials, API key, token, or cookie",
      );
    });

    it("should forget a saved token when logging out before a resume", async () => {
      const owner = makeClient();
      await login(owner);
      const token = owner.token;
      if (token == null) throw new Error("expected a token");
      const resumed = makeResumed(token);
      await resumed.logout();
      await expect(resumed.retrieveSessions()).rejects.toThrow(
        "no credentials, API key, token, or cookie",
      );
      await owner.logout();
    });

    it("should not open a new session when logging out of an ended session", async () => {
      const admin = createTestClient();
      const { username, password } = await createTestUser(admin, {
        password: "seldon",
      });
      const credentials = { username, password };
      const ending = makeClient(credentials);
      const other = makeClient(credentials);
      await login(ending);
      await login(other);
      const key = ending.session?.key;
      if (key == null) throw new Error("expected a session");
      await other.deleteSessions(key);
      await ending.logout();
      expect(ending.authenticated).toBe(false);
      const sessions = await other.retrieveSessions();
      expect(sessions.map((s) => s.key)).toEqual([other.session?.key]);
      await admin.close();
    });

    it("should end a session whose resume is still in flight", async () => {
      const owner = makeClient();
      await login(owner);
      const token = owner.token;
      if (token == null) throw new Error("expected a token");
      const resumed = makeResumed(token);
      const resuming = login(resumed);
      await resumed.logout();
      await resuming;
      expect(resumed.authenticated).toBe(false);
      const observer = makeClient();
      const keys = (await observer.retrieveSessions()).map((s) => s.key);
      expect(keys).not.toContain(owner.session?.key);
      await observer.logout();
    });

    it("should end each session when two logouts overlap", async () => {
      const client = makeClient();
      await login(client);
      const first = client.session?.key;
      const firstLogout = client.logout();
      client.setCredentials(TEST_CREDENTIALS);
      await login(client);
      const second = client.session?.key;
      await Promise.all([firstLogout, client.logout()]);
      expect(second).not.toEqual(first);
      const observer = makeClient();
      const keys = (await observer.retrieveSessions()).map((s) => s.key);
      expect(keys).not.toContain(first);
      expect(keys).not.toContain(second);
      await observer.logout();
    });

    it("should keep a session another request opened during the logout", async () => {
      const client = makeClient();
      await login(client);
      const first = client.session?.key;
      const logout = client.logout();
      // A rejected token makes the middleware log in again while the delete is
      // still in flight.
      await expect(
        client.middleware()(DUMMY_CTX, async () => {
          throw new InvalidTokenError();
        }),
      ).rejects.toThrow(InvalidTokenError);
      await logout;
      expect(client.authenticated).toBe(true);
      expect(client.session?.key).toBeDefined();
      expect(client.session?.key).not.toEqual(first);
      await client.logout();
      expect(client.authenticated).toBe(false);
    });

    it("should resume a session from its token", async () => {
      const first = makeClient();
      await first.middleware()(DUMMY_CTX, async () => DUMMY_CTX);
      const token = first.token;
      expect(token).toBeDefined();
      const transport = new Transport(
        new url.URL({
          host: TEST_CLIENT_PARAMS.host,
          port: Number(TEST_CLIENT_PARAMS.port),
        }),
      );
      let succeeded = false;
      let username: string | undefined;
      const resumed = new auth.Client(
        transport.unary,
        { token },
        {
          onUser: (u) => (username = u?.name),
          onSuccess: () => (succeeded = true),
        },
      );
      transport.use(resumed.middleware());
      await resumed.middleware()(DUMMY_CTX, async () => DUMMY_CTX);
      expect(succeeded).toBe(true);
      expect(resumed.session?.key).toEqual(first.session?.key);
      expect(username).toEqual(TEST_CREDENTIALS.username);
      await expect(resumed.renew().then((s) => s.key)).resolves.toEqual(
        first.session?.key,
      );
    });

    it("should report a dead token as a credential failure", async () => {
      const transport = new Transport(
        new url.URL({
          host: TEST_CLIENT_PARAMS.host,
          port: Number(TEST_CLIENT_PARAMS.port),
        }),
      );
      let failure: Error | undefined;
      const client = new auth.Client(
        transport.unary,
        { token: "sys_dead" },
        { onFailure: (e) => (failure = e) },
      );
      transport.use(client.middleware());
      await expect(
        client.middleware()(DUMMY_CTX, async () => DUMMY_CTX),
      ).rejects.toSatisfy((e) => AuthError.matches(e));
      expect(failure).toBeDefined();
      expect(client.authenticated).toBe(false);
    });

    it("should keep a saved token when the Core cannot be reached", async () => {
      const transport = new Transport(new url.URL({ host: "localhost", port: 1 }));
      let failure: Error | undefined;
      const client = new auth.Client(
        transport.unary,
        { token: "sys_saved" },
        { onFailure: (e) => (failure = e) },
      );
      transport.use(client.middleware());
      const attempt = async () =>
        await client.middleware()(DUMMY_CTX, async () => DUMMY_CTX);
      await expect(attempt()).rejects.toSatisfy((e) => !AuthError.matches(e));
      // The token is still there to try: a second attempt reaches for the Core again
      // instead of failing for want of credentials.
      await expect(attempt()).rejects.toSatisfy((e) => !AuthError.matches(e));
      expect(failure).toBeUndefined();
    });

    it("should need credentials once the Core has rejected the token", async () => {
      const client = makeResumed("sys_dead");
      const attempt = async () =>
        await client.middleware()(DUMMY_CTX, async () => DUMMY_CTX);
      await expect(attempt()).rejects.toSatisfy((e) => AuthError.matches(e));
      await expect(attempt()).rejects.toThrow(
        "no credentials, API key, token, or cookie",
      );
    });

    it("should let the newest login win when two interleave", async () => {
      const admin = createTestClient();
      const { username: secondUsername, password } = await createTestUser(admin);
      const second = { username: secondUsername, password };
      let username: string | undefined;
      const client = makeClient(TEST_CREDENTIALS, {
        onUser: (u) => (username = u?.name),
      });
      const first = client.middleware()(DUMMY_CTX, async () => DUMMY_CTX);
      client.setCredentials(second);
      await login(client);
      await first;
      expect(username).toEqual(second.username);
      await admin.close();
    });

    it("should keep a newer login when an older renew answers late", async () => {
      const admin = createTestClient();
      const { username: secondUsername, password } = await createTestUser(admin);
      const second = { username: secondUsername, password };
      let username: string | undefined;
      const client = makeClient(TEST_CREDENTIALS, {
        onUser: (u) => (username = u?.name),
      });
      await login(client);
      const first = client.session?.key;
      const renewing = client.renew();
      client.setCredentials(second);
      await login(client);
      const newer = client.session?.key;
      await renewing;
      expect(newer).not.toEqual(first);
      expect(client.session?.key).toEqual(newer);
      expect(username).toEqual(second.username);
      await admin.close();
    });

    it("should keep a newer login when an older request is rejected late", async () => {
      const admin = createTestClient();
      const { username: secondUsername, password } = await createTestUser(admin);
      const second = { username: secondUsername, password };
      let username: string | undefined;
      const client = makeClient(TEST_CREDENTIALS, {
        onUser: (u) => (username = u?.name),
      });
      await login(client);
      let reject: (() => void) | undefined;
      let calls = 0;
      const late = client.middleware()(DUMMY_CTX, async (ctx) => {
        calls += 1;
        if (calls > 1) return ctx;
        await new Promise<void>((resolve) => (reject = resolve));
        throw new InvalidTokenError();
      });
      await expect.poll(() => reject).toBeDefined();
      client.setCredentials(second);
      await login(client);
      const newer = client.session?.key;
      reject?.();
      await late;
      expect(client.session?.key).toEqual(newer);
      expect(username).toEqual(second.username);
      await admin.close();
    });

    it("should end its own session when released", async () => {
      const client = makeClient();
      await login(client);
      const key = client.session?.key;
      await client.release();
      expect(client.authenticated).toBe(false);
      const observer = makeClient();
      const sessions = await observer.retrieveSessions();
      expect(sessions.map((s) => s.key)).not.toContain(key);
    });

    it("should end a session whose login is still in flight when released", async () => {
      const transport = new Transport(
        new url.URL({
          host: TEST_CLIENT_PARAMS.host,
          port: Number(TEST_CLIENT_PARAMS.port),
        }),
      );
      let key: string | undefined;
      const client = new auth.Client(
        transport.unary,
        { credentials: TEST_CREDENTIALS },
        { onSuccess: () => (key = client.session?.key) },
      );
      transport.use(client.middleware());
      const logging = login(client);
      await client.release();
      await logging;
      expect(client.authenticated).toBe(false);
      expect(key).toBeDefined();
      const observer = makeClient();
      const sessions = await observer.retrieveSessions();
      expect(sessions.map((s) => s.key)).not.toContain(key);
    });

    it("should leave a persistent session open when released", async () => {
      const transport = new Transport(
        new url.URL({
          host: TEST_CLIENT_PARAMS.host,
          port: Number(TEST_CLIENT_PARAMS.port),
        }),
      );
      const client = new auth.Client(transport.unary, {
        credentials: TEST_CREDENTIALS,
        persistent: true,
      });
      transport.use(client.middleware());
      await login(client);
      const token = client.token;
      if (token == null) throw new Error("expected a token");
      await client.release();
      const resumed = makeResumed(token);
      await login(resumed);
      expect(resumed.authenticated).toBe(true);
      await resumed.logout();
    });

    it("should leave a resumed session open when released", async () => {
      const owner = makeClient();
      await login(owner);
      const token = owner.token;
      if (token == null) throw new Error("expected a token");
      const resumed = makeResumed(token);
      await login(resumed);
      await resumed.release();
      expect(resumed.authenticated).toBe(false);
      await expect(owner.renew().then((s) => s.key)).resolves.toEqual(
        owner.session?.key,
      );
    });

    it("should keep a newer session when a request on an older one fails", async () => {
      const client = makeClient();
      await login(client);
      let newer: string | undefined;
      let failed = false;
      await client.middleware()(DUMMY_CTX, async (ctx) => {
        if (failed) return ctx;
        failed = true;
        let rejected = false;
        await client.middleware()(DUMMY_CTX, async (inner) => {
          if (rejected) return inner;
          rejected = true;
          throw new InvalidTokenError();
        });
        newer = client.token;
        throw new SessionEndedError();
      });
      expect(newer).toBeDefined();
      expect(client.token).toEqual(newer);
      await client.logout();
    });

    it("should decode a session ended error", () => {
      expect(
        SessionEndedError.matches(
          errors.decode({ type: "sy.auth.session_ended", data: "ended" }),
        ),
      ).toBe(true);
    });
  });

  describe("OpenID Connect links", () => {
    const makeClient = () => {
      const transport = new Transport(
        new url.URL({
          host: TEST_CLIENT_PARAMS.host,
          port: Number(TEST_CLIENT_PARAMS.port),
        }),
      );
      const client = new auth.Client(transport.unary, {
        credentials: TEST_CREDENTIALS,
      });
      transport.use(client.middleware());
      return client;
    };

    it("should list no links for a user without any", async () => {
      expect(await makeClient().retrieveOIDCLinks()).toEqual([]);
    });

    it("should ignore an unknown link when unlinking", async () => {
      await expect(makeClient().unlinkOIDC(uuid.create())).resolves.toBeUndefined();
    });

    it("should refuse a link while the Core accepts no provider", async () => {
      await expect(
        makeClient().linkOIDC({
          provider: "idp",
          code: "code",
          verifier: "verifier",
          redirectUri: "http://127.0.0.1:4242/callback",
        }),
      ).rejects.toThrow(ValidationError);
    });
  });

  describe("OpenID Connect login", () => {
    const CODE: auth.OIDCCredentials = {
      provider: "idp",
      code: "code",
      verifier: "verifier",
      redirectUri: "http://127.0.0.1:4242/callback",
    };
    const SESSION = {
      key: uuid.create(),
      subject: { type: "user", key: uuid.create() },
      credential: uuid.create(),
      lastActive: 0,
    };

    // records each request and answers a login or a renewal like a Core would, after
    // failing the first `unreachable` requests as a refused connection does
    const createRecordingUnary = (unreachable = 0) => {
      const sent: { target: string; req: unknown; params?: Record<string, string> }[] =
        [];
      let failures = unreachable;
      const unary: UnaryClient = {
        use: () => {},
        send: async (target, req, _, resSchema, params) => {
          sent.push({ target, req, params });
          if (failures > 0) {
            failures -= 1;
            throw new Error("connection refused");
          }
          return resSchema.parse(
            target.endsWith("/auth/session/create")
              ? { token: "sys_token", session: SESSION }
              : { session: SESSION },
          );
        },
      };
      return { unary, sent };
    };

    it("should log in once with the code, then resume its session", async () => {
      const { unary, sent } = createRecordingUnary();
      const client = new auth.Client(unary, { oidc: CODE });
      const mw = client.middleware();
      let failed = false;
      await mw(DUMMY_CTX, async (ctx) => {
        if (failed) return ctx;
        failed = true;
        throw new InvalidTokenError("expired");
      });
      expect(sent.map((r) => r.target)).toEqual([
        "/auth/session/create",
        "/auth/session/renew",
      ]);
      expect(sent[0].req).toEqual({ proof: { method: "oidc", ...CODE } });
      expect(sent[1].params).toEqual({ "synnax-own-token": "sys_token" });
    });

    it("should keep the code through a login that never reached the Core", async () => {
      const { unary, sent } = createRecordingUnary(1);
      const client = new auth.Client(unary, { oidc: CODE });
      const mw = client.middleware();
      await expect(mw(DUMMY_CTX, async (ctx) => ctx)).rejects.toThrow(
        "connection refused",
      );
      await mw(DUMMY_CTX, async (ctx) => ctx);
      expect(sent.map((r) => r.req)).toEqual([
        { proof: { method: "oidc", ...CODE } },
        { proof: { method: "oidc", ...CODE } },
      ]);
    });

    it("should not resume a code login that a newer login replaced", async () => {
      const sent: { target: string; req: unknown; params?: Record<string, string> }[] =
        [];
      let answerCode: (() => void) | undefined;
      const unary: UnaryClient = {
        use: () => {},
        send: async (target, req, _, resSchema, params) => {
          sent.push({ target, req, params });
          if (!target.endsWith("/auth/session/create"))
            return resSchema.parse(
              target.endsWith("/auth/session/renew") ? { session: SESSION } : {},
            );
          const { proof } = req as { proof: { method: string } };
          if (proof.method === "oidc")
            await new Promise<void>((resolve) => (answerCode = resolve));
          return resSchema.parse({ token: `sys_${proof.method}`, session: SESSION });
        },
      };
      const client = new auth.Client(unary, { oidc: CODE });
      const stale = client.middleware()(DUMMY_CTX, async (ctx) => ctx);
      await expect.poll(() => answerCode).toBeDefined();
      client.setCredentials(TEST_CREDENTIALS);
      await client.middleware()(DUMMY_CTX, async (ctx) => ctx);
      await client.logout();
      answerCode?.();
      await stale;
      await expect(client.middleware()(DUMMY_CTX, async (ctx) => ctx)).rejects.toThrow(
        "no credentials, API key, token, or cookie",
      );
      expect(sent.filter((r) => r.target.endsWith("/auth/session/renew"))).toEqual([]);
    });

    it("should forget the code when logging out before a login", async () => {
      const { unary, sent } = createRecordingUnary();
      const client = new auth.Client(unary, { oidc: CODE });
      await client.logout();
      await expect(client.middleware()(DUMMY_CTX, async (ctx) => ctx)).rejects.toThrow(
        "no credentials, API key, token, or cookie",
      );
      expect(sent).toEqual([]);
    });

    it("should end the session it opened after resuming it", async () => {
      const { unary, sent } = createRecordingUnary();
      const client = new auth.Client(unary, { oidc: CODE });
      const mw = client.middleware();
      let failed = false;
      await mw(DUMMY_CTX, async (ctx) => {
        if (failed) return ctx;
        failed = true;
        throw new InvalidTokenError("expired");
      });
      await client.release();
      expect(sent.map((r) => r.target)).toEqual([
        "/auth/session/create",
        "/auth/session/renew",
        "/auth/session/delete",
      ]);
    });

    it("should reject an OpenID Connect code given with an API key", () => {
      expect(
        () =>
          new auth.Client(createRecordingUnary().unary, {
            oidc: CODE,
            apiKey: "syk_key",
          }),
      ).toThrow(/one of a username and password, an API key, or an OpenID Connect/);
    });
  });

  describe("setPassword", () => {
    const makeClient = (authority: auth.Authority, observer?: auth.Observer) => {
      const transport = new Transport(
        new url.URL({
          host: TEST_CLIENT_PARAMS.host,
          port: Number(TEST_CLIENT_PARAMS.port),
        }),
      );
      const client = new auth.Client(transport.unary, authority, observer);
      transport.use(client.middleware());
      return client;
    };
    const createUser = async () => {
      const username = `set-password-${id.create()}`;
      const u = await createTestUser(createTestClient(), { username, password: "old" });
      return { username, subject: user.ontologyID(u.user.key) };
    };

    it("should let an owner set another user's password", async () => {
      const { username, subject } = await createUser();
      const owner = createLoginClient(TEST_CREDENTIALS);
      await owner.setPassword({ subject, password: "new" });
      await createLoginClient({ username, password: "new" }).retrieveSessions();
      await expect(
        createLoginClient({ username, password: "old" }).retrieveSessions(),
      ).rejects.toThrow(AuthError);
    });

    it("should switch to a new session when setting its own password", async () => {
      const { username } = await createUser();
      let successes = 0;
      const self = makeClient(
        { credentials: { username, password: "old" } },
        { onSuccess: () => successes++ },
      );
      await expect(self.setPassword({ password: "new" })).rejects.toThrow();
      await self.retrieveSessions();
      const old = self.session?.key;
      const before = successes;
      await self.setPassword({ password: "new", currentPassword: "old" });
      expect(successes).toEqual(before + 1);
      expect(self.session?.key).toBeDefined();
      expect(self.session?.key).not.toEqual(old);
      const sessions = await self.retrieveSessions();
      expect(sessions.map((s) => s.key)).not.toContain(old);
      await createLoginClient({ username, password: "new" }).retrieveSessions();
    });

    it("should keep working after a resumed client sets its own password", async () => {
      const { username } = await createUser();
      const opener = createLoginClient({ username, password: "old" });
      await opener.retrieveSessions();
      const token = opener.token;
      if (token == null) throw new Error("no token");
      const resumed = makeClient({ token });
      await resumed.setPassword({ password: "new", currentPassword: "old" });
      expect(resumed.token).not.toEqual(token);
      await resumed.retrieveSessions();
      await expect(opener.retrieveSessions()).rejects.toThrow(AuthError);
    });

    it("should keep the owner's session when setting another's password", async () => {
      const { subject } = await createUser();
      const owner = createLoginClient(TEST_CREDENTIALS);
      await owner.retrieveSessions();
      const before = owner.session?.key;
      await owner.setPassword({ subject, password: "new" });
      expect(owner.session?.key).toEqual(before);
    });
  });

  describe("createPassword", () => {
    it("should refuse a user that already has a password", async () => {
      const client = createTestClient();
      await expect(
        client.auth.createPassword({ username: id.create(), password: "p" }),
      ).rejects.toThrow(ValidationError);
    });

    it("should let an owner give a new user its first password", async () => {
      const owner = createTestClient();
      const u = await owner.users.create({ name: "first password" });
      const username = id.create();
      await owner.auth.createPassword({
        subject: user.ontologyID(u.key),
        username,
        password: "first",
      });
      const client = createLoginClient({ username, password: "first" });
      await client.retrieveSessions();
      expect(client.subject?.key).toEqual(u.key);
    });

    it("should refuse a username another user holds", async () => {
      const owner = createTestClient();
      const { username } = await createTestUser(owner);
      const u = await owner.users.create({ name: "repeated username" });
      await expect(
        owner.auth.createPassword({
          subject: user.ontologyID(u.key),
          username,
          password: "test",
        }),
      ).rejects.toThrow(AuthError);
    });
  });

  describe("API keys", () => {
    const makeClient = (authority: auth.Authority) => {
      const transport = new Transport(
        new url.URL({
          host: TEST_CLIENT_PARAMS.host,
          port: Number(TEST_CLIENT_PARAMS.port),
        }),
      );
      const client = new auth.Client(transport.unary, authority);
      transport.use(client.middleware());
      return client;
    };

    it("should forget an API key when logging out before a login", async () => {
      const keyed = makeClient({ apiKey: `syk_${id.create()}secret` });
      await keyed.logout();
      await expect(keyed.retrieveSessions()).rejects.toThrow(
        "no credentials, API key, token, or cookie",
      );
    });

    it("should create a key, log in with it, list it, and delete it", async () => {
      const owner = makeClient({ credentials: TEST_CREDENTIALS });
      const { token, key } = await owner.createKey({ name: "spec" });
      expect(token.startsWith("syk_")).toBe(true);
      expect(key.name).toEqual("spec");
      expect(key.subject).toEqual(owner.subject);
      const keyed = makeClient({ apiKey: token });
      await keyed.retrieveSessions();
      expect(keyed.session?.credential).toEqual(key.key);
      expect(keyed.subject).toEqual(owner.subject);
      expect((await owner.retrieveKeys()).map((k) => k.key)).toContain(key.key);
      await owner.deleteKeys(key.key);
      expect((await owner.retrieveKeys()).map((k) => k.key)).not.toContain(key.key);
    });

    it("should fail the next request of a client whose key was deleted", async () => {
      const owner = makeClient({ credentials: TEST_CREDENTIALS });
      const { token, key } = await owner.createKey({ name: "deleted" });
      const keyed = makeClient({ apiKey: token });
      await keyed.retrieveSessions();
      await owner.deleteKeys(key.key);
      await expect(keyed.retrieveSessions()).rejects.toThrow(AuthError);
    });

    it("should forget its key on logout", async () => {
      const owner = makeClient({ credentials: TEST_CREDENTIALS });
      const { token, key } = await owner.createKey({ name: "logout" });
      const keyed = makeClient({ apiKey: token });
      await keyed.retrieveSessions();
      await keyed.logout();
      await expect(keyed.retrieveSessions()).rejects.toThrow(
        "no credentials, API key, token, or cookie",
      );
      await owner.deleteKeys(key.key);
    });

    it("should reject a key the Core never issued", async () => {
      const keyed = makeClient({ apiKey: `syk_${id.create()}secret` });
      await expect(keyed.retrieveSessions()).rejects.toThrow(AuthError);
    });

    it("should connect a Synnax client with a key", async () => {
      const owner = createTestClient();
      const { token, key } = await owner.auth.createKey({ name: "client" });
      const keyed = createTestClient({
        username: undefined,
        password: undefined,
        apiKey: token,
      });
      await keyed.auth.retrieveSessions();
      expect(keyed.auth.subject).toEqual(key.subject);
      await owner.auth.deleteKeys(key.key);
    });

    it("should reject an API key given with a username and password", () => {
      expect(() => createTestClient({ apiKey: "syk_key" })).toThrow(
        /one of a username and password, an API key, or an OpenID Connect code/,
      );
    });

    it("should reject an auth client given both credentials and an API key", () => {
      expect(() =>
        makeClient({ credentials: TEST_CREDENTIALS, apiKey: "syk_key" }),
      ).toThrow(
        /one of a username and password, an API key, or an OpenID Connect code/,
      );
    });
  });
});
