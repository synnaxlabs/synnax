// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { computed, renderThemed, resolve } from "@synnaxlabs/lyra/styletest";
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

  it("should leave space between the divider and the groups on each side", async () => {
    const { divider, above, below } = await renderGrouped();
    const line = divider.getBoundingClientRect();
    expect(line.top).toBeGreaterThan(above.bottom);
    expect(below.top).toBeGreaterThan(line.bottom);
  });
});
