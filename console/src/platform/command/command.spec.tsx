// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";
import { act, fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Command } from "@/platform/command";
import { createConsoleWrapper } from "@/testutil";

describe("Command.create", () => {
  it("should invoke the hook-produced callback when the command is selected", async () => {
    const onSelect = vi.fn();
    const Cmd = Command.create({
      key: "cc",
      name: "Hook Command",
      icon: <Icon.Close />,
      useOnSelect: () => onSelect,
    });
    const { wrapper } = await createConsoleWrapper({ client: null });
    render(<Cmd key={Cmd.key} itemKey={Cmd.key} index={0} />, { wrapper });
    await act(async () => {
      fireEvent.click(screen.getByText("Hook Command"), { detail: 0 });
    });
    expect(onSelect).toHaveBeenCalledTimes(1);
  });

  it("should not fire the callback for a real pointer click routed by the list", async () => {
    const onSelect = vi.fn();
    const Cmd = Command.create({
      key: "cc",
      name: "Hook Command",
      icon: <Icon.Close />,
      useOnSelect: () => onSelect,
    });
    const { wrapper } = await createConsoleWrapper({ client: null });
    render(<Cmd key={Cmd.key} itemKey={Cmd.key} index={0} />, { wrapper });
    fireEvent.click(screen.getByText("Hook Command"), { detail: 1 });
    expect(onSelect).not.toHaveBeenCalled();
  });

  it("should show the shortcut of a command that a global trigger also runs", async () => {
    const Cmd = Command.create({
      key: "cc",
      name: "Hook Command",
      icon: <Icon.Close />,
      useOnSelect: () => vi.fn(),
      trigger: ["Control", "O"],
    });
    const { wrapper } = await createConsoleWrapper({ client: null });
    const c = render(<Cmd key={Cmd.key} itemKey={Cmd.key} index={0} />, { wrapper });
    // Some palette entries duplicate a shortcut bound elsewhere in the app. Without
    // the hint, the entry is the only place that shortcut is discoverable.
    expect(c.getByText("O")).toBeTruthy();
  });

  it("should leave a command with no shortcut unadorned", async () => {
    const Cmd = Command.create({
      key: "cc",
      name: "Hook Command",
      icon: <Icon.Close />,
      useOnSelect: () => vi.fn(),
    });
    const { wrapper } = await createConsoleWrapper({ client: null });
    const c = render(<Cmd key={Cmd.key} itemKey={Cmd.key} index={0} />, { wrapper });
    expect(c.getByText("Hook Command")).toBeTruthy();
    expect(c.container.querySelector(".pluto-text--keyboard")).toBeNull();
  });
});
