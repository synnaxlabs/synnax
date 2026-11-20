// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { device, rack, type Synnax, task } from "@synnaxlabs/client";
import {
  createTestClient,
  createTestClientWithRole,
} from "@synnaxlabs/client/testutil";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { type Status } from "@synnaxlabs/lyra/status";
import { deep, sleep, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { act, fireEvent, screen, waitFor, within } from "@testing-library/react";
import { type FC } from "react";
import { beforeAll, describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { createTestSink } from "@/platform/analytics/testutil";
import { Task } from "@/platform/task";
import {
  awaitCommand,
  clickDeploy,
  clickRedeploy,
  commitFieldInput,
  createTaskStatus,
  renderTaskFormTab,
  reportTaskRunning,
} from "@/platform/task/testutil";
import { findIconButton } from "@/testutil";

const path = deep.path<Task.Values<typeof schemas>>();

const schemas = {
  type: z.literal("opc_read"),
  config: z.object({
    device: z.string().default(""),
    channels: z.array(z.object({ key: z.string(), disabled: z.boolean() })).default([]),
  }),
  statusData: z.object({ running: z.boolean() }).nullish(),
};

const ChildForm: FC = () => <div>child-form-body</div>;
ChildForm.displayName = "TestChildForm";

const RackProbe: FC = () => <div>{`rack:${PForm.useFieldValue(path.rack)}`}</div>;
RackProbe.displayName = "RackProbe";

const DeviceProbe: FC = () => (
  <div>{`device:${PForm.useFieldValue(path.config.device)}`}</div>
);
DeviceProbe.displayName = "DeviceProbe";

const DeviceField: FC = () => <PForm.Field path={path.config.device} />;
DeviceField.displayName = "DeviceField";

interface MakeRendererParams {
  showControls?: boolean;
  onConfigure?: Task.OnConfigure<(typeof schemas)["config"]>;
  Form?: FC;
  deployConfigZ?: z.ZodType;
}

const INITIAL_VALUES = {
  name: "New Test Task",
  type: "opc_read",
  config: { device: "", channels: [] },
};

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
  // The Core refuses a config that names a device it does not hold.
  beforeAll(async () => {
    const client = createTestClient();
    const rack = await client.racks.create({ name: uniqueName("rack") });
    await client.devices.create(
      ["dev", "valid", "filled", "remote"].map((key) => ({
        key,
        name: key,
        rack: rack.key,
        location: "test",
        make: "test",
        model: "test",
        properties: {},
      })),
    );
  });

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
      expect(container.querySelector(".lyra-icon--play")).toBeTruthy(),
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
    expect(container.querySelector(".lyra-icon--play")).toBeNull();
  });

  it("should keep the controls on the driver status through an edit", async () => {
    const client = createTestClient();
    const tsk = await createTask(client);
    const Renderer = createRenderer();
    await renderTaskFormTab(Renderer, { client, taskKey: tsk.key });
    const input = await waitFor(findNameInput);
    await act(async () => commitFieldInput(input, "Renamed Test Task"));
    await waitFor(async () => {
      const updated = await client.tasks.retrieve({ key: tsk.key });
      expect(updated.name).toBe("Renamed Test Task");
    });
    expect(screen.queryByText(/updating task|updated task/iu)).toBeNull();
    expect(
      screen.getByRole("button", { name: "Start" }).getAttribute("aria-disabled"),
    ).toBeNull();
  });

  it("should save an edit once its nested forms send theirs", async () => {
    const client = createTestClient();
    const tsk = await createTask(client);
    const gate = Promise.withResolvers<void>();
    const flush = vi.fn(async () => {
      await gate.promise;
      return true;
    });
    const NestedForm: FC = () => {
      Task.useFlushFirst(flush);
      return null;
    };
    await renderTaskFormTab(createRenderer({ Form: NestedForm }), {
      client,
      taskKey: tsk.key,
    });
    const input = await waitFor(findNameInput);
    await act(async () => commitFieldInput(input, "Renamed Test Task"));
    await waitFor(() => expect(flush).toHaveBeenCalled());
    // The form's client caches the edit before it sends, so another client reads.
    const stored = async () => (await createTestClient().tasks.retrieve(tsk.key)).name;
    expect(await stored()).toBe(tsk.name);
    gate.resolve();
    await waitFor(async () => expect(await stored()).toBe("Renamed Test Task"));
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
      const { status, onChange } = PForm.useField(path.config.device);
      return (
        <button
          onClick={() => onChange("dev")}
        >{`device-status:${status.message}`}</button>
      );
    };
    DeviceStatusProbe.displayName = "DeviceStatusProbe";

    const requiredDeviceZ = schemas.config.extend({
      device: z.string().min(1, "Device is required"),
    });

    const noPortZ = schemas.config.extend({
      device: z.string().min(1, "The device has no such port"),
    });

    const PortCheckProbe: FC = () => {
      PForm.useCheck(path.config, noPortZ);
      return <DeviceStatusProbe />;
    };
    PortCheckProbe.displayName = "PortCheckProbe";

    it("should block deploy and surface field errors for an invalid config", async () => {
      const client = createTestClient();
      const draft = await client.tasks.create({ ...INITIAL_VALUES, rack: 0 });
      let configured = false;
      const Renderer = createRenderer({
        Form: DeviceStatusProbe,
        deployConfigZ: requiredDeviceZ,
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

    it("should show an error that no field shows in a toast", async () => {
      const client = createTestClient();
      const draft = await client.tasks.create({ ...INITIAL_VALUES, rack: 0 });
      const statuses: Status.NotificationSpec[] = [];
      const { container } = await renderTaskFormTab(
        createRenderer({ deployConfigZ: requiredDeviceZ }),
        {
          client,
          taskKey: draft.key,
          onStatuses: (next) => {
            statuses.length = 0;
            statuses.push(...next);
          },
        },
      );
      await clickDeploy(container);
      await waitFor(() =>
        expect(statuses).toContainEqual(
          expect.objectContaining({
            message: "Failed to start task",
            description: "Device is required",
          }),
        ),
      );
    });

    it("should leave an error to the field that shows it", async () => {
      const client = createTestClient();
      const draft = await client.tasks.create({ ...INITIAL_VALUES, rack: 0 });
      const statuses: Status.NotificationSpec[] = [];
      const { container } = await renderTaskFormTab(
        createRenderer({ Form: DeviceField, deployConfigZ: requiredDeviceZ }),
        {
          client,
          taskKey: draft.key,
          onStatuses: (next) => {
            statuses.length = 0;
            statuses.push(...next);
          },
        },
      );
      await clickDeploy(container);
      expect(await screen.findByText("Device is required")).toBeTruthy();
      await act(async () => {});
      expect(statuses).not.toContainEqual(
        expect.objectContaining({ message: "Failed to start task" }),
      );
    });

    it("should put an error of a keyed list entry on the field of that entry", async () => {
      const ChannelStatus: FC = () => {
        const { status } = PForm.useField(path.config.channels.ch1.key);
        return <div>{`channel-status:${status.message}`}</div>;
      };
      ChannelStatus.displayName = "ChannelStatus";
      // The form mounts with its initial values, before the task loads.
      const ChannelStatusProbe: FC = () => {
        const channels = PForm.useFieldValue(path.config.channels);
        return channels.length > 0 ? <ChannelStatus /> : null;
      };
      ChannelStatusProbe.displayName = "ChannelStatusProbe";
      const client = createTestClient();
      const draft = await client.tasks.create({
        ...INITIAL_VALUES,
        config: { device: "dev", channels: [{ key: "ch1", disabled: false }] },
        rack: 0,
      });
      const Renderer = createRenderer({
        Form: ChannelStatusProbe,
        deployConfigZ: schemas.config.extend({
          channels: z.array(z.object({ key: z.string().min(4, "Key is too short") })),
        }),
      });
      const { container } = await renderTaskFormTab(Renderer, {
        client,
        taskKey: draft.key,
      });
      await clickDeploy(container);
      await waitFor(() =>
        expect(screen.getByText("channel-status:Key is too short")).toBeTruthy(),
      );
    });

    it("should block deploy on an error from a check the form runs", async () => {
      const client = createTestClient();
      const draft = await client.tasks.create({ ...INITIAL_VALUES, rack: 0 });
      let configured = false;
      const Renderer = createRenderer({
        Form: PortCheckProbe,
        onConfigure: async (_client, config) => {
          configured = true;
          return [config, 0];
        },
      });
      const { container } = await renderTaskFormTab(Renderer, {
        client,
        taskKey: draft.key,
      });
      expect(
        screen.getByText("device-status:The device has no such port"),
      ).toBeTruthy();
      await clickDeploy(container);
      await act(async () => {});
      expect(configured).toBe(false);
    });

    it("should not show a deploy error before the first deploy", async () => {
      const client = createTestClient();
      const draft = await client.tasks.create({ ...INITIAL_VALUES, rack: 0 });
      const Renderer = createRenderer({
        Form: DeviceStatusProbe,
        deployConfigZ: requiredDeviceZ,
      });
      await renderTaskFormTab(Renderer, { client, taskKey: draft.key });
      expect(screen.getByText("device-status:")).toBeTruthy();
    });

    it("should clear a deploy error once an edit fixes it", async () => {
      const client = createTestClient();
      const draft = await client.tasks.create({ ...INITIAL_VALUES, rack: 0 });
      const Renderer = createRenderer({
        Form: DeviceStatusProbe,
        deployConfigZ: requiredDeviceZ,
      });
      const { container } = await renderTaskFormTab(Renderer, {
        client,
        taskKey: draft.key,
      });
      await clickDeploy(container);
      const probe = await screen.findByText("device-status:Device is required");
      await act(async () => probe.click());
      expect(screen.getByText("device-status:")).toBeTruthy();
    });

    it("should deploy when the only issues are warnings", async () => {
      const client = createTestClient();
      const draft = await client.tasks.create({ ...INITIAL_VALUES, rack: 0 });
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

    it("should flush a nested form before it configures", async () => {
      const client = createTestClient();
      const draft = await client.tasks.create({ ...INITIAL_VALUES, rack: 0 });
      const order: string[] = [];
      const NestedForm: FC = () => {
        Task.useFlushFirst(async () => {
          order.push("flush");
          return true;
        });
        return null;
      };
      const Renderer = createRenderer({
        Form: NestedForm,
        deployConfigZ: z.unknown(),
        onConfigure: async (_client, config) => {
          order.push("configure");
          return [config, 0];
        },
      });
      const { container } = await renderTaskFormTab(Renderer, {
        client,
        taskKey: draft.key,
      });
      await clickDeploy(container);
      // Writing the configured task flushes the nested form again.
      await waitFor(() => expect(order.slice(0, 2)).toEqual(["flush", "configure"]));
    });

    it("should not deploy while a nested form holds an unsent edit", async () => {
      const client = createTestClient();
      const draft = await client.tasks.create({ ...INITIAL_VALUES, rack: 0 });
      const flush = vi.fn(async () => false);
      const NestedForm: FC = () => {
        Task.useFlushFirst(flush);
        return null;
      };
      const onConfigure = vi.fn<Task.OnConfigure<(typeof schemas)["config"]>>(
        async (_client, config) => [config, 0],
      );
      const Renderer = createRenderer({
        Form: NestedForm,
        deployConfigZ: z.unknown(),
        onConfigure,
      });
      const { container } = await renderTaskFormTab(Renderer, {
        client,
        taskKey: draft.key,
      });
      await clickDeploy(container);
      await waitFor(() => expect(flush).toHaveBeenCalledOnce());
      await act(async () => {});
      expect(onConfigure).not.toHaveBeenCalled();
    });

    it("should not deploy while a field holds a value that fails its schema", async () => {
      const client = createTestClient();
      const draft = await client.tasks.create({ ...INITIAL_VALUES, rack: 0 });
      const DeviceForm: FC = () => {
        const { set } = PForm.useContext();
        return (
          <>
            <button onClick={() => set<unknown>(path.config.device, 5)}>invalid</button>
            <button onClick={() => set(path.config.device, "valid")}>valid</button>
          </>
        );
      };
      DeviceForm.displayName = "DeviceForm";
      const onConfigure = vi.fn<Task.OnConfigure<(typeof schemas)["config"]>>(
        async (_client, config) => [config, 0],
      );
      const Renderer = createRenderer({
        Form: DeviceForm,
        deployConfigZ: z.unknown(),
        onConfigure,
      });
      const { container } = await renderTaskFormTab(Renderer, {
        client,
        taskKey: draft.key,
      });
      await act(async () => fireEvent.click(screen.getByText("invalid")));
      await clickDeploy(container);
      await act(async () => {});
      expect(onConfigure).not.toHaveBeenCalled();
      await act(async () => fireEvent.click(screen.getByText("valid")));
      await clickDeploy(container);
      await waitFor(() => expect(onConfigure).toHaveBeenCalledOnce());
      expect(onConfigure.mock.calls[0][1]).toMatchObject({ device: "valid" });
    });

    it("should keep a deploy error after an edit to another field", async () => {
      const client = createTestClient();
      const draft = await client.tasks.create({ ...INITIAL_VALUES, rack: 0 });
      const Renderer = createRenderer({
        Form: DeviceStatusProbe,
        deployConfigZ: requiredDeviceZ,
      });
      const { container } = await renderTaskFormTab(Renderer, {
        client,
        taskKey: draft.key,
      });
      await clickDeploy(container);
      await waitFor(() =>
        expect(screen.getByText("device-status:Device is required")).toBeTruthy(),
      );
      const input = findNameInput();
      await act(async () => commitFieldInput(input, "Renamed Test Task"));
      await waitFor(async () => {
        const updated = await client.tasks.retrieve({ key: draft.key });
        expect(updated.name).toBe("Renamed Test Task");
      });
      expect(screen.getByText("device-status:Device is required")).toBeTruthy();
    });

    it("should keep a deploy warning after the task deploys", async () => {
      const client = createTestClient();
      const draft = await client.tasks.create({ ...INITIAL_VALUES, rack: 0 });
      const Renderer = createRenderer({
        Form: DeviceStatusProbe,
        deployConfigZ: schemas.config.check(({ value, issues }) => {
          issues.push({
            code: "custom",
            message: "device is suspicious",
            path: ["device"],
            params: { variant: "warning" },
            input: value,
          });
        }),
        onConfigure: async (_client, config) => [{ ...config }, 0],
      });
      const streamer = await client.openStreamer(task.COMMAND_CHANNEL_NAME);
      try {
        const { container } = await renderTaskFormTab(Renderer, {
          client,
          taskKey: draft.key,
        });
        await clickDeploy(container);
        await awaitCommand(streamer, draft.key);
      } finally {
        streamer.close();
      }
      expect(screen.getByText("device-status:device is suspicious")).toBeTruthy();
    });

    it("should clear a fixed deploy error on the next deploy", async () => {
      const client = createTestClient();
      const draft = await client.tasks.create({ ...INITIAL_VALUES, rack: 0 });
      let offline = true;
      let configured = false;
      const Renderer = createRenderer({
        Form: DeviceStatusProbe,
        deployConfigZ: schemas.config.check(({ value, issues }) => {
          if (!offline) return;
          issues.push({
            code: "custom",
            message: "Device is offline",
            path: ["device"],
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
      await waitFor(() =>
        expect(screen.getByText("device-status:Device is offline")).toBeTruthy(),
      );
      offline = false;
      await clickDeploy(container);
      await waitFor(() => expect(configured).toBe(true));
      expect(screen.getByText("device-status:")).toBeTruthy();
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
        ...INITIAL_VALUES,
        rack: rack.key,
        config: { device: deviceKey, channels: [] },
      });
      let renders = 0;
      const CountingForm: FC = () => {
        renders++;
        return <div>{`rack:${PForm.useFieldValue(path.rack)}`}</div>;
      };
      CountingForm.displayName = "CountingForm";
      const Renderer = createRenderer({ Form: CountingForm });
      await renderTaskFormTab(Renderer, { client, taskKey: tsk.key });
      await waitFor(() => expect(screen.getByText(`rack:${rack.key}`)).toBeTruthy());
      const seen = vi.fn();
      const off = client.devices
        .watch({ key: deviceKey, includeStatus: true })
        .onChange(seen);
      // Let the one-time setup renders settle, the permission grant among them, so the
      // count reflects only what the status change causes.
      await act(async () => await sleep.sleep(TimeSpan.milliseconds(30)));
      const before = renders;
      await client.statuses.set({
        owner: device.resourceID(deviceKey),
        name: "",
        variant: "warning",
        message: "device degraded",
        time: TimeStamp.now(),
        details: { rack: rack.key },
      });
      await waitFor(() => expect(seen).toHaveBeenCalled());
      await act(async () => await sleep.sleep(TimeSpan.milliseconds(30)));
      expect(renders).toBe(before);
      off();
    });
  });

  describe("deploying against a live Core", () => {
    it("should persist the configured rack and issue a pinned start", async () => {
      const client = createTestClient();
      const rack = await client.racks.create({ name: uniqueName("rack") });
      const draft = await client.tasks.create({
        ...INITIAL_VALUES,
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
        expect(cmd.pinned).toBe(true);
        const updated = await createTestClient().tasks.retrieve({ key: draft.key });
        expect(cmd.configHash).toBe(updated.configHash);
        expect(updated.name).toBe("New Test Task");
        expect(updated.rack).toBe(rack.key);
      } finally {
        streamer.close();
      }
    });

    it("should write the config fields that onConfigure fills", async () => {
      const client = createTestClient();
      const draft = await client.tasks.create({ ...INITIAL_VALUES, rack: 0 });
      const Renderer = createRenderer({
        onConfigure: async (_client, config) => [{ ...config, device: "filled" }, 0],
      });
      const streamer = await client.openStreamer(task.COMMAND_CHANNEL_NAME);
      try {
        const { container } = await renderTaskFormTab(Renderer, {
          client,
          taskKey: draft.key,
        });
        await clickDeploy(container);
        const cmd = await awaitCommand(streamer, draft.key);
        const updated = await createTestClient().tasks.retrieve({ key: draft.key });
        expect(updated.config).toMatchObject({ device: "filled" });
        expect(cmd.configHash).toBe(updated.configHash);
      } finally {
        streamer.close();
      }
    });

    it("should keep a field another client edits while onConfigure runs", async () => {
      const client = createTestClient();
      const draft = await client.tasks.create({ ...INITIAL_VALUES, rack: 0 });
      const other = createTestClient();
      await other.tasks.retrieve({ key: draft.key });
      const Renderer = createRenderer({
        Form: DeviceProbe,
        onConfigure: async (_client, config) => {
          expect(
            await other.tasks.dispatch(
              draft.key,
              task.set({ path: ["config", "device"], value: "remote" }),
            ),
          ).toBe(true);
          await screen.findByText("device:remote");
          return [config, 0];
        },
      });
      const streamer = await client.openStreamer(task.COMMAND_CHANNEL_NAME);
      try {
        const { container } = await renderTaskFormTab(Renderer, {
          client,
          taskKey: draft.key,
        });
        await clickDeploy(container);
        const cmd = await awaitCommand(streamer, draft.key);
        const updated = await createTestClient().tasks.retrieve({ key: draft.key });
        expect(updated.config).toMatchObject({ device: "remote" });
        expect(cmd.configHash).toBe(updated.configHash);
      } finally {
        streamer.close();
      }
    });

    it("should report the type of the task it started", async () => {
      const client = createTestClient();
      const draft = await client.tasks.create({ ...INITIAL_VALUES, rack: 0 });
      const analytics = createTestSink();
      const { container } = await renderTaskFormTab(createRenderer(), {
        client,
        taskKey: draft.key,
        analytics,
      });
      await clickDeploy(container);
      await waitFor(() =>
        expect(analytics.capture).toHaveBeenCalledWith("task_started", {
          type: "opc_read",
        }),
      );
    });

    it("should start without writing for a subject who cannot update the task", async () => {
      const client = createTestClient();
      const operator = await createTestClientWithRole(client, "Operator");
      const rack = await client.racks.create({ name: uniqueName("rack") });
      const deployed = await client.tasks.create({
        ...INITIAL_VALUES,
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

    it("should start again after a start that got no response", async () => {
      const client = createTestClient();
      const running = await client.racks.create({ name: uniqueName("rack") });
      const draft = await client.tasks.create({ ...INITIAL_VALUES, rack: running.key });
      // The rack reports a running Driver, so the client waits for a reply.
      const beat = async () =>
        await client.statuses.set({
          owner: rack.resourceID(running.key),
          name: "Rack Status",
          variant: "success",
          message: "Driver is running",
          time: TimeStamp.now(),
          details: { running: true },
        });
      await beat();
      const heartbeat = setInterval(() => void beat(), 1000);
      const streamer = await client.openStreamer(task.COMMAND_CHANNEL_NAME);
      vi.useFakeTimers({
        toFake: ["setTimeout", "clearTimeout"],
        shouldAdvanceTime: true,
      });
      try {
        const { container } = await renderTaskFormTab(createRenderer(), {
          client,
          taskKey: draft.key,
        });
        await clickDeploy(container);
        await awaitCommand(streamer, draft.key);
        await screen.findByText("Sending start command");
        await act(
          async () =>
            await vi.advanceTimersByTimeAsync(TimeSpan.seconds(10).milliseconds),
        );
        await screen.findByText("No response to the start command");
        await clickDeploy(container);
        expect((await awaitCommand(streamer, draft.key)).type).toBe("start");
      } finally {
        vi.useRealTimers();
        clearInterval(heartbeat);
        streamer.close();
      }
    });

    it("should start a task whose status fails its status data schema", async () => {
      const client = createTestClient();
      const draft = await client.tasks.create({ ...INITIAL_VALUES, rack: 0 });
      const streamer = await client.openStreamer(task.COMMAND_CHANNEL_NAME);
      try {
        const { container } = await renderTaskFormTab(createRenderer(), {
          client,
          taskKey: draft.key,
        });
        // The status data of the task type requires running.
        await client.statuses.set(
          createTaskStatus({
            owner: task.resourceID(draft.key),
            message: "Task stopped",
            details: { data: {} },
          }),
        );
        await screen.findByText("Task stopped");
        await clickDeploy(container);
        expect((await awaitCommand(streamer, draft.key)).type).toBe("start");
      } finally {
        streamer.close();
      }
    });

    it("should not start a redeploy that a stop cancels while it configures", async () => {
      const client = createTestClient();
      const draft = await client.tasks.create({ ...INITIAL_VALUES, rack: 0 });
      const { promise: released, resolve: release } = Promise.withResolvers<void>();
      const Renderer = createRenderer({
        onConfigure: async (_client, config) => {
          await released;
          return [config, 0];
        },
      });
      const streamer = await client.openStreamer(task.COMMAND_CHANNEL_NAME);
      try {
        const { container } = await renderTaskFormTab(Renderer, {
          client,
          taskKey: draft.key,
        });
        await reportTaskRunning(client, draft, "stale");
        const redeploy = await screen.findByRole("button", { name: "Redeploy" });
        await act(async () => fireEvent.click(redeploy));
        await screen.findByText("Configuring task");
        fireEvent.click(await findIconButton(container, "stop"));
        expect((await awaitCommand(streamer, draft.key)).type).toBe("stop");
        release();
        await waitFor(() => expect(screen.queryByText("Configuring task")).toBeNull());
        // A start sent after the stop would come before this second stop.
        fireEvent.click(await findIconButton(container, "stop"));
        expect((await awaitCommand(streamer, draft.key)).type).toBe("stop");
      } finally {
        streamer.close();
      }
    });

    it("should show that a start configures until its command goes out", async () => {
      const client = createTestClient();
      const draft = await client.tasks.create({ ...INITIAL_VALUES, rack: 0 });
      const { promise: released, resolve: release } = Promise.withResolvers<void>();
      const Renderer = createRenderer({
        onConfigure: async (_client, config) => {
          await released;
          return [config, 0];
        },
      });
      const streamer = await client.openStreamer(task.COMMAND_CHANNEL_NAME);
      try {
        const { container } = await renderTaskFormTab(Renderer, {
          client,
          taskKey: draft.key,
        });
        await clickDeploy(container);
        await screen.findByText("Configuring task");
        expect(
          within(container)
            .getByRole("button", { name: "Start" })
            .getAttribute("aria-disabled"),
        ).toBe("true");
        release();
        expect((await awaitCommand(streamer, draft.key)).type).toBe("start");
        await waitFor(() => expect(screen.queryByText("Configuring task")).toBeNull());
      } finally {
        streamer.close();
      }
    });
  });

  describe("deploy failure", () => {
    const failing = createRenderer({
      onConfigure: async () => {
        throw new Error("The device has no port 7");
      },
    });

    it("should show a failed start in the footer until the task status changes", async () => {
      const client = createTestClient();
      const draft = await client.tasks.create({ ...INITIAL_VALUES, rack: 0 });
      const { container } = await renderTaskFormTab(failing, {
        client,
        taskKey: draft.key,
      });
      await clickDeploy(container);
      await screen.findByText("Failed to start task");
      expect(screen.getByText("The device has no port 7")).toBeTruthy();
      await reportTaskRunning(client, draft);
      await screen.findByText("Running smoothly");
      expect(screen.queryByText("Failed to start task")).toBeNull();
    });

    it("should say that a failed redeploy leaves the last deployed config running", async () => {
      const client = createTestClient();
      const draft = await client.tasks.create({ ...INITIAL_VALUES, rack: 0 });
      const statuses: Status.NotificationSpec[] = [];
      await renderTaskFormTab(failing, {
        client,
        taskKey: draft.key,
        onStatuses: (next) => {
          statuses.length = 0;
          statuses.push(...next);
        },
      });
      await reportTaskRunning(client, draft, "stale");
      await clickRedeploy();
      await screen.findByText("Failed to deploy changes");
      expect(
        screen.getByText(
          "The task still runs its last deployed config. The device has no port 7",
        ),
      ).toBeTruthy();
      expect(statuses).toContainEqual(
        expect.objectContaining({
          name: "New Test Task",
          message: "Failed to deploy changes",
          description: "The device has no port 7",
        }),
      );
    });

    it("should clear a failed deploy from the footer on the next deploy", async () => {
      const client = createTestClient();
      const draft = await client.tasks.create({ ...INITIAL_VALUES, rack: 0 });
      let offline = false;
      const Renderer = createRenderer({
        Form: DeviceField,
        deployConfigZ: schemas.config.check(({ value, issues }) => {
          if (!offline) return;
          issues.push({
            code: "custom",
            message: "Device is offline",
            path: ["device"],
            input: value,
          });
        }),
        onConfigure: async () => {
          throw new Error("The device has no port 7");
        },
      });
      const { container } = await renderTaskFormTab(Renderer, {
        client,
        taskKey: draft.key,
      });
      await clickDeploy(container);
      await screen.findByText("Failed to start task");
      offline = true;
      await clickDeploy(container);
      await screen.findByText("Device is offline");
      await act(async () => {});
      expect(screen.queryByText("Failed to start task")).toBeNull();
    });
  });
});
