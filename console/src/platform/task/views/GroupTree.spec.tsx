// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Component } from "@synnaxlabs/lyra/component";
import { Form } from "@synnaxlabs/lyra/form";
import { type Status } from "@synnaxlabs/lyra/status";
import { id } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { Profiler } from "react";
import { describe, expect, it } from "vitest";

import { Task } from "@/platform/task";
import { renderInTaskForm } from "@/platform/task/testutil";
import { CaptureStatuses } from "@/testutil";

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

const groupLabel = Component.renderProp(({ groupKey }: Task.Views.GroupProps) => (
  <span>group-{groupKey}</span>
));

const groupDetails = Component.renderProp(({ path }: Task.Views.GroupProps) => (
  <span>details-for-{path}</span>
));

const fieldDetails = Component.renderProp(({ path }: Task.Views.FieldProps) => (
  <span>details-for-{path}</span>
));

const createGroups = async (): Promise<Item[]> => [
  { key: id.create(), label: "", disabled: false, index: "", fields: [] },
];

const createField = (): Omit<Field, "key"> => field("", "");

const renderTree = (
  groups: Item[],
  props: Partial<Task.Views.GroupTreeProps<Item, Field>> = {},
  onStatuses?: (statuses: Status.NotificationSpec[]) => void,
) =>
  renderInTaskForm(
    <>
      <Task.Views.GroupTree<Item, Field>
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
    </>,
    { values: { config: { groups } } },
  );

const GROUP: Group = {
  key: "g",
  label: "",
  disabled: false,
  index: "ts",
  fields: [field("ts", "/time", 1), field("v", "/value", 2)],
};

const LEAF: Leaf = { key: "l", label: "", disabled: false, channel: 5, name: "flow" };

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
      expect(form.current?.get("config.groups").value).toHaveLength(1),
    );
    const [added] = form.current?.get<Group[]>("config.groups").value ?? [];
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
      expect(
        form.current?.get<Item[]>("config.groups").value.map(({ key }) => key),
      ).toEqual(["g", "a", "b"]),
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
      expect(form.current?.get<Item[]>("config.groups").value).toEqual([replacement]),
    );
  });

  it("should add a field copied from the last one, without its channel", async () => {
    const { form } = await renderTree([GROUP]);
    await screen.findAllByText("group-g");
    fireEvent.click(screen.getByRole("button", { name: "Add field" }));
    await waitFor(() =>
      expect(form.current?.get("config.groups.g.fields").value).toHaveLength(3),
    );
    const added = form.current?.get<Field[]>("config.groups.g.fields").value[2];
    expect(added?.pointer).toBe("/value");
    expect(added?.channel).toBe(0);
  });

  it("should duplicate a group with fresh fields and a remapped index", async () => {
    const { form } = await renderTree([GROUP]);
    // The selected group also names the details header; the tree item comes first.
    fireEvent.contextMenu((await screen.findAllByText("group-g"))[0]);
    fireEvent.click(await screen.findByText("Duplicate"));
    await waitFor(() =>
      expect(form.current?.get("config.groups").value).toHaveLength(2),
    );
    const copy = form.current?.get<Group[]>("config.groups").value[1];
    expect(copy?.key).not.toBe("g");
    expect(copy?.fields.map((f) => f.channel)).toEqual([0, 0]);
    expect(copy?.index).toBe(copy?.fields[0].key);
    expect(copy?.index).not.toBe("ts");
  });

  it("should remove a field through the context menu", async () => {
    const { form } = await renderTree([GROUP]);
    fireEvent.contextMenu(await screen.findByText("/value"));
    fireEvent.click(await screen.findByText("Remove"));
    await waitFor(() =>
      expect(form.current?.get("config.groups.g.fields").value).toHaveLength(1),
    );
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
      expect(form.current?.get("config.groups").value).toHaveLength(2),
    );
    const copy = form.current?.get<Leaf[]>("config.groups").value[1];
    expect(copy).toMatchObject({ channel: 0, name: "" });
    expect(copy?.key).not.toBe("l");
  });

  it("should switch a group of a switchable tree through its checkbox and menu", async () => {
    const { form } = await renderTree([GROUP], { groupsSwitchable: true });
    const disabled = () => form.current?.get("config.groups.g.disabled").value;
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
    const countedLabel = Component.renderProp(({ groupKey }: Task.Views.GroupProps) => (
      <Profiler
        id={groupKey}
        onRender={(groupKey, phase) => {
          if (phase !== "mount") updates.push(groupKey);
        }}
      >
        <span>group-{groupKey}</span>
      </Profiler>
    ));
    const pointerDetails = Component.renderProp(({ path }: Task.Views.FieldProps) => (
      <Form.TextField path={`${path}.pointer`} label="Pointer" />
    ));
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
});
