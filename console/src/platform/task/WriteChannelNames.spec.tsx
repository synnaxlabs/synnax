// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Task } from "@/platform/task";
import {
  createTestChannel,
  renderInTaskForm,
  renderInTaskFormWithClient,
} from "@/platform/task/testutil";

const NAME_PATHS = {
  cmdNamePath: "config.cmdName",
  stateNamePath: "config.stateName",
  itemKey: "chan1",
};

const unresolved = { device: undefined, resolve: () => ({ command: 0, state: 0 }) };

describe("WriteChannelNames", () => {
  it("should render default names for both command and state channels", async () => {
    await renderInTaskForm(
      <Task.WriteChannelNames
        cmdChannel={0}
        stateChannel={0}
        {...NAME_PATHS}
        {...unresolved}
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
        {...NAME_PATHS}
        {...unresolved}
      />,
      { values: { config: { cmdName: "My Command", stateName: "My State" } } },
    );
    await waitFor(() => expect(screen.getByText("My Command")).toBeTruthy());
    expect(screen.getByText("My State")).toBeTruthy();
  });

  describe("resolving from the device map", () => {
    const client = createTestClient();
    const values = { config: { cmdName: "", stateName: "" } };

    const createPair = async () => ({
      command: await createTestChannel(client, "cmd"),
      state: await createTestChannel(client, "state"),
    });

    it("should split the resolved pair across the command and state names", async () => {
      const pair = await createPair();
      await renderInTaskFormWithClient(
        <Task.WriteChannelNames
          cmdChannel={0}
          stateChannel={0}
          {...NAME_PATHS}
          device={{}}
          resolve={() => ({ command: pair.command.key, state: pair.state.key })}
        />,
        { client, values },
      );
      await screen.findByText(pair.command.name);
      await screen.findByText(pair.state.name);
    });

    it("should keep the row's channels while the device is undefined", async () => {
      const pair = await createPair();
      await renderInTaskFormWithClient(
        <Task.WriteChannelNames
          cmdChannel={pair.command.key}
          stateChannel={pair.state.key}
          {...NAME_PATHS}
          {...unresolved}
        />,
        { client, values },
      );
      await screen.findByText(pair.command.name);
      await screen.findByText(pair.state.name);
    });

    it("should show no channels when the resolver finds none, even if the row holds them", async () => {
      const pair = await createPair();
      await renderInTaskFormWithClient(
        <Task.WriteChannelNames
          cmdChannel={pair.command.key}
          stateChannel={pair.state.key}
          {...NAME_PATHS}
          device={{}}
          resolve={() => ({ command: 0, state: 0 })}
        />,
        { client, values },
      );
      await screen.findByText("No command channel");
      await screen.findByText("No state channel");
      expect(screen.queryByText(pair.command.name)).toBeNull();
      expect(screen.queryByText(pair.state.name)).toBeNull();
    });

    it("should resolve the command and state independently", async () => {
      const pair = await createPair();
      await renderInTaskFormWithClient(
        <Task.WriteChannelNames
          cmdChannel={0}
          stateChannel={pair.state.key}
          {...NAME_PATHS}
          device={{}}
          resolve={() => ({ command: pair.command.key, state: 0 })}
        />,
        { client, values },
      );
      await screen.findByText(pair.command.name);
      await screen.findByText("No state channel");
      expect(screen.queryByText(pair.state.name)).toBeNull();
    });
  });
});
