// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";
import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, type Mock, vi } from "vitest";

import { Selector } from "@/platform/selector";
import { renderWithConsole } from "@/testutil";

const createSelectable = (
  title: string,
  rest: Partial<Selector.Selectable> = {},
): Selector.Selectable & { onSelect: Mock } => {
  const onSelect = vi.fn();
  return {
    type: title.toLowerCase(),
    title,
    icon: <Icon.Add />,
    useOnSelect: () => onSelect,
    onSelect,
    ...rest,
  };
};

const renderSelector = async (sections: Selector.Section[]) => {
  const { Content } = Selector.create({
    sections,
    tabTitle: "Create",
    text: "Choose one",
    icon: <Icon.Add />,
    searchPlaceholder: "Search animals",
  });
  await renderWithConsole(<Content />);
  return screen.getByPlaceholderText("Search animals");
};

const disabled = (name: string): boolean =>
  screen.getByRole("button", { name }).getAttribute("aria-disabled") === "true";

const marked = (name: string): boolean =>
  screen.getByRole("button", { name }).classList.contains("console--hovered");

describe("Selector.create", () => {
  it("should call the onSelect of the clicked item with no arguments", async () => {
    const cat = createSelectable("Cat");
    const dog = createSelectable("Dog");
    await renderSelector([{ key: "pets", selectables: [cat, dog] }]);
    fireEvent.click(screen.getByRole("button", { name: "Dog" }));
    // A create hook takes optional arguments, so the click event must not reach it.
    expect(dog.onSelect).toHaveBeenCalledExactlyOnceWith();
    expect(cat.onSelect).not.toHaveBeenCalled();
  });

  it("should leave out an item whose useVisible returns false", async () => {
    const dog = createSelectable("Dog", { useVisible: () => false });
    await renderSelector([
      { key: "pets", selectables: [createSelectable("Cat"), dog] },
    ]);
    expect(screen.getByRole("button", { name: "Cat" })).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Dog" })).toBeNull();
  });

  it("should leave out a section whose useVisible returns false", async () => {
    await renderSelector([
      { key: "pets", title: "Pets", selectables: [createSelectable("Cat")] },
      {
        key: "farm",
        title: "Farm",
        useVisible: () => false,
        selectables: [createSelectable("Cow")],
      },
    ]);
    expect(screen.getByText("Pets")).toBeTruthy();
    expect(screen.queryByText("Farm")).toBeNull();
    expect(screen.queryByRole("button", { name: "Cow" })).toBeNull();
  });

  it("should show the short label of an item in a group", async () => {
    const food = createSelectable("Cat food", { label: "Food" });
    await renderSelector([
      {
        key: "shop",
        groups: [{ key: "cat", title: "Cat", icon: <Icon.Add />, selectables: [food] }],
      },
    ]);
    expect(screen.getByRole("button", { name: "Cat food" }).textContent).toBe("Food");
  });

  describe("search", () => {
    const renderAnimals = async () => {
      const animals = ["Cat", "Camel", "Dog"].map((title) => createSelectable(title));
      const search = await renderSelector([{ key: "pets", selectables: animals }]);
      const press = (code: string): void => {
        fireEvent.keyDown(search, { key: code, code });
        fireEvent.keyUp(search, { key: code, code });
      };
      const [cat, camel, dog] = animals;
      return { cat, camel, dog, search, press };
    };

    it("should disable the items that do not match", async () => {
      const { search } = await renderAnimals();
      fireEvent.change(search, { target: { value: "ca" } });
      expect(disabled("Cat")).toBe(false);
      expect(disabled("Camel")).toBe(false);
      expect(disabled("Dog")).toBe(true);
    });

    it("should pick the first match with Enter", async () => {
      const { cat, dog, search, press } = await renderAnimals();
      fireEvent.change(search, { target: { value: "do" } });
      press("Enter");
      expect(dog.onSelect).toHaveBeenCalledOnce();
      expect(cat.onSelect).not.toHaveBeenCalled();
    });

    it("should pick the next match with the down arrow and Enter", async () => {
      const { cat, camel, search, press } = await renderAnimals();
      fireEvent.change(search, { target: { value: "ca" } });
      press("ArrowDown");
      press("Enter");
      expect(camel.onSelect).toHaveBeenCalledOnce();
      expect(cat.onSelect).not.toHaveBeenCalled();
    });

    it("should move the mark off the pointer item with the down arrow", async () => {
      const { press } = await renderAnimals();
      const cat = screen.getByRole("button", { name: "Cat" });
      fireEvent.pointerMove(cat, { pointerType: "mouse", clientX: 1 });
      fireEvent.pointerMove(cat, { pointerType: "mouse", clientX: 2 });
      expect(marked("Cat")).toBe(true);
      press("ArrowDown");
      expect(marked("Cat")).toBe(false);
      expect(marked("Camel")).toBe(true);
    });

    it("should drop the mark when Escape leaves an empty search", async () => {
      const { search, press } = await renderAnimals();
      press("ArrowDown");
      expect(marked("Cat")).toBe(true);
      fireEvent.keyDown(search, { key: "Escape", code: "Escape" });
      expect(marked("Cat")).toBe(false);
    });

    it("should clear the search on Escape", async () => {
      const { search } = await renderAnimals();
      fireEvent.change(search, { target: { value: "ca" } });
      fireEvent.keyDown(search, { key: "Escape", code: "Escape" });
      expect(search).toHaveProperty("value", "");
      expect(disabled("Dog")).toBe(false);
    });
  });
});
