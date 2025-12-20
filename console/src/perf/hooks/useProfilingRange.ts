// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type label, type Synnax as SynnaxClient, TimeStamp } from "@synnaxlabs/client";
import { Synnax } from "@synnaxlabs/pluto";
import { math, TimeRange } from "@synnaxlabs/x";
import { useCallback, useEffect, useRef } from "react";
import { useDispatch } from "react-redux";

import { useSelectRangeKey, useSelectRangeStartTime } from "@/perf/selectors";
import * as Perf from "@/perf/slice";
import { type HarnessStatus } from "@/perf/slice";

// This is where we want to brainstorm


/**
 * Retrieves an existing label by name, or creates it if it doesn't exist.
 *
 * This is a console-specific helper (not in x/) because it requires a Synnax client.
 * The pattern differs from the Label modal which uses client.labels.create() directly -
 * we need check-by-name to reuse existing system labels for profiling sessions.
 */
const getOrCreateLabel = async (
  client: SynnaxClient,
  name: string,
  color: string,
): Promise<label.Label> => {
  const existing = await client.labels.retrieve({ names: [name] });
  if (existing.length > 0) return existing[0];
  try {
    return await client.labels.create({ name, color });
  } catch {
    // Handle race condition: another call may have created it between retrieve and create
    const retryExisting = await client.labels.retrieve({ names: [name] });
    if (retryExisting.length > 0) return retryExisting[0];
    throw new Error(`Failed to get or create label: ${name}`);
  }
};

export interface RangeMetadata {
  avgFps: number | null;
  avgCpu: number | null;
  avgGpu: number | null;
  avgHeap: number | null;
}

// Temporary value. Don't know what strategy I want to use yet.
const METADATA_WRITE_INTERVAL_MS = 5_000;

export interface UseProfilingRangeOptions {
  status: HarnessStatus;
  getMetadata: () => RangeMetadata | null;
}

export type Verdict = "Passed" | "Failed";

export interface FinalizeRangeInput {
  rangeKey: string;
  startTime: number;
  aggregates: RangeMetadata;
}

export interface UseProfilingRangeResult {
  rangeKey: string | null;
  rangeStartTime: number | null;
  createRange: () => void;
  updateEndTime: (endTime: TimeStamp) => void;
  finalizeRange: (input: FinalizeRangeInput) => void;
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
        name: "Console Profiling",
        timeRange: new TimeRange(now, TimeStamp.MAX).numeric,
      })
      .then(async (range) => {
        dispatch(Perf.setRangeKey(range.key));
        dispatch(Perf.setRangeStartTime(Number(now.valueOf())));

        const [nominalLabel] = await Promise.all([
          getOrCreateLabel(client, "Nominal", "#3B82F6"), // Blue
          getOrCreateLabel(client, "Passed", "#16A34A"), // Green
          getOrCreateLabel(client, "Failed", "#DC2626"), // Red
        ]);

        await range.addLabel(nominalLabel.key);
      })
      .catch((error: Error) => {
        console.error("Failed to create profiling range:", error);
      });
  }, [client, dispatch]);

  const updateEndTime = useCallback(
    (endTime: TimeStamp) => {
      if (client == null || rangeKey == null || rangeStartTime == null) return;

      client.ranges
        .create({
          key: rangeKey,
          name: "Console Profiling",
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
        const kv = range.kv;

        const writes: Promise<void>[] = [];
        if (metadata.avgFps != null)
          writes.push(kv.set("avgFps", String(math.roundTo(metadata.avgFps, 1))));
        if (metadata.avgCpu != null)
          writes.push(kv.set("avgCpu", String(math.roundTo(metadata.avgCpu, 1))));
        if (metadata.avgGpu != null)
          writes.push(kv.set("avgGpu", String(math.roundTo(metadata.avgGpu, 1))));
        if (metadata.avgHeap != null)
          writes.push(kv.set("avgHeap", String(math.roundTo(metadata.avgHeap, 1))));

        await Promise.all(writes);
      } catch (error) {
        console.error("Failed to write range metadata:", error);
      }
    };

    void writeMetadata();
    const intervalId = setInterval(() => void writeMetadata(), METADATA_WRITE_INTERVAL_MS);

    return () => clearInterval(intervalId);
  }, [status, client, rangeKey]);

  /**
   * Stub analyzer that "analyzes" the aggregates and returns a verdict.
   * For now, always returns PASS.
   */
  const analyzeSession = useCallback((aggregates: RangeMetadata): Verdict => {
    console.log("[useProfilingRange] Analyzing session:", aggregates);
    // TODO: Implement actual analysis logic
    return "Passed";
  }, []);

  /**
   * Finalizes the profiling range by:
   * 1. Updating end time
   * 2. Running analysis
   * 3. Adding verdict label (keeps "nominal" label)
   */
  const finalizeRange = useCallback(
    (input: FinalizeRangeInput) => {
      const { rangeKey: key, startTime, aggregates } = input;
      console.log("[useProfilingRange] finalizeRange called", {
        hasClient: client != null,
        rangeKey: key,
        startTime,
      });
      if (client == null) return;

      const finalize = async () => {
        const endTime = TimeStamp.now();

        // Update the range end time (no update() available)
        await client.ranges.create({
          key,
          name: "Console Profiling",
          timeRange: new TimeRange(new TimeStamp(startTime), endTime).numeric,
        });

        // Retrieve the range and add verdict label
        console.log("[useProfilingRange] Retrieving range for verdict label");
        const range = await client.ranges.retrieve(key);
        const verdict = analyzeSession(aggregates);

        console.log("[useProfilingRange] Getting/creating verdict label:", verdict);
        const verdictLabel = await getOrCreateLabel(
          client,
          verdict,
          verdict === "Passed" ? "#16A34A" : "#DC2626",
        );
        console.log("[useProfilingRange] Adding label to range:", verdictLabel);
        await range.addLabel(verdictLabel.key);
        console.log("[useProfilingRange] Label added successfully");
      };

      finalize().catch((error: Error) => {
        console.error("Failed to finalize profiling range:", error);
      });
    },
    [client, analyzeSession],
  );

  return {
    rangeKey,
    rangeStartTime,
    createRange,
    updateEndTime,
    finalizeRange,
  };
};
