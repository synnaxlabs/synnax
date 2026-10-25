// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { spawn } from "node:child_process";
import { once } from "node:events";
import path from "node:path";

import { portInUse } from "@/fixtures/core";

const CLIENT_PY = path.resolve(import.meta.dirname, "../../../client/py");

const READY_TIMEOUT_MS = 30_000;

/** The Python code that runs each server from `client/py/examples`. */
const PROGRAMS = {
  modbus: (host: string, port: number) =>
    "import asyncio\n" +
    "from examples.modbus.server import run_server\n" +
    `asyncio.run(run_server("${host}", ${port}))`,
  opcua: (host: string, port: number) =>
    "import asyncio\n" +
    "from examples.opcua.sim import run_server\n" +
    `asyncio.run(run_server("opc.tcp://${host}:${port}"))`,
  http: (host: string, port: number) =>
    "from examples.http_sim.server import run_server\n" +
    `run_server("${host}", ${port})`,
};

export interface SimulatorOptions {
  /** The protocol of the server. */
  protocol: keyof typeof PROGRAMS;
  /** Port the server listens on. */
  port: number;
  /** Address the server listens on. Defaults to 127.0.0.1. */
  host?: string;
}

export interface SimulatorFixture {
  /** Stops the server. */
  stop: () => Promise<void>;
}

/**
 * simulator runs a device server from the Python client examples, the same servers the
 * driver integration tests use, so the real Driver connects to it and reads its values.
 * Refuses to start if the port is in use.
 */
export const simulator = async ({
  protocol,
  port,
  host = "127.0.0.1",
}: SimulatorOptions): Promise<SimulatorFixture> => {
  if (await portInUse(port)) throw new Error(`port ${port} is already in use`);
  const env = { ...process.env };
  delete env.VIRTUAL_ENV;
  const proc = spawn("uv", ["run", "python", "-c", PROGRAMS[protocol](host, port)], {
    cwd: CLIENT_PY,
    env,
    stdio: "ignore",
    detached: true,
  });
  const stop = async (): Promise<void> => {
    if (proc.exitCode != null || proc.pid == null) return;
    // uv runs Python as a child, so signal the whole group.
    process.kill(-proc.pid, "SIGTERM");
    await once(proc, "exit");
  };
  const deadline = Date.now() + READY_TIMEOUT_MS;
  while (!(await portInUse(port))) {
    if (proc.exitCode != null)
      throw new Error(`${protocol} server exited with code ${proc.exitCode}`);
    if (Date.now() > deadline) {
      await stop();
      throw new Error(`${protocol} server did not open port ${port} within 30s`);
    }
    await new Promise((resolve) => setTimeout(resolve, 200));
  }
  return { stop };
};
