// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";
import { userEvent } from "vitest/browser";

import { computed, renderThemed, resolve } from "@/styletest";
import { Tabs } from "@/tabs";

const renderTabs = async (
  variant?: Tabs.Variant,
): Promise<{
  tab: (name: string) => Element;
  strip: Element;
  panel: (text: string) => Element | null;
}> => {
  const { getByRole, getByText } = await renderThemed(
    <Tabs.Frame initialValue="a">
      <Tabs.Selector variant={variant}>
        <Tabs.Tab itemKey="a">Tab A</Tabs.Tab>
        <Tabs.Tab itemKey="b">Tab B</Tabs.Tab>
      </Tabs.Selector>
      <Tabs.Content itemKey="a">Content A</Tabs.Content>
      <Tabs.Content itemKey="b">Content B</Tabs.Content>
    </Tabs.Frame>,
  );
  return {
    tab: (name) => getByRole("tab", { name }).element(),
    strip: getByRole("tablist").element(),
    panel: (text) => getByText(text).query(),
  };
};

describe("Tabs", () => {
  it("should mark the selected tab in the subtle selection tier", async () => {
    const { tab } = await renderTabs();
    const selected = tab("Tab A");
    expect(computed(selected, "background-color")).toBe(
      resolve("var(--pluto-selected-subtle-bg)"),
    );
    expect(computed(selected, "color")).toBe(
      resolve("var(--pluto-selected-subtle-color)"),
    );
    expect(computed(selected, "box-shadow")).toContain(
      resolve("var(--pluto-selected-subtle-border-color)"),
    );
  });

  it("should shade an unselected tab and fill it on hover", async () => {
    const { tab } = await renderTabs();
    const other = tab("Tab B");
    expect(computed(other, "color")).toBe(resolve("var(--pluto-gray-l9)"));
    await userEvent.hover(other);
    expect(computed(other, "background-color")).toBe(
      resolve("var(--pluto-fill-hover)"),
    );
  });

  it("should fit every tab inside the strip at one height", async () => {
    const { tab, strip } = await renderTabs();
    const a = tab("Tab A").getBoundingClientRect();
    const b = tab("Tab B").getBoundingClientRect();
    const s = strip.getBoundingClientRect();
    expect(a.height).toBe(b.height);
    expect(a.top).toBeGreaterThanOrEqual(s.top);
    expect(a.bottom).toBeLessThanOrEqual(s.bottom);
  });

  it("should show only the selected panel", async () => {
    const { panel } = await renderTabs();
    expect(panel("Content A")?.checkVisibility()).toBe(true);
    expect(panel("Content B")?.checkVisibility() ?? false).toBe(false);
  });

  it("should give fixed tabs one width", async () => {
    const { tab } = await renderTabs("pill");
    expect(tab("Tab A").getBoundingClientRect().width).toBe(
      tab("Tab B").getBoundingClientRect().width,
    );
  });
});
