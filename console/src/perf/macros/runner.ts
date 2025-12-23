// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Import macro modules to trigger registration
import "@/perf/macros/lineplot";
import "@/perf/macros/schematic";

import { getMacro } from "@/perf/macros/registry";
import {
  DEFAULT_MACRO_CONFIG,
  type MacroConfig,
  type MacroContext,
  type MacroResult,
  type MacroStep,
  type MacroType,
} from "@/perf/macros/types";

export interface MacroRunnerCallbacks {
  /** Called when a macro completes. */
  onMacroComplete?: (result: MacroResult) => void;
}

/**
 * Executes macros in a configurable loop for performance testing.
 */
export class MacroRunner {
  private context: MacroContext;
  private config: MacroConfig;
  private callbacks: MacroRunnerCallbacks;
  private results: MacroResult[] = [];
  private running = false;
  private iterationsCompleted = 0;

  constructor(
    context: MacroContext,
    config: Partial<MacroConfig> = {},
    callbacks: MacroRunnerCallbacks = {},
  ) {
    this.context = context;
    this.config = { ...DEFAULT_MACRO_CONFIG, ...config };
    this.callbacks = callbacks;
  }

  /** Start running macros. */
  async run(): Promise<MacroResult[]> {
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

  /** Stop running macros. */
  stop(): void {
    this.running = false;
  }

  /** Check if the runner is currently running. */
  isRunning(): boolean {
    return this.running;
  }

  /** Get all results so far. */
  getResults(): MacroResult[] {
    return [...this.results];
  }

  private async runIteration(): Promise<void> {
    for (const macroType of this.config.macros) {
      if (!this.running) break;

      const macro = this.getMacroSteps(macroType);
      const result = await this.executeMacro(macro, macroType);
      this.results.push(result);
      this.callbacks.onMacroComplete?.(result);

      if (this.running) await this.delay(this.config.delayBetweenMacrosMs);
    }
  }

  private async executeMacro(
    steps: MacroStep[],
    type: MacroType,
  ): Promise<MacroResult> {
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
      console.error(`Macro ${type} failed:`, e);
    }

    const endTime = performance.now();
    return {
      macroType: type,
      startTime,
      endTime,
      durationMs: endTime - startTime,
      error,
    };
  }

  private getMacroSteps(type: MacroType): MacroStep[] {
    return getMacro(type);
  }

  private delay(ms: number): Promise<void> {
    return new Promise((resolve) => setTimeout(resolve, ms));
  }
}
