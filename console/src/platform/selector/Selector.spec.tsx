// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Selector } from "@/platform/selector";
import { Session } from "@/session";
import { renderWithConsole } from "@/testutil";

const layout: Session.Layout.BaseState = {
  type: "cat",
  name: "Cat",
  location: "mosaic",
};

describe("Selector.Selector", () => {
  it("renders the prompt text and one item per selectable", async () => {
    const cat = Selector.createSimpleItem({ title: "Cat", icon: <Icon.Add />, layout });
    const dog = Selector.createSimpleItem({
      title: "Dog",
      icon: <Icon.Add />,
      layout: { type: "dog", name: "Dog", location: "mosaic" },
    });
    await renderWithConsole(
      <Selector.Selector
        layoutKey="lk"
        visible
        focused={false}
        onClose={() => {}}
        selectables={[cat, dog]}
        text="Choose one"
      />,
    );
    expect(screen.getByText("Choose one")).toBeTruthy();
    expect(screen.getByText("Cat")).toBeTruthy();
    expect(screen.getByText("Dog")).toBeTruthy();
  });

  it("places the selectable's layout under the selector's key when clicked", async () => {
    const cat = Selector.createSimpleItem({ title: "Cat", icon: <Icon.Add />, layout });
    const { store } = await renderWithConsole(
      <Selector.Selector
        layoutKey="lk"
        visible
        focused={false}
        onClose={() => {}}
        selectables={[cat]}
        text="Choose one"
      />,
    );
    fireEvent.click(screen.getByText("Cat"));
    await waitFor(() => {
      const placed = Session.Layout.select(store.getState(), "lk");
      expect(placed?.type).toBe("cat");
      expect(placed?.name).toBe("Cat");
      expect(placed?.location).toBe("mosaic");
    });
  });
});

describe("Selector.create", () => {
  it("builds a renderer that mounts the Selector with the given selectables", async () => {
    const cat = Selector.createSimpleItem({ title: "Cat", icon: <Icon.Add />, layout });
    const Renderer = Selector.create([cat], "Pick a visualization");
    expect(Renderer.displayName).toBe("LayoutSelector");
    const { store } = await renderWithConsole(
      <Renderer layoutKey="lk" visible focused={false} onClose={() => {}} />,
    );
    expect(screen.getByText("Pick a visualization")).toBeTruthy();
    fireEvent.click(screen.getByText("Cat"));
    await waitFor(() => {
      expect(Session.Layout.select(store.getState(), "lk")?.type).toBe("cat");
    });
  });
});
