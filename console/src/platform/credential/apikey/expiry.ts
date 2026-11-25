// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan, type TimeStamp } from "@synnaxlabs/client";

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
 * undefined for "never".
 */
export const resolve = (
  expiry: Expiry,
  custom: TimeStamp,
  now: TimeStamp,
): TimeStamp | undefined => {
  if (expiry === "never") return undefined;
  if (expiry === "custom") return custom;
  return now.add(LIFETIMES[expiry]);
};

const RELATIVE = new Intl.RelativeTimeFormat("en", { numeric: "always" });

/** @returns how long until expires, such as "in 12 days". */
export const relative = (expires: TimeStamp, now: TimeStamp): string => {
  const left = now.span(expires);
  if (left.greaterThanOrEqual(TimeSpan.DAY))
    return RELATIVE.format(Math.round(left.days), "day");
  if (left.greaterThanOrEqual(TimeSpan.HOUR))
    return RELATIVE.format(Math.round(left.hours), "hour");
  return RELATIVE.format(Math.max(1, Math.round(left.minutes)), "minute");
};

/**
 * @returns when the label of a key that expires at expires next changes, or null when
 * the key never expires or has expired.
 */
export const nextChange = (
  expires: TimeStamp | undefined,
  now: TimeStamp,
): TimeStamp | null => {
  if (expires == null || !expires.after(now)) return null;
  const step = now.span(expires).lessThan(TimeSpan.HOUR)
    ? TimeSpan.MINUTE
    : TimeSpan.HOUR;
  const next = now.add(step);
  return next.before(expires) ? next : expires;
};
