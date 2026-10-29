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

import { List } from "@/list";
import { computed, renderThemed, resolve } from "@/styletest";

const SELECTED = new Set(["2", "3"]);

const renderList = async (): Promise<(key: string) => Element> => {
  const { getByText } = await renderThemed(
    <List.Frame data={["1", "2", "3", "4"]}>
      <List.Scroll>
        <List.Items>
          {({ key, ...rest }: List.ItemProps<string>) => (
            <List.Item key={key} {...rest} selected={SELECTED.has(rest.itemKey)}>
              {key}
            </List.Item>
          )}
        </List.Items>
      </List.Scroll>
    </List.Frame>,
  );
  return (key) => getByText(key, { exact: true }).element();
};

const radius = (el: Element, corner: string): string =>
  computed(el, `border-${corner}-radius`);

describe("List", () => {
  it("should fill a selected row with the list selection tier", async () => {
    const row = await renderList();
    expect(computed(row("2"), "background-color")).toBe(
      resolve("var(--pluto-selected-list-bg)"),
    );
    expect(computed(row("2"), "color")).toBe(resolve("var(--pluto-text-color)"));
  });

  it("should leave an unselected row clear until hovered", async () => {
    const row = await renderList();
    expect(computed(row("1"), "background-color")).toBe(resolve("transparent"));
    await userEvent.hover(row("1"));
    expect(computed(row("1"), "background-color")).toBe(
      resolve("var(--pluto-fill-hover)"),
    );
  });

  it("should merge contiguous selected rows into one box", async () => {
    const row = await renderList();
    const rounded = resolve(
      "var(--pluto-border-radius-small)",
      "border-top-left-radius",
    );
    expect(radius(row("2"), "top-left")).toBe(rounded);
    expect(radius(row("2"), "bottom-left")).toBe("0px");
    expect(radius(row("3"), "top-left")).toBe("0px");
    expect(radius(row("3"), "bottom-left")).toBe(rounded);
  });
});
