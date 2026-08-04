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
import { beforeEach, describe, expect, it } from "vitest";

import { ContextMenu } from "@/platform/context-menu";
import { Link } from "@/platform/link";
import { renderWithConsole } from "@/testutil";

describe("ContextMenu.ReloadConsoleItem", () => {
  beforeEach(() => localStorage.removeItem(Link.SHOULD_IGNORE_KEY));

  it("flags the next deep link as ignored before reloading", async () => {
    await renderWithConsole(
      <Menu.Menu>
        <ContextMenu.ReloadConsoleItem />
      </Menu.Menu>,
    );
    expect(localStorage.getItem(Link.SHOULD_IGNORE_KEY)).toBeNull();
    fireEvent.click(screen.getByText("Reload console"));
    expect(localStorage.getItem(Link.SHOULD_IGNORE_KEY)).toBe("true");
  });
});
