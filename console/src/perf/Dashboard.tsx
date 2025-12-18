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
  type CpuReport,
  ZERO_CPU_REPORT,
  ZERO_DEGRADATION_REPORT,
  ZERO_GPU_REPORT,
  ZERO_LEAK_REPORT,
} from "@/perf/analyzer/types";
import { type Aggregates, SampleBuffer, ZERO_AGGREGATES } from "@/perf/metrics/buffer";
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

const THRESHOLDS = {
  fps: { warn: 50, error: 28, inverted: true },
  fpsDegradation: { warn: 10, error: 15 },
  cpu: { warn: 25, error: 50 },
  cpuChange: { warn: 20, error: 40 },
  gpu: { warn: 25, error: 50 },
  gpuChange: { warn: 20, error: 40 },
  heapGrowth: { warn: 5, error: 10 },
  longTasks: { warn: 5, error: 10 },
  networkRequests: { warn: 5, error: 10 },
} as const;

const LIVE_DISPLAY_INTERVAL_MS = 1000;
const SAMPLE_INTERVAL_MS = 1000;

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

const getAvgPeakStatus = (
  avg: number | null,
  peak: number | null,
  avgThreshold: number,
  peakThreshold: number,
): Status =>
  (avg ?? 0) > avgThreshold || (peak ?? 0) > peakThreshold ? "warning" : undefined;

const formatPair = (
  first: number | null,
  second: number | null,
  suffix = "",
): string => {
  if (first == null && second == null) return "—";
  const firstStr = first != null ? first.toFixed(1) : "—";
  const secondStr = second != null ? second.toFixed(1) : "—";
  return `${firstStr} / ${secondStr}${suffix}`;
};

const formatDelta = (
  start: number | null,
  end: number | null,
  suffix = "",
): string => {
  if (start == null || end == null) return "—";
  const delta = end - start;
  const sign = delta >= 0 ? "+" : "";
  return `${sign}${delta.toFixed(1)}${suffix}`;
};

const formatPercentChange = (percent: number | null, invertSign = false): string => {
  if (percent == null) return "—";
  const value = invertSign ? -percent : percent;
  const sign = value >= 0 ? "+" : "";
  return `${sign}${value.toFixed(1)}%`;
};

type MetricType = "fps" | "memory" | "cpu" | "gpu" | "tasks";
type MetricCategory = "live" | "change" | "stats";

interface MetricDef {
  key: string;
  type: MetricType;
  category: MetricCategory;
  getValue: () => string;
  getStatus?: () => Status;
  tooltip: string;
}

const TYPE_LABELS: Record<MetricType, string> = {
  fps: "FPS",
  memory: "Memory",
  cpu: "CPU",
  gpu: "GPU",
  tasks: "Tasks",
};

const CATEGORY_LABELS: Record<MetricCategory, string> = {
  live: "Live",
  change: "Change",
  stats: "Stats",
};

const TYPE_MODE_LABELS: Record<MetricCategory, string> = {
  live: "Live",
  change: "Change",
  stats: "Avg / Min",
};

const TYPE_ORDER: MetricType[] = ["fps", "memory", "cpu", "gpu", "tasks"];
const CATEGORY_ORDER: MetricCategory[] = ["live", "change", "stats"];

const groupMetrics = <K extends string>(
  metrics: MetricDef[],
  getKey: (m: MetricDef) => K,
  order: K[],
): Map<K, MetricDef[]> => {
  const groups = new Map<K, MetricDef[]>();
  for (const key of order) groups.set(key, []);
  for (const metric of metrics) {
    const key = getKey(metric);
    groups.get(key)?.push(metric);
  }
  return groups;
};

type ResourceReport = Omit<CpuReport, "detected">;

const createFpsMetrics = (
  liveValue: () => number,
  degradationPercent: () => number | null,
  hasData: () => boolean,
  avgFps: () => number | null,
  minFps: () => number | null,
): MetricDef[] => [
  {
    key: "fps-live",
    type: "fps",
    category: "live",
    getValue: () => liveValue().toFixed(1),
    getStatus: () =>
      getThresholdStatus(
        liveValue(),
        THRESHOLDS.fps.warn,
        THRESHOLDS.fps.error,
        THRESHOLDS.fps.inverted,
      ),
    tooltip: `Current FPS (warn <${THRESHOLDS.fps.warn}, error <${THRESHOLDS.fps.error})`,
  },
  {
    key: "fps-change",
    type: "fps",
    category: "change",
    getValue: () => formatPercentChange(hasData() ? degradationPercent() : null, true),
    getStatus: () =>
      getThresholdStatus(
        degradationPercent(),
        THRESHOLDS.fpsDegradation.warn,
        THRESHOLDS.fpsDegradation.error,
      ),
    tooltip: `FPS change during session (warn >${THRESHOLDS.fpsDegradation.warn}%, error >${THRESHOLDS.fpsDegradation.error}%)`,
  },
  {
    key: "fps-stats",
    type: "fps",
    category: "stats",
    getValue: () => formatPair(avgFps(), minFps()),
    getStatus: () =>
      getThresholdStatus(
        minFps(),
        THRESHOLDS.fps.warn,
        THRESHOLDS.fps.error,
        THRESHOLDS.fps.inverted,
      ),
    tooltip: `Average and minimum FPS (warn <${THRESHOLDS.fps.warn}, error <${THRESHOLDS.fps.error})`,
  },
];

/** Factory to create Memory metric definitions */
const createMemoryMetrics = (
  liveHeap: () => number | null,
  growthPercent: () => number | null,
  hasData: () => boolean,
  avgHeap: () => number | null,
  peakHeap: () => number | null,
): MetricDef[] => [
  {
    key: "memory-live",
    type: "memory",
    category: "live",
    getValue: () => formatMB(liveHeap()),
    tooltip: "Current heap usage",
  },
  {
    key: "memory-change",
    type: "memory",
    category: "change",
    getValue: () => formatPercentChange(hasData() ? growthPercent() : null),
    getStatus: () =>
      getThresholdStatus(
        growthPercent(),
        THRESHOLDS.heapGrowth.warn,
        THRESHOLDS.heapGrowth.error,
      ),
    tooltip: `Heap change during session (warn >${THRESHOLDS.heapGrowth.warn}%, error >${THRESHOLDS.heapGrowth.error}%)`,
  },
  {
    key: "memory-stats",
    type: "memory",
    category: "stats",
    getValue: () => formatPair(avgHeap(), peakHeap(), " MB"),
    tooltip: "Average and peak heap usage",
  },
];

const createResourceMetrics = (
  type: "cpu" | "gpu",
  liveValue: () => number | null,
  report: ResourceReport,
  thresholds: { warn: number; error: number },
  changeThresholds: { warn: number; error: number },
): MetricDef[] => {
  const label = type.toUpperCase();
  return [
    {
      key: `${type}-live`,
      type,
      category: "live",
      getValue: () => formatPercent(liveValue()),
      getStatus: () => getThresholdStatus(liveValue(), thresholds.warn, thresholds.error),
      tooltip: `Current ${label} (warn >${thresholds.warn}%, error >${thresholds.error}%)`,
    },
    {
      key: `${type}-change`,
      type,
      category: "change",
      getValue: () => formatDelta(report.startPercent, report.endPercent, "%"),
      getStatus: () => {
        if (report.startPercent == null || report.endPercent == null) return undefined;
        const delta = Math.abs(report.endPercent - report.startPercent);
        return getThresholdStatus(delta, changeThresholds.warn, changeThresholds.error);
      },
      tooltip: `${label} change during session (warn >${changeThresholds.warn}%, error >${changeThresholds.error}%)`,
    },
    {
      key: `${type}-stats`,
      type,
      category: "stats",
      getValue: () => formatPair(report.avgPercent, report.peakPercent, "%"),
      getStatus: () =>
        getAvgPeakStatus(
          report.avgPercent,
          report.peakPercent,
          thresholds.warn,
          thresholds.error,
        ),
      tooltip: `Average and peak ${label} (warn >${thresholds.warn}%, error >${thresholds.error}%)`,
    },
  ];
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
  secondaryStatus?: Status;
  defaultOpen?: boolean;
  children: ReactNode;
}

const Section = memo(({
  title,
  secondaryText,
  secondaryStatus,
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
          <Text.Text
            level="small"
            className="console-perf-section-header-value"
            color={secondaryStatus != null ? STATUS_COLORS[secondaryStatus] : 9}
          >
            {secondaryText}
          </Text.Text>
        )}
      </Flex.Box>
      {open && children}
    </Flex.Box>
  );
});
Section.displayName = "Section";

interface MetricSectionsProps {
  groupByType: boolean;
  liveMetrics: LiveMetrics;
  aggregates: Aggregates;
  latestSample: MetricSample | null;
  degradationReport: ReturnType<typeof useSelectDegradationReport>;
  leakReport: ReturnType<typeof useSelectLeakReport>;
  cpuReport: ReturnType<typeof useSelectCpuReport>;
  gpuReport: ReturnType<typeof useSelectGpuReport>;
}

const MetricSections = memo(
  ({
    groupByType,
    liveMetrics,
    aggregates,
    latestSample,
    degradationReport,
    leakReport,
    cpuReport,
    gpuReport,
  }: MetricSectionsProps): ReactElement => {
    // Build metric definitions on each render. Memoization would be ineffective here
    // because props update every LIVE_DISPLAY_INTERVAL_MS, invalidating any cache.
    const metrics: MetricDef[] = [
      ...createFpsMetrics(
        () => liveMetrics.frameRate,
        () => degradationReport.frameRateDegradationPercent,
        () => latestSample != null,
        () => aggregates.avgFps,
        () => aggregates.minFps,
      ),

      ...createMemoryMetrics(
        () => liveMetrics.heapUsedMB,
        () => leakReport.heapGrowthPercent,
        () => latestSample != null,
        () => aggregates.avgHeap,
        () => aggregates.peakHeap,
      ),

      ...createResourceMetrics(
        "cpu",
        () => liveMetrics.cpuPercent,
        cpuReport,
        THRESHOLDS.cpu,
        THRESHOLDS.cpuChange,
      ),

      ...createResourceMetrics(
        "gpu",
        () => liveMetrics.gpuPercent,
        gpuReport,
        THRESHOLDS.gpu,
        THRESHOLDS.gpuChange,
      ),

      {
        key: "tasks-long",
        type: "tasks",
        category: "live",
        getValue: () => latestSample?.longTaskCount.toString() ?? "—",
        getStatus: () =>
          getThresholdStatus(
            latestSample?.longTaskCount ?? null,
            THRESHOLDS.longTasks.warn,
            THRESHOLDS.longTasks.error,
          ),
        tooltip: `Tasks blocking main thread >50ms (warn >${THRESHOLDS.longTasks.warn}, error >${THRESHOLDS.longTasks.error})`,
      },
      {
        key: "tasks-network",
        type: "tasks",
        category: "live",
        getValue: () => latestSample?.networkRequestCount.toString() ?? "—",
        getStatus: () =>
          getThresholdStatus(
            latestSample?.networkRequestCount ?? null,
            THRESHOLDS.networkRequests.warn,
            THRESHOLDS.networkRequests.error,
          ),
        tooltip: `Network requests (warn >${THRESHOLDS.networkRequests.warn}, error >${THRESHOLDS.networkRequests.error})`,
      },
    ];

    const getLabel = useCallback(
      (metric: MetricDef): string => {
        if (metric.type === "tasks")
          return metric.key === "tasks-long" ? "Long Tasks" : "Network";
        // Grouped by type (FPS, Memory, etc.) - show category as label
        if (groupByType) return TYPE_MODE_LABELS[metric.category];
        // Grouped by category (Live, Change, Stats) - show type as label
        return TYPE_LABELS[metric.type];
      },
      [groupByType],
    );

    // Helper to render metric rows
    const renderMetricRows = useCallback(
      (metricsToRender: MetricDef[]) =>
        metricsToRender.map((metric) => (
          <MetricRow
            key={metric.key}
            label={getLabel(metric)}
            value={metric.getValue()}
            status={metric.getStatus?.()}
            tooltip={metric.tooltip}
          />
        )),
      [getLabel],
    );

    if (groupByType) {
      const grouped = groupMetrics(metrics, (m) => m.type, TYPE_ORDER);
      return (
        <>
          {Array.from(grouped.entries()).map(([type, typeMetrics]) => {
            const isTaskType = type === "tasks";
            const liveMetric = isTaskType
              ? null
              : typeMetrics.find((m) => m.category === "live");
            const rowMetrics = isTaskType
              ? typeMetrics
              : typeMetrics.filter((m) => m.category !== "live");
            return (
              <Section
                key={type}
                title={TYPE_LABELS[type]}
                secondaryText={liveMetric?.getValue()}
                secondaryStatus={liveMetric?.getStatus?.()}
              >
                {renderMetricRows(rowMetrics)}
              </Section>
            );
          })}
        </>
      );
    }

    const taskMetrics = metrics.filter((m) => m.type === "tasks");
    const nonTaskMetrics = metrics.filter((m) => m.type !== "tasks");
    const grouped = groupMetrics(nonTaskMetrics, (m) => m.category, CATEGORY_ORDER);

    return (
      <>
        {Array.from(grouped.entries()).map(([category, catMetrics]) => (
          <Section key={category} title={CATEGORY_LABELS[category]}>
            {renderMetricRows(catMetrics)}
          </Section>
        ))}
        <Section title="Tasks">{renderMetricRows(taskMetrics)}</Section>
      </>
    );
  },
);
MetricSections.displayName = "MetricSections";

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

  // Track aggregates from buffer for stats display
  const [aggregates, setAggregates] = useState<Aggregates>(ZERO_AGGREGATES);

  // Grouping mode: "time" (Live, Changes) or "type" (Live, Delta, Stats)
  const [groupByType, setGroupByType] = useState(false);

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
    }, LIVE_DISPLAY_INTERVAL_MS);

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
        setAggregates(sampleBufferRef.current.getAggregates());
        runAnalysis(sample.frameRate, sample.cpuPercent, sample.gpuPercent);
      }, SAMPLE_INTERVAL_MS);
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
      setAggregates(ZERO_AGGREGATES);

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

  const handleStart = useCallback(() => dispatch(Perf.start(undefined)), [dispatch]);
  const handleStop = useCallback(() => dispatch(Perf.stop()), [dispatch]);
  const handleReset = useCallback(() => dispatch(Perf.reset()), [dispatch]);

  // Auto-start profiling when dashboard opens
  const hasAutoStarted = useRef(false);
  useEffect(() => {
    if (!hasAutoStarted.current && status === "idle") {
      hasAutoStarted.current = true;
      dispatch(Perf.start(undefined));
    }
  }, [status, dispatch]);

  const buttonConfigs = useMemo(
    () => ({
      idle: {
        icon: <Icon.Play />,
        text: "Start",
        handler: handleStart,
        variant: "filled" as const,
      },
      running: {
        icon: <Icon.Pause />,
        text: formatTime(elapsedSeconds),
        handler: handleStop,
        variant: "outlined" as const,
      },
      paused: {
        icon: <Icon.Play />,
        text: formatTime(elapsedSeconds),
        handler: handleStart,
        variant: "outlined" as const,
      },
      completed: {
        icon: <Icon.Refresh />,
        text: formatTime(elapsedSeconds),
        handler: handleReset,
        variant: "outlined" as const,
      },
      error: {
        icon: <Icon.Refresh />,
        text: "Reset",
        handler: handleReset,
        variant: "outlined" as const,
      },
    }),
    [elapsedSeconds, handleStart, handleStop, handleReset],
  );
  const btn = buttonConfigs[status];

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
        degradationReport={degradationReport}
        leakReport={leakReport}
        cpuReport={cpuReport}
        gpuReport={gpuReport}
      />

      {status === "error" && (
        <Text.Text status="error" className="console-perf-error">
          An error occurred during performance testing
        </Text.Text>
      )}
    </Flex.Box>
  );
};
