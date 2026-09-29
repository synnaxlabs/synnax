// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Component } from "@synnaxlabs/lyra/component";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Select } from "@synnaxlabs/lyra/select";
import { act, fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Analytics } from "@/platform/analytics";
import { Command } from "@/platform/command";
import { createConsoleWrapper } from "@/testutil";

const renderCommand = async (
  Cmd: Command.Command,
  sink: Analytics.Sink = Analytics.NOOP,
): Promise<void> => {
  const { wrapper } = await createConsoleWrapper({ client: null });
  render(
    <Analytics.Provider sink={sink}>
      <Select.Frame<string, undefined> data={[Cmd.key]} onChange={vi.fn()}>
        <List.Scroll>
          <List.Items<string>>{Component.renderProp(Cmd)}</List.Items>
        </List.Scroll>
      </Select.Frame>
    </Analytics.Provider>,
    { wrapper },
  );
};

describe("Command.create", () => {
  it("should invoke the hook-produced callback when the command is selected", async () => {
    const onSelect = vi.fn();
    const Cmd = Command.create({
      key: "cc",
      name: "Hook Command",
      icon: <Icon.Close />,
      useOnSelect: () => onSelect,
    });
    await renderCommand(Cmd);
    await act(async () => {
      fireEvent.click(screen.getByText("Hook Command"));
    });
    expect(onSelect).toHaveBeenCalledTimes(1);
  });

  it("should record the key of the command that ran", async () => {
    const capture = vi.fn();
    const Cmd = Command.create({
      key: "cc",
      name: "Hook Command",
      icon: <Icon.Close />,
      useOnSelect: () => vi.fn(),
    });
    await renderCommand(Cmd, { ...Analytics.NOOP, capture });
    await act(async () => {
      fireEvent.click(screen.getByText("Hook Command"));
    });
    expect(capture).toHaveBeenCalledWith("command_run", { command: "cc" });
  });

  it("should show the shortcut of a command that a global trigger also runs", async () => {
    const Cmd = Command.create({
      key: "cc",
      name: "Hook Command",
      icon: <Icon.Close />,
      useOnSelect: () => vi.fn(),
      trigger: ["Control", "O"],
    });
    await renderCommand(Cmd);
    // Some palette entries duplicate a shortcut bound elsewhere in the app. Without
    // the hint, the entry is the only place that shortcut is discoverable.
    expect(screen.getByText("O")).toBeTruthy();
  });

  it("should leave a command with no shortcut unadorned", async () => {
    const Cmd = Command.create({
      key: "cc",
      name: "Hook Command",
      icon: <Icon.Close />,
      useOnSelect: () => vi.fn(),
    });
    await renderCommand(Cmd);
    expect(screen.getByText("Hook Command")).toBeTruthy();
    expect(document.querySelector(".pluto-text--keyboard")).toBeNull();
  });
});
