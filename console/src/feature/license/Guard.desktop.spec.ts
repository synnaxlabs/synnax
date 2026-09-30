// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { renderGuard } from "@/feature/license/testutil";
import { UNLICENSED_STATUS } from "@/testutil";

describe("License.Guard", () => {
  it("should offer no log out action", async () => {
    await renderGuard(null, UNLICENSED_STATUS);
    expect(screen.getByText("This Core needs a license")).toBeTruthy();
    expect(screen.queryByText("Log out")).toBeNull();
  });
});
