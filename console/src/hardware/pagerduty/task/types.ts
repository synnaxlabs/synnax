// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type task } from "@synnaxlabs/client";
import { z } from "zod";

export const PREFIX = "pagerduty";
export const ALERT_TYPE = `${PREFIX}_alert`;

const alertConfigZ = z.object({
  key: z.string(),
  status: z.string().min(1, "Status key is required"),
  treatErrorAsCritical: z.boolean().default(false),
  component: z.string().default(""),
  group: z.string().default(""),
  class: z.string().default(""),
  enabled: z.boolean().default(true),
});

export interface AlertConfig extends z.infer<typeof alertConfigZ> {}

export const ZERO_ALERT_CONFIG: AlertConfig = {
  key: "",
  status: "",
  treatErrorAsCritical: false,
  component: "",
  group: "",
  class: "",
  enabled: true,
};

const alertTaskConfigZ = z.object({
  routingKey: z.string().length(32, "Routing key must be 32 characters"),
  autoStart: z.boolean().default(false),
  alerts: z
    .array(alertConfigZ)
    .default([])
    .refine((alerts) => alerts.some(({ enabled }) => enabled), {
      message: "At least one alert must be enabled",
    }),
});

export interface AlertTaskConfig extends z.infer<typeof alertTaskConfigZ> {}

export const ZERO_ALERT_TASK_CONFIG: AlertTaskConfig = {
  routingKey: "",
  autoStart: false,
  alerts: [],
};

export const ALERT_SCHEMAS = {
  type: z.literal(ALERT_TYPE),
  config: alertTaskConfigZ,
  statusData: z.unknown(),
} as const satisfies task.Schemas;

export type AlertSchemas = typeof ALERT_SCHEMAS;

export interface AlertPayload extends task.Payload<AlertSchemas> {}

export const ZERO_ALERT_PAYLOAD: AlertPayload = {
  key: "",
  type: ALERT_TYPE,
  name: "PagerDuty Alert Task",
  config: ZERO_ALERT_TASK_CONFIG,
};
