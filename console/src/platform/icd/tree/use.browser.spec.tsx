// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, icd, mqtt } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import type { Tree } from "@synnaxlabs/lyra/tree";
import { deep, uuid } from "@synnaxlabs/x";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { type FC, useRef, useState } from "react";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { ICD } from "@/platform/icd";
import { Task } from "@/platform/task";
import {
  awaitTare,
  renderTaskFormTab,
  reportTaskRunning,
} from "@/platform/task/testutil";
import { getIconButtons } from "@/testutil";

// The Core keeps the config of a known task type only, so the sources are MQTT topics.
const schemas = {
  type: z.literal("mqtt_read"),
  config: mqtt.readConfigZ,
  statusData: z.object({ running: z.boolean() }).nullish(),
};

const path = deep.path<Task.Values<typeof schemas>>();

const topicPath = (key: string) =>
  deep.variant(path.config.entries[key], "type", "plain");

const SourceItem = (props: Tree.ItemRenderProps<string>) => (
  <Task.Views.TreeGroupItem {...props} label={props.itemKey} />
);

const Sources: FC = () => {
  const entries = PForm.useFieldValue(path.config.entries);
  const [selected, setSelected] = useState<string[]>([]);
  const [expanded, setExpanded] = useState(() => ICD.Tree.initialExpanded(entries));
  const made = useRef(new Set<string>());
  const { tree, handleSelect, contextMenu, renderItem } = ICD.Tree.use<mqtt.ReadEntry>({
    sources: path.config.entries,
    sourceOf: (e) => (e.type === "plain" ? e : null),
    pathOf: topicPath,
    SourceItem,
    bindingOf: (_, __, { key }) => icd.readBindingZ.parse({ field: key }),
    tareable: true,
    messageName: () => "Message",
    made,
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
      emptyMessage="No sources"
      addLabel="Add source"
      onAdd={() => {}}
    >
      {renderItem}
    </Task.Views.TreeList>
  );
};

const Form: FC = () => {
  // The task form renders before it loads the config of the task.
  const entries = PForm.useFieldValue(path.config.entries, { optional: true });
  const [entry] = entries ?? [];
  if (entry?.type !== "plain") return null;
  return (
    <ICD.WithFormats icdKeys={[entry.icd]}>
      <ICD.Plans plans={[]}>
        <Sources />
      </ICD.Plans>
    </ICD.WithFormats>
  );
};

const Renderer = Task.wrapForm<typeof schemas>({
  Form,
  schemas,
  deployConfigZ: schemas.config,
  type: "mqtt_read",
  onConfigure: async (_client, config) => [config, 0],
});

describe("ICD.Tree.use", () => {
  describe("tare", () => {
    const client = createTestClient();

    const createChannel = async (dataType: DataType) =>
      (
        await client.channels.create({
          name: uniqueName("tare"),
          dataType,
          virtual: true,
        })
      ).key;

    // Creates a running task with a source for each value of disabled. Each source
    // reads a float and an integer field of one message.
    const createRunningTask = async (disabled = [false]) => {
      const fields = [
        icd.jsonFieldZ.parse({ name: "float", pointer: "/float", type: "float" }),
        icd.jsonFieldZ.parse({ name: "count", pointer: "/count", type: "integer" }),
      ];
      const message = {
        key: uuid.create(),
        name: "Status",
        payload: { format: "json" as const, fields },
      };
      const ic = await client.icds.create({
        name: uniqueName("icd"),
        messages: [message],
      });
      const floats: number[] = [];
      const entries = await Promise.all(
        disabled.map(async (off, i) => {
          const [float, integer] = await Promise.all([
            createChannel(DataType.FLOAT32),
            createChannel(DataType.INT32),
          ]);
          floats[i] = float;
          return mqtt.plainReadEntryZ.parse({
            type: "plain",
            key: `s${i}`,
            topic: `plant/oven${i}`,
            disabled: off,
            icd: ic.key,
            messages: [
              icd.readMessageZ.parse({
                message: message.key,
                bindings: [
                  { field: fields[0].key, channel: float },
                  { field: fields[1].key, channel: integer },
                ],
              }),
            ],
          });
        }),
      );
      const rack = await client.racks.create({ name: uniqueName("rack") });
      const tsk = await rack.createTask({
        name: uniqueName("tsk"),
        type: "mqtt_read",
        config: mqtt.readConfigZ.parse({ entries }),
      });
      await reportTaskRunning(client, tsk);
      return { task: tsk.key, floats };
    };

    // Resolves with the tare command that the action on the form of a task sends.
    const tareThrough = async (
      action: (container: HTMLElement) => Promise<void>,
      disabled?: boolean[],
    ): Promise<{ args: unknown; floats: number[] }> => {
      const { task, floats } = await createRunningTask(disabled);
      const args = await awaitTare(client, task, async () => {
        const { container } = await renderTaskFormTab(Renderer, {
          client,
          taskKey: task,
        });
        await action(container);
      });
      return { args, floats };
    };

    it("should tare from the one button, which a float field shows", async () => {
      const { args, floats } = await tareThrough(async (container) => {
        await waitFor(() => expect(getIconButtons(container, "tare")).toHaveLength(1));
        fireEvent.click(getIconButtons(container, "tare")[0]);
      });
      expect(args).toEqual({ keys: floats });
    });

    it("should tare the channel of a field from its menu", async () => {
      const { args, floats } = await tareThrough(async () => {
        fireEvent.contextMenu(await screen.findByText("float"));
        fireEvent.click(await screen.findByText("Tare"));
      });
      expect(args).toEqual({ keys: floats });
    });

    it("should tare no field of a source that the task skips from the menu", async () => {
      const { args, floats } = await tareThrough(async () => {
        const [first, second] = await waitFor(() => {
          const found = screen.getAllByText("float");
          expect(found).toHaveLength(2);
          return found;
        });
        fireEvent.click(first);
        fireEvent.keyDown(window, { code: "Shift" });
        fireEvent.click(second, { shiftKey: true });
        fireEvent.keyUp(window, { code: "Shift" });
        fireEvent.contextMenu(second);
        fireEvent.click(await screen.findByText("Tare"));
      }, [false, true]);
      expect(args).toEqual({ keys: [floats[0]] });
    });

    it("should disable the tare button of a field whose source the task skips", async () => {
      const { task } = await createRunningTask([true]);
      const { container } = await renderTaskFormTab(Renderer, {
        client,
        taskKey: task,
      });
      await waitFor(() => expect(getIconButtons(container, "tare")).toHaveLength(1));
      expect(getIconButtons(container, "tare")[0].getAttribute("aria-disabled")).toBe(
        "true",
      );
    });
  });
});
