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

import { renderCommands } from "@/app/command/testutil";

describe("app/command/List", () => {
  it("should offer the Core commands and no Desktop commands", async () => {
    await renderCommands("lo");
    expect(await screen.findByText("Log out")).toBeTruthy();
    expect(screen.queryByText("Show logs")).toBeNull();
  });
});
