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

import { breaker, TimeSpan } from "@synnaxlabs/x";

import { portInUse } from "@/fixtures/core";

const CLIENT_PY = path.resolve(import.meta.dirname, "../../../client/py");
const CORE = path.resolve(import.meta.dirname, "../../../core");
/** The fake servers that run as Go programs from the core module. */
const GO_MAINS = {
  kafka: path.join(import.meta.dirname, "kafka/main.go"),
  sift: path.join(import.meta.dirname, "sift/main.go"),
};

const READY_TIMEOUT_MS = 30_000;

/** Long enough for the first `go run` of a Go server to compile. */
const GO_READY_TIMEOUT_MS = 180_000;

const STOP_TIMEOUT_MS = 10_000;

const POLL_MS = 200;

/** The Python code that runs each server from `client/py/examples`. */
const PROGRAMS = {
  mqtt: (host: string, port: number) =>
    "import asyncio\n" +
    "from examples.mqtt_sim.server import run_server\n" +
    `asyncio.run(run_server("${host}", ${port}))`,
  modbus: (host: string, port: number) =>
    "import asyncio\n" +
    "from examples.modbus.server import run_server\n" +
    `asyncio.run(run_server("${host}", ${port}))`,
  opcua: (host: string, port: number) =>
    "import asyncio\n" +
    "from examples.opcua.sim import run_server\n" +
    `asyncio.run(run_server(endpoint="opc.tcp://${host}:${port}"))`,
  // The certificate shows on screen and names the machine, so the server makes a new
  // one with a fixed host name instead of reusing the one the examples keep.
  "opcua-encrypted": (host: string, port: number) =>
    "import asyncio, socket, tempfile\n" +
    "from pathlib import Path\n" +
    "from examples.opcua import sim\n" +
    'socket.gethostname = lambda: "opcua-server"\n' +
    "sim.CERT_DIR = Path(tempfile.mkdtemp())\n" +
    `asyncio.run(sim.run_server(endpoint="opc.tcp://${host}:${port}", encrypted=True))`,
  http: (host: string, port: number) =>
    "from examples.http_sim.server import run_server\n" +
    `run_server("${host}", ${port})`,
};

export interface SimulatorOptions {
  /** The protocol of the server. */
  protocol: keyof typeof PROGRAMS | keyof typeof GO_MAINS;
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
 * A Kafka or Sift server is the Go fake in `kafka/main.go` or `sift/main.go`. Refuses
 * to start if the port is in use.
 */
export const simulator = async ({
  protocol,
  port,
  host = "127.0.0.1",
}: SimulatorOptions): Promise<SimulatorFixture> => {
  if (await portInUse(port)) throw new Error(`port ${port} is already in use`);
  const env = { ...process.env };
  delete env.VIRTUAL_ENV;
  const goMain = protocol === "kafka" || protocol === "sift";
  const [command, args, cwd]: [string, string[], string] = goMain
    ? ["go", ["run", GO_MAINS[protocol], `--port=${port}`], CORE]
    : ["uv", ["run", "python", "-c", PROGRAMS[protocol](host, port)], CLIENT_PY];
  const proc = spawn(command, args, { cwd, env, stdio: "ignore", detached: true });
  const stop = async (): Promise<void> => {
    if (proc.exitCode != null || proc.pid == null) return;
    // uv and go run their programs as children, so signal the whole group.
    process.kill(-proc.pid, "SIGTERM");
    await once(proc, "exit");
    // `go run` exits before the server it started closes its port.
    const closing = new breaker.Breaker({
      baseInterval: TimeSpan.milliseconds(POLL_MS),
      maxRetries: STOP_TIMEOUT_MS / POLL_MS,
    });
    while (await portInUse(port))
      if (!(await closing.wait()))
        throw new Error(`${protocol} server did not close port ${port} in time`);
  };
  const timeout = goMain ? GO_READY_TIMEOUT_MS : READY_TIMEOUT_MS;
  const ready = new breaker.Breaker({
    baseInterval: TimeSpan.milliseconds(POLL_MS),
    maxRetries: timeout / POLL_MS,
  });
  while (!(await portInUse(port))) {
    if (proc.exitCode != null)
      throw new Error(`${protocol} server exited with code ${proc.exitCode}`);
    if (!(await ready.wait())) {
      await stop();
      throw new Error(`${protocol} server did not open port ${port} in time`);
    }
  }
  return { stop };
};
