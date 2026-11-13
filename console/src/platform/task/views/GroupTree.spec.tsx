// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, type Synnax, type task } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Component } from "@synnaxlabs/lyra/component";
import { Form } from "@synnaxlabs/lyra/form";
import { type Status } from "@synnaxlabs/lyra/status";
import { deep, id } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { Profiler } from "react";
import { describe, expect, it } from "vitest";

import { Task } from "@/platform/task";
import {
  awaitTare,
  renderInTaskForm,
  renderInTaskFormWithClient,
} from "@/platform/task/testutil";
import { CaptureStatuses, getIconButtons, uniqueName } from "@/testutil";

type Field = Task.Views.GroupField;

interface Group extends Task.Views.Group<Field> {
  label: string;
  disabled: boolean;
}

interface Leaf {
  key: string;
  label: string;
  disabled: boolean;
  channel: number;
  name: string;
}

type Item = Group | Leaf;

const field = (key: string, pointer: string, channel = 0): Field => ({
  key,
  pointer,
  channel,
  name: "",
  disabled: false,
});

const path = deep.path<{ config: { groups: Item[] } }>();

/** Any group read as one that may hold fields. */
interface Branch {
  key: string;
  index?: string;
  fields?: Field[];
}

const branches = deep.widen<Branch[]>(path.config.groups);

const fieldsOf = (groupKey: string) => deep.present(branches[groupKey].fields);

const groupLabel = Component.renderProp(({ groupKey }: Task.Views.GroupProps<Item>) => (
  <span>group-{groupKey}</span>
));

const groupDetails = Component.renderProp(({ path }: Task.Views.GroupProps<Item>) => (
  <span>details-for-{String(path)}</span>
));

const fieldDetails = Component.renderProp(({ path }: Task.Views.FieldProps<Field>) => (
  <span>details-for-{String(path)}</span>
));

const createGroups = async (): Promise<Item[]> => [
  { key: id.create(), label: "", disabled: false, index: "", fields: [] },
];

const createField = (): Omit<Field, "key"> => field("", "");

interface RenderOptions extends Partial<Task.Views.GroupTreeProps<Item, Field>> {
  mode?: "preview";
  /** Resolves the channels of the groups and fields. */
  client?: Synnax;
  /** The key of a running task that the form edits. */
  runningTask?: task.Key;
}

const renderTree = async (
  groups: Item[],
  { mode, client, runningTask, ...props }: RenderOptions = {},
  onStatuses?: (statuses: Status.NotificationSpec[]) => void,
) => {
  const ui = (
    <>
      <Task.Views.GroupTree<Item, Field>
        path={path.config.groups}
        listTitle="Groups"
        emptyMessage="No groups"
        placeholder="Select a group or field"
        addLabel="Add group"
        createGroups={createGroups}
        createField={createField}
        groupLabel={groupLabel}
        groupDetails={groupDetails}
        fieldDetails={fieldDetails}
        {...props}
      />
      {onStatuses != null && <CaptureStatuses onStatuses={onStatuses} />}
    </>
  );
  const running =
    runningTask == null
      ? {}
      : { key: runningTask, status: { details: { running: true } } };
  const options = { values: { ...running, config: { groups } }, mode };
  if (client == null) return await renderInTaskForm(ui, options);
  return await renderInTaskFormWithClient(ui, { ...options, client });
};

const GROUP: Group = {
  key: "g",
  label: "",
  disabled: false,
  index: "ts",
  fields: [field("ts", "/time", 1), field("v", "/value", 2)],
};

const LEAF: Leaf = { key: "l", label: "", disabled: false, channel: 5, name: "flow" };

const EMPTY_GROUP: Group = {
  key: "e",
  label: "",
  disabled: false,
  index: "",
  fields: [],
};

describe("GroupTree", () => {
  it("should list a group's fields beneath it and hide its index field", async () => {
    await renderTree([GROUP]);
    await screen.findAllByText("group-g");
    expect(screen.getByText("/value")).toBeTruthy();
    expect(screen.queryByText("/time")).toBeNull();
  });

  it("should show the details of the selected field", async () => {
    await renderTree([GROUP]);
    fireEvent.click(await screen.findByText("/value"));
    await screen.findByText("details-for-config.groups.g.fields.v");
  });

  it("should add a group and select it", async () => {
    const { form } = await renderTree([]);
    fireEvent.click(await screen.findByRole("button", { name: "Add group" }));
    await waitFor(() =>
      expect(form.current?.get(path.config.groups).value).toHaveLength(1),
    );
    const [added] = form.current?.get(path.config.groups).value ?? [];
    await screen.findByText(`details-for-config.groups.${added.key}`);
  });

  it("should add every resolved group and select the first", async () => {
    const resolved: Group[] = [
      { key: "a", label: "", disabled: false, index: "", fields: [] },
      { key: "b", label: "", disabled: false, index: "", fields: [] },
    ];
    const { form } = await renderTree([GROUP], {
      createGroups: async () => resolved,
    });
    fireEvent.click(await screen.findByRole("button", { name: "Add group" }));
    await waitFor(() =>
      expect(form.current?.get(path.config.groups).value.map(({ key }) => key)).toEqual(
        ["g", "a", "b"],
      ),
    );
    await screen.findByText("details-for-config.groups.a");
  });

  it("should replace a group whose key it holds", async () => {
    const replacement: Group = {
      ...GROUP,
      fields: [...GROUP.fields, field("w", "/wind")],
    };
    const { form } = await renderTree([GROUP], {
      createGroups: async () => [replacement],
    });
    fireEvent.click(await screen.findByRole("button", { name: "Add group" }));
    await waitFor(() =>
      expect(form.current?.get(path.config.groups).value).toEqual([replacement]),
    );
  });

  it("should add a field copied from the last one, without its channel", async () => {
    const { form } = await renderTree([GROUP]);
    await screen.findAllByText("group-g");
    fireEvent.click(screen.getByRole("button", { name: "Add field" }));
    await waitFor(() => expect(form.current?.get(fieldsOf("g")).value).toHaveLength(3));
    const added = form.current?.get(fieldsOf("g")).value[2];
    expect(added?.pointer).toBe("/value");
    expect(added?.channel).toBe(0);
  });

  it("should offer an add field item only under a group with no fields", async () => {
    await renderTree([EMPTY_GROUP, GROUP]);
    await screen.findAllByText("group-e");
    expect(screen.getAllByText("Add field")).toHaveLength(1);
  });

  it("should add and select the first field when the add field item is chosen", async () => {
    const { form } = await renderTree([EMPTY_GROUP]);
    fireEvent.click(await screen.findByText("Add field"));
    const fields = fieldsOf("e");
    await waitFor(() => expect(form.current?.get(fields).value).toHaveLength(1));
    const [added] = form.current?.get(fields).value ?? [];
    await screen.findByText(`details-for-config.groups.e.fields.${added.key}`);
    expect(screen.queryByText("Add field")).toBeNull();
  });

  it("should not offer an add field item in preview", async () => {
    await renderTree([EMPTY_GROUP], { mode: "preview" });
    await screen.findAllByText("group-e");
    expect(screen.queryByText("Add field")).toBeNull();
  });

  it("should not offer item actions on an add field item", async () => {
    await renderTree([EMPTY_GROUP]);
    fireEvent.contextMenu(await screen.findByText("Add field"));
    await screen.findByText("Reload Console");
    expect(screen.queryByText("Remove")).toBeNull();
  });

  it("should not move the selection onto an add field item after a remove", async () => {
    await renderTree([EMPTY_GROUP, GROUP]);
    fireEvent.click((await screen.findAllByText("group-g"))[0]);
    fireEvent.keyDown(document.body, { code: "ControlLeft" });
    fireEvent.click(screen.getByText("/value"));
    fireEvent.keyUp(document.body, { code: "ControlLeft" });
    fireEvent.contextMenu(screen.getAllByText("/value")[0]);
    fireEvent.click(await screen.findByText("Remove"));
    await screen.findByText("details-for-config.groups.e");
  });

  it("should duplicate a group with fresh fields and a remapped index", async () => {
    const { form } = await renderTree([GROUP]);
    // The selected group also names the details header; the tree item comes first.
    fireEvent.contextMenu((await screen.findAllByText("group-g"))[0]);
    fireEvent.click(await screen.findByText("Duplicate"));
    await waitFor(() =>
      expect(form.current?.get(path.config.groups).value).toHaveLength(2),
    );
    const copy = form.current?.get(branches).value[1];
    expect(copy?.key).not.toBe("g");
    expect(copy?.fields?.map((f) => f.channel)).toEqual([0, 0]);
    expect(copy?.index).toBe(copy?.fields?.[0].key);
    expect(copy?.index).not.toBe("ts");
  });

  it("should remove a field through the context menu", async () => {
    const { form } = await renderTree([GROUP]);
    fireEvent.contextMenu(await screen.findByText("/value"));
    fireEvent.click(await screen.findByText("Remove"));
    await waitFor(() => expect(form.current?.get(fieldsOf("g")).value).toHaveLength(1));
  });

  it("should name a field with an empty pointer by the empty pointer label", async () => {
    await renderTree([{ ...GROUP, fields: [...GROUP.fields, field("e", "")] }], {
      emptyPointerLabel: "Whole payload",
    });
    await screen.findByText("Whole payload");
  });

  it("should report a failure to create groups", async () => {
    let statuses: Status.NotificationSpec[] = [];
    const rejection = new Error("broker unreachable");
    await renderTree(
      [],
      { createGroups: async () => await Promise.reject(rejection) },
      (s) => (statuses = s),
    );
    fireEvent.click(await screen.findByRole("button", { name: "Add group" }));
    await waitFor(() =>
      expect(
        statuses.some(
          (s) =>
            s.variant === "error" &&
            s.message === "Failed to add group" &&
            s.description === "broker unreachable",
        ),
      ).toBe(true),
    );
  });

  it("should offer no add field action on a leaf group", async () => {
    await renderTree([LEAF]);
    const item = await screen.findByRole("treeitem", { name: /group-l/ });
    expect(within(item).queryByRole("button", { name: "Add field" })).toBeNull();
    fireEvent.contextMenu(item);
    await screen.findByText("Duplicate");
    expect(screen.queryByText("Add field")).toBeNull();
  });

  it("should duplicate a leaf group without its channels", async () => {
    const { form } = await renderTree([LEAF]);
    fireEvent.contextMenu(await screen.findByRole("treeitem", { name: /group-l/ }));
    fireEvent.click(await screen.findByText("Duplicate"));
    await waitFor(() =>
      expect(form.current?.get(path.config.groups).value).toHaveLength(2),
    );
    const copy = form.current?.get(path.config.groups).value[1];
    expect(copy).toMatchObject({ channel: 0, name: "" });
    expect(copy?.key).not.toBe("l");
  });

  it("should switch a group of a switchable tree through its checkbox and menu", async () => {
    const { form } = await renderTree([GROUP], { groupsSwitchable: true });
    const disabled = () => form.current?.get(path.config.groups.g.disabled).value;
    const item = await screen.findByRole("treeitem", { name: /group-g/ });
    fireEvent.click(within(item).getByRole("checkbox", { name: "Enabled" }));
    await waitFor(() => expect(disabled()).toBe(true));
    fireEvent.contextMenu(item);
    expect(screen.queryByText("Disable")).toBeNull();
    fireEvent.click(await screen.findByText("Enable"));
    await waitFor(() => expect(disabled()).toBe(false));
  });

  it("should offer no switch on a group of a tree that does not switch groups", async () => {
    await renderTree([GROUP]);
    const item = await screen.findByRole("treeitem", { name: /group-g/ });
    expect(within(item).queryByRole("checkbox")).toBeNull();
    fireEvent.contextMenu(item);
    await screen.findByText("Duplicate");
    expect(screen.queryByText("Disable")).toBeNull();
  });

  it("should re-render no group when a field takes a keystroke", async () => {
    const updates: string[] = [];
    const countedLabel = Component.renderProp(
      ({ groupKey }: Task.Views.GroupProps<Item>) => (
        <Profiler
          id={groupKey}
          onRender={(groupKey, phase) => {
            if (phase !== "mount") updates.push(groupKey);
          }}
        >
          <span>group-{groupKey}</span>
        </Profiler>
      ),
    );
    const pointerDetails = Component.renderProp(
      ({ path }: Task.Views.FieldProps<Field>) => (
        <Form.TextField path={path.pointer} label="Pointer" />
      ),
    );
    const other: Group = { ...GROUP, key: "h", fields: [field("w", "/wind")] };
    await renderTree([GROUP, other], {
      groupLabel: countedLabel,
      fieldDetails: pointerDetails,
    });
    fireEvent.click(await screen.findByText("/value"));
    const input = await screen.findByDisplayValue("/value");
    updates.length = 0;
    fireEvent.change(input, { target: { value: "/value2" } });
    await screen.findAllByText("/value2");
    expect(updates).toEqual([]);
  });

  it("should mark the end of an item whose channel deploying creates, unless skipped", async () => {
    await renderTree(
      [
        {
          ...GROUP,
          index: "",
          fields: [field("a", "/a"), { ...field("b", "/b"), disabled: true }],
        },
        { ...GROUP, key: "off", index: "", disabled: true, fields: [field("c", "/c")] },
        { ...LEAF, channel: 0 },
      ],
      { groupsSwitchable: true },
    );
    const newIcon = async (name: RegExp) =>
      within(await screen.findByRole("treeitem", { name })).queryByRole("img", {
        name: "New channel",
      });
    expect(await newIcon(/\/b/)).toBeNull();
    expect(await newIcon(/\/c/)).toBeNull();
    expect(await newIcon(/group-l/)).not.toBeNull();
    const item = await screen.findByRole("treeitem", { name: /\/a/ });
    const icon = within(item).getByRole("img", { name: "New channel" });
    expect(within(item).getByText("/a").compareDocumentPosition(icon)).toBe(
      Node.DOCUMENT_POSITION_FOLLOWING,
    );
    fireEvent.pointerOver(icon, { pointerType: "mouse" });
    expect((await screen.findByRole("tooltip")).textContent).toBe(
      "New channelDeploying the task creates it.",
    );
  });

  describe("with a live client", () => {
    const client = createTestClient();

    const createDeleted = async (): Promise<number> => {
      const ch = await client.channels.create({
        name: uniqueName("gone"),
        dataType: DataType.FLOAT32,
        virtual: true,
      });
      await client.channels.delete(ch.key);
      return ch.key;
    };

    it("should mark a field and a leaf whose channels are gone", async () => {
      const fieldChannel = await createDeleted();
      const leafChannel = await createDeleted();
      await renderTree(
        [
          {
            ...GROUP,
            fields: [field("ts", "/time"), field("v", "/value", fieldChannel)],
          },
          { ...LEAF, channel: leafChannel, disabled: true },
        ],
        { client, groupsSwitchable: true },
      );
      const icons = await screen.findAllByRole("img", { name: "Channel not found" });
      expect(icons).toHaveLength(2);
      fireEvent.pointerOver(icons[1], { pointerType: "mouse" });
      expect((await screen.findByRole("tooltip")).textContent).toBe(
        "Channel not foundIt may have been deleted. Deploying the task does not " +
          "create a disabled channel.",
      );
    });

    describe("tare", () => {
      const createChannel = async (dataType: DataType) =>
        (
          await client.channels.create({
            name: uniqueName("tare"),
            dataType,
            virtual: true,
          })
        ).key;

      // Renders a float and an integer field of a running task, then resolves with the
      // tare command that the action sends.
      const tareThrough = async (
        trigger: (container: HTMLElement) => Promise<void>,
      ): Promise<{ args: unknown; float: number }> => {
        const [float, integer] = await Promise.all([
          createChannel(DataType.FLOAT32),
          createChannel(DataType.UINT8),
        ]);
        const group: Group = {
          ...GROUP,
          fields: [
            field("ts", "/time"),
            field("f", "/float", float),
            field("i", "/integer", integer),
          ],
        };
        const rack = await client.racks.create({ name: uniqueName("rack") });
        const tsk = await rack.createTask({
          name: uniqueName("tsk"),
          type: "http_read",
          config: {},
        });
        const args = await awaitTare(client, tsk.key, async () => {
          const { container } = await renderTree([group], {
            client,
            tareable: true,
            runningTask: tsk.key,
          });
          await trigger(container);
        });
        return { args, float };
      };

      it("should tare from the one button, which a float field shows", async () => {
        const { args, float } = await tareThrough(async (container) => {
          await waitFor(() =>
            expect(getIconButtons(container, "tare")).toHaveLength(1),
          );
          fireEvent.click(getIconButtons(container, "tare")[0]);
        });
        expect(args).toEqual({ keys: [float] });
      });

      it("should tare the channel of a field from its menu", async () => {
        const { args, float } = await tareThrough(async () => {
          fireEvent.contextMenu(await screen.findByText("/float"));
          fireEvent.click(await screen.findByText("Tare"));
        });
        expect(args).toEqual({ keys: [float] });
      });
    });
  });
});
