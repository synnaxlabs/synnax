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

const entryZ = z.looseObject({
  level: z.string(),
  /** Seconds since the Unix epoch. */
  ts: z.number(),
  logger: z.string().optional(),
  msg: z.string(),
});

// The caller and the stack trace fill the dialog without helping a reader. The export
// keeps them.
const HIDDEN_FIELDS = new Set(["level", "ts", "logger", "msg", "caller", "stacktrace"]);

// The Core colors some messages for a terminal.
// eslint-disable-next-line no-control-regex
const ANSI_PATTERN = /\u001b\[[0-9;]*m/g;

/** One line of the Core log. A line that is not a structured entry keeps only text. */
export interface Entry {
  /** The level as the Core wrote it, such as "info" or "error". */
  level: string | null;
  time: TimeStamp | null;
  logger: string;
  /** The message, then the other fields of the entry as key=value pairs. */
  text: string;
}

const formatValue = (value: unknown): string =>
  typeof value === "string" && /^[^\s"=]+$/.test(value) ? value : JSON.stringify(value);

const parseLine = (line: string): Entry => {
  let parsed: z.infer<typeof entryZ>;
  try {
    parsed = entryZ.parse(JSON.parse(line));
  } catch {
    // A panic trace or a line that was cut is still worth reading as it is.
    return { level: null, time: null, logger: "", text: line };
  }
  const { level, ts, logger = "", msg } = parsed;
  const fields = Object.entries(parsed)
    .filter(([key]) => !HIDDEN_FIELDS.has(key))
    .map(([key, value]) => `${key}=${formatValue(value)}`);
  const text = [msg.replace(ANSI_PATTERN, "").trim(), ...fields].join(" ");
  return { level, time: TimeStamp.milliseconds(ts * 1000), logger, text };
};

/** Splits the structured log of the embedded Core into entries a person can read. */
export const parse = (log: string): Entry[] =>
  log
    .split("\n")
    .filter((line) => line.trim().length > 0)
    .map(parseLine);

/** @returns the entries as plain text, with full UTC times for a reader elsewhere. */
export const toText = (entries: Entry[]): string =>
  entries
    .map(({ level, time, logger, text }) =>
      [time?.toString("ISO", "UTC"), level?.toUpperCase(), logger, text]
        .filter((part) => part != null && part !== "")
        .join(" "),
    )
    .join("\n");
