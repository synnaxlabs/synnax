// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { Context } from "@synnaxlabs/freighter";
import { url } from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";

import { auth } from "@/auth";
import { MissingCredentialsError, SessionEndedError } from "@/index";
import { TEST_CREDENTIALS, testClientParams } from "@/testutil";
import { Transport } from "@/transport";

const DUMMY_CTX: Context = {
  target: "test",
  role: "client",
  protocol: "http",
  params: {},
};

const createTransport = () =>
  new Transport({
    url: new url.URL({
      host: testClientParams().host,
      port: Number(testClientParams().port),
    }),
  });

const makeClient = () => {
  const transport = createTransport();
  const client = new auth.Client(transport.unary, { credentials: TEST_CREDENTIALS });
  transport.use(client.middleware());
  return client;
};

const login = async (client: auth.Client) => {
  await client.middleware()(DUMMY_CTX, async () => DUMMY_CTX);
};

describe("auth cookie sessions", () => {
  it("should end a cookie session once the browser drops its cookie", async () => {
    const owner = makeClient();
    await login(owner);
    let cookie: string | undefined =
      `synnax-session-${testClientParams().port}=${owner.token}`;
    const transport = createTransport();
    transport.use(async (ctx, next) => {
      if (cookie != null) ctx.params.Cookie = cookie;
      return await next(ctx);
    });
    const onFailure = vi.fn();
    const client = new auth.Client(transport.unary, { cookie: true }, { onFailure });
    transport.use(client.middleware());
    await client.retrieveSessions();
    cookie = undefined;
    await expect(client.retrieveSessions()).rejects.toThrow(SessionEndedError);
    expect(onFailure).toHaveBeenCalledOnce();
    expect(SessionEndedError.matches(onFailure.mock.calls[0][0])).toBe(true);
    // the next request logs in again instead of going out with no session
    await expect(client.retrieveSessions()).rejects.toThrow(MissingCredentialsError);
    expect(onFailure).toHaveBeenCalledTimes(2);
    await owner.logout();
  });

  it("should log in again instead of ending a cookie session with credentials", async () => {
    const owner = makeClient();
    await login(owner);
    let cookie: string | undefined =
      `synnax-session-${testClientParams().port}=${owner.token}`;
    let logins = 0;
    const transport = createTransport();
    transport.use(async (ctx, next) => {
      if (ctx.target.endsWith("/auth/session/create")) logins++;
      if (cookie != null) ctx.params.Cookie = cookie;
      return await next(ctx);
    });
    const onFailure = vi.fn();
    const client = new auth.Client(
      transport.unary,
      { credentials: TEST_CREDENTIALS, cookie: true },
      { onFailure },
    );
    transport.use(client.middleware());
    await client.retrieveSessions();
    expect(logins).toBe(1);
    cookie = undefined;
    // Node keeps no cookies, so each fresh session is lost the same way
    await expect(client.retrieveSessions()).rejects.toThrow(MissingCredentialsError);
    expect(logins).toBeGreaterThan(1);
    expect(onFailure).toHaveBeenCalledOnce();
    expect(MissingCredentialsError.matches(onFailure.mock.calls[0][0])).toBe(true);
    await owner.logout();
  });
});
