// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type task } from "@synnaxlabs/client";
import { JSON } from "@synnaxlabs/pluto";
import { Rate } from "@synnaxlabs/x";
import { z } from "zod";

import { Common } from "@/hardware/common";

export const PREFIX = "http";

export const SCAN_TYPE = `${PREFIX}_scan`;

export const scanTypeZ = z.literal(SCAN_TYPE);

const responseValidationZ = z.object({
  field: z.string().min(1, "JSON pointer is required"),
  expectedValue: JSON.primitiveZ,
});

export interface ResponseValidation extends z.infer<typeof responseValidationZ> {}

export const ZERO_RESPONSE_VALIDATION: ResponseValidation = {
  field: "",
  expectedValue: null,
};

export const scanConfigZ = Common.Task.baseConfigZ.extend({
  rate: z.number().positive("Rate must be positive"),
  path: z.string().min(1, "Path is required"),
  response: responseValidationZ.optional(),
});

export interface ScanConfig extends z.infer<typeof scanConfigZ> {}

export const ZERO_SCAN_CONFIG = {
  ...Common.Task.ZERO_BASE_CONFIG,
  rate: Rate.hz(1).valueOf(),
  path: "",
} as const satisfies ScanConfig;

interface ScanPayload extends task.Payload<typeof scanTypeZ, typeof scanConfigZ> {}

export const ZERO_SCAN_PAYLOAD: ScanPayload = {
  key: "",
  name: "HTTP Scan Task",
  config: ZERO_SCAN_CONFIG,
  type: SCAN_TYPE,
};

export const SCAN_SCHEMAS: task.Schemas<typeof scanTypeZ, typeof scanConfigZ> = {
  typeSchema: scanTypeZ,
  configSchema: scanConfigZ,
  statusDataSchema: z.unknown(),
};
