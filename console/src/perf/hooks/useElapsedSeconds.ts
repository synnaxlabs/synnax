// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useEffect, useState } from "react";
import { useSelector, useStore } from "react-redux";

import { type HarnessStatus, SLICE_NAME } from "@/perf/slice";
import { type RootState } from "@/store";

const selectStatus = (state: RootState): HarnessStatus => state[SLICE_NAME].status;
const selectEndTime = (state: RootState): number | null => state[SLICE_NAME].endTime;

/**
 * Hook that returns the elapsed profiling time in seconds.
 *
 * This hook intentionally bypasses useMemoSelect because elapsed seconds depends on
 * performance.now(), which isn't tracked by proxy-memoize. Instead, it uses a dedicated
 * interval to update the value every second while profiling is running.
 *
 * When paused, the elapsed time freezes at the pause moment.
 * When running, the elapsed time updates every second.
 *
 * Note: startTime is read directly from the store rather than subscribed to because
 * it only changes when status changes, making a separate subscription redundant.
 */
export const useElapsedSeconds = (): number => {
  const store = useStore<RootState>();
  const status = useSelector(selectStatus);
  const endTime = useSelector(selectEndTime);

  const [elapsedSeconds, setElapsedSeconds] = useState(0);

  useEffect(() => {
    const calcElapsed = () => {
      const startTime = store.getState()[SLICE_NAME].startTime;
      if (startTime == null) return 0;
      const end = endTime ?? performance.now();
      return (end - startTime) / 1000;
    };

    setElapsedSeconds(calcElapsed());

    if (status !== "running") return;

    const interval = setInterval(() => {
      setElapsedSeconds(calcElapsed());
    }, 1000);

    return () => clearInterval(interval);
  }, [store, status, endTime]);

  return elapsedSeconds;
};
