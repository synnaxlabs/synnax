// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Context } from "@synnaxlabs/freighter";
import { errors, url } from "@synnaxlabs/x";
import { describe, expect, it, test } from "vitest";

import { auth } from "@/auth";
import {
  AuthError,
  ExpiredTokenError,
  InvalidTokenError,
  SessionEndedError,
} from "@/errors";
import { TEST_CLIENT_PARAMS } from "@/testutil";
import { Transport } from "@/transport";

const DUMMY_CTX: Context = {
  target: "test",
  role: "client",
  protocol: "http",
  params: {},
};

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
    const ERROR_TYPES = [InvalidTokenError, ExpiredTokenError];
    ERROR_TYPES.forEach((ErrorType) => {
      it(`should re-authenticate and retry the request for ${ErrorType.name}`, async () => {
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
      const client = new auth.Client(transport.unary, TEST_CLIENT_PARAMS);
      const mw = client.middleware();
      await expect(
        mw(DUMMY_CTX, async () => {
          throw new InvalidTokenError();
        }),
      ).rejects.toThrow(InvalidTokenError);
    });
  });

  describe("sessions", () => {
    const makeClient = (): auth.Client => {
      const transport = new Transport(
        new url.URL({
          host: TEST_CLIENT_PARAMS.host,
          port: Number(TEST_CLIENT_PARAMS.port),
        }),
      );
      const client = new auth.Client(transport.unary, TEST_CLIENT_PARAMS);
      transport.use(client.middleware());
      return client;
    };

    it("should hold the session and subject after login", async () => {
      const client = makeClient();
      await client.middleware()(DUMMY_CTX, async () => DUMMY_CTX);
      expect(client.session).toBeDefined();
      expect(client.subject?.type).toEqual("user");
      expect(client.user?.username).toEqual(TEST_CLIENT_PARAMS.username);
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

    it("should decode a session ended error", () => {
      expect(
        SessionEndedError.matches(
          errors.decode({ type: "sy.auth.session_ended", data: "ended" }),
        ),
      ).toBe(true);
    });
  });
});
