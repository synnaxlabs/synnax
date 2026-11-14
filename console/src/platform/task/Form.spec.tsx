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
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { deep, TimeStamp } from "@synnaxlabs/x";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { type FC } from "react";
import { beforeAll, describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { createTestSink } from "@/platform/analytics/testutil";
import { Task } from "@/platform/task";
import {
  awaitCommand,
  clickDeploy,
  commitFieldInput,
  renderTaskFormTab,
} from "@/platform/task/testutil";
import { uniqueName } from "@/testutil";

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
      await act(async () => await new Promise((resolve) => setTimeout(resolve, 30)));
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
      await act(async () => await new Promise((resolve) => setTimeout(resolve, 30)));
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
  });
});
