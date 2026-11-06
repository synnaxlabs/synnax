// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic } from "@synnaxlabs/client";
import { renderThemed } from "@synnaxlabs/lyra/styletest";
import { describe, expect, it } from "vitest";

import { Button } from "@/schematic/node/general/button";

const LONG = "Open the main fuel valve on the upper stage";
const UNBROKEN = "MAIN_FUEL_ISOLATION_VALVE_ON_THE_UPPER_STAGE";
const TWO_LINES = "LAUNCH\nPUMP VC";

const renderButton = async (
  label: Partial<schematic.LabelConfig>,
): Promise<HTMLElement> => {
  const { container } = await renderThemed(
    <Button.spec.Preview
      {...schematic.buttonNodeConfigZ.parse({ variant: "button", label })}
    />,
  );
  const el = container.querySelector<HTMLElement>(".lyra-btn");
  if (el == null) throw new Error("expected a button element");
  return el;
};

const lineRects = (el: HTMLElement): DOMRect[] => {
  const range = document.createRange();
  range.selectNodeContents(el);
  return [...range.getClientRects()].filter((r) => r.width > 0);
};

const lineCount = (el: HTMLElement): number =>
  new Set(lineRects(el).map((r) => Math.round(r.top))).size;

describe("button symbol layout", () => {
  it("should wrap a long label at the default wrap width", async () => {
    const el = await renderButton({ label: LONG });
    expect(lineCount(el)).toBeGreaterThan(1);
    expect(el.getBoundingClientRect().width).toBeLessThanOrEqual(150);
  });

  it("should keep a long label on one line at a wrap width of 0", async () => {
    const el = await renderButton({ label: LONG, maxInlineSize: 0 });
    expect(lineCount(el)).toBe(1);
    expect(el.getBoundingClientRect().width).toBeGreaterThan(150);
  });

  it("should wrap a long label at a chosen wrap width", async () => {
    const el = await renderButton({ label: LONG, maxInlineSize: 100 });
    expect(lineCount(el)).toBeGreaterThan(1);
    expect(el.getBoundingClientRect().width).toBeLessThanOrEqual(100);
  });

  it("should break an unbroken label inside the button", async () => {
    const el = await renderButton({ label: UNBROKEN, maxInlineSize: 100 });
    const { left, right } = el.getBoundingClientRect();
    const rects = lineRects(el);
    expect(lineCount(el)).toBeGreaterThan(1);
    expect(Math.min(...rects.map((r) => r.left))).toBeGreaterThanOrEqual(left);
    expect(Math.max(...rects.map((r) => r.right))).toBeLessThanOrEqual(right);
  });

  it("should start a new line at a newline", async () => {
    expect(lineCount(await renderButton({ label: TWO_LINES }))).toBe(2);
  });

  it("should grow the button to hold every line", async () => {
    const single = await renderButton({ label: "LAUNCH" });
    const double = await renderButton({ label: TWO_LINES });
    expect(double.getBoundingClientRect().height).toBeGreaterThan(
      single.getBoundingClientRect().height,
    );
  });

  it("should collapse repeated spaces", async () => {
    const single = await renderButton({ label: "A B" });
    const repeated = await renderButton({ label: "A   B" });
    expect(repeated.getBoundingClientRect().width).toBe(
      single.getBoundingClientRect().width,
    );
  });

  it("should align lines to the start", async () => {
    const [first, second] = lineRects(
      await renderButton({ label: TWO_LINES, align: "start" }),
    );
    expect(first.left).toBeCloseTo(second.left, 0);
  });

  it("should align lines to the end", async () => {
    const [first, second] = lineRects(
      await renderButton({ label: TWO_LINES, align: "end" }),
    );
    expect(first.right).toBeCloseTo(second.right, 0);
  });

  it("should center lines by default", async () => {
    const [first, second] = lineRects(await renderButton({ label: TWO_LINES }));
    expect(first.left - second.left).toBeCloseTo(second.right - first.right, 0);
  });
});
