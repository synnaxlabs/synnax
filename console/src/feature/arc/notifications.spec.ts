// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Arc } from "@/feature/arc";
import {
  renderForward,
  settleWrites,
  writeStatus,
} from "@/platform/notifications/testutil";
import { uniqueName } from "@/testutil";

const client = createTestClient();

describe("Arc.NOTIFICATION_RULE", () => {
  it("should not notify a module compile error", async () => {
    await renderForward(client, { arc: Arc.NOTIFICATION_RULE });
    const module = await client.arcs.create({
      name: uniqueName("module"),
      type: "module",
      source: { mode: "text" },
    });
    const message = await writeStatus(client, arc.resourceID(module.key), "error");
    await settleWrites(client);
    expect(screen.queryByText(message)).toBeNull();
  });
});
