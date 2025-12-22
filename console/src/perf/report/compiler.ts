// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { math } from "@synnaxlabs/x";

import { THRESHOLDS } from "@/perf/constants";
import {
  type CompileInput,
  type DetectedIssue,
  type FinalReport,
  type MetricsReport,
  type Verdict,
} from "@/perf/report/types";

const detectIssues = (input: CompileInput): DetectedIssue[] => {
  const issues: DetectedIssue[] = [];
  const { aggregates, analysisResults } = input;

  // FPS issues (inverted: lower is worse)
  if (aggregates.avgFps != null) 
    if (aggregates.avgFps < THRESHOLDS.fps.error) 
      issues.push({
        category: "fps",
        severity: "critical",
        message: `Average FPS (${math.roundTo(aggregates.avgFps, 1)}) is below ${THRESHOLDS.fps.error}`,
        value: aggregates.avgFps,
        threshold: THRESHOLDS.fps.error,
      });
     else if (aggregates.avgFps < THRESHOLDS.fps.warn) 
      issues.push({
        category: "fps",
        severity: "warning",
        message: `Average FPS (${math.roundTo(aggregates.avgFps, 1)}) is below ${THRESHOLDS.fps.warn}`,
        value: aggregates.avgFps,
        threshold: THRESHOLDS.fps.warn,
      });
    
  

  const hasFpsIssue =
    analysisResults.fps.peakSeverity !== "none" ||
    analysisResults.fps.avgSeverity !== "none";

  if (hasFpsIssue && analysisResults.fps.changePercent > THRESHOLDS.fpsChange.error)
    issues.push({
      category: "fps",
      severity: "critical",
      message: `FPS dropped ${math.roundTo(analysisResults.fps.changePercent, 1)}% during session`,
      value: analysisResults.fps.changePercent,
      threshold: THRESHOLDS.fpsChange.error,
    });
  else if (hasFpsIssue && analysisResults.fps.changePercent > THRESHOLDS.fpsChange.warn)
    issues.push({
      category: "fps",
      severity: "warning",
      message: `FPS dropped ${math.roundTo(analysisResults.fps.changePercent, 1)}% during session`,
      value: analysisResults.fps.changePercent,
      threshold: THRESHOLDS.fpsChange.warn,
    });
  

  if (aggregates.avgCpu != null) 
    if (aggregates.avgCpu > THRESHOLDS.cpu.error) 
      issues.push({
        category: "cpu",
        severity: "critical",
        message: `Average CPU (${math.roundTo(aggregates.avgCpu, 1)}%) exceeds ${THRESHOLDS.cpu.error}%`,
        value: aggregates.avgCpu,
        threshold: THRESHOLDS.cpu.error,
      });
     else if (aggregates.avgCpu > THRESHOLDS.cpu.warn) 
      issues.push({
        category: "cpu",
        severity: "warning",
        message: `Average CPU (${math.roundTo(aggregates.avgCpu, 1)}%) exceeds ${THRESHOLDS.cpu.warn}%`,
        value: aggregates.avgCpu,
        threshold: THRESHOLDS.cpu.warn,
      });
    
  

  if (aggregates.avgGpu != null) 
    if (aggregates.avgGpu > THRESHOLDS.gpu.error) 
      issues.push({
        category: "gpu",
        severity: "critical",
        message: `Average GPU (${math.roundTo(aggregates.avgGpu, 1)}%) exceeds ${THRESHOLDS.gpu.error}%`,
        value: aggregates.avgGpu,
        threshold: THRESHOLDS.gpu.error,
      });
     else if (aggregates.avgGpu > THRESHOLDS.gpu.warn) 
      issues.push({
        category: "gpu",
        severity: "warning",
        message: `Average GPU (${math.roundTo(aggregates.avgGpu, 1)}%) exceeds ${THRESHOLDS.gpu.warn}%`,
        value: aggregates.avgGpu,
        threshold: THRESHOLDS.gpu.warn,
      });
    
  

  if (analysisResults.leak.severity !== "none") {
    const growth = analysisResults.leak.heapGrowthPercent;
    if (growth > THRESHOLDS.heapGrowth.error) 
      issues.push({
        category: "memory",
        severity: "critical",
        message: `Memory grew ${math.roundTo(growth, 1)}% - potential leak detected`,
        value: growth,
        threshold: THRESHOLDS.heapGrowth.error,
      });
     else if (growth > THRESHOLDS.heapGrowth.warn) 
      issues.push({
        category: "memory",
        severity: "warning",
        message: `Memory grew ${math.roundTo(growth, 1)}%`,
        value: growth,
        threshold: THRESHOLDS.heapGrowth.warn,
      });
    
  }

  return issues;
};

const determineVerdict = (issues: DetectedIssue[]): Verdict => {
  const hasCritical = issues.some((issue) => issue.severity === "critical");
  return hasCritical ? "Failed" : "Passed";
};

const buildMetricsReport = (input: CompileInput): MetricsReport => {
  const { aggregates, analysisResults, captured } = input;

  const fpsChange =
    captured.initialFPS != null && captured.finalFPS != null && captured.initialFPS > 0
      ? ((captured.initialFPS - captured.finalFPS) / captured.initialFPS) * 100
      : null;

  const cpuChange =
    captured.initialCPU != null && captured.finalCPU != null && captured.initialCPU > 0
      ? ((captured.finalCPU - captured.initialCPU) / captured.initialCPU) * 100
      : null;

  const gpuChange =
    captured.initialGPU != null && captured.finalGPU != null && captured.initialGPU > 0
      ? ((captured.finalGPU - captured.initialGPU) / captured.initialGPU) * 100
      : null;

  return {
    fps: {
      avg: aggregates.avgFps != null ? math.roundTo(aggregates.avgFps, 1) : null,
      min: aggregates.minFps != null ? math.roundTo(aggregates.minFps, 1) : null,
      max: aggregates.maxFps != null ? math.roundTo(aggregates.maxFps, 1) : null,
      changePercent: fpsChange != null ? math.roundTo(fpsChange, 1) : null,
    },
    cpu: {
      avg: aggregates.avgCpu != null ? math.roundTo(aggregates.avgCpu, 1) : null,
      max: aggregates.maxCpu != null ? math.roundTo(aggregates.maxCpu, 1) : null,
      changePercent: cpuChange != null ? math.roundTo(cpuChange, 1) : null,
    },
    gpu: {
      avg: aggregates.avgGpu != null ? math.roundTo(aggregates.avgGpu, 1) : null,
      max: aggregates.maxGpu != null ? math.roundTo(aggregates.maxGpu, 1) : null,
      changePercent: gpuChange != null ? math.roundTo(gpuChange, 1) : null,
    },
    memory: {
      minHeapMB: aggregates.minHeap != null ? math.roundTo(aggregates.minHeap, 1) : null,
      maxHeapMB: aggregates.maxHeap != null ? math.roundTo(aggregates.maxHeap, 1) : null,
      growthPercent:
        analysisResults.leak.heapGrowthPercent != null
          ? math.roundTo(analysisResults.leak.heapGrowthPercent, 1)
          : null,
    },
  };
};

/**
 * Compiles a final report from profiling session data.
 *
 * This is a pure function that:
 * 1. Detects issues based on thresholds
 * 2. Determines verdict (PASS/FAIL)
 * 3. Builds metrics summary
 *
 * The verdict determines which label to apply to the range.
 */
export const compileReport = (input: CompileInput): FinalReport => {
  const issues = detectIssues(input);
  const verdict = determineVerdict(issues);
  const metrics = buildMetricsReport(input);

  return {
    summary: {
      verdict,
      durationMs: input.endTime - input.startTime,
      totalSamples: input.samples.length,
      issueCount: issues.length,
    },
    metrics,
    issues,
  };
};
