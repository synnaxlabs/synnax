// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.
import { waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import { renderPalette } from "@/feature/command/testutil";
import { Embedded } from "@/feature/embedded";
import { clearSupervisor, mockSupervisor } from "@/feature/embedded/testutil";

describe("Embedded commands", () => {
  afterEach(clearSupervisor);

  it("should ask the shell to show the logs", async () => {
    const { commands } = mockSupervisor(() => ({ state: "starting" }));
    const { openCommandPalette, selectCommand } = await renderPalette({
      commands: Embedded.COMMANDS,
    });
    await openCommandPalette();
    await selectCommand("Show logs");
    await waitFor(() => expect(commands).toHaveBeenCalledWith("supervisor_show_logs"));
  });
});
