// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc, createTestClient } from "@synnaxlabs/client";
import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Arc } from "@/feature/arc";
import { Modals } from "@/platform/modals";
import { assertDefined, createConsoleWrapper } from "@/testutil";

const client = createTestClient();

describe("arc/Selectable", () => {
  it("opens the arc create modal when the tile is clicked", async () => {
    const Selectable = Arc.SELECTABLES.find(
      (s) => s.type === arc.TYPE_ONTOLOGY_ID.type,
    );
    assertDefined(Selectable, "no selectable registered for the arc editor");
    const { wrapper } = await createConsoleWrapper({ client });
    render(
      <>
        <Selectable />
        <Modals.Stack />
      </>,
      { wrapper },
    );
    fireEvent.click(await screen.findByText("Arc Automation"));
    expect(await screen.findByPlaceholderText("Automation Name")).toBeTruthy();
  });
});
