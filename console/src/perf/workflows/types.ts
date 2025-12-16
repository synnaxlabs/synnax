// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Dispatch } from "@reduxjs/toolkit";
import { type Synnax } from "@synnaxlabs/client";

import { type Layout } from "@/layout";
import { type RootStore } from "@/store";

/** Types of workflows that can be executed. */
export type WorkflowType =
  | "createLinePlot"
  | "addChannelsToPlot"
  | "panZoomPlot"
  | "createSchematic"
  | "closePlot";

/** Context provided to workflow steps for execution. */
export interface WorkflowContext {
  /** Redux store for state access */
  store: RootStore;
  /** Redux dispatch function */
  dispatch: Dispatch;
  /** Synnax client for data access */
  client: Synnax | null;
  /** Layout placer for creating visualizations */
  placer: Layout.Placer;
  /** Keys of layouts created during this harness run */
  createdLayoutKeys: string[];
  /** Channel keys available for use */
  availableChannelKeys: number[];
}

/** A single step in a workflow. */
export interface WorkflowStep {
  /** Human-readable name of the step */
  name: string;
  /** Execute the step */
  execute: (context: WorkflowContext) => Promise<void>;
  /** Optional delay in ms after this step completes */
  delayAfterMs?: number;
}

/** Configuration for the workflow runner. */
export interface WorkflowConfig {
  /** Workflows to execute in each iteration */
  workflows: WorkflowType[];
  /** Number of iterations (-1 for unlimited) */
  iterations: number;
  /** Delay between workflow iterations in ms */
  delayBetweenIterationsMs: number;
  /** Delay between individual workflows in ms */
  delayBetweenWorkflowsMs: number;
}

/** Default workflow configuration. */
export const DEFAULT_WORKFLOW_CONFIG: WorkflowConfig = {
  workflows: ["createLinePlot", "addChannelsToPlot", "panZoomPlot"],
  iterations: -1,
  delayBetweenIterationsMs: 5000,
  delayBetweenWorkflowsMs: 2000,
};

/** Result of executing a single workflow. */
export interface WorkflowResult {
  /** Type of workflow executed */
  workflowType: WorkflowType;
  /** Start timestamp (performance.now()) */
  startTime: number;
  /** End timestamp (performance.now()) */
  endTime: number;
  /** Duration in milliseconds */
  durationMs: number;
  /** Error message if the workflow failed */
  error?: string;
}
