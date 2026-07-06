// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Menu as PMenu } from "@synnaxlabs/pluto";
import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { ContextMenu } from "@/platform/context-menu";
import { renderWithConsole } from "@/testutil";

describe("ContextMenu.Menu", () => {
  it("renders its children", async () => {
    await renderWithConsole(
      <ContextMenu.Menu>
        <PMenu.Item itemKey="child">Child Item</PMenu.Item>
      </ContextMenu.Menu>,
    );
    expect(screen.getByText("Child Item")).toBeTruthy();
  });
});
