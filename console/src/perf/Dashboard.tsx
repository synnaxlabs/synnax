// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/perf/Dashboard.css";

import { Button, Flex, Header, Icon, Text, Tooltip } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useDispatch } from "react-redux";

import { type Layout } from "@/layout";
import { MacroPanel } from "@/perf/components/MacroPanel";
import { MetricSections } from "@/perf/components/MetricSections";
import { useCollectors } from "@/perf/hooks/useCollectors";
import { useElapsedSeconds } from "@/perf/hooks/useElapsedSeconds";
import { useProfilingSession } from "@/perf/hooks/useProfilingSession";
import { TickProvider } from "@/perf/hooks/useTick";
import { ZERO_AGGREGATES } from "@/perf/metrics/buffer";
import {
  useSelectCpuReport,
  useSelectFpsReport,
  useSelectGpuReport,
  useSelectLeakReport,
  useSelectStatus,
} from "@/perf/selectors";
import * as Perf from "@/perf/slice";
import { formatTime } from "@/perf/utils/formatting";

const DEV = import.meta.env.DEV;

const DashboardContent = (): ReactElement => {
  const dispatch = useDispatch();
  const status = useSelectStatus();
  const elapsedSeconds = useElapsedSeconds();
  const leakReport = useSelectLeakReport();
  const fpsReport = useSelectFpsReport();
  const cpuReport = useSelectCpuReport();
  const gpuReport = useSelectGpuReport();

  const [groupByType, setGroupByType] = useState(true);
  const [showMacros, setShowMacros] = useState(false);

  // Track current status for cleanup on unmount
  const currentStatusRef = useRef<string>(status);

  /**
   * CIRCULAR DEPENDENCY RESOLUTION PATTERN
   *
   * Problem: We have a circular dependency between two hooks:
   *   1. useCollectors needs `onSample` callback to report collected samples
   *   2. useProfilingSession needs `collectors` and `sampleBuffer` from useCollectors
   *   3. useProfilingSession returns `handleSample` which IS the onSample callback
   *
   * Solution: Use a ref + stable wrapper pattern:
   *   1. Create a ref (handleSampleRef) to hold the eventual handleSample function
   *   2. Create a stable wrapper (onSample) that delegates to handleSampleRef.current
   *   3. Pass the stable wrapper to useCollectors (it never changes identity)
   *   4. Call useProfilingSession to get the real handleSample
   *   5. Update handleSampleRef.current with the real function
   *
   * This works because:
   *   - The wrapper's identity is stable (empty deps), so useCollectors doesn't re-run
   *   - The ref is updated synchronously during render, before any effects run
   *   - When useCollectors calls onSample, it will use the latest handleSample
   */
  const handleSampleRef = useRef<
    ReturnType<typeof useProfilingSession>["handleSample"] | undefined
  >(undefined);

  // Stable wrapper that delegates to handleSampleRef - identity never changes
  const onSample = useCallback<NonNullable<typeof handleSampleRef.current>>(
    (sample, buffer) => handleSampleRef.current?.(sample, buffer),
    [],
  );

  // Get collector data and refs
  const {
    liveMetrics,
    tableData,
    aggregates,
    collectors,
    sampleBuffer,
    resetEventCollectors,
    resetTableData,
    resetBuffer,
  } = useCollectors({ status, onSample });

  const getAggregates = useCallback(
    () => sampleBuffer.current?.getAggregates() ?? ZERO_AGGREGATES,
    [],
  );

  // Use the profiling session hook to orchestrate everything
  const { handleSample } = useProfilingSession({
    status,
    collectors,
    sampleBuffer,
    getAggregates,
    resetEventCollectors,
    resetTableData,
    resetBuffer,
  });

  handleSampleRef.current = handleSample;

  useEffect(() => {
    currentStatusRef.current = status;
  }, [status]);

  useEffect(() => () => {
      const currentStatus = currentStatusRef.current;
      if (currentStatus !== "idle") 
        dispatch(Perf.reset());
      
    }, [dispatch]);

  const handleStart = useCallback(() => dispatch(Perf.start(undefined)), [dispatch]);
  const handlePause = useCallback(() => dispatch(Perf.pause()), [dispatch]);
  const handleResume = useCallback(() => dispatch(Perf.resume()), [dispatch]);
  const handleReset = useCallback(() => dispatch(Perf.reset()), [dispatch]);
  const toggleGroupByType = useCallback(() => setGroupByType((prev) => !prev), []);
  const toggleMacros = useCallback(() => setShowMacros((prev) => !prev), []);

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
            onClick={toggleGroupByType}
          >
            <Icon.Filter />
            {groupByType ? "By Resource" : "By Category"}
          </Button.Button>
          {DEV && (
            <Tooltip.Dialog location="bottom">
              <Text.Text level="small">Toggle Macros (Dev Only)</Text.Text>
              <Button.Button variant="text" size="tiny" onClick={toggleMacros}>
                <Icon.Bolt />
              </Button.Button>
            </Tooltip.Dialog>
          )}
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

      {showMacros && <MacroPanel />}

      <MetricSections
        groupByType={groupByType}
        liveMetrics={liveMetrics}
        aggregates={aggregates}
        topEndpoints={tableData.endpoints}
        topLongTasks={tableData.longTasks}
        topConsoleLogs={tableData.consoleLogs}
        fpsReport={fpsReport}
        leakReport={leakReport}
        cpuReport={cpuReport}
        gpuReport={gpuReport}
        severities={{
          fps: {
            peakSeverity: fpsReport.peakSeverity,
            avgSeverity: fpsReport.avgSeverity,
          },
          cpu: {
            peakSeverity: cpuReport.peakSeverity,
            avgSeverity: cpuReport.avgSeverity,
          },
          gpu: {
            peakSeverity: gpuReport.peakSeverity,
            avgSeverity: gpuReport.avgSeverity,
          },
          heap: leakReport.severity,
        }}
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

export const Dashboard: Layout.Renderer = (): ReactElement => (
  <TickProvider>
    <DashboardContent />
  </TickProvider>
);
