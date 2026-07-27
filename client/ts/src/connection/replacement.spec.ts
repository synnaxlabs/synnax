// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ChildProcess, spawn } from "node:child_process";
import { existsSync, mkdtempSync, rmSync } from "node:fs";
import os from "node:os";
import path from "node:path";

import { afterEach, describe, expect, it } from "vitest";

import Synnax from "@/client";

const PORT = 9092;
const ADDRESS = `localhost:${PORT}`;
const BINARY = path.resolve(__dirname, "../../../../core/synnax");

const waitUntil = async (
  predicate: () => boolean | Promise<boolean>,
  timeout: number,
): Promise<void> => {
  const start = Date.now();
  while (Date.now() - start < timeout) {
    if (await predicate()) return;
    await new Promise((resolve) => setTimeout(resolve, 100));
  }
  throw new Error("timed out");
};

const portOccupied = async (): Promise<boolean> => {
  try {
    await fetch(`http://${ADDRESS}`);
    return true;
  } catch {
    return false;
  }
};

const startCore = async (dataDir?: string): Promise<ChildProcess> => {
  // A leaked core from an earlier run answers the readiness poll and keeps the
  // old cluster key alive, making every replacement assertion meaningless.
  if (await portOccupied()) throw new Error(`port ${PORT} is already occupied`);
  const storage = dataDir == null ? ["-m"] : ["-d", dataDir];
  const core = spawn(BINARY, ["start", "-i", "-l", ADDRESS, ...storage], {
    stdio: "ignore",
  });
  await waitUntil(async () => {
    if (core.exitCode != null) throw new Error("core exited during startup");
    return await portOccupied();
  }, 20000);
  return core;
};

const stopCore = async (core: ChildProcess): Promise<void> => {
  if (core.exitCode != null || core.signalCode != null) return;
  const exited = new Promise<void>((resolve) => core.once("exit", () => resolve()));
  core.kill();
  await exited;
};

// Boots its own cores on a dedicated port, so it needs the core binary built.
describe.skipIf(!existsSync(BINARY))("cluster replacement", () => {
  let core: ChildProcess | null = null;

  afterEach(async () => {
    if (core != null) await stopCore(core);
    core = null;
  });

  it("reconnects to the same cluster after escalated downtime", async () => {
    const dataDir = mkdtempSync(path.join(os.tmpdir(), "synnax-replacement-"));
    const internal: Error[] = [];
    const client = new Synnax({
      host: "localhost",
      port: PORT,
      username: "synnax",
      password: "seldon",
      onInternalError: (error) => {
        internal.push(error);
      },
    });
    try {
      core = await startCore(dataDir);
      await client.connect();
      const firstKey = client.connection.status.details.clusterKey;
      const project = await client.projects.create({ name: "survivor", layout: {} });
      await client.projects.retrieve({ keys: [project.key] });
      await stopCore(core);
      core = null;
      // sit through escalation so the machine parks in error(unreachable) and
      // the stream backoff climbs, matching a restart after real downtime
      await waitUntil(() => client.connection.status.variant === "error", 30000);
      await new Promise((resolve) => setTimeout(resolve, 5000));
      core = await startCore(dataDir);
      await waitUntil(
        () =>
          client.connection.status.variant === "success" &&
          client.connection.status.details.streamLive,
        15000,
      );
      expect(client.connection.status.details.clusterKey).toBe(firstKey);
      const chains = internal.map((err) => {
        const parts: string[] = [];
        let current: unknown = err;
        while (current instanceof Error) {
          parts.push(current.message);
          current = current.cause;
        }
        return parts.join(" <- ");
      });
      expect(chains).toEqual([]);
    } finally {
      client.close();
      rmSync(dataDir, { recursive: true, force: true });
    }
  }, 120000);

  it("adopts the replaced cluster without surfacing internal errors", async () => {
    core = await startCore();
    const internal: Error[] = [];
    const client = new Synnax({
      host: "localhost",
      port: PORT,
      username: "synnax",
      password: "seldon",
      onInternalError: (error) => {
        internal.push(error);
      },
    });
    try {
      await client.connect();
      const firstKey = client.connection.status.details.clusterKey;
      const project = await client.projects.create({ name: "doomed", layout: {} });
      // populate the cache so the post-reconnect reconcile has keys to check
      await client.projects.retrieve({ keys: [project.key] });
      await stopCore(core);
      await new Promise((resolve) => setTimeout(resolve, 2000));
      core = await startCore();
      await waitUntil(
        () =>
          client.connection.status.variant === "success" &&
          client.connection.status.details.clusterKey !== firstKey,
        30000,
      );
      // let late reconcile passes and stream churn settle before judging
      await new Promise((resolve) => setTimeout(resolve, 3000));
      const chains = internal.map((err) => {
        const parts: string[] = [];
        let current: unknown = err;
        while (current instanceof Error) {
          parts.push(current.message);
          current = current.cause;
        }
        return parts.join(" <- ");
      });
      expect(chains).toEqual([]);
    } finally {
      client.close();
    }
  }, 90000);
});
