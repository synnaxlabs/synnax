// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot, panel } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import type { Status } from "@synnaxlabs/lyra/status";
import { stubClipboardWriteText } from "@synnaxlabs/lyra/testutil";
import { deep } from "@synnaxlabs/x";
import { assertDefined, uniqueName } from "@synnaxlabs/x/testutil";
import { act, fireEvent, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, type Mock } from "vitest";
import { z } from "zod";

import { Task } from "@/platform/task";
import {
  createRegistry,
  renderInTaskForm,
  renderInTaskFormWithClient,
} from "@/platform/task/testutil";
import { Session } from "@/session";
import { createCore, createCoreState } from "@/session/core/testutil";
import { CaptureStatuses, getIconButton, queryIcon } from "@/testutil";

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
    const TYPE = "test_read";
    const plotChannels: Task.PlotChannels = (config) =>
      Task.enabledChannels(configZ.parse(config).channels);
    const ICD_TYPE = "test_icd_read";
    const icdConfigZ = z.object({
      channels: z.array(
        z.object({
          channel: z.number(),
          field: z.object({ icd: z.string(), message: z.string(), key: z.string() }),
        }),
      ),
    });
    const icdPlotChannels: Task.PlotChannels = (config) =>
      icdConfigZ.parse(config).channels;

    const renderPlot = async (
      channels: object[],
      onStatuses?: (statuses: Status.NotificationSpec[]) => void,
      type: string = TYPE,
    ) => {
      const { container, store, form, panelKey } = await renderInTaskFormWithClient(
        <>
          <Task.UtilityButtons />
          {onStatuses != null && <CaptureStatuses onStatuses={onStatuses} />}
        </>,
        {
          client,
          values: {
            key: "task_123",
            name: "My Task",
            type,
            config: { channels },
          },
          registry: createRegistry({
            plotChannels: { [TYPE]: plotChannels, [ICD_TYPE]: icdPlotChannels },
          }),
        },
      );
      const proj = await client.projects.create({ name: uniqueName("proj") });
      store.dispatch(Session.Project.select(proj.key));
      clickIcon(container, "line-plot");
      return { proj, container, form, panelKey };
    };

    const path = deep.path<{ config: { channels: object[] } }>();

    const findPlot = async (project: string) =>
      await waitFor(async () => {
        const [created] = await client.lineplots.retrieve({ projects: [project] });
        assertDefined(created, "the line plot was not created");
        return created;
      });

    it("should plot the enabled channels that the form's config holds", async () => {
      const key = await createChannel("float32");
      const { proj } = await renderPlot([
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
      const { proj } = await renderPlot(
        [value, index, text].map((channel) => ({ channel, disabled: false })),
      );
      expect((await findPlot(proj.key)).channels.y1).toEqual([value]);
    });

    /** @returns a channel for each units, filled by an ICD field with those units. */
    const createFieldChannels = async (units: string[]) => {
      const { key, messages } = await client.icds.create({
        name: uniqueName("icd"),
        messages: [
          {
            name: "m",
            payload: {
              format: "json",
              fields: units.map((u, i) => ({
                name: `f${i}`,
                pointer: `/f${i}`,
                type: "float",
                units: u,
              })),
            },
          },
        ],
      });
      const [message] = messages;
      return await Promise.all(
        message.payload.fields.map(async (field) => ({
          channel: await createChannel("float32"),
          field: { icd: key, message: message.key, key: field.key },
        })),
      );
    };

    it("should give each unit an axis labeled with it, and channels with no units y1", async () => {
      const fields = await createFieldChannels(["rpm", "", "degC", "rpm"]);
      const { proj } = await renderPlot(fields, undefined, ICD_TYPE);
      const { channels, axes } = await findPlot(proj.key);
      const [rpm, none, degC, rpm2] = fields.map(({ channel }) => channel);
      expect(channels).toMatchObject({
        y1: [none],
        y2: [rpm, rpm2],
        y3: [degC],
        y4: [],
      });
      expect([axes.y1.label, axes.y2.label, axes.y3.label]).toEqual([
        "",
        "rpm",
        "degC",
      ]);
    });

    it("should put the units past the fourth axis on y4", async () => {
      const fields = await createFieldChannels(["V", "A", "Pa", "K", "Hz"]);
      const { proj } = await renderPlot(fields, undefined, ICD_TYPE);
      const { channels, axes } = await findPlot(proj.key);
      const keys = fields.map(({ channel }) => channel);
      expect(channels).toMatchObject({
        y1: [keys[0]],
        y2: [keys[1]],
        y3: [keys[2]],
        y4: [keys[3], keys[4]],
      });
      expect(axes.y4.label).toBe("K, Hz");
    });

    it("should reopen the plot it made, with a new unit on its own axis", async () => {
      const [rpm, degC] = await createFieldChannels(["rpm", "degC"]);
      const { proj, container, form } = await renderPlot([rpm], undefined, ICD_TYPE);
      const first = await findPlot(proj.key);
      act(() => form.current?.set(path.config.channels, [rpm, degC]));
      clickIcon(container, "line-plot");
      await waitFor(async () => {
        const plots = await client.lineplots.retrieve({ projects: [proj.key] });
        expect(plots.map(({ key }) => key)).toEqual([first.key]);
        expect(plots[0].channels).toMatchObject({
          y1: [rpm.channel],
          y2: [degC.channel],
        });
        expect(plots[0].axes.y2.label).toBe("degC");
      });
    });

    it("should keep what the user changed on the plot it reopens", async () => {
      const [rpm, degC] = await createFieldChannels(["rpm", "degC"]);
      const extra = await createChannel("float32");
      const { proj, container } = await renderPlot([rpm, degC], undefined, ICD_TYPE);
      const { key } = await findPlot(proj.key);
      await client.lineplots.dispatch(key, [
        lineplot.removeChannel({ axisKey: "y2", channel: degC.channel }),
        lineplot.addChannel({ axisKey: "y3", channel: extra }),
        lineplot.setAxisLabel({ key: "y1", label: "Speed" }),
      ]);
      clickIcon(container, "line-plot");
      await waitFor(async () => {
        const reopened = await client.lineplots.retrieve(key);
        expect(reopened.channels).toMatchObject({
          y1: [rpm.channel],
          y2: [degC.channel],
          y3: [extra],
        });
        expect(reopened.axes.y1.label).toBe("Speed");
      });
    });

    it("should open the tab of the plot it reopens", async () => {
      const key = await createChannel("float32");
      const { proj, container, panelKey } = await renderPlot([
        { channel: key, disabled: false },
      ]);
      const plot = await findPlot(proj.key);
      await client.panels.create({ key: panelKey, name: "Panel", project: proj.key });
      clickIcon(container, "line-plot");
      await waitFor(async () => {
        const { root } = await client.panels.retrieve(panelKey);
        expect(panel.tabs(root)).toHaveLength(1);
        expect(
          panel.findTabByResource(root, lineplot.resourceID(plot.key)),
        ).toBeDefined();
      });
    });

    it("should create a plot when the plot it made was deleted", async () => {
      const key = await createChannel("float32");
      const { proj, container } = await renderPlot([{ channel: key, disabled: false }]);
      const first = await findPlot(proj.key);
      await client.lineplots.delete(first.key);
      clickIcon(container, "line-plot");
      await waitFor(async () => {
        const [second] = await client.lineplots.retrieve({ projects: [proj.key] });
        expect(second.key).not.toBe(first.key);
        expect(second.channels.y1).toEqual([key]);
      });
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
