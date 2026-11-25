// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { icd, mqtt } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import type { Tree } from "@synnaxlabs/lyra/tree";
import { deep, uuid } from "@synnaxlabs/x";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { type FC, useState } from "react";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { ICD } from "@/platform/icd";
import { Task } from "@/platform/task";
import { renderTaskFormTab } from "@/platform/task/testutil";

// The Core keeps the config of a known task type only, so the targets are MQTT topics.
const schemas = {
  type: z.literal("mqtt_write"),
  config: mqtt.writeConfigZ,
  statusData: z.object({ running: z.boolean() }).nullish(),
};

const path = deep.path<Task.Values<typeof schemas>>();

const messageOf = (target: mqtt.WriteTarget): ICD.Write.TargetMessage | null =>
  target.type === "plain"
    ? {
        icd: target.icd,
        message: target.message.message,
        path: deep.variant(path.config.targets[target.key], "type", "plain").message,
        channelName: ({ name }) => `${name}_cmd`,
      }
    : null;

// The targets group by topic, as Kafka records do.
const groups: ICD.Write.Groups<mqtt.WriteTarget> = {
  of: (target) => (target.type === "plain" ? target.topic : ""),
  Item: ({ name, ...props }) => (
    <Task.Views.TreeGroupItem {...props} label={`Topic ${name}`} />
  ),
};

const TargetItem = (props: Tree.ItemRenderProps<string>) => (
  <Task.Views.TreeGroupItem {...props} label={`Target ${props.itemKey}`} />
);

const Targets: FC = () => {
  const targets = PForm.useFieldValue(path.config.targets);
  const [selected, setSelected] = useState<string[]>([]);
  const [expanded, setExpanded] = useState(() => ICD.Write.expandedOf(targets, groups));
  const { tree, handleSelect, contextMenu, renderItem } =
    ICD.Write.useTree<mqtt.WriteTarget>({
      path: path.config.targets,
      messageOf,
      groups,
      TargetItem,
      targetPane: (target) => ({
        path: path.config.targets[target.key],
        title: null,
        details: null,
      }),
      selected,
      onSelect: setSelected,
      expanded,
      onExpand: setExpanded,
    });
  return (
    <Task.Views.TreeList
      {...tree}
      onSelect={handleSelect}
      menu={contextMenu}
      emptyMessage="No targets"
      addLabel="Add target"
      onAdd={() => {}}
    >
      {renderItem}
    </Task.Views.TreeList>
  );
};

const Form: FC = () => {
  // The task form renders before it loads the config of the task.
  const targets = PForm.useFieldValue(path.config.targets, { optional: true });
  const [target] = targets ?? [];
  if (target?.type !== "plain") return null;
  return (
    <ICD.WithFormats icdKeys={[target.icd]}>
      <Targets />
    </ICD.WithFormats>
  );
};

const Renderer = Task.wrapForm<typeof schemas>({
  Form,
  schemas,
  deployConfigZ: schemas.config,
  type: "mqtt_write",
  onConfigure: async (_client, config) => [config, 0],
});

describe("ICD.Write.useTree", () => {
  const client = createTestClient();

  const createSetpoint = () =>
    icd.jsonFieldZ.parse({ name: "setpoint", pointer: "/setpoint", type: "float" });

  // Creates a draft with a target for each topic. Each sends one message of fields, on
  // channels unless bindingsOf says otherwise.
  const renderTargets = async (
    topics: string[],
    fields: icd.JSONField[] = [createSetpoint()],
    bindingsOf: (i: number) => icd.WriteBinding[] = () => ICD.Write.suit([], fields),
  ) => {
    const message = {
      key: uuid.create(),
      name: "Command",
      payload: { format: "json" as const, fields },
    };
    const ic = await client.icds.create({
      name: uniqueName("icd"),
      messages: [message],
    });
    const targets = topics.map((topic, i) =>
      mqtt.plainWriteTargetZ.parse({
        type: "plain",
        key: `t${i}`,
        topic,
        icd: ic.key,
        message: { message: message.key, bindings: bindingsOf(i) },
      }),
    );
    const draft = await client.tasks.create(
      {
        name: uniqueName("tsk"),
        type: "mqtt_write",
        config: mqtt.writeConfigZ.parse({ targets }),
      },
      schemas,
    );
    await renderTaskFormTab(Renderer, { client, taskKey: draft.key });
    const retrieveTargets = async () =>
      (await client.tasks.retrieve({ key: draft.key, schemas })).config.targets;
    const retrieveFields = async () => {
      const [{ payload }] = (await client.icds.retrieve({ key: ic.key })).messages;
      if (payload.format !== "json") throw new Error("The message is not JSON");
      return payload.fields;
    };
    return { retrieveTargets, retrieveFields };
  };

  const createRamp = () =>
    icd.jsonFieldZ.parse({ name: "ramp", pointer: "/ramp", type: "float" });

  /** @returns the bindings of a target, each as its field name and its source. */
  const bindingsOf = (target: mqtt.WriteTarget, fields: icd.JSONField[]) => {
    if (target.type !== "plain") throw new Error("The target sends no message");
    const names = new Map(fields.map(({ key, name }) => [key, name]));
    return target.message.bindings.map((b) => [
      names.get(b.field),
      b.type === "channel" ? b.channel : b.value.value,
    ]);
  };

  const levelOf = (name: string): string | null =>
    screen.getByRole("treeitem", { name: new RegExp(`^${name}`) }).ariaLevel;

  it("should put the targets of one group under it, each with its fields", async () => {
    await renderTargets(["a", "a", "b"]);
    await screen.findByRole("treeitem", { name: /^Topic a/ });
    expect(levelOf("Topic a")).toBe("1");
    expect(levelOf("Topic b")).toBe("1");
    expect(levelOf("Target t0")).toBe("2");
    expect(levelOf("Target t2")).toBe("2");
    const fields = screen.getAllByRole("treeitem", { name: /^setpoint/ });
    expect(fields.map(({ ariaLevel }) => ariaLevel)).toEqual(["3", "3", "3"]);
  });

  it("should name the source of a field that is on no channel", async () => {
    // The Core refuses a message with no field on a channel, so setpoint keeps one.
    const setpoint = createSetpoint();
    const mode = icd.jsonFieldZ.parse({
      name: "mode",
      pointer: "/mode",
      type: "float",
      default: 0,
    });
    const value: icd.WriteBinding = {
      type: "value",
      field: mode.key,
      value: { type: "float", value: 5 },
    };
    await renderTargets(["a", "b"], [setpoint, mode], (i) => [
      ...ICD.Write.suit([], [setpoint]),
      ...(i === 0 ? [value] : []),
    ]);
    const [valued, defaulted] = await screen.findAllByRole("treeitem", {
      name: /^mode/,
    });
    expect(within(valued).getByText("Value")).toBeTruthy();
    expect(within(defaulted).getByText("Default")).toBeTruthy();
    const [bound] = screen.getAllByRole("treeitem", { name: /^setpoint/ });
    expect(within(bound).queryByText(/^(Value|Default)$/)).toBeNull();
  });

  it("should disable each target of a group from the menu of the group", async () => {
    const { retrieveTargets } = await renderTargets(["a", "a", "b"]);
    fireEvent.contextMenu(await screen.findByRole("treeitem", { name: /^Topic a/ }));
    fireEvent.click(await screen.findByText("Disable"));
    await waitFor(async () =>
      expect(
        (await retrieveTargets()).map(({ key, disabled }) => [key, disabled]),
      ).toEqual([
        ["t0", true],
        ["t1", true],
        ["t2", false],
      ]),
    );
  });

  it("should enable the disabled targets of a group from the menu of the group", async () => {
    const { retrieveTargets } = await renderTargets(["a", "a", "b"]);
    const group = await screen.findByRole("treeitem", { name: /^Topic a/ });
    fireEvent.contextMenu(group);
    fireEvent.click(await screen.findByText("Disable"));
    // The targets start enabled, so the Core must store the disable first.
    await waitFor(async () =>
      expect((await retrieveTargets()).map(({ disabled }) => disabled)).toEqual([
        true,
        true,
        false,
      ]),
    );
    fireEvent.contextMenu(group);
    const enable = await screen.findByText("Enable");
    expect(screen.queryByText("Disable")).toBeNull();
    fireEvent.click(enable);
    await waitFor(async () =>
      expect((await retrieveTargets()).map(({ disabled }) => disabled)).toEqual([
        false,
        false,
        false,
      ]),
    );
  });

  it("should remove a target from its menu", async () => {
    const { retrieveTargets } = await renderTargets(["a", "b"]);
    fireEvent.contextMenu(await screen.findByRole("treeitem", { name: /^Target t0/ }));
    fireEvent.click(await screen.findByText("Remove"));
    await waitFor(async () =>
      expect((await retrieveTargets()).map(({ key }) => key)).toEqual(["t1"]),
    );
  });

  it("should remove a field from its message and from each target that sends it", async () => {
    const { retrieveTargets, retrieveFields } = await renderTargets(
      ["a", "b"],
      [createSetpoint(), createRamp()],
    );
    const [ramp] = await screen.findAllByRole("treeitem", { name: /^ramp/ });
    fireEvent.contextMenu(ramp);
    fireEvent.click(await screen.findByText("Remove"));
    await waitFor(async () => {
      const fields = await retrieveFields();
      expect(fields.map(({ name }) => name)).toEqual(["setpoint"]);
      const targets = await retrieveTargets();
      expect(targets.map((t) => bindingsOf(t, fields))).toEqual([
        [["setpoint", 0]],
        [["setpoint", 0]],
      ]);
    });
    expect(screen.queryByRole("treeitem", { name: /^ramp/ })).toBeNull();
  });

  it("should copy a field after it, bound as the field is but on a new channel", async () => {
    const ch = await client.channels.create({
      name: uniqueName("cmd"),
      dataType: "float64",
      virtual: true,
    });
    const setpoint = createSetpoint();
    const ramp = createRamp();
    const { retrieveTargets, retrieveFields } = await renderTargets(
      ["a"],
      [setpoint, ramp],
      () => [
        { type: "channel", field: setpoint.key, channel: ch.key },
        { type: "value", field: ramp.key, value: { type: "float", value: 5 } },
      ],
    );
    fireEvent.contextMenu(await screen.findByRole("treeitem", { name: /^setpoint/ }));
    fireEvent.click(await screen.findByText("Duplicate"));
    const copy = await screen.findByRole("treeitem", { name: /^setpoint_2/ });
    expect(copy.ariaSelected).toBe("true");
    fireEvent.contextMenu(screen.getByRole("treeitem", { name: /^ramp/ }));
    fireEvent.click(await screen.findByText("Duplicate"));
    await waitFor(async () => {
      const fields = await retrieveFields();
      expect(fields.map(({ name, pointer }) => [name, pointer])).toEqual([
        ["setpoint", "/setpoint"],
        ["setpoint_2", "/setpoint_2"],
        ["ramp", "/ramp"],
        ["ramp_2", "/ramp_2"],
      ]);
      const [target] = await retrieveTargets();
      expect(bindingsOf(target, fields)).toEqual([
        ["setpoint", ch.key],
        ["ramp", 5],
        ["setpoint_2", 0],
        ["ramp_2", 5],
      ]);
    });
  });

  it("should offer no Duplicate on a field at the whole document", async () => {
    const value = icd.jsonFieldZ.parse({ name: "value", pointer: "", type: "float" });
    await renderTargets(["a"], [value]);
    fireEvent.contextMenu(await screen.findByRole("treeitem", { name: /^value/ }));
    expect(await screen.findByText("Remove")).toBeTruthy();
    expect(screen.queryByText("Duplicate")).toBeNull();
  });

  it("should add a field to a target from its menu, and select the field", async () => {
    await renderTargets(["a", "b"]);
    fireEvent.contextMenu(await screen.findByRole("treeitem", { name: /^Target t1/ }));
    fireEvent.click(await screen.findByText("Add field"));
    // The targets send one message, so the field shows under each.
    const added = await screen.findAllByRole("treeitem", { name: /^Field 2/ });
    expect(added.map(({ ariaSelected }) => ariaSelected === "true")).toEqual([
      false,
      true,
    ]);
  });

  it("should offer Add field only on one target", async () => {
    await renderTargets(["a"]);
    fireEvent.contextMenu(await screen.findByRole("treeitem", { name: /^Topic a/ }));
    expect(await screen.findByText("Disable")).toBeTruthy();
    expect(screen.queryByText("Add field")).toBeNull();
  });
});
