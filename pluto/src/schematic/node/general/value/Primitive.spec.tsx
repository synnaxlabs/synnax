// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { CSS } from "@/css";
import { Value } from "@/schematic/node/general/value/Primitive";

const INLINE_SIZE = 70;

const content = (container: HTMLElement): HTMLElement => {
  const el = container.querySelector<HTMLElement>(`.${CSS.BE("value", "content")}`);
  if (el == null) throw new Error("expected a value content box");
  return el;
};

describe("value symbol", () => {
  it("should take its width from the configured inline size", () => {
    const { container } = render(
      <Value inlineSize={INLINE_SIZE} orientation="left" units="psi" />,
    );
    expect(content(container).style.inlineSize).toBe(`${INLINE_SIZE}px`);
  });

  it("should keep that width as the value grows and shrinks", () => {
    const value = (v: string) => (
      <Value inlineSize={INLINE_SIZE} orientation="left" units="psi">
        {v}
      </Value>
    );
    const { container, rerender } = render(value("1"));
    const { style } = content(container);
    for (const v of ["1".repeat(60), "1"]) {
      rerender(value(v));
      expect(style.inlineSize).toBe(`${INLINE_SIZE}px`);
      // A telemetry-driven width reached the box through these two.
      expect(style.minWidth).toBe("");
      expect(style.maxWidth).toBe("");
    }
  });
});
