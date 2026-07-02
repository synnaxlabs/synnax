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

describe("ReadChannelContextMenuItem", () => {
  it("should render a Rename item for a single selected channel", async () => {
    await renderWithConsole(
      <Menu.Menu>
        <Task.ReadChannelContextMenuItem keys={["ch1"]} channels={[]} />
      </Menu.Menu>,
    );
    expect(screen.getByText("Rename")).toBeTruthy();
  });

  it("should render nothing when multiple channels are selected", async () => {
    await renderWithConsole(
      <Menu.Menu>
        <Task.ReadChannelContextMenuItem keys={["ch1", "ch2"]} channels={[]} />
      </Menu.Menu>,
    );
    expect(screen.queryByText("Rename")).toBeNull();
  });

  it("should handle a rename click without throwing", async () => {
    await renderWithConsole(
      <Menu.Menu>
        <Task.ReadChannelContextMenuItem keys={["ch1"]} channels={[]} />
      </Menu.Menu>,
    );
    fireEvent.click(screen.getByText("Rename"));
    expect(screen.getByText("Rename")).toBeTruthy();
  });
});
