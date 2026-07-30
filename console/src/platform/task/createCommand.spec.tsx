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
import { waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Task } from "@/platform/task";
import { renderHookWithConsole } from "@/testutil";

const client = createTestClient();

const Command = Task.createCommand({
  key: "test_create_task",
  name: "Create a Test Task",
  icon: <Icon.Task />,
  type: "test_task_form",
});

describe("Task.createCommand", () => {
  it("is visible when the user may create tasks", async () => {
    const { result } = await renderHookWithConsole(() => Command.useVisible?.(), {
      client,
    });
    await waitFor(() => expect(result.current).toBe(true));
  });
});
