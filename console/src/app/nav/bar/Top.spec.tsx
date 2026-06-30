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

import { renderBar, TIMEOUT, withActiveProject } from "@/app/nav/bar/testutil";
import { Top } from "@/app/nav/bar/Top";

describe("app/nav/bar/Top", () => {
  it("should render the active project name in the selector", async () => {
    await renderBar(<Top />, withActiveProject());
    expect(await screen.findByText("Ops", {}, TIMEOUT)).toBeDefined();
  });

  it("should render the authenticated user from the live client", async () => {
    await renderBar(<Top />, withActiveProject());
    expect(await screen.findByText("synnax", {}, TIMEOUT)).toBeDefined();
  });
});
