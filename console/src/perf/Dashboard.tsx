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
import { type ReactElement, useCallback,useEffect, useRef, useState } from "react";
import { useDispatch } from "react-redux";

import { type Layout } from "@/layout";
import { DegradationDetector } from "@/perf/analyzer/degradation";
import { LeakDetector } from "@/perf/analyzer/leak-detector";
import { ZERO_DEGRADATION_REPORT, ZERO_LEAK_REPORT } from "@/perf/analyzer/types";
import { CpuCollector } from "@/perf/metrics/cpu";
import { FrameRateCollector } from "@/perf/metrics/framerate";
import { HeapCollector } from "@/perf/metrics/heap";
import { LongTaskCollector } from "@/perf/metrics/longtasks";
import { NetworkCollector } from "@/perf/metrics/network";
import { type MetricSample } from "@/perf/metrics/types";
import {
  useSelectDegradationReport,
  useSelectElapsedSeconds,
  useSelectLatestSample,
  useSelectLeakReport,
  useSelectSamples,
  useSelectStatus,
} from "@/perf/selectors";
import * as Perf from "@/perf/slice";

interface LiveMetrics {
  frameRate: number;
  cpuPercent: number | null;
  heapUsedMB: number | null;
  heapTotalMB: number | null;
  longTaskCount: number;
}

/** Default display value for percentage metrics with no data. */
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
  const latestSample = useSelectLatestSample();
  const elapsedSeconds = useSelectElapsedSeconds();
  const leakReport = useSelectLeakReport();
  const degradationReport = useSelectDegradationReport();
  const samples = useSelectSamples();

  // Live metrics state (updated while dashboard is open)
  const [liveMetrics, setLiveMetrics] = useState<LiveMetrics>({
    frameRate: 0,
    cpuPercent: null,
    heapUsedMB: null,
    heapTotalMB: null,
    longTaskCount: 0,
  });

  // Single set of collectors - shared between live display and test recording
  const cpuRef = useRef<CpuCollector | null>(null);
  const frameRateRef = useRef<FrameRateCollector | null>(null);
  const heapRef = useRef<HeapCollector | null>(null);
  const longTaskRef = useRef<LongTaskCollector | null>(null);
  const networkRef = useRef<NetworkCollector | null>(null);

  // Analysis refs
  const leakDetectorRef = useRef(new LeakDetector());
  const degradationDetectorRef = useRef(new DegradationDetector());

  // Ref to access latest samples without restarting intervals
  const samplesRef = useRef(samples);
  samplesRef.current = samples;

  // Sample collection interval ref (only active during test)
  const sampleIntervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  // Collect a sample from the shared collectors
  const collectSample = useCallback((): MetricSample => ({
      timestamp: performance.now(),
      cpuPercent: cpuRef.current?.getCpuPercent() ?? null,
      heapUsedMB: heapRef.current?.getHeapUsedMB() ?? null,
      heapTotalMB: heapRef.current?.getHeapTotalMB() ?? null,
      frameRate: frameRateRef.current?.getCurrentFPS() ?? 0,
      longTaskCount: longTaskRef.current?.getCountSinceLastSample() ?? 0,
      longTaskDurationMs: longTaskRef.current?.getDurationSinceLastSample() ?? 0,
      networkRequestCount: networkRef.current?.getCountSinceLastSample() ?? 0,
    }), []);

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
    // Note: network collector only starts during test (invasive - uses monkey-patching)

    // Update live metrics display every 500ms
    const liveInterval = setInterval(() => {
      setLiveMetrics({
        frameRate: frameRateRef.current?.getCurrentFPS() ?? 0,
        cpuPercent: cpuRef.current?.getCpuPercent() ?? null,
        heapUsedMB: heapRef.current?.getHeapUsedMB() ?? null,
        heapTotalMB: heapRef.current?.getHeapTotalMB() ?? null,
        longTaskCount: longTaskRef.current?.getCountInWindow() ?? 0,
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
      // Start network collector only during test (invasive - uses monkey-patching)
      networkRef.current?.start();

      // Start recording samples to Redux
      sampleIntervalRef.current = setInterval(() => {
        const sample = collectSample();
        dispatch(Perf.addSample(sample));

        // Capture heap snapshot
        const snapshot = heapRef.current?.captureSnapshot();
        if (snapshot != null) 
          dispatch(Perf.addHeapSnapshot(snapshot));
        
      }, 1000);
    } else if (sampleIntervalRef.current != null) {
      // Stop recording
      clearInterval(sampleIntervalRef.current);
      sampleIntervalRef.current = null;

      // Stop network collector
      networkRef.current?.stop();
    }

    return () => {
      if (sampleIntervalRef.current != null) {
        clearInterval(sampleIntervalRef.current);
        sampleIntervalRef.current = null;
      }
    };
  }, [status, dispatch, collectSample]);

  // Store the initial and final values when test starts/stops
  const initialFPSRef = useRef<number>(0);
  const finalFPSRef = useRef<number>(0);
  const initialCPURef = useRef<number | null>(null);
  const finalCPURef = useRef<number | null>(null);
  const prevStatusRef = useRef<string>(status);

  // CPU analysis state (computed from samples)
  const [cpuAnalysis, setCpuAnalysis] = useState({
    avgPercent: null as number | null,
    peakPercent: null as number | null,
    startPercent: null as number | null,
    endPercent: null as number | null,
  });

  // Capture initial FPS/heap/CPU when test starts, final values when test stops
  useEffect(() => {
    const prevStatus = prevStatusRef.current;
    prevStatusRef.current = status;

    // Capture initial FPS, heap, and CPU when test starts
    if (status === "running" && prevStatus !== "running") {
      const initialFPS = frameRateRef.current?.getCurrentFPS() ?? 0;
      initialFPSRef.current = initialFPS;
      finalFPSRef.current = 0; // End remains 0 until test completes
      dispatch(
        Perf.setDegradationReport({
          ...ZERO_DEGRADATION_REPORT,
          averageFrameRateStart: initialFPS,
        }),
      );

      // Capture initial heap
      const initialHeap = heapRef.current?.getHeapUsedMB() ?? 0;
      dispatch(
        Perf.setLeakReport({
          ...ZERO_LEAK_REPORT,
          heapStartMB: math.roundTo(initialHeap),
        }),
      );

      // Capture initial CPU
      const initialCPU = cpuRef.current?.getCpuPercent() ?? null;
      initialCPURef.current = initialCPU;
      finalCPURef.current = null;
      setCpuAnalysis({
        avgPercent: null,
        peakPercent: null,
        startPercent: initialCPU != null ? math.roundTo(initialCPU) : null,
        endPercent: null,
      });
    }

    // Capture final FPS and CPU when test stops
    if (status === "completed" && prevStatus === "running") {
      const finalFPS = frameRateRef.current?.getCurrentFPS() ?? 0;
      finalFPSRef.current = finalFPS;

      const finalCPU = cpuRef.current?.getCpuPercent() ?? null;
      finalCPURef.current = finalCPU;
    }

    // Reset CPU analysis when test is reset to idle
    if (status === "idle" && prevStatus !== "idle") {
      initialCPURef.current = null;
      finalCPURef.current = null;
      initialFPSRef.current = 0;
      finalFPSRef.current = 0;
      setCpuAnalysis({
        avgPercent: null,
        peakPercent: null,
        startPercent: null,
        endPercent: null,
      });
    }
  }, [status]);

  // Update analysis reports periodically during test
  useEffect(() => {
    if (status !== "running") return;

    const interval = setInterval(() => {
      const currentSamples = samplesRef.current;
      if (currentSamples.length < 2) return;

      const leakResult = leakDetectorRef.current.analyze(
        currentSamples
          .filter((s) => s.heapUsedMB != null)
          .map((s) => ({
            timestamp: s.timestamp,
            heapUsedMB: s.heapUsedMB!,
            heapTotalMB: s.heapTotalMB!,
          })),
      );

      // Get long task analysis, but preserve initial FPS and use current live FPS
      const degradationResult = degradationDetectorRef.current.analyze(currentSamples);
      const startFPS = initialFPSRef.current;
      const currentFPS = frameRateRef.current?.getCurrentFPS() ?? 0;
      const fpsDrop = startFPS > 0 ? ((startFPS - currentFPS) / startFPS) * 100 : 0;

      dispatch(Perf.setLeakReport(leakResult));
      dispatch(
        Perf.setDegradationReport({
          ...degradationResult,
          averageFrameRateStart: math.roundTo(startFPS),
          averageFrameRateEnd: math.roundTo(currentFPS),
          frameRateDegradationPercent: math.roundTo(fpsDrop, 2),
          detected: fpsDrop > 15,
        }),
      );

      // Compute CPU analysis from samples
      const cpuValues = currentSamples
        .map((s) => s.cpuPercent)
        .filter((v): v is number => v != null);
      if (cpuValues.length > 0) {
        const avgCpu = math.average(cpuValues);
        const peakCpu = Math.max(...cpuValues);
        const currentCPU = cpuRef.current?.getCpuPercent() ?? null;
        setCpuAnalysis({
          avgPercent: math.roundTo(avgCpu),
          peakPercent: math.roundTo(peakCpu),
          startPercent: initialCPURef.current != null ? math.roundTo(initialCPURef.current) : null,
          endPercent: currentCPU != null ? math.roundTo(currentCPU) : null,
        });
      }
    }, 5000);

    return () => clearInterval(interval);
  }, [status, dispatch]);

  // Run final analysis when test completes
  useEffect(() => {
    if (status !== "completed" || samples.length < 2) return;

    const leakResult = leakDetectorRef.current.analyze(
      samples
        .filter((s) => s.heapUsedMB != null)
        .map((s) => ({
          timestamp: s.timestamp,
          heapUsedMB: s.heapUsedMB!,
          heapTotalMB: s.heapTotalMB!,
        })),
    );

    // Get long task analysis from DegradationDetector, but preserve our captured FPS values
    const degradationResult = degradationDetectorRef.current.analyze(samples);

    // Use preserved FPS values instead of calculated ones
    const startFPS = initialFPSRef.current;
    const endFPS = finalFPSRef.current;
    const fpsDrop = startFPS > 0 ? ((startFPS - endFPS) / startFPS) * 100 : 0;

    dispatch(Perf.setLeakReport(leakResult));
    dispatch(
      Perf.setDegradationReport({
        ...degradationResult,
        averageFrameRateStart: math.roundTo(startFPS),
        averageFrameRateEnd: math.roundTo(endFPS),
        frameRateDegradationPercent: math.roundTo(fpsDrop, 2),
        detected: fpsDrop > 15,
      }),
    );

    // Compute final CPU analysis
    const cpuValues = samples
      .map((s) => s.cpuPercent)
      .filter((v): v is number => v != null);
    if (cpuValues.length > 0) {
      const avgCpu = math.average(cpuValues);
      const peakCpu = Math.max(...cpuValues);
      setCpuAnalysis({
        avgPercent: math.roundTo(avgCpu),
        peakPercent: math.roundTo(peakCpu),
        startPercent: initialCPURef.current != null ? math.roundTo(initialCPURef.current) : null,
        endPercent: finalCPURef.current != null ? math.roundTo(finalCPURef.current) : null,
      });
    }
  }, [status, samples, dispatch]);

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
          value={liveMetrics.longTaskCount.toString()}
          status={liveMetrics.longTaskCount > 10 ? "warning" : undefined}
          tooltip="JavaScript tasks blocking the main thread for >50ms in the last 10 minutes. High counts indicate UI jank."
        />
      </Flex.Box>

      <Header.Header level="h4" className="console-perf-analysis-header">
        <Header.Title>Analysis</Header.Title>
        <Header.Actions>
          {status === "idle" && (
            <Button.Button variant="filled" size="small" onClick={handleStart}>
              <Icon.Play />
            </Button.Button>
          )}
          {status === "running" && (
            <Button.Button variant="outlined" size="small" onClick={handleStop}>
              <Icon.Pause />
            </Button.Button>
          )}
          {(status === "completed" || status === "error") && (
            <Button.Button variant="outlined" size="small" onClick={handleReset}>
              <Icon.Refresh />
            </Button.Button>
          )}
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
          value={
            cpuAnalysis.avgPercent != null
              ? cpuAnalysis.avgPercent > 50 || (cpuAnalysis.peakPercent ?? 0) > 80
                ? "DETECTED"
                : "None"
              : "None"
          }
          status={
            cpuAnalysis.avgPercent != null
              ? cpuAnalysis.avgPercent > 50 || (cpuAnalysis.peakPercent ?? 0) > 80
                ? "error"
                : "success"
              : "success"
          }
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
            cpuAnalysis.avgPercent != null && cpuAnalysis.peakPercent != null
              ? `${cpuAnalysis.avgPercent.toFixed(1)} / ${cpuAnalysis.peakPercent.toFixed(1)}%`
              : ZERO_PERCENT_DEFAULT
          }
          status={
            cpuAnalysis.avgPercent != null && cpuAnalysis.avgPercent > 50
              ? "warning"
              : cpuAnalysis.peakPercent != null && cpuAnalysis.peakPercent > 80
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
          tooltip="Average frame rate at the start vs end of the test. Compares first 10% and last 10% of samples."
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
            cpuAnalysis.startPercent != null && cpuAnalysis.endPercent != null
              ? `${cpuAnalysis.startPercent.toFixed(1)} / ${cpuAnalysis.endPercent.toFixed(1)}%`
              : cpuAnalysis.startPercent != null
                ? `${cpuAnalysis.startPercent.toFixed(1)} / —%`
                : "— / —%"
          }
          tooltip="CPU usage at start vs end of test."
        />
        <MetricCard
          label="Network Requests"
          value={latestSample?.networkRequestCount.toString() ?? "—"}
          tooltip="Number of fetch/XHR requests made since last sample (only tracked during test). High counts may indicate excessive API polling."
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
