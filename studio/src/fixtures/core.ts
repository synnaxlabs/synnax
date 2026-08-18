// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ChildProcess, spawn } from "node:child_process";
import { once } from "node:events";
import { createWriteStream, existsSync } from "node:fs";
import { mkdir } from "node:fs/promises";
import net from "node:net";
import path from "node:path";

export interface CoreOptions {
  /** Path to the synnax core binary. Defaults to $SYNNAX_CORE_BIN, then the
   * first `core/synnax` found walking up from this package. */
  binary?: string;
  /** Port to listen on. Must be 9090 for the dev Console, which hardcodes its
   * dev connection to localhost:9090. */
  port?: number;
  /** Directory to write core.log into. */
  outDir: string;
  /** Enable the embedded driver. Off by default so captures never show rack or
   * driver state that isn't part of the script. */
  driver?: boolean;
}

export interface Core {
  port: number;
  stop: () => Promise<void>;
}

const READY_TIMEOUT_MS = 30_000;

const findBinary = (): string => {
  const fromEnv = process.env.SYNNAX_CORE_BIN;
  if (fromEnv != null && fromEnv.length > 0) return fromEnv;
  let dir = import.meta.dirname;
  while (true) {
    const candidate = path.join(dir, "core", "synnax");
    if (existsSync(candidate)) return candidate;
    const parent = path.dirname(dir);
    if (parent === dir) break;
    dir = parent;
  }
  throw new Error(
    "synnax core binary not found: set SYNNAX_CORE_BIN or build core/synnax " +
      "(cd core && go build -o synnax .)",
  );
};

const portInUse = async (port: number): Promise<boolean> =>
  await new Promise((resolve) => {
    const socket = net.connect({ port, host: "localhost" });
    socket.once("connect", () => {
      socket.destroy();
      resolve(true);
    });
    socket.once("error", () => resolve(false));
  });

const waitReady = async (port: number, proc: ChildProcess): Promise<void> => {
  const deadline = Date.now() + READY_TIMEOUT_MS;
  while (Date.now() < deadline) {
    if (proc.exitCode != null)
      throw new Error(`core exited with code ${proc.exitCode} before ready`);
    if (await portInUse(port)) return;
    await new Promise((resolve) => setTimeout(resolve, 200));
  }
  throw new Error(`core did not become ready on port ${port} within 30s`);
};

/**
 * startCore launches a fresh in-memory Synnax core for the duration of a
 * capture, so every capture sees an empty cluster: no ranges, racks, or
 * projects left behind by dev work or integration test runs.
 *
 * Refuses to start if the port is already in use: a shared long-lived core is
 * exactly the contamination this exists to prevent, and silently reusing it
 * would make captures non-deterministic. Stop the running core first.
 */
export const startCore = async ({
  binary = findBinary(),
  port = 9090,
  outDir,
  driver = false,
}: CoreOptions): Promise<Core> => {
  if (await portInUse(port))
    throw new Error(
      `port ${port} is already in use: stop the running core before capturing ` +
        "(the studio manages its own ephemeral core), or pass --core external " +
        "to capture against the running one",
    );
  await mkdir(outDir, { recursive: true });
  const log = createWriteStream(path.join(outDir, "core.log"));
  const args = ["start", "-mi", "-l", `localhost:${port}`];
  if (!driver) args.push("--no-driver");
  const proc = spawn(binary, args, { stdio: ["ignore", "pipe", "pipe"] });
  proc.stdout.pipe(log);
  proc.stderr.pipe(log);
  await waitReady(port, proc);
  return {
    port,
    stop: async () => {
      if (proc.exitCode != null) return;
      proc.kill("SIGTERM");
      await Promise.race([
        once(proc, "exit"),
        new Promise((resolve) => setTimeout(resolve, 5000)),
      ]);
      if (proc.exitCode == null) proc.kill("SIGKILL");
    },
  };
};
