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
import { fireEvent, screen, waitFor } from "@testing-library/react";
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
        bindings: deep.variant(path.config.targets[target.key], "type", "plain").message
          .bindings,
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

  // Creates a draft with a target for each topic. Each sends one message of one field.
  const renderTargets = async (topics: string[]) => {
    const field = icd.jsonFieldZ.parse({
      name: "setpoint",
      pointer: "/setpoint",
      type: "float",
    });
    const message = {
      key: uuid.create(),
      name: "Command",
      payload: { format: "json" as const, fields: [field] },
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
        message: { message: message.key, bindings: ICD.Write.suit([], [field]) },
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
    return { retrieveTargets };
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

  it("should remove a target from its menu", async () => {
    const { retrieveTargets } = await renderTargets(["a", "b"]);
    fireEvent.contextMenu(await screen.findByRole("treeitem", { name: /^Target t0/ }));
    fireEvent.click(await screen.findByText("Remove"));
    await waitFor(async () =>
      expect((await retrieveTargets()).map(({ key }) => key)).toEqual(["t1"]),
    );
  });

  it("should give a field no menu items", async () => {
    await renderTargets(["a"]);
    fireEvent.contextMenu(await screen.findByRole("treeitem", { name: /^setpoint/ }));
    // Every menu ends with Reload Console, so its presence proves the menu rendered.
    await screen.findByText("Reload Console");
    expect(screen.queryByText("Disable")).toBeNull();
    expect(screen.queryByText("Remove")).toBeNull();
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
