// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan, type TimeStamp } from "@synnaxlabs/x";
import { useLayoutEffect, useState } from "react";

import { useClock } from "@/hooks/clock";

// setTimeout overflows past this delay and fires at once.
const MAX_DELAY_MS = 2 ** 31 - 1;

/**
 * @returns the current time of {@link useClock}, held as state. The caller re-renders
 * at the instant `next(now)` returns, or never when it returns null. `next` must be
 * pure and return an instant after `now`. When that instant has already passed, such
 * as when an input to `next` changes under an old time, the time catches up before the
 * browser paints.
 *
 * @example const now = useNow((now) => now.add(TimeSpan.SECOND));
 */
export const useNow = (next: (now: TimeStamp) => TimeStamp | null): TimeStamp => {
  const clock = useClock();
  const [now, setNow] = useState(clock);
  const instant = next(now)?.valueOf() ?? null;
  useLayoutEffect(() => {
    if (instant == null) return;
    const current = clock();
    if (instant <= current.valueOf()) {
      setNow(current);
      return;
    }
    const remaining = new TimeSpan(instant - current.valueOf());
    const delay = Math.min(Math.ceil(remaining.milliseconds), MAX_DELAY_MS);
    const t = setTimeout(() => setNow(clock()), delay);
    return () => clearTimeout(t);
  }, [clock, now, instant]);
  return now;
};
