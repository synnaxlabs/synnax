// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, sift, type task } from "@synnaxlabs/client";
import { record } from "@synnaxlabs/x";
import { z } from "zod";

import { Task } from "@/platform/task";

export const PREFIX = "sift";

export const SCAN_TYPE = `${PREFIX}_scan`;

export const TEST_CONNECTION_COMMAND_TYPE = "test_connection";

export const LIST_ASSETS_COMMAND_TYPE = "list_assets";

export const UPLOAD_COMMAND_TYPE = "upload";

export const LIST_UPLOADS_COMMAND_TYPE = "list_uploads";

export const SCAN_SCHEMAS = {
  type: z.literal(SCAN_TYPE),
  config: record.nullishToEmpty(),
  statusData: z.unknown().optional(),
} as const satisfies task.Schemas;

export const STREAM_TYPE = `${PREFIX}_stream`;

/** Starts the key of the progress status of every stream task. */
export const PROGRESS_KEY_PREFIX = `${STREAM_TYPE}:`;

export const deployStreamConfigZ = sift.streamConfigZ.extend({
  device: Task.deviceKeyZ,
  asset: z.string().min(1, "Asset is required"),
  channels: channel.keyZ.array().min(1, "Select at least one channel"),
});

export const STREAM_SCHEMAS = {
  type: z.literal(STREAM_TYPE),
  config: sift.streamConfigZ,
  statusData: z.unknown().optional(),
} as const satisfies task.Schemas;

export type StreamSchemas = typeof STREAM_SCHEMAS;
