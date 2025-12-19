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
import {
  type ReactElement,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
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
import { type MetricTableData } from "@/perf/components/MetricTable";
import { SAMPLE_INTERVAL_MS } from "@/perf/constants";
import { type Aggregates, SampleBuffer, ZERO_AGGREGATES } from "@/perf/metrics/buffer";
import { CpuCollector } from "@/perf/metrics/cpu";
import { FrameRateCollector } from "@/perf/metrics/framerate";
import { GpuCollector } from "@/perf/metrics/gpu";
import { HeapCollector } from "@/perf/metrics/heap";
import {
  LongTaskCollector,
  type LongTaskStats,
} from "@/perf/metrics/longtasks";
import {
  type EndpointStats,
  NetworkCollector,
} from "@/perf/metrics/network";
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
import { type LiveMetrics } from "@/perf/types";
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

  // Live metrics state (updated while dashboard is open)
  const [liveMetrics, setLiveMetrics] = useState<LiveMetrics>({
    frameRate: null,
    cpuPercent: null,
    gpuPercent: null,
    heapUsedMB: null,
    heapTotalMB: null,
    networkRequestCount: null,
    longTaskCount: null,
    totalNetworkRequests: null,
    totalLongTasks: null,
  });

  // Pre-allocated sample buffer (memory allocated on mount, not during test)
  const sampleBufferRef = useRef(new SampleBuffer());

  // Track latest sample for network requests display
  const [latestSample, setLatestSample] = useState<MetricSample | null>(null);
  const latestSampleRef = useRef<MetricSample | null>(null);

  // Track aggregates from buffer for stats display
  const [aggregates, setAggregates] = useState<Aggregates>(ZERO_AGGREGATES);

  // Track top endpoints for profiling display
  const [topEndpoints, setTopEndpoints] = useState<MetricTableData<EndpointStats>>({ data: [], total: 0, truncated: false });

  // Track top long tasks for profiling display
  const [topLongTasks, setTopLongTasks] = useState<MetricTableData<LongTaskStats>>({ data: [], total: 0, truncated: false });

  // Grouping mode: "time" (Live, Changes) or "type" (Live, Delta, Stats)
  const [groupByType, setGroupByType] = useState(true);

  // Collectors - shared between live display and test recording
  const collectorsRef = useRef({
    cpu: null as CpuCollector | null,
    gpu: null as GpuCollector | null,
    frameRate: null as FrameRateCollector | null,
    heap: null as HeapCollector | null,
    longTask: null as LongTaskCollector | null,
    network: null as NetworkCollector | null,
  });

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

  const runAnalysis = useCallback(
    (endFPS: number | null, endCPU: number | null, endGPU: number | null) => {
      const captured = capturedRef.current;

      if (captured.initialCPU == null && endCPU != null) {
        captured.initialCPU = endCPU;
        dispatch(
          Perf.setCpuReport({
            ...ZERO_CPU_REPORT,
            startPercent: math.roundTo(endCPU),
          }),
        );
      }
      if (captured.initialGPU == null && endGPU != null) {
        captured.initialGPU = endGPU;
        dispatch(
          Perf.setGpuReport({
            ...ZERO_GPU_REPORT,
            startPercent: math.roundTo(endGPU),
          }),
        );
      }

      const buffer = sampleBufferRef.current;
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

      const fpsContext: FPSContext = { startFPS: captured.initialFPS, endFPS };
      const degradationResult = analyzers.degradation.analyze(fpsContext);

      const aggregates = buffer.getAggregates();
      const cpuResult = analyzers.cpu.analyze({
        startPercent: captured.initialCPU,
        endPercent: endCPU,
        avgPercent: aggregates.avgCpu,
        maxPercent: aggregates.maxCpu,
      });

      const gpuResult = analyzers.gpu.analyze({
        startPercent: captured.initialGPU,
        endPercent: endGPU,
        avgPercent: aggregates.avgGpu,
        maxPercent: aggregates.maxGpu,
      });

      dispatch(Perf.setLeakReport(leakResult));
      dispatch(Perf.setDegradationReport(degradationResult));
      dispatch(Perf.setCpuReport(cpuResult));
      dispatch(Perf.setGpuReport(gpuResult));
    },
    [dispatch],
  );

  // Collect a sample from the shared collectors
  const collectSample = useCallback((): MetricSample => {
    const c = collectorsRef.current;
    return {
      timestamp: performance.now(),
      cpuPercent: c.cpu?.getCpuPercent() ?? null,
      gpuPercent: c.gpu?.getGpuPercent() ?? null,
      heapUsedMB: c.heap?.getHeapUsedMB() ?? null,
      heapTotalMB: c.heap?.getHeapTotalMB() ?? null,
      frameRate: c.frameRate?.getCurrentFPS() ?? null,
      longTaskCount: c.longTask?.getCountSinceLastSample() ?? 0,
      longTaskDurationMs: c.longTask?.getDurationSinceLastSample() ?? 0,
      networkRequestCount: c.network?.getCountSinceLastSample() ?? 0,
    };
  }, []);

  // Start collectors and unified update loop on mount
  useEffect(() => {
    const c = collectorsRef.current;
    c.cpu = new CpuCollector();
    c.gpu = new GpuCollector();
    c.frameRate = new FrameRateCollector();
    c.heap = new HeapCollector();
    c.longTask = new LongTaskCollector();
    c.network = new NetworkCollector();

    c.cpu.start();
    c.gpu.start();
    c.frameRate.start();
    c.heap.start();
    c.longTask.start();
    c.network.start();

    // Update everything together
    const updateInterval = setInterval(() => {

      const sample = collectSample();
      latestSampleRef.current = sample;

      setLatestSample(sample);
      setLiveMetrics({
        frameRate: c.frameRate?.getCurrentFPS() ?? null,
        cpuPercent: c.cpu?.getCpuPercent() ?? null,
        gpuPercent: c.gpu?.getGpuPercent() ?? null,
        heapUsedMB: c.heap?.getHeapUsedMB() ?? null,
        heapTotalMB: c.heap?.getHeapTotalMB() ?? null,
        networkRequestCount: sample.networkRequestCount,
        longTaskCount: sample.longTaskCount,
        totalNetworkRequests: c.network?.getTotalCount() ?? null,
        totalLongTasks: c.longTask?.getTotalCount() ?? null,
      });

      if (status === "running") {
        sampleBufferRef.current.push(sample);
        setAggregates(sampleBufferRef.current.getAggregates());
        setTopEndpoints(c.network?.getTopEndpoints() ?? { data: [], total: 0, truncated: false });
        setTopLongTasks(c.longTask?.getTopLongTasks() ?? { data: [], total: 0, truncated: false });
        runAnalysis(sample.frameRate, sample.cpuPercent, sample.gpuPercent);
      }
    }, SAMPLE_INTERVAL_MS);

    return () => {
      clearInterval(updateInterval);
      c.cpu?.stop();
      c.gpu?.stop();
      c.frameRate?.stop();
      c.heap?.stop();
      c.longTask?.stop();
      c.network?.stop();
    };
  }, [collectSample, status, runAnalysis]);

  // Capture initial FPS/heap/CPU when test starts, final values when test stops
  useEffect(() => {
    const prevStatus = prevStatusRef.current;
    prevStatusRef.current = status;
    const c = collectorsRef.current;
    const captured = capturedRef.current;

    // Capture initial FPS, heap, and CPU when test starts fresh (not when resuming)
    if (status === "running" && prevStatus === "idle") {
      c.longTask?.reset();
      c.network?.reset();
      setTopEndpoints({ data: [], total: 0, truncated: false });
      setTopLongTasks({ data: [], total: 0, truncated: false });

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
      const samples = sampleBufferRef.current.getAllSamples();
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
      latestSampleRef.current = null;
      setLatestSample(null);
      setAggregates(ZERO_AGGREGATES);
      setTopEndpoints({ data: [], total: 0, truncated: false });
      setTopLongTasks({ data: [], total: 0, truncated: false });

      c.longTask?.reset();
      c.network?.reset();
      sampleBufferRef.current.reset();
    }
  }, [status, dispatch, rangeKey, createProfilingRange, updateRangeEndTime]);

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
        topEndpoints={topEndpoints}
        topLongTasks={topLongTasks}
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
