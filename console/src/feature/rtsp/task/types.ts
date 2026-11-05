// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { rtsp, type task } from "@synnaxlabs/client";
import { z } from "zod";

export const PREFIX = "rtsp";
export const READ_TYPE = `${PREFIX}_read`;

export const READ_SCHEMAS = {
  type: z.literal(READ_TYPE),
  config: rtsp.readConfigZ,
  statusData: z.unknown().optional(),
} as const satisfies task.Schemas;

export type ReadSchemas = typeof READ_SCHEMAS;

export const deployReadConfigZ = rtsp.readConfigZ.extend({
  url: z.string().regex(/^rtsps?:\/\/./, "URL must start with rtsp:// or rtsps://"),
});
