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
export const connectionZ = z.object({
  host: z.string(),
  port: z.number(),
  username: z.string(),
  password: z.string(),
  version: z.string(),
});
export interface Connection extends z.infer<typeof connectionZ> {}

/** The state of the embedded Core, as the Desktop shell reports it. */
export const statusZ = z.discriminatedUnion("state", [
  z.object({ state: z.literal("starting") }),
  z.object({ state: z.literal("running"), connection: connectionZ }),
  z.object({ state: z.literal("restarting") }),
  z.object({ state: z.literal("failed"), message: z.string() }),
  z.object({ state: z.literal("stopping") }),
  z.object({ state: z.literal("stopped") }),
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
 * Stops the embedded Core, erases everything it stored, and starts the app again. It
 * resolves only when the erase fails, because a success ends this launch.
 */
export const reset = async (): Promise<void> => await invoke("supervisor_reset");

/** Opens the log directory in the file manager of the operating system. */
export const showLogs = async (): Promise<void> => await invoke("supervisor_show_logs");

/** Opens the data directory in the file manager of the operating system. */
export const showData = async (): Promise<void> => await invoke("supervisor_show_data");

/** What the embedded Cores of this launch have done, and where their files are. */
export const diagnosticsZ = z.object({
  version: z.string(),
  history: z.object({
    /** The number of Cores started in this launch. */
    starts: z.number(),
    /** When the current Core became ready, in milliseconds since the Unix epoch. */
    readyAt: z.number().nullable(),
    /** Why the last Core exited without a stop request. */
    lastExit: z.string().nullable(),
  }),
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
