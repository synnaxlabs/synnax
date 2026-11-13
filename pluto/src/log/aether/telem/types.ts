// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, type TimeRange, type TimeStamp } from "@synnaxlabs/client";
import { z } from "zod";

import { type Source, sourceSpecZ } from "@/telem/aether/telem";

export interface LogEntry {
  channelKey: channel.Key;
  timestamp: TimeStamp;
  value: string;
  // True when this entry was produced by splitting a sample on \n;
  continuation?: boolean;
  // The alignment of the sample, which tells apart samples that share a time.
  alignment?: bigint;
}

export interface LogSource extends Source<LogEntry[]> {
  setChannels?: (channels: channel.Key[]) => void;
  /** Reads entries older than the oldest one, if the channels store any. */
  loadOlder?: () => void;
  /** Reads entries newer than the newest one, when the source is not live. */
  loadNewer?: () => void;
  /** The times the channels store, or null until the oldest is known. */
  readonly timeRange?: TimeRange | null;
  /** Whether the entries hold the oldest stored sample of each channel. A source
   * without it holds all it has. */
  readonly exhausted?: boolean;
  /** Whether the entries end at the newest samples. A source without it is live. */
  readonly live?: boolean;
  /** Moves a source that is not live back to the newest samples. */
  resume?: () => void;
  /** The count of lines streamed in after the first read of their channel. */
  readonly streamedLines?: number;
}

export const logSourceSpecZ = sourceSpecZ.extend({ valueType: z.literal("log") });
export type LogSourceSpec = z.infer<typeof logSourceSpecZ>;
