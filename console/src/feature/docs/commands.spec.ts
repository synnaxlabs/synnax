// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { renderPalette } from "@/feature/command/testutil";
import { Docs } from "@/feature/docs";
import { Session } from "@/session";
import { stubGeometry } from "@/testutil";

stubGeometry();

describe("Docs Commands", () => {
  it("should place the docs layout when the read command is selected", async () => {
    const { store, openCommandPalette, selectCommand } = await renderPalette({
      commands: Docs.COMMANDS,
    });
    await openCommandPalette();
    await selectCommand("Read the documentation");
    await waitFor(() =>
      expect(Session.Layout.select(store.getState(), Docs.LAYOUT_TYPE)?.type).toBe(
        Docs.LAYOUT_TYPE,
      ),
    );
  });
});
