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

/** Starts the embedded Core again after it failed or was stopped. */
export const restart = async (): Promise<void> => await invoke("supervisor_restart");

/** Stops the embedded Core and resolves once its process has exited. */
export const stop = async (): Promise<void> => await invoke("supervisor_stop");

/** Opens the log directory in the file manager of the operating system. */
export const showLogs = async (): Promise<void> => await invoke("supervisor_show_logs");
