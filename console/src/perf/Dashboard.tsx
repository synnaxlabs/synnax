// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/perf/Dashboard.css";

import { Button, Flex, Header, Icon, Text } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useDispatch } from "react-redux";

import { type Layout } from "@/layout";
import { MetricSections } from "@/perf/components/MetricSections";
import { useCollectors } from "@/perf/hooks/useCollectors";
import { useProfilingSession } from "@/perf/hooks/useProfilingSession";
import { ZERO_AGGREGATES } from "@/perf/metrics/buffer";
import {
  useSelectCpuReport,
  useSelectElapsedSeconds,
  useSelectFpsReport,
  useSelectGpuReport,
  useSelectLeakReport,
  useSelectStatus,
} from "@/perf/selectors";
import * as Perf from "@/perf/slice";
import { formatTime } from "@/perf/utils/formatting";

export const Dashboard: Layout.Renderer = ({ layoutKey: _layoutKey }): ReactElement => {
  const dispatch = useDispatch();
  const status = useSelectStatus();
  const elapsedSeconds = useSelectElapsedSeconds();
  const leakReport = useSelectLeakReport();
  const fpsReport = useSelectFpsReport();
  const cpuReport = useSelectCpuReport();
  const gpuReport = useSelectGpuReport();

  // Grouping mode: "time" (Live, Changes) or "type" (Live, Delta, Stats)
  const [groupByType, setGroupByType] = useState(true);

  // Track current status for cleanup on unmount
  const currentStatusRef = useRef<string>(status);

  // Store handleSample in a ref to break circular dependency:
  // - useCollectors needs onSample (handleSample)
  // - useProfilingSession needs collectors/sampleBuffer from useCollectors
  // - useProfilingSession returns handleSample
  // The ref allows useCollectors to use the latest handleSample via onSampleRef pattern
  const handleSampleRef = useRef<
    ReturnType<typeof useProfilingSession>["handleSample"] | undefined
  >(undefined);

  // Wrapper that delegates to handleSampleRef
  const onSample = useCallback<NonNullable<typeof handleSampleRef.current>>(
    (sample, buffer) => handleSampleRef.current?.(sample, buffer),
    [],
  );

  // Get collector data and refs
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

  // Update the ref so onSample can delegate to the real handleSample
  handleSampleRef.current = handleSample;

  useEffect(() => {
    currentStatusRef.current = status;
  }, [status]);

  // Cleanup on unmount: reset if still profiling
  useEffect(() => () => {
      const currentStatus = currentStatusRef.current;
      if (currentStatus !== "idle") 
        dispatch(Perf.reset());
      
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
