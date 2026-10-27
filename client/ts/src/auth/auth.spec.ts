// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Context } from "@synnaxlabs/freighter";
import { errors, id, url } from "@synnaxlabs/x";
import { describe, expect, it, test } from "vitest";

import { auth } from "@/auth";
import { AuthError } from "@/index";
import { createTestClient, TEST_CLIENT_PARAMS } from "@/testutil";
import { Transport } from "@/transport";

const DUMMY_CTX: Context = {
  target: "test",
  role: "client",
  protocol: "http",
  params: {},
};

const authError = (type: string): Error =>
  errors.decode({ type: `sy.auth.${type}`, data: "" }) as Error;

describe("auth", () => {
  test("valid credentials", async () => {
    const transport = new Transport(
      new url.URL({
        host: TEST_CLIENT_PARAMS.host,
        port: Number(TEST_CLIENT_PARAMS.port),
      }),
    );
    const client = new auth.Client(transport.unary, TEST_CLIENT_PARAMS);
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
      ...TEST_CLIENT_PARAMS,
      password: "wrong",
    });
    const mw = client.middleware();
    await expect(mw(DUMMY_CTX, async () => DUMMY_CTX)).rejects.toThrow(AuthError);
  });

  describe("token retry", () => {
    const ERROR_TYPES = ["invalid_token", "expired_token"];
    ERROR_TYPES.forEach((type) => {
      it(`should re-authenticate and retry the request for ${type}`, async () => {
        const transport = new Transport(
          new url.URL({
            host: TEST_CLIENT_PARAMS.host,
            port: Number(TEST_CLIENT_PARAMS.port),
          }),
        );
        const client = new auth.Client(transport.unary, TEST_CLIENT_PARAMS);
        const mw = client.middleware();
        let isFirst = true;
        let tkOne: string | undefined;
        let tkTwo: string | undefined;
        const res = await mw(DUMMY_CTX, async () => {
          if (isFirst) {
            isFirst = false;
            tkOne = client.token;
            throw authError(type);
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
      const client = new auth.Client(transport.unary, TEST_CLIENT_PARAMS);
      const mw = client.middleware();
      await expect(
        mw(DUMMY_CTX, async () => {
          throw authError("invalid_token");
        }),
      ).rejects.toMatchObject({ type: "sy.auth.invalid_token" });
    });
  });

  describe("sessions", () => {
    const makeClient = (credentials: auth.Credentials = TEST_CLIENT_PARAMS) => {
      const transport = new Transport(
        new url.URL({
          host: TEST_CLIENT_PARAMS.host,
          port: Number(TEST_CLIENT_PARAMS.port),
        }),
      );
      const client = new auth.Client(transport.unary, credentials);
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
      // the next request logs in again with a fresh session
      const after = await client.retrieveSessions();
      expect(after.map((s) => s.key)).not.toContain(key);
      expect(client.session?.key).toBeDefined();
      expect(after.map((s) => s.key)).toContain(client.session?.key);
    });

    it("should not open a new session when logging out of an ended session", async () => {
      const admin = createTestClient();
      const credentials = { username: id.create(), password: "seldon" };
      await admin.users.create(credentials);
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

    it("should keep a session another request opened during the logout", async () => {
      const client = makeClient();
      await login(client);
      const first = client.session?.key;
      const logout = client.logout();
      // A rejected token makes the middleware log in again while the delete is
      // still in flight.
      await expect(
        client.middleware()(DUMMY_CTX, async () => {
          throw authError("invalid_token");
        }),
      ).rejects.toMatchObject({ type: "sy.auth.invalid_token" });
      await logout;
      expect(client.authenticated).toBe(true);
      expect(client.session?.key).toBeDefined();
      expect(client.session?.key).not.toEqual(first);
      await client.logout();
      expect(client.authenticated).toBe(false);
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
          throw authError("invalid_token");
        });
        newer = client.token;
        throw authError("session_ended");
      });
      expect(newer).toBeDefined();
      expect(client.token).toEqual(newer);
      await client.logout();
    });
  });
});
