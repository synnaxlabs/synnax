// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { type Source, sourceSpecZ } from "@/telem/aether/telem";

export interface LogEntry {
  channelKey: number;
  timestamp: bigint;
  value: string;
}

export interface LogSource extends Source<LogEntry[]> {
  readonly evictedCount: number;
  setChannels?: (channels: Array<number | string>) => void;
}

export const logSourceSpecZ = sourceSpecZ.extend({ valueType: z.literal("log") });
export type LogSourceSpec = z.infer<typeof logSourceSpecZ>;
