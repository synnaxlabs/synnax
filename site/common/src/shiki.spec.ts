// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { grammar, tokens } from "@synnaxlabs/arc";
import { createHighlighter } from "shiki";
import { describe, expect, it } from "vitest";

import { highlighter, symbols, theme } from "./shiki";

const color = ({ light, dark }: { light: string; dark: string }): string =>
  `light-dark(${light}, ${dark})`;

const ARC = `sequence main {
  stage press {
    valve_cmd = true
  }
  pressure > 100.0 => stage vent
  time.interval -> pressure
}`;

describe("shiki", () => {
  describe("highlighter", () => {
    it("should color Arc code with the Console palette", async () => {
      const render = await highlighter([grammar]);
      const html = render(ARC, "arc");
      expect(html).toContain(`color:${color(tokens.keyword)}`);
      expect(html).toContain(`color:${color(tokens.edgeConditional)}`);
      expect(html).toContain(`color:${color(tokens.edgeContinuous)}`);
    });

    it("should leave other languages on the CSS variable theme", async () => {
      const render = await highlighter(["python"]);
      const html = render("def f():\n    return 1", "python");
      expect(html).toContain('color:var(--astro-code-token-keyword)">def</span>');
    });
  });

  describe("symbols", () => {
    const render = async (code: string, meta: string): Promise<string> => {
      const h = await createHighlighter({ themes: [theme], langs: [grammar] });
      return h.codeToHtml(code, {
        lang: "arc",
        theme,
        meta: { __raw: meta },
        transformers: [symbols],
      });
    };

    const span = (name: string, color: string): string =>
      `<span style="color:${color}">${name}</span>`;

    it("should color declared channels and bodies", async () => {
      const html = await render(
        "a = pressure_1\nstage main {}",
        'channels="pressure_1" bodies="main"',
      );
      expect(html).toContain(span("pressure_1", color(tokens.channel)));
      expect(html).toContain(span("main", color(tokens.function)));
    });

    it("should leave undeclared identifiers as variables", async () => {
      const html = await render("a = pressure_1", 'channels="other"');
      // Shiki merges same-colored runs, so the name arrives with its leading space.
      expect(html).toContain(`color:${color(tokens.variable)}"> pressure_1</span>`);
    });

    it("should not recolor a declared name inside a string", async () => {
      const html = await render('a = "pressure_1"', 'channels="pressure_1"');
      expect(html).not.toContain(span("pressure_1", color(tokens.channel)));
    });
  });
});
