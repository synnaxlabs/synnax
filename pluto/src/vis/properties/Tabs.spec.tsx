// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Tabs } from "@synnaxlabs/lyra/tabs";
import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Properties } from "@/vis/properties";

const renderTabs = (props: Omit<Properties.TabsProps, "children">) =>
  render(
    <Properties.Tabs {...props}>
      {props.tabs.map((key) => (
        <Tabs.Content key={key} itemKey={key}>
          {key} content
        </Tabs.Content>
      ))}
    </Properties.Tabs>,
  );

const tab = (name: string) => screen.getByRole("tab", { name });

describe("Properties.Tabs", () => {
  it("should select the first tab", () => {
    renderTabs({ tabs: ["telemetry", "style"] });
    expect(tab("Telemetry").ariaSelected).toBe("true");
    expect(tab("Style").ariaSelected).toBe("false");
    expect(screen.getByText("telemetry content")).toBeTruthy();
  });

  it("should render the actions in the tab rail", () => {
    renderTabs({ tabs: ["style"], actions: <button>Swap</button> });
    const rail = tab("Style").parentElement;
    expect(rail?.contains(screen.getByRole("button", { name: "Swap" }))).toBe(true);
  });

  it("should open on the given tab", () => {
    renderTabs({ tabs: ["telemetry", "style"], tab: "style", onTabChange: vi.fn() });
    expect(tab("Style").ariaSelected).toBe("true");
    expect(screen.getByText("style content")).toBeTruthy();
  });

  it("should show the first tab when the given tab is not in the rail", () => {
    renderTabs({ tabs: ["control", "style"], tab: "redline", onTabChange: vi.fn() });
    expect(tab("Control").ariaSelected).toBe("true");
  });

  it("should report a selected tab without switching to it", () => {
    const onTabChange = vi.fn();
    renderTabs({ tabs: ["telemetry", "style"], tab: "telemetry", onTabChange });
    fireEvent.click(tab("Style"));
    expect(onTabChange).toHaveBeenCalledWith("style");
    expect(tab("Telemetry").ariaSelected).toBe("true");
  });

  it("should switch tabs on its own without onTabChange", () => {
    renderTabs({ tabs: ["telemetry", "style"] });
    fireEvent.click(tab("Style"));
    expect(tab("Style").ariaSelected).toBe("true");
  });
});
