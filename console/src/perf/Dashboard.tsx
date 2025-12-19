// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/perf/Dashboard.css";

import { TimeStamp } from "@synnaxlabs/client";
import { Button, Flex, Header, Icon, Synnax, Text } from "@synnaxlabs/pluto";
import { math, TimeRange } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useDispatch } from "react-redux";

import { type Layout } from "@/layout";
import { CpuAnalyzer } from "@/perf/analyzer/cpu-analyzer";
import { DegradationDetector, type FPSContext } from "@/perf/analyzer/degradation";
import { GpuAnalyzer } from "@/perf/analyzer/gpu-analyzer";
import { LeakDetector } from "@/perf/analyzer/leak-detector";
import {
  ZERO_CPU_REPORT,
  ZERO_DEGRADATION_REPORT,
  ZERO_GPU_REPORT,
  ZERO_LEAK_REPORT,
} from "@/perf/analyzer/types";
import { MetricSections } from "@/perf/components/MetricSections";
import { useCollectors } from "@/perf/hooks/useCollectors";
import { type SampleBuffer } from "@/perf/metrics/buffer";
import { type MetricSample } from "@/perf/metrics/types";
import {
  useSelectCpuReport,
  useSelectDegradationReport,
  useSelectElapsedSeconds,
  useSelectGpuReport,
  useSelectLeakReport,
  useSelectRangeKey,
  useSelectRangeStartTime,
  useSelectStatus,
} from "@/perf/selectors";
import * as Perf from "@/perf/slice";
import { formatTime } from "@/perf/utils/formatting";

export const Dashboard: Layout.Renderer = ({ layoutKey: _layoutKey }): ReactElement => {
  const dispatch = useDispatch();
  const status = useSelectStatus();
  const elapsedSeconds = useSelectElapsedSeconds();
  const leakReport = useSelectLeakReport();
  const degradationReport = useSelectDegradationReport();
  const cpuReport = useSelectCpuReport();
  const gpuReport = useSelectGpuReport();
  const rangeKey = useSelectRangeKey();
  const rangeStartTime = useSelectRangeStartTime();

  // Synnax client for range creation
  const client = Synnax.use();

  // Grouping mode: "time" (Live, Changes) or "type" (Live, Delta, Stats)
  const [groupByType, setGroupByType] = useState(true);

  // Analyzers for detecting performance issues
  const analyzersRef = useRef({
    leak: new LeakDetector(),
    degradation: new DegradationDetector(),
    cpu: new CpuAnalyzer(),
    gpu: new GpuAnalyzer(),
  });

  // Captured values when test starts/stops
  const capturedRef = useRef({
    initialFPS: null as number | null,
    finalFPS: null as number | null,
    initialCPU: null as number | null,
    finalCPU: null as number | null,
    initialGPU: null as number | null,
    finalGPU: null as number | null,
  });

  const prevStatusRef = useRef<string>(status);
  const currentStatusRef = useRef<string>(status);

  const handleSample = useCallback(
    (sample: MetricSample, buffer: SampleBuffer) => {
      const captured = capturedRef.current;

      if (captured.initialCPU == null && sample.cpuPercent != null) {
        captured.initialCPU = sample.cpuPercent;
        dispatch(
          Perf.setCpuReport({
            ...ZERO_CPU_REPORT,
            startPercent: math.roundTo(sample.cpuPercent),
          }),
        );
      }
      if (captured.initialGPU == null && sample.gpuPercent != null) {
        captured.initialGPU = sample.gpuPercent;
        dispatch(
          Perf.setGpuReport({
            ...ZERO_GPU_REPORT,
            startPercent: math.roundTo(sample.gpuPercent),
          }),
        );
      }

      const allSamples = buffer.getAllSamples();
      if (allSamples.length < 2) return;

      const analyzers = analyzersRef.current;

      const leakResult = analyzers.leak.analyze(
        allSamples
          .filter((s) => s.heapUsedMB != null)
          .map((s) => ({
            timestamp: s.timestamp,
            heapUsedMB: s.heapUsedMB!,
            heapTotalMB: s.heapTotalMB!,
          })),
      );

      const fpsContext: FPSContext = {
        startFPS: captured.initialFPS,
        endFPS: sample.frameRate,
      };
      const degradationResult = analyzers.degradation.analyze(fpsContext);

      const agg = buffer.getAggregates();
      const cpuResult = analyzers.cpu.analyze({
        startPercent: captured.initialCPU,
        endPercent: sample.cpuPercent,
        avgPercent: agg.avgCpu,
        maxPercent: agg.maxCpu,
      });

      const gpuResult = analyzers.gpu.analyze({
        startPercent: captured.initialGPU,
        endPercent: sample.gpuPercent,
        avgPercent: agg.avgGpu,
        maxPercent: agg.maxGpu,
      });

      dispatch(Perf.setLeakReport(leakResult));
      dispatch(Perf.setDegradationReport(degradationResult));
      dispatch(Perf.setCpuReport(cpuResult));
      dispatch(Perf.setGpuReport(gpuResult));
    },
    [dispatch],
  );

  const {
    liveMetrics,
    tableData,
    aggregates,
    latestSample,
    collectors,
    sampleBuffer,
    resetEventCollectors,
    resetTableData,
    resetBuffer,
  } = useCollectors({ status, onSample: handleSample });

  // Create a new profiling range with MAX end time (indicates ongoing session)
  const createProfilingRange = useCallback(() => {
    if (client == null) return;

    const now = TimeStamp.now();
    const maxTimestamp = TimeStamp.MAX;
    client.ranges
      .create({
        name: `Console Profiling - ${now.toLocaleString()}`,
        timeRange: new TimeRange(now, maxTimestamp).numeric,
      })
      .then((range) => {
        dispatch(Perf.setRangeKey(range.key));
        dispatch(Perf.setRangeStartTime(Number(now.valueOf())));
      })
      .catch((error: Error) => {
        console.error("Failed to create profiling range:", error);
      });
  }, [client, dispatch]);

  // Update the end time of the current profiling range
  const updateRangeEndTime = useCallback(
    (newEndTime: TimeStamp) => {
      if (client == null || rangeKey == null || rangeStartTime == null) return;

      const rangeName = `Console Profiling - ${new TimeStamp(rangeStartTime).toLocaleString()}`;
      // Note: Synnax uses create() with existing key to update ranges
      client.ranges
        .create({
          key: rangeKey,
          name: rangeName,
          timeRange: new TimeRange(new TimeStamp(rangeStartTime), newEndTime).numeric,
        })
        .catch((error: Error) => {
          console.error("Failed to update profiling range:", error);
        });
    },
    [client, rangeKey, rangeStartTime],
  );

  // Handle status transitions: capture initial/final values, manage ranges
  useEffect(() => {
    const prevStatus = prevStatusRef.current;
    prevStatusRef.current = status;
    const c = collectors.current;
    const captured = capturedRef.current;

    // Capture initial FPS, heap, and CPU when test starts fresh (not when resuming)
    if (status === "running" && prevStatus === "idle") {
      resetEventCollectors();
      resetTableData();

      const initialFPS = c.frameRate?.getCurrentFPS() ?? null;
      captured.initialFPS = initialFPS;
      captured.finalFPS = null;
      dispatch(
        Perf.setDegradationReport({
          ...ZERO_DEGRADATION_REPORT,
          averageFrameRateStart: initialFPS ?? 0,
        }),
      );

      const initialHeap = c.heap?.getHeapUsedMB() ?? 0;
      dispatch(
        Perf.setLeakReport({
          ...ZERO_LEAK_REPORT,
          heapStartMB: math.roundTo(initialHeap),
        }),
      );

      const initialCPU = c.cpu?.getCpuPercent() ?? null;
      captured.initialCPU = initialCPU;
      captured.finalCPU = null;
      dispatch(
        Perf.setCpuReport({
          ...ZERO_CPU_REPORT,
          startPercent: initialCPU != null ? math.roundTo(initialCPU) : null,
        }),
      );

      const initialGPU = c.gpu?.getGpuPercent() ?? null;
      captured.initialGPU = initialGPU;
      captured.finalGPU = null;
      dispatch(
        Perf.setGpuReport({
          ...ZERO_GPU_REPORT,
          startPercent: initialGPU != null ? math.roundTo(initialGPU) : null,
        }),
      );

      // Create a new range when profiling starts
      createProfilingRange();
    }

    if (status === "running" && prevStatus === "running" && rangeKey == null) {
      createProfilingRange();
    }

    // Capture final values when pausing
    if (status === "paused" && prevStatus === "running") {
      const samples = sampleBuffer.current.getAllSamples();
      const lastSample = samples.at(-1);
      if (lastSample != null) {
        captured.finalFPS = lastSample.frameRate;
        captured.finalCPU = lastSample.cpuPercent;
        captured.finalGPU = lastSample.gpuPercent;
      } else {
        captured.finalFPS = c.frameRate?.getCurrentFPS() ?? null;
        captured.finalCPU = c.cpu?.getCpuPercent() ?? null;
        captured.finalGPU = c.gpu?.getGpuPercent() ?? null;
      }

      // Pause
      updateRangeEndTime(TimeStamp.now());
    }

    // Resume
    if (status === "running" && prevStatus === "paused") {
      updateRangeEndTime(TimeStamp.MAX);
    }

    // Reset when test is reset to idle
    if (status === "idle" && prevStatus !== "idle") {
      // Finalize the range with current timestamp before resetting
      if (prevStatus === "running" || prevStatus === "paused") {
        updateRangeEndTime(TimeStamp.now());
      }

      captured.initialCPU = null;
      captured.finalCPU = null;
      captured.initialGPU = null;
      captured.finalGPU = null;
      captured.initialFPS = null;
      captured.finalFPS = null;

      resetEventCollectors();
      resetTableData();
      resetBuffer();
    }
  }, [
    status,
    dispatch,
    rangeKey,
    createProfilingRange,
    updateRangeEndTime,
    collectors,
    sampleBuffer,
    resetEventCollectors,
    resetTableData,
    resetBuffer,
  ]);

  useEffect(() => {
    currentStatusRef.current = status;
  }, [status]);

  useEffect(() => {
    return () => {
      const currentStatus = currentStatusRef.current;
      if (currentStatus !== "idle") {
        dispatch(Perf.reset());
      }
    };
  }, [dispatch]);

  const handleStart = useCallback(() => dispatch(Perf.start(undefined)), [dispatch]);
  const handlePause = useCallback(() => dispatch(Perf.pause()), [dispatch]);
  const handleResume = useCallback(() => dispatch(Perf.resume()), [dispatch]);
  const handleReset = useCallback(() => dispatch(Perf.reset()), [dispatch]);

  const btn = useMemo(() => {
    switch (status) {
      case "idle":
        return {
          icon: <Icon.Play />,
          text: "Start",
          handler: handleStart,
          variant: "filled" as const,
        };
      case "running":
        return {
          icon: <Icon.Pause />,
          text: formatTime(elapsedSeconds),
          handler: handlePause,
          variant: "outlined" as const,
        };
      case "paused":
        return {
          icon: <Icon.Play />,
          text: formatTime(elapsedSeconds),
          handler: handleResume,
          variant: "outlined" as const,
        };
      case "error":
      default:
        return {
          icon: <Icon.Refresh />,
          text: "Reset",
          handler: handleReset,
          variant: "outlined" as const,
        };
    }
  }, [status, elapsedSeconds, handleStart, handlePause, handleResume, handleReset]);

  const showResetButton = status === "paused";

  return (
    <Flex.Box y className="console-perf-dashboard" grow>
      <Header.Header level="h4">
        <Header.Actions grow>
          <Button.Button
            variant="text"
            size="tiny"
            onClick={() => setGroupByType(!groupByType)}
          >
            <Icon.Filter />
            {groupByType ? "By Resource" : "By Category"}
          </Button.Button>
          <Flex.Box grow />
          {showResetButton && (
            <Button.Button
              variant="text"
              size="tiny"
              onClick={handleReset}
              className="console-perf-reset-button"
            >
              <Icon.Refresh />
            </Button.Button>
          )}
          <Button.Button variant={btn.variant} size="tiny" onClick={btn.handler}>
            {btn.icon}
            {btn.text}
          </Button.Button>
        </Header.Actions>
      </Header.Header>

      <MetricSections
        groupByType={groupByType}
        liveMetrics={liveMetrics}
        aggregates={aggregates}
        latestSample={latestSample}
        topEndpoints={tableData.endpoints}
        topLongTasks={tableData.longTasks}
        topConsoleLogs={tableData.consoleLogs}
        degradationReport={degradationReport}
        leakReport={leakReport}
        cpuReport={cpuReport}
        gpuReport={gpuReport}
        status={status}
      />

      {status === "error" && (
        <Text.Text status="error" className="console-perf-error">
          An error occurred during performance testing
        </Text.Text>
      )}
    </Flex.Box>
  );
};
