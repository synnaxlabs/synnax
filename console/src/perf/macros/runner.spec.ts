// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { beforeEach, describe, expect, it, vi } from "vitest";

import {
  createMacroType,
  type MacroCategory,
  registerMacro,
} from "@/perf/macros/registry";
import { MacroRunner } from "@/perf/macros/runner";
import { type MacroContext, type MacroStep } from "@/perf/macros/types";

const TEST_MACRO = createMacroType("testMacro");
const FAILING_MACRO = createMacroType("failingMacro");

const createMockContext = (): MacroContext =>
  ({
    store: {} as any,
    dispatch: vi.fn(),
    client: null,
    placer: {} as any,
    createdLayoutKeys: [],
    availableChannelKeys: [],
  }) as MacroContext;

const createMockSteps = (count = 2): MacroStep[] =>
  Array.from({ length: count }, (_, i) => ({
    name: `Step ${i + 1}`,
    execute: vi.fn().mockResolvedValue(undefined),
  }));

describe("MacroRunner", () => {
  beforeEach(() => {
    registerMacro({
      type: TEST_MACRO,
      name: "Test Macro",
      description: "A test macro",
      category: "general" as MacroCategory,
      factory: () => createMockSteps(),
    });

    registerMacro({
      type: FAILING_MACRO,
      name: "Failing Macro",
      description: "A macro that fails",
      category: "general" as MacroCategory,
      factory: () => [
        { name: "Fail", execute: vi.fn().mockRejectedValue(new Error("Test error")) },
      ],
    });
  });

  describe("run", () => {
    it("should execute macros for the configured number of iterations", async () => {
      const context = createMockContext();
      const runner = new MacroRunner(context, {
        macros: [TEST_MACRO],
        iterations: 3,
        delayBetweenMacrosMs: 0,
        delayBetweenStepsMs: 0,
      });

      const results = await runner.run();
      expect(results).toHaveLength(3);
      results.forEach((r) => expect(r.macroType).toBe(TEST_MACRO));
    });

    it("should execute multiple macros per iteration", async () => {
      const context = createMockContext();
      const runner = new MacroRunner(context, {
        macros: [TEST_MACRO, TEST_MACRO],
        iterations: 2,
        delayBetweenMacrosMs: 0,
        delayBetweenStepsMs: 0,
      });

      const results = await runner.run();
      expect(results).toHaveLength(4);
    });

    it("should record timing information", async () => {
      const context = createMockContext();
      const runner = new MacroRunner(context, {
        macros: [TEST_MACRO],
        iterations: 1,
        delayBetweenMacrosMs: 0,
        delayBetweenStepsMs: 0,
      });

      const results = await runner.run();
      expect(results[0].startTime).toBeDefined();
      expect(results[0].endTime).toBeDefined();
      expect(results[0].durationMs).toBeGreaterThanOrEqual(0);
      expect(results[0].endTime).toBeGreaterThanOrEqual(results[0].startTime);
    });

    it("should capture error message when macro fails", async () => {
      const context = createMockContext();
      const runner = new MacroRunner(context, {
        macros: [FAILING_MACRO],
        iterations: 1,
        delayBetweenMacrosMs: 0,
        delayBetweenStepsMs: 0,
      });

      const results = await runner.run();
      expect(results[0].error).toBe("Test error");
    });
  });

  describe("stop", () => {
    it("should stop execution when stop is called", async () => {
      const context = createMockContext();
      const runner = new MacroRunner(context, {
        macros: [TEST_MACRO],
        iterations: 100,
        delayBetweenMacrosMs: 10,
        delayBetweenStepsMs: 0,
      });

      const runPromise = runner.run();
      await new Promise((resolve) => setTimeout(resolve, 50));
      runner.stop();

      const results = await runPromise;
      expect(results.length).toBeLessThan(100);
    });

    it("should set isRunning to false when stopped", async () => {
      const context = createMockContext();
      const runner = new MacroRunner(context, {
        macros: [TEST_MACRO],
        iterations: 100,
        delayBetweenMacrosMs: 10,
        delayBetweenStepsMs: 0,
      });

      const runPromise = runner.run();
      expect(runner.isRunning()).toBe(true);

      runner.stop();
      await runPromise;

      expect(runner.isRunning()).toBe(false);
    });
  });

  describe("isRunning", () => {
    it("should return false before run is called", () => {
      const context = createMockContext();
      const runner = new MacroRunner(context, {
        macros: [TEST_MACRO],
        iterations: 1,
        delayBetweenMacrosMs: 0,
        delayBetweenStepsMs: 0,
      });

      expect(runner.isRunning()).toBe(false);
    });

    it("should return true during execution", async () => {
      const context = createMockContext();
      const runner = new MacroRunner(context, {
        macros: [TEST_MACRO],
        iterations: 1,
        delayBetweenMacrosMs: 50,
        delayBetweenStepsMs: 0,
      });

      const runPromise = runner.run();
      expect(runner.isRunning()).toBe(true);

      await runPromise;
      // Note: running flag stays true after natural completion; only stop() sets it false
      expect(runner.isRunning()).toBe(true);
    });
  });

  describe("getResults", () => {
    it("should return a copy of results", async () => {
      const context = createMockContext();
      const runner = new MacroRunner(context, {
        macros: [TEST_MACRO],
        iterations: 2,
        delayBetweenMacrosMs: 0,
        delayBetweenStepsMs: 0,
      });

      await runner.run();
      const results = runner.getResults();

      expect(results).toHaveLength(2);
      results.push({} as any);
      expect(runner.getResults()).toHaveLength(2);
    });
  });

  describe("callbacks", () => {
    it("should call onMacroStart before each macro", async () => {
      const onMacroStart = vi.fn();
      const context = createMockContext();
      const runner = new MacroRunner(
        context,
        {
          macros: [TEST_MACRO],
          iterations: 2,
          delayBetweenMacrosMs: 0,
          delayBetweenStepsMs: 0,
        },
        { onMacroStart },
      );

      await runner.run();

      expect(onMacroStart).toHaveBeenCalledTimes(2);
      expect(onMacroStart).toHaveBeenNthCalledWith(1, TEST_MACRO, 0, 0);
      expect(onMacroStart).toHaveBeenNthCalledWith(2, TEST_MACRO, 0, 1);
    });

    it("should call onMacroComplete after each macro", async () => {
      const onMacroComplete = vi.fn();
      const context = createMockContext();
      const runner = new MacroRunner(
        context,
        {
          macros: [TEST_MACRO],
          iterations: 2,
          delayBetweenMacrosMs: 0,
          delayBetweenStepsMs: 0,
        },
        { onMacroComplete },
      );

      await runner.run();

      expect(onMacroComplete).toHaveBeenCalledTimes(2);
      expect(onMacroComplete.mock.calls[0][0].macroType).toBe(TEST_MACRO);
    });
  });

  describe("unlimited iterations", () => {
    it("should run until stopped when iterations is -1", async () => {
      const context = createMockContext();
      const runner = new MacroRunner(context, {
        macros: [TEST_MACRO],
        iterations: -1,
        delayBetweenMacrosMs: 10,
        delayBetweenStepsMs: 0,
      });

      const runPromise = runner.run();
      await new Promise((resolve) => setTimeout(resolve, 100));
      runner.stop();

      const results = await runPromise;
      expect(results.length).toBeGreaterThan(0);
    });
  });
});
