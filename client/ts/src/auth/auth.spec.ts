// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Context } from "@synnaxlabs/freighter";
import { URL } from "@synnaxlabs/x/url";
import { describe, expect, it, test } from "vitest";

import { auth } from "@/auth";
import { AuthError, InvalidTokenError } from "@/errors";
import { HOST, PORT } from "@/setupspecs";
import { Transport } from "@/transport";

const DUMMY_CTX: Context = {
  target: "test",
  role: "client",
  protocol: "http",
  params: {},
};

describe("auth", () => {
  test("valid credentials", async () => {
    const transport = new Transport(new URL({ host: HOST, port: PORT }));
    const client = new auth.Client(transport.unary, {
      username: "synnax",
      password: "seldon",
    });
    const mw = client.middleware();
    const res = await mw(DUMMY_CTX, async () => [DUMMY_CTX, null]);
    expect(res).toEqual([DUMMY_CTX, null]);
  });

  test("invalid credentials", async () => {
    const transport = new Transport(new URL({ host: HOST, port: PORT }));
    const client = new auth.Client(transport.unary, {
      username: "synnax",
      password: "wrong",
    });
    const mw = client.middleware();
    const [, err] = await mw(DUMMY_CTX, async () => [DUMMY_CTX, null]);
    expect(err).toBeInstanceOf(AuthError);
  });

  describe("invalid token retry", async () => {
    it("should re-authenticate and retry the request", async () => {
      const transport = new Transport(new URL({ host: HOST, port: PORT }));
      const client = new auth.Client(transport.unary, {
        username: "synnax",
        password: "seldon",
      });
      const mw = client.middleware();
      let isFirst = true;
      let tkOne: string | undefined;
      let tkTwo: string | undefined;
      const [, err] = await mw(DUMMY_CTX, async () => {
        if (isFirst) {
          isFirst = false;
          tkOne = client.token;
          return [DUMMY_CTX, new InvalidTokenError()];
        }
        tkTwo = client.token;
        return [DUMMY_CTX, null];
      });
      expect(err).toBeNull();
      expect(tkOne).toBeDefined();
      expect(tkTwo).toBeDefined();
    });
    it("should fail after MAX_RETRIES", async () => {
      const transport = new Transport(new URL({ host: HOST, port: PORT }));
      const client = new auth.Client(transport.unary, {
        username: "synnax",
        password: "seldon",
      });
      const mw = client.middleware();
      const [, err] = await mw(DUMMY_CTX, async () => [
        DUMMY_CTX,
        new InvalidTokenError(),
      ]);
      expect(err).toBeInstanceOf(InvalidTokenError);
    });
  });
});
