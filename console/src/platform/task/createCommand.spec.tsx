// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { task } from "@synnaxlabs/client";
import { createTestClient, RoleClients } from "@synnaxlabs/client/testutil";
import { Access, Icon } from "@synnaxlabs/pluto";
import { waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Task } from "@/platform/task";
import { renderHookWithConsole } from "@/testutil";

const client = createTestClient();
const roles = new RoleClients(client);

const Command = Task.createCommand({
  key: "test_create_task",
  name: "Create a Test Task",
  icon: <Icon.Task />,
  useOnSelect: () => () => {},
});

describe("Task.createCommand", () => {
  it("is visible when the user may create tasks", async () => {
    const { result } = await renderHookWithConsole(() => Command.useVisible?.(), {
      client,
    });
    await waitFor(() => expect(result.current).toBe(true));
  });
});

describe("Task.createCommand permissions", () => {
  it("should withhold the command from a viewer", async () => {
    const viewer = await roles.get("Viewer");
    const { result } = await renderHookWithConsole(
      () => ({
        visible: Command.useVisible?.(),
        loaded: Access.useRetrieveGranted(task.TYPE_ONTOLOGY_ID),
      }),
      { client: viewer },
    );
    await waitFor(() => expect(result.current.loaded).toBe(true));
    expect(result.current.visible).toBe(false);
  });

  // Starting and stopping tasks is an operator's job, but minting one is not.
  it("should withhold the command from an operator", async () => {
    const { result } = await renderHookWithConsole(() => Command.useVisible?.(), {
      client: await roles.get("Operator"),
    });
    await waitFor(() => expect(result.current).toBe(false));
  });

  it("should offer the command to an engineer", async () => {
    const { result } = await renderHookWithConsole(() => Command.useVisible?.(), {
      client: await roles.get("Engineer"),
    });
    await waitFor(() => expect(result.current).toBe(true));
  });
});
