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

    it("should log back in under the new username after a rename", async () => {
      const client = createTestClient();
      const username = id.create();
      const created = await client.users.create({ username, password: "test" });
      const transport = new Transport(
        new url.URL({
          host: TEST_CLIENT_PARAMS.host,
          port: Number(TEST_CLIENT_PARAMS.port),
        }),
      );
      const authClient = new auth.Client(transport.unary, {
        username,
        password: "test",
      });
      const mw = authClient.middleware();
      await mw(DUMMY_CTX, async () => DUMMY_CTX);
      const renamed = id.create();
      await client.users.changeUsername(created.key, renamed);
      authClient.setUser({ ...created, username: renamed });
      let isFirst = true;
      await expect(
        mw(DUMMY_CTX, async () => {
          if (isFirst) {
            isFirst = false;
            throw new InvalidTokenError();
          }
          return DUMMY_CTX;
        }),
      ).resolves.toEqual(DUMMY_CTX);
      expect(authClient.user?.username).toEqual(renamed);
      await client.close();
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
