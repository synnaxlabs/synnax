// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Tabs } from "@synnaxlabs/lyra/tabs";
import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

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
});
