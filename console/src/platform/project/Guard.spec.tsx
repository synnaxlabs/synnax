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

import { Guard } from "@/platform/project/Guard";
import { Session } from "@/session";
import { renderWithConsole } from "@/testutil";

describe("project/Guard", () => {
  const selected = "00000000-0000-0000-0000-000000000001";

  it("should render the splash instead of children when no project is active", async () => {
    await renderWithConsole(
      <Guard>
        <div>protected content</div>
      </Guard>,
    );
    expect(screen.queryByText("protected content")).toBeNull();
    expect(screen.getByText("New Project")).toBeDefined();
  });

  it("should render children when a project is active", async () => {
    await renderWithConsole(
      <Guard>
        <div>protected content</div>
      </Guard>,
      {
        preloadedState: {
          [Session.Project.SLICE_NAME]: {
            ...Session.Project.ZERO_SLICE_STATE,
            selected,
          },
        },
      },
    );
    expect(screen.getByText("protected content")).toBeDefined();
    expect(screen.queryByText("New Project")).toBeNull();
  });
});
