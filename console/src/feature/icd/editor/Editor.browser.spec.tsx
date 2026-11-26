// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { icd, type Synnax } from "@synnaxlabs/client";
import {
  createTestClient,
  createTestClientWithRole,
  spyOnSend,
} from "@synnaxlabs/client/testutil";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Panel as PlutoPanel } from "@synnaxlabs/pluto";
import { TimeSpan, TimeStamp, uuid } from "@synnaxlabs/x";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { act, fireEvent, screen, waitFor, within } from "@testing-library/react";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";

import { ICD } from "@/feature/icd";
import { client, createICD, ENGINE_CSV } from "@/feature/icd/testutil";
import { createResourceTab, primePanel } from "@/platform/panel/testutil";
import { Session } from "@/session";
import {
  createConsoleWrapper,
  fakePickedFile,
  interceptFilePicker,
  renderSuspended,
} from "@/testutil";

const flags = vi.hoisted((): { bus: boolean } => ({ bus: true }));

vi.mock("@/flags", async (importOriginal) => {
  const { FLAGS } = await importOriginal<{ FLAGS: object }>();
  return {
    FLAGS: {
      ...FLAGS,
      get bus() {
        return flags.bus;
      },
    },
  };
});

const { Content } = ICD.TABS[icd.TYPE_RESOURCE_ID.type];

beforeAll(Content.preload);

const renderEditor = async (
  ic: icd.ICD,
  as: Synnax = client,
  selected: icd.MessageKey[] = [],
): Promise<void> => {
  const { wrapper, store } = await createConsoleWrapper({ client: as });
  store.dispatch(Session.ICD.setSelected({ key: ic.key, selected }));
  const { panelKey, tabKey } = await createResourceTab(client, icd.resourceID(ic.key));
  await primePanel(wrapper, panelKey);
  await renderSuspended(
    <PlutoPanel.Scope.Provider value={panelKey}>
      <PlutoPanel.TabScope.Provider value={tabKey}>
        <Content />
      </PlutoPanel.TabScope.Provider>
    </PlutoPanel.Scope.Provider>,
    { wrapper },
  );
  // The editor is a preview until the grant check resolves, and only an editor shows
  // Import.
  if (as === client) await screen.findByRole("button", { name: "Import" });
};

const cell = (column: string, row: number): HTMLInputElement =>
  screen.getByLabelText(`${column} ${row}`);

const commit = async (column: string, row: number, value: string): Promise<void> => {
  await screen.findByLabelText(`${column} ${row}`);
  const input = cell(column, row);
  fireEvent.focus(input);
  fireEvent.change(input, { target: { value } });
  fireEvent.blur(input);
};

const typeCycleTime = async (text: string): Promise<void> => {
  const input = await screen.findByLabelText("Cycle time");
  fireEvent.focus(input);
  fireEvent.change(input, { target: { value: text } });
  fireEvent.blur(input);
};

const createBinaryField = (name: string, startBit: number): icd.BinaryField => ({
  key: uuid.create(),
  name,
  startBit,
  bitLength: 8,
  byteOrder: "little_endian",
  kind: "unsigned",
  scale: 1,
  offset: 0,
  units: "",
  multiplexValues: [],
});

const createMessage = (
  fields: icd.BinaryField[],
  name: string = "Engine",
): icd.Message => ({
  key: uuid.create(),
  name,
  payload: { format: "binary", fields },
});

const createJSONBase = (name: string, pointer: string) => ({
  key: uuid.create(),
  name,
  pointer,
  scale: 1,
  offset: 0,
  units: "",
});

const createJSONField = (name: string, pointer: string): icd.JSONField => ({
  ...createJSONBase(name, pointer),
  type: "float",
  enumValues: [],
});

const createJSONMessage = (fields: icd.JSONField[]): icd.Message => ({
  key: uuid.create(),
  name: "Reading",
  payload: { format: "json", fields },
});

const createTextMessage = (): icd.Message => ({
  key: uuid.create(),
  name: "Line",
  payload: { format: "text", delimiter: ",", prefix: "", fields: [] },
});

const retrieveMessage = async (key: icd.Key): Promise<icd.Message> => {
  const [message] = (await client.icds.retrieve({ key })).messages;
  return message;
};

/** Reads the ICD through a new client, so the result is what the Core stored. */
const retrieveStoredMessage = async (key: icd.Key): Promise<icd.Message> => {
  const core = createTestClient();
  try {
    return (await core.icds.retrieve({ key })).messages[0];
  } finally {
    await core.close();
  }
};

const createFramedICD = async (framing?: icd.Framing): Promise<icd.ICD> =>
  await client.icds.create({ name: uniqueName("icd"), framing });

/** Reads the framing through a new client, so the result is what the Core stored. */
const retrieveStoredFraming = async (
  key: icd.Key,
): Promise<icd.Framing | undefined> => {
  const core = createTestClient();
  try {
    return (await core.icds.retrieve({ key })).framing;
  } finally {
    await core.close();
  }
};

const SYNC: icd.SyncFraming = {
  type: "sync",
  sync: new Uint8Array([0xaa, 0x55]),
  lengthOffset: 2,
  lengthSize: "one",
  byteOrder: "little_endian",
  lengthAdjustment: 0,
  maxLength: 65536,
};

const pick = async (select: RegExp, option: string): Promise<void> => {
  fireEvent.click(await screen.findByRole("button", { name: select }));
  fireEvent.click(await screen.findByRole("option", { name: option }));
};

const typeText = async (label: RegExp, text: string): Promise<HTMLInputElement> => {
  const input = await screen.findByLabelText<HTMLInputElement>(label);
  fireEvent.focus(input);
  fireEvent.change(input, { target: { value: text } });
  fireEvent.blur(input);
  return input;
};

const retrieveJSONField = async (key: icd.Key): Promise<icd.JSONField> => {
  const { payload } = await retrieveMessage(key);
  if (payload.format !== "json") throw new Error("expected a JSON message");
  return payload.fields[0];
};

const openOptions = async (row: number): Promise<void> => {
  fireEvent.click(await screen.findByRole("button", { name: `Options ${row}` }));
};

const retrieveNames = async (key: icd.Key): Promise<string[]> =>
  (await client.icds.retrieve({ key })).messages.map(({ name }) => name);

/** Edits the ICD through a client that the editor does not use. */
const dispatchElsewhere = async (key: icd.Key, action: icd.Action): Promise<void> => {
  const other = createTestClient();
  await other.icds.retrieve({ key });
  expect(await other.icds.dispatch(key, action)).toBe(true);
};

const findMessageItem = async (name: string): Promise<HTMLElement> => {
  const texts = await screen.findAllByText(name);
  const item = texts
    .map((t) => t.closest<HTMLElement>(`.${Menu.CONTEXT_TARGET}`))
    .find((el) => el != null);
  if (item == null) throw new Error(`no message in the list is named ${name}`);
  return item;
};

const removeFromContextMenu = async (item: HTMLElement): Promise<void> => {
  await act(async () => {
    fireEvent.contextMenu(item);
  });
  fireEvent.click(await screen.findByText("Remove"));
};

const paste = (target: HTMLElement, text: string): void => {
  const clipboardData = new DataTransfer();
  clipboardData.setData("text/plain", text);
  // fireEvent.paste copies only own properties onto a new DataTransfer, which drops
  // the data set above.
  fireEvent(target, new ClipboardEvent("paste", { clipboardData, bubbles: true }));
};

describe("ICD editor", () => {
  it("should add a message with a unique name", async () => {
    const ic = await createICD([createMessage([], "Message 1")]);
    await renderEditor(ic);
    await findMessageItem("Message 1");
    fireEvent.click(await screen.findByRole("button", { name: "Add message" }));
    await findMessageItem("Message 2");
    await waitFor(async () =>
      expect(await retrieveNames(ic.key)).toEqual(["Message 1", "Message 2"]),
    );
  });

  it("should open at the message that the session selects", async () => {
    const b = createMessage([], "B");
    const ic = await createICD([createMessage([], "A"), b]);
    await renderEditor(ic, client, [b.key]);
    await screen.findByDisplayValue("B");
    expect(screen.queryByDisplayValue("A")).toBeNull();
  });

  it("should remove the selected messages from the context menu", async () => {
    const ic = await createICD([
      createMessage([], "A"),
      createMessage([], "B"),
      createMessage([], "C"),
    ]);
    await renderEditor(ic);
    await findMessageItem("A");
    const c = await findMessageItem("C");
    fireEvent.keyDown(document.body, { code: "Control" });
    fireEvent.click(c);
    fireEvent.keyUp(document.body, { code: "Control" });
    await removeFromContextMenu(c);
    await waitFor(() => expect(screen.queryByText("A")).toBeNull());
    expect(screen.queryByText("C")).toBeNull();
    await waitFor(async () => expect(await retrieveNames(ic.key)).toEqual(["B"]));
  });

  it("should keep field keys when a field is edited", async () => {
    const fields = [createBinaryField("Rpm", 0), createBinaryField("Temp", 8)];
    const ic = await createICD([createMessage(fields)]);
    await renderEditor(ic);
    await commit("Start bit", 2, "16");
    await waitFor(async () => {
      const { payload } = await retrieveMessage(ic.key);
      expect(payload.fields.map(({ key }) => key)).toEqual(
        fields.map(({ key }) => key),
      );
      expect(payload.fields[1]).toMatchObject({ name: "Temp", startBit: 16 });
    });
  });

  it("should keep the keys of the fields a removal leaves", async () => {
    const fields = [createBinaryField("Rpm", 0), createBinaryField("Temp", 8)];
    const ic = await createICD([createMessage(fields)]);
    await renderEditor(ic);
    await screen.findByLabelText("Name 1");
    const remove = cell("Name", 1)
      .closest("tr")
      ?.querySelector<HTMLButtonElement>("td:last-child button");
    if (remove == null) throw new Error("expected a remove button on row 1");
    fireEvent.click(remove);
    await waitFor(async () => {
      const { payload } = await retrieveMessage(ic.key);
      expect(payload.fields.map(({ key, name }) => ({ key, name }))).toEqual([
        { key: fields[1].key, name: "Temp" },
      ]);
    });
  });

  it("should clear the bitrate switch when CAN FD turns off", async () => {
    const ic = await createICD([
      {
        ...createMessage([]),
        payload: {
          format: "binary",
          fields: [createBinaryField("Rpm", 0)],
          identifier: {
            type: "can",
            id: 0x100,
            extended: false,
            fd: true,
            bitrateSwitched: true,
          },
        },
      },
    ]);
    await renderEditor(ic);
    const brs = await screen.findByLabelText<HTMLInputElement>("Bitrate switch");
    expect(brs.checked).toBe(true);
    fireEvent.click(screen.getByLabelText("CAN FD"));
    await waitFor(() => expect(brs.checked).toBe(false));
    expect(brs.disabled).toBe(true);
    await waitFor(async () => {
      expect((await retrieveMessage(ic.key)).payload).toMatchObject({
        identifier: { fd: false, bitrateSwitched: false },
      });
    });
  });

  it("should replace the payload when the format changes", async () => {
    const ic = await createICD([createMessage([createBinaryField("Rpm", 0)])]);
    await renderEditor(ic);
    fireEvent.click(await screen.findByRole("button", { name: "Text" }));
    const delimiter = await screen.findByLabelText(/^Delimiter/);
    fireEvent.change(delimiter, { target: { value: ";" } });
    await waitFor(async () => {
      const { payload } = await retrieveMessage(ic.key);
      expect(payload).toEqual({
        format: "text",
        delimiter: ";",
        prefix: "",
        fields: [],
      });
    });
  });

  it("should store a cycle time typed alone in milliseconds", async () => {
    const ic = await createICD([createMessage([])]);
    await renderEditor(ic);
    await typeCycleTime("100");
    await waitFor(async () =>
      expect((await retrieveMessage(ic.key)).cycleTime).toEqual(
        TimeSpan.milliseconds(100),
      ),
    );
  });

  it("should store a cycle time typed with a unit", async () => {
    const ic = await createICD([createMessage([])]);
    await renderEditor(ic);
    await typeCycleTime("1.5s");
    await waitFor(async () =>
      expect((await retrieveMessage(ic.key)).cycleTime).toEqual(
        TimeSpan.milliseconds(1500),
      ),
    );
  });

  it("should show the cycle time of a message", async () => {
    const ic = await createICD([
      { ...createMessage([]), cycleTime: TimeSpan.milliseconds(250) },
    ]);
    await renderEditor(ic);
    const cycleTime = await screen.findByLabelText<HTMLInputElement>("Cycle time");
    expect(cycleTime.value).toBe("250ms");
  });

  it("should remove the cycle time when its text is cleared", async () => {
    const ic = await createICD([
      { ...createMessage([]), cycleTime: TimeSpan.milliseconds(250) },
    ]);
    await renderEditor(ic);
    await typeCycleTime("");
    await waitFor(async () =>
      expect((await retrieveMessage(ic.key)).cycleTime).toBeUndefined(),
    );
  });

  it("should check the ICD with the Core after a cycle time is cleared", async () => {
    const ic = await createICD([
      {
        ...createMessage([createBinaryField("Rpm", 0), createBinaryField("Temp", 8)]),
        cycleTime: TimeSpan.milliseconds(250),
      },
    ]);
    await renderEditor(ic);
    const input = await screen.findByLabelText("Cycle time");
    fireEvent.focus(input);
    fireEvent.change(input, { target: { value: "" } });
    fireEvent.keyDown(input, { key: "Enter" });
    await waitFor(async () =>
      expect((await retrieveMessage(ic.key)).cycleTime).toBeUndefined(),
    );
    await commit("Scale", 2, "0");
    expect(await screen.findByText(/^Field 2: scale must be non-zero/)).toBeTruthy();
  });

  it("should refuse a cycle time of zero", async () => {
    const ic = await createICD([
      { ...createMessage([]), cycleTime: TimeSpan.milliseconds(250) },
    ]);
    await renderEditor(ic);
    const input = await screen.findByLabelText("Cycle time");
    fireEvent.focus(input);
    fireEvent.change(input, { target: { value: "0" } });
    expect(await screen.findByText("Cycle time must be 1ns or more")).toBeTruthy();
    fireEvent.blur(input);
    expect((await retrieveMessage(ic.key)).cycleTime).toEqual(
      TimeSpan.milliseconds(250),
    );
  });

  it("should refuse a cycle time past the largest time span", async () => {
    const ic = await createICD([
      { ...createMessage([]), cycleTime: TimeSpan.milliseconds(250) },
    ]);
    await renderEditor(ic);
    const input = await screen.findByLabelText("Cycle time");
    fireEvent.focus(input);
    fireEvent.change(input, { target: { value: "3000000h" } });
    expect(await screen.findByText("Cycle time is too long")).toBeTruthy();
    fireEvent.blur(input);
    expect((await retrieveMessage(ic.key)).cycleTime).toEqual(
      TimeSpan.milliseconds(250),
    );
  });

  it("should store the pointer of a JSON field", async () => {
    const ic = await createICD([createJSONMessage([createJSONField("Temp", "/temp")])]);
    await renderEditor(ic);
    await commit("Pointer", 1, "/sensors/0/temp");
    await waitFor(async () =>
      expect((await retrieveMessage(ic.key)).payload).toMatchObject({
        format: "json",
        fields: [{ name: "Temp", pointer: "/sensors/0/temp" }],
      }),
    );
  });

  it("should match the JSON documents whose value at a pointer equals a string", async () => {
    const ic = await createICD([createJSONMessage([createJSONField("Temp", "/temp")])]);
    await renderEditor(ic);
    fireEvent.click(await screen.findByRole("button", { name: /^Match/ }));
    fireEvent.click(await screen.findByText("String"));
    const pointer = await screen.findByPlaceholderText("/type");
    fireEvent.change(pointer, { target: { value: "/kind" } });
    fireEvent.blur(pointer);
    const value = await screen.findByLabelText(/^Value/);
    fireEvent.change(value, { target: { value: "reading" } });
    fireEvent.blur(value);
    await waitFor(async () =>
      expect((await retrieveMessage(ic.key)).payload).toMatchObject({
        match: { type: "string", pointer: "/kind", value: "reading" },
      }),
    );
  });

  it("should keep the pointer of a match when its type changes", async () => {
    const ic = await createICD([
      {
        ...createJSONMessage([]),
        payload: {
          format: "json",
          fields: [],
          match: { type: "string", pointer: "/kind", value: "reading" },
        },
      },
    ]);
    await renderEditor(ic);
    fireEvent.click(await screen.findByRole("button", { name: /^Match/ }));
    fireEvent.click(await screen.findByText("Signed integer"));
    await waitFor(async () =>
      expect((await retrieveMessage(ic.key)).payload).toMatchObject({
        match: { type: "integer", pointer: "/kind", value: "0" },
      }),
    );
  });

  it("should match the JSON documents that hold a value at a pointer", async () => {
    const ic = await createICD([
      {
        ...createJSONMessage([]),
        payload: {
          format: "json",
          fields: [],
          match: { type: "string", pointer: "/fault", value: "open" },
        },
      },
    ]);
    await renderEditor(ic);
    fireEvent.click(await screen.findByRole("button", { name: /^Match/ }));
    fireEvent.click(await screen.findByText("Present"));
    await waitFor(async () =>
      expect((await retrieveMessage(ic.key)).payload).toEqual({
        format: "json",
        fields: [],
        match: { type: "present", pointer: "/fault" },
      }),
    );
    expect(screen.getByDisplayValue("/fault")).toBeTruthy();
    expect(screen.queryByLabelText(/^Value/)).toBeNull();
  });

  it("should remove the match of a message that takes any document", async () => {
    const ic = await createICD([
      {
        ...createJSONMessage([]),
        payload: {
          format: "json",
          fields: [],
          match: { type: "boolean", pointer: "/alarm", value: true },
        },
      },
    ]);
    await renderEditor(ic);
    fireEvent.click(await screen.findByRole("button", { name: /^Match/ }));
    fireEvent.click(await screen.findByText("Any document"));
    await waitFor(async () =>
      expect((await retrieveMessage(ic.key)).payload).toEqual({
        format: "json",
        fields: [],
      }),
    );
    expect(screen.queryByPlaceholderText("/type")).toBeNull();
  });

  it("should add the JSON fields a paste adds as numbers", async () => {
    const ic = await createICD([createJSONMessage([createJSONField("Temp", "/temp")])]);
    await renderEditor(ic);
    const input = await screen.findByLabelText("Name 1");
    fireEvent.focus(input);
    paste(input, "Temp\t/temp\nSpeed\t/speed");
    await waitFor(async () =>
      expect((await retrieveMessage(ic.key)).payload.fields).toMatchObject([
        { name: "Temp", pointer: "/temp", type: "float" },
        { name: "Speed", pointer: "/speed", type: "float", scale: 1 },
      ]),
    );
  });

  it("should give each field a paste adds its own name", async () => {
    const ic = await createICD([createJSONMessage([createJSONField("Temp", "/temp")])]);
    await renderEditor(ic);
    const input = await screen.findByLabelText("Pointer 1");
    fireEvent.focus(input);
    paste(input, "/a\n/b\n/c");
    await waitFor(async () =>
      expect(
        (await retrieveMessage(ic.key)).payload.fields.map(({ name }) => name),
      ).toEqual(["Temp", "Field 2", "Field 3"]),
    );
  });

  it("should add a JSON field at a numbered pointer that no field reads", async () => {
    const ic = await createICD([
      createJSONMessage([
        createJSONField("Temp", "/field_3"),
        createJSONField("Raw", "/field_4/raw"),
      ]),
    ]);
    await renderEditor(ic);
    fireEvent.click(await screen.findByRole("button", { name: "Add field" }));
    await waitFor(async () =>
      expect((await retrieveMessage(ic.key)).payload.fields).toMatchObject([
        { name: "Temp" },
        { name: "Raw" },
        { name: "Field 5", pointer: "/field_5" },
      ]),
    );
  });

  it("should store the type of a binary field", async () => {
    const ic = await createICD([createMessage([createBinaryField("Temp", 0)])]);
    await renderEditor(ic);
    fireEvent.click(await screen.findByText("Unsigned integer"));
    fireEvent.click(await screen.findByText("Signed integer"));
    await waitFor(async () =>
      expect((await retrieveMessage(ic.key)).payload.fields).toMatchObject([
        { name: "Temp", kind: "integer" },
      ]),
    );
  });

  it("should keep a binary default that the new type holds exactly", async () => {
    const ic = await createICD([
      createMessage([
        { ...createBinaryField("Gear", 0), kind: "unsigned", default: "200" },
      ]),
    ]);
    await renderEditor(ic);
    fireEvent.click(await screen.findByText("Unsigned integer"));
    fireEvent.click(await screen.findByText("Signed integer"));
    await waitFor(async () =>
      expect((await retrieveMessage(ic.key)).payload.fields).toMatchObject([
        { name: "Gear", kind: "integer", default: "200" },
      ]),
    );
  });

  it("should drop a binary default that the new type cannot hold", async () => {
    const ic = await createICD([
      createMessage([
        { ...createBinaryField("Gain", 0), kind: "float", bitLength: 32, default: 0.5 },
      ]),
    ]);
    await renderEditor(ic);
    fireEvent.click(await screen.findByText("Float"));
    fireEvent.click(await screen.findByText("Unsigned integer"));
    await waitFor(async () => {
      const [field] = (await retrieveStoredMessage(ic.key)).payload.fields;
      expect(field).toMatchObject({ name: "Gain", kind: "unsigned" });
      expect(field).not.toHaveProperty("default");
    });
  });

  it("should set the default of a binary field in its grid", async () => {
    const ic = await createICD([createMessage([createBinaryField("Odometer", 0)])]);
    await renderEditor(ic);
    await commit("Default", 1, "18446744073709551615");
    await waitFor(async () =>
      expect((await retrieveMessage(ic.key)).payload.fields).toMatchObject([
        { name: "Odometer", default: "18446744073709551615" },
      ]),
    );
  });

  it("should remove a binary default when its cell is cleared", async () => {
    const ic = await createICD([
      createMessage([
        { ...createBinaryField("Gear", 0), kind: "unsigned", default: "3" },
      ]),
    ]);
    await renderEditor(ic);
    await commit("Default", 1, "");
    await waitFor(async () =>
      expect((await retrieveMessage(ic.key)).payload.fields[0]).not.toHaveProperty(
        "default",
      ),
    );
  });

  it.each([
    {
      kind: "a float",
      message: createMessage([
        { ...createBinaryField("Gain", 0), kind: "float", bitLength: 32, default: 0.5 },
      ]),
    },
    {
      kind: "an unsigned",
      message: createMessage([
        { ...createBinaryField("Gear", 0), kind: "unsigned", default: "3" },
      ]),
    },
    {
      kind: "a text",
      message: {
        key: uuid.create(),
        name: "Line",
        payload: {
          format: "text",
          delimiter: ",",
          prefix: "",
          fields: [
            {
              key: uuid.create(),
              name: "Speed",
              scale: 1,
              offset: 0,
              units: "",
              encoding: "delimited",
              position: 0,
              default: 0.5,
            },
          ],
        },
      } satisfies icd.Message,
    },
  ])("should keep and mark $kind default that is not a number", async ({ message }) => {
    const ic = await createICD([message]);
    await renderEditor(ic);
    await commit("Default", 1, "fast");
    expect(await screen.findByText("Field 1: default is not valid")).toBeTruthy();
    expect(cell("Default", 1).value).toBe("fast");
    expect((await retrieveMessage(ic.key)).payload.fields[0]).toMatchObject({
      default: "fast",
    });
  });

  it("should drop the scale when a JSON field stops being a number", async () => {
    const ic = await createICD([
      createJSONMessage([{ ...createJSONField("Time", "/ts"), scale: 0.1 }]),
    ]);
    await renderEditor(ic);
    fireEvent.click(await screen.findByText("Float"));
    fireEvent.click(await screen.findByText("Timestamp (ms)"));
    await waitFor(async () =>
      expect((await retrieveMessage(ic.key)).payload.fields).toMatchObject([
        { type: "timestamp", timeFormat: "unix_ms", scale: 1 },
      ]),
    );
  });

  it("should keep the scale when a JSON field becomes another number", async () => {
    const ic = await createICD([
      createJSONMessage([{ ...createJSONField("Count", "/n"), scale: 0.1 }]),
    ]);
    await renderEditor(ic);
    fireEvent.click(await screen.findByText("Float"));
    fireEvent.click(await screen.findByText("Signed integer"));
    await waitFor(async () =>
      expect((await retrieveMessage(ic.key)).payload.fields).toMatchObject([
        { type: "integer", scale: 0.1 },
      ]),
    );
  });

  it("should keep the mappings and default a JSON field's new number type holds", async () => {
    const ic = await createICD([
      createJSONMessage([
        {
          ...createJSONBase("State", "/state"),
          type: "integer",
          enumValues: [
            { label: "RUN", value: "1" },
            { label: "HUGE", value: "9007199254740993" },
          ],
          default: "2",
        },
      ]),
    ]);
    await renderEditor(ic);
    fireEvent.click(await screen.findByText("Signed integer"));
    fireEvent.click(await screen.findByText("Float"));
    await waitFor(async () =>
      expect(await retrieveJSONField(ic.key)).toMatchObject({
        type: "float",
        enumValues: [{ label: "RUN", value: 1 }],
        default: 2,
      }),
    );
  });

  it("should change the type of a JSON field with a blank name", async () => {
    const ic = await createICD([createJSONMessage([createJSONField("Temp", "/temp")])]);
    await renderEditor(ic);
    await commit("Name", 1, "");
    await screen.findByText(/^Field 1: name is required/);
    fireEvent.click(await screen.findByText("Float"));
    fireEvent.click(await screen.findByText("String"));
    await commit("Name", 1, "State");
    await waitFor(async () =>
      expect((await retrieveMessage(ic.key)).payload.fields).toMatchObject([
        { name: "State", type: "string", enumValues: [] },
      ]),
    );
  });

  it("should map labels to the values of a JSON field", async () => {
    const ic = await createICD([
      createJSONMessage([
        { ...createJSONBase("State", "/state"), type: "integer", enumValues: [] },
      ]),
    ]);
    await renderEditor(ic);
    await openOptions(1);
    fireEvent.click(await screen.findByRole("button", { name: "Add mapping" }));
    await commit("Label", 1, "RUN");
    await commit("Value", 1, "1");
    await waitFor(async () =>
      expect(await retrieveJSONField(ic.key)).toMatchObject({
        type: "integer",
        enumValues: [{ label: "RUN", value: "1" }],
      }),
    );
    expect(await screen.findByText("1 mapping")).toBeTruthy();
  });

  it("should mark an enum label that is empty", async () => {
    const ic = await createICD([
      createJSONMessage([
        {
          ...createJSONBase("State", "/state"),
          type: "unsigned",
          enumValues: [{ label: "", value: "1" }],
        },
      ]),
    ]);
    await renderEditor(ic);
    await openOptions(1);
    const label = await screen.findByLabelText("Label 1");
    await waitFor(() => expect(label.getAttribute("aria-invalid")).toBe("true"));
  });

  it("should set and remove the default of an optional JSON field", async () => {
    const ic = await createICD([createJSONMessage([createJSONField("Temp", "/temp")])]);
    await renderEditor(ic);
    await openOptions(1);
    fireEvent.click(await screen.findByRole("checkbox", { name: "Optional" }));
    await waitFor(async () =>
      expect(await retrieveJSONField(ic.key)).toMatchObject({ default: 0 }),
    );
    expect(await screen.findByText("Default 0")).toBeTruthy();
    fireEvent.click(screen.getByRole("checkbox", { name: "Optional" }));
    await waitFor(async () =>
      expect(await retrieveJSONField(ic.key)).not.toHaveProperty("default"),
    );
    const options = await screen.findByRole("button", { name: "Options 1" });
    expect(await within(options).findByText("None")).toBeTruthy();
  });

  it("should show, set, and clear the min and max of a JSON field", async () => {
    const ic = await createICD([
      createJSONMessage([{ ...createJSONField("Temp", "/temp"), min: 0, max: 100 }]),
    ]);
    await renderEditor(ic);
    expect(await screen.findByText("Min 0, Max 100")).toBeTruthy();
    await openOptions(1);
    const min = await screen.findByLabelText<HTMLInputElement>("Min");
    fireEvent.change(min, { target: { value: "5" } });
    fireEvent.blur(min);
    const max = screen.getByLabelText<HTMLInputElement>("Max");
    fireEvent.change(max, { target: { value: "" } });
    fireEvent.blur(max);
    await waitFor(async () => {
      const field = await retrieveJSONField(ic.key);
      expect(field.min).toBe(5);
      expect(field).not.toHaveProperty("max");
    });
    expect(await screen.findByText("Min 5")).toBeTruthy();
  });

  it("should store the default of an integer JSON field as text", async () => {
    const ic = await createICD([
      createJSONMessage([
        { ...createJSONBase("State", "/state"), type: "unsigned", enumValues: [] },
      ]),
    ]);
    await renderEditor(ic);
    await openOptions(1);
    fireEvent.click(await screen.findByRole("checkbox", { name: "Optional" }));
    const input = await screen.findByLabelText<HTMLInputElement>("Default");
    fireEvent.change(input, { target: { value: "18446744073709551615" } });
    fireEvent.blur(input);
    await waitFor(async () =>
      expect(await retrieveJSONField(ic.key)).toMatchObject({
        default: "18446744073709551615",
      }),
    );
  });

  it("should set the default of a boolean JSON field", async () => {
    const ic = await createICD([
      createJSONMessage([
        {
          ...createJSONBase("Valve", "/valve"),
          type: "boolean",
          enumValues: [{ label: "OPEN", value: true }],
        },
      ]),
    ]);
    await renderEditor(ic);
    expect(await screen.findByText("1 mapping")).toBeTruthy();
    await openOptions(1);
    expect(await screen.findByDisplayValue("OPEN")).toBeTruthy();
    fireEvent.click(screen.getByRole("checkbox", { name: "Optional" }));
    fireEvent.click(await screen.findByText("True"));
    await waitFor(async () =>
      expect(await retrieveJSONField(ic.key)).toMatchObject({ default: true }),
    );
    expect(await screen.findByText("Default True, 1 mapping")).toBeTruthy();
  });

  it("should show and remove the default of a timestamp JSON field", async () => {
    const at = TimeStamp.milliseconds(1_700_000_000_000);
    const ic = await createICD([
      createJSONMessage([
        {
          ...createJSONBase("Time", "/time"),
          type: "timestamp",
          timeFormat: "unix_ms",
          default: { type: "constant", value: at },
        },
      ]),
    ]);
    await renderEditor(ic);
    expect(
      await screen.findByText(`Default ${at.toString("dateTime", "local")}`),
    ).toBeTruthy();
    await openOptions(1);
    fireEvent.click(await screen.findByRole("checkbox", { name: "Optional" }));
    await waitFor(async () =>
      expect(await retrieveJSONField(ic.key)).not.toHaveProperty("default"),
    );
    const options = await screen.findByRole("button", { name: "Options 1" });
    expect(await within(options).findByText("None")).toBeTruthy();
  });

  it("should show the default of a timestamp JSON field that takes the time now", async () => {
    const ic = await createICD([
      createJSONMessage([
        {
          ...createJSONBase("Time", "/time"),
          type: "timestamp",
          timeFormat: "unix_ms",
          default: { type: "now" },
        },
      ]),
    ]);
    await renderEditor(ic);
    expect(await screen.findByText("Default Now")).toBeTruthy();
  });

  it("should show the default of a UUID, JSON, and bytes JSON field", async () => {
    const ic = await createICD([
      createJSONMessage([
        {
          ...createJSONBase("Device", "/device"),
          type: "uuid",
          default: { type: "random" },
        },
        { ...createJSONBase("Config", "/config"), type: "json", default: '{"a":1}' },
        { ...createJSONBase("Raw", "/raw"), type: "bytes", default: "AQI=" },
      ]),
    ]);
    await renderEditor(ic);
    expect(await screen.findByText("Default Random")).toBeTruthy();
    expect(await screen.findByText('Default {"a":1}')).toBeTruthy();
    expect(await screen.findByText('Default "AQI="')).toBeTruthy();
  });

  it("should set the default of a timestamp JSON field to the time now", async () => {
    const ic = await createICD([
      createJSONMessage([
        {
          ...createJSONBase("Time", "/time"),
          type: "timestamp",
          timeFormat: "unix_ms",
        },
      ]),
    ]);
    await renderEditor(ic);
    await openOptions(1);
    fireEvent.click(await screen.findByRole("checkbox", { name: "Optional" }));
    await waitFor(async () =>
      expect(await retrieveJSONField(ic.key)).toMatchObject({
        default: { type: "now" },
      }),
    );
    expect(await screen.findByText("Default Now")).toBeTruthy();
  });

  it("should change the time of a constant timestamp default", async () => {
    const ic = await createICD([
      createJSONMessage([
        {
          ...createJSONBase("Time", "/time"),
          type: "timestamp",
          timeFormat: "unix_ms",
          default: {
            type: "constant",
            value: TimeStamp.milliseconds(1_700_000_000_000),
          },
        },
      ]),
    ]);
    await renderEditor(ic);
    await openOptions(1);
    fireEvent.click(await screen.findByRole("button", { name: /2023/ }));
    const input = await screen.findByPlaceholderText<HTMLInputElement>(
      "14:05, tomorrow 3pm, now - 5min",
    );
    fireEvent.change(input, { target: { value: "2026-01-02 03:04:05" } });
    expect(await screen.findByText(/local time/)).toBeTruthy();
    fireEvent.keyDown(input, { key: "Enter" });
    const at = new TimeStamp(new Date(2026, 0, 2, 3, 4, 5));
    await waitFor(async () =>
      expect(await retrieveJSONField(ic.key)).toMatchObject({
        default: { type: "constant", value: at },
      }),
    );
    expect(
      await screen.findByText(`Default ${at.toString("dateTime", "local")}`),
    ).toBeTruthy();
  });

  it("should switch a timestamp default from now to a constant", async () => {
    const ic = await createICD([
      createJSONMessage([
        {
          ...createJSONBase("Time", "/time"),
          type: "timestamp",
          timeFormat: "unix_ms",
          default: { type: "now" },
        },
      ]),
    ]);
    await renderEditor(ic);
    await openOptions(1);
    fireEvent.click(await screen.findByText("Constant"));
    await waitFor(async () =>
      expect(await retrieveJSONField(ic.key)).toMatchObject({
        default: { type: "constant" },
      }),
    );
  });

  it("should set a random and then a constant default of a UUID JSON field", async () => {
    const ic = await createICD([
      createJSONMessage([{ ...createJSONBase("Device", "/device"), type: "uuid" }]),
    ]);
    await renderEditor(ic);
    await openOptions(1);
    fireEvent.click(await screen.findByRole("checkbox", { name: "Optional" }));
    expect(await screen.findByText("Default Random")).toBeTruthy();
    fireEvent.click(await screen.findByText("Constant"));
    const station = uuid.create();
    const input = await screen.findByLabelText<HTMLInputElement>("Value");
    fireEvent.change(input, { target: { value: station } });
    fireEvent.blur(input);
    await waitFor(async () =>
      expect(await retrieveJSONField(ic.key)).toMatchObject({
        default: { type: "constant", value: station },
      }),
    );
    expect(await screen.findByText(`Default ${station}`)).toBeTruthy();
  });

  it("should keep the default of a timestamp JSON field on a time format change", async () => {
    const at = TimeStamp.milliseconds(1_700_000_000_000);
    const ic = await createICD([
      createJSONMessage([
        {
          ...createJSONBase("Time", "/time"),
          type: "timestamp",
          timeFormat: "unix_ms",
          default: { type: "constant", value: at },
        },
      ]),
    ]);
    await renderEditor(ic);
    fireEvent.click(await screen.findByText("Timestamp (ms)"));
    fireEvent.click(await screen.findByText("Timestamp (ISO 8601)"));
    await waitFor(async () =>
      expect(await retrieveJSONField(ic.key)).toMatchObject({
        type: "timestamp",
        timeFormat: "iso8601",
        default: { type: "constant", value: at },
      }),
    );
  });

  it("should set the default of a JSON field that holds any value", async () => {
    const ic = await createICD([
      createJSONMessage([{ ...createJSONBase("Raw", "/raw"), type: "json" }]),
    ]);
    await renderEditor(ic);
    await openOptions(1);
    fireEvent.click(await screen.findByRole("checkbox", { name: "Optional" }));
    expect(await screen.findByText("Default null")).toBeTruthy();
    const input = await screen.findByLabelText<HTMLInputElement>("Default");
    fireEvent.change(input, { target: { value: '{"mode":"auto"}' } });
    fireEvent.blur(input);
    await waitFor(async () =>
      expect(await retrieveJSONField(ic.key)).toMatchObject({
        default: '{"mode":"auto"}',
      }),
    );
  });

  it("should store an edit that leaves a problem", async () => {
    const ic = await createICD([
      createMessage([createBinaryField("Rpm", 0), createBinaryField("Temp", 8)]),
    ]);
    await renderEditor(ic);
    await commit("Scale", 2, "0");
    expect(await screen.findByText(/^Field 2: scale must be non-zero/)).toBeTruthy();
    await waitFor(async () =>
      expect((await retrieveMessage(ic.key)).payload.fields[1].scale).toBe(0),
    );
  });

  it("should mark a problem shortly after an edit", async () => {
    const ic = await createICD([
      createMessage([createBinaryField("Rpm", 0), createBinaryField("Temp", 8)]),
    ]);
    await renderEditor(ic);
    await commit("Scale", 2, "0");
    expect(await screen.findByText(/^Field 2: scale must be non-zero/)).toBeTruthy();
  });

  it("should mark a blank name shortly after an edit", async () => {
    const ic = await createICD([
      createMessage([createBinaryField("Rpm", 0), createBinaryField("Temp", 8)]),
    ]);
    await renderEditor(ic);
    await commit("Name", 2, "");
    expect(await screen.findByText(/^Field 2: name is required/)).toBeTruthy();
  });

  it("should clear the mark when an edit fixes the problem", async () => {
    const ic = await createICD([
      createMessage([createBinaryField("Rpm", 0), createBinaryField("Temp", 8)]),
    ]);
    await renderEditor(ic);
    await commit("Scale", 2, "0");
    await screen.findByText(/^Field 2: scale must be non-zero/);
    await commit("Scale", 2, "2");
    await waitFor(() =>
      expect(screen.queryByText(/^Field 2: scale must be non-zero/)).toBeNull(),
    );
  });

  it("should show an edit made elsewhere", async () => {
    const rpm = createBinaryField("Rpm", 0);
    const ic = await createICD([createMessage([rpm])]);
    await renderEditor(ic);
    await screen.findByLabelText("Name 1");
    const [engine] = ic.messages;
    await dispatchElsewhere(
      ic.key,
      icd.set({
        path: ["messages", engine.key, "payload", "fields", rpm.key, "name"],
        value: "Speed",
      }),
    );
    await waitFor(() => expect(cell("Name", 1).value).toBe("Speed"));
  });

  it("should send a cell edit as an edit to its field", async () => {
    const fields = [createBinaryField("Rpm", 0), createBinaryField("Temp", 8)];
    const ic = await createICD([createMessage(fields)]);
    await renderEditor(ic);
    const [engine] = ic.messages;
    await screen.findByLabelText("Start bit 2");
    const send = spyOnSend(client);
    await commit("Start bit", 2, "16");
    await waitFor(() =>
      expect(
        send.mock.calls.find(([target]) => target === "/icd/dispatch")?.[1],
      ).toMatchObject({
        actions: [
          {
            type: "set",
            set: {
              path: [
                "messages",
                engine.key,
                "payload",
                "fields",
                fields[1].key,
                "start_bit",
              ],
              value: 16,
            },
          },
        ],
      }),
    );
  });

  it("should show a viewer the ICD without the controls that edit it", async () => {
    const ic = await createICD([createMessage([createBinaryField("Mode", 0)])]);
    await renderEditor(ic, await createTestClientWithRole(client, "Viewer"));
    expect(await screen.findByText("Mode")).toBeTruthy();
    expect(screen.getByText("Binary")).toBeTruthy();
    expect(screen.queryByText("JSON")).toBeNull();
    expect(screen.queryByRole("button", { name: "Add field" })).toBeNull();
    expect(screen.queryByText("Add message")).toBeNull();
    expect(screen.queryByRole("button", { name: "Import" })).toBeNull();
  });

  it("should show a viewer the options of a JSON field without editing them", async () => {
    const ic = await createICD([
      createJSONMessage([
        {
          ...createJSONBase("State", "/state"),
          type: "string",
          enumValues: [{ label: "ON", value: "running" }],
          default: "idle",
        },
      ]),
    ]);
    await renderEditor(ic, await createTestClientWithRole(client, "Viewer"));
    await openOptions(1);
    expect(await screen.findByText("running")).toBeTruthy();
    expect(screen.getByText('Default "idle", 1 mapping')).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Add mapping" })).toBeNull();
  });

  it("should add a tagged field before it has a tag", async () => {
    const ic = await createICD([createTextMessage()]);
    await renderEditor(ic);
    const [, addTagged] = await screen.findAllByRole("button", { name: "Add field" });
    fireEvent.click(addTagged);
    await waitFor(async () =>
      expect((await retrieveMessage(ic.key)).payload.fields).toMatchObject([
        { encoding: "tagged", name: "Field 1", tag: "" },
      ]),
    );
    expect(await screen.findByText(/^Field 1: tag is required/)).toBeTruthy();
  });

  it("should set the default of a text field in its grid", async () => {
    const speed: icd.TextField = {
      key: uuid.create(),
      name: "Speed",
      scale: 1,
      offset: 0,
      units: "",
      encoding: "delimited",
      position: 0,
    };
    const ic = await createICD([
      {
        key: uuid.create(),
        name: "Line",
        payload: { format: "text", delimiter: ",", prefix: "", fields: [speed] },
      },
    ]);
    await renderEditor(ic);
    await commit("Default", 1, "0.5");
    await waitFor(async () =>
      expect((await retrieveMessage(ic.key)).payload.fields).toMatchObject([
        { name: "Speed", default: 0.5 },
      ]),
    );
  });

  it("should offer a field identifier only when the message has a field", async () => {
    const ic = await createICD([createMessage([])]);
    await renderEditor(ic);
    fireEvent.click(await screen.findByRole("button", { name: /^Identifier/ }));
    await screen.findByText("CAN");
    expect(screen.queryByText("Field value")).toBeNull();
  });

  it("should offer a field identifier when the message has a field", async () => {
    const ic = await createICD([createMessage([createBinaryField("Mode", 0)])]);
    await renderEditor(ic);
    fireEvent.click(await screen.findByRole("button", { name: /^Identifier/ }));
    expect(await screen.findByText("Field value")).toBeTruthy();
  });

  it("should show the messages of an imported file", async () => {
    const picker = interceptFilePicker();
    const ic = await createICD();
    await renderEditor(ic);
    fireEvent.click(await screen.findByRole("button", { name: "Import" }));
    await waitFor(() => expect(picker.lastInput()).toBeDefined());
    picker.selectFiles([fakePickedFile("engine.csv", ENGINE_CSV)]);
    await findMessageItem("Engine");
    expect(cell("Name", 1).value).toBe("Rpm");
  });

  describe("framing", () => {
    afterEach(() => {
      flags.bus = true;
    });

    it("should hide the framing while the bus flag is off", async () => {
      flags.bus = false;
      await renderEditor(await createFramedICD());
      expect(screen.queryByText("Framing")).toBeNull();
      expect(screen.queryByRole("button", { name: /^Framing/ })).toBeNull();
    });

    it.each<[string, icd.Framing]>([
      ["Delimited", { type: "delimited", delimiter: "\n" }],
      ["Fixed length", { type: "fixed", length: 1 }],
      ["Sync and length", SYNC],
      ["COBS", { type: "cobs" }],
      ["SLIP", { type: "slip" }],
    ])("should store the framing that %s picks", async (option, want) => {
      const ic = await createFramedICD();
      await renderEditor(ic);
      await pick(/^Framing/, option);
      await waitFor(async () =>
        expect(await retrieveStoredFraming(ic.key)).toEqual(want),
      );
    });

    it("should remove the framing when None is picked", async () => {
      const ic = await createFramedICD({ type: "cobs" });
      await renderEditor(ic);
      await pick(/^Framing/, "None");
      await waitFor(async () =>
        expect(await retrieveStoredFraming(ic.key)).toBeUndefined(),
      );
    });

    it("should store a delimiter written with escapes", async () => {
      const ic = await createFramedICD({ type: "delimited", delimiter: "\n" });
      await renderEditor(ic);
      const input = await typeText(/^Delimiter/, "\\r\\n");
      await waitFor(async () =>
        expect(await retrieveStoredFraming(ic.key)).toEqual({
          type: "delimited",
          delimiter: "\r\n",
        }),
      );
      expect(input.value).toBe("\\r\\n");
    });

    it("should store the length of a fixed framing", async () => {
      const ic = await createFramedICD({ type: "fixed", length: 1 });
      await renderEditor(ic);
      await typeText(/^Length/, "64");
      await waitFor(async () =>
        expect(await retrieveStoredFraming(ic.key)).toEqual({
          type: "fixed",
          length: 64,
        }),
      );
    });

    it("should store sync bytes written in hexadecimal", async () => {
      const ic = await createFramedICD(SYNC);
      await renderEditor(ic);
      const input = await typeText(/^Sync bytes/, "0x7E 0x7e01");
      await waitFor(async () =>
        expect(await retrieveStoredFraming(ic.key)).toEqual({
          ...SYNC,
          sync: new Uint8Array([0x7e, 0x7e, 0x01]),
        }),
      );
      expect(input.value).toBe("7E 7E 01");
    });

    it("should set and remove the checksum of a sync framing", async () => {
      const ic = await createFramedICD(SYNC);
      await renderEditor(ic);
      await pick(/^Checksum/, "CRC-32");
      await waitFor(async () =>
        expect(await retrieveStoredFraming(ic.key)).toEqual({
          ...SYNC,
          checksum: { algorithm: "crc32", byteOrder: "little_endian" },
        }),
      );
      await pick(/^Checksum/, "None");
      await waitFor(async () =>
        expect(await retrieveStoredFraming(ic.key)).toEqual(SYNC),
      );
    });
  });
});
