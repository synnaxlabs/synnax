// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactNode } from "react";

import { type MetricCategory, type MetricType } from "@/perf/constants";

export type Status = "success" | "warning" | "error" | "info" | undefined;

export interface LiveMetrics {
  frameRate: number | null;
  cpuPercent: number | null;
  gpuPercent: number | null;
  heapUsedMB: number | null;
  heapTotalMB: number | null;
  networkRequestCount: number | null;
  longTaskCount: number | null;
  totalNetworkRequests: number | null;
  totalLongTasks: number | null;
  consoleLogCount: number | null;
  totalConsoleLogs: number | null;
}

export interface MetricDef {
  key: string;
  type: MetricType;
  category: MetricCategory;
  getValue: () => string;
  getStatus?: () => Status;
  tooltip: string;
  label?: string;
}

export interface SectionConfig {
  key: string;
  title: string;
  secondaryText?: ReactNode;
  secondaryStatus?: Status;
  secondaryTooltip?: string;
  content: ReactNode;
}
