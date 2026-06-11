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

import { Guard } from "@/project/Guard";
import { SLICE_NAME } from "@/project/slice";
import { ZERO_SLICE_STATE } from "@/project/types";
import { renderWithConsole } from "@/testUtils";

describe("project/Guard", () => {
  const active = { key: "00000000-0000-0000-0000-000000000001", name: "Ops" };

  it("should render the splash instead of children when no project is active", () => {
    renderWithConsole(
      <Guard>
        <div>protected content</div>
      </Guard>,
      { preloadedState: { [SLICE_NAME]: ZERO_SLICE_STATE } },
    );
    expect(screen.queryByText("protected content")).toBeNull();
    expect(screen.getByText("New Project")).toBeDefined();
  });

  it("should render children when a project is active", () => {
    renderWithConsole(
      <Guard>
        <div>protected content</div>
      </Guard>,
      { preloadedState: { [SLICE_NAME]: { ...ZERO_SLICE_STATE, active } } },
    );
    expect(screen.getByText("protected content")).toBeDefined();
    expect(screen.queryByText("New Project")).toBeNull();
  });
});
