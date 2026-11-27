// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan, TimeStamp } from "@synnaxlabs/client";

const LIFETIMES = {
  "30d": TimeSpan.days(30),
  "90d": TimeSpan.days(90),
  "1y": TimeSpan.days(365),
} as const;

/** When a new API key expires, as the user picks it. */
export type Expiry = keyof typeof LIFETIMES | "custom" | "never";

/** The expiry a new API key gets unless the user picks another. */
export const DEFAULT_EXPIRY: Expiry = "90d";

/**
 * @returns when a key created at now expires: the custom time for "custom", and
 * TimeStamp.MAX for "never".
 */
export const resolve = (
  expiry: Expiry,
  custom: TimeStamp,
  now: TimeStamp,
): TimeStamp => {
  if (expiry === "never") return TimeStamp.MAX;
  if (expiry === "custom") return custom;
  return now.add(LIFETIMES[expiry]);
};

const RELATIVE = new Intl.RelativeTimeFormat("en", { numeric: "always" });

const UNITS = [
  { span: TimeSpan.DAY, name: "day" },
  { span: TimeSpan.HOUR, name: "hour" },
  { span: TimeSpan.MINUTE, name: "minute" },
] as const;

// Rounds up, so the label changes a whole unit before the expiry.
const count = (left: TimeSpan): { unit: (typeof UNITS)[number]; n: number } => {
  const unit = UNITS.find((u) => left.greaterThan(u.span)) ?? UNITS[2];
  return { unit, n: Math.ceil(Number(left.valueOf()) / Number(unit.span.valueOf())) };
};

/** @returns how long until expires, such as "in 12 days". */
export const relative = (expires: TimeStamp, now: TimeStamp): string => {
  const { unit, n } = count(now.span(expires));
  return RELATIVE.format(n, unit.name);
};

/**
 * @returns when the label of a key that expires at expires next changes, or null when
 * the key never expires or has expired.
 */
export const nextChange = (expires: TimeStamp, now: TimeStamp): TimeStamp | null => {
  if (expires.equals(TimeStamp.MAX) || !expires.after(now)) return null;
  const { unit, n } = count(now.span(expires));
  return expires.sub(unit.span.mult(n - 1));
};
