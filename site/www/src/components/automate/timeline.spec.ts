// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// @vitest-environment node

import { beforeAll, describe, expect, it } from "vitest";

import Automate from "@/components/automate/Automate.astro";
import { EXAMPLES } from "@/components/automate/timeline";
import { islandProps, renderAstro } from "@/testutil";

const text = (html: string): string => html.replace(/<[^>]+>/g, "");

const lines = (html: string): number => html.match(/<span class="line">/g)?.length ?? 0;

describe("timeline", () => {
  let codeHtmls: string[];

  beforeAll(async () => {
    const doc = await renderAstro(Automate);
    ({ codeHtmls } = islandProps<{ codeHtmls: string[] }>(
      doc,
      "@/components/automate/AutomateVisualizer",
    ));
  });

  it("should pair each example with one code sample", () => {
    expect(EXAMPLES.map((e) => e.id)).toEqual(["pressure", "alarm", "abort"]);
    expect(codeHtmls).toHaveLength(EXAMPLES.length);
  });

  it("should show the pressurization sequence with the pressure example", () => {
    expect(text(codeHtmls[0])).toMatch(/^sequence main/);
  });

  it("should show the alarm function with the alarm example", () => {
    expect(text(codeHtmls[1])).toMatch(/^func check_pressure/);
  });

  it("should show the authority declaration with the abort example", () => {
    expect(text(codeHtmls[2])).toMatch(/^authority \(/);
  });

  describe.each(EXAMPLES.map((e, i) => [e.id, e, i] as const))(
    "%s",
    (_, example, index) => {
      it("should highlight only lines that exist in its code", () => {
        const count = lines(codeHtmls[index]);
        for (const { activeLines } of example.steps) {
          expect(activeLines.length).toBeGreaterThan(0);
          for (const line of activeLines) {
            expect(line).toBeGreaterThanOrEqual(1);
            expect(line).toBeLessThanOrEqual(count);
          }
        }
      });

      it("should hold each step for a positive time", () => {
        for (const { duration } of example.steps) expect(duration).toBeGreaterThan(0);
      });
    },
  );

  it("should declare each stage that a stage-based example enters", () => {
    for (const [index, id] of [
      [0, "pressure"],
      [2, "abort"],
    ] as const) {
      const example = EXAMPLES.find((e) => e.id === id)!;
      const code = text(codeHtmls[index]);
      const stages = new Set(example.steps.map((s) => s.state.stage));
      const declared = [...code.matchAll(/stage (\w+)/g)].map((m) => m[1]);
      for (const stage of declared) expect(stages).toContain(stage);
    }
  });
});
