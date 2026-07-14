// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { screen } from "@testing-library/react";
import { type FC } from "react";
import { describe, it } from "vitest";

import { HTTP } from "@/feature/http";
import { Task } from "@/feature/task";
import { type Task as PTask } from "@/platform/task";
import { renderTaskFormTab } from "@/platform/task/testutil";
import { uniqueName } from "@/testutil";

const client = createTestClient();

const TabContent: FC<PTask.FormTabProps> = () => <Task.TAB.Content />;
const TabName: FC<PTask.FormTabProps> = () => <Task.TAB.Name />;

describe("task tab", () => {
  it("should render the form editor matching the task row's type", async () => {
    const { key: _key, ...zero } = HTTP.Task.ZERO_READ_PAYLOAD;
    const draft = await client.tasks.create({ ...zero, name: uniqueName("http") });
    await renderTaskFormTab(TabContent, { client, taskKey: draft.key });
    await screen.findByText("Add an endpoint");
  });

  it("should show an error for a task type with no registered editor", async () => {
    const created = await client.tasks.create({
      name: uniqueName("mystery"),
      type: "mystery_type",
      config: {},
    });
    await renderTaskFormTab(TabContent, { client, taskKey: created.key });
    await screen.findByText("No editor for task type mystery_type");
  });

  it("should render the task's name in the tab", async () => {
    const name = uniqueName("named");
    const created = await client.tasks.create({
      name,
      type: "mystery_type",
      config: {},
    });
    await renderTaskFormTab(TabName, { client, taskKey: created.key });
    await screen.findByText(name);
  });
});
