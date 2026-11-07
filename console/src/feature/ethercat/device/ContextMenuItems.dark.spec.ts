// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

// The flags read the variable once, when they load, as in a Console build.
vi.hoisted(() => vi.stubEnv("VITE_FLAGS_ENABLED", "false"));

import { createSlave, renderContextMenu } from "@/feature/ethercat/device/testutil";

const client = createTestClient();

describe("EtherCAT device ContextMenuItems in a Console build", () => {
  it("should not offer Set sync mode", async () => {
    const slave = await createSlave(client, true);
    await renderContextMenu(client, [slave]);
    await screen.findByText("Disable");
    expect(screen.queryByText("Set sync mode")).toBeNull();
  });
});
