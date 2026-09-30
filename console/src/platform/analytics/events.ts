// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

/** The kinds of resource the app counts. */
export const resourceZ = z.enum([
  "channel",
  "device",
  "rack",
  "task",
  "range",
  "schematic",
  "lineplot",
  "log",
  "table",
  "arc",
]);

export type Resource = z.infer<typeof resourceZ>;

const exitReasonZ = z.enum(["failed_to_start", "crashed", "not_ready", "unresponsive"]);

/**
 * The properties of every event, keyed by event name. The schemas are strict, so a
 * property that is not declared here never reaches the sink. Values stay numbers,
 * booleans, and strings from a closed set: no name a user typed ever leaves the machine.
 */
export const schemas = {
  app_opened: z.strictObject({
    version: z.string(),
    os: z.string(),
    arch: z.string(),
    first_launch: z.boolean(),
    /** Null on a first launch, so retention can tell it apart from a fast relaunch. */
    hours_since_last_launch: z.number().nullable(),
  }),
  app_active: z.strictObject({
    /** False while the window sits open showing live data that nobody touches. */
    interacted: z.boolean(),
  }),
  command_run: z.strictObject({ command: z.string() }),
  resource_created: z.strictObject({ resource: resourceZ }),
  core_ready: z.strictObject({
    time_to_ready_ms: z.number(),
    starts: z.number(),
  }),
  core_exited: z.strictObject({
    reason: exitReasonZ,
    uptime_seconds: z.number(),
  }),
  /** The reason is that of the last exit before the app gave up. */
  core_restart_exhausted: z.strictObject({ reason: exitReasonZ }),
  core_reset: z.strictObject({ data_size_bytes: z.number() }),
} as const;

export type Name = keyof typeof schemas;

export type Properties<N extends Name> = z.infer<(typeof schemas)[N]>;

/** The Synnax account the app is signed in to, which is the person events belong to. */
export const accountZ = z.strictObject({
  /** The Clerk ID of the account. */
  id: z.string().min(1),
  email: z.email(),
});

export interface Account extends z.infer<typeof accountZ> {}

/**
 * What the user has built, recorded against the install rather than as an event. It
 * turns every count into a segment: an empty install and a loaded one are otherwise
 * indistinguishable.
 */
export const workspaceZ = z.strictObject({
  channel_count: z.number(),
  device_count: z.number(),
  rack_count: z.number(),
  task_count: z.number(),
  range_count: z.number(),
  schematic_count: z.number(),
  line_plot_count: z.number(),
  log_count: z.number(),
  table_count: z.number(),
  arc_count: z.number(),
});

export interface Workspace extends z.infer<typeof workspaceZ> {}
