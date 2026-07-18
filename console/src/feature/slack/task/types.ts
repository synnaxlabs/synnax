// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type task } from "@synnaxlabs/client";
import { record } from "@synnaxlabs/x";
import { z } from "zod";

export const PREFIX = "slack";
export const ALERT_TYPE = `${PREFIX}_alert`;
export const SCAN_TYPE = `${PREFIX}_scan`;

export const TEST_CONNECTION_COMMAND_TYPE = "test_connection";

export const SCAN_SCHEMAS = {
  type: z.literal(SCAN_TYPE),
  config: record.nullishToEmpty(),
  statusData: z.null().optional(),
} as const satisfies task.Schemas;

const alertTaskConfigZ = z.object({
  device: z.string().min(1, "A workspace is required"),
  channel: z.string().min(1, "A channel is required"),
  statuses: z
    .array(z.string().min(1, "Status key is required"))
    .default([])
    .refine((s) => s.length > 0, { message: "At least one status is required" }),
  autoStart: z.boolean().default(false),
});

export interface AlertTaskConfig extends z.infer<typeof alertTaskConfigZ> {}

export const ZERO_ALERT_TASK_CONFIG: AlertTaskConfig = {
  device: "",
  channel: "",
  statuses: [],
  autoStart: false,
};

export const ALERT_SCHEMAS = {
  type: z.literal(ALERT_TYPE),
  config: alertTaskConfigZ,
  statusData: z.unknown().optional(),
} as const satisfies task.Schemas;

export type AlertSchemas = typeof ALERT_SCHEMAS;

export interface AlertPayload extends task.Payload<AlertSchemas> {}

export const ZERO_ALERT_PAYLOAD: AlertPayload = {
  key: "",
  type: ALERT_TYPE,
  name: "Slack Alert Task",
  config: ZERO_ALERT_TASK_CONFIG,
  internal: false,
  snapshot: false,
};
