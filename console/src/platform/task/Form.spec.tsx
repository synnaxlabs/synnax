// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { task } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Drift } from "@synnaxlabs/drift";
import { Form as PForm } from "@synnaxlabs/pluto";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { act, type FC } from "react";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { Task } from "@/platform/task";
import { commitFieldInput, renderTaskFormHook } from "@/platform/task/testutil";
import { Session } from "@/session";
import {
  createConsoleWrapper,
  createTestStore,
  renderWithConsole,
  type TestStore,
  uniqueName,
} from "@/testutil";

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

interface MakeRendererArgs {
  showControls?: boolean;
  onConfigure?: Task.OnConfigure<(typeof schemas)["config"]>;
  Form?: FC<Task.FormProps<typeof schemas>>;
}

const createRenderer = ({
  showControls = true,
  onConfigure = async (_client, config) => [config, 0],
  Form = ChildForm,
}: MakeRendererArgs = {}) =>
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

const createLayout = (
  store: TestStore,
  key: string,
  args: Task.FormLayoutArgs = {},
): void =>
  act(() => {
    store.dispatch(
      Session.Layout.place({
        key,
        type: "test_task",
        name: key,
        location: "mosaic",
        windowKey: Drift.MAIN_WINDOW,
        window: { title: key },
        args,
      }),
    );
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
    expect(Renderer.displayName).toContain("TestChildForm");
  });

  it("should render the header name field, the child form, and the controls", async () => {
    const Renderer = createRenderer();
    const store = await createTestStore();
    createLayout(store, "l1");
    await renderWithConsole(
      <Renderer layoutKey="l1" visible focused={false} onClose={() => {}} />,
      { store },
    );
    await waitFor(() => expect(screen.getByText("child-form-body")).toBeTruthy());
    expect(findNameInput()).toBeTruthy();
    expect(screen.getByRole("button", { name: /Configure/ })).toBeTruthy();
  });

  it("should omit the controls when showControls is false", async () => {
    const Renderer = createRenderer({ showControls: false });
    const store = await createTestStore();
    createLayout(store, "l2");
    await renderWithConsole(
      <Renderer layoutKey="l2" visible focused={false} onClose={() => {}} />,
      { store },
    );
    await waitFor(() => expect(screen.getByText("child-form-body")).toBeTruthy());
    expect(screen.queryByRole("button", { name: /Configure/ })).toBeNull();
  });

  it("should mark the layout as having unsaved changes when the form is edited", async () => {
    const Renderer = createRenderer();
    const store = await createTestStore();
    createLayout(store, "l3");
    await renderWithConsole(
      <Renderer layoutKey="l3" visible focused={false} onClose={() => {}} />,
      { store },
    );
    await waitFor(() => expect(findNameInput()).toBeTruthy());
    expect(Session.Layout.select(store.getState(), "l3")?.unsavedChanges).not.toBe(
      true,
    );
    commitFieldInput(findNameInput(), "Renamed Task");
    await waitFor(() =>
      expect(Session.Layout.select(store.getState(), "l3")?.unsavedChanges).toBe(true),
    );
  });

  describe("initial rackKey", () => {
    const renderProbe = async (layoutKey: string, args: Task.FormLayoutArgs = {}) => {
      const Renderer = createRenderer({ Form: RackKeyProbe });
      const store = await createTestStore();
      createLayout(store, layoutKey, args);
      await renderWithConsole(
        <Renderer layoutKey={layoutKey} visible focused={false} onClose={() => {}} />,
        { store },
      );
    };

    it("should prefill from the rackKey layout arg", async () => {
      await renderProbe("rk1", { rackKey: 5 });
      await waitFor(() => expect(screen.getByText("rack-key:5")).toBeTruthy());
    });

    it("should derive it from the task key when no rackKey arg is given", async () => {
      const taskKey = ((7n << 32n) | 1n).toString();
      await renderProbe("rk2", { taskKey });
      await waitFor(() => expect(screen.getByText("rack-key:7")).toBeTruthy());
    });

    it("should default to zero when neither rackKey nor taskKey is given", async () => {
      await renderProbe("rk3");
      await waitFor(() => expect(screen.getByText("rack-key:0")).toBeTruthy());
    });
  });

  describe("saving against a live cluster", () => {
    it("should create the task on the configured rack and update the layout", async () => {
      const client = createTestClient();
      const rack = await client.racks.create({ name: uniqueName("rack") });
      const Renderer = createRenderer({
        onConfigure: async (_client, config) => [config, rack.key],
      });
      const layoutKey = uniqueName("layout");
      const store = await createTestStore();
      createLayout(store, layoutKey);
      const { wrapper } = await createConsoleWrapper({ client, store });
      render(
        <Renderer layoutKey={layoutKey} visible focused={false} onClose={() => {}} />,
        { wrapper },
      );
      fireEvent.click(await screen.findByRole("button", { name: /Configure/ }));
      const taskKey = await waitFor(() => {
        const args = Session.Layout.selectArgs<Task.FormLayoutArgs>(
          store.getState(),
          layoutKey,
        );
        if (args.taskKey == null) throw new Error("task key not set on layout args");
        return args.taskKey;
      });
      const created = await client.tasks.retrieve({ key: taskKey });
      expect(created.name).toBe("New Test Task");
      expect(created.type).toBe("test_task");
      expect(task.rackKey(created.key)).toBe(rack.key);
      const layout = Session.Layout.selectRequired(store.getState(), layoutKey);
      expect(layout.name).toBe("New Test Task");
      expect(Session.Layout.selectAltKey(store.getState(), layoutKey)).toBe(
        created.key,
      );
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
