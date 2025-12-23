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
import {
  MetricTable,
  type MetricTableColumn,
  type MetricTableData,
} from "@/perf/components/MetricTable";
import { Section } from "@/perf/components/Section";
import {
  CATEGORY_LABELS,
  CATEGORY_ORDER,
  METRIC_NAMES,
  METRIC_ORDER,
  type MetricCategory,
  type MetricType,
  THRESHOLDS,
  TYPE_MODE_LABELS,
} from "@/perf/constants";
import { type Aggregates } from "@/perf/metrics/buffer";
import {
  CONSOLE_LOG_TABLE_COLUMNS,
  type ConsoleLogEntry,
  getConsoleLogTableKey,
  getConsoleLogTableTooltip,
} from "@/perf/metrics/console";
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
import { type DisplayStatus, type LiveMetrics, type MetricDef } from "@/perf/ui-types";
import { formatCount } from "@/perf/utils/formatting";
import {
  createFpsMetrics,
  createMemoryMetrics,
  createResourceMetrics,
  type MetricSeverities,
  type ResourceReport,
} from "@/perf/utils/metrics-factory";


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

const NETWORK_TOOLTIP = "Requests per second";
const LONG_TASKS_TOOLTIP = "Tasks blocking main thread >50ms";
const CONSOLE_LOGS_TOOLTIP = "Console messages per second";

/** Metrics section data */
interface MetricSectionData {
  type: "metrics";
  key: string;
  title: string;
  secondaryText?: string;
  secondaryStatus?: DisplayStatus;
  secondaryTooltip?: string;
  metrics: MetricDef[];
}

interface BaseEventSection {
  type: "event";
  title: string;
  secondaryText: string;
  secondaryStatus: DisplayStatus;
  secondaryTooltip: string;
  showTable: boolean;
}

interface NetworkEventSection extends BaseEventSection {
  key: "network";
  tableData: MetricTableData<EndpointStats>;
  columns: MetricTableColumn<EndpointStats>[];
  getKey: (item: EndpointStats, index: number) => string;
  getTooltip: (item: EndpointStats) => string;
}

interface LongTaskEventSection extends BaseEventSection {
  key: "long-tasks";
  tableData: MetricTableData<LongTaskStats>;
  columns: MetricTableColumn<LongTaskStats>[];
  getKey: (item: LongTaskStats, index: number) => string;
  getTooltip?: undefined;
}

interface ConsoleLogEventSection extends BaseEventSection {
  key: "console-logs";
  tableData: MetricTableData<ConsoleLogEntry>;
  columns: MetricTableColumn<ConsoleLogEntry>[];
  getKey: (item: ConsoleLogEntry, index: number) => string;
  getTooltip: (item: ConsoleLogEntry) => string;
}

type EventSectionData = NetworkEventSection | LongTaskEventSection | ConsoleLogEventSection;

type SectionData = MetricSectionData | EventSectionData;

const renderEventTable = (section: EventSectionData): ReactElement => {
  switch (section.key) {
    case "network":
      return (
        <MetricTable
          result={section.tableData}
          columns={section.columns}
          getKey={section.getKey}
          getTooltip={section.getTooltip}
        />
      );
    case "long-tasks":
      return (
        <MetricTable
          result={section.tableData}
          columns={section.columns}
          getKey={section.getKey}
        />
      );
    case "console-logs":
      return (
        <MetricTable
          result={section.tableData}
          columns={section.columns}
          getKey={section.getKey}
          getTooltip={section.getTooltip}
        />
      );
  }
};

export interface MetricSectionsProps {
  groupByType: boolean;
  liveMetrics: LiveMetrics;
  aggregates: Aggregates;
  latestSample: MetricSample | null;
  topEndpoints: MetricTableData<EndpointStats>;
  topLongTasks: MetricTableData<LongTaskStats>;
  topConsoleLogs: MetricTableData<ConsoleLogEntry>;
  fpsReport: {
    changePercent: number | null;
  };
  leakReport: {
    heapGrowthPercent: number | null;
  };
  cpuReport: ResourceReport;
  gpuReport: ResourceReport;

  severities: {
    fps: MetricSeverities;
    cpu: MetricSeverities;
    gpu: MetricSeverities;
    heap: "none" | "warning" | "error";
  };
  status: string;
}

const MetricSectionsImpl = ({
  groupByType,
  liveMetrics,
  aggregates,
  latestSample,
  topEndpoints,
  topLongTasks,
  topConsoleLogs,
  fpsReport,
  leakReport,
  cpuReport,
  gpuReport,
  severities,
  status,
}: MetricSectionsProps): ReactElement => {
  const metrics = useMemo(
    (): MetricDef[] => [
      ...createFpsMetrics(
        liveMetrics.frameRate,
        fpsReport.changePercent,
        aggregates.avgFps != null,
        aggregates.avgFps,
        aggregates.minFps,
        severities.fps,
      ),

      ...createMemoryMetrics(
        liveMetrics.heapUsedMB,
        leakReport.heapGrowthPercent,
        aggregates.minHeap != null,
        aggregates.minHeap,
        aggregates.maxHeap,
        severities.heap,
      ),

      ...createResourceMetrics(
        "cpu",
        liveMetrics.cpuPercent,
        cpuReport,
        THRESHOLDS.cpu,
        THRESHOLDS.cpuChange,
        severities.cpu,
      ),

      ...createResourceMetrics(
        "gpu",
        liveMetrics.gpuPercent,
        gpuReport,
        THRESHOLDS.gpu,
        THRESHOLDS.gpuChange,
        severities.gpu,
      ),
    ],
    [
      liveMetrics,
      fpsReport,
      latestSample,
      aggregates,
      leakReport,
      cpuReport,
      gpuReport,
      severities,
    ],
  );

  const networkStatus: DisplayStatus = undefined;
  const longTasksStatus: DisplayStatus = undefined;
  const consoleLogsStatus: DisplayStatus = undefined;

  const isProfilingActive = status === "running" || status === "paused";

  const getLabel = useCallback(
    (metric: MetricDef): string => {
      if (groupByType) return metric.label ?? TYPE_MODE_LABELS[metric.category];
      return METRIC_NAMES[metric.type];
    },
    [groupByType],
  );

  const sections = useMemo((): SectionData[] => {
    const result: SectionData[] = [];

    if (groupByType) {
      const grouped = groupMetrics(metrics, (m) => m.type, METRIC_ORDER);
      Object.entries(grouped).forEach(([type, typeMetrics]) => {
        const liveMetric = typeMetrics.find((m) => m.category === "live");
        const rowMetrics = typeMetrics.filter((m) => m.category !== "live");
        result.push({
          type: "metrics",
          key: type,
          title: METRIC_NAMES[type as MetricType],
          secondaryText: liveMetric?.getValue(),
          secondaryStatus: liveMetric?.getStatus?.(),
          secondaryTooltip: liveMetric?.tooltip,
          metrics: rowMetrics,
        });
      });
    } else {
      const grouped = groupMetrics(metrics, (m) => m.category, CATEGORY_ORDER);
      Object.entries(grouped).forEach(([category, catMetrics]) => {
        result.push({
          type: "metrics",
          key: category,
          title: CATEGORY_LABELS[category as MetricCategory],
          metrics: catMetrics,
        });
      });
    }

    result.push(
      {
        type: "event",
        key: "network",
        title: "Network",
        secondaryText: isProfilingActive
          ? `${formatCount(liveMetrics.networkRequestCount)} / ${formatCount(liveMetrics.totalNetworkRequests)}`
          : formatCount(liveMetrics.networkRequestCount),
        secondaryStatus: networkStatus,
        secondaryTooltip: NETWORK_TOOLTIP,
        tableData: topEndpoints,
        columns: NETWORK_TABLE_COLUMNS,
        getKey: getNetworkTableKey,
        getTooltip: getNetworkTableTooltip,
        showTable: isProfilingActive,
      },
      {
        type: "event",
        key: "long-tasks",
        title: "Long Tasks",
        secondaryText: isProfilingActive
          ? `${formatCount(liveMetrics.longTaskCount)} / ${formatCount(liveMetrics.totalLongTasks)}`
          : formatCount(liveMetrics.longTaskCount),
        secondaryStatus: longTasksStatus,
        secondaryTooltip: LONG_TASKS_TOOLTIP,
        tableData: topLongTasks,
        columns: LONG_TASK_TABLE_COLUMNS,
        getKey: getLongTaskTableKey,
        showTable: isProfilingActive,
      },
      {
        type: "event",
        key: "console-logs",
        title: "Console Logs",
        secondaryText: isProfilingActive
          ? `${formatCount(liveMetrics.consoleLogCount)} / ${formatCount(liveMetrics.totalConsoleLogs)}`
          : formatCount(liveMetrics.consoleLogCount),
        secondaryStatus: consoleLogsStatus,
        secondaryTooltip: CONSOLE_LOGS_TOOLTIP,
        tableData: topConsoleLogs,
        columns: CONSOLE_LOG_TABLE_COLUMNS,
        getKey: getConsoleLogTableKey,
        getTooltip: getConsoleLogTableTooltip,
        showTable: isProfilingActive,
      },
    );

    return result;
  }, [
    groupByType,
    metrics,
    isProfilingActive,
    liveMetrics.networkRequestCount,
    liveMetrics.longTaskCount,
    liveMetrics.consoleLogCount,
    liveMetrics.totalNetworkRequests,
    liveMetrics.totalLongTasks,
    liveMetrics.totalConsoleLogs,
    networkStatus,
    longTasksStatus,
    consoleLogsStatus,
    topEndpoints,
    topLongTasks,
    topConsoleLogs,
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
          {section.type === "metrics" ? (
            section.metrics.map((metric) => (
              <MetricRow
                key={metric.key}
                label={getLabel(metric)}
                value={metric.getValue()}
                status={metric.getStatus?.()}
                tooltip={metric.tooltip}
              />
            ))
          ) : section.showTable ? (
            renderEventTable(section)
          ) : null}
        </Section>
      ))}
    </>
  );
};

export const MetricSections = memo(MetricSectionsImpl);
