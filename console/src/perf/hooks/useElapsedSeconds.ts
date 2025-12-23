// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useEffect, useState } from "react";
import { useSelector } from "react-redux";

import { selectSlice } from "@/perf/selectors";

/**
 * Hook that returns the elapsed profiling time in seconds.
 *
 * This hook intentionally bypasses useMemoSelect because elapsed seconds depends on
 * performance.now(), which isn't tracked by proxy-memoize. Instead, it uses a dedicated
 * interval to update the value every second while profiling is running.
 *
 * When paused, the elapsed time freezes at the pause moment.
 * When running, the elapsed time updates every second.
 */
export const useElapsedSeconds = (): number => {
  const { status, startTime, endTime } = useSelector(selectSlice);

  const [elapsedSeconds, setElapsedSeconds] = useState(0);

  useEffect(() => {
    const calcElapsed = () => {
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
  }, [status, startTime, endTime]);

  return elapsedSeconds;
};
