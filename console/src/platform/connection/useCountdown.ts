// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useNow } from "@synnaxlabs/lyra/hooks";
import { TimeSpan, type TimeStamp } from "@synnaxlabs/x";

const secondsUntil = (at: TimeStamp, now: TimeStamp): number =>
  Math.max(0, Math.ceil(new TimeSpan(at.valueOf() - now.valueOf()).seconds));

/**
 * Whole seconds remaining until `at`, clamped at zero. Re-renders the caller each time
 * the count drops.
 */
export const useCountdown = (at: TimeStamp): number => {
  const now = useNow((now) => {
    const remaining = secondsUntil(at, now);
    return remaining === 0 ? null : at.sub(TimeSpan.seconds(remaining - 1));
  });
  return secondsUntil(at, now);
};
