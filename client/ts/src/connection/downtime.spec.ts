// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type breaker, sleep, TimeSpan } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import Synnax from "@/client";
import { DisconnectedError } from "@/errors";
import {
  createSeverableProxy,
  type SeverableProxy,
  TEST_CLIENT_PARAMS,
  waitForStatus,
} from "@/testutil";

const FAST_RETRY: breaker.Config = {
  baseInterval: TimeSpan.milliseconds(10),
  maxInterval: TimeSpan.milliseconds(50),
  scale: 1.5,
};

interface ProxiedClient {
  proxy: SeverableProxy;
  client: Synnax;
  internal: Error[];
}

const createProxiedClient = async (
  retry: breaker.Config = FAST_RETRY,
): Promise<ProxiedClient> => {
  const proxy = await createSeverableProxy();
  const internal: Error[] = [];
  const client = new Synnax({
    host: "localhost",
    port: proxy.port,
    username: TEST_CLIENT_PARAMS.username,
    password: TEST_CLIENT_PARAMS.password,
    retry,
    onInternalError: (error) => internal.push(error),
  });
  return { proxy, client, internal };
};

const chainOf = (err: unknown): string => {
  const parts: string[] = [];
  let current: unknown = err;
  while (current instanceof Error) {
    parts.push(current.message);
    current = current.cause;
  }
  return parts.join(" <- ");
};

describe("downtime", () => {
  it("should reconnect to the same cluster after escalated downtime", async () => {
    const { proxy, client, internal } = await createProxiedClient();
    try {
      await client.connect();
      const firstKey = client.connection.status.details.clusterKey;
      const project = await client.projects.create({ name: "survivor", layout: {} });
      await client.projects.retrieve({ keys: [project.key] });
      await proxy.sever();
      // The default 30s heartbeat cannot notice the outage this fast: reaching
      // error proves the dropped change stream drove the state machine.
      await waitForStatus(
        client.connection,
        ({ variant, details }) =>
          variant === "error" && details.reason === "unreachable",
      );
      // A cached read never leaves the process, so it survives the outage.
      const offline = await client.projects.retrieve({ keys: [project.key] });
      expect(offline).toHaveLength(1);
      // A write must reach the cluster, so it short-circuits instead of hanging.
      await expect(
        client.projects.create({ name: "unreachable", layout: {} }),
      ).rejects.toThrow(DisconnectedError);
      await proxy.restore();
      const status = await waitForStatus(
        client.connection,
        ({ variant, details }) => variant === "success" && details.streamLive,
      );
      expect(status.details.clusterKey).toBe(firstKey);
      // A fresh write proves the unary path recovered, not just the probe loop.
      const recovered = await client.projects.create({ name: "recovered", layout: {} });
      expect(recovered.name).toBe("recovered");
      expect(internal.map(chainOf)).toEqual([]);
    } finally {
      client.close();
      await proxy.close();
    }
  });

  it("should carry an in-flight request through a brief blip", async () => {
    // Backoff slow enough that escalation cannot win the race against restore,
    // so the retrying request must ride through instead of short-circuiting.
    const { proxy, client, internal } = await createProxiedClient({
      baseInterval: TimeSpan.milliseconds(250),
      scale: 1.5,
    });
    try {
      await client.connect();
      await proxy.sever();
      // A create cannot be served from cache, so the breaker must retry it
      // across the outage and land it once the link returns.
      const inFlight = client.projects.create({ name: "blip", layout: {} });
      await sleep.sleep(TimeSpan.milliseconds(50));
      await proxy.restore();
      const created = await inFlight;
      expect(created.name).toBe("blip");
      await waitForStatus(
        client.connection,
        ({ variant, details }) => variant === "success" && details.streamLive,
      );
      expect(internal.map(chainOf)).toEqual([]);
    } finally {
      client.close();
      await proxy.close();
    }
  });
});
