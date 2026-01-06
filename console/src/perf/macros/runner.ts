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
  onMacroStart?: (macroType: MacroType, macroIndex: number, iteration: number) => void;
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

  async run(): Promise<MacroResult[]> {
    this.running = true;
    this.results = [];
    this.iterationsCompleted = 0;

    const maxIterations =
      this.config.iterations === -1 ? Infinity : this.config.iterations;

    while (this.running && this.iterationsCompleted < maxIterations) {
      await this.runIteration();
      this.iterationsCompleted++;
    }

    return this.results;
  }

  stop(): void {
    this.running = false;
  }

  isRunning(): boolean {
    return this.running;
  }

  getResults(): MacroResult[] {
    return [...this.results];
  }

  private async runIteration(): Promise<void> {
    for (let i = 0; i < this.config.macros.length; i++) {
      if (!this.running) break;

      const macroType = this.config.macros[i];
      this.callbacks.onMacroStart?.(macroType, i, this.iterationsCompleted);

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
      for (let i = 0; i < steps.length; i++) {
        if (!this.running) break;
        await steps[i].execute(this.context);
        if (this.running && i < steps.length - 1 && this.config.delayBetweenStepsMs > 0)
          await this.delay(this.config.delayBetweenStepsMs);
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
