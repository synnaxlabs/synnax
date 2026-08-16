// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";
import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Selector } from "@/platform/selector";
import { renderWithConsole } from "@/testutil";

describe("Selector.create", () => {
  it("renders one item per selectable and routes the click to that item's callback", async () => {
    const onCat = vi.fn();
    const onDog = vi.fn();
    const cat = Selector.createSelectable({
      type: "cat",
      title: "Cat",
      icon: <Icon.Add />,
      useOnSelect: () => onCat,
    });
    const dog = Selector.createSelectable({
      type: "dog",
      title: "Dog",
      icon: <Icon.Add />,
      useOnSelect: () => onDog,
    });
    const { Content } = Selector.create({
      selectables: [cat, dog],
      tabTitle: "Create",
      text: "Choose one",
      icon: <Icon.Add />,
    });
    await renderWithConsole(<Content />);
    expect(screen.getByText("Choose one")).toBeTruthy();
    expect(screen.getByText("Cat")).toBeTruthy();
    fireEvent.click(screen.getByText("Dog"));
    expect(onDog).toHaveBeenCalledTimes(1);
    expect(onCat).not.toHaveBeenCalled();
  });

  it("hides selectables whose useVisible returns false", async () => {
    const cat = Selector.createSelectable({
      type: "cat",
      title: "Cat",
      icon: <Icon.Add />,
      useOnSelect: () => vi.fn(),
    });
    const dog = Selector.createSelectable({
      type: "dog",
      title: "Dog",
      icon: <Icon.Add />,
      useOnSelect: () => vi.fn(),
      useVisible: () => false,
    });
    const { Content } = Selector.create({
      selectables: [cat, dog],
      tabTitle: "Create",
      text: "Choose one",
      icon: <Icon.Add />,
    });
    await renderWithConsole(<Content />);
    expect(screen.getByText("Cat")).toBeTruthy();
    expect(screen.queryByText("Dog")).toBeNull();
  });
});
