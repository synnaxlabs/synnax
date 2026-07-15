// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { Form as PForm } from "@synnaxlabs/pluto";
import { screen, waitFor } from "@testing-library/react";
import { type FC } from "react";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { Task } from "@/platform/task";
import {
  awaitTaskKey,
  clickConfigure,
  renderTaskFormHook,
  renderTaskFormTab,
  selectViewArgs,
} from "@/platform/task/testutil";
import { uniqueName } from "@/testutil";

const schemas = {
  type: z.literal("test_task"),
  config: z.object({
    device: z.string().default(""),
    channels: z.array(z.object({ key: z.string(), enabled: z.boolean() })).default([]),
  }),
  statusData: z.object({ running: z.boolean() }).nullish(),
};

const ChildForm: FC<Task.FormProps<typeof schemas>> = () => <div>child-form-body</div>;
ChildForm.displayName = "TestChildForm";

const RackKeyProbe: FC<Task.FormProps<typeof schemas>> = () => (
  <div>{`rack-key:${PForm.useFieldValue<number>("rackKey")}`}</div>
);
RackKeyProbe.displayName = "RackKeyProbe";

interface MakeRendererParams {
  showControls?: boolean;
  onConfigure?: Task.OnConfigure<(typeof schemas)["config"]>;
  Form?: FC<Task.FormProps<typeof schemas>>;
}

const createRenderer = ({
  showControls = true,
  onConfigure = async (_client, config) => [config, 0],
  Form = ChildForm,
}: MakeRendererParams = {}) =>
  Task.wrapForm<typeof schemas>({
    Form,
    schemas,
    type: "test_task",
    getInitialValues: () => ({
      name: "New Test Task",
      type: "test_task",
      config: { device: "", channels: [] },
    }),
    onConfigure,
    showControls,
  });

const findNameInput = (): HTMLInputElement => {
  const input = document.body.querySelector<HTMLInputElement>(
    "input[value='New Test Task']",
  );
  if (input == null) throw new Error("name input not found");
  return input;
};

describe("wrapForm", () => {
  it("should produce a renderer whose displayName references the child form", () => {
    const Renderer = createRenderer();
    expect(Renderer.Content.displayName).toContain("TestChildForm");
  });

  it("should render the header name field, the child form, and the controls", async () => {
    const Renderer = createRenderer();
    await renderTaskFormTab(Renderer, "test_task");
    await waitFor(() => expect(screen.getByText("child-form-body")).toBeTruthy());
    expect(findNameInput()).toBeTruthy();
    expect(screen.getByRole("button", { name: /Configure/ })).toBeTruthy();
  });

  it("should omit the controls when showControls is false", async () => {
    const Renderer = createRenderer({ showControls: false });
    await renderTaskFormTab(Renderer, "test_task");
    await waitFor(() => expect(screen.getByText("child-form-body")).toBeTruthy());
    expect(screen.queryByRole("button", { name: /Configure/ })).toBeNull();
  });

  describe("initial rackKey", () => {
    const renderProbe = async (params: Task.FormViewParams = {}) => {
      const Renderer = createRenderer({ Form: RackKeyProbe });
      await renderTaskFormTab(Renderer, "test_task", { params });
    };

    it("should prefill from the rackKey view arg", async () => {
      await renderProbe({ rackKey: 5 });
      await waitFor(() => expect(screen.getByText("rack-key:5")).toBeTruthy());
    });

    it("should load it from the retrieved task when no rackKey arg is given", async () => {
      const client = createTestClient();
      const rack = await client.racks.create({ name: uniqueName("rack") });
      const tsk = await rack.createTask({
        name: uniqueName("tsk"),
        type: "test_task",
        config: { device: "", channels: [] },
      });
      const Renderer = createRenderer({ Form: RackKeyProbe });
      await renderTaskFormTab(Renderer, "test_task", {
        client,
        args: { taskKey: tsk.key },
      });
      await waitFor(() =>
        expect(screen.getByText(`rack-key:${rack.key}`)).toBeTruthy(),
      );
    });

    it("should default to zero when neither rackKey nor taskKey is given", async () => {
      await renderProbe();
      await waitFor(() => expect(screen.getByText("rack-key:0")).toBeTruthy());
    });
  });

  describe("saving against a live cluster", () => {
    it("should create the task on the configured rack and write it back to the tab", async () => {
      const client = createTestClient();
      const rack = await client.racks.create({ name: uniqueName("rack") });
      const Renderer = createRenderer({
        onConfigure: async (_client, config) => [config, rack.key],
      });
      const result = await renderTaskFormTab(Renderer, "test_task", { client });
      await clickConfigure();
      const taskKey = await awaitTaskKey(result);
      const created = await client.tasks.retrieve({ key: taskKey });
      expect(created.name).toBe("New Test Task");
      expect(created.type).toBe("test_task");
      expect(created.rack).toBe(rack.key);
      expect(selectViewArgs(result)).toEqual({ taskKey });
    });
  });
});

describe("useIsRunning", () => {
  it("should be true when the status reports the task as running", async () => {
    const { result } = await renderTaskFormHook(
      { status: { details: { running: true } } },
      (ctx) => Task.useIsRunning(ctx),
    );
    expect(result.current.value).toBe(true);
  });

  it("should be false when the status reports the task as not running", async () => {
    const { result } = await renderTaskFormHook(
      { status: { details: { running: false } } },
      (ctx) => Task.useIsRunning(ctx),
    );
    expect(result.current.value).toBe(false);
  });

  it("should default to false when there is no status", async () => {
    const { result } = await renderTaskFormHook({}, (ctx) => Task.useIsRunning(ctx));
    expect(result.current.value).toBe(false);
  });
});

describe("useIsSnapshot", () => {
  it("should return the value of the snapshot field", async () => {
    const { result } = await renderTaskFormHook({ snapshot: true }, (ctx) =>
      Task.useIsSnapshot(ctx),
    );
    expect(result.current.value).toBe(true);
  });

  it("should return false for a non-snapshot task", async () => {
    const { result } = await renderTaskFormHook({ snapshot: false }, (ctx) =>
      Task.useIsSnapshot(ctx),
    );
    expect(result.current.value).toBe(false);
  });
});
