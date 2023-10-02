// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnknownRecord } from "@synnaxlabs/x";

import { Meta } from "@/meta";

export const LOG_LEVELS = ["debug", "info", "warn", "error"] as const;
export type LogLevel = (typeof LOG_LEVELS)[number];

export interface LogLevelFilterProps {
  key: string;
  path: string;
  level: LogLevel;
}

/**
 * LogLevelFilter is a function that returns true if the log at the given
 * level should be emitted.
 */
export type LogLevelFilter = (props: LogLevelFilterProps) => boolean;

export const logThresholdFilter = (thresh: LogLevel): LogLevelFilter => {
  const threshIdx = LOG_LEVELS.indexOf(thresh);
  return ({ level }) => LOG_LEVELS.indexOf(level) >= threshIdx;
};

export interface LogLevelKeyFilterProps {
  include?: string[];
  exclude?: string[];
}
export const logLevelKeyFiler = (props: LogLevelKeyFilterProps): LogLevelFilter => {
  const { include, exclude } = props;
  return ({ key }) => {
    if (include != null && !include.includes(key)) return false;
    if (exclude != null && exclude.includes(key)) return false;
    return true;
  };
};

export interface LoggerProps {
  filters?: LogLevelFilter[];
}

export class Logger {
  meta: Meta = Meta.NOOP;
  filters: LogLevelFilter[];

  constructor(p: LoggerProps = {}) {
    const { filters = [] } = p;
    this.filters = filters;
  }

  private filter(level: LogLevel): boolean {
    return (
      !this.meta.noop &&
      this.filters.every((f) =>
        f({
          key: this.meta.key,
          path: this.meta.path,
          level,
        }),
      )
    );
  }

  child(meta: Meta): Logger {
    const l = new Logger({ filters: this.filters });
    l.meta = meta;
    return l;
  }

  debug(msg: string, kv?: UnknownRecord): void {
    if (!this.filter("debug")) return;
    console.log("%cDEBUG", "color: #8c00f0;", this.meta.path, msg, kv);
  }

  info(msg: string, kv?: UnknownRecord): void {
    if (!this.filter("info")) return;
    console.log("INFO", "color: #005eff;", this.meta.path, msg, kv);
  }

  warn(msg: string, kv?: UnknownRecord): void {
    if (!this.filter("warn")) return;
    console.warn("WARN", this.meta.path, msg, kv);
  }

  error(msg: string, kv?: UnknownRecord): void {
    if (!this.filter("error")) return;
    console.error("ERROR", this.meta.path, msg, kv);
  }

  static readonly NOOP = new Logger();
}
