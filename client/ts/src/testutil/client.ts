// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { Middleware } from "@synnaxlabs/freighter";
import { TimeSpan } from "@synnaxlabs/x";
import { afterAll, onTestFinished, TestRunner, vi } from "vitest";

import Synnax, { type SynnaxParams } from "@/client";
import { testCoreAddress } from "@/testutil/address";
import { createSeverableProxy, type SeverableProxy } from "@/testutil/proxy";

/** The root credentials of the test Core. */
export const TEST_CREDENTIALS = { username: "synnax", password: "seldon" };

/**
 * @returns the params of a client that signs in to the test Core as root.
 * @throws where {@link testCoreAddress} does.
 */
export const testClientParams = (): SynnaxParams => ({
  ...testCoreAddress(),
  ...TEST_CREDENTIALS,
  // The test Core runs with --insecure.
  secure: false,
  retry: {
    maxRetries: 4,
    baseInterval: TimeSpan.seconds(1),
    scale: 1.5,
  },
  // An exhausted budget still throws, so a retry that succeeds is not a failure.
  onRetry: () => {},
});

// Vitest drops an afterAll registered mid-test, so this module-level hook registers
// during spec collection and closes the clients that outlive a test.
const openClients: Synnax[] = [];
// Awaited so every close finishes before vitest tears down the worker; a close
// still in flight at teardown races its logging against the closing RPC channel.
afterAll(async () => {
  const results = await Promise.allSettled(openClients.map((client) => client.close()));
  const failures = results.filter((result) => result.status === "rejected");
  if (failures.length > 0)
    throw new AggregateError(
      failures.map((failure) => failure.reason),
      "failed to close test clients",
    );
});

/**
 * Creates a client connected to the local test cluster. A client made inside a test or
 * its hooks closes when that test finishes, and any other closes once every test in the
 * spec file has finished, so callers do not need to close it themselves.
 */
export const createTestClient = (params?: Partial<SynnaxParams>): Synnax => {
  // An open client keeps its change stream, so later tests would pay for its frames.
  if (TestRunner.getCurrentTest() == null) return createFileTestClient(params);
  const client = new Synnax({ ...testClientParams(), ...params });
  onTestFinished(async () => await client.close());
  return client;
};

/**
 * Creates a client connected to the local test cluster that stays open until every test
 * in the current spec file has finished, for a client that a spec reuses across tests.
 */
export const createFileTestClient = (params?: Partial<SynnaxParams>): Synnax => {
  const client = new Synnax({ ...testClientParams(), ...params });
  openClients.push(client);
  return client;
};

export interface ProxiedTestClient {
  proxy: SeverableProxy;
  client: Synnax;
}

/**
 * Creates a client that reaches the local test cluster through a severable proxy. Call
 * it inside a test: the proxy and the client close when that test finishes.
 */
export const createProxiedTestClient = async (
  params?: Partial<SynnaxParams>,
): Promise<ProxiedTestClient> => {
  const proxy = await createSeverableProxy();
  const client = new Synnax({
    ...testClientParams(),
    ...params,
    // The proxy listens on this machine, even when the Core runs on another.
    host: "localhost",
    port: proxy.port,
  });
  onTestFinished(async () => {
    // A blackholed stream never acknowledges a close, so the proxy goes first.
    await proxy.close();
    await client.close();
  });
  return { proxy, client };
};

/** Spies on the client's unary send so specs can count requests on the wire. */
export const spyOnSend = (client: Synnax, field: "transport" = "transport") =>
  vi.spyOn(client[field].unary, "send");

/**
 * @returns the request bodies that send carried to target. The client sends requests
 * of its own in the background, so specs check one target, not every call.
 */
export const requestsTo = (
  send: ReturnType<typeof spyOnSend>,
  target: string,
): unknown[] => send.mock.calls.filter(([t]) => t === target).map(([, req]) => req);

/** Records each response to target from now on, as the promise the caller awaits. */
export const recordResponses = (
  client: Synnax,
  target: string,
  field: "transport" = "transport",
): Promise<unknown>[] => {
  const { unary } = client[field];
  const send = unary.send.bind(unary);
  const responses: Promise<unknown>[] = [];
  vi.spyOn(unary, "send").mockImplementation(async (...args) => {
    const res = send(...args);
    if (args[0] === target) responses.push(res);
    return await res;
  });
  return responses;
};

/**
 * Holds each request to target until release is called, so the Core does not get it.
 * Call it inside a test: the hold ends when that test finishes.
 */
export const holdRequests = (
  client: Synnax,
  target: string,
  field: "transport" = "transport",
) => {
  const { unary } = client[field];
  const send = unary.send.bind(unary);
  const released = Promise.withResolvers<void>();
  const spy = vi.spyOn(unary, "send").mockImplementation(async (...args) => {
    if (args[0] === target) await released.promise;
    return await send(...args);
  });
  onTestFinished(() => {
    released.resolve();
    spy.mockRestore();
  });
  return { release: () => released.resolve() };
};

export interface RequestHold {
  /** Holds each matching request while engaged. Give it to the client as middleware. */
  middleware: Middleware;
  /** Starts holding the matching requests. */
  engage: () => void;
  /** Sends the held requests on to the Core, and stops holding later ones. */
  release: () => void;
}

/**
 * Creates a hold on each request whose target the predicate matches. A client built
 * with the hold's middleware sends those requests on as usual until engage is called,
 * then holds them until release is called. Engage after the client's setup reads.
 */
export const createRequestHold = (
  matches: (target: string) => boolean,
): RequestHold => {
  const released = Promise.withResolvers<void>();
  let engaged = false;
  return {
    middleware: async (ctx, next) => {
      if (engaged && matches(ctx.target)) await released.promise;
      return await next(ctx);
    },
    engage: () => {
      engaged = true;
    },
    release: () => released.resolve(),
  };
};

/** Rejects with error each request for which matches is true. Sends the others on. */
export const rejectRequests = (
  client: Synnax,
  matches: (target: string, req: unknown) => boolean,
  error: Error,
  field: "transport" = "transport",
) => {
  const { unary } = client[field];
  const send = unary.send.bind(unary);
  return vi.spyOn(unary, "send").mockImplementation(async (...args) => {
    if (matches(args[0], args[1])) throw error;
    return await send(...args);
  });
};

/**
 * Fails each request to target with the error, so the Core does not get it. Call it
 * inside a test: the failures end when that test finishes.
 */
export const failRequests = (
  client: Synnax,
  target: string,
  error: Error,
  field: "transport" = "transport",
) => {
  const { unary } = client[field];
  const send = unary.send.bind(unary);
  const spy = vi.spyOn(unary, "send").mockImplementation(async (...args) => {
    if (args[0] === target) throw error;
    return await send(...args);
  });
  onTestFinished(() => spy.mockRestore());
};

/**
 * Holds each response to target, or to each request that target matches, until release
 * is called or the test finishes. A held request that fails holds its error too.
 * responded resolves when the first held response arrives, and sent returns the count
 * of held requests. via picks the transport client: auth requests use unaryNoRetry.
 */
export const holdResponses = (
  client: Synnax,
  target: string | ((target: string, req: unknown) => boolean),
  field: "transport" = "transport",
  via: "unary" | "unaryNoRetry" = "unary",
) => {
  const matches = typeof target === "string" ? (t: string) => t === target : target;
  const unary = client[field][via];
  const send = unary.send.bind(unary);
  const { promise: responded, resolve: respond } = Promise.withResolvers<void>();
  const { promise: released, resolve: release } = Promise.withResolvers<void>();
  let sent = 0;
  const spy = vi.spyOn(unary, "send").mockImplementation(async (...args) => {
    const held = matches(args[0], args[1]);
    if (held) sent++;
    const res = send(...args);
    if (held) {
      await res.catch(() => undefined);
      respond();
      await released;
    }
    return await res;
  });
  onTestFinished(() => {
    release();
    spy.mockRestore();
  });
  return { responded, release, sent: () => sent };
};
