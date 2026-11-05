// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type task, whip } from "@synnaxlabs/client";
import { z } from "zod";

export const PREFIX = "whip";
export const READ_TYPE = `${PREFIX}_read`;

export const READ_SCHEMAS = {
  type: z.literal(READ_TYPE),
  config: whip.readConfigZ,
  statusData: z.unknown().optional(),
} as const satisfies task.Schemas;

export type ReadSchemas = typeof READ_SCHEMAS;

/** @returns the URL that an encoder publishes the video of a task to. */
export const publishURL = (
  { host, port, secure }: { host: string; port: number; secure: boolean },
  key: task.Key,
): string => `${secure ? "https" : "http"}://${host}:${port}/api/v1/whip/${key}`;
