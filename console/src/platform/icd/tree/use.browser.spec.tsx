// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, icd, mqtt } from "@synnaxlabs/client";
import { createTestClient, holdRequests } from "@synnaxlabs/client/testutil";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import type { Tree } from "@synnaxlabs/lyra/tree";
import { deep, payload, unique, uuid } from "@synnaxlabs/x";
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
  const { tree, handleSelect, contextMenu, renderItem, setNewMessage } =
    ICD.Tree.use<mqtt.ReadEntry>({
      sources: path.config.entries,
      sourceOf: (e) => (e.type === "plain" ? e : null),
      pathOf: topicPath,
      SourceItem,
      missingDescription: () => "The ICD has no such message.",
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
    <>
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
      {payload.FORMATS.map((format) => (
        <button
          key={format}
          type="button"
          onClick={() => setNewMessage(entries[0].key, format)}
        >
          New {format} message
        </button>
      ))}
    </>
  );
};

const Form: FC = () => {
  // The task form renders before it loads the config of the task.
  const entries = PForm.useFieldValue(path.config.entries, { optional: true });
  const icdKeys = unique.unique(
    (entries ?? []).flatMap((e) => (e.type === "plain" ? [e.icd] : [])),
  );
  if (icdKeys.length === 0) return null;
  return (
    <ICD.WithFormats icdKeys={icdKeys}>
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

  describe("delete from ICD", () => {
    const client = createTestClient();

    const createMessage = (name: string, fields: string[]): icd.Message => ({
      key: uuid.create(),
      name,
      payload: {
        format: "json",
        fields: fields.map((f) =>
          icd.jsonFieldZ.parse({ name: f, pointer: `/${f}`, type: "float" }),
        ),
      },
    });

    const createICD = async (...messages: icd.Message[]): Promise<icd.ICD> =>
      await client.icds.create({ name: uniqueName("icd"), messages });

    // Renders a task with a source for each message, which reads it.
    const renderSources = async (
      ...sources: [icd.ICD, icd.Message][]
    ): Promise<{ name: string }> => {
      const entries = sources.map(([ic, message], i) =>
        mqtt.plainReadEntryZ.parse({
          type: "plain",
          key: `s${i}`,
          topic: `plant/oven${i}`,
          icd: ic.key,
          messages: [icd.readMessageZ.parse({ message: message.key })],
        }),
      );
      const rack = await client.racks.create({ name: uniqueName("rack") });
      const tsk = await rack.createTask({
        name: uniqueName("tsk"),
        type: "mqtt_read",
        config: mqtt.readConfigZ.parse({ entries }),
      });
      await renderTaskFormTab(Renderer, { client, taskKey: tsk.key });
      return tsk;
    };

    // Chooses the items and opens the menu on the last of them.
    const openMenu = async (...items: HTMLElement[]): Promise<void> => {
      fireEvent.click(items[0]);
      fireEvent.keyDown(window, { code: "Control" });
      for (const item of items.slice(1)) fireEvent.click(item);
      fireEvent.keyUp(window, { code: "Control" });
      fireEvent.contextMenu(items.at(-1) as HTMLElement);
    };

    const findField = async (name: string): Promise<HTMLElement[]> =>
      await screen.findAllByRole("treeitem", { name: new RegExp(`^${name}`) });

    const fieldNames = async (key: icd.Key): Promise<string[]> =>
      (await client.icds.retrieve({ key })).messages.flatMap(({ payload }) =>
        payload.fields.map(({ name }) => name),
      );

    /** Edits the ICD through a client that the form does not use. */
    const dispatchElsewhere = async (
      key: icd.Key,
      action: icd.Action,
    ): Promise<void> => {
      const other = createTestClient();
      await other.icds.retrieve({ key });
      expect(await other.icds.dispatch(key, action)).toBe(true);
    };

    const clickDelete = async (): Promise<void> => {
      fireEvent.click(await screen.findByRole("menuitem", { name: "Delete from ICD" }));
    };

    it("should offer Delete from ICD in place of Remove on a field", async () => {
      const status = createMessage("Status", ["temperature", "pressure"]);
      await renderSources([await createICD(status), status]);
      await openMenu(...(await findField("pressure")));
      await screen.findByRole("menuitem", { name: "Delete from ICD" });
      expect(screen.queryByRole("menuitem", { name: "Remove" })).toBeNull();
    });

    it("should ask before it deletes a field, and keep it on cancel", async () => {
      const status = createMessage("Status", ["temperature", "pressure"]);
      const ic = await createICD(status);
      await renderSources([ic, status]);
      await openMenu(...(await findField("pressure")));
      await clickDelete();
      await screen.findByText("Delete pressure from Status?");
      screen.getByText(
        `This deletes it from the ICD ${ic.name}. ` +
          "Tasks that bind it lose that channel.",
      );
      fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
      await waitFor(() =>
        expect(screen.queryByText("Delete pressure from Status?")).toBeNull(),
      );
      expect(await fieldNames(ic.key)).toEqual(["temperature", "pressure"]);
    });

    it("should delete the chosen fields once the user confirms", async () => {
      const status = createMessage("Status", ["temperature", "pressure", "flow"]);
      const ic = await createICD(status);
      await renderSources([ic, status]);
      const [temperature] = await findField("temperature");
      const [pressure] = await findField("pressure");
      await openMenu(temperature, pressure);
      await clickDelete();
      await screen.findByText("Delete 2 fields from Status?");
      screen.getByText(
        `This deletes them from the ICD ${ic.name}. ` +
          "Tasks that bind them lose those channels.",
      );
      fireEvent.click(screen.getByRole("button", { name: "Delete" }));
      await waitFor(async () => expect(await fieldNames(ic.key)).toEqual(["flow"]));
    });

    it("should list the tasks that use the ICD", async () => {
      const status = createMessage("Status", ["temperature"]);
      const ic = await createICD(status);
      const tsk = await renderSources([ic, status]);
      await openMenu(...(await findField("temperature")));
      await clickDelete();
      await screen.findByText(`Tasks that use ${ic.name}`);
      await screen.findByText(tsk.name);
    });

    it("should name the ICD when the fields are in several of its messages", async () => {
      const status = createMessage("Status", ["temperature"]);
      const alarm = createMessage("Alarm", ["level"]);
      const ic = await createICD(status, alarm);
      await renderSources([ic, status], [ic, alarm]);
      const [temperature] = await findField("temperature");
      const [level] = await findField("level");
      await openMenu(temperature, level);
      await clickDelete();
      await screen.findByText(`Delete 2 fields from ${ic.name}?`);
    });

    it("should count a field once when several sources read it", async () => {
      const status = createMessage("Status", ["temperature", "pressure"]);
      const ic = await createICD(status);
      await renderSources([ic, status], [ic, status]);
      await openMenu(...(await findField("pressure")));
      await clickDelete();
      await screen.findByText("Delete pressure from Status?");
    });

    it("should delete the rest when another client deletes a chosen field while it asks", async () => {
      const status = createMessage("Status", ["temperature", "pressure", "flow"]);
      const ic = await createICD(status);
      await renderSources([ic, status]);
      const [temperature] = await findField("temperature");
      const [pressure] = await findField("pressure");
      await openMenu(temperature, pressure);
      await clickDelete();
      await screen.findByText("Delete 2 fields from Status?");
      await dispatchElsewhere(
        ic.key,
        icd.remove({
          path: ["messages", status.key, "payload", "fields"],
          key: status.payload.fields[1].key,
        }),
      );
      await waitFor(() =>
        expect(screen.queryByRole("treeitem", { name: /^pressure/ })).toBeNull(),
      );
      fireEvent.click(screen.getByRole("button", { name: "Delete" }));
      await waitFor(async () => expect(await fieldNames(ic.key)).toEqual(["flow"]));
    });

    it("should say this field and this message for blank names", async () => {
      const status = createMessage("Status", ["temperature", "pressure"]);
      const ic = await createICD(status);
      await renderSources([ic, status]);
      const message = ["messages", status.key];
      for (const path of [
        [...message, "name"],
        [...message, "payload", "fields", status.payload.fields[1].key, "name"],
      ])
        await dispatchElsewhere(ic.key, icd.set({ path, value: " " }));
      await waitFor(() =>
        expect(screen.queryByRole("treeitem", { name: /^pressure/ })).toBeNull(),
      );
      const items = screen.getAllByRole("treeitem");
      const [temperature] = await findField("temperature");
      await openMenu(items[items.indexOf(temperature) + 1]);
      await clickDelete();
      await screen.findByText("Delete this field from this message?");
    });

    it("should say this ICD for a blank ICD name", async () => {
      const status = createMessage("Status", ["temperature"]);
      const alarm = createMessage("Alarm", ["level"]);
      const ic = await createICD(status, alarm);
      await renderSources([ic, status], [ic, alarm]);
      await dispatchElsewhere(ic.key, icd.set({ path: ["name"], value: " " }));
      const [temperature] = await findField("temperature");
      const [level] = await findField("level");
      await openMenu(temperature, level);
      await clickDelete();
      await screen.findByText("Delete 2 fields from this ICD?");
      screen.getByText(
        "This deletes them from this ICD. Tasks that bind them lose those channels.",
      );
    });

    it("should offer no delete for fields of several ICDs", async () => {
      const first = createMessage("Status", ["temperature"]);
      const second = createMessage("Status", ["pressure"]);
      await renderSources(
        [await createICD(first), first],
        [await createICD(second), second],
      );
      const [temperature] = await findField("temperature");
      const [pressure] = await findField("pressure");
      await openMenu(temperature, pressure);
      await screen.findByText("Reload Console");
      expect(screen.queryByRole("menuitem", { name: "Delete from ICD" })).toBeNull();
      expect(screen.queryByRole("menuitem", { name: "Remove" })).toBeNull();
    });
  });

  describe("addMessage", () => {
    const client = createTestClient();

    /** Renders a source that reads one message of payload. @returns its ICD. */
    const renderSource = async (payload: icd.Payload): Promise<icd.Key> => {
      const message: icd.Message = { key: uuid.create(), name: "Status", payload };
      const ic = await client.icds.create({
        name: uniqueName("icd"),
        messages: [message],
      });
      const entry = mqtt.plainReadEntryZ.parse({
        type: "plain",
        key: "s0",
        topic: "plant/oven",
        icd: ic.key,
        messages: [icd.readMessageZ.parse({ message: message.key })],
      });
      const rack = await client.racks.create({ name: uniqueName("rack") });
      const tsk = await rack.createTask({
        name: uniqueName("tsk"),
        type: "mqtt_read",
        config: mqtt.readConfigZ.parse({ entries: [entry] }),
      });
      await renderTaskFormTab(Renderer, { client, taskKey: tsk.key });
      // The Core reads only JSON messages over MQTT, so it would undo the read of each
      // new message. Held saves keep the reads in the form.
      holdRequests(client, "/task/create");
      return ic.key;
    };

    const addMessage = async () => {
      fireEvent.contextMenu(await screen.findByRole("treeitem", { name: /^s0/ }));
      fireEvent.click(await screen.findByText("Add message"));
    };

    const retrieveFormats = async (key: icd.Key) =>
      (await client.icds.retrieve({ key })).messages.map(
        ({ payload }) => payload.format,
      );

    it("should add a text message to a source that reads a text message", async () => {
      const key = await renderSource(icd.textPayloadZ.parse({ format: "text" }));
      await addMessage();
      await waitFor(async () =>
        expect(await retrieveFormats(key)).toEqual(["text", "text"]),
      );
    });

    it("should add a binary message to a source that reads a binary message", async () => {
      const key = await renderSource(icd.binaryPayloadZ.parse({ format: "binary" }));
      await addMessage();
      await waitFor(async () =>
        expect(await retrieveFormats(key)).toEqual(["binary", "binary"]),
      );
    });
  });

  describe("setNewMessage", () => {
    const client = createTestClient();

    it("should delete an empty message of any format that the task made when the source reads a new one", async () => {
      const message: icd.Message = {
        key: uuid.create(),
        name: "Status",
        payload: { format: "json", fields: [] },
      };
      const ic = await client.icds.create({
        name: uniqueName("icd"),
        messages: [message],
      });
      const entry = mqtt.plainReadEntryZ.parse({
        type: "plain",
        key: "s0",
        topic: "plant/oven",
        icd: ic.key,
        messages: [icd.readMessageZ.parse({ message: message.key })],
      });
      const rack = await client.racks.create({ name: uniqueName("rack") });
      const tsk = await rack.createTask({
        name: uniqueName("tsk"),
        type: "mqtt_read",
        config: mqtt.readConfigZ.parse({ entries: [entry] }),
      });
      await renderTaskFormTab(Renderer, { client, taskKey: tsk.key });
      // The Core reads only JSON messages over MQTT, so it would undo the read of each
      // new message. Held saves keep the reads in the form.
      holdRequests(client, "/task/create");
      const retrieveMessages = async () =>
        (await client.icds.retrieve({ key: ic.key })).messages;
      fireEvent.click(await screen.findByRole("button", { name: "New text message" }));
      const text = await waitFor(async () => {
        const messages = await retrieveMessages();
        expect(messages).toHaveLength(2);
        return messages[1];
      });
      expect(text).toMatchObject({
        name: "Message",
        payload: { format: "text", delimiter: ",", prefix: "", fields: [] },
      });
      fireEvent.click(screen.getByRole("button", { name: "New binary message" }));
      await waitFor(async () =>
        expect(await retrieveMessages()).toMatchObject([
          { key: message.key },
          { name: "Message", payload: { format: "binary", fields: [] } },
        ]),
      );
    });
  });
});
