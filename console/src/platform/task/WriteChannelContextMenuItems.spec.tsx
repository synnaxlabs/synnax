// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Menu } from "@synnaxlabs/pluto";
import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Task } from "@/platform/task";
import { renderWithConsole } from "@/testutil";

const channel: Task.WriteChannel = {
  key: "ch1",
  enabled: true,
  cmdChannel: 0,
  stateChannel: 0,
  cmdChannelName: "",
  stateChannelName: "",
};

describe("WriteChannelContextMenuItems", () => {
  it("should render rename items for the command and state channels", async () => {
    await renderWithConsole(
      <Menu.Menu>
        <Task.WriteChannelContextMenuItems keys={["ch1"]} channels={[channel]} />
      </Menu.Menu>,
    );
    expect(screen.getByText("Rename command channel")).toBeTruthy();
    expect(screen.getByText("Rename state channel")).toBeTruthy();
  });

  it("should render nothing when no matching channel is selected", async () => {
    await renderWithConsole(
      <Menu.Menu>
        <Task.WriteChannelContextMenuItems keys={["missing"]} channels={[channel]} />
      </Menu.Menu>,
    );
    expect(screen.queryByText("Rename command channel")).toBeNull();
  });

  it("should handle rename clicks for both channels without throwing", async () => {
    await renderWithConsole(
      <Menu.Menu>
        <Task.WriteChannelContextMenuItems keys={["ch1"]} channels={[channel]} />
      </Menu.Menu>,
    );
    fireEvent.click(screen.getByText("Rename command channel"));
    fireEvent.click(screen.getByText("Rename state channel"));
    expect(screen.getByText("Rename command channel")).toBeTruthy();
  });
});
