// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Task } from "@/platform/task";
import { renderInTaskForm } from "@/platform/task/testutil";

describe("WriteChannelNames", () => {
  it("should render default names for both command and state channels", async () => {
    await renderInTaskForm(
      <Task.WriteChannelNames
        cmdChannel={0}
        stateChannel={0}
        cmdNamePath="config.cmdName"
        stateNamePath="config.stateName"
        itemKey="chan1"
      />,
      { values: { config: { cmdName: "", stateName: "" } } },
    );
    await waitFor(() => expect(screen.getByText("No command channel")).toBeTruthy());
    expect(screen.getByText("No state channel")).toBeTruthy();
  });

  it("should render the form names when provided", async () => {
    await renderInTaskForm(
      <Task.WriteChannelNames
        cmdChannel={0}
        stateChannel={0}
        cmdNamePath="config.cmdName"
        stateNamePath="config.stateName"
        itemKey="chan1"
      />,
      { values: { config: { cmdName: "My Command", stateName: "My State" } } },
    );
    await waitFor(() => expect(screen.getByText("My Command")).toBeTruthy());
    expect(screen.getByText("My State")).toBeTruthy();
  });
});
