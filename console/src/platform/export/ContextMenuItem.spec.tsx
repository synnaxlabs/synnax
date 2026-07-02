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

import { Export } from "@/platform/export";
import { renderWithConsole } from "@/testutil/testutil";

describe("Export.ContextMenuItem", () => {
  it("renders an Export menu entry", async () => {
    await renderWithConsole(
      <Menu.Menu>
        <Export.ContextMenuItem />
      </Menu.Menu>,
    );
    expect(screen.getByText("Export")).toBeTruthy();
  });

  it("invokes onClick when the entry is selected", async () => {
    const onClick = vi.fn();
    await renderWithConsole(
      <Menu.Menu>
        <Export.ContextMenuItem onClick={onClick} />
      </Menu.Menu>,
    );
    fireEvent.click(screen.getByText("Export"));
    expect(onClick).toHaveBeenCalledTimes(1);
  });
});
