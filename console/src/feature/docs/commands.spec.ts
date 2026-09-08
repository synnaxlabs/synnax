// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { describe, expect, it, vi } from "vitest";

import { renderPalette } from "@/feature/command/testutil";
import { Docs } from "@/feature/docs";

const client = createTestClient();

describe("Docs Commands", () => {
  it("should open the docs in the browser when the read command is selected", async () => {
    const open = vi.spyOn(window, "open").mockReturnValue(null);
    const { openCommandPalette, selectCommand } = await renderPalette({
      commands: Docs.COMMANDS,
      client,
    });
    await openCommandPalette();
    await selectCommand("Read documentation");
    await vi.waitFor(() => {
      expect(open).toHaveBeenCalledWith(Docs.URL, "_blank", "noopener,noreferrer");
    });
  });
});
