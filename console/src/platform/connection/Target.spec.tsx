// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Connection } from "@/platform/connection";
import { createStatusConsoleWrapper, UNLICENSED_STATUS } from "@/testutil";

describe("Connection.Target", () => {
  it("should label a Core that answers without a license as unlicensed", async () => {
    const { wrapper } = await createStatusConsoleWrapper({
      client: null,
      status: UNLICENSED_STATUS,
    });
    render(<Connection.Target />, { wrapper });
    expect(await screen.findAllByText("Unlicensed")).not.toHaveLength(0);
    expect(screen.queryByText("Unreachable")).toBeNull();
  });
});
