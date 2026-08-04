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
import { Form as PForm } from "@synnaxlabs/pluto";
import { screen, waitFor } from "@testing-library/react";
import { type FC } from "react";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { Task } from "@/platform/task";
import {
  awaitCommand,
  clickDeploy,
  renderTaskFormHook,
  renderTaskFormTab,
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

const RackProbe: FC<Task.FormProps<typeof schemas>> = () => (
  <div>{`rack:${PForm.useFieldValue<number>("rack")}`}</div>
);
RackProbe.displayName = "RackProbe";

interface MakeRendererParams {
  showControls?: boolean;
  onConfigure?: Task.OnConfigure<(typeof schemas)["config"]>;
  Form?: FC<Task.FormProps<typeof schemas>>;
  deployConfigZ?: z.ZodType;
}

const getInitialValues: Task.GetInitialValues<typeof schemas> = () => ({
  name: "New Test Task",
  type: "test_task",
  config: { device: "", channels: [] },
});

const createRenderer = ({
  showControls = true,
  onConfigure = async (_client, config) => [config, 0],
  Form = ChildForm,
  deployConfigZ = schemas.config,
}: MakeRendererParams = {}) =>
  Task.wrapForm<typeof schemas>({
    Form,
    schemas,
    deployConfigZ,
    type: "test_task",
    getInitialValues,
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
    expect(Renderer.displayName).toContain("TestChildForm");
  });

  it("should render the header name field, the child form, and the controls", async () => {
    const Renderer = createRenderer();
    const { container } = await renderTaskFormTab(Renderer);
    await waitFor(() => expect(screen.getByText("child-form-body")).toBeTruthy());
    expect(findNameInput()).toBeTruthy();
    expect(container.querySelector("[aria-label='pluto-icon--play']")).toBeTruthy();
  });

  it("should omit the controls when showControls is false", async () => {
    const Renderer = createRenderer({ showControls: false });
    const { container } = await renderTaskFormTab(Renderer);
    await waitFor(() => expect(screen.getByText("child-form-body")).toBeTruthy());
    expect(container.querySelector("[aria-label='pluto-icon--play']")).toBeNull();
  });

  describe("rack", () => {
    it("should load it from the retrieved task", async () => {
      const client = createTestClient();
      const rack = await client.racks.create({ name: uniqueName("rack") });
      const tsk = await rack.createTask({
        name: uniqueName("tsk"),
        type: "test_task",
        config: { device: "", channels: [] },
      });
      const Renderer = createRenderer({ Form: RackProbe });
      await renderTaskFormTab(Renderer, { client, taskKey: tsk.key });
      await waitFor(() => expect(screen.getByText(`rack:${rack.key}`)).toBeTruthy());
    });

    it("should default to zero for a draft task", async () => {
      const Renderer = createRenderer({ Form: RackProbe });
      await renderTaskFormTab(Renderer);
      await waitFor(() => expect(screen.getByText("rack:0")).toBeTruthy());
    });
  });

  describe("deploy validation gate", () => {
    const DeviceStatusProbe: FC<Task.FormProps<typeof schemas>> = () => {
      const { status } = PForm.useField<string>("config.device");
      return <div>{`device-status:${status.message}`}</div>;
    };
    DeviceStatusProbe.displayName = "DeviceStatusProbe";

    it("should block deploy and surface field errors for an invalid config", async () => {
      const client = createTestClient();
      const draft = await client.tasks.create({ ...getInitialValues({}), rack: 0 });
      let configured = false;
      const Renderer = createRenderer({
        Form: DeviceStatusProbe,
        deployConfigZ: schemas.config.extend({
          device: z.string().min(1, "Must specify a device"),
        }),
        onConfigure: async (_client, config) => {
          configured = true;
          return [config, 0];
        },
      });
      const { container } = await renderTaskFormTab(Renderer, {
        client,
        taskKey: draft.key,
      });
      await clickDeploy(container);
      await waitFor(() =>
        expect(screen.getByText("device-status:Must specify a device")).toBeTruthy(),
      );
      expect(configured).toBe(false);
    });

    it("should deploy when the only issues are warnings", async () => {
      const client = createTestClient();
      const draft = await client.tasks.create({ ...getInitialValues({}), rack: 0 });
      let configured = false;
      const Renderer = createRenderer({
        deployConfigZ: schemas.config.check(({ value, issues }) => {
          issues.push({
            code: "custom",
            message: "device is suspicious",
            path: ["device"],
            params: { variant: "warning" },
            input: value,
          });
        }),
        onConfigure: async (_client, config) => {
          configured = true;
          return [config, 0];
        },
      });
      const { container } = await renderTaskFormTab(Renderer, {
        client,
        taskKey: draft.key,
      });
      await clickDeploy(container);
      await waitFor(() => expect(configured).toBe(true));
    });
  });

  describe("deploying against a live cluster", () => {
    it("should persist the configured rack and issue a start command", async () => {
      const client = createTestClient();
      const rack = await client.racks.create({ name: uniqueName("rack") });
      const draft = await client.tasks.create({
        ...getInitialValues({}),
        rack: 0,
      });
      const Renderer = createRenderer({
        onConfigure: async (_client, config) => [config, rack.key],
      });
      const streamer = await client.openStreamer(task.COMMAND_CHANNEL_NAME);
      try {
        const { container } = await renderTaskFormTab(Renderer, {
          client,
          taskKey: draft.key,
        });
        await clickDeploy(container);
        const cmd = await awaitCommand(streamer, draft.key);
        expect(cmd.type).toBe("start");
      } finally {
        streamer.close();
      }
      const updated = await client.tasks.retrieve({ key: draft.key });
      expect(updated.name).toBe("New Test Task");
      expect(updated.rack).toBe(rack.key);
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
