// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  addChannelsToPlotWorkflow,
  closePlotWorkflow,
  createLinePlotWorkflow,
  panZoomPlotWorkflow,
} from "@/perf/workflows/lineplot";
import { createSchematicWorkflow } from "@/perf/workflows/schematic";
import {
  DEFAULT_WORKFLOW_CONFIG,
  type WorkflowConfig,
  type WorkflowContext,
  type WorkflowResult,
  type WorkflowStep,
  type WorkflowType,
} from "@/perf/workflows/types";

export interface WorkflowRunnerCallbacks {
  /** Called when a workflow completes. */
  onWorkflowComplete?: (result: WorkflowResult) => void;
}

/**
 * Executes workflows in a configurable loop for performance testing.
 */
export class WorkflowRunner {
  private context: WorkflowContext;
  private config: WorkflowConfig;
  private callbacks: WorkflowRunnerCallbacks;
  private results: WorkflowResult[] = [];
  private running = false;
  private iterationsCompleted = 0;

  constructor(
    context: WorkflowContext,
    config: Partial<WorkflowConfig> = {},
    callbacks: WorkflowRunnerCallbacks = {},
  ) {
    this.context = context;
    this.config = { ...DEFAULT_WORKFLOW_CONFIG, ...config };
    this.callbacks = callbacks;
  }

  /** Start running workflows. */
  async run(): Promise<WorkflowResult[]> {
    this.running = true;
    this.results = [];
    this.iterationsCompleted = 0;

    const maxIterations =
      this.config.iterations === -1 ? Infinity : this.config.iterations;

    while (this.running && this.iterationsCompleted < maxIterations) {
      await this.runIteration();
      this.iterationsCompleted++;

      if (this.running && this.iterationsCompleted < maxIterations) 
        await this.delay(this.config.delayBetweenIterationsMs);
      
    }

    return this.results;
  }

  /** Stop running workflows. */
  stop(): void {
    this.running = false;
  }

  /** Check if the runner is currently running. */
  isRunning(): boolean {
    return this.running;
  }

  /** Get all results so far. */
  getResults(): WorkflowResult[] {
    return [...this.results];
  }

  private async runIteration(): Promise<void> {
    for (const workflowType of this.config.workflows) {
      if (!this.running) break;

      const workflow = this.getWorkflow(workflowType);
      const result = await this.executeWorkflow(workflow, workflowType);
      this.results.push(result);
      this.callbacks.onWorkflowComplete?.(result);

      if (this.running) 
        await this.delay(this.config.delayBetweenWorkflowsMs);
      
    }
  }

  private async executeWorkflow(
    steps: WorkflowStep[],
    type: WorkflowType,
  ): Promise<WorkflowResult> {
    const startTime = performance.now();
    let error: string | undefined;

    try {
      for (const step of steps) {
        if (!this.running) break;
        await step.execute(this.context);
        if (step.delayAfterMs != null && step.delayAfterMs > 0) 
          await this.delay(step.delayAfterMs);
        
      }
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
      console.error(`Workflow ${type} failed:`, e);
    }

    const endTime = performance.now();
    return {
      workflowType: type,
      startTime,
      endTime,
      durationMs: endTime - startTime,
      error,
    };
  }

  private getWorkflow(type: WorkflowType): WorkflowStep[] {
    switch (type) {
      case "createLinePlot":
        return createLinePlotWorkflow();
      case "addChannelsToPlot":
        return addChannelsToPlotWorkflow();
      case "panZoomPlot":
        return panZoomPlotWorkflow();
      case "createSchematic":
        return createSchematicWorkflow();
      case "closePlot":
        return closePlotWorkflow();
      default:
        throw new Error(`Unknown workflow type: ${type as string}`);
    }
  }

  private delay(ms: number): Promise<void> {
    return new Promise((resolve) => setTimeout(resolve, ms));
  }
}
