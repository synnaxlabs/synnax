// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/**
 * Console log collector for performance profiling.
 *
 * Current implementation (Option 1): Captures all console messages
 *
 * Future extension (Option 3): Add hybrid mode that:
 * - Counts all levels (log, warn, error, info)
 * - Only stores warn/error messages
 * - Shows count breakdown: "X logs, Y warnings, Z errors"
 * - Reduces memory footprint while focusing on actionable issues
 *
 * Design is extensible via mode toggle:
 * - this.mode: 'full' | 'minimal'
 * - Conditional storage based on mode
 * - Always track counts for rate metrics
 */

import { type MetricTableColumn } from "@/perf/components/MetricTable";
import {
  CONSOLE_LOG_WINDOW_MS,
  MAX_MESSAGE_LENGTH,
  MAX_STORED_MESSAGES,
  TEXT_ROW_COLOR,
} from "@/perf/constants";
import { formatAge } from "@/perf/utils/formatting";

interface CapturedMessage {
  level: "log" | "warn" | "error" | "info";
  message: string;
  timestamp: number;
  stack?: string;
}

export interface ConsoleLogEntry {
  level: "log" | "warn" | "error" | "info";
  message: string;
  timestamp: number;
  age: number;
  stack?: string;
}

export const CONSOLE_LOG_TABLE_COLUMNS: MetricTableColumn<ConsoleLogEntry>[] = [
  {
    getValue: (entry) => entry.level.toUpperCase(),
    color: TEXT_ROW_COLOR,
  },
  {
    getValue: (entry) => formatAge(entry.age),
    color: TEXT_ROW_COLOR,
  },
  {
    getValue: (entry) => {
      const maxLength = 60;
      return entry.message.length > maxLength
        ? entry.message.slice(0, maxLength) + "..."
        : entry.message;
    },
    color: TEXT_ROW_COLOR,
  },
];

export const getConsoleLogTableKey = (
  entry: ConsoleLogEntry,
  index: number,
): string => `${entry.timestamp}-${index}`;

export const getConsoleLogTableTooltip = (entry: ConsoleLogEntry): string => {
  let tooltip = entry.message;
  if (entry.stack != null) {
    tooltip += "\n\nStack trace:\n" + entry.stack;
  }
  return tooltip;
};

/**
 * Tracks console log messages for performance profiling.
 * Intercepts console.log, console.warn, console.error, and console.info.
 */
export class ConsoleCollector {
  private messages: CapturedMessage[] = [];
  private totalCount = 0;
  private countAtLastSample = 0;
  private windowMs: number;
  private isActive = false;

  private originalConsole: {
    log: typeof console.log;
    warn: typeof console.warn;
    error: typeof console.error;
    info: typeof console.info;
  };

  constructor(windowMs = CONSOLE_LOG_WINDOW_MS) {
    this.windowMs = windowMs;
    this.originalConsole = {
      log: console.log.bind(console),
      warn: console.warn.bind(console),
      error: console.error.bind(console),
      info: console.info.bind(console),
    };
  }

  /**
   * Format console arguments into a single string.
   * Handles primitives, objects, arrays, and errors.
   */
  private formatMessage(args: any[]): string {
    const parts: string[] = [];

    for (const arg of args) {
      if (arg == null) {
        parts.push(String(arg));
      } else if (typeof arg === "string") {
        parts.push(arg);
      } else if (arg instanceof Error) {
        parts.push(arg.message);
      } else if (typeof arg === "object") {
        try {
          const json = JSON.stringify(arg);
          const maxObjectLength = 100;
          parts.push(
            json.length > maxObjectLength
              ? json.slice(0, maxObjectLength) + "..."
              : json,
          );
        } catch {
          parts.push("[Object]");
        }
      } else {
        parts.push(String(arg));
      }
    }

    const message = parts.join(" ");
    return message.length > MAX_MESSAGE_LENGTH
      ? message.slice(0, MAX_MESSAGE_LENGTH) + "..."
      : message;
  }

  /**
   * Capture a console message with timestamp and optional stack trace.
   */
  private captureMessage(level: "log" | "warn" | "error" | "info", args: any[]): void {
    if (!this.isActive) return;

    try {
      const timestamp = performance.now();
      const message = this.formatMessage(args);

      let stack: string | undefined;
      if (level === "error") {
        const error = new Error();
        const lines = error.stack?.split("\n") ?? [];
        stack = lines.slice(3).join("\n");
      }

      this.messages.push({
        level,
        message,
        timestamp,
        stack,
      });

      this.totalCount++;

      if (this.messages.length > MAX_STORED_MESSAGES) {
        this.messages = this.messages.slice(-MAX_STORED_MESSAGES);
      }
    } catch (error) {
      // Silently fail to avoid breaking console
    }
  }

  start(): void {
    if (this.isActive) return;
    this.isActive = true;

    this.totalCount = 0;
    this.countAtLastSample = 0;
    this.messages = [];

    const self = this;

    console.log = function (...args: any[]) {
      self.originalConsole.log(...args);
      self.captureMessage("log", args);
    };

    console.warn = function (...args: any[]) {
      self.originalConsole.warn(...args);
      self.captureMessage("warn", args);
    };

    console.error = function (...args: any[]) {
      self.originalConsole.error(...args);
      self.captureMessage("error", args);
    };

    console.info = function (...args: any[]) {
      self.originalConsole.info(...args);
      self.captureMessage("info", args);
    };
  }

  stop(): void {
    if (!this.isActive) return;
    this.isActive = false;

    console.log = this.originalConsole.log;
    console.warn = this.originalConsole.warn;
    console.error = this.originalConsole.error;
    console.info = this.originalConsole.info;
  }

  reset(): void {
    this.totalCount = 0;
    this.countAtLastSample = 0;
    this.messages = [];
  }

  getCountSinceLastSample(): number {
    const count = this.totalCount - this.countAtLastSample;
    this.countAtLastSample = this.totalCount;
    return count;
  }

  getTotalCount(): number {
    return this.totalCount;
  }

  /**
   * Get recent console log entries sorted by timestamp (descending).
   * Performs automatic cleanup of messages outside the tracking window.
   */
  getTopLogs(): { data: ConsoleLogEntry[]; total: number; truncated: boolean } {
    const now = performance.now();
    const cutoff = now - this.windowMs;

    this.messages = this.messages.filter((m) => m.timestamp >= cutoff);

    const data = this.messages
      .slice()
      .reverse()
      .map((msg) => ({
        level: msg.level,
        message: msg.message,
        timestamp: msg.timestamp,
        age: now - msg.timestamp,
        stack: msg.stack,
      }));

    return {
      data,
      total: data.length,
      truncated: false,
    };
  }
}
