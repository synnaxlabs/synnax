// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

const exitReasonZ = z.enum([
  "failed_to_start",
  "crashed",
  "not_ready",
  "unresponsive",
  "locked",
]);

/**
 * The properties of every event, keyed by event name. The schemas are strict, so a
 * property that is not declared here never reaches the sink. Values stay numbers,
 * booleans, and identifiers the code sets, so no name a user typed reaches an event.
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
  channel_created: z.strictObject({ calculated: z.boolean() }),
  device_connected: z.strictObject({
    integration: z.string(),
  }),
  device_configured: z.strictObject({ make: z.string() }),
  task_started: z.strictObject({ type: z.string() }),
  automation_deployed: z.strictObject({}),
  plot_created: z.strictObject({}),
  schematic_created: z.strictObject({}),
  range_created: z.strictObject({}),
  core_ready: z.strictObject({
    time_to_ready_ms: z.number(),
    starts: z.number(),
  }),
  core_exited: z.strictObject({
    reason: exitReasonZ,
    uptime_seconds: z.number(),
    exit_code: z.number().nullable(),
  }),
  /** The reason and code are those of the last exit before the app gave up. */
  core_restart_exhausted: z.strictObject({
    reason: exitReasonZ,
    exit_code: z.number().nullable(),
  }),
  /**
   * The Core waited for another process to free its data directory. A wait that timed
   * out is followed by `core_restart_exhausted` with the reason `locked`.
   */
  core_waited: z.strictObject({ wait_ms: z.number() }),
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
