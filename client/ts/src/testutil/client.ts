// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan } from "@synnaxlabs/x";
import { afterAll, onTestFinished, vi } from "vitest";

import Synnax, { type SynnaxParams } from "@/client";
import { TEST_CORE_ADDRESS } from "@/testutil/address";
import { createSeverableProxy, type SeverableProxy } from "@/testutil/proxy";

/** The root credentials of the test Core. */
export const TEST_CREDENTIALS = { username: "synnax", password: "seldon" };

export const TEST_CLIENT_PARAMS: SynnaxParams = {
  ...TEST_CORE_ADDRESS,
  ...TEST_CREDENTIALS,
  retry: {
    maxRetries: 4,
    baseInterval: TimeSpan.seconds(1),
    scale: 1.5,
  },
};

// A hook registered mid-test is silently dropped by vitest, so per-client afterAll
// calls leak clients created inside test bodies. This module-level hook registers
// during spec collection (import time) and covers every call site.
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
 * Creates a client connected to the local test cluster. The client is closed
 * automatically once every test in the current spec file has finished, so callers do
 * not need to close it themselves.
 */
export const createTestClient = (params?: Partial<SynnaxParams>): Synnax => {
  const client = new Synnax({ ...TEST_CLIENT_PARAMS, ...params });
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
    ...TEST_CLIENT_PARAMS,
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
 * Holds each response to target until release is called. responded resolves when the
 * first held response arrives.
 */
export const holdResponses = (
  client: Synnax,
  target: string,
  field: "transport" = "transport",
) => {
  const { unary } = client[field];
  const send = unary.send.bind(unary);
  let respond = (): void => {};
  const responded = new Promise<void>((resolve) => (respond = resolve));
  let release = (): void => {};
  const released = new Promise<void>((resolve) => (release = resolve));
  vi.spyOn(unary, "send").mockImplementation(async (...args) => {
    const res = await send(...args);
    if (args[0] === target) {
      respond();
      await released;
    }
    return res;
  });
  return { responded, release };
};
