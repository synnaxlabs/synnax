// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { renderThemed, resolve } from "@synnaxlabs/lyra/styletest";
import { describe, expect, it } from "vitest";

import { Problem } from "@/vis/problem";
import { type problem } from "@/vis/problem/aether";

const FAILED: problem.Problem = {
  variant: "error",
  name: "",
  message: "Failed to read channel data",
  description: "",
};

const px = (value: string, property: string): number =>
  parseFloat(resolve(value, property));

const renderMark = async (variant?: Problem.MarkProps["variant"]) => {
  const { container } = await renderThemed(
    <div style={{ position: "relative", width: 200, height: 200 }}>
      <Problem.Mark problems={[FAILED]} variant={variant} />
    </div>,
  );
  const parent = container.firstElementChild;
  const mark = parent?.querySelector(".pluto-problem-mark");
  const glyph = mark?.querySelector(".lyra-icon");
  if (parent == null || mark == null || glyph == null) throw new Error("no mark");
  return {
    parent: parent.getBoundingClientRect(),
    mark: mark.getBoundingClientRect(),
    glyph: glyph.getBoundingClientRect(),
    tile: mark.querySelector(".lyra-icon-tile"),
  };
};

describe("Problem.Mark", () => {
  it("should frame the glyph in a tiny tile inside the top right corner", async () => {
    const { parent, mark, glyph, tile } = await renderMark();
    const side = px("var(--pluto-height-tiny)", "width");
    expect(tile?.getBoundingClientRect().width).toBeCloseTo(side);
    expect(tile?.getBoundingClientRect().height).toBeCloseTo(side);
    expect(glyph.width).toBeCloseTo((tile?.clientWidth ?? 0) / 2);
    expect(mark.top).toBeGreaterThan(parent.top);
    expect(mark.right).toBeLessThan(parent.right);
  });

  it("should size a glyph to the tiny text inside the top right corner", async () => {
    const { parent, mark, glyph, tile } = await renderMark("glyph");
    expect(tile).toBeNull();
    expect(glyph.height).toBeCloseTo(px("var(--pluto-tiny-size)", "font-size"));
    expect(mark.top).toBeGreaterThan(parent.top);
    expect(mark.right).toBeLessThan(parent.right);
  });

  it("should center a badge on the top right corner", async () => {
    const { parent, mark, glyph } = await renderMark("badge");
    expect(glyph.height).toBeCloseTo(px("var(--pluto-tiny-size)", "font-size"));
    expect(mark.width).toBeGreaterThan(glyph.width);
    expect(mark.left + mark.width / 2).toBeCloseTo(parent.right, 0);
    expect(mark.top + mark.height / 2).toBeCloseTo(parent.top, 0);
  });
});
