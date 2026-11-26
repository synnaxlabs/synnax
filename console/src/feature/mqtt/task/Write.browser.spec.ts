// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, icd, mqtt, type Synnax, type task } from "@synnaxlabs/client";
import {
  createTestClient,
  holdResponses,
  rejectRequests,
} from "@synnaxlabs/client/testutil";
import type { Status } from "@synnaxlabs/lyra/status";
import { scheduler } from "@synnaxlabs/x";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { act, fireEvent, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it, onTestFinished, vi } from "vitest";

import { MQTT } from "@/feature/mqtt";
import { createBroker, openScanner } from "@/feature/mqtt/testutil";
import { Device } from "@/platform/device";
import { ICD } from "@/platform/icd";
import {
  isDimmed,
  pickMessageFormat,
  removeField,
  retype,
  selectedIn,
  setField,
  titleOf,
} from "@/platform/icd/testutil";
import {
  clickDeploy,
  countFailedSaves,
  deployAndAwaitTask,
  preloadForms,
  renderTaskFormTab,
  type RenderTaskFormTabOptions,
  selectFromDropdown,
  toggleRefused,
} from "@/platform/task/testutil";
import { getSwitchInput } from "@/testutil";

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

describe("MQTT Write form", () => {
  // This first test pays the one-time load of the lazy form, which takes over 10 s on
  // a loaded CI runner.
  it("should show the empty state and add + select a target", async () => {
    await renderWrite();
    await screen.findByText("Add targets to write to a topic or a tag");
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
    fireEvent.contextMenu(
      await screen.findByRole("treeitem", { name: /plant\/valve/ }),
    );
    fireEvent.click(await screen.findByText("Duplicate"));
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

  it("should mark a new command channel, unless its target is disabled", async () => {
    await renderWrite([
      createTagTarget("s1", "setpoint"),
      createTagTarget("s2", "spare", { disabled: true }),
    ]);
    const on = await screen.findByRole("treeitem", { name: /setpoint/ });
    expect(within(on).getByRole("img", { name: "New channel" })).toBeTruthy();
    const off = screen.getByRole("treeitem", { name: /spare/ });
    expect(within(off).queryByRole("img", { name: "New channel" })).toBeNull();
  });

  it("should mark a field that deploying puts on a new channel, but not one that sends its default", async () => {
    const setpoint = createField("setpoint", "float");
    const mode = createField("mode", "string", { default: "auto" });
    const { target } = await createFormatTarget([setpoint, mode]);
    await renderWrite([target]);
    const on = await screen.findByRole("treeitem", { name: /^setpoint/ });
    expect(within(on).getByRole("img", { name: "New channel" })).toBeTruthy();
    const off = screen.getByRole("treeitem", { name: /^mode/ });
    expect(within(off).queryByRole("img", { name: "New channel" })).toBeNull();
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
    const item = await screen.findByRole("treeitem", { name: /plant\/valve\/set/ });
    fireEvent.click(within(item).getByRole("checkbox", { name: "Enabled" }));
    await waitFor(async () =>
      expect(await retrieveTargets(draft.key)).toMatchObject([{ disabled: true }]),
    );
  });

  it("should show the planned command channel name of a field after a rename", async () => {
    const { target } = await createFormatTarget([createField("speed", "float")]);
    const { dev } = await renderWrite([
      { ...target, topic: "plant/valve-set" },
      { ...target, key: "t2", topic: "plant/valve_set" },
    ]);
    const prefix = `${channel.escapeInvalidName(dev.name)}_plant_valve_set`;
    const [, second] = await screen.findAllByRole("treeitem", { name: /^speed/ });
    fireEvent.click(second);
    await screen.findByDisplayValue(`${prefix}_speed_cmd_2`);
    fireEvent.change(screen.getByDisplayValue("speed"), { target: { value: "rate" } });
    expect(await screen.findByDisplayValue(`${prefix}_rate_cmd_2`)).toBeTruthy();
  });

  describe("Sparkplug B targets", () => {
    it("should say that deploying numbers a typed command channel name that another channel holds", async () => {
      const name = uniqueName("mqtt_cmd");
      await client.channels.create({ name, dataType: "float64", virtual: true });
      await renderWrite([createTagTarget("s1", "setpoint", { name })]);
      fireEvent.click(await screen.findByRole("treeitem", { name: /setpoint/ }));
      expect(await screen.findByText(`Saves as ${name}_2`)).toBeTruthy();
      expect(screen.getByDisplayValue(name)).toBeTruthy();
    });

    it("should list a tag by its edge node, with its command channel in its pane", async () => {
      const ch = await client.channels.create({
        name: uniqueName("mqtt_cmd"),
        dataType: "float64",
        virtual: true,
      });
      await renderWrite([
        await createPlainTarget("t1", "plant/valve/set"),
        createTagTarget("s1", "setpoint", { device: "ovenA", channel: ch.key }),
      ]);
      await screen.findByRole("treeitem", { name: /JSON.*plant\/valve\/set/ });
      fireEvent.click(
        screen.getByRole("treeitem", { name: /SpB.*plant\/line1\/ovenA\/setpoint/ }),
      );
      await screen.findByDisplayValue(ch.name);
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
      fireEvent.contextMenu(await screen.findByRole("treeitem", { name: /setpoint/ }));
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
    /** Selects a field in the tree, and returns the section that fills it. */
    const fieldGroup = async (name: string) => {
      fireEvent.click(
        await screen.findByRole("treeitem", { name: new RegExp(`^${name}`) }),
      );
      return within(await screen.findByRole("group", { name: "Source" }));
    };

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
        await screen.findByRole("treeitem", { name: /^setpoint/ });
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

      it("should send a refused binding once, and again on the next format change", async () => {
        const setpoint = createField("setpoint", "float");
        const { target } = await createFormatTarget([setpoint]);
        let statuses: Status.NotificationSpec[] = [];
        const { draft } = await renderWrite([target], {
          onStatuses: (s) => (statuses = s),
        });
        await screen.findByRole("treeitem", { name: /^setpoint/ });
        // Pointer "" reads the whole document, which holds the value of setpoint.
        const whole = createField("whole", "float", { pointer: "" });
        await client.icds.dispatch(
          target.icd,
          icd.insert({
            path: ["messages", target.message.message, "payload", "fields"],
            value: whole,
            after: setpoint.key,
          }),
        );
        await waitFor(() => expect(countFailedSaves(statuses)).toBeGreaterThan(0));
        await toggleRefused("Retain");
        expect(countFailedSaves(statuses)).toBe(2);
        await setField(client, messageOf(target), { ...whole, pointer: "/whole" });
        await waitFor(async () =>
          expect((await retrieveTargets(draft.key))[0]).toMatchObject({
            message: {
              bindings: [
                { type: "channel", field: setpoint.key, channel: 0 },
                { type: "channel", field: whole.key, channel: 0 },
              ],
            },
          }),
        );
        expect(countFailedSaves(statuses)).toBe(2);
      });

      it("should send a refused unbind once, and again on the next format change", async () => {
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
        let statuses: Status.NotificationSpec[] = [];
        const { draft } = await renderWrite(
          [
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
          ],
          { onStatuses: (s) => (statuses = s) },
        );
        await (await fieldGroup(count.name)).findByDisplayValue(cmd.name);
        // Pointer /count/x reads inside count, so the Core refuses each save.
        await setField(client, messageOf(target), { ...setpoint, pointer: "/count/x" });
        await retype(client, messageOf(target), count, "string", setpoint.key);
        await waitFor(() => expect(countFailedSaves(statuses)).toBeGreaterThan(0));
        fireEvent.click(screen.getByRole("treeitem", { name: /plant\/valve\/set/ }));
        await toggleRefused("Retain");
        expect(countFailedSaves(statuses)).toBe(2);
        await setField(client, messageOf(target), setpoint);
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
        expect(countFailedSaves(statuses)).toBe(2);
      });

      it("should correct for a format that changes while a refused save is in flight", async () => {
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
        let statuses: Status.NotificationSpec[] = [];
        const { draft } = await renderWrite(
          [
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
          ],
          { onStatuses: (s) => (statuses = s) },
        );
        await (await fieldGroup(count.name)).findByDisplayValue(cmd.name);
        const { responded, release } = holdResponses(client, "/task/dispatch");
        // Pointer /count/x reads inside count, so the Core refuses the unbind of count.
        await setField(client, messageOf(target), { ...setpoint, pointer: "/count/x" });
        await retype(client, messageOf(target), count, "string", setpoint.key);
        await responded;
        const renamed = { ...setpoint, name: "target" };
        await setField(client, messageOf(target), renamed);
        await fieldGroup(renamed.name);
        // The form corrects for the format a task after it renders the format.
        await act(async () => await scheduler.nextTask());
        release();
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
        expect(countFailedSaves(statuses)).toBe(1);
      });

      it("should suit the bindings when the channels of the message fail to load", async () => {
        const setpoint = createField("setpoint", "float");
        const count = createField("count", "integer");
        const [{ target }, cmd, dev] = await Promise.all([
          createFormatTarget([setpoint, count]),
          client.channels.create({
            name: uniqueName("mqtt_cmd"),
            dataType: "int64",
            virtual: true,
          }),
          createBroker(client),
        ]);
        const draft = await createDraft(
          client,
          createWriteConfig(dev.key, [
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
          ]),
        );
        await removeField(client, messageOf(target), setpoint);
        // A new client caches no channel, so its retrievals of cmd reach the Core.
        const as = createTestClient();
        const reject = rejectRequests(
          as,
          (t, req) =>
            t === "/channel/retrieve" && JSON.stringify(req).includes(`${cmd.key}`),
          new Error("channels unavailable"),
        );
        onTestFinished(() => reject.mockRestore());
        // The plan of the command channel names fails on the same retrieval.
        const error = vi.spyOn(console, "error").mockImplementation(() => {});
        onTestFinished(() => error.mockRestore());
        let statuses: Status.NotificationSpec[] = [];
        await renderTaskFormTab(MQTT.Task.FORMS[MQTT.Task.WRITE_TYPE], {
          client,
          as,
          taskKey: draft.key,
          onStatuses: (s) => (statuses = s),
        });
        await waitFor(async () =>
          expect((await retrieveTargets(draft.key))[0]).toMatchObject({
            message: {
              bindings: [{ type: "channel", field: count.key, channel: cmd.key }],
            },
          }),
        );
        await waitFor(() =>
          expect(statuses).toContainEqual(
            expect.objectContaining({
              message: "Failed to retrieve the channels of a message",
              description: "channels unavailable",
            }),
          ),
        );
        expect(error).toHaveBeenCalledWith(
          expect.objectContaining({ message: "channels unavailable" }),
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

    const retrieveFields = async (ic: icd.ICD, message: icd.Message) => {
      const [retrieved] = await client.icds.retrieve([ic.key]);
      return ICD.jsonFields(retrieved.messages.find(({ key }) => key === message.key));
    };

    it("should add a field to the message in its ICD, with the focus on its name, then put it on a channel", async () => {
      const setpoint = createField("setpoint", "float");
      const { target, ic, message } = await createFormatTarget([setpoint]);
      const { draft } = await renderWrite([target]);
      fireEvent.click(await screen.findByRole("button", { name: "Add field" }));
      const name = await screen.findByDisplayValue<HTMLInputElement>("Field 2");
      await waitFor(() => expect(document.activeElement).toBe(name));
      expect([name.selectionStart, name.selectionEnd]).toEqual([0, "Field 2".length]);
      screen.getByDisplayValue("/field_2");
      fireEvent.change(screen.getByPlaceholderText("/temperature"), {
        target: { value: "/limit" },
      });
      let added: icd.JSONField | undefined;
      await waitFor(async () => {
        added = (await retrieveFields(ic, message)).find((f) => f.pointer === "/limit");
        expect(added).toBeDefined();
      });
      await waitFor(async () =>
        expect((await retrieveTargets(draft.key))[0]).toMatchObject({
          message: {
            bindings: [
              { type: "channel", field: setpoint.key },
              { type: "channel", field: added?.key, channel: 0 },
            ],
          },
        }),
      );
    });

    it("should add a first field to a message with no fields", async () => {
      const { target } = await createFormatTarget([]);
      await renderWrite([target]);
      await screen.findByText("Message has no fields");
      fireEvent.click(screen.getByRole("button", { name: "Add field" }));
      await screen.findByDisplayValue("Field 1");
      fireEvent.click(screen.getByRole("treeitem", { name: /plant\/valve\/set/ }));
      await screen.findByText("Publish");
      expect(screen.queryByText("Message has no fields")).toBeNull();
    });

    it("should rename a field in its ICD from its pane", async () => {
      const setpoint = createField("setpoint", "float");
      const { target, ic, message } = await createFormatTarget([setpoint]);
      await renderWrite([target]);
      await fieldGroup(setpoint.name);
      fireEvent.change(screen.getByDisplayValue(setpoint.name), {
        target: { value: "target" },
      });
      await waitFor(async () =>
        expect((await retrieveFields(ic, message)).map(({ name }) => name)).toEqual([
          "target",
        ]),
      );
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
      shapes: [
        {
          fields: [
            { pointer: "/setpoint", type: "number", kind: "float", sample: "1" },
            { pointer: "/mode", type: "string", kind: "string", sample: '"auto"' },
          ],
          count: 1,
        },
      ],
      retained: true,
    };

    // Opens the dialog over a broker whose scan task lists topic, and picks it.
    const openOnPickedTopic = async (
      targets: MQTT.Task.WriteTarget[] = [],
      topic: typeof TOPIC = TOPIC,
    ) => {
      const { dev, draft } = await renderWrite(targets);
      const scanner = await openScanner(client, dev.rack);
      const dialog = await openAddTargets();
      await scanner.answer({ data: { topics: [topic], truncated: false } });
      scanner.close();
      fireEvent.click(await dialog.findByRole("checkbox", { name: "Pick set" }));
      return { dialog, draft };
    };

    const retrievePlainTarget = async (key: task.Key, topic: string) => {
      let found: MQTT.Task.WriteTarget | undefined;
      await waitFor(async () => {
        found = (await retrieveTargets(key)).find(
          (t) => t.type === "plain" && t.topic === topic,
        );
        expect(found).toBeDefined();
      });
      if (found?.type !== "plain")
        throw new Error(`expected a plain target of ${topic}`);
      return found;
    };

    it("should add a whole picked topic with a format from its messages, in an ICD named after the task", async () => {
      const { dialog, draft } = await openOnPickedTopic();
      dialog.getByText("1 topic selected");
      fireEvent.click(dialog.getByRole("button", { name: "Add 1 target" }));
      const target = await retrievePlainTarget(draft.key, TOPIC.topic);
      const [ic] = await client.icds.retrieve([target.icd]);
      expect(ic).toMatchObject({
        name: "MQTT write task",
        messages: [
          {
            key: target.message.message,
            name: TOPIC.topic,
            payload: {
              format: "json",
              fields: [
                { name: "setpoint", pointer: "/setpoint", type: "float" },
                { name: "mode", pointer: "/mode", type: "string" },
              ],
            },
          },
        ],
      });
      expect(target.message.bindings).toEqual(
        ICD.jsonFields(ic.messages[0]).map((f) => ({
          type: "channel",
          field: f.key,
          channel: 0,
        })),
      );
    });

    it("should infer the format of a topic from the shape that the most messages had, and mark it", async () => {
      const topic = {
        topic: "plant/pump/set",
        payload: '{"rpm": 5, "mode": "auto"}',
        shapes: [
          {
            fields: [
              { pointer: "/setpoint", type: "number", kind: "float", sample: "1" },
            ],
            present: "/setpoint",
            count: 1,
          },
          {
            fields: [
              { pointer: "/rpm", type: "number", kind: "float", sample: "5" },
              { pointer: "/mode", type: "string", kind: "string", sample: '"auto"' },
            ],
            present: "/rpm",
            count: 3,
          },
        ],
        retained: true,
      };
      const { dev, draft } = await renderWrite();
      const scanner = await openScanner(client, dev.rack);
      const dialog = await openAddTargets();
      await scanner.answer({ data: { topics: [topic], truncated: false } });
      scanner.close();
      const note = await dialog.findByText("Used for the format");
      expect(titleOf(note)).toBe("3 messages with /rpm");
      expect(
        ["setpoint", "rpm", "mode"].map((n) => isDimmed(dialog.getByText(n))),
      ).toEqual([true, false, false]);
      fireEvent.click(dialog.getByRole("checkbox", { name: "Pick set" }));
      fireEvent.click(dialog.getByRole("button", { name: "Add 1 target" }));
      const target = await retrievePlainTarget(draft.key, topic.topic);
      const [ic] = await client.icds.retrieve([target.icd]);
      expect(ICD.jsonFields(ic.messages[0])).toMatchObject([
        { name: "rpm", pointer: "/rpm", type: "float" },
        { name: "mode", pointer: "/mode", type: "string" },
      ]);
    });

    it("should infer the format of a picked topic from the last listen", async () => {
      const listen = (setpoints: number, rpms: number) => ({
        topic: "plant/pump/set",
        payload: '{"rpm": 5, "mode": "auto"}',
        shapes: [
          {
            fields: [
              { pointer: "/setpoint", type: "number", kind: "float", sample: "1" },
            ],
            present: "/setpoint",
            count: setpoints,
          },
          {
            fields: [
              { pointer: "/rpm", type: "number", kind: "float", sample: "5" },
              { pointer: "/mode", type: "string", kind: "string", sample: '"auto"' },
            ],
            present: "/rpm",
            count: rpms,
          },
        ],
        retained: true,
      });
      const { dev, draft } = await renderWrite();
      const scanner = await openScanner(client, dev.rack);
      const dialog = await openAddTargets();
      await scanner.answer({ data: { topics: [listen(3, 1)], truncated: false } });
      fireEvent.click(await dialog.findByRole("checkbox", { name: "Pick set" }));
      fireEvent.click(dialog.getByRole("button", { name: "Listen again" }));
      await scanner.answer({ data: { topics: [listen(1, 3)], truncated: false } });
      scanner.close();
      await waitFor(() =>
        expect(titleOf(dialog.getByText("Used for the format"))).toBe(
          "3 messages with /rpm",
        ),
      );
      fireEvent.click(dialog.getByRole("button", { name: "Add 1 target" }));
      const target = await retrievePlainTarget(draft.key, "plant/pump/set");
      const [ic] = await client.icds.retrieve([target.icd]);
      expect(ICD.jsonFields(ic.messages[0]).map((f) => f.pointer)).toEqual([
        "/rpm",
        "/mode",
      ]);
    });

    it("should show the fields of each target that one pick adds", async () => {
      const { dev } = await renderWrite();
      const scanner = await openScanner(client, dev.rack);
      const dialog = await openAddTargets();
      const pump = { ...TOPIC, topic: "plant/pump/set" };
      await scanner.answer({ data: { topics: [TOPIC, pump], truncated: false } });
      scanner.close();
      fireEvent.click(await dialog.findByRole("checkbox", { name: "Pick plant" }));
      fireEvent.click(dialog.getByRole("button", { name: "Add 2 targets" }));
      await waitFor(() =>
        expect(screen.getAllByRole("treeitem", { name: /^setpoint/ })).toHaveLength(2),
      );
    });

    it("should send the time of each write from an inferred timestamp field", async () => {
      const time = {
        pointer: "/time",
        type: "number",
        sample: "1700000000000",
        kind: "timestamp",
        timeFormat: "unix_ms",
      };
      const [shape] = TOPIC.shapes;
      const { dialog, draft } = await openOnPickedTopic([], {
        ...TOPIC,
        shapes: [{ ...shape, fields: [...shape.fields, time] }],
      });
      fireEvent.click(dialog.getByRole("button", { name: "Add 1 target" }));
      const target = await retrievePlainTarget(draft.key, TOPIC.topic);
      const [ic] = await client.icds.retrieve([target.icd]);
      const fields = ICD.jsonFields(ic.messages[0]);
      expect(fields.at(-1)).toMatchObject({
        pointer: "/time",
        type: "timestamp",
        default: { type: "now" },
      });
      expect(target.message.bindings.map(({ field }) => field)).toEqual(
        fields.slice(0, -1).map(({ key }) => key),
      );
    });

    it("should send the picked message format over the one it would infer", async () => {
      const setpoint = createField("setpoint", "float");
      const { ic, message } = await createFormat([setpoint]);
      const { dialog, draft } = await openOnPickedTopic();
      await pickMessageFormat("Select message format", message.name, ic.name);
      fireEvent.click(dialog.getByRole("button", { name: "Add 1 target" }));
      await waitFor(async () =>
        expect(await retrieveTargets(draft.key)).toMatchObject([
          {
            type: "plain",
            topic: TOPIC.topic,
            icd: ic.key,
            message: {
              message: message.key,
              bindings: [{ type: "channel", field: setpoint.key, channel: 0 }],
            },
          },
        ]),
      );
    });

    it("should add the inferred format to the ICD of the most targets, numbered when its name is taken", async () => {
      const setpoint = createField("setpoint", "float");
      const held = icd.messageZ.parse({
        name: TOPIC.topic,
        payload: { format: "json", fields: [setpoint] },
      });
      const ic = await client.icds.create({
        name: uniqueName("valve"),
        messages: [held],
      });
      const other = mqtt.plainWriteTargetZ.parse({
        type: "plain",
        key: "t1",
        topic: "plant/other/set",
        icd: ic.key,
        message: { message: held.key, bindings: ICD.Write.suit([], [setpoint]) },
      });
      const { dialog, draft } = await openOnPickedTopic([other]);
      fireEvent.click(dialog.getByRole("button", { name: "Add 1 target" }));
      const target = await retrievePlainTarget(draft.key, TOPIC.topic);
      expect(target.icd).toEqual(ic.key);
      await waitFor(async () => {
        const [retrieved] = await client.icds.retrieve([ic.key]);
        expect(retrieved.messages.map(({ key, name }) => ({ key, name }))).toEqual([
          { key: held.key, name: TOPIC.topic },
          { key: target.message.message, name: `${TOPIC.topic} 2` },
        ]);
      });
    });

    it("should show the fields of a topic with no boxes, and drop it through its own box", async () => {
      const { dialog } = await openOnPickedTopic();
      dialog.getByText("mode");
      expect(dialog.queryByRole("checkbox", { name: "Record mode" })).toBeNull();
      dialog.getByText("1 topic selected");
      fireEvent.click(dialog.getByRole("checkbox", { name: "Pick set" }));
      expect(dialog.queryByText("1 topic selected")).toBeNull();
    });

    it("should add a topic by name with an empty format named after it", async () => {
      const { draft } = await renderWrite();
      const dialog = await openAddTargets();
      fireEvent.change(dialog.getByPlaceholderText("Search topics"), {
        target: { value: "plant/new/set" },
      });
      fireEvent.click(
        await dialog.findByRole("button", { name: "Add “plant/new/set” as a topic" }),
      );
      await screen.findByText("Message has no fields");
      const target = await retrievePlainTarget(draft.key, "plant/new/set");
      expect(target.message.bindings).toEqual([]);
      const [ic] = await client.icds.retrieve([target.icd]);
      expect(ic.messages).toMatchObject([
        {
          key: target.message.message,
          name: "plant/new/set",
          payload: { format: "json", fields: [] },
        },
      ]);
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
