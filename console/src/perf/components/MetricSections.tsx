// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { memo, type ReactElement, useCallback, useMemo } from "react";

import { MetricRow } from "@/perf/components/MetricRow";
import { MetricTable, type MetricTableData } from "@/perf/components/MetricTable";
import { Section } from "@/perf/components/Section";
import {
  CATEGORY_LABELS,
  CATEGORY_ORDER,
  type MetricCategory,
  type MetricType,
  THRESHOLDS,
  TYPE_LABELS,
  TYPE_MODE_LABELS,
  TYPE_ORDER,
} from "@/perf/constants";
import {
  getLongTaskTableKey,
  LONG_TASK_TABLE_COLUMNS,
  type LongTaskStats,
} from "@/perf/metrics/longtasks";
import {
  type EndpointStats,
  getNetworkTableKey,
  getNetworkTableTooltip,
  NETWORK_TABLE_COLUMNS,
} from "@/perf/metrics/network";
import { type MetricSample } from "@/perf/metrics/types";
import {
  type LiveMetrics,
  type MetricDef,
  type SectionConfig,
} from "@/perf/types";
import { formatCount } from "@/perf/utils/formatting";
import {
  createFpsMetrics,
  createMemoryMetrics,
  createResourceMetrics,
  type ResourceReport,
} from "@/perf/utils/metrics-factory";
import { getThresholdStatus } from "@/perf/utils/status";

interface Aggregates {
  avgFps: number | null;
  minFps: number | null;
  avgHeap: number | null;
  peakHeap: number | null;
}

const groupMetrics = <K extends string>(
  metrics: MetricDef[],
  getKey: (m: MetricDef) => K,
  order: K[],
): Record<K, MetricDef[]> => {
  const groups = Object.fromEntries(
    order.map((key) => [key, [] as MetricDef[]]),
  ) as Record<K, MetricDef[]>;
  metrics.forEach((metric) => {
    const key = getKey(metric);
    groups[key]?.push(metric);
  });
  return groups;
};

const NETWORK_TOOLTIP = `Requests per second (warn >${THRESHOLDS.networkRequests.warn}, error >${THRESHOLDS.networkRequests.error})`;
const LONG_TASKS_TOOLTIP = `Tasks blocking main thread >50ms (warn >${THRESHOLDS.longTasks.warn}, error >${THRESHOLDS.longTasks.error})`;

export interface MetricSectionsProps {
  groupByType: boolean;
  liveMetrics: LiveMetrics;
  aggregates: Aggregates;
  latestSample: MetricSample | null;
  topEndpoints: MetricTableData<EndpointStats>;
  topLongTasks: MetricTableData<LongTaskStats>;
  degradationReport: {
    frameRateDegradationPercent: number | null;
  };
  leakReport: {
    heapGrowthPercent: number | null;
  };
  cpuReport: ResourceReport;
  gpuReport: ResourceReport;
  status: string;
}

const MetricSectionsImpl = ({
  groupByType,
  liveMetrics,
  aggregates,
  latestSample,
  topEndpoints,
  topLongTasks,
  degradationReport,
  leakReport,
  cpuReport,
  gpuReport,
  status,
}: MetricSectionsProps): ReactElement => {
  const metrics = useMemo(
    (): MetricDef[] => [
      ...createFpsMetrics(
        liveMetrics.frameRate,
        degradationReport.frameRateDegradationPercent,
        latestSample != null,
        aggregates.avgFps,
        aggregates.minFps,
      ),

      ...createMemoryMetrics(
        liveMetrics.heapUsedMB,
        leakReport.heapGrowthPercent,
        latestSample != null,
        aggregates.avgHeap,
        aggregates.peakHeap,
      ),

      ...createResourceMetrics(
        "cpu",
        liveMetrics.cpuPercent,
        cpuReport,
        THRESHOLDS.cpu,
        THRESHOLDS.cpuChange,
      ),

      ...createResourceMetrics(
        "gpu",
        liveMetrics.gpuPercent,
        gpuReport,
        THRESHOLDS.gpu,
        THRESHOLDS.gpuChange,
      ),
    ],
    [
      liveMetrics,
      degradationReport,
      latestSample,
      aggregates,
      leakReport,
      cpuReport,
      gpuReport,
    ],
  );

  const networkStatus = useMemo(
    () =>
      getThresholdStatus(
        liveMetrics.networkRequestCount,
        THRESHOLDS.networkRequests.warn,
        THRESHOLDS.networkRequests.error,
      ),
    [liveMetrics.networkRequestCount],
  );

  const longTasksStatus = useMemo(
    () =>
      getThresholdStatus(
        liveMetrics.longTaskCount,
        THRESHOLDS.longTasks.warn,
        THRESHOLDS.longTasks.error,
      ),
    [liveMetrics.longTaskCount],
  );

  const getLabel = useCallback(
    (metric: MetricDef): string => {
      if (groupByType) 
        return metric.label ?? TYPE_MODE_LABELS[metric.category];
      
      return TYPE_LABELS[metric.type];
    },
    [groupByType],
  );

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

  const sections = useMemo((): SectionConfig[] => {
    const result: SectionConfig[] = [];
    const isProfilingActive = status === "running" || status === "paused";

    if (groupByType) {
      const grouped = groupMetrics(metrics, (m) => m.type, TYPE_ORDER);
      Object.entries(grouped).forEach(([type, typeMetrics]) => {
        const liveMetric = typeMetrics.find((m) => m.category === "live");
        const rowMetrics = typeMetrics.filter((m) => m.category !== "live");
        result.push({
          key: type,
          title: TYPE_LABELS[type as MetricType],
          secondaryText: liveMetric?.getValue(),
          secondaryStatus: liveMetric?.getStatus?.(),
          secondaryTooltip: liveMetric?.tooltip,
          content: <>{renderMetricRows(rowMetrics)}</>,
        });
      });
    } else {
      const grouped = groupMetrics(metrics, (m) => m.category, CATEGORY_ORDER);
      Object.entries(grouped).forEach(([category, catMetrics]) => {
        result.push({
          key: category,
          title: CATEGORY_LABELS[category as MetricCategory],
          content: <>{renderMetricRows(catMetrics)}</>,
        });
      });
    }

    result.push({
      key: "network",
      title: "Network",
      secondaryText: isProfilingActive
        ? `${formatCount(liveMetrics.networkRequestCount)} / ${formatCount(liveMetrics.totalNetworkRequests)}`
        : formatCount(liveMetrics.networkRequestCount),
      secondaryStatus: networkStatus,
      secondaryTooltip: NETWORK_TOOLTIP,
      content: isProfilingActive ? (
        <MetricTable
          result={topEndpoints}
          columns={NETWORK_TABLE_COLUMNS}
          getKey={getNetworkTableKey}
          getTooltip={getNetworkTableTooltip}
        />
      ) : undefined,
    });

    result.push({
      key: "long-tasks",
      title: "Long Tasks",
      secondaryText: isProfilingActive
        ? `${formatCount(liveMetrics.longTaskCount)} / ${formatCount(liveMetrics.totalLongTasks)}`
        : formatCount(liveMetrics.longTaskCount),
      secondaryStatus: longTasksStatus,
      secondaryTooltip: LONG_TASKS_TOOLTIP,
      content: isProfilingActive ? (
        <MetricTable
          result={topLongTasks}
          columns={LONG_TASK_TABLE_COLUMNS}
          getKey={getLongTaskTableKey}
        />
      ) : undefined,
    });

    return result;
  }, [
    groupByType,
    metrics,
    renderMetricRows,
    liveMetrics.networkRequestCount,
    liveMetrics.longTaskCount,
    liveMetrics.totalNetworkRequests,
    liveMetrics.totalLongTasks,
    networkStatus,
    longTasksStatus,
    topEndpoints,
    topLongTasks,
    status,
  ]);

  return (
    <>
      {sections.map((section) => (
        <Section
          key={section.key}
          title={section.title}
          secondaryText={section.secondaryText}
          secondaryStatus={section.secondaryStatus}
          secondaryTooltip={section.secondaryTooltip}
        >
          {section.content}
        </Section>
      ))}
    </>
  );
};

export const MetricSections = memo(MetricSectionsImpl);
