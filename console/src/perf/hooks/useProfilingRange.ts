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
import { math, runtime, TimeRange } from "@synnaxlabs/x";
import { useCallback, useEffect, useRef } from "react";
import { useDispatch } from "react-redux";

import { useSelect as useSelectCluster } from "@/cluster/selectors";
import { type Severity } from "@/perf/analyzer/types";
import {
  getMetricLabelName,
  getProfilingLabelConfigs,
  LABEL_COLORS,
  METRIC_ORDER,
  type MetricType,
  NOMINAL_LABEL_NAME,
} from "@/perf/constants";
import { useSelectRangeKey, useSelectRangeStartTime } from "@/perf/selectors";
import * as Perf from "@/perf/slice";
import { type HarnessStatus } from "@/perf/slice";
import { useSelectVersion } from "@/version/selectors";

/**
 * Retrieves an existing label by name, or creates it if it doesn't exist.
 * Handles race conditions where another call may create the label concurrently.
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
    const retryExisting = await client.labels.retrieve({ names: [name] });
    if (retryExisting.length > 0) return retryExisting[0];
    throw new Error(`Failed to get or create label: ${name}`);
  }
};

const ensureProfilingLabelsExist = async (client: SynnaxClient): Promise<void> => {
  const configs = getProfilingLabelConfigs();
  await Promise.all(
    configs.map(({ name, color }) => getOrCreateLabel(client, name, color)),
  );
};

interface RangeWithLabels {
  addLabel: (key: string) => Promise<void>;
  removeLabel: (key: string) => Promise<void>;
}

/**
 * Removes the nominal label from a range if it hasn't been removed yet.
 * Updates the ref to track removal state.
 */
const removeNominalLabelIfNeeded = async (
  client: SynnaxClient,
  range: RangeWithLabels,
  nominalRemovedRef: { current: boolean },
): Promise<void> => {
  if (nominalRemovedRef.current) return;
  const nominalLabels = await client.labels.retrieve({ names: [NOMINAL_LABEL_NAME] });
  if (nominalLabels.length > 0) {
    await range.removeLabel(nominalLabels[0].key);
    nominalRemovedRef.current = true;
  }
};

/**
 * Removes a warning label when upgrading to error (error supersedes warning).
 */
const removeSupersededWarningLabel = async (
  client: SynnaxClient,
  range: RangeWithLabels,
  metric: MetricType,
  currentSeverity: "warning" | "error" | undefined,
): Promise<void> => {
  if (currentSeverity !== "warning") return;
  const warnLabelName = getMetricLabelName(metric, "warning");
  const warnLabels = await client.labels.retrieve({ names: [warnLabelName] });
  if (warnLabels.length > 0) await range.removeLabel(warnLabels[0].key);
};

export interface AverageMetrics {
  cpu: number | null;
  fps: number | null;
  gpu: number | null;
}

export interface PeakMetrics {
  cpu: number | null;
  fps: number | null;
  gpu: number | null;
  heap: number | null;
}

/**
 * Live metric values at a point in time.
 * Used for start/end snapshots.
 */
export interface LiveValues {
  fps: number | null;
  cpu: number | null;
  gpu: number | null;
  heap: number | null;
}

/**
 * Structured metrics for range metadata.
 * - averages: Running averages during the session
 * - peaks: Worst-case values (max for cpu/gpu/heap, min for fps)
 */
export interface RangeMetrics {
  averages: AverageMetrics;
  peaks: PeakMetrics;
}

// Temporary value. Don't know what strategy I want to use yet.
const METADATA_WRITE_INTERVAL_MS = 5_000;

const roundLiveValues = (values: LiveValues): LiveValues => ({
  fps: values.fps != null ? math.roundTo(values.fps, 1) : null,
  cpu: values.cpu != null ? math.roundTo(values.cpu, 1) : null,
  gpu: values.gpu != null ? math.roundTo(values.gpu, 1) : null,
  heap: values.heap != null ? math.roundTo(values.heap, 1) : null,
});

export interface UseProfilingRangeOptions {
  status: HarnessStatus;
  getMetrics: () => RangeMetrics | null;
}

interface SeverityPair {
  peak: Severity;
  avg: Severity;
}

export interface AnalysisSeverities {
  fps: SeverityPair;
  cpu: SeverityPair;
  gpu: SeverityPair;
  heap: Severity;
}

export interface FinalizeRangeInput {
  rangeKey: string;
  startTime: number;
  severities: AnalysisSeverities;
  stopValues: LiveValues;
}

export interface AddMetricLabelInput {
  metric: MetricType;
  severity: "warning" | "error";
  latched: boolean;
}

export interface RemoveTransientLabelInput {
  metric: MetricType;
}

interface LabelState {
  severity: "warning" | "error";
  latched: boolean;
}

export interface CreateRangeInput {
  startValues?: LiveValues;
}

export interface UseProfilingRangeResult {
  rangeKey: string | null;
  rangeStartTime: number | null;
  createRange: (input?: CreateRangeInput) => void;
  updateEndTime: (endTime: TimeStamp) => void;
  clearStopValues: () => void;
  finalizeRange: (input: FinalizeRangeInput) => void;
  addMetricLabel: (input: AddMetricLabelInput) => void;
  removeTransientLabel: (input: RemoveTransientLabelInput) => void;
  isMetricLatched: (metric: MetricType) => boolean;
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
  getMetrics,
}: UseProfilingRangeOptions): UseProfilingRangeResult => {
  const dispatch = useDispatch();
  const client = Synnax.use();
  const rangeKey = useSelectRangeKey();
  const rangeStartTime = useSelectRangeStartTime();
  const activeCluster = useSelectCluster();
  const username = activeCluster?.username ?? "Unknown";
  const version = useSelectVersion();

  const getMetricsRef = useRef(getMetrics);
  getMetricsRef.current = getMetrics;

  // AbortController for cancelling pending async operations on unmount.
  // Prevents state updates and API calls after the component is destroyed.
  const abortRef = useRef<AbortController | null>(null);

  useEffect(() => {
    abortRef.current = new AbortController();
    return () => abortRef.current?.abort();
  }, []);

  const rangeNameRef = useRef<string | null>(null);

  const labelStatesRef = useRef<Map<MetricType, LabelState>>(new Map());
  const nominalRemovedRef = useRef(false);

  const getRangeName = useCallback(
    (hostname: string): string => `Console Profiling (${username}@${hostname})`,
    [username],
  );

  const createRange = useCallback(
    (input?: CreateRangeInput) => {
      if (client == null) return;

      labelStatesRef.current.clear();
      nominalRemovedRef.current = false;

      const create = async () => {
        const [osInfo, nominalLabel] = await Promise.all([
          runtime.getOSInfoAsync(),
          ensureProfilingLabelsExist(client).then(() =>
            getOrCreateLabel(client, NOMINAL_LABEL_NAME, LABEL_COLORS.nominal),
          ),
        ]);

        const rangeName = getRangeName(osInfo.hostname);
        rangeNameRef.current = rangeName;

        const now = TimeStamp.now();
        const range = await client.ranges.create({
          name: rangeName,
          timeRange: new TimeRange(now, TimeStamp.MAX).numeric,
        });

        dispatch(Perf.setRangeKey(range.key));
        dispatch(Perf.setRangeStartTime(Number(now.valueOf())));

        await range.addLabel(nominalLabel.key);

        const kv = range.kv;
        const kvWrites: Promise<void>[] = [
          kv.set("hostname", osInfo.hostname),
          kv.set("platform", osInfo.platform),
          kv.set("osVersion", osInfo.version),
          kv.set("username", username),
          kv.set("version", version),
        ];

        if (input?.startValues != null)
          kvWrites.push(
            kv.set("startValues", JSON.stringify(roundLiveValues(input.startValues))),
          );

        await Promise.all(kvWrites);
      };

      create().catch((error: Error) => {
        if (abortRef.current?.signal.aborted) return;
        console.error("Failed to create profiling range:", error);
      });
    },
    [client, dispatch, getRangeName, username, version],
  );

  const updateEndTime = useCallback(
    (endTime: TimeStamp) => {
      if (client == null || rangeKey == null || rangeStartTime == null) return;
      if (rangeNameRef.current == null) return;

      client.ranges
        .create({
          key: rangeKey,
          name: rangeNameRef.current,
          timeRange: new TimeRange(new TimeStamp(rangeStartTime), endTime).numeric,
        })
        .catch((error: Error) => {
          if (abortRef.current?.signal.aborted) return;
          console.error("Failed to update profiling range:", error);
        });
    },
    [client, rangeKey, rangeStartTime],
  );

  const clearStopValues = useCallback(() => {
    if (client == null || rangeKey == null) return;

    const clear = async () => {
      const range = await client.ranges.retrieve(rangeKey);
      await range.kv.delete("stopValues");
    };

    clear().catch((error: Error) => {
      if (abortRef.current?.signal.aborted) return;
      console.error("Failed to clear stop values:", error);
    });
  }, [client, rangeKey]);

  const lastWrittenRef = useRef<{ averages: string; peaks: string } | null>(null);

  useEffect(() => {
    if (status !== "running" || client == null || rangeKey == null) return;

    const roundAverages = (values: AverageMetrics): AverageMetrics => ({
      cpu: values.cpu != null ? math.roundTo(values.cpu, 1) : null,
      fps: values.fps != null ? math.roundTo(values.fps, 1) : null,
      gpu: values.gpu != null ? math.roundTo(values.gpu, 1) : null,
    });

    const roundPeaks = (values: PeakMetrics): PeakMetrics => ({
      cpu: values.cpu != null ? math.roundTo(values.cpu, 1) : null,
      fps: values.fps != null ? math.roundTo(values.fps, 1) : null,
      gpu: values.gpu != null ? math.roundTo(values.gpu, 1) : null,
      heap: values.heap != null ? math.roundTo(values.heap, 1) : null,
    });

    const writeMetrics = async () => {
      const metrics = getMetricsRef.current();
      if (metrics == null) return;

      const averagesJson = JSON.stringify(roundAverages(metrics.averages));
      const peaksJson = JSON.stringify(roundPeaks(metrics.peaks));

      // Skip write if nothing changed
      const last = lastWrittenRef.current;
      if (last != null && last.averages === averagesJson && last.peaks === peaksJson)
        return;

      try {
        const range = await client.ranges.retrieve(rangeKey);
        const kv = range.kv;

        await Promise.all([
          kv.set("averages", averagesJson),
          kv.set("peaks", peaksJson),
        ]);

        lastWrittenRef.current = { averages: averagesJson, peaks: peaksJson };
      } catch (error) {
        if (abortRef.current?.signal.aborted) return;
        console.error("Failed to write range metrics:", error);
      }
    };

    void writeMetrics();
    const intervalId = setInterval(
      () => void writeMetrics(),
      METADATA_WRITE_INTERVAL_MS,
    );

    return () => clearInterval(intervalId);
  }, [status, client, rangeKey]);

  /**
   * Finalizes the profiling range by:
   * 1. Updating end time
   * 2. Adding warning/error labels for each metric with issues
   * 3. Removing superseded labels (warning removed when error is added)
   */
  const finalizeRange = useCallback(
    (input: FinalizeRangeInput) => {
      const { rangeKey: key, startTime, severities, stopValues } = input;
      if (client == null) return;

      const finalize = async () => {
        const endTime = TimeStamp.now();
        if (rangeNameRef.current == null) return;

        await client.ranges.create({
          key,
          name: rangeNameRef.current,
          timeRange: new TimeRange(new TimeStamp(startTime), endTime).numeric,
        });

        const range = await client.ranges.retrieve(key);

        await range.kv.set("stopValues", JSON.stringify(roundLiveValues(stopValues)));

        const hasIssues = METRIC_ORDER.some((m) => {
          if (m === "heap") return severities.heap !== "none";
          const ms = severities[m];
          return ms.peak !== "none" || ms.avg !== "none";
        });

        // Remove nominal label if any issues were detected
        if (hasIssues)
          await removeNominalLabelIfNeeded(client, range, nominalRemovedRef);

        for (const metric of METRIC_ORDER) {
          let severity: Severity;
          if (metric === "heap") severity = severities.heap;
          else {
            const ms = severities[metric];
            if (ms.peak === "error" || ms.avg === "error") severity = "error";
            else if (ms.peak === "warning" || ms.avg === "warning")
              severity = "warning";
            else severity = "none";
          }

          if (severity === "none") continue;

          // Label replacement: error supersedes warning
          if (severity === "error") {
            const currentState = labelStatesRef.current.get(metric);
            await removeSupersededWarningLabel(
              client,
              range,
              metric,
              currentState?.severity,
            );
          }

          const labelName = getMetricLabelName(metric, severity);
          const color =
            severity === "error" ? LABEL_COLORS.error : LABEL_COLORS.warning;
          const label = await getOrCreateLabel(client, labelName, color);
          await range.addLabel(label.key);
        }
      };

      finalize().catch((error: Error) => {
        if (abortRef.current?.signal.aborted) return;
        console.error("Failed to finalize profiling range:", error);
      });
    },
    [client],
  );

  /**
   * Adds a warning/error label for a metric in real-time.
   *
   * Latching behavior:
   * - Latched labels (peak-triggered) are permanent and cannot be removed
   * - Transient labels (avg-triggered) can be removed if avg improves
   * - Once latched, any further changes for this metric are ignored
   *
   * Implements label replacement: error supersedes warning.
   */
  const addMetricLabel = useCallback(
    ({ metric, severity, latched }: AddMetricLabelInput) => {
      if (client == null || rangeKey == null) return;

      const currentState = labelStatesRef.current.get(metric);

      // If already latched, no changes allowed
      if (currentState?.latched) return;

      // Skip if same severity already set
      if (currentState?.severity === severity) return;

      // Skip adding warning if error already exists
      if (severity === "warning" && currentState?.severity === "error") return;

      const labelName = getMetricLabelName(metric, severity);
      const previousState = currentState;
      labelStatesRef.current.set(metric, { severity, latched });

      const add = async () => {
        const color = severity === "error" ? LABEL_COLORS.error : LABEL_COLORS.warning;
        const label = await getOrCreateLabel(client, labelName, color);
        const range = await client.ranges.retrieve(rangeKey);

        // Remove nominal label on first warning/error
        await removeNominalLabelIfNeeded(client, range, nominalRemovedRef);

        // Label replacement: error supersedes warning
        if (severity === "error")
          await removeSupersededWarningLabel(
            client,
            range,
            metric,
            previousState?.severity,
          );

        await range.addLabel(label.key);
      };

      add().catch((error: Error) => {
        console.error(`Failed to add label ${labelName}:`, error);
        // Rollback state on failure
        if (previousState != null) labelStatesRef.current.set(metric, previousState);
        else labelStatesRef.current.delete(metric);
      });
    },
    [client, rangeKey],
  );

  /**
   * Removes a transient (avg-triggered) label for a metric.
   * Only removes if the label is not latched (peak-triggered).
   */
  const removeTransientLabel = useCallback(
    ({ metric }: RemoveTransientLabelInput) => {
      if (client == null || rangeKey == null) return;

      const currentState = labelStatesRef.current.get(metric);

      // Only remove if transient (not latched)
      if (currentState == null || currentState.latched) return;

      const labelName = getMetricLabelName(metric, currentState.severity);
      labelStatesRef.current.delete(metric);

      const remove = async () => {
        const range = await client.ranges.retrieve(rangeKey);
        const labels = await client.labels.retrieve({ names: [labelName] });

        if (labels.length > 0) await range.removeLabel(labels[0].key);

        // Restore nominal label if no other issues remain
        if (labelStatesRef.current.size === 0 && nominalRemovedRef.current) {
          const nominalLabel = await getOrCreateLabel(
            client,
            NOMINAL_LABEL_NAME,
            LABEL_COLORS.nominal,
          );
          await range.addLabel(nominalLabel.key);
          nominalRemovedRef.current = false;
        }
      };

      remove().catch((error: Error) => {
        if (abortRef.current?.signal.aborted) return;
        console.error(`Failed to remove label ${labelName}:`, error);
        // Rollback state on failure
        labelStatesRef.current.set(metric, currentState);
      });
    },
    [client, rangeKey],
  );

  /**
   * Checks if a metric has a latched (peak-triggered) label.
   * Used to skip further calculations for that metric.
   */
  const isMetricLatched = useCallback((metric: MetricType): boolean => {
    const state = labelStatesRef.current.get(metric);
    return state?.latched === true;
  }, []);

  return {
    rangeKey,
    rangeStartTime,
    createRange,
    updateEndTime,
    clearStopValues,
    finalizeRange,
    addMetricLabel,
    removeTransientLabel,
    isMetricLatched,
  };
};
