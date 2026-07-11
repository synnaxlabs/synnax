// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { Icon } from "@synnaxlabs/pluto";
import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Task } from "@/feature/task";
import { Selector } from "@/platform/selector";
import { createConsoleWrapper } from "@/testutil";

const client = createTestClient();

const { Content } = Selector.create({
  selectables: Task.SELECTABLES,
  tabTitle: "Task",
  text: "Select a Task Type",
  icon: <Icon.Task />,
});

const renderSelector = async (renderClient: typeof client | null) => {
  const { wrapper } = await createConsoleWrapper({ client: renderClient });
  render(<Content />, { wrapper });
};

describe("task/Selector", () => {
  it("should render vendor task types once creation is granted", async () => {
    await renderSelector(client);
    expect(await screen.findByText("NI Analog Read Task")).toBeTruthy();
    expect(screen.getByText("Select a Task Type")).toBeTruthy();
    expect(screen.getByText("OPC UA Read Task")).toBeTruthy();
    expect(screen.getByText("LabJack Read Task")).toBeTruthy();
  });

  it("should render no vendor task types without a client to grant creation", async () => {
    await renderSelector(null);
    expect(await screen.findByText("Select a Task Type")).toBeTruthy();
    expect(screen.queryByText("NI Analog Read Task")).toBeNull();
  });
});
