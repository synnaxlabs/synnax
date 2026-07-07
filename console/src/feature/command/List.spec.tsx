// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";
import { screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Command } from "@/feature/command";
import { renderPalette } from "@/feature/command/testutil";
import { stubGeometry } from "@/testutil";

stubGeometry();

const command = (
  key: string,
  name: string,
  opts: Partial<Pick<Command.CreateParams, "useVisible" | "sortOrder">> = {},
): Command.Command =>
  Command.create({
    key,
    name,
    icon: <Icon.Close />,
    useOnSelect: () => () => {},
    ...opts,
  });

describe("Command.createList", () => {
  it("should hide commands whose useVisible returns false", async () => {
    const { openCommandPalette } = await renderPalette({
      commands: [
        command("a", "Alpha"),
        command("b", "Beta", { useVisible: () => false }),
      ],
    });
    await openCommandPalette();
    expect(await screen.findByText("Alpha")).toBeTruthy();
    await waitFor(() => expect(screen.queryByText("Beta")).toBeNull());
  });

  it("should order commands by sortOrder ahead of their name", async () => {
    const { openCommandPalette } = await renderPalette({
      commands: [
        command("z", "Zeta", { sortOrder: 1 }),
        command("a", "Alpha", { sortOrder: 2 }),
      ],
    });
    await openCommandPalette();
    const zeta = await screen.findByText("Zeta");
    const alpha = screen.getByText("Alpha");
    expect(
      zeta.compareDocumentPosition(alpha) & Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy();
  });
});
