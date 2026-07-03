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
import { renderInTaskForm } from "@/platform/task/testutil";
import { awaitTextEditing, commitTextEdit } from "@/testutil";

describe("ReadChannelContextMenuItem", () => {
  it("should rename the selected channel through the form", async () => {
    const { form } = await renderInTaskForm(
      <>
        <Task.ChannelName
          channel={0}
          namePath="config.name"
          id={Task.getChannelNameID("ch1")}
        />
        <Menu.Menu>
          <Task.ReadChannelContextMenuItem keys={["ch1"]} channels={[]} />
        </Menu.Menu>
      </>,
      { values: { config: { name: "name_before" } } },
    );
    fireEvent.click(screen.getByText("Rename"));
    const el = await awaitTextEditing(Task.getChannelNameID("ch1"));
    act(() => commitTextEdit(el, "name_after"));
    await waitFor(() =>
      expect(form.current?.get("config.name").value).toBe("name_after"),
    );
  });

  it("should render nothing when multiple channels are selected", async () => {
    await renderInTaskForm(
      <Menu.Menu>
        <Task.ReadChannelContextMenuItem keys={["ch1", "ch2"]} channels={[]} />
      </Menu.Menu>,
    );
    expect(screen.queryByText("Rename")).toBeNull();
  });
});
