// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type CpuReport,
  type FpsReport,
  type GpuReport,
  type LeakReport,
} from "@/perf/analyzer/types";
import { type CapturedValues } from "@/perf/hooks/useCapturedValues";
import { type Aggregates } from "@/perf/metrics/buffer";
import { type MetricSample } from "@/perf/metrics/types";

export type Verdict = "Passed" | "Failed";
export type IssueCategory = "fps" | "cpu" | "gpu" | "memory";

export type IssueSeverity = "warning" | "critical";
export interface DetectedIssue {
  category: IssueCategory;
  severity: IssueSeverity;
  message: string;
  value: number;
  threshold: number;
}

export interface ReportSummary {
  verdict: Verdict;
  durationMs: number;
  totalSamples: number;
  issueCount: number;
}

export interface MetricsReport {
  fps: {
    avg: number | null;
    min: number | null;
    max: number | null;
    changePercent: number | null;
  };
  cpu: {
    avg: number | null;
    max: number | null;
    changePercent: number | null;
  };
  gpu: {
    avg: number | null;
    max: number | null;
    changePercent: number | null;
  };
  memory: {
    minHeapMB: number | null;
    maxHeapMB: number | null;
    growthPercent: number | null;
  };
}

export interface AnalysisResults {
  leak: LeakReport;
  fps: FpsReport;
  cpu: CpuReport;
  gpu: GpuReport;
}

export interface CompileInput {
  samples: MetricSample[];
  captured: CapturedValues;
  aggregates: Aggregates;
  analysisResults: AnalysisResults;
  startTime: number;
  endTime: number;
}

export interface FinalReport {
  summary: ReportSummary;
  metrics: MetricsReport;
  issues: DetectedIssue[];
  // Future: trends, recommendations, etc.
}
