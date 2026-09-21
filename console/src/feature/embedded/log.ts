// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeStamp } from "@synnaxlabs/x";
import z from "zod";

const lineZ = z.object({
  level: z.string(),
  /** Seconds since the Unix epoch. */
  ts: z.number(),
  logger: z.string().optional(),
  msg: z.string(),
});

// The Core colors some messages for a terminal.
// eslint-disable-next-line no-control-regex
const ANSI_PATTERN = /\u001b\[[0-9;]*m/g;

const formatLine = (line: string): string => {
  let parsed: z.infer<typeof lineZ>;
  try {
    parsed = lineZ.parse(JSON.parse(line));
  } catch {
    // A panic trace or a line that was cut is still worth reading as it is.
    return line;
  }
  const { level, ts, logger, msg } = parsed;
  const time = TimeStamp.milliseconds(ts * 1000).toString("time", "local");
  const source = logger == null ? "" : ` ${logger}`;
  return `${time} ${level.toUpperCase()}${source} ${msg.replace(ANSI_PATTERN, "").trim()}`;
};

/**
 * Turns the structured log of the embedded Core into lines a person can read. A line
 * that is not a structured entry stays as it is.
 */
export const format = (log: string): string =>
  log
    .split("\n")
    .filter((line) => line.trim().length > 0)
    .map(formatLine)
    .join("\n");
