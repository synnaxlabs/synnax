// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Menu } from "@synnaxlabs/pluto";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { act } from "react";
import { describe, expect, it } from "vitest";

import { Task } from "@/platform/task";
import {
  renderInTaskForm,
  type RenderInTaskFormResult,
} from "@/platform/task/testutil";
import { awaitTextEditing, commitTextEdit } from "@/testutil";

const channel: Task.WriteChannel = {
  key: "ch1",
  enabled: true,
  cmdChannel: 0,
  stateChannel: 0,
  cmdChannelName: "",
  stateChannelName: "",
};

const renderMenuWithNames = async (): Promise<RenderInTaskFormResult> =>
  await renderInTaskForm(
    <>
      <Task.WriteChannelNames
        cmdChannel={0}
        stateChannel={0}
        cmdNamePath="config.cmdName"
        stateNamePath="config.stateName"
        itemKey="ch1"
      />
      <Menu.Menu>
        <Task.WriteChannelContextMenuItems keys={["ch1"]} channels={[channel]} />
      </Menu.Menu>
    </>,
    { values: { config: { cmdName: "cmd_before", stateName: "state_before" } } },
  );

describe("WriteChannelContextMenuItems", () => {
  it("should rename the command channel through the form", async () => {
    const { form } = await renderMenuWithNames();
    fireEvent.click(screen.getByText("Rename command channel"));
    const el = await awaitTextEditing(Task.getChannelNameID("ch1", "cmd"));
    act(() => commitTextEdit(el, "cmd_after"));
    await waitFor(() =>
      expect(form.current?.get("config.cmdName").value).toBe("cmd_after"),
    );
    expect(form.current?.get("config.stateName").value).toBe("state_before");
  });

  it("should rename the state channel through the form", async () => {
    const { form } = await renderMenuWithNames();
    fireEvent.click(screen.getByText("Rename state channel"));
    const el = await awaitTextEditing(Task.getChannelNameID("ch1", "state"));
    act(() => commitTextEdit(el, "state_after"));
    await waitFor(() =>
      expect(form.current?.get("config.stateName").value).toBe("state_after"),
    );
    expect(form.current?.get("config.cmdName").value).toBe("cmd_before");
  });

  it("should render nothing when no matching channel is selected", async () => {
    await renderInTaskForm(
      <Menu.Menu>
        <Task.WriteChannelContextMenuItems keys={["missing"]} channels={[channel]} />
      </Menu.Menu>,
    );
    expect(screen.queryByText("Rename command channel")).toBeNull();
    expect(screen.queryByText("Rename state channel")).toBeNull();
  });
});
