// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, DataType } from "@synnaxlabs/client";
import { id } from "@synnaxlabs/x";
import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Task } from "@/platform/task";
import { renderInTaskForm, TaskFormProvider } from "@/platform/task/testutil";
import { createConsoleWrapper } from "@/testutil";

describe("ChannelName", () => {
  it("should render the default name when no channel is selected", async () => {
    await renderInTaskForm(
      <Task.ChannelName channel={0} namePath="config.name" defaultName="No channel" />,
      { values: { config: { name: "" } } },
    );
    await waitFor(() => expect(screen.getByText("No channel")).toBeTruthy());
  });

  it("should prefer the form name over the default when set", async () => {
    await renderInTaskForm(
      <Task.ChannelName channel={0} namePath="config.name" defaultName="No channel" />,
      { values: { config: { name: "Manually Named" } } },
    );
    await waitFor(() => expect(screen.getByText("Manually Named")).toBeTruthy());
  });

  describe("with a live client", () => {
    it("should resolve and display the name of a real channel", async () => {
      const client = createTestClient();
      const { wrapper } = await createConsoleWrapper({ client });
      const ch = await client.channels.create({
        name: id.create(),
        dataType: DataType.FLOAT32,
        virtual: true,
      });
      render(
        <TaskFormProvider values={{ name: "" }}>
          <Task.ChannelName channel={ch.key} namePath="name" />
        </TaskFormProvider>,
        { wrapper },
      );
      await waitFor(() => expect(screen.getByText(ch.name)).toBeTruthy(), {
        timeout: 5000,
      });
    });
  });
});
