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

describe("Properties.Tabs", () => {
  it("should select the first tab", () => {
    render(
      <Properties.Tabs tabs={["telemetry", "style"]}>
        <Tabs.Content itemKey="telemetry">telemetry content</Tabs.Content>
        <Tabs.Content itemKey="style">style content</Tabs.Content>
      </Properties.Tabs>,
    );
    expect(screen.getByRole("tab", { name: "Telemetry" }).ariaSelected).toBe("true");
    expect(screen.getByRole("tab", { name: "Style" }).ariaSelected).toBe("false");
    expect(screen.getByText("telemetry content")).toBeTruthy();
  });

  it("should render the actions in the tab rail", () => {
    render(
      <Properties.Tabs tabs={["style"]} actions={<button>Swap</button>}>
        <Tabs.Content itemKey="style">style content</Tabs.Content>
      </Properties.Tabs>,
    );
    const rail = screen.getByRole("tab", { name: "Style" }).parentElement;
    expect(rail?.contains(screen.getByRole("button", { name: "Swap" }))).toBe(true);
  });

  it("should open on the given tab", () => {
    render(
      <Properties.Tabs tabs={["telemetry", "style"]} tab="style" onTabChange={vi.fn()}>
        <Tabs.Content itemKey="telemetry">telemetry content</Tabs.Content>
        <Tabs.Content itemKey="style">style content</Tabs.Content>
      </Properties.Tabs>,
    );
    expect(screen.getByRole("tab", { name: "Style" }).ariaSelected).toBe("true");
    expect(screen.getByText("style content")).toBeTruthy();
  });

  it("should show the first tab when the given tab is not in the rail", () => {
    render(
      <Properties.Tabs tabs={["control", "style"]} tab="redline" onTabChange={vi.fn()}>
        <Tabs.Content itemKey="control">control content</Tabs.Content>
        <Tabs.Content itemKey="style">style content</Tabs.Content>
      </Properties.Tabs>,
    );
    expect(screen.getByRole("tab", { name: "Control" }).ariaSelected).toBe("true");
  });

  it("should report a selected tab without switching to it", () => {
    const onTabChange = vi.fn();
    render(
      <Properties.Tabs
        tabs={["telemetry", "style"]}
        tab="telemetry"
        onTabChange={onTabChange}
      >
        <Tabs.Content itemKey="telemetry">telemetry content</Tabs.Content>
        <Tabs.Content itemKey="style">style content</Tabs.Content>
      </Properties.Tabs>,
    );
    fireEvent.click(screen.getByRole("tab", { name: "Style" }));
    expect(onTabChange).toHaveBeenCalledWith("style");
    expect(screen.getByRole("tab", { name: "Telemetry" }).ariaSelected).toBe("true");
  });

  it("should switch tabs on its own without onTabChange", () => {
    render(
      <Properties.Tabs tabs={["telemetry", "style"]}>
        <Tabs.Content itemKey="telemetry">telemetry content</Tabs.Content>
        <Tabs.Content itemKey="style">style content</Tabs.Content>
      </Properties.Tabs>,
    );
    fireEvent.click(screen.getByRole("tab", { name: "Style" }));
    expect(screen.getByRole("tab", { name: "Style" }).ariaSelected).toBe("true");
  });
});
