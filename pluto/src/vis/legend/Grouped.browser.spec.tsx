// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { computed, renderThemed, resolve } from "@synnaxlabs/lyra/styletest";
import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { CSS } from "@/css";
import { Legend } from "@/vis/legend";

const DATA: Legend.GroupData[] = [
  { key: "y1", name: "Y1", data: [{ key: "a", label: "a", color: "#ff0000" }] },
  { key: "y2", name: "Y2", data: [{ key: "b", label: "b", color: "#00ff00" }] },
];

interface Rendered {
  divider: Element;
  above: DOMRect;
  below: DOMRect;
}

const renderGrouped = async (): Promise<Rendered> => {
  const { container } = await renderThemed(
    <div style={{ position: "relative", width: 400, height: 400 }}>
      <Legend.Grouped data={DATA} />
    </div>,
  );
  const divider = container.querySelector(`.${CSS.B("legend-divider")}`);
  if (divider == null) throw new Error("legend divider not found");
  const entries = container.querySelectorAll(`.${CSS.B("legend-entry")}`);
  if (entries.length !== 2) throw new Error(`found ${entries.length} legend entries`);
  return {
    divider,
    above: entries[0].getBoundingClientRect(),
    below: entries[1].getBoundingClientRect(),
  };
};

describe("Legend.Grouped", () => {
  it("should draw the divider between groups with the shared border", async () => {
    const { divider } = await renderGrouped();
    expect(computed(divider, "border-top")).toBe(
      resolve("var(--pluto-border)", "border-top"),
    );
  });

  it("should line up nested labels with the group name with no start", async () => {
    await renderThemed(
      <div style={{ position: "relative", width: 400, height: 400 }}>
        <Legend.Grouped
          nested
          data={[
            {
              key: "r",
              name: "Range",
              color: "#ff0000",
              data: [{ key: "a", label: "a" }],
            },
          ]}
        />
      </div>,
    );
    const name = screen.getByText("Range").getBoundingClientRect();
    const label = screen.getByText("a").getBoundingClientRect();
    expect(label.left).toBeCloseTo(name.left, 0);
  });

  it("should put the badge of a nested group at the right edge", async () => {
    await renderThemed(
      <div style={{ position: "relative", width: 400, height: 400 }}>
        <Legend.Grouped
          nested
          data={[
            {
              key: "r",
              name: "R",
              color: "#ff0000",
              badge: { label: "T0", tooltip: "Custom T0" },
              data: [{ key: "a", label: "a much longer channel name" }],
            },
          ]}
        />
      </div>,
    );
    const badge = screen.getByText("T0").getBoundingClientRect();
    const label = screen
      .getByText("a much longer channel name")
      .getBoundingClientRect();
    expect(badge.right).toBeGreaterThanOrEqual(label.right);
  });

  it("should wrap a long range name with its badge on the first line", async () => {
    const name =
      "Hot fire 3 LOX tank pressurization with the new regulator, second try";
    await renderThemed(
      <div style={{ position: "relative", width: 1000, height: 400 }}>
        <Legend.Grouped
          nested
          data={[
            {
              key: "r",
              name,
              color: "#ff0000",
              badge: { label: "T0", tooltip: "Custom T0" },
              data: [{ key: "a", label: "a" }],
            },
          ]}
        />
      </div>,
    );
    const text = screen.getByText(name);
    const lineHeight = parseFloat(getComputedStyle(text).lineHeight);
    const rem = parseFloat(getComputedStyle(document.documentElement).fontSize);
    const lines = text.clientHeight - parseFloat(getComputedStyle(text).paddingTop) * 2;
    expect(lines).toBeCloseTo(2 * lineHeight, 0);
    expect(text.getBoundingClientRect().width).toBeLessThanOrEqual(40 * rem);
    const badge = screen.getByText("T0").getBoundingClientRect();
    expect(badge.bottom).toBeLessThan(
      text.getBoundingClientRect().top + lineHeight + 4,
    );
  });

  it("should leave space between the divider and the groups on each side", async () => {
    const { divider, above, below } = await renderGrouped();
    const line = divider.getBoundingClientRect();
    expect(line.top).toBeGreaterThan(above.bottom);
    expect(below.top).toBeGreaterThan(line.bottom);
  });
});
