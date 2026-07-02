// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Drift } from "@synnaxlabs/drift";
import { screen, waitFor } from "@testing-library/react";
import { act, type FC } from "react";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { Task } from "@/platform/task";
import { renderTaskFormHook } from "@/platform/task/testutil";
import { Session } from "@/session";
import { createTestStore, renderWithConsole, type TestStore } from "@/testutil";

const schemas = {
  type: z.literal("test_task"),
  config: z.object({
    device: z.string().default(""),
    channels: z.array(z.object({ key: z.string(), enabled: z.boolean() })).default([]),
  }),
  statusData: z.object({ running: z.boolean() }),
};

const ChildForm: FC<Task.FormProps<typeof schemas>> = () => <div>child-form-body</div>;
ChildForm.displayName = "TestChildForm";

const makeRenderer = (showControls = true) =>
  Task.wrapForm<typeof schemas>({
    Form: ChildForm,
    schemas,
    type: "test_task",
    getInitialValues: () => ({
      name: "New Test Task",
      type: "test_task",
      config: { device: "", channels: [] },
    }),
    onConfigure: async (_client, config) => [config, 0],
    showControls,
  });

const seedLayout = (store: TestStore, key: string): void =>
  act(() => {
    store.dispatch(
      Session.Layout.place({
        key,
        type: "test_task",
        name: key,
        location: "mosaic",
        windowKey: Drift.MAIN_WINDOW,
        window: { title: key },
        args: {},
      }),
    );
  });

describe("wrapForm", () => {
  it("should produce a renderer whose displayName references the child form", () => {
    const Renderer = makeRenderer();
    expect(Renderer.displayName).toContain("TestChildForm");
  });

  it("should render the header name field, the child form, and the controls", async () => {
    const Renderer = makeRenderer();
    const store = await createTestStore();
    seedLayout(store, "l1");
    await renderWithConsole(
      <Renderer layoutKey="l1" visible focused={false} onClose={() => {}} />,
      { store },
    );
    await waitFor(() => expect(screen.getByText("child-form-body")).toBeTruthy());
    const nameInput = document.body.querySelector<HTMLInputElement>(
      "input[value='New Test Task']",
    );
    expect(nameInput).toBeTruthy();
    expect(screen.getByRole("button", { name: /Configure/ })).toBeTruthy();
  });

  it("should omit the controls when showControls is false", async () => {
    const Renderer = makeRenderer(false);
    const store = await createTestStore();
    seedLayout(store, "l2");
    await renderWithConsole(
      <Renderer layoutKey="l2" visible focused={false} onClose={() => {}} />,
      { store },
    );
    await waitFor(() => expect(screen.getByText("child-form-body")).toBeTruthy());
    expect(screen.queryByRole("button", { name: /Configure/ })).toBeNull();
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
