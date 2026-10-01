// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { beforeEach, onTestFinished } from "vitest";

const METHODS = ["error", "warn"] as const;

/**
 * Fails every test that writes to console.error or console.warn, which includes React's
 * act and prop warnings. A test that expects output mocks the method with vi.spyOn,
 * which replaces the recorder for that test. Call it once from a setup file.
 */
export const failOnConsoleOutput = (): void => {
  const originals = { error: console.error, warn: console.warn };
  beforeEach(() => {
    const messages: string[] = [];
    for (const method of METHODS)
      console[method] = (...args: unknown[]) => {
        messages.push(`console.${method}: ${args.map(String).join(" ")}`);
        originals[method](...args);
      };
    // Runs after every afterEach hook, so cleanup still runs and its output counts.
    onTestFinished(() => {
      for (const method of METHODS) console[method] = originals[method];
      if (messages.length > 0)
        throw new Error(`Unexpected console output:\n${messages.join("\n")}`);
    });
  });
};
