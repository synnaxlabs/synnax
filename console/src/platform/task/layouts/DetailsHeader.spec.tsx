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

import { Task } from "@/platform/task";
import { renderInTaskForm } from "@/platform/task/testutil";

describe("layouts.DetailsHeader", () => {
  it("should render the Details title and a copy button", async () => {
    const { container } = await renderInTaskForm(
      <Task.Layouts.DetailsHeader path="config" />,
      { values: { config: { channels: [] } } },
    );
    expect(screen.getByText("Details")).toBeTruthy();
    expect(container.querySelector('[aria-label="pluto-icon--json"]')).toBeTruthy();
  });

  it("should disable the copy button when disabled", async () => {
    await renderInTaskForm(<Task.Layouts.DetailsHeader path="config" disabled />, {
      values: { config: { channels: [] } },
    });
    expect(screen.getByRole("button").className).toContain("pluto--disabled");
  });
});
