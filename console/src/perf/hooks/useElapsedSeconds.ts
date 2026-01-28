// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useMemo } from "react";
import { useSelector, useStore } from "react-redux";

import { useTick } from "@/perf/hooks/useTick";
import { type HarnessStatus, SLICE_NAME } from "@/perf/slice";
import { type RootState } from "@/store";

const selectStatus = (state: RootState): HarnessStatus => state[SLICE_NAME].status;
const selectEndTime = (state: RootState): number | null => state[SLICE_NAME].endTime;

/**
 * Hook that returns the elapsed profiling time in seconds.
 *
 * Uses the shared tick context to ensure synchronized updates with other
 * timer-dependent components (e.g., macro panel timer).
 *
 * When paused, the elapsed time freezes at the pause moment.
 * When running, the elapsed time updates every tick (1 second).
 */
export const useElapsedSeconds = (): number => {
  const store = useStore<RootState>();
  const status = useSelector(selectStatus);
  const endTime = useSelector(selectEndTime);
  const tick = useTick();

  return useMemo(() => {
    // Subscribe to tick only when running (for recalculation)
    void (status === "running" && tick);

    const startTime = store.getState()[SLICE_NAME].startTime;
    if (startTime == null) return 0;
    const end = endTime ?? performance.now();
    return (end - startTime) / 1000;
  }, [store, status, endTime, tick]);
};
