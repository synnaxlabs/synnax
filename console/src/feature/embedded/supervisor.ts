// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { invoke } from "@tauri-apps/api/core";
import { listen, type UnlistenFn } from "@tauri-apps/api/event";
import z from "zod";

/** How a client reaches the embedded Core for the length of one launch. */
const connectionZ = z.object({
  host: z.string(),
  port: z.number(),
  username: z.string(),
  password: z.string(),
});
export interface Connection extends z.infer<typeof connectionZ> {}

/** Why a Core exited without a stop request. */
const reasonZ = z.enum([
  "failed_to_start",
  "crashed",
  "not_ready",
  "unresponsive",
  "locked",
]);

/** The state of the embedded Core, as the Desktop shell reports it. */
export const statusZ = z.discriminatedUnion("state", [
  z.object({ state: z.literal("starting") }),
  z.object({ state: z.literal("running"), connection: connectionZ }),
  z.object({ state: z.literal("restarting") }),
  /** Another process holds the data directory. A Core starts once it is free. */
  z.object({ state: z.literal("waiting") }),
  z.object({ state: z.literal("failed"), reason: reasonZ, message: z.string() }),
  /** The Core was asked to stop, and no new one follows. */
  z.object({ state: z.literal("stopping") }),
  z.object({ state: z.literal("stopped") }),
  z.object({ state: z.literal("exiting") }),
]);
export type Status = z.infer<typeof statusZ>;

export const STARTING_STATUS: Status = { state: "starting" };

const STATUS_EVENT = "supervisor://status";

/** @returns The current status of the embedded Core. */
export const retrieveStatus = async (): Promise<Status> =>
  statusZ.parse(await invoke("supervisor_status"));

/**
 * Calls the handler on each status change of the embedded Core.
 * @returns A function that stops the subscription.
 */
export const onStatusChange = async (
  handler: (status: Status) => void,
): Promise<UnlistenFn> =>
  await listen(STATUS_EVENT, ({ payload }) => handler(statusZ.parse(payload)));

/** Starts a new embedded Core. One that runs stops in order first. */
export const restart = async (): Promise<void> => await invoke("supervisor_restart");

/** Stops the embedded Core and resolves once its process has exited. */
export const stop = async (): Promise<void> => await invoke("supervisor_stop");

/**
 * Stops the embedded Core while the windows stay open, then exits the app. A launch of
 * the app during the stop starts it again.
 */
export const quit = async (): Promise<void> => await invoke("supervisor_quit");

/**
 * Stops the embedded Core, erases everything it stored, and starts the app again. It
 * resolves only when the erase fails, because a success ends this launch.
 */
export const reset = async (): Promise<void> => await invoke("supervisor_reset");

/** Opens the log directory in the file manager of the operating system. */
export const showLogs = async (): Promise<void> => await invoke("supervisor_show_logs");

/** Opens the data directory in the file manager of the operating system. */
export const showData = async (): Promise<void> => await invoke("supervisor_show_data");

/** What the embedded Cores of this launch have done. */
const historyZ = z.object({
  /** The number of Cores started in this launch. */
  starts: z.number(),
  /**
   * The number of Cores that exited without a stop request. A wait for the data
   * directory counts once, when it times out.
   */
  exits: z.number(),
  /** The number of times the restart policy gave up on the Core. */
  failures: z.number(),
  /** The number of Cores that became ready. */
  readies: z.number(),
  /** When the current Core became ready, in milliseconds since the Unix epoch. */
  readyAt: z.number().nullable(),
  /** How long the last Core that became ready took to do so. */
  timeToReadyMs: z.number().nullable(),
  /** How the last Core exited without a stop request. */
  lastExit: z
    .object({
      reason: reasonZ,
      /** What went wrong, for a person to read. */
      message: z.string(),
      uptimeSeconds: z.number(),
      /** The exit code of a process that exited on its own. */
      exitCode: z.number().nullable(),
    })
    .nullable(),
  /** The number of waits for the data directory that ended. */
  waits: z.number(),
  /** How long the last wait for the data directory took. */
  lastWaitMs: z.number().nullable(),
});
export interface History extends z.infer<typeof historyZ> {}

/** @returns What the embedded Cores of this launch have done. */
export const retrieveHistory = async (): Promise<History> =>
  historyZ.parse(await invoke("supervisor_history"));

/** What the embedded Cores of this launch have done, and where their files are. */
const diagnosticsZ = z.object({
  version: z.string(),
  history: historyZ,
  dataDir: z.string(),
  logDir: z.string(),
  /** The size of the data directory in bytes. */
  dataSize: z.number(),
});
export interface Diagnostics extends z.infer<typeof diagnosticsZ> {}

export const retrieveDiagnostics = async (): Promise<Diagnostics> =>
  diagnosticsZ.parse(await invoke("supervisor_diagnostics"));

/** @returns The last lines of the log of the embedded Core. */
export const retrieveLogTail = async (): Promise<string> =>
  z.string().parse(await invoke("supervisor_log_tail"));

/** Writes a zip archive of the logs and a summary to the given path. */
export const exportDiagnostics = async (path: string): Promise<void> =>
  await invoke("supervisor_export_diagnostics", { path });

/**
 * @returns The API key that scripts on this computer use, or null when none is saved.
 */
export const retrieveAPIKey = async (): Promise<string | null> =>
  z
    .string()
    .nullable()
    .parse(await invoke("supervisor_api_key"));

/**
 * Saves the API key, with the address of the running Core, where the client libraries
 * on this computer find it. The save happens only while the saved key is `replaces`.
 * @param replaces - The key the caller read, or null when none was saved.
 * @returns False, with no save, when another window changed the key first.
 */
export const saveAPIKey = async (
  apiKey: string,
  replaces: string | null,
): Promise<boolean> =>
  z.boolean().parse(await invoke("supervisor_save_api_key", { apiKey, replaces }));
