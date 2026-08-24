// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { device, type Synnax, task } from "@synnaxlabs/client";
import {
  createTestClient,
  createTestClientWithRole,
} from "@synnaxlabs/client/testutil";
import { Form as PForm } from "@synnaxlabs/pluto";
import { TimeStamp } from "@synnaxlabs/x";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { type FC } from "react";
import { describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { Task } from "@/platform/task";
import {
  awaitCommand,
  clickDeploy,
  renderTaskFormHook,
  renderTaskFormTab,
} from "@/platform/task/testutil";
import { getIconButton, uniqueName } from "@/testutil";

const schemas = {
  type: z.literal("opc_read"),
  config: z.object({
    device: z.string().default(""),
    channels: z.array(z.object({ key: z.string(), enabled: z.boolean() })).default([]),
  }),
  statusData: z.object({ running: z.boolean() }).nullish(),
};

const ChildForm: FC = () => <div>child-form-body</div>;
ChildForm.displayName = "TestChildForm";

const RackProbe: FC = () => <div>{`rack:${PForm.useFieldValue<number>("rack")}`}</div>;
RackProbe.displayName = "RackProbe";

interface MakeRendererParams {
  showControls?: boolean;
  onConfigure?: Task.OnConfigure<(typeof schemas)["config"]>;
  Form?: FC;
  deployConfigZ?: z.ZodType;
}

const getInitialValues: Task.GetInitialValues<typeof schemas> = () => ({
  name: "New Test Task",
  type: "opc_read",
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
    type: "opc_read",
    getInitialValues,
    onConfigure,
    showControls,
  });

const createTask = async (client: Synnax) =>
  await client.tasks.create({
    name: "New Test Task",
    type: "opc_read",
    config: { device: "", channels: [] },
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
    const client = createTestClient();
    const tsk = await createTask(client);
    const Renderer = createRenderer();
    const { container } = await renderTaskFormTab(Renderer, {
      client,
      taskKey: tsk.key,
    });
    await waitFor(() => expect(screen.getByText("child-form-body")).toBeTruthy());
    // The name input and the play button appear once the async update and command
    // grants resolve; until then the form renders in preview mode.
    await waitFor(findNameInput);
    await waitFor(() =>
      expect(container.querySelector(".pluto-icon--play")).toBeTruthy(),
    );
  });

  it("should omit the controls when showControls is false", async () => {
    const client = createTestClient();
    const tsk = await createTask(client);
    const Renderer = createRenderer({ showControls: false });
    const { container } = await renderTaskFormTab(Renderer, {
      client,
      taskKey: tsk.key,
    });
    await waitFor(() => expect(screen.getByText("child-form-body")).toBeTruthy());
    expect(container.querySelector(".pluto-icon--play")).toBeNull();
  });

  it("should keep the controls on the driver status through an autosave", async () => {
    const client = createTestClient();
    const tsk = await createTask(client);
    const Renderer = createRenderer();
    const { container } = await renderTaskFormTab(Renderer, {
      client,
      taskKey: tsk.key,
    });
    const input = await waitFor(findNameInput);
    fireEvent.change(input, { target: { value: "Renamed Test Task" } });
    fireEvent.blur(input);
    await waitFor(async () => {
      const updated = await client.tasks.retrieve({ key: tsk.key });
      expect(updated.name).toBe("Renamed Test Task");
    });
    expect(screen.queryByText(/updating task|updated task/iu)).toBeNull();
    expect(getIconButton(container, "play").disabled).toBe(false);
  });

  describe("rack", () => {
    it("should load it from the retrieved task", async () => {
      const client = createTestClient();
      const rack = await client.racks.create({ name: uniqueName("rack") });
      const tsk = await rack.createTask({
        name: uniqueName("tsk"),
        type: "opc_read",
        config: { device: "", channels: [] },
      });
      const Renderer = createRenderer({ Form: RackProbe });
      await renderTaskFormTab(Renderer, { client, taskKey: tsk.key });
      await waitFor(() => expect(screen.getByText(`rack:${rack.key}`)).toBeTruthy());
    });

    it("should default to zero for a draft task", async () => {
      const client = createTestClient();
      const tsk = await createTask(client);
      const Renderer = createRenderer({ Form: RackProbe });
      await renderTaskFormTab(Renderer, { client, taskKey: tsk.key });
      await waitFor(() => expect(screen.getByText("rack:0")).toBeTruthy());
    });
  });

  describe("deploy validation gate", () => {
    const DeviceStatusProbe: FC = () => {
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
          device: z.string().min(1, "Device is required"),
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
        expect(screen.getByText("device-status:Device is required")).toBeTruthy(),
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

  describe("device rack sync", () => {
    it("should not re-render the form when the device's status changes", async () => {
      const client = createTestClient();
      const rack = await client.racks.create({ name: uniqueName("rack") });
      const deviceKey = uniqueName("dev");
      await client.devices.create({
        key: deviceKey,
        rack: rack.key,
        location: "dev",
        name: uniqueName("device"),
        make: "ni",
        model: "test",
        properties: {},
      });
      const tsk = await client.tasks.create({
        ...getInitialValues({}),
        rack: rack.key,
        config: { device: deviceKey, channels: [] },
      });
      let renders = 0;
      const CountingForm: FC = () => {
        renders++;
        return <div>{`rack:${PForm.useFieldValue<number>("rack")}`}</div>;
      };
      CountingForm.displayName = "CountingForm";
      const Renderer = createRenderer({ Form: CountingForm });
      await renderTaskFormTab(Renderer, { client, taskKey: tsk.key });
      await waitFor(() => expect(screen.getByText(`rack:${rack.key}`)).toBeTruthy());
      const seen = vi.fn();
      const off = client.devices.onChange(
        { key: deviceKey, includeStatus: true },
        seen,
      );
      // Let the one-time setup renders settle, the permission grant among them, so the
      // count reflects only what the status change causes.
      await act(async () => await new Promise((resolve) => setTimeout(resolve, 30)));
      const before = renders;
      await client.statuses.set({
        key: device.statusKey(deviceKey),
        name: "",
        variant: "warning",
        message: "device degraded",
        time: TimeStamp.now(),
        details: { rack: rack.key, device: deviceKey },
      });
      await waitFor(() => expect(seen).toHaveBeenCalled());
      await act(async () => await new Promise((resolve) => setTimeout(resolve, 30)));
      expect(renders).toBe(before);
      off();
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

    it("should start without saving for a subject who cannot update the task", async () => {
      const client = createTestClient();
      const operator = await createTestClientWithRole(client, "Operator");
      const rack = await client.racks.create({ name: uniqueName("rack") });
      const deployed = await client.tasks.create({
        ...getInitialValues({}),
        rack: rack.key,
      });
      const onConfigure = vi.fn<Task.OnConfigure<(typeof schemas)["config"]>>(
        async (_client, config) => [config, 0],
      );
      const Renderer = createRenderer({ onConfigure });
      const streamer = await client.openStreamer(task.COMMAND_CHANNEL_NAME);
      try {
        const { container } = await renderTaskFormTab(Renderer, {
          client,
          as: operator,
          taskKey: deployed.key,
        });
        await clickDeploy(container);
        const cmd = await awaitCommand(streamer, deployed.key);
        expect(cmd.type).toBe("start");
      } finally {
        streamer.close();
      }
      expect(onConfigure).not.toHaveBeenCalled();
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

  it("should throw when the form carries no status", async () => {
    await expect(
      renderTaskFormHook({}, (ctx) => Task.useIsRunning(ctx)),
    ).rejects.toThrow("Path status does not exist");
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
