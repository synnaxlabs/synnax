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
import { math } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useEffect, useRef, useState } from "react";
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
import { SampleBuffer } from "@/perf/metrics/buffer";
import { CpuCollector } from "@/perf/metrics/cpu";
import { FrameRateCollector } from "@/perf/metrics/framerate";
import { GpuCollector } from "@/perf/metrics/gpu";
import { HeapCollector } from "@/perf/metrics/heap";
import { LongTaskCollector } from "@/perf/metrics/longtasks";
import { NetworkCollector } from "@/perf/metrics/network";
import { type MetricSample } from "@/perf/metrics/types";
import {
  useSelectCpuReport,
  useSelectDegradationReport,
  useSelectElapsedSeconds,
  useSelectGpuReport,
  useSelectLeakReport,
  useSelectStatus,
} from "@/perf/selectors";
import * as Perf from "@/perf/slice";

interface LiveMetrics {
  frameRate: number;
  cpuPercent: number | null;
  gpuPercent: number | null;
  heapUsedMB: number | null;
  heapTotalMB: number | null;
}

const ZERO_PERCENT_DEFAULT = "0.0%";
const NA = "N/A";

/** Format seconds as MM:SS. */
const formatTime = (seconds: number): string => {
  const mins = Math.floor(seconds / 60);
  const secs = Math.floor(seconds % 60);
  return `${mins.toString().padStart(2, "0")}:${secs.toString().padStart(2, "0")}`;
};

const formatPercent = (value: number | null): string =>
  value != null ? `${value.toFixed(1)}%` : NA;

const formatMB = (value: number | null): string =>
  value != null ? `${value.toFixed(1)} MB` : NA;

type Status = "success" | "warning" | "error" | "info" | undefined;

const getThresholdStatus = (
  value: number | null,
  warningThreshold: number,
  errorThreshold: number,
): Status => {
  if (value == null) return undefined;
  if (value > errorThreshold) return "error";
  if (value > warningThreshold) return "warning";
  return undefined;
};

const getLowThresholdStatus = (
  value: number,
  warningThreshold: number,
  errorThreshold: number,
): Status => {
  if (value < errorThreshold) return "error";
  if (value < warningThreshold) return "warning";
  return undefined;
};

const getWarningStatus = (value: number, threshold: number): Status =>
  value > threshold ? "warning" : undefined;

const getAvgPeakStatus = (
  avg: number | null,
  peak: number | null,
  avgThreshold: number,
  peakThreshold: number,
): Status =>
  (avg ?? 0) > avgThreshold || (peak ?? 0) > peakThreshold ? "warning" : undefined;

const formatStartEnd = (
  start: number,
  end: number,
  suffix: string = "",
): string => {
  if (start === 0 && end === 0) return `— / —${suffix}`;
  const endStr = end === 0 ? "—" : end.toFixed(1);
  return `${start.toFixed(1)} / ${endStr}${suffix}`;
};

interface MetricCardProps {
  label: string;
  value: string;
  status?: Status;
  tooltip?: string;
}

const MetricCard = ({
  label,
  value,
  status,
  tooltip,
}: MetricCardProps): ReactElement => {
  const card = (
    <Flex.Box y className="console-perf-metric-card">
      <Text.Text level="small" color={7}>
        {label}
      </Text.Text>
      <Text.Text
        level="h4"
        color={
          status === "error"
            ? "var(--pluto-error-z)"
            : status === "warning"
              ? "var(--pluto-warning-z)"
              : status === "success"
                ? "var(--pluto-success-z)"
                : undefined
        }
      >
        {value}
      </Text.Text>
    </Flex.Box>
  );

  if (tooltip == null) return card;

  return (
    <Tooltip.Dialog location={{ x: "right", y: "bottom" }}>
      {tooltip}
      {card}
    </Tooltip.Dialog>
  );
};

/** Props for detection status cards (Degradation, Memory Leak, High CPU, High GPU). */
interface DetectionCardProps {
  label: string;
  detected: boolean;
  isIdle: boolean;
  tooltip: string;
}

/** Card that shows detection status with consistent formatting. */
const DetectionCard = ({
  label,
  detected,
  isIdle,
  tooltip,
}: DetectionCardProps): ReactElement => (
  <MetricCard
    label={label}
    value={isIdle ? "—" : detected ? "DETECTED" : "None"}
    status={isIdle ? undefined : detected ? "error" : "success"}
    tooltip={tooltip}
  />
);

/** Format a start/end percent pair for display. */
const formatStartEndPercent = (
  start: number | null,
  end: number | null,
): string => {
  if (start != null && end != null)
    return `${start.toFixed(1)} / ${end.toFixed(1)}%`;
  if (start != null) return `${start.toFixed(1)} / —%`;
  return "— / —%";
};

/** Format avg/peak percent pair for display. */
const formatAvgPeakPercent = (
  avg: number | null,
  peak: number | null,
): string => {
  if (avg != null && peak != null)
    return `${avg.toFixed(1)} / ${peak.toFixed(1)}%`;
  return ZERO_PERCENT_DEFAULT;
};

export const Dashboard: Layout.Renderer = ({ layoutKey: _layoutKey }): ReactElement => {
  const dispatch = useDispatch();
  const status = useSelectStatus();
  const elapsedSeconds = useSelectElapsedSeconds();
  const leakReport = useSelectLeakReport();
  const degradationReport = useSelectDegradationReport();
  const cpuReport = useSelectCpuReport();
  const gpuReport = useSelectGpuReport();

  const isIdle = status === "idle";

  // Live metrics state (updated while dashboard is open)
  const [liveMetrics, setLiveMetrics] = useState<LiveMetrics>({
    frameRate: 0,
    cpuPercent: null,
    gpuPercent: null,
    heapUsedMB: null,
    heapTotalMB: null,
  });

  // Pre-allocated sample buffer (memory allocated on mount, not during test)
  const sampleBufferRef = useRef(new SampleBuffer());

  // Track latest sample for network requests display
  const [latestSample, setLatestSample] = useState<MetricSample | null>(null);

  // Single set of collectors - shared between live display and test recording
  const cpuRef = useRef<CpuCollector | null>(null);
  const gpuRef = useRef<GpuCollector | null>(null);
  const frameRateRef = useRef<FrameRateCollector | null>(null);
  const heapRef = useRef<HeapCollector | null>(null);
  const longTaskRef = useRef<LongTaskCollector | null>(null);
  const networkRef = useRef<NetworkCollector | null>(null);

  // Analysis refs
  const leakDetectorRef = useRef(new LeakDetector());
  const degradationDetectorRef = useRef(new DegradationDetector());
  const cpuAnalyzerRef = useRef(new CpuAnalyzer());
  const gpuAnalyzerRef = useRef(new GpuAnalyzer());

  // Sample collection interval ref (only active during test)
  const sampleIntervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  // Store the initial and final values when test starts/stops
  const initialFPSRef = useRef<number>(0);
  const finalFPSRef = useRef<number>(0);
  const initialCPURef = useRef<number | null>(null);
  const finalCPURef = useRef<number | null>(null);
  const initialGPURef = useRef<number | null>(null);
  const finalGPURef = useRef<number | null>(null);
  const prevStatusRef = useRef<string>(status);

  // Helper to run analysis and dispatch results (used by both periodic and final effects)
  const runAnalysis = useCallback(
    (endFPS: number, endCPU: number | null, endGPU: number | null) => {
      const buffer = sampleBufferRef.current;
      const allSamples = buffer.getAllSamples();
      if (allSamples.length < 2) return;

      const leakResult = leakDetectorRef.current.analyze(
        allSamples
          .filter((s) => s.heapUsedMB != null)
          .map((s) => ({
            timestamp: s.timestamp,
            heapUsedMB: s.heapUsedMB!,
            heapTotalMB: s.heapTotalMB!,
          })),
      );

      const fpsContext: FPSContext = { startFPS: initialFPSRef.current, endFPS };
      const degradationResult = degradationDetectorRef.current.analyze(fpsContext);

      const aggregates = buffer.getAggregates();
      const cpuResult = cpuAnalyzerRef.current.analyze({
        startPercent: initialCPURef.current,
        endPercent: endCPU,
        avgPercent: aggregates.avgCpu,
        peakPercent: aggregates.peakCpu,
      });

      const gpuResult = gpuAnalyzerRef.current.analyze({
        startPercent: initialGPURef.current,
        endPercent: endGPU,
        avgPercent: aggregates.avgGpu,
        peakPercent: aggregates.peakGpu,
      });

      dispatch(Perf.setLeakReport(leakResult));
      dispatch(Perf.setDegradationReport(degradationResult));
      dispatch(Perf.setCpuReport(cpuResult));
      dispatch(Perf.setGpuReport(gpuResult));
    },
    [dispatch],
  );

  // Collect a sample from the shared collectors
  const collectSample = useCallback(
    (): MetricSample => ({
      timestamp: performance.now(),
      cpuPercent: cpuRef.current?.getCpuPercent() ?? null,
      gpuPercent: gpuRef.current?.getGpuPercent() ?? null,
      heapUsedMB: heapRef.current?.getHeapUsedMB() ?? null,
      heapTotalMB: heapRef.current?.getHeapTotalMB() ?? null,
      frameRate: frameRateRef.current?.getCurrentFPS() ?? 0,
      longTaskCount: longTaskRef.current?.getCountSinceLastSample() ?? 0,
      longTaskDurationMs: longTaskRef.current?.getDurationSinceLastSample() ?? 0,
      networkRequestCount: networkRef.current?.getCountSinceLastSample() ?? 0,
    }),
    [],
  );

  // Start collectors on mount (live metrics always available while dashboard is open)
  useEffect(() => {
    cpuRef.current = new CpuCollector();
    gpuRef.current = new GpuCollector();
    frameRateRef.current = new FrameRateCollector();
    heapRef.current = new HeapCollector();
    longTaskRef.current = new LongTaskCollector();
    networkRef.current = new NetworkCollector();

    cpuRef.current.start();
    gpuRef.current.start();
    frameRateRef.current.start();
    heapRef.current.start();

    // Update live metrics display every 500ms
    const liveInterval = setInterval(() => {
      setLiveMetrics({
        frameRate: frameRateRef.current?.getCurrentFPS() ?? 0,
        cpuPercent: cpuRef.current?.getCpuPercent() ?? null,
        gpuPercent: gpuRef.current?.getGpuPercent() ?? null,
        heapUsedMB: heapRef.current?.getHeapUsedMB() ?? null,
        heapTotalMB: heapRef.current?.getHeapTotalMB() ?? null,
      });
    }, 500);

    return () => {
      clearInterval(liveInterval);
      cpuRef.current?.stop();
      gpuRef.current?.stop();
      frameRateRef.current?.stop();
      heapRef.current?.stop();
      longTaskRef.current?.stop();
      networkRef.current?.stop();
    };
  }, []);

  // Start/stop analysis
  useEffect(() => {
    if (status === "running") {
      networkRef.current?.start();
      longTaskRef.current?.start();

      sampleIntervalRef.current = setInterval(() => {
        const sample = collectSample();
        sampleBufferRef.current.push(sample);
        setLatestSample(sample);
        runAnalysis(sample.frameRate, sample.cpuPercent, sample.gpuPercent);
      }, 1000);
    } else if (sampleIntervalRef.current != null) {
      // Stop recording
      clearInterval(sampleIntervalRef.current);
      sampleIntervalRef.current = null;
      networkRef.current?.stop();
      longTaskRef.current?.stop();
    }

    return () => {
      if (sampleIntervalRef.current != null) {
        clearInterval(sampleIntervalRef.current);
        sampleIntervalRef.current = null;
      }
    };
  }, [status, collectSample, runAnalysis]);

  // Capture initial FPS/heap/CPU when test starts, final values when test stops
  useEffect(() => {
    const prevStatus = prevStatusRef.current;
    prevStatusRef.current = status;

    // Capture initial FPS, heap, and CPU when test starts
    if (status === "running" && prevStatus !== "running") {
      const initialFPS = frameRateRef.current?.getCurrentFPS() ?? 0;
      initialFPSRef.current = initialFPS;
      finalFPSRef.current = 0;
      dispatch(
        Perf.setDegradationReport({
          ...ZERO_DEGRADATION_REPORT,
          averageFrameRateStart: initialFPS,
        }),
      );

      const initialHeap = heapRef.current?.getHeapUsedMB() ?? 0;
      dispatch(
        Perf.setLeakReport({
          ...ZERO_LEAK_REPORT,
          heapStartMB: math.roundTo(initialHeap),
        }),
      );

      const initialCPU = cpuRef.current?.getCpuPercent() ?? null;
      initialCPURef.current = initialCPU;
      finalCPURef.current = null;
      dispatch(
        Perf.setCpuReport({
          ...ZERO_CPU_REPORT,
          startPercent: initialCPU != null ? math.roundTo(initialCPU) : null,
        }),
      );

      const initialGPU = gpuRef.current?.getGpuPercent() ?? null;
      initialGPURef.current = initialGPU;
      finalGPURef.current = null;
      dispatch(
        Perf.setGpuReport({
          ...ZERO_GPU_REPORT,
          startPercent: initialGPU != null ? math.roundTo(initialGPU) : null,
        }),
      );
    }

    // Use last buffered sample to avoid capturing stop-logic overhead
    if (status === "completed" && prevStatus === "running") {
      const samples = sampleBufferRef.current.getAllSamples();
      const lastSample = samples.at(-1);
      if (lastSample != null) {
        finalFPSRef.current = lastSample.frameRate;
        finalCPURef.current = lastSample.cpuPercent;
        finalGPURef.current = lastSample.gpuPercent;
      } else {
        finalFPSRef.current = frameRateRef.current?.getCurrentFPS() ?? 0;
        finalCPURef.current = cpuRef.current?.getCpuPercent() ?? null;
        finalGPURef.current = gpuRef.current?.getGpuPercent() ?? null;
      }
    }

    // Reset when test is reset to idle
    if (status === "idle" && prevStatus !== "idle") {
      initialCPURef.current = null;
      finalCPURef.current = null;
      initialGPURef.current = null;
      finalGPURef.current = null;
      initialFPSRef.current = 0;
      finalFPSRef.current = 0;
      setLatestSample(null);

      cpuRef.current?.reset();
      gpuRef.current?.reset();
      frameRateRef.current?.reset();
      heapRef.current?.reset();
      longTaskRef.current?.reset();
      networkRef.current?.reset();
      sampleBufferRef.current.reset();
    }
  }, [status, dispatch]);

  // Run final analysis when test completes
  useEffect(() => {
    if (status !== "completed") return;
    runAnalysis(finalFPSRef.current, finalCPURef.current, finalGPURef.current);
  }, [status, runAnalysis]);

  const handleStart = () => dispatch(Perf.start(undefined));
  const handleStop = () => dispatch(Perf.stop());
  const handleReset = () => dispatch(Perf.reset());

  return (
    <Flex.Box y className="console-perf-dashboard" grow>
      <Header.Header level="h4">
        <Header.Title>Live Metrics</Header.Title>
      </Header.Header>

      <Flex.Box x wrap>
        <MetricCard
          label="Frame Rate"
          value={`${liveMetrics.frameRate.toFixed(1)} FPS`}
          status={getLowThresholdStatus(liveMetrics.frameRate, 30, 15)}
          tooltip="Current frames per second measured via requestAnimationFrame. Target is 60 FPS. Warning below 30, error below 15."
        />
        <MetricCard
          label="Memory"
          value={formatMB(liveMetrics.heapUsedMB)}
          tooltip="Current process memory usage."
        />
        <MetricCard
          label="CPU"
          value={formatPercent(liveMetrics.cpuPercent)}
          status={getThresholdStatus(liveMetrics.cpuPercent, 50, 80)}
          tooltip="Current process CPU usage percentage. Warning above 50%, error above 80%. Not available in browser."
        />
        <MetricCard
          label="GPU"
          value={formatPercent(liveMetrics.gpuPercent)}
          status={getThresholdStatus(liveMetrics.gpuPercent, 80, 95)}
          tooltip="Current GPU utilization percentage. Warning above 80%, error above 95%. Available on macOS (via IOKit) and Windows/Linux (via NVML for NVIDIA GPUs)."
        />
      </Flex.Box>

      <Header.Header level="h4" className="console-perf-analysis-header">
        <Header.Title>Analysis</Header.Title>
        <Header.Actions>
          <Button.Button
            variant={status === "idle" ? "filled" : "outlined"}
            size="tiny"
            onClick={
              status === "idle"
                ? handleStart
                : status === "running"
                  ? handleStop
                  : handleReset
            }
          >
            {status === "idle" ? (
              <Icon.Play />
            ) : status === "running" ? (
              <Icon.Pause />
            ) : (
              <Icon.Refresh />
            )}
            {status === "idle" ? "Start" : formatTime(elapsedSeconds)}
          </Button.Button>
        </Header.Actions>
      </Header.Header>

      {/* Row 1: Status indicators */}
      <Flex.Box x wrap>
        <DetectionCard
          label="Degradation"
          detected={degradationReport.detected}
          isIdle={isIdle}
          tooltip="Frame rate degradation over time. Detected when FPS drops >15% from start to end of test."
        />
        <DetectionCard
          label="Memory Leak"
          detected={leakReport.detected}
          isIdle={isIdle}
          tooltip="Potential memory leak detected via linear regression on heap samples. Detected when heap grows >20% with consistent upward trend."
        />
        <DetectionCard
          label="High CPU"
          detected={cpuReport.detected}
          isIdle={isIdle}
          tooltip="High CPU usage. Detected when average CPU >50% or peak CPU >80%."
        />
        <DetectionCard
          label="High GPU"
          detected={gpuReport.detected}
          isIdle={isIdle}
          tooltip="High GPU usage. Detected when average GPU >80% or peak GPU >95%."
        />
      </Flex.Box>

      {/* Row 2: Quantitative changes */}
      <Flex.Box x wrap className="console-perf-metric-row">
        <MetricCard
          label="FPS Drop"
          value={`${degradationReport.frameRateDegradationPercent.toFixed(1)}%`}
          status={getThresholdStatus(degradationReport.frameRateDegradationPercent, 10, 15)}
          tooltip="Percentage decrease in frame rate. Warning at >10%, error at >15%."
        />
        <MetricCard
          label="Heap Growth"
          value={`${leakReport.heapGrowthPercent.toFixed(1)}%`}
          status={getWarningStatus(leakReport.heapGrowthPercent, 20)}
          tooltip="Percentage change in heap memory from start to end of test. Warning at >20%."
        />
        <MetricCard
          label="Avg / Peak CPU"
          value={formatAvgPeakPercent(cpuReport.avgPercent, cpuReport.peakPercent)}
          status={getAvgPeakStatus(cpuReport.avgPercent, cpuReport.peakPercent, 50, 80)}
          tooltip="Average and peak CPU usage during the test. Warning if avg >50% or peak >80%."
        />
        <MetricCard
          label="Avg / Peak GPU"
          value={formatAvgPeakPercent(gpuReport.avgPercent, gpuReport.peakPercent)}
          status={getAvgPeakStatus(gpuReport.avgPercent, gpuReport.peakPercent, 80, 95)}
          tooltip="Average and peak GPU usage during the test. Warning if avg >80% or peak >95%."
        />
      </Flex.Box>

      {/* Row 3: Supporting details */}
      <Flex.Box x wrap className="console-perf-metric-row">
        <MetricCard
          label="FPS Start / End"
          value={formatStartEnd(
            degradationReport.averageFrameRateStart,
            degradationReport.averageFrameRateEnd,
          )}
          status={getWarningStatus(degradationReport.frameRateDegradationPercent, 15)}
          tooltip="Frame rate at the start vs end of the test."
        />
        <MetricCard
          label="Heap Start / End"
          value={formatStartEnd(leakReport.heapStartMB, leakReport.heapEndMB, " MB")}
          status={getWarningStatus(leakReport.heapGrowthPercent, 20)}
          tooltip="Heap memory usage at start vs end of test. Warning at >20% growth."
        />
        <MetricCard
          label="CPU Start / End"
          value={formatStartEndPercent(cpuReport.startPercent, cpuReport.endPercent)}
          tooltip="CPU usage at start vs end of test."
        />
        <MetricCard
          label="GPU Start / End"
          value={formatStartEndPercent(gpuReport.startPercent, gpuReport.endPercent)}
          tooltip="GPU usage at start vs end of test."
        />
      </Flex.Box>

      {/* Row 4: Task metrics */}
      <Flex.Box x wrap className="console-perf-metric-row">
        <MetricCard
          label="Long Tasks"
          value={latestSample?.longTaskCount.toString() ?? "—"}
          status={latestSample != null && latestSample.longTaskCount > 5 ? "warning" : undefined}
          tooltip="JavaScript tasks blocking the main thread for >50ms since last sample. High counts indicate UI jank."
        />
        <MetricCard
          label="Network Requests"
          value={latestSample?.networkRequestCount.toString() ?? "—"}
          tooltip="Number of fetch/XHR requests made since last sample. High counts may indicate excessive API polling."
        />
      </Flex.Box>

      {status === "error" && (
        <Text.Text status="error" className="console-perf-error">
          An error occurred during performance testing
        </Text.Text>
      )}
    </Flex.Box>
  );
};
