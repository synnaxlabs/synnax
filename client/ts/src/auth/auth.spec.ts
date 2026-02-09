// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Context } from "@synnaxlabs/freighter";
import { TimeSpan, URL } from "@synnaxlabs/x";
import { describe, expect, it, test, vi } from "vitest";

import { auth } from "@/auth";
import { AuthError, ExpiredTokenError, InvalidTokenError } from "@/errors";
import { TEST_CLIENT_PARAMS } from "@/testutil/client";
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
      new URL({
        host: TEST_CLIENT_PARAMS.host,
        port: Number(TEST_CLIENT_PARAMS.port),
      }),
    );
    const client = new auth.Client(transport.unary, TEST_CLIENT_PARAMS);
    const mw = client.middleware();
    const res = await mw(DUMMY_CTX, async () => [DUMMY_CTX, null]);
    expect(res).toEqual([DUMMY_CTX, null]);
  });

  test("invalid credentials", async () => {
    const transport = new Transport(
      new URL({
        host: TEST_CLIENT_PARAMS.host,
        port: Number(TEST_CLIENT_PARAMS.port),
      }),
    );
    const client = new auth.Client(transport.unary, {
      ...TEST_CLIENT_PARAMS,
      password: "wrong",
    });
    const mw = client.middleware();
    const [, err] = await mw(DUMMY_CTX, async () => [DUMMY_CTX, null]);
    expect(err).toBeInstanceOf(AuthError);
  });

  describe("token retry", () => {
    const ERROR_TYPES = [InvalidTokenError, ExpiredTokenError];
    ERROR_TYPES.forEach((ErrorType) => {
      it(`should re-authenticate and retry the request for ${ErrorType.name}`, async () => {
        const transport = new Transport(
          new URL({
            host: TEST_CLIENT_PARAMS.host,
            port: Number(TEST_CLIENT_PARAMS.port),
          }),
        );
        const client = new auth.Client(transport.unary, TEST_CLIENT_PARAMS);
        const mw = client.middleware();
        let isFirst = true;
        let tkOne: string | undefined;
        let tkTwo: string | undefined;
        const [, err] = await mw(DUMMY_CTX, async () => {
          if (isFirst) {
            isFirst = false;
            tkOne = client.token;
            return [DUMMY_CTX, new ErrorType()];
          }
          tkTwo = client.token;
          return [DUMMY_CTX, null];
        });
        expect(err).toBeNull();
        expect(tkOne).toBeDefined();
        expect(tkTwo).toBeDefined();
      });
    });

    it("should fail after MAX_RETRIES", async () => {
      const transport = new Transport(
        new URL({
          host: TEST_CLIENT_PARAMS.host,
          port: Number(TEST_CLIENT_PARAMS.port),
        }),
      );
      const client = new auth.Client(transport.unary, TEST_CLIENT_PARAMS);
      const mw = client.middleware();
      const [, err] = await mw(DUMMY_CTX, async () => [
        DUMMY_CTX,
        new InvalidTokenError(),
      ]);
      expect(err).toBeInstanceOf(InvalidTokenError);
    });
  });

  describe("clock skew detection", () => {
    test("should not warn when skew is within threshold", async () => {
      const warnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
      const transport = new Transport(
        new URL({
          host: TEST_CLIENT_PARAMS.host,
          port: Number(TEST_CLIENT_PARAMS.port),
        }),
      );
      const client = new auth.Client(
        transport.unary,
        TEST_CLIENT_PARAMS,
        TimeSpan.seconds(5),
      );
      const mw = client.middleware();
      await mw(DUMMY_CTX, async () => [DUMMY_CTX, null]);
      const clockSkewWarns = warnSpy.mock.calls.filter(
        (call) =>
          typeof call[0] === "string" && call[0].includes("clock skew"),
      );
      expect(clockSkewWarns).toHaveLength(0);
      warnSpy.mockRestore();
    });

    test("should populate clockSkew and clockSkewExcessive after auth", async () => {
      const warnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
      const transport = new Transport(
        new URL({
          host: TEST_CLIENT_PARAMS.host,
          port: Number(TEST_CLIENT_PARAMS.port),
        }),
      );
      const client = new auth.Client(
        transport.unary,
        TEST_CLIENT_PARAMS,
        TimeSpan.seconds(5),
      );
      const mw = client.middleware();
      await mw(DUMMY_CTX, async () => [DUMMY_CTX, null]);
      expect(client.clockSkew).toBeDefined();
      expect(typeof client.clockSkewExcessive).toBe("boolean");
      warnSpy.mockRestore();
    });
  });
});
