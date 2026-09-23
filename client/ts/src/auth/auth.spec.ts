// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Context } from "@synnaxlabs/freighter";
import { id, url } from "@synnaxlabs/x";
import { describe, expect, it, test } from "vitest";

import { auth } from "@/auth";
import { AuthError, ExpiredTokenError, InvalidTokenError } from "@/errors";
import { createTestClient, TEST_CLIENT_PARAMS } from "@/testutil";
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

  describe("changePassword", () => {
    const createSelf = async () => {
      const username = id.create();
      await createTestClient().users.create({ username, password: "old" });
      const asUser = createTestClient({ username, password: "old" });
      await asUser.connect();
      return { asUser, username };
    };

    it("should replace the signed-in user's password", async () => {
      const { asUser, username } = await createSelf();
      await asUser.auth.changePassword("old", "new");
      await expect(
        createTestClient({ username, password: "new" }).connect(),
      ).resolves.toBeDefined();
      await expect(
        createTestClient({ username, password: "old" }).connect(),
      ).rejects.toThrow(AuthError);
    });

    it("should reject a current password the caller mistyped", async () => {
      const { asUser, username } = await createSelf();
      // The session is already authenticated, so only a real check of the supplied
      // value can reject this.
      await expect(
        asUser.auth.changePassword("not-the-password", "new"),
      ).rejects.toThrow(AuthError);
      await expect(
        createTestClient({ username, password: "old" }).connect(),
      ).resolves.toBeDefined();
    });
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
});
