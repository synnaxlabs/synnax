// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology } from "@synnaxlabs/client";
import { Menu } from "@synnaxlabs/pluto";
import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Export } from "@/platform/export";
import { renderWithConsole } from "@/testutil";

const ID: ontology.ID = { type: "log", key: "k" };

describe("Export.MenuItem", () => {
  it("renders an Export menu entry for the given id", async () => {
    await renderWithConsole(
      <Menu.Menu>
        <Export.MenuItem id={ID} />
      </Menu.Menu>,
    );
    expect(screen.getByText("Export")).toBeTruthy();
  });
});
