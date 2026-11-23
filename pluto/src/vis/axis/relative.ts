// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

const NS = { us: 1e3, ms: 1e6, s: 1e9, min: 6e10, h: 3.6e12, d: 8.64e13 };

const SUB_SECOND_STEPS = [1, 2, 5].flatMap((m) =>
  [1, 10, 100, 1e3, 1e4, 1e5, 1e6, 1e7, 1e8].map((p) => m * p),
);

/** Tick steps, in nanoseconds, that land on round times from a zero. */
export const RELATIVE_STEPS: number[] = [
  ...SUB_SECOND_STEPS.sort((a, b) => a - b),
  ...[1, 2, 5, 10, 15, 30].map((s) => s * NS.s),
  ...[1, 2, 5, 10, 15, 30].map((m) => m * NS.min),
  ...[1, 2, 3, 6, 12].map((h) => h * NS.h),
  ...[1, 2, 5, 10, 20, 50, 100].map((d) => d * NS.d),
];

// digits returns the fraction digits that show a multiple of step in unit.
const digits = (step: number, unit: number): number =>
  step >= unit ? 0 : Math.min(9, Math.ceil(Math.log10(unit / step) - 1e-9));

// UNITS are the units of a span under a minute, each for spans below its limit.
const UNITS = [
  { below: NS.us, size: 1, label: "ns" },
  { below: NS.ms, size: NS.us, label: "μs" },
  { below: NS.s, size: NS.ms, label: "ms" },
  { below: NS.min, size: NS.s, label: "s" },
];

const clock = (abs: number, step: number, hours: boolean): string => {
  const d = digits(step, NS.s);
  const q = NS.s / 10 ** d;
  const ticks = Math.round(abs / q);
  const perMinute = 60 * 10 ** d;
  const h = Math.floor(ticks / (60 * perMinute));
  const m = Math.floor(ticks / perMinute) % 60;
  const s = ((ticks % perMinute) / 10 ** d).toFixed(d).padStart(d > 0 ? d + 3 : 2, "0");
  if (hours) return `${h}:${m.toString().padStart(2, "0")}:${s}`;
  return `${m + h * 60}:${s}`;
};

/**
 * @param value - A span from a zero, in nanoseconds.
 * @param step - The precision to show, in nanoseconds.
 * @param magnitude - The largest span shown with value, which selects the unit, so
 * that every label on an axis shares one. Defaults to the size of value.
 * @returns value as a time from a zero: "T0", "T+12.5 s", "T-250 ms", "T+1:30", or
 * "T+1:02:30".
 */
export const formatRelativeTime = (
  value: number,
  step: number,
  magnitude: number = Math.abs(value),
): string => {
  const abs = Math.abs(value);
  if (abs < step / 2) return "T0";
  const sign = value < 0 ? "T-" : "T+";
  if (magnitude >= NS.min) return `${sign}${clock(abs, step, magnitude >= NS.h)}`;
  const { size, label } = UNITS.find((u) => magnitude < u.below) ?? UNITS[3];
  return `${sign}${(abs / size).toFixed(digits(step, size))} ${label}`;
};
