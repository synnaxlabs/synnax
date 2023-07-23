// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Meta } from "./meta";

const LOG_LEVELS = ["debug", "info", "warn", "error"] as const;
export type LogLevel = typeof LOG_LEVELS[number];

export type LogLevelFilter = (level: LogLevel) => boolean;

export const logThresholdFilter = (thresh: LogLevel): LogLevelFilter => {
  const threshIdx = LOG_LEVELS.indexOf(thresh);
  return (level) => LOG_LEVELS.indexOf(level) >= threshIdx;
};

export class Logger {
  meta: Meta = Meta.NOOP;
  filter: LogLevelFilter;

  constructor(filter: LogLevelFilter = logThresholdFilter("info")) {
    this.filter = filter;
  }

  child(_: Meta): Logger {
    return new Logger();
  }

  debug(msg: string): void {
    if (this.meta.noop) return;
    console.log(msg);
  }

  info(msg: string): void {
    if (this.meta.noop) return;
    console.log(msg);
  }

  warn(msg: string): void {
    if (this.meta.noop) return;
    console.warn(msg);
  }

  error(msg: string): void {
    if (this.meta.noop) return;
    console.error(msg);
  }

  static readonly NOOP = new Logger();
}
