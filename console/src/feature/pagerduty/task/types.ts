// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { pagerduty, type task } from "@synnaxlabs/client";
import { z } from "zod";

export const PREFIX = "pagerduty";
export const ALERT_TYPE = `${PREFIX}_alert`;

export interface AlertConfig extends pagerduty.Alert {}

const alertTaskConfigZ = pagerduty.taskConfigZ;

const deployAlertConfigZ = pagerduty.alertZ.extend({
  status: z.string().min(1, "Status key is required"),
});

export const deployAlertTaskConfigZ = alertTaskConfigZ.extend({
  routingKey: z.string().length(32, "Routing key must be 32 characters"),
  alerts: z
    .array(deployAlertConfigZ)
    .refine((alerts) => alerts.some(({ disabled }) => !disabled), {
      message: "At least one alert must be enabled",
    }),
});

export interface AlertTaskConfig extends z.infer<typeof alertTaskConfigZ> {}

export const ALERT_SCHEMAS = {
  type: z.literal(ALERT_TYPE),
  config: alertTaskConfigZ,
  statusData: z.unknown().optional(),
} as const satisfies task.Schemas;

export type AlertSchemas = typeof ALERT_SCHEMAS;

export interface AlertPayload extends task.Payload<AlertSchemas> {}
