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
import {
  memo,
  type ReactElement,
  type ReactNode,
  useCallback,
  useEffect,
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

const STATUS_COLORS: Record<string, string> = {
  error: "var(--pluto-error-z)",
  warning: "var(--pluto-warning-z)",
  success: "var(--pluto-success-z)",
};

/** Get status based on threshold. Use inverted=true when lower values are worse (e.g., FPS). */
const getThresholdStatus = (
  value: number | null,
  warningThreshold: number,
  errorThreshold: number,
  inverted = false,
): Status => {
  if (value == null) return undefined;
  const compare = inverted
    ? (v: number, t: number) => v < t
    : (v: number, t: number) => v > t;
  if (compare(value, errorThreshold)) return "error";
  if (compare(value, warningThreshold)) return "warning";
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
  start: number | null,
  end: number | null,
  zeroAsNull: boolean = false,
): string => {
  const startVal = zeroAsNull && start === 0 ? null : start;
  const endVal = zeroAsNull && end === 0 ? null : end;
  const startStr = startVal != null ? startVal.toFixed(1) : "—";
  const endStr = endVal != null ? endVal.toFixed(1) : "—";
  return `${startStr} → ${endStr}`;
};

const formatAvgPeak = (avg: number | null, peak: number | null): string => {
  const avgStr = avg != null ? avg.toFixed(1) : "—";
  const peakStr = peak != null ? peak.toFixed(1) : "—";
  return `${avgStr} / ${peakStr}%`;
};

interface MetricRowProps {
  label: string;
  value: string;
  status?: Status;
  tooltip?: string;
}

const MetricRow = memo(({ label, value, status, tooltip }: MetricRowProps): ReactElement => {
  const row = (
    <Flex.Box
      x
      justify="between"
      align="center"
      className="console-perf-row"
      tabIndex={0}
    >
      <Text.Text level="small" color={7}>
        {label}
      </Text.Text>
      <Text.Text
        level="small"
        color={status != null ? STATUS_COLORS[status] : 9}
      >
        {value}
      </Text.Text>
    </Flex.Box>
  );

  if (tooltip == null) return row;

  return (
    <Tooltip.Dialog location={{ x: "right", y: "center" }}>
      {tooltip}
      {row}
    </Tooltip.Dialog>
  );
});
MetricRow.displayName = "MetricRow";

interface SectionProps {
  title: string;
  secondaryText?: ReactNode;
  defaultOpen?: boolean;
  children: ReactNode;
}

const Section = memo(({
  title,
  secondaryText,
  defaultOpen = true,
  children,
}: SectionProps): ReactElement => {
  const [open, setOpen] = useState(defaultOpen);

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" || e.key === " ") {
      e.preventDefault();
      setOpen(!open);
    }
  };

  return (
    <Flex.Box y className="console-perf-section">
      <Flex.Box
        x
        className="console-perf-section-header"
        onClick={() => setOpen(!open)}
        onKeyDown={handleKeyDown}
        role="button"
        tabIndex={0}
        aria-expanded={open}
        data-open={open}
      >
        <Icon.Caret.Right />
        <Text.Text level="small" color={8} weight={500}>
          {title}
        </Text.Text>
        {secondaryText != null && (
          <Text.Text level="small" color={6} grow style={{ textAlign: "right" }}>
            {secondaryText}
          </Text.Text>
        )}
      </Flex.Box>
      {open && children}
    </Flex.Box>
  );
});
Section.displayName = "Section";

export const Dashboard: Layout.Renderer = ({ layoutKey: _layoutKey }): ReactElement => {
  const dispatch = useDispatch();
  const status = useSelectStatus();
  const elapsedSeconds = useSelectElapsedSeconds();
  const leakReport = useSelectLeakReport();
  const degradationReport = useSelectDegradationReport();
  const cpuReport = useSelectCpuReport();
  const gpuReport = useSelectGpuReport();

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
    initialFPS: 0,
    finalFPS: 0,
    initialCPU: null as number | null,
    finalCPU: null as number | null,
    initialGPU: null as number | null,
    finalGPU: null as number | null,
  });

  const sampleIntervalRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const prevStatusRef = useRef<string>(status);

  // Helper to run analysis and dispatch results (used by both periodic and final effects)
  const runAnalysis = useCallback(
    (endFPS: number, endCPU: number | null, endGPU: number | null) => {
      const buffer = sampleBufferRef.current;
      const allSamples = buffer.getAllSamples();
      if (allSamples.length < 2) return;

      const analyzers = analyzersRef.current;
      const captured = capturedRef.current;

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
        peakPercent: aggregates.peakCpu,
      });

      const gpuResult = analyzers.gpu.analyze({
        startPercent: captured.initialGPU,
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
  const collectSample = useCallback((): MetricSample => {
    const c = collectorsRef.current;
    return {
      timestamp: performance.now(),
      cpuPercent: c.cpu?.getCpuPercent() ?? null,
      gpuPercent: c.gpu?.getGpuPercent() ?? null,
      heapUsedMB: c.heap?.getHeapUsedMB() ?? null,
      heapTotalMB: c.heap?.getHeapTotalMB() ?? null,
      frameRate: c.frameRate?.getCurrentFPS() ?? 0,
      longTaskCount: c.longTask?.getCountSinceLastSample() ?? 0,
      longTaskDurationMs: c.longTask?.getDurationSinceLastSample() ?? 0,
      networkRequestCount: c.network?.getCountSinceLastSample() ?? 0,
    };
  }, []);

  // Start collectors on mount (live metrics always available while dashboard is open)
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

    // Update live metrics display every 500ms
    const liveInterval = setInterval(() => {
      setLiveMetrics({
        frameRate: c.frameRate?.getCurrentFPS() ?? 0,
        cpuPercent: c.cpu?.getCpuPercent() ?? null,
        gpuPercent: c.gpu?.getGpuPercent() ?? null,
        heapUsedMB: c.heap?.getHeapUsedMB() ?? null,
        heapTotalMB: c.heap?.getHeapTotalMB() ?? null,
      });
    }, 500);

    return () => {
      clearInterval(liveInterval);
      c.cpu?.stop();
      c.gpu?.stop();
      c.frameRate?.stop();
      c.heap?.stop();
      c.longTask?.stop();
      c.network?.stop();
    };
  }, []);

  // Start/stop analysis
  useEffect(() => {
    const c = collectorsRef.current;
    if (status === "running") {
      c.network?.start();
      c.longTask?.start();

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
      c.network?.stop();
      c.longTask?.stop();
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
    const c = collectorsRef.current;
    const captured = capturedRef.current;

    // Capture initial FPS, heap, and CPU when test starts
    if (status === "running" && prevStatus !== "running") {
      const initialFPS = c.frameRate?.getCurrentFPS() ?? 0;
      captured.initialFPS = initialFPS;
      captured.finalFPS = 0;
      dispatch(
        Perf.setDegradationReport({
          ...ZERO_DEGRADATION_REPORT,
          averageFrameRateStart: initialFPS,
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
    }

    // Use last buffered sample to avoid capturing stop-logic overhead
    if (status === "completed" && prevStatus === "running") {
      const samples = sampleBufferRef.current.getAllSamples();
      const lastSample = samples.at(-1);
      if (lastSample != null) {
        captured.finalFPS = lastSample.frameRate;
        captured.finalCPU = lastSample.cpuPercent;
        captured.finalGPU = lastSample.gpuPercent;
      } else {
        captured.finalFPS = c.frameRate?.getCurrentFPS() ?? 0;
        captured.finalCPU = c.cpu?.getCpuPercent() ?? null;
        captured.finalGPU = c.gpu?.getGpuPercent() ?? null;
      }
    }

    // Reset when test is reset to idle
    if (status === "idle" && prevStatus !== "idle") {
      captured.initialCPU = null;
      captured.finalCPU = null;
      captured.initialGPU = null;
      captured.finalGPU = null;
      captured.initialFPS = 0;
      captured.finalFPS = 0;
      setLatestSample(null);

      c.longTask?.reset();
      c.network?.reset();
      sampleBufferRef.current.reset();
    }
  }, [status, dispatch]);

  // Run final analysis when test completes
  useEffect(() => {
    if (status !== "completed") return;
    const captured = capturedRef.current;
    runAnalysis(captured.finalFPS, captured.finalCPU, captured.finalGPU);
  }, [status, runAnalysis]);

  const handleStart = () => dispatch(Perf.start(undefined));
  const handleStop = () => dispatch(Perf.stop());
  const handleReset = () => dispatch(Perf.reset());

  const buttonConfigs: Record<
    string,
    {
      icon: ReactElement;
      text: string;
      handler: () => void;
      variant: "filled" | "outlined";
    }
  > = {
    idle: {
      icon: <Icon.Play />,
      text: "Start",
      handler: handleStart,
      variant: "filled",
    },
    running: {
      icon: <Icon.Pause />,
      text: formatTime(elapsedSeconds),
      handler: handleStop,
      variant: "outlined",
    },
    completed: {
      icon: <Icon.Refresh />,
      text: formatTime(elapsedSeconds),
      handler: handleReset,
      variant: "outlined",
    },
    error: {
      icon: <Icon.Refresh />,
      text: "Reset",
      handler: handleReset,
      variant: "outlined",
    },
  };
  const btn = buttonConfigs[status] ?? buttonConfigs.idle;

  return (
    <Flex.Box y className="console-perf-dashboard" grow>
      <Header.Header level="h4">
        <Header.Title>Performance</Header.Title>
        <Header.Actions>
          <Button.Button variant={btn.variant} size="tiny" onClick={btn.handler}>
            {btn.icon}
            {btn.text}
          </Button.Button>
        </Header.Actions>
      </Header.Header>

      {/* Live Metrics Section */}
      <Section title="Live">
        <MetricRow
          label="Frame Rate"
          value={`${liveMetrics.frameRate.toFixed(1)} FPS`}
          status={getThresholdStatus(liveMetrics.frameRate, 30, 15, true)}
          tooltip="Current frames per second. Target is 60 FPS. Warning below 30, error below 15."
        />
        <MetricRow
          label="Memory"
          value={formatMB(liveMetrics.heapUsedMB)}
          tooltip="Current process memory usage."
        />
        <MetricRow
          label="CPU"
          value={formatPercent(liveMetrics.cpuPercent)}
          status={getThresholdStatus(liveMetrics.cpuPercent, 50, 80)}
          tooltip="Current CPU usage. Warning above 50%, error above 80%."
        />
        <MetricRow
          label="GPU"
          value={formatPercent(liveMetrics.gpuPercent)}
          status={getThresholdStatus(liveMetrics.gpuPercent, 80, 95)}
          tooltip="Current GPU utilization. Warning above 80%, error above 95%."
        />
      </Section>

      {/* Changes Section */}
      <Section
        title="Changes"
        secondaryText={status !== "idle" ? formatTime(elapsedSeconds) : undefined}
      >
        <MetricRow
          label="FPS Drop"
          value={`${degradationReport.frameRateDegradationPercent.toFixed(1)}%`}
          status={getThresholdStatus(
            degradationReport.frameRateDegradationPercent,
            10,
            15,
          )}
          tooltip="Percentage decrease in frame rate. Warning at >10%, error at >15%."
        />
        <MetricRow
          label="Heap Growth"
          value={`${leakReport.heapGrowthPercent.toFixed(1)}%`}
          status={getWarningStatus(leakReport.heapGrowthPercent, 20)}
          tooltip="Percentage change in heap memory. Warning at >20%."
        />
        <MetricRow
          label="Avg / Peak CPU"
          value={formatAvgPeak(cpuReport.avgPercent, cpuReport.peakPercent)}
          status={getAvgPeakStatus(cpuReport.avgPercent, cpuReport.peakPercent, 50, 80)}
          tooltip="Average and peak CPU usage. Warning if avg >50% or peak >80%."
        />
        <MetricRow
          label="Avg / Peak GPU"
          value={formatAvgPeak(gpuReport.avgPercent, gpuReport.peakPercent)}
          status={getAvgPeakStatus(gpuReport.avgPercent, gpuReport.peakPercent, 80, 95)}
          tooltip="Average and peak GPU usage. Warning if avg >80% or peak >95%."
        />
      </Section>

      {/* Start/End Section */}
      <Section title="Start / End">
        <MetricRow
          label="FPS"
          value={formatStartEnd(
            degradationReport.averageFrameRateStart,
            degradationReport.averageFrameRateEnd,
            true,
          )}
          status={getWarningStatus(degradationReport.frameRateDegradationPercent, 15)}
          tooltip="Frame rate at the start vs end of the test."
        />
        <MetricRow
          label="Heap (MB)"
          value={formatStartEnd(leakReport.heapStartMB, leakReport.heapEndMB, true)}
          status={getWarningStatus(leakReport.heapGrowthPercent, 20)}
          tooltip="Heap memory usage at start vs end of test."
        />
        <MetricRow
          label="CPU (%)"
          value={formatStartEnd(cpuReport.startPercent, cpuReport.endPercent)}
          tooltip="CPU usage at start vs end of test."
        />
        <MetricRow
          label="GPU (%)"
          value={formatStartEnd(gpuReport.startPercent, gpuReport.endPercent)}
          tooltip="GPU usage at start vs end of test."
        />
      </Section>

      {/* Tasks Section */}
      <Section title="Tasks">
        <MetricRow
          label="Long Tasks"
          value={latestSample?.longTaskCount.toString() ?? "—"}
          status={
            latestSample != null && latestSample.longTaskCount > 5
              ? "warning"
              : undefined
          }
          tooltip="JavaScript tasks blocking the main thread for >50ms. High counts indicate UI jank."
        />
        <MetricRow
          label="Network Requests"
          value={latestSample?.networkRequestCount.toString() ?? "—"}
          tooltip="Fetch/XHR requests made since last sample."
        />
      </Section>

      {status === "error" && (
        <Text.Text status="error" className="console-perf-error">
          An error occurred during performance testing
        </Text.Text>
      )}
    </Flex.Box>
  );
};
