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
import { describe, expect, it, vi } from "vitest";

import { Task } from "@/platform/task";
import { renderWithConsole } from "@/testutil";

describe("CreateMenuItem", () => {
  it("should invoke onClick when selected", async () => {
    const onClick = vi.fn();
    await renderWithConsole(
      <Menu.Menu>
        <Task.CreateMenuItem itemKey="create" onClick={onClick}>
          Create Task
        </Task.CreateMenuItem>
      </Menu.Menu>,
    );
    fireEvent.click(screen.getByText("Create Task"));
    expect(onClick).toHaveBeenCalled();
  });
});
