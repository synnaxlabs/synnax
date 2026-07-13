// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { uuid } from "@synnaxlabs/x";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Table } from "@/feature/table";
import { client, project } from "@/feature/table/testutil";
import { createActiveState } from "@/platform/project/testutil";
import { Session } from "@/session";
import { createConsoleWrapper, waitForPlacedLayout } from "@/testutil";

describe("table Selectable", () => {
  it("creates a table for the pending layout key when clicked", async () => {
    const proj = await client.projects.retrieve(await project());
    const { wrapper, store } = await createConsoleWrapper({
      client,
      preloadedState: { [Session.Project.SLICE_NAME]: createActiveState(proj) },
    });
    const Selectable = Table.SELECTABLES[0];
    expect(Selectable.type).toBe(Table.LAYOUT_TYPE);
    const layoutKey = uuid.create();
    render(
      <Selectable
        layoutKey={layoutKey}
        rename={vi.fn(async () => null)}
        onPlace={vi.fn(() => ({ windowKey: "", key: "" }))}
        handleError={vi.fn()}
      />,
      { wrapper },
    );
    fireEvent.click(await screen.findByText("Table"));
    const placedKey = await waitForPlacedLayout(store, Table.LAYOUT_TYPE);
    expect(placedKey).toBe(layoutKey);
    await waitFor(async () => {
      const created = await client.tables.retrieve({ key: layoutKey });
      expect(created.name).toBe("Table");
    });
  });
});
