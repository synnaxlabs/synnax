// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { useEffect, useState } from "react";

// Twice a second, so a whole-second readout never sticks on a stale value.
const TICK = TimeSpan.milliseconds(500);

/**
 * Whole seconds remaining until `at`, clamped at zero. Re-renders the caller
 * twice a second for as long as it is mounted.
 */
export const useCountdown = (at: TimeStamp): number => {
  const [now, setNow] = useState(() => TimeStamp.now());
  useEffect(() => {
    const interval = setInterval(() => setNow(TimeStamp.now()), TICK.milliseconds);
    return () => clearInterval(interval);
  }, []);
  return Math.max(0, Math.ceil(new TimeSpan(at.valueOf() - now.valueOf()).seconds));
};
