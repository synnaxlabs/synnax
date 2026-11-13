// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { type Status } from "@synnaxlabs/lyra/status";
import { stubClipboardWriteText } from "@synnaxlabs/lyra/testutil";
import { fireEvent, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, type Mock } from "vitest";
import { z } from "zod";

import { Task } from "@/platform/task";
import { renderInTaskForm, renderInTaskFormWithClient } from "@/platform/task/testutil";
import { Session } from "@/session";
import { createCore, createCoreState } from "@/session/core/testutil";
import {
  assertDefined,
  CaptureStatuses,
  getIconButton,
  queryIcon,
  uniqueName,
} from "@/testutil";

const clickIcon = (container: HTMLElement, icon: string): void => {
  fireEvent.click(getIconButton(container, icon));
};

describe("UtilityButtons", () => {
  let writeText: Mock;

  beforeEach(() => {
    writeText = stubClipboardWriteText();
  });

  it("should only show the JSON config button when the task has no key", async () => {
    const { container } = await renderInTaskForm(<Task.UtilityButtons />, {
      values: { key: undefined, name: "T", config: {} },
    });
    expect(queryIcon(container, "json")).toBeTruthy();
    expect(queryIcon(container, "typescript")).toBeNull();
    expect(queryIcon(container, "python")).toBeNull();
    expect(queryIcon(container, "link")).toBeNull();
    expect(queryIcon(container, "export")).toBeNull();
  });

  it("should show the code, link, and export buttons when the task has a key", async () => {
    const { container } = await renderInTaskForm(<Task.UtilityButtons />, {
      values: { key: "task_123", name: "T", config: {} },
    });
    expect(queryIcon(container, "json")).toBeTruthy();
    expect(queryIcon(container, "typescript")).toBeTruthy();
    expect(queryIcon(container, "python")).toBeTruthy();
    expect(queryIcon(container, "link")).toBeTruthy();
    expect(queryIcon(container, "export")).toBeTruthy();
    expect(queryIcon(container, "line-plot")).toBeNull();
  });

  describe("plot", () => {
    const client = createTestClient();

    const createChannel = async (dataType: string, isIndex = false) =>
      (
        await client.channels.create({
          name: uniqueName("value"),
          dataType,
          isIndex,
          virtual: !isIndex,
        })
      ).key;

    const configZ = z.object({
      channels: z.array(z.object({ channel: z.number(), disabled: z.boolean() })),
    });
    const plotChannels: Task.PlotChannels = (config) =>
      Task.enabledChannels(configZ.parse(config).channels);

    const renderPlot = async (
      channels: { channel: number; disabled: boolean }[],
      onStatuses?: (statuses: Status.NotificationSpec[]) => void,
    ) => {
      const { container, store } = await renderInTaskFormWithClient(
        <>
          <Task.UtilityButtons plotChannels={plotChannels} />
          {onStatuses != null && <CaptureStatuses onStatuses={onStatuses} />}
        </>,
        { client, values: { key: "task_123", name: "My Task", config: { channels } } },
      );
      const proj = await client.projects.create({ name: uniqueName("proj") });
      store.dispatch(Session.Project.select(proj.key));
      clickIcon(container, "line-plot");
      return proj;
    };

    const findPlot = async (project: string) =>
      await waitFor(async () => {
        const [created] = await client.lineplots.retrieve({ projects: [project] });
        assertDefined(created, "the line plot was not created");
        return created;
      });

    it("should plot the enabled channels that the form's config holds", async () => {
      const key = await createChannel("float32");
      const proj = await renderPlot([
        { channel: key, disabled: false },
        { channel: key + 1, disabled: true },
        { channel: 0, disabled: false },
      ]);
      const plot = await findPlot(proj.key);
      expect(plot.name).toBe("Plot for My Task");
      expect(plot.channels.y1).toEqual([key]);
    });

    it("should leave out index channels and channels that are not numeric", async () => {
      const [value, index, text] = await Promise.all([
        createChannel("float32"),
        createChannel("timestamp", true),
        createChannel("string"),
      ]);
      const proj = await renderPlot(
        [value, index, text].map((channel) => ({ channel, disabled: false })),
      );
      expect((await findPlot(proj.key)).channels.y1).toEqual([value]);
    });

    it("should report a task with no numeric channel to plot", async () => {
      const text = await createChannel("string");
      let statuses: Status.NotificationSpec[] = [];
      await renderPlot([{ channel: text, disabled: false }], (s) => {
        statuses = s;
      });
      await waitFor(() =>
        expect(statuses).toContainEqual(
          expect.objectContaining({
            message: "Failed to plot task",
            description: "The task has no enabled channels with numeric values.",
          }),
        ),
      );
    });
  });

  it("should copy TypeScript code that retrieves the task", async () => {
    const { container } = await renderInTaskForm(<Task.UtilityButtons />, {
      values: { key: "task_123", name: "My Task", config: {} },
    });
    clickIcon(container, "typescript");
    await waitFor(() => expect(writeText).toHaveBeenCalledTimes(1));
    const code = writeText.mock.calls[0][0];
    expect(code).toContain('const task = client.tasks.retrieve("task_123")');
    expect(code).toContain("// Retrieve My Task");
  });

  it("should copy Python code that retrieves the task", async () => {
    const { container } = await renderInTaskForm(<Task.UtilityButtons />, {
      values: { key: "task_123", name: "My Task", config: {} },
    });
    clickIcon(container, "python");
    await waitFor(() => expect(writeText).toHaveBeenCalledTimes(1));
    const code = writeText.mock.calls[0][0];
    expect(code).toContain('task = client.tasks.retrieve("task_123")');
    expect(code).toContain("# Retrieve My Task");
  });

  it("should copy the JSON-encoded task configuration", async () => {
    const { container } = await renderInTaskForm(<Task.UtilityButtons />, {
      values: { name: "My Task", config: { device: "dev_1", sampleRate: 25 } },
    });
    clickIcon(container, "json");
    await waitFor(() => expect(writeText).toHaveBeenCalledTimes(1));
    expect(JSON.parse(writeText.mock.calls[0][0])).toEqual({
      channels: [],
      device: "dev_1",
      sample_rate: 25,
    });
  });

  it("should copy a link containing the active cluster and the task key", async () => {
    const core = createCore("Alpha", { clusterKey: "cluster-1" });
    const { container } = await renderInTaskForm(<Task.UtilityButtons />, {
      values: { key: "task_123", name: "My Task", config: {} },
      preloadedState: createCoreState([core], core.key),
    });
    clickIcon(container, "link");
    await waitFor(() => expect(writeText).toHaveBeenCalledTimes(1));
    expect(writeText.mock.calls[0][0]).toBe("synnax://cluster/cluster-1/task/task_123");
  });
});
