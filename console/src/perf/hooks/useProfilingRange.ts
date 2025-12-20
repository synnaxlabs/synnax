// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeStamp } from "@synnaxlabs/client";
import { Synnax } from "@synnaxlabs/pluto";
import { TimeRange } from "@synnaxlabs/x";
import { useCallback, useEffect, useRef } from "react";
import { useDispatch } from "react-redux";

import { useSelectRangeKey, useSelectRangeStartTime } from "@/perf/selectors";
import * as Perf from "@/perf/slice";
import { type HarnessStatus } from "@/perf/slice";

export interface RangeMetadata {
  avgFps: number | null;
  avgCpu: number | null;
  avgGpu: number | null;
  avgHeap: number | null;
}

// Temporary value. Don't know what strategy I want to use yet.
const METADATA_WRITE_INTERVAL_MS = 10_000;

export interface UseProfilingRangeOptions {
  status: HarnessStatus;
  getMetadata: () => RangeMetadata | null;
}

export interface UseProfilingRangeResult {
  rangeKey: string | null;
  createRange: () => void;
  updateEndTime: (endTime: TimeStamp) => void;
}

/**
 * Hook that manages Synnax range lifecycle for profiling sessions.
 *
 * Handles:
 * - Range creation when profiling starts
 * - End time updates on pause/resume/stop
 * - Periodic metadata writes (every 10s while running)
 */
export const useProfilingRange = ({
  status,
  getMetadata,
}: UseProfilingRangeOptions): UseProfilingRangeResult => {
  const dispatch = useDispatch();
  const client = Synnax.use();
  const rangeKey = useSelectRangeKey();
  const rangeStartTime = useSelectRangeStartTime();

  const getMetadataRef = useRef(getMetadata);
  getMetadataRef.current = getMetadata;

  const createRange = useCallback(() => {
    if (client == null) return;

    const now = TimeStamp.now();
    client.ranges
      .create({
        name: `Console Profiling - ${now.toLocaleString()}`,
        timeRange: new TimeRange(now, TimeStamp.MAX).numeric,
      })
      .then((range) => {
        dispatch(Perf.setRangeKey(range.key));
        dispatch(Perf.setRangeStartTime(Number(now.valueOf())));
      })
      .catch((error: Error) => {
        console.error("Failed to create profiling range:", error);
      });
  }, [client, dispatch]);

  const updateEndTime = useCallback(
    (endTime: TimeStamp) => {
      if (client == null || rangeKey == null || rangeStartTime == null) return;

      const rangeName = `Console Profiling - ${new TimeStamp(rangeStartTime).toLocaleString()}`;
      client.ranges
        .create({
          key: rangeKey,
          name: rangeName,
          timeRange: new TimeRange(new TimeStamp(rangeStartTime), endTime).numeric,
        })
        .catch((error: Error) => {
          console.error("Failed to update profiling range:", error);
        });
    },
    [client, rangeKey, rangeStartTime],
  );

  useEffect(() => {
    if (status !== "running" || client == null || rangeKey == null) return;

    const writeMetadata = async () => {
      const metadata = getMetadataRef.current();
      if (metadata == null) return;

      try {
        const range = await client.ranges.retrieve(rangeKey);
        await range.kv.set(
          "profiling_metrics",
          JSON.stringify({
            ...metadata,
            updatedAt: Date.now(),
          }),
        );
      } catch (error) {
        console.error("Failed to write range metadata:", error);
      }
    };

    void writeMetadata();
    const intervalId = setInterval(() => void writeMetadata(), METADATA_WRITE_INTERVAL_MS);

    return () => clearInterval(intervalId);
  }, [status, client, rangeKey]);

  return {
    rangeKey,
    createRange,
    updateEndTime,
  };
};
