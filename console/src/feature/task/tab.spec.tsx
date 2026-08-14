// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { query, type task } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Errors, Flux } from "@synnaxlabs/pluto";
import { act, screen, waitFor } from "@testing-library/react";
import { type ComponentType, type FC } from "react";
import { assert, describe, expect, it } from "vitest";

import { HTTP } from "@/feature/http";
import { Task } from "@/feature/task";
import { type Task as PTask } from "@/platform/task";
import { createTaskStatus, renderTaskFormTab } from "@/platform/task/testutil";
import { uniqueName } from "@/testutil";

const client = createTestClient();

// The mosaic renders tab content inside a boundary, and the content's read
// throws there once the task is deleted.
const renderContent = async (
  taskKey: task.Key,
  FallbackComponent?: ComponentType<Errors.FallbackProps>,
): Promise<void> => {
  const TabContent: FC<PTask.FormTabProps> = () => (
    <Errors.SuspenseBoundary FallbackComponent={FallbackComponent}>
      <Task.TAB.Content />
    </Errors.SuspenseBoundary>
  );
  TabContent.displayName = "TabContent";
  await renderTaskFormTab(TabContent, { client, taskKey });
};

const TabName: FC<PTask.FormTabProps> = () => <Task.TAB.Name />;

describe("task tab", () => {
  it("should render the form editor matching the task row's type", async () => {
    const { key: _key, ...zero } = HTTP.Task.ZERO_READ_PAYLOAD;
    const draft = await client.tasks.create({ ...zero, name: uniqueName("http") });
    await renderContent(draft.key);
    await screen.findByText("Add an endpoint");
  });

  it("should show an error for a task type with no registered editor", async () => {
    const created = await client.tasks.create({
      name: uniqueName("mystery"),
      type: "mystery_type",
      config: {},
    });
    await renderContent(created.key);
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

  it("throws the deleted task to the tab's boundary while it is open", async () => {
    const name = uniqueName("doomed");
    const created = await client.tasks.create({
      name,
      type: "mystery_type",
      config: {},
    });
    const DeletedProbe = ({ error }: Errors.FallbackProps) => (
      <div>{`deleted-${Flux.DeletedError.matches(error) ? error.corpseName : ""}`}</div>
    );
    DeletedProbe.displayName = "DeletedProbe";
    await renderContent(created.key, DeletedProbe);
    await screen.findByText("No editor for task type mystery_type");
    await act(async () => {
      await client.tasks.delete(created.key);
    });
    expect(await screen.findByText(`deleted-${name}`)).toBeTruthy();
  });

  it("restores a deleted task under its original key", async () => {
    const { restore } = Task.TAB;
    assert(restore != null);
    const name = uniqueName("restorable");
    const created = await client.tasks.create({
      name,
      type: "mystery_type",
      config: {},
    });
    await client.tasks.create({
      ...created.payload,
      status: createTaskStatus({ details: { task: created.key, running: true } }),
    });
    // Restore rebuilds from the corpse the cache holds, which exists only once
    // the client has seen the task live. The tab reads statuses, so the corpse
    // carries the running one the driver last reported.
    await client.tasks.retrieve({ key: created.key, includeStatus: true });
    await client.tasks.delete(created.key);
    await waitFor(() =>
      expect(query.Deleted.matches(client.tasks.getCached(created.key))).toBe(true),
    );

    const project = await client.projects.create({
      name: uniqueName("proj"),
      layout: {},
    });
    await restore({ client, project: project.key, resource: created.ontologyID });

    // Read back through a second client: the restoring client write-throughs its
    // own cache, so retrieving on it would pass even if the cluster never got it.
    const remote = createTestClient();
    const restored = await remote.tasks.retrieve({
      key: created.key,
      includeStatus: true,
    });
    expect(restored.name).toEqual(name);
    // The running instance died with the row, so core seeds the restored task as
    // never deployed instead of carrying the corpse's status forward.
    expect(restored.status?.variant).toEqual("disabled");
  });
});
