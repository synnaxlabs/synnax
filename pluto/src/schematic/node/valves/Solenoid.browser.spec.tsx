// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { computed, renderThemed } from "@synnaxlabs/lyra/styletest";
import { type FC } from "react";
import { describe, expect, it } from "vitest";
import { server, userEvent } from "vitest/browser";

import { CSS } from "@/css";
import { Node } from "@/schematic/node";

interface SolenoidProps {
  normallyOpen?: boolean;
  enabled?: boolean;
}

const Solenoid = Node.resolveSpec("solenoid_valve").Preview as FC<SolenoidProps>;

const SHAPES = "path, rect, circle, ellipse, line, polyline, polygon";

// A computed color has more digits than its math keeps the same on each platform.
const roundAll = (text: string): string =>
  text.replace(/\d+\.\d+/g, (n) => (Math.round(Number(n) * 100) / 100).toString());

const paint = (container: Element): string[] =>
  Array.from(container.querySelectorAll(SHAPES), (shape) =>
    [shape.tagName, computed(shape, "stroke"), computed(shape, "fill")]
      .map(roundAll)
      .join(" | "),
  );

// The name, normallyOpen, and enabled of each case.
const CASES: [string, boolean, boolean][] = [
  ["normally closed", false, false],
  ["normally closed on", false, true],
  ["normally open", true, false],
  ["normally open on", true, true],
];

describe("Solenoid", () => {
  it("should paint each shape at rest and hovered as recorded", async () => {
    const record: Record<string, string[]> = {};
    for (const [name, normallyOpen, enabled] of CASES)
      for (const editable of [false, true]) {
        const key = editable ? `${name} editing` : name;
        // Hover paints a toggle only inside a symbol container, and edit mode changes
        // the paint.
        const { container, unmount } = await renderThemed(
          <div
            className={CSS.cls(CSS.BE("symbol", "container"), CSS.editable(editable))}
          >
            <Solenoid normallyOpen={normallyOpen} enabled={enabled} />
          </div>,
        );
        const button = container.querySelector("button");
        if (button == null) throw new Error("solenoid not rendered");
        record[key] = paint(container);
        await userEvent.hover(button);
        record[`${key} hovered`] = paint(container);
        await userEvent.unhover(button);
        await unmount();
      }
    await expect(`${JSON.stringify(record, null, 2)}\n`).toMatchFileSnapshot(
      `__snapshots__/Solenoid.${server.browser}.json`,
    );
  });
});
