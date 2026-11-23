// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Telem } from "@synnaxlabs/lyra/telem";
import { TimeSpan, type TimeStamp } from "@synnaxlabs/x";

const pad = (n: number): string => String(n).padStart(2, "0");

const seconds = (from: TimeStamp, to: TimeStamp): number =>
  Math.max(0, Math.floor((to.nanoseconds - from.nanoseconds) / 1e9));

/** "32s", "4m 31s", "6h 12m", "23d 6h". */
export const formatSpan = (from: TimeStamp, to: TimeStamp): string => {
  const s = seconds(from, to);
  if (s < 60) return `${s}s`;
  const m = Math.floor(s / 60);
  if (m < 60) return `${m}m ${pad(s % 60)}s`;
  const h = Math.floor(m / 60);
  if (h < 24) return `${h}h ${pad(m % 60)}m`;
  return `${Math.floor(h / 24)}d ${h % 24}h`;
};

/** The time of day for a start today, and the date for an earlier one. */
export const formatStart = (at: TimeStamp, now: TimeStamp): string => {
  if (Telem.Text.describeDay(at, now) === "Today")
    return Telem.Text.formatTime(at, TimeSpan.MINUTE);
  return at.date().toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
    year: at.localYear === now.localYear ? undefined : "numeric",
  });
};

/** "Just now", "12m ago", "3h ago", "2d ago", then the date. */
export const formatAgo = (end: TimeStamp, now: TimeStamp): string => {
  const m = Math.floor(seconds(end, now) / 60);
  if (m < 1) return "Just now";
  if (m < 60) return `${m}m ago`;
  const h = Math.floor(m / 60);
  if (h < 24) return `${h}h ago`;
  const d = Math.floor(h / 24);
  if (d < 7) return `${d}d ago`;
  return formatStart(end, now);
};

export const nextSecond = (now: TimeStamp): TimeStamp => now.add(TimeSpan.SECOND);

export const nextMinute = (now: TimeStamp): TimeStamp => now.add(TimeSpan.MINUTE);
