// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Menu } from "@synnaxlabs/pluto";
import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { ContextMenu } from "@/platform/context-menu";
import { renderWithConsole } from "@/testutil";

describe("ContextMenu.DeleteItem", () => {
  it("renders the Delete label", async () => {
    await renderWithConsole(
      <Menu.Menu>
        <ContextMenu.DeleteItem />
      </Menu.Menu>,
    );
    expect(screen.getByText("Delete")).toBeTruthy();
  });
});
