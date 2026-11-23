// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, icd, mqtt, type Synnax, type task } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import type { Status } from "@synnaxlabs/lyra/status";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { MQTT } from "@/feature/mqtt";
import { createBroker, openScanner } from "@/feature/mqtt/testutil";
import { Device } from "@/platform/device";
import { ICD } from "@/platform/icd";
import {
  pickMessageFormat,
  removeField,
  retype,
  selectedIn,
} from "@/platform/icd/testutil";
import {
  clickDeploy,
  createChannelReadOnlyClient,
  deployAndAwaitTask,
  findChannelListItem,
  preloadForms,
  renderTaskFormTab,
  type RenderTaskFormTabOptions,
  selectFromDropdown,
} from "@/platform/task/testutil";
import {
  awaitTextEditingElement,
  commitTextEdit,
  getSwitchInput,
  isSelectButtonSelected,
} from "@/testutil";

const client = createTestClient();

preloadForms(MQTT.Task.FORMS);

// Drafts carry no key; the created task mints its own.
const ZERO_DRAFT: task.New<MQTT.Task.WriteSchemas> = {
  name: "MQTT write task",
  type: MQTT.Task.WRITE_TYPE,
  config: MQTT.Task.WRITE_SCHEMAS.config.parse({}),
};

const createDraft = async (client: Synnax, config: mqtt.WriteConfig) =>
  await client.tasks.create({ ...ZERO_DRAFT, config }, MQTT.Task.WRITE_SCHEMAS);

/**
 * Creates an ICD that holds one message with a float field at /value, and a plain
 * target that sends the commands of a new channel to it.
 */
const createPlainTarget = async (
  key: string,
  topic: string,
  overrides: Partial<MQTT.Task.PlainWriteTarget> = {},
): Promise<MQTT.Task.PlainWriteTarget> => {
  const message = icd.messageZ.parse({
    name: "Command",
    payload: {
      format: "json",
      fields: [{ name: "value", pointer: "/value", type: "float" }],
    },
  });
  const [ic, cmd] = await Promise.all([
    client.icds.create({ name: uniqueName("format"), messages: [message] }),
    client.channels.create({
      name: uniqueName("mqtt_cmd"),
      dataType: "float64",
      virtual: true,
    }),
  ]);
  if (message.payload.format !== "json") throw new Error("expected a JSON message");
  const [field] = message.payload.fields;
  return {
    ...mqtt.plainWriteTargetZ.parse({
      type: "plain",
      icd: ic.key,
      message: {
        message: message.key,
        bindings: [{ type: "channel", field: field.key, channel: cmd.key }],
      },
    }),
    key,
    topic,
    ...overrides,
  };
};

const createTagTarget = (
  key: string,
  tag: string,
  overrides: Partial<MQTT.Task.SparkplugWriteTarget> = {},
): MQTT.Task.SparkplugWriteTarget => ({
  ...mqtt.sparkplugWriteTargetZ.parse({ type: "sparkplug" }),
  key,
  group: "plant",
  edgeNode: "line1",
  tag,
  ...overrides,
});

const retrieveTargets = async (key: task.Key) =>
  (await client.tasks.retrieve({ key, schemas: MQTT.Task.WRITE_SCHEMAS })).config
    .targets;

const createWriteConfig = (
  device: string,
  targets: MQTT.Task.WriteTarget[],
): mqtt.WriteConfig => ({
  ...MQTT.Task.WRITE_SCHEMAS.config.parse({}),
  device,
  targets,
});

// The form body renders against a configured broker only.
const renderWrite = async (
  targets: MQTT.Task.WriteTarget[] = [],
  options: RenderTaskFormTabOptions = {},
) => {
  const dev = await createBroker(client);
  const draft = await createDraft(client, createWriteConfig(dev.key, targets));
  const result = await renderTaskFormTab(MQTT.Task.FORMS[MQTT.Task.WRITE_TYPE], {
    client,
    taskKey: draft.key,
    ...options,
  });
  return { ...result, dev, draft };
};

/** Creates an ICD that holds one JSON message of the fields, with unique names. */
const createFormat = async (fields: icd.JSONField[]) => {
  const message = icd.messageZ.parse({
    name: uniqueName("command"),
    payload: { format: "json", fields },
  });
  const ic = await client.icds.create({
    name: uniqueName("valve"),
    messages: [message],
  });
  return { ic, message };
};

const createField = (
  name: string,
  type: icd.JSONField["type"],
  rest: Partial<icd.JSONField> = {},
): icd.JSONField => icd.jsonFieldZ.parse({ name, pointer: `/${name}`, type, ...rest });

/** Creates a plain target that sends a message of the fields, with no channels. */
const createFormatTarget = async (fields: icd.JSONField[]) => {
  const { ic, message } = await createFormat(fields);
  const target = mqtt.plainWriteTargetZ.parse({
    type: "plain",
    key: "t1",
    topic: "plant/valve/set",
    icd: ic.key,
    message: { message: message.key, bindings: ICD.Write.suit([], fields) },
  });
  return { ic, message, target };
};

// The add button renders once the form leaves its loading preview.
const openAddTargets = async () => {
  fireEvent.click(await screen.findByRole("button", { name: "Add targets" }));
  return within(await screen.findByRole("dialog"));
};

const addTag = async (): Promise<void> => {
  const dialog = await openAddTargets();
  fireEvent.click(dialog.getByRole("button", { name: "Sparkplug B" }));
  fireEvent.click(await dialog.findByRole("button", { name: "Add a tag by name" }));
  await screen.findByText("Edge node");
};

// The name deploying gives the command channel of the tag of createTagTarget.
const tagCommandName = (deviceName: string, tag: string): string =>
  `${channel.escapeInvalidName(`${deviceName}_plant_line1_${tag}`)}_cmd`;

describe("MQTT Write form", () => {
  // This first test pays the one-time load of the lazy form, which takes over 10 s on
  // a loaded CI runner.
  it("should show the empty state and add + select a target", async () => {
    await renderWrite();
    await screen.findByText("Select a target to configure");
    await addTag();
    expect(screen.getByText("Sparkplug B type")).toBeTruthy();
  }, 30_000);

  it("should show the topic settings of a plain target", async () => {
    await renderWrite([
      await createPlainTarget("t1", "plant/valve/set", { retained: true }),
    ]);
    await screen.findByText("Publish");
    expect(screen.getByDisplayValue("plant/valve/set")).toBeTruthy();
    expect(screen.getByText("At least once (1)")).toBeTruthy();
    expect(getSwitchInput("Retain").checked).toBe(true);
    expect(screen.queryByText("JSON type")).toBeNull();
  });

  it("should duplicate a plain target with its message and bindings, but no command channels", async () => {
    const plain = await createPlainTarget("t1", "plant/valve/set");
    const [binding] = plain.message.bindings;
    const { draft } = await renderWrite([plain]);
    fireEvent.contextMenu(await screen.findByRole("option", { name: /plant\/valve/ }));
    expect(await screen.findByText("Duplicate")).toBeTruthy();
    expect(screen.queryByText("Rename")).toBeNull();
    fireEvent.click(screen.getByText("Duplicate"));
    await waitFor(async () => {
      const targets = await retrieveTargets(draft.key);
      expect(targets).toMatchObject([
        { key: "t1", message: plain.message },
        {
          topic: plain.topic,
          icd: plain.icd,
          message: { ...plain.message, bindings: [{ ...binding, channel: 0 }] },
        },
      ]);
      expect(targets[1].key).not.toBe("t1");
    });
  });

  it("should rename the command channel of a tag via the context menu", async () => {
    const { dev } = await renderWrite([createTagTarget("s1", "setpoint")]);
    const name = tagCommandName(dev.name, "setpoint");
    fireEvent.contextMenu(within(await screen.findByRole("option")).getByText(name));
    fireEvent.click(await screen.findByText("Rename"));
    const editable = await awaitTextEditingElement();
    commitTextEdit(editable, "my_cmd_channel");
    await screen.findByText("my_cmd_channel");
  });

  it("should mark a new command channel, unless its target is disabled", async () => {
    const { dev } = await renderWrite([
      createTagTarget("s1", "setpoint"),
      createTagTarget("s2", "spare", { disabled: true }),
    ]);
    const on = await findChannelListItem(tagCommandName(dev.name, "setpoint"));
    expect(within(on).getByRole("img", { name: "New channel" })).toBeTruthy();
    const off = await findChannelListItem(tagCommandName(dev.name, "spare"));
    expect(within(off).queryByRole("img", { name: "New channel" })).toBeNull();
  });

  it("should withhold rename from a subject who cannot update channels", async () => {
    const { dev } = await renderWrite([createTagTarget("s1", "setpoint")], {
      as: await createChannelReadOnlyClient(client),
    });
    const name = tagCommandName(dev.name, "setpoint");
    fireEvent.contextMenu(within(await screen.findByRole("option")).getByText(name));
    // Duplicate is ungated, so its presence proves the menu resolved before the
    // absence below is read.
    expect(await screen.findByText("Duplicate")).toBeTruthy();
    expect(screen.queryByText("Rename")).toBeNull();
  });

  it("should bind a tag to the command channel the broker stores, and leave a plain target as it is", async () => {
    const tagCh = await client.channels.create({
      name: uniqueName("mqtt_tag_cmd"),
      dataType: "string",
      virtual: true,
    });
    const dev = await createBroker(client, {
      properties: { write: { "spBv1.0/plant/line1//setpoint": tagCh.key } },
    });
    const plain = await createPlainTarget("t1", "plant/cmd");
    const draft = await createDraft(
      client,
      createWriteConfig(dev.key, [plain, createTagTarget("s1", "setpoint")]),
    );
    await renderTaskFormTab(MQTT.Task.FORMS[MQTT.Task.WRITE_TYPE], {
      client,
      taskKey: draft.key,
    });
    await waitFor(async () =>
      expect(await retrieveTargets(draft.key)).toMatchObject([
        { message: plain.message },
        { channel: tagCh.key },
      ]),
    );
  });

  it("should disable a target through its checkbox", async () => {
    const { draft } = await renderWrite([
      await createPlainTarget("t1", "plant/valve/set"),
    ]);
    const item = await screen.findByRole("option", { name: /plant\/valve\/set/ });
    fireEvent.click(within(item).getByRole("checkbox", { name: "Enabled" }));
    await waitFor(async () =>
      expect(await retrieveTargets(draft.key)).toMatchObject([{ disabled: true }]),
    );
  });

  describe("Sparkplug B targets", () => {
    it("should list a tag with its edge node and its command channel", async () => {
      const ch = await client.channels.create({
        name: uniqueName("mqtt_cmd"),
        dataType: "float64",
        virtual: true,
      });
      await renderWrite([
        await createPlainTarget("t1", "plant/valve/set"),
        createTagTarget("s1", "setpoint", { device: "ovenA", channel: ch.key }),
      ]);
      await screen.findByRole("option", { name: /JSON.*plant\/valve\/set/ });
      expect(
        screen.getByRole("option", { name: /SpB.*plant\/line1\/ovenA\/setpoint/ }),
      ).toBeTruthy();
      await screen.findByText(ch.name);
    });

    it("should focus the group of a tag added by name", async () => {
      await renderWrite();
      const dialog = await openAddTargets();
      fireEvent.click(dialog.getByRole("button", { name: "Sparkplug B" }));
      fireEvent.click(await dialog.findByRole("button", { name: "Add a tag by name" }));
      const group = await screen.findByPlaceholderText("plant");
      await waitFor(() => expect(document.activeElement).toBe(group));
    });

    it("should add a tag by name and select its Sparkplug B type", async () => {
      const { draft } = await renderWrite();
      await screen.findByText("No targets");
      await addTag();
      expect(screen.getByPlaceholderText("Optional")).toBeTruthy();
      expect(screen.queryByText("JSON type")).toBeNull();
      await selectFromDropdown("Double", "DateTime");
      await waitFor(async () =>
        expect(await retrieveTargets(draft.key)).toMatchObject([
          { type: "sparkplug", tagType: "date_time", channel: 0 },
        ]),
      );
    });

    it("should duplicate a tag with no command channel", async () => {
      const ch = await client.channels.create({
        name: uniqueName("mqtt_cmd"),
        dataType: "float32",
        virtual: true,
      });
      const { draft } = await renderWrite([
        createTagTarget("s1", "setpoint", { channel: ch.key, tagType: "float" }),
      ]);
      fireEvent.contextMenu(await screen.findByRole("option", { name: /setpoint/ }));
      fireEvent.click(await screen.findByText("Duplicate"));
      await waitFor(async () => {
        const targets = await retrieveTargets(draft.key);
        expect(targets).toMatchObject([
          { key: "s1", channel: ch.key },
          { tag: "setpoint", tagType: "float", channel: 0 },
        ]);
        expect(targets[1].key).not.toBe("s1");
      });
    });
  });

  describe("binding the fields of a plain target", () => {
    const fieldGroup = async (name: string) =>
      within(await screen.findByRole("group", { name }));

    it("should put each field with no default on a channel when a message is picked", async () => {
      const { target, message: first } = await createFormatTarget([
        createField("value", "float"),
      ]);
      const setpoint = createField("setpoint", "float");
      const mode = createField("mode", "string", { default: "auto" });
      const { ic, message } = await createFormat([setpoint, mode]);
      const { draft } = await renderWrite([target]);
      await pickMessageFormat(first.name, message.name, ic.name);
      await waitFor(async () =>
        expect(await retrieveTargets(draft.key)).toMatchObject([
          {
            icd: ic.key,
            message: {
              message: message.key,
              bindings: [{ type: "channel", field: setpoint.key, channel: 0 }],
            },
          },
        ]),
      );
      const modeGroup = await fieldGroup(mode.name);
      expect(modeGroup.getByText('Sends its default, "auto".')).toBeTruthy();
    });

    it("should send a fixed value for a field set to Value", async () => {
      const setpoint = createField("setpoint", "float");
      const mode = createField("mode", "string");
      const { target } = await createFormatTarget([setpoint, mode]);
      const { draft } = await renderWrite([target]);
      const modeGroup = await fieldGroup(mode.name);
      fireEvent.click(modeGroup.getByRole("button", { name: "Value" }));
      const input = await modeGroup.findByRole("textbox");
      fireEvent.change(input, { target: { value: "auto" } });
      fireEvent.blur(input);
      await waitFor(async () =>
        expect(await retrieveTargets(draft.key)).toMatchObject([
          {
            message: {
              bindings: [
                { type: "channel", field: setpoint.key, channel: 0 },
                {
                  type: "value",
                  field: mode.key,
                  value: { type: "string", value: "auto" },
                },
              ],
            },
          },
        ]),
      );
    });

    it("should let a field that sends its default take a fixed value", async () => {
      const setpoint = createField("setpoint", "float");
      const mode = createField("mode", "string", { default: "auto" });
      const { target } = await createFormatTarget([setpoint, mode]);
      const { draft } = await renderWrite([target]);
      const modeGroup = await fieldGroup(mode.name);
      fireEvent.click(modeGroup.getByRole("button", { name: "Value" }));
      await modeGroup.findByRole("textbox");
      await waitFor(async () =>
        expect((await retrieveTargets(draft.key))[0]).toMatchObject({
          message: {
            bindings: [
              { type: "channel", field: setpoint.key },
              { type: "value", field: mode.key, value: { type: "string", value: "" } },
            ],
          },
        }),
      );
    });

    it("should start a fixed value inside the min and max of its field", async () => {
      const setpoint = createField("setpoint", "float");
      const low = createField("low", "float", { min: 5 });
      const high = createField("high", "integer", { max: -2.5 });
      // Each limit maps to a raw value through the scale and offset of its field.
      const scaled = createField("scaled", "float", { scale: 0.1, min: 5 });
      const shifted = createField("shifted", "integer", { offset: 10, max: 5 });
      const flipped = createField("flipped", "float", { scale: -2, min: 10 });
      const fields = [setpoint, low, high, scaled, shifted, flipped];
      const { target } = await createFormatTarget(fields);
      const { draft } = await renderWrite([target]);
      for (const field of fields.slice(1))
        fireEvent.click(
          (await fieldGroup(field.name)).getByRole("button", { name: "Value" }),
        );
      await waitFor(async () =>
        expect((await retrieveTargets(draft.key))[0]).toMatchObject({
          message: {
            bindings: [
              { type: "channel", field: setpoint.key },
              { type: "value", field: low.key, value: { type: "float", value: 5 } },
              {
                type: "value",
                field: high.key,
                value: { type: "integer", value: "-3" },
              },
              { type: "value", field: scaled.key, value: { type: "float", value: 50 } },
              {
                type: "value",
                field: shifted.key,
                value: { type: "integer", value: "-5" },
              },
              {
                type: "value",
                field: flipped.key,
                value: { type: "float", value: -5 },
              },
            ],
          },
        }),
      );
    });

    describe("when the message changes in the ICD", () => {
      const messageOf = ({ icd: key, message }: MQTT.Task.PlainWriteTarget) => ({
        icd: key,
        message: message.message,
      });

      it("should warn when the last field leaves the message", async () => {
        const setpoint = createField("setpoint", "float");
        const { target } = await createFormatTarget([setpoint]);
        await renderWrite([target]);
        await fieldGroup(setpoint.name);
        await removeField(client, messageOf(target), setpoint);
        await screen.findByText("Message has no fields");
      });

      it("should drop the binding of a field that leaves the message", async () => {
        const setpoint = createField("setpoint", "float");
        const mode = createField("mode", "string");
        const { target } = await createFormatTarget([setpoint, mode]);
        const { draft } = await renderWrite([target]);
        await fieldGroup(mode.name);
        await removeField(client, messageOf(target), mode);
        await waitFor(async () =>
          expect((await retrieveTargets(draft.key))[0]).toMatchObject({
            message: {
              bindings: [{ type: "channel", field: setpoint.key, channel: 0 }],
            },
          }),
        );
      });

      it("should put a field on a channel when its fixed value stops suiting its kind", async () => {
        const setpoint = createField("setpoint", "float");
        const mode = createField("mode", "string");
        const { target } = await createFormatTarget([setpoint, mode]);
        const { draft } = await renderWrite([
          {
            ...target,
            message: {
              ...target.message,
              bindings: [
                { type: "channel", field: setpoint.key, channel: 0 },
                {
                  type: "value",
                  field: mode.key,
                  value: { type: "string", value: "auto" },
                },
              ],
            },
          },
        ]);
        await fieldGroup(mode.name);
        await retype(client, messageOf(target), mode, "float");
        await waitFor(async () =>
          expect((await retrieveTargets(draft.key))[0]).toMatchObject({
            message: {
              bindings: [
                { type: "channel", field: setpoint.key, channel: 0 },
                { type: "channel", field: mode.key, channel: 0 },
              ],
            },
          }),
        );
      });

      it("should put a field on a channel when it loses its default", async () => {
        const setpoint = createField("setpoint", "float");
        const mode = createField("mode", "string", { default: "auto" });
        const { ic, message, target } = await createFormatTarget([setpoint, mode]);
        const { draft } = await renderWrite([target]);
        await (await fieldGroup(mode.name)).findByText('Sends its default, "auto".');
        await client.icds.dispatch(
          ic.key,
          icd.set({
            path: ["messages", message.key, "payload", "fields", mode.key],
            value: { ...mode, default: undefined },
          }),
        );
        await waitFor(async () =>
          expect((await retrieveTargets(draft.key))[0]).toMatchObject({
            message: {
              bindings: [
                { type: "channel", field: setpoint.key, channel: 0 },
                { type: "channel", field: mode.key, channel: 0 },
              ],
            },
          }),
        );
      });

      it("should unbind a channel whose data type its field stops taking", async () => {
        const setpoint = createField("setpoint", "float");
        const count = createField("count", "integer");
        const [{ target }, cmd] = await Promise.all([
          createFormatTarget([setpoint, count]),
          client.channels.create({
            name: uniqueName("mqtt_cmd"),
            dataType: "int64",
            virtual: true,
          }),
        ]);
        const { draft } = await renderWrite([
          {
            ...target,
            message: {
              ...target.message,
              bindings: [
                { type: "channel", field: setpoint.key, channel: 0 },
                { type: "channel", field: count.key, channel: cmd.key },
              ],
            },
          },
        ]);
        await (await fieldGroup(count.name)).findByDisplayValue(cmd.name);
        await retype(client, messageOf(target), count, "string");
        await waitFor(async () =>
          expect((await retrieveTargets(draft.key))[0]).toMatchObject({
            message: {
              bindings: [
                { type: "channel", field: setpoint.key, channel: 0 },
                { type: "channel", field: count.key, channel: 0 },
              ],
            },
          }),
        );
      });
    });

    it("should send the default of a field set to Default", async () => {
      const setpoint = createField("setpoint", "float");
      const mode = createField("mode", "string", { default: "auto" });
      const { target } = await createFormatTarget([setpoint, mode]);
      const { draft } = await renderWrite([
        {
          ...target,
          message: {
            ...target.message,
            bindings: [
              ...target.message.bindings,
              { type: "channel", field: mode.key, channel: 0 },
            ],
          },
        },
      ]);
      const modeGroup = await fieldGroup(mode.name);
      fireEvent.click(modeGroup.getByRole("button", { name: "Default" }));
      await modeGroup.findByText('Sends its default, "auto".');
      await waitFor(async () =>
        expect((await retrieveTargets(draft.key))[0]).toMatchObject({
          message: { bindings: [{ type: "channel", field: setpoint.key }] },
        }),
      );
    });

    it("should keep the first field on a channel when every field has a default", async () => {
      const mode = createField("mode", "string", { default: "auto" });
      const open = createField("open", "boolean", { default: false });
      const { target } = await createFormatTarget([mode, open]);
      await renderWrite([target]);
      const group = await fieldGroup(mode.name);
      for (const name of ["Value", "Default"])
        expect(group.getByRole("button", { name }).getAttribute("aria-disabled")).toBe(
          "true",
        );
      const openGroup = await fieldGroup(open.name);
      expect(openGroup.getByText("Sends its default, False.")).toBeTruthy();
    });

    it("should open the editor at the message of the target", async () => {
      const { target, ic, message } = await createFormatTarget([
        createField("setpoint", "float"),
      ]);
      const { store } = await renderWrite([target]);
      fireEvent.click(await screen.findByRole("button", { name: "Open in editor" }));
      expect(selectedIn(store, ic.key)).toEqual([message.key]);
    });
  });

  describe("picking targets in the add targets dialog", () => {
    const NODES = [{ group: "plant", edgeNode: "line1", devices: ["ovenA"] }];

    // Opens the dialog over a broker whose scan task answers the edge nodes, then the
    // tags of the first.
    const openOnTags = async (reply: { tags?: object[]; message?: string }) => {
      const { dev, draft } = await renderWrite();
      const scanner = await openScanner(client, dev.rack);
      const dialog = await openAddTargets();
      await scanner.answer({ data: { topics: [], truncated: false } });
      fireEvent.click(dialog.getByRole("button", { name: "Sparkplug B" }));
      await scanner.answer({ data: { nodes: NODES, tags: [] } });
      if (reply.message != null)
        await scanner.answer({ variant: "error", message: reply.message });
      else await scanner.answer({ data: { nodes: [], tags: reply.tags } });
      return { dialog, draft, scanner };
    };

    const TOPIC = {
      topic: "plant/valve/set",
      payload: '{"setpoint": 1, "mode": "auto"}',
      fields: [
        { pointer: "/setpoint", type: "number", sample: "1" },
        { pointer: "/mode", type: "string", sample: '"auto"' },
      ],
      retained: true,
    };
    const FORMAT_REQUIRED = "Select a message format to add topics.";

    // Opens the dialog over a broker whose scan task lists TOPIC, and picks it.
    const openOnPickedTopic = async () => {
      const { dev, draft } = await renderWrite();
      const scanner = await openScanner(client, dev.rack);
      const dialog = await openAddTargets();
      await scanner.answer({ data: { topics: [TOPIC], truncated: false } });
      scanner.close();
      fireEvent.click(await dialog.findByRole("checkbox", { name: "Record setpoint" }));
      return { dialog, draft };
    };

    it("should ask for a message format before it adds a whole picked topic", async () => {
      const setpoint = createField("setpoint", "float");
      const { ic, message } = await createFormat([setpoint]);
      const { dialog, draft } = await openOnPickedTopic();
      const mode = dialog.getByRole<HTMLInputElement>("checkbox", {
        name: "Record mode",
      });
      expect(mode.checked).toBe(true);
      dialog.getByText("1 topic selected");
      const add = () => dialog.getByRole("button", { name: "Add 1 target" });
      fireEvent.click(add());
      await dialog.findByText(FORMAT_REQUIRED);
      await pickMessageFormat("Select message format", message.name, ic.name);
      await waitFor(() => expect(dialog.queryByText(FORMAT_REQUIRED)).toBeNull());
      fireEvent.click(add());
      await waitFor(async () =>
        expect(await retrieveTargets(draft.key)).toMatchObject([
          {
            type: "plain",
            topic: "plant/valve/set",
            icd: ic.key,
            message: {
              message: message.key,
              bindings: [{ type: "channel", field: setpoint.key, channel: 0 }],
            },
          },
        ]),
      );
    });

    it("should go back to the topics to ask for a message format when a tag adds by name", async () => {
      const { dialog } = await openOnPickedTopic();
      fireEvent.click(dialog.getByRole("button", { name: "Sparkplug B" }));
      fireEvent.click(await dialog.findByRole("button", { name: "Add a tag by name" }));
      await dialog.findByText(FORMAT_REQUIRED);
      expect(isSelectButtonSelected("JSON topics")).toBe(true);
    });

    it("should drop a whole picked topic when any of its fields is unticked", async () => {
      const { dialog } = await openOnPickedTopic();
      const box = (name: string) =>
        dialog.getByRole<HTMLInputElement>("checkbox", { name: `Record ${name}` });
      dialog.getByText("1 topic selected");
      fireEvent.click(box("mode"));
      expect([box("setpoint").checked, box("mode").checked]).toEqual([false, false]);
      expect(dialog.queryByText("1 topic selected")).toBeNull();
    });

    it("should ask for a message format before it adds a topic by name", async () => {
      const setpoint = createField("setpoint", "float");
      const mode = createField("mode", "string", { default: "auto" });
      const { ic, message } = await createFormat([setpoint, mode]);
      const { draft } = await renderWrite();
      const dialog = await openAddTargets();
      const byName = () => dialog.getByRole("button", { name: "Add a topic by name" });
      fireEvent.click(
        await dialog.findByRole("button", { name: "Add a topic by name" }),
      );
      await dialog.findByText(FORMAT_REQUIRED);
      await pickMessageFormat("Select message format", message.name, ic.name);
      await waitFor(() => expect(dialog.queryByText(FORMAT_REQUIRED)).toBeNull());
      fireEvent.click(byName());
      await screen.findByText(setpoint.name);
      await waitFor(async () =>
        expect(await retrieveTargets(draft.key)).toMatchObject([
          {
            type: "plain",
            topic: "",
            icd: ic.key,
            message: {
              message: message.key,
              bindings: [{ type: "channel", field: setpoint.key, channel: 0 }],
            },
          },
        ]),
      );
    });

    it("should show a failed listen in place of the list, with no notification", async () => {
      let statuses: Status.NotificationSpec[] = [];
      await renderWrite([], { onStatuses: (s) => (statuses = s) });
      const dialog = await openAddTargets();
      await dialog.findByText(/Found no task where rack = \d+ and type = "mqtt_scan"/);
      expect(statuses).toHaveLength(0);
    });

    it("should add a picked tag as a target of its Sparkplug B type, once", async () => {
      const tags = [
        { device: "", name: "flow", dataType: "float", value: "1", supported: true },
      ];
      const { dialog, draft, scanner } = await openOnTags({ tags });
      fireEvent.click(await dialog.findByRole("checkbox", { name: "Record flow" }));
      fireEvent.click(dialog.getByRole("button", { name: "Add 1 target" }));
      await screen.findByText("Sparkplug B type");
      await waitFor(async () =>
        expect(await retrieveTargets(draft.key)).toMatchObject([
          {
            type: "sparkplug",
            group: "plant",
            edgeNode: "line1",
            device: "",
            tag: "flow",
            tagType: "float",
            channel: 0,
          },
        ]),
      );
      const again = await openAddTargets();
      await scanner.answer({ data: { topics: [], truncated: false } });
      fireEvent.click(again.getByRole("button", { name: "Sparkplug B" }));
      await scanner.answer({ data: { nodes: NODES, tags: [] } });
      await scanner.answer({ data: { nodes: [], tags } });
      scanner.close();
      const flow = await again.findByRole<HTMLInputElement>("checkbox", {
        name: "Record flow",
      });
      expect(flow.disabled).toBe(true);
      expect(again.getByText("In task")).toBeTruthy();
    });

    it("should list the tags of an edge node under their devices and lock an unsupported one", async () => {
      const tags = [
        {
          device: "",
          name: "Node Control/Rebirth",
          dataType: "boolean",
          value: "false",
        },
        { device: "ovenA", name: "zoneTemp", dataType: "double", value: "21.5" },
        { device: "ovenA", name: "recipe", dataType: "DataSet", value: "" },
      ].map((t) => ({ ...t, supported: t.dataType !== "DataSet" }));
      const { dialog, scanner } = await openOnTags({ tags });
      scanner.close();
      const zoneTemp = await dialog.findByRole<HTMLInputElement>("checkbox", {
        name: "Record zoneTemp",
      });
      expect(zoneTemp.disabled).toBe(false);
      expect(dialog.getByText("Edge node")).toBeTruthy();
      expect(dialog.getByText("ovenA")).toBeTruthy();
      const recipe = dialog.getByRole<HTMLInputElement>("checkbox", {
        name: "Record recipe",
      });
      expect(recipe.disabled).toBe(true);
      expect(dialog.getByText("Not supported")).toBeTruthy();
    });

    it("should keep the edge nodes when one does not answer", async () => {
      const message = "edge node plant/line1 did not answer the rebirth request";
      const { dialog, scanner } = await openOnTags({ message });
      scanner.close();
      await dialog.findByText(message);
      expect(dialog.getByText("line1")).toBeTruthy();
    });
  });

  describe("deploying against a live Core", () => {
    it("should put the deploy errors of a tag on its fields", async () => {
      const { container } = await renderWrite([
        createTagTarget("s1", "", { device: "oven#1" }),
      ]);
      await screen.findByText("Edge node");
      await clickDeploy(container);
      await screen.findByText("Device must not hold /, +, or #");
      expect(screen.getByText("Tag is required")).toBeTruthy();
    });

    it("should create a command channel of the type of each tag", async () => {
      const { container, dev, draft } = await renderWrite([
        createTagTarget("s1", "zone 1/setpoint", {
          device: "ovenA",
          tagType: "float",
        }),
        createTagTarget("s2", "running", { tagType: "boolean" }),
        createTagTarget("s3", "mode", { tagType: "string" }),
        createTagTarget("s4", "started", { tagType: "date_time" }),
      ]);
      const created = await deployAndAwaitTask(
        client,
        container,
        draft.key,
        MQTT.Task.WRITE_SCHEMAS,
      );
      const keys = created.config.targets.map((t) =>
        t.type === "sparkplug" ? t.channel : 0,
      );
      const updated = await client.devices.retrieve({
        key: dev.key,
        schemas: MQTT.Device.SCHEMAS,
      });
      expect(updated.properties.write).toEqual({
        "spBv1.0/plant/line1/ovenA/zone 1/setpoint": keys[0],
        "spBv1.0/plant/line1//running": keys[1],
        "spBv1.0/plant/line1//mode": keys[2],
        "spBv1.0/plant/line1//started": keys[3],
      });
      const channels = await client.channels.retrieve(keys);
      expect(channels.map((ch) => ch.dataType.toString())).toEqual([
        "float32",
        "uint8",
        "string",
        "timestamp",
      ]);
      const cmdName = `${dev.name}_plant_line1_ovenA_zone_1_setpoint_cmd`;
      expect(channels[0].name).toBe(cmdName);
      const indexCh = await client.channels.retrieve(channels[0].index);
      expect(indexCh.name).toBe(`${cmdName}_time`);
      expect(channels[2].virtual).toBe(true);
      expect(channels[3].isIndex).toBe(false);
    });

    it("should give the command channel the name of the target", async () => {
      const name = uniqueName("oven_setpoint_cmd");
      const { container, draft } = await renderWrite([
        createTagTarget("s1", "setpoint", { name }),
      ]);
      const created = await deployAndAwaitTask(
        client,
        container,
        draft.key,
        MQTT.Task.WRITE_SCHEMAS,
      );
      const [target] = created.config.targets;
      if (target.type !== "sparkplug") throw new Error("expected a Sparkplug B target");
      const cmdCh = await client.channels.retrieve(target.channel);
      expect(cmdCh.name).toBe(name);
      expect((await client.channels.retrieve(cmdCh.index)).name).toBe(`${name}_time`);
    });

    it("should reuse the command channel of a tag that the device stores", async () => {
      const dev = await createBroker(client);
      const storedCh = await client.channels.create({
        name: uniqueName("mqtt_cmd"),
        dataType: "float64",
        virtual: true,
      });
      const properties = {
        ...MQTT.Device.ZERO_PROPERTIES,
        write: { "spBv1.0/plant/line1//setpoint": storedCh.key },
      };
      await Device.setProperties(
        client,
        { key: dev.key, properties },
        MQTT.Device.ZERO_PROPERTIES,
        MQTT.Device.SCHEMAS,
      );
      const config = createWriteConfig(dev.key, [createTagTarget("s1", "setpoint")]);
      const draft = await createDraft(client, config);
      const { container } = await renderTaskFormTab(
        MQTT.Task.FORMS[MQTT.Task.WRITE_TYPE],
        {
          client,
          taskKey: draft.key,
        },
      );
      const created = await deployAndAwaitTask(
        client,
        container,
        draft.key,
        MQTT.Task.WRITE_SCHEMAS,
      );
      expect(created.config.targets).toMatchObject([{ channel: storedCh.key }]);
    });

    it("should put the deploy errors on the fields they belong to", async () => {
      const { container } = await renderWrite([
        await createPlainTarget("t1", "plant/#"),
      ]);
      await screen.findByText("Publish");
      await clickDeploy(container);
      await screen.findByText("Topic must not hold the wildcards + or #");
    });

    it("should create a command channel for each binding that needs one", async () => {
      const message = icd.messageZ.parse({
        name: "Command",
        payload: {
          format: "json",
          fields: [
            { name: "value", pointer: "/value", type: "float" },
            { name: "open", pointer: "/open", type: "boolean" },
            { name: "unit", pointer: "/unit", type: "string" },
          ],
        },
      });
      if (message.payload.format !== "json") throw new Error("expected a JSON message");
      const [value, open, unit] = message.payload.fields;
      const ic = await client.icds.create({
        name: uniqueName("format"),
        messages: [message],
      });
      const topic = "plant/valve/set";
      const { container, dev, draft } = await renderWrite([
        mqtt.plainWriteTargetZ.parse({
          type: "plain",
          key: "t1",
          topic,
          icd: ic.key,
          message: {
            message: message.key,
            bindings: [
              { type: "channel", field: value.key, dataType: "int32" },
              { type: "channel", field: open.key },
              {
                type: "value",
                field: unit.key,
                value: { type: "string", value: "degC" },
              },
            ],
          },
        }),
      ]);
      const created = await deployAndAwaitTask(
        client,
        container,
        draft.key,
        MQTT.Task.WRITE_SCHEMAS,
      );
      const [target] = created.config.targets;
      if (target.type !== "plain") throw new Error("expected a plain target");
      const [valueCmd, openCmd, unitFixed] = target.message.bindings;
      expect(unitFixed).toEqual({
        type: "value",
        field: unit.key,
        value: { type: "string", value: "degC" },
      });
      if (valueCmd.type !== "channel" || openCmd.type !== "channel")
        throw new Error("expected channel bindings");
      const keys = [valueCmd.channel, openCmd.channel];
      const channels = await client.channels.retrieve(keys);
      expect(channels.map((ch) => ch.dataType.toString())).toEqual(["int32", "uint8"]);
      const prefix = `${dev.name}_plant_valve_set`;
      expect(channels.map(({ name }) => name)).toEqual([
        `${prefix}_value_cmd`,
        `${prefix}_open_cmd`,
      ]);
      const updated = await client.devices.retrieve({
        key: dev.key,
        schemas: MQTT.Device.SCHEMAS,
      });
      expect(updated.properties.write).toEqual({
        [`${topic}#${value.key}`]: keys[0],
        [`${topic}#${open.key}`]: keys[1],
      });
    });

    it("should replace the integer command channel of a field that gains a scale", async () => {
      const message = icd.messageZ.parse({
        name: "Command",
        payload: {
          format: "json",
          fields: [{ name: "count", pointer: "/count", type: "integer" }],
        },
      });
      if (message.payload.format !== "json") throw new Error("expected a JSON message");
      const [count] = message.payload.fields;
      const ic = await client.icds.create({
        name: uniqueName("format"),
        messages: [message],
      });
      const dev = await createBroker(client);
      const name = `${dev.name}_plant_valve_set_count_cmd`;
      const index = await client.channels.create({
        name: `${name}_time`,
        dataType: "timestamp",
        isIndex: true,
      });
      const old = await client.channels.create({
        name,
        dataType: "int64",
        index: index.key,
      });
      // Configure stores each command channel that it gives a field.
      await Device.setProperties(
        client,
        {
          key: dev.key,
          properties: {
            ...MQTT.Device.ZERO_PROPERTIES,
            write: { [`plant/valve/set#${count.key}`]: old.key },
          },
        },
        MQTT.Device.ZERO_PROPERTIES,
        MQTT.Device.SCHEMAS,
      );
      const draft = await createDraft(
        client,
        createWriteConfig(dev.key, [
          mqtt.plainWriteTargetZ.parse({
            type: "plain",
            key: "t1",
            topic: "plant/valve/set",
            icd: ic.key,
            message: {
              message: message.key,
              bindings: [
                {
                  type: "channel",
                  field: count.key,
                  channel: old.key,
                  dataType: "int64",
                },
              ],
            },
          }),
        ]),
      );
      await client.icds.dispatch(
        ic.key,
        icd.set({
          path: ["messages", message.key, "payload", "fields", count.key, "scale"],
          value: 0.5,
        }),
      );
      const { container } = await renderTaskFormTab(
        MQTT.Task.FORMS[MQTT.Task.WRITE_TYPE],
        { client, taskKey: draft.key },
      );
      const created = await deployAndAwaitTask(
        client,
        container,
        draft.key,
        MQTT.Task.WRITE_SCHEMAS,
      );
      const [target] = created.config.targets;
      if (target.type !== "plain") throw new Error("expected a plain target");
      const [binding] = target.message.bindings;
      if (binding.type !== "channel") throw new Error("expected a channel binding");
      const ch = await client.channels.retrieve(binding.channel);
      expect([ch.name, ch.dataType.toString()]).toEqual([name, "float64"]);
      expect((await client.channels.retrieve(old.key)).name).toBe(`${name}_2`);
    });

    it("should deploy a plain target as it is stored", async () => {
      const plain = await createPlainTarget("t1", "plant/valve/set");
      const { container, draft } = await renderWrite([plain]);
      const created = await deployAndAwaitTask(
        client,
        container,
        draft.key,
        MQTT.Task.WRITE_SCHEMAS,
      );
      expect(created.config.targets).toMatchObject([
        { topic: plain.topic, icd: plain.icd, message: plain.message },
      ]);
    });
  });
});
