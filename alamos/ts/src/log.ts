// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type record } from "@synnaxlabs/x";

import { Meta } from "@/meta";

export const LOG_LEVELS = ["debug", "info", "warn", "error"] as const;
export type LogLevel = (typeof LOG_LEVELS)[number];

export interface LogLevelFilterProps {
  key: string;
  path: string;
  level: LogLevel;
}

type KV = record.Unknown | (() => record.Unknown);

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
  return ({ path }) => {
    if (include != null && !include.some((k) => path.startsWith(k))) return false;
    if (exclude != null && exclude.some((k) => path.startsWith(k))) return false;
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

  debug(msg: string, kv?: KV, force: boolean = false): void {
    if (!this.filter("debug") && !force) return;
    if (kv == null) console.log("%cDEBUG", "color: #8c00f0;", this.meta.path, msg);
    else console.log("%cDEBUG", "color: #8c00f0;", this.meta.path, msg, parseKV(kv));
  }

  info(msg: string, kv?: KV, force: boolean = false): void {
    if (!this.filter("info") && !force) return;
    if (kv == null) console.log("%cINFO", "color: #005eff;", this.meta.path, msg);
    else console.log("%cINFO", "color: #005eff;", this.meta.path, msg, parseKV(kv));
  }

  warn(msg: string, kv?: KV, force: boolean = false): void {
    if (!this.filter("warn") && !force) return;
    if (kv == null) console.warn("WARN", this.meta.path, msg);
    else console.warn("WARN", this.meta.path, msg, parseKV(kv));
  }

  error(msg: string, kv?: KV, force: boolean = false): void {
    if (!this.filter("error") && !force) return;
    if (kv == null) console.error("ERROR", this.meta.path, msg);
    else console.error("ERROR", this.meta.path, msg, parseKV(kv));
  }

  static readonly NOOP = new Logger();
}

const parseKV = (kv: KV): record.Unknown => (typeof kv === "function" ? kv() : kv);
