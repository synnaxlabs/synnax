// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback, useEffect, useRef, useState } from "react";

import { type MetricTableData } from "@/perf/components/MetricTable";
import { SAMPLE_INTERVAL_MS } from "@/perf/constants";
import { type Aggregates, SampleBuffer, ZERO_AGGREGATES } from "@/perf/metrics/buffer";
import { ConsoleCollector, type ConsoleLogEntry } from "@/perf/metrics/console";
import { CpuCollector } from "@/perf/metrics/cpu";
import { FpsCollector } from "@/perf/metrics/fps";
import { GpuCollector } from "@/perf/metrics/gpu";
import { HeapCollector } from "@/perf/metrics/heap";
import { LongTaskCollector, type LongTaskStats } from "@/perf/metrics/longtasks";
import { type EndpointStats, NetworkCollector } from "@/perf/metrics/network";
import { type MetricSample } from "@/perf/metrics/types";
import { type HarnessStatus } from "@/perf/slice";
import { type LiveMetrics } from "@/perf/ui-types";

interface Collector {
  start(): void;
  stop(): void;
}

interface ResettableCollector extends Collector {
  reset(): void;
}

const getAllCollectors = (c: CollectorsState): (Collector | null)[] => [
  c.cpu,
  c.gpu,
  c.fps,
  c.heap,
  c.longTask,
  c.network,
  c.console,
];

const getResettableCollectors = (c: CollectorsState): (ResettableCollector | null)[] => [
  c.longTask,
  c.network,
  c.console,
];

export interface CollectorsState {
  cpu: CpuCollector | null;
  gpu: GpuCollector | null;
  fps: FpsCollector | null;
  heap: HeapCollector | null;
  longTask: LongTaskCollector | null;
  network: NetworkCollector | null;
  console: ConsoleCollector | null;
}

interface TableDataState {
  endpoints: MetricTableData<EndpointStats>;
  longTasks: MetricTableData<LongTaskStats>;
  consoleLogs: MetricTableData<ConsoleLogEntry>;
}

const emptyTableData = <T,>(): MetricTableData<T> => ({
  data: [],
  total: 0,
  truncated: false,
});

const ZERO_TABLE_DATA: TableDataState = {
  endpoints: emptyTableData<EndpointStats>(),
  longTasks: emptyTableData<LongTaskStats>(),
  consoleLogs: emptyTableData<ConsoleLogEntry>(),
};

const ZERO_LIVE_METRICS: LiveMetrics = {
  frameRate: null,
  cpuPercent: null,
  gpuPercent: null,
  heapUsedMB: null,
  heapTotalMB: null,
  networkRequestCount: null,
  longTaskCount: null,
  totalNetworkRequests: null,
  totalLongTasks: null,
  consoleLogCount: null,
  totalConsoleLogs: null,
};

/** Combined state to enable single setState call per interval tick. */
interface CollectorDataState {
  liveMetrics: LiveMetrics;
  latestSample: MetricSample | null;
  aggregates: Aggregates;
  tableData: TableDataState;
}

const ZERO_COLLECTOR_DATA: CollectorDataState = {
  liveMetrics: ZERO_LIVE_METRICS,
  latestSample: null,
  aggregates: ZERO_AGGREGATES,
  tableData: ZERO_TABLE_DATA,
};

export interface UseCollectorsOptions {
  status: HarnessStatus;
  onSample?: (sample: MetricSample, sampleBuffer: SampleBuffer) => void;
}

export interface UseCollectorsResult {
  liveMetrics: LiveMetrics;
  tableData: TableDataState;
  aggregates: Aggregates;
  latestSample: MetricSample | null;
  collectors: React.RefObject<CollectorsState>;
  sampleBuffer: React.RefObject<SampleBuffer>;
  resetEventCollectors: () => void;
  resetTableData: () => void;
  resetBuffer: () => void;
  collectSample: () => MetricSample;
}

/**
 * Custom hook that manages collector lifecycle and data collection.
 *
 * Encapsulates:
 * - Collector instantiation and lifecycle (start/stop)
 * - The setInterval update loop
 * - State management for liveMetrics, tableData, aggregates, latestSample
 */
export const useCollectors = ({
  status,
  onSample,
}: UseCollectorsOptions): UseCollectorsResult => {
  const [data, setData] = useState<CollectorDataState>(ZERO_COLLECTOR_DATA);

  const onSampleRef = useRef(onSample);
  onSampleRef.current = onSample;

  // Use ref for status to avoid recreating collectors on status changes.
  // The interval callback reads from this ref instead of closing over status.
  const statusRef = useRef(status);
  statusRef.current = status;

  const sampleBufferRef = useRef(new SampleBuffer());
  const latestSampleRef = useRef<MetricSample | null>(null);

  const collectorsRef = useRef<CollectorsState>({
    cpu: null,
    gpu: null,
    fps: null,
    heap: null,
    longTask: null,
    network: null,
    console: null,
  });

  const collectSample = useCallback((): MetricSample => {
    const c = collectorsRef.current;
    return {
      timestamp: performance.now(),
      cpuPercent: c.cpu?.getCpuPercent() ?? null,
      gpuPercent: c.gpu?.getGpuPercent() ?? null,
      heapUsedMB: c.heap?.getHeapUsedMB() ?? null,
      heapTotalMB: c.heap?.getHeapTotalMB() ?? null,
      frameRate: c.fps?.getCurrentFPS() ?? null,
      longTaskCount: c.longTask?.getCountSinceLastSample() ?? 0,
      longTaskDurationMs: c.longTask?.getDurationSinceLastSample() ?? 0,
      networkRequestCount: c.network?.getCountSinceLastSample() ?? 0,
      consoleLogCount: c.console?.getCountSinceLastSample() ?? 0,
    };
  }, []);

  const resetEventCollectors = useCallback(() => {
    getResettableCollectors(collectorsRef.current).forEach((col) => col?.reset());
  }, []);

  const resetTableData = useCallback(() => {
    setData((prev) => ({ ...prev, tableData: ZERO_TABLE_DATA }));
  }, []);

  const resetBuffer = useCallback(() => {
    latestSampleRef.current = null;
    setData((prev) => ({
      ...prev,
      latestSample: null,
      aggregates: ZERO_AGGREGATES,
    }));
    sampleBufferRef.current.reset();
  }, []);

  useEffect(() => {
    const c = collectorsRef.current;
    c.cpu = new CpuCollector();
    c.gpu = new GpuCollector();
    c.fps = new FpsCollector();
    c.heap = new HeapCollector();
    c.longTask = new LongTaskCollector();
    c.network = new NetworkCollector();
    c.console = new ConsoleCollector();

    getAllCollectors(c).forEach((col) => col?.start());

    const updateInterval = setInterval(() => {
      const sample = collectSample();
      latestSampleRef.current = sample;

      const liveMetrics: LiveMetrics = {
        frameRate: c.fps?.getCurrentFPS() ?? null,
        cpuPercent: c.cpu?.getCpuPercent() ?? null,
        gpuPercent: c.gpu?.getGpuPercent() ?? null,
        heapUsedMB: c.heap?.getHeapUsedMB() ?? null,
        heapTotalMB: c.heap?.getHeapTotalMB() ?? null,
        networkRequestCount: sample.networkRequestCount,
        longTaskCount: sample.longTaskCount,
        totalNetworkRequests: c.network?.getTotalCount() ?? null,
        totalLongTasks: c.longTask?.getTotalCount() ?? null,
        consoleLogCount: sample.consoleLogCount,
        totalConsoleLogs: c.console?.getTotalCount() ?? null,
      };

      if (statusRef.current === "running") {
        sampleBufferRef.current.push(sample);
        setData({
          latestSample: sample,
          liveMetrics,
          aggregates: sampleBufferRef.current.getAggregates(),
          tableData: {
            endpoints: c.network?.getTopEndpoints() ?? emptyTableData(),
            longTasks: c.longTask?.getTopLongTasks() ?? emptyTableData(),
            consoleLogs: c.console?.getTopLogs() ?? emptyTableData(),
          },
        });
        onSampleRef.current?.(sample, sampleBufferRef.current);
      } else 
        setData((prev) => ({ ...prev, latestSample: sample, liveMetrics }));
      
    }, SAMPLE_INTERVAL_MS);

    return () => {
      clearInterval(updateInterval);
      getAllCollectors(c).forEach((col) => col?.stop());
    };
  }, [collectSample]);

  // Update console collector's capturing state based on profiling status.
  // This minimizes overhead when profiler panel is open but not actively profiling.
  useEffect(() => {
    collectorsRef.current.console?.setCapturing(status === "running");
  }, [status]);

  return {
    liveMetrics: data.liveMetrics,
    tableData: data.tableData,
    aggregates: data.aggregates,
    latestSample: data.latestSample,
    collectors: collectorsRef,
    sampleBuffer: sampleBufferRef,
    resetEventCollectors,
    resetTableData,
    resetBuffer,
    collectSample,
  };
};
