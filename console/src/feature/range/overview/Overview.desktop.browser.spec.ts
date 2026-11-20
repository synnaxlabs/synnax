// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { renderOverview } from "@/feature/range/overview/testutil";
import { createTestRange } from "@/platform/range/testutil";

const client = createTestClient();

describe("range/overview", () => {
  it("should offer no copy link in the more menu when links are disabled", async () => {
    const rng = await createTestRange(client);
    await renderOverview(rng.key);
    fireEvent.click(await screen.findByRole("button", { name: "More" }));
    await screen.findByRole("menuitem", { name: "Download as CSV" });
    expect(screen.queryByRole("menuitem", { name: "Copy link" })).toBeNull();
  });
});
