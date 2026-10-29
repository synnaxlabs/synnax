// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// A user is active on a UTC day when one of their Desktop machines logs in or renews
// that day. The app renews at launch and every six hours while it runs.

import { type EventKind } from "@/server/db/schema";

export const DAY_MS = 24 * 60 * 60 * 1000;

/** DESKTOP are the events that count as Desktop use. */
export const DESKTOP: EventKind[] = ["link", "renew"];

/** Usage is how one user has used Synnax Desktop. */
export interface Usage {
  /** lastLogin is the last Desktop login, or null for none. */
  lastLogin: Date | null;
  /** lastActive is the last login or renewal from any of the user's machines. */
  lastActive: Date | null;
  /** days counts the active days among the last {@link MONTH} UTC days. */
  days: number;
}

export const NONE: Usage = { lastLogin: null, lastActive: null, days: 0 };

/** Period is how many UTC days, ending with today, a count of active users spans. */
export type Period = "day" | "week" | "month";

export const PERIODS: Period[] = ["day", "week", "month"];

export const MONTH = 30;

const PERIOD_DAYS: Record<Period, number> = { day: 1, week: 7, month: MONTH };

/** dayOf returns the start of the UTC day that holds `t`. */
export const dayOf = (t: Date): Date =>
  new Date(Math.floor(t.getTime() / DAY_MS) * DAY_MS);

/** since returns the start of the period that ends with the UTC day of `now`. */
export const since = (period: Period, now: Date): Date =>
  new Date(dayOf(now).getTime() - (PERIOD_DAYS[period] - 1) * DAY_MS);

/**
 * isActive says whether the user was active in the period that ends with the UTC day
 * of `now`. For the period `day`, it says whether the user is a daily active user.
 */
export const isActive = ({ lastActive }: Usage, period: Period, now: Date): boolean =>
  lastActive != null && lastActive >= since(period, now);

/**
 * startsDay says whether Desktop use at `now` is the user's first of its UTC day, given
 * their `previous` use, or null for none.
 */
export const startsDay = (previous: Date | null, now: Date): boolean =>
  previous == null || previous < dayOf(now);
