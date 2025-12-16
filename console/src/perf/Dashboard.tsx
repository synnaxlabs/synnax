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
import { LeakDetector } from "@/perf/analyzer/leak-detector";
import {
  ZERO_CPU_REPORT,
  ZERO_DEGRADATION_REPORT,
  ZERO_LEAK_REPORT,
} from "@/perf/analyzer/types";
import { SampleBuffer } from "@/perf/metrics/buffer";
import { CpuCollector } from "@/perf/metrics/cpu";
import { FrameRateCollector } from "@/perf/metrics/framerate";
import { HeapCollector } from "@/perf/metrics/heap";
import { LongTaskCollector } from "@/perf/metrics/longtasks";
import { NetworkCollector } from "@/perf/metrics/network";
import { type MetricSample } from "@/perf/metrics/types";
import {
  useSelectCpuReport,
  useSelectDegradationReport,
  useSelectElapsedSeconds,
  useSelectLeakReport,
  useSelectStatus,
} from "@/perf/selectors";
import * as Perf from "@/perf/slice";

interface LiveMetrics {
  frameRate: number;
  cpuPercent: number | null;
  heapUsedMB: number | null;
  heapTotalMB: number | null;
  longTaskCount: number | null;
}

const ZERO_PERCENT_DEFAULT = "0.0%";

/** Format seconds as MM:SS. */
const formatTime = (seconds: number): string => {
  const mins = Math.floor(seconds / 60);
  const secs = Math.floor(seconds % 60);
  return `${mins.toString().padStart(2, "0")}:${secs.toString().padStart(2, "0")}`;
};

interface MetricCardProps {
  label: string;
  value: string;
  status?: "success" | "warning" | "error" | "info";
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

export const Dashboard: Layout.Renderer = ({ layoutKey: _layoutKey }): ReactElement => {
  const dispatch = useDispatch();
  const status = useSelectStatus();
  const elapsedSeconds = useSelectElapsedSeconds();
  const leakReport = useSelectLeakReport();
  const degradationReport = useSelectDegradationReport();
  const cpuReport = useSelectCpuReport();

  // Live metrics state (updated while dashboard is open)
  const [liveMetrics, setLiveMetrics] = useState<LiveMetrics>({
    frameRate: 0,
    cpuPercent: null,
    heapUsedMB: null,
    heapTotalMB: null,
    longTaskCount: null,
  });

  // Pre-allocated sample buffer (memory allocated on mount, not during test)
  const sampleBufferRef = useRef(new SampleBuffer());

  // Track latest sample for network requests display
  const [latestSample, setLatestSample] = useState<MetricSample | null>(null);

  // Single set of collectors - shared between live display and test recording
  const cpuRef = useRef<CpuCollector | null>(null);
  const frameRateRef = useRef<FrameRateCollector | null>(null);
  const heapRef = useRef<HeapCollector | null>(null);
  const longTaskRef = useRef<LongTaskCollector | null>(null);
  const networkRef = useRef<NetworkCollector | null>(null);

  // Analysis refs
  const leakDetectorRef = useRef(new LeakDetector());
  const degradationDetectorRef = useRef(new DegradationDetector());
  const cpuAnalyzerRef = useRef(new CpuAnalyzer());

  // Sample collection interval ref (only active during test)
  const sampleIntervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  // Store the initial and final values when test starts/stops
  const initialFPSRef = useRef<number>(0);
  const finalFPSRef = useRef<number>(0);
  const initialCPURef = useRef<number | null>(null);
  const finalCPURef = useRef<number | null>(null);
  const prevStatusRef = useRef<string>(status);

  // Helper to run analysis and dispatch results (used by both periodic and final effects)
  const runAnalysis = useCallback(
    (endFPS: number, endCPU: number | null) => {
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

      dispatch(Perf.setLeakReport(leakResult));
      dispatch(Perf.setDegradationReport(degradationResult));
      dispatch(Perf.setCpuReport(cpuResult));
    },
    [dispatch],
  );

  // Collect a sample from the shared collectors
  const collectSample = useCallback(
    (): MetricSample => ({
      timestamp: performance.now(),
      cpuPercent: cpuRef.current?.getCpuPercent() ?? null,
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
    frameRateRef.current = new FrameRateCollector();
    heapRef.current = new HeapCollector();
    longTaskRef.current = new LongTaskCollector();
    networkRef.current = new NetworkCollector();

    cpuRef.current.start();
    frameRateRef.current.start();
    heapRef.current.start();
    longTaskRef.current.start();

    // Not available on macOS/Linux WebKit.
    const longTasksSupported = LongTaskCollector.isAvailable();

    // Update live metrics display every 500ms
    const liveInterval = setInterval(() => {
      setLiveMetrics({
        frameRate: frameRateRef.current?.getCurrentFPS() ?? 0,
        cpuPercent: cpuRef.current?.getCpuPercent() ?? null,
        heapUsedMB: heapRef.current?.getHeapUsedMB() ?? null,
        heapTotalMB: heapRef.current?.getHeapTotalMB() ?? null,
        longTaskCount: longTasksSupported
          ? (longTaskRef.current?.getCountInWindow() ?? 0)
          : null,
      });
    }, 500);

    return () => {
      clearInterval(liveInterval);
      cpuRef.current?.stop();
      frameRateRef.current?.stop();
      heapRef.current?.stop();
      longTaskRef.current?.stop();
      networkRef.current?.stop();
    };
  }, []);

  // Start/stop sample recording based on test status
  useEffect(() => {
    if (status === "running") {
      networkRef.current?.start();

      // Start recording samples to pre-allocated buffer
      sampleIntervalRef.current = setInterval(() => {
        const sample = collectSample();
        sampleBufferRef.current.push(sample);
        setLatestSample(sample);
      }, 1000);
    } else if (sampleIntervalRef.current != null) {
      // Stop recording
      clearInterval(sampleIntervalRef.current);
      sampleIntervalRef.current = null;
      networkRef.current?.stop();
    }

    return () => {
      if (sampleIntervalRef.current != null) {
        clearInterval(sampleIntervalRef.current);
        sampleIntervalRef.current = null;
      }
    };
  }, [status, collectSample]);

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
    }

    // Use last buffered sample to avoid capturing stop-logic overhead
    if (status === "completed" && prevStatus === "running") {
      const samples = sampleBufferRef.current.getAllSamples();
      const lastSample = samples.at(-1);
      if (lastSample != null) {
        finalFPSRef.current = lastSample.frameRate;
        finalCPURef.current = lastSample.cpuPercent;
      } else {
        finalFPSRef.current = frameRateRef.current?.getCurrentFPS() ?? 0;
        finalCPURef.current = cpuRef.current?.getCpuPercent() ?? null;
      }
    }

    // Reset when test is reset to idle
    if (status === "idle" && prevStatus !== "idle") {
      initialCPURef.current = null;
      finalCPURef.current = null;
      initialFPSRef.current = 0;
      finalFPSRef.current = 0;
      setLatestSample(null);

      cpuRef.current?.reset();
      frameRateRef.current?.reset();
      heapRef.current?.reset();
      longTaskRef.current?.reset();
      networkRef.current?.reset();
      sampleBufferRef.current.reset();
    }
  }, [status, dispatch]);

  // Update analysis reports periodically during test
  useEffect(() => {
    if (status !== "running") return;

    const interval = setInterval(() => {
      const currentFPS = frameRateRef.current?.getCurrentFPS() ?? 0;
      const currentCPU = cpuRef.current?.getCpuPercent() ?? null;
      runAnalysis(currentFPS, currentCPU);
    }, 5000);

    return () => clearInterval(interval);
  }, [status, runAnalysis]);

  // Run final analysis when test completes
  useEffect(() => {
    if (status !== "completed") return;
    runAnalysis(finalFPSRef.current, finalCPURef.current);
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
          status={
            liveMetrics.frameRate < 15
              ? "error"
              : liveMetrics.frameRate < 30
                ? "warning"
                : undefined
          }
          tooltip="Current frames per second measured via requestAnimationFrame. Target is 60 FPS. Warning below 30, error below 15."
        />
        <MetricCard
          label="Memory"
          value={
            liveMetrics.heapUsedMB != null
              ? `${liveMetrics.heapUsedMB.toFixed(1)} MB`
              : "N/A"
          }
          tooltip="Current process memory usage."
        />
        <MetricCard
          label="CPU"
          value={
            liveMetrics.cpuPercent != null
              ? `${liveMetrics.cpuPercent.toFixed(1)}%`
              : "N/A"
          }
          status={
            liveMetrics.cpuPercent != null && liveMetrics.cpuPercent > 80
              ? "error"
              : liveMetrics.cpuPercent != null && liveMetrics.cpuPercent > 50
                ? "warning"
                : undefined
          }
          tooltip="Current process CPU usage percentage. Warning above 50%, error above 80%. Not available in browser."
        />
        <MetricCard
          label="Long Tasks"
          value={
            liveMetrics.longTaskCount != null
              ? liveMetrics.longTaskCount.toString()
              : "—"
          }
          status={
            liveMetrics.longTaskCount != null && liveMetrics.longTaskCount > 10
              ? "warning"
              : undefined
          }
          tooltip={
            liveMetrics.longTaskCount != null
              ? "JavaScript tasks blocking the main thread for >50ms in the last 10 minutes. High counts indicate UI jank."
              : "Long Tasks API not supported on this platform (WebKit/Safari)."
          }
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
          </Button.Button>
        </Header.Actions>
      </Header.Header>

      {/* Row 1: Status indicators */}
      <Flex.Box x wrap>
        <MetricCard
          label="Degradation"
          value={degradationReport.detected ? "DETECTED" : "None"}
          status={degradationReport.detected ? "error" : "success"}
          tooltip="Frame rate degradation over time. Detected when FPS drops >15% from start to end of test."
        />
        <MetricCard
          label="Memory Leak"
          value={leakReport.detected ? "DETECTED" : "None"}
          status={leakReport.detected ? "error" : "success"}
          tooltip="Potential memory leak detected via linear regression on heap samples. Detected when heap grows >20% with consistent upward trend."
        />
        <MetricCard
          label="High CPU"
          value={cpuReport.detected ? "DETECTED" : "None"}
          status={cpuReport.detected ? "error" : "success"}
          tooltip="High CPU usage. Detected when average CPU >50% or peak CPU >80%."
        />
        <MetricCard
          label="Status"
          value={status.charAt(0).toUpperCase() + status.slice(1)}
        />
      </Flex.Box>

      {/* Row 2: Quantitative changes */}
      <Flex.Box x wrap className="console-perf-metric-row">
        <MetricCard
          label="FPS Drop"
          value={`${degradationReport.frameRateDegradationPercent.toFixed(1)}%`}
          status={
            degradationReport.frameRateDegradationPercent > 15
              ? "error"
              : degradationReport.frameRateDegradationPercent > 10
                ? "warning"
                : undefined
          }
          tooltip="Percentage decrease in frame rate. Warning at >10%, error at >15%."
        />
        <MetricCard
          label="Heap Growth"
          value={`${leakReport.heapGrowthPercent.toFixed(1)}%`}
          status={leakReport.heapGrowthPercent > 20 ? "warning" : undefined}
          tooltip="Percentage change in heap memory from start to end of test. Warning at >20%."
        />
        <MetricCard
          label="Avg / Peak CPU"
          value={
            cpuReport.avgPercent != null && cpuReport.peakPercent != null
              ? `${cpuReport.avgPercent.toFixed(1)} / ${cpuReport.peakPercent.toFixed(1)}%`
              : ZERO_PERCENT_DEFAULT
          }
          status={
            cpuReport.avgPercent != null && cpuReport.avgPercent > 50
              ? "warning"
              : cpuReport.peakPercent != null && cpuReport.peakPercent > 80
                ? "warning"
                : undefined
          }
          tooltip="Average and peak CPU usage during the test. Warning if avg >50% or peak >80%."
        />
        <MetricCard label="Elapsed Time" value={formatTime(elapsedSeconds)} />
      </Flex.Box>

      {/* Row 3: Supporting details */}
      <Flex.Box x wrap className="console-perf-metric-row">
        <MetricCard
          label="FPS Start / End"
          value={
            degradationReport.averageFrameRateStart === 0 &&
            degradationReport.averageFrameRateEnd === 0
              ? "— / —"
              : `${degradationReport.averageFrameRateStart.toFixed(1)} / ${degradationReport.averageFrameRateEnd === 0 ? "—" : degradationReport.averageFrameRateEnd.toFixed(1)}`
          }
          status={
            degradationReport.frameRateDegradationPercent > 15 ? "warning" : undefined
          }
          tooltip="Frame rate at the start vs end of the test."
        />
        <MetricCard
          label="Heap Start / End"
          value={
            leakReport.heapStartMB === 0 && leakReport.heapEndMB === 0
              ? "— / — MB"
              : `${leakReport.heapStartMB.toFixed(1)} / ${leakReport.heapEndMB === 0 ? "—" : leakReport.heapEndMB.toFixed(1)} MB`
          }
          status={leakReport.heapGrowthPercent > 20 ? "warning" : undefined}
          tooltip="Heap memory usage at start vs end of test. Warning at >20% growth."
        />
        <MetricCard
          label="CPU Start / End"
          value={
            cpuReport.startPercent != null && cpuReport.endPercent != null
              ? `${cpuReport.startPercent.toFixed(1)} / ${cpuReport.endPercent.toFixed(1)}%`
              : cpuReport.startPercent != null
                ? `${cpuReport.startPercent.toFixed(1)} / —%`
                : "— / —%"
          }
          tooltip="CPU usage at start vs end of test."
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
