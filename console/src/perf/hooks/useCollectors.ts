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
import { FrameRateCollector } from "@/perf/metrics/framerate";
import { GpuCollector } from "@/perf/metrics/gpu";
import { HeapCollector } from "@/perf/metrics/heap";
import { LongTaskCollector, type LongTaskStats } from "@/perf/metrics/longtasks";
import { type EndpointStats, NetworkCollector } from "@/perf/metrics/network";
import { type MetricSample } from "@/perf/metrics/types";
import { type HarnessStatus } from "@/perf/slice";
import { type LiveMetrics } from "@/perf/types";

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
  c.frameRate,
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

interface CollectorsState {
  cpu: CpuCollector | null;
  gpu: GpuCollector | null;
  frameRate: FrameRateCollector | null;
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

export interface UseCollectorsOptions {
  /** Current profiling status - determines when to record samples */
  status: HarnessStatus;
  /** Callback invoked when a sample is collected during "running" status */
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
  const [liveMetrics, setLiveMetrics] = useState<LiveMetrics>(ZERO_LIVE_METRICS);

  const onSampleRef = useRef(onSample);
  onSampleRef.current = onSample;

  const sampleBufferRef = useRef(new SampleBuffer());

  const [latestSample, setLatestSample] = useState<MetricSample | null>(null);
  const latestSampleRef = useRef<MetricSample | null>(null);

  const [aggregates, setAggregates] = useState<Aggregates>(ZERO_AGGREGATES);

  const [tableData, setTableData] = useState<TableDataState>(ZERO_TABLE_DATA);

  const collectorsRef = useRef<CollectorsState>({
    cpu: null,
    gpu: null,
    frameRate: null,
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
      frameRate: c.frameRate?.getCurrentFPS() ?? null,
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
    setTableData(ZERO_TABLE_DATA);
  }, []);

  const resetBuffer = useCallback(() => {
    latestSampleRef.current = null;
    setLatestSample(null);
    setAggregates(ZERO_AGGREGATES);
    sampleBufferRef.current.reset();
  }, []);

  useEffect(() => {
    const c = collectorsRef.current;
    c.cpu = new CpuCollector();
    c.gpu = new GpuCollector();
    c.frameRate = new FrameRateCollector();
    c.heap = new HeapCollector();
    c.longTask = new LongTaskCollector();
    c.network = new NetworkCollector();
    c.console = new ConsoleCollector();

    getAllCollectors(c).forEach((col) => col?.start());

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
        consoleLogCount: sample.consoleLogCount,
        totalConsoleLogs: c.console?.getTotalCount() ?? null,
      });

      if (status === "running") {
        sampleBufferRef.current.push(sample);
        setAggregates(sampleBufferRef.current.getAggregates());
        setTableData({
          endpoints: c.network?.getTopEndpoints() ?? emptyTableData(),
          longTasks: c.longTask?.getTopLongTasks() ?? emptyTableData(),
          consoleLogs: c.console?.getTopLogs() ?? emptyTableData(),
        });
        onSampleRef.current?.(sample, sampleBufferRef.current);
      }
    }, SAMPLE_INTERVAL_MS);

    return () => {
      clearInterval(updateInterval);
      getAllCollectors(c).forEach((col) => col?.stop());
    };
  }, [collectSample, status]);

  return {
    liveMetrics,
    tableData,
    aggregates,
    latestSample,
    collectors: collectorsRef,
    sampleBuffer: sampleBufferRef,
    resetEventCollectors,
    resetTableData,
    resetBuffer,
    collectSample,
  };
};
